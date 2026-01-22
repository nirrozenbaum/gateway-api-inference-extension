/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package asyncdetector

import (
	"context"
	"math"
	"sync"
	"time"

	"gonum.org/v1/gonum/stat"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/datastore"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/framework/interface/plugin"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/metrics"
)

const (
	ImbalanceDetectorType = "imbalance-detector"
	// dwellTimeCycles is the number of cycles that we need to wait after signal changes before changing it again.
	dwellTimeCycles = 3

	// TODO weights should be configurable/tunable
	weightKVUtil           = 0.5
	weightAssignedRequests = 0.5
)

// compile-time type assertion
var _ AsyncDetector = &ImbalanceDetector{}

// NewImbalanceDetector initializes a new ImbalanceDetector and returns its pointer.
func NewImbalanceDetector(ds datastore.Datastore, interval time.Duration) *ImbalanceDetector {
	return &ImbalanceDetector{
		typedName: plugin.TypedName{Type: ImbalanceDetectorType, Name: ImbalanceDetectorType},
		ds:        ds,
		interval:  interval,
		ratio:     0,
	}
}

type ImbalanceDetector struct {
	typedName plugin.TypedName
	ds        datastore.Datastore // temp, we should consume endpoints through datalayer endpoints data source (doesn't exist yet, Etai is working on)
	interval  time.Duration
	ratio     float64
	lock      sync.Mutex // TODO lock on ratio

	startOnce sync.Once // ensures the refresh loop goroutine is started only once
	stopOnce  sync.Once // ensures the done channel is closed only once
	done      chan struct{}
}

func (d *ImbalanceDetector) TypedName() plugin.TypedName {
	return d.typedName
}

// Consumes returns the list of data that is consumed by the plugin.
func (d *ImbalanceDetector) Consumes() map[string]any {
	return map[string]any{
		metrics.RunningRequestsSizeKey: int(0),
		metrics.WaitingQueueSizeKey:    int(0),
		metrics.KVCacheUsagePercentKey: float64(0),
	}
}

func (d *ImbalanceDetector) Start(ctx context.Context) {
	d.startOnce.Do(func() {
		go func() {
			log.FromContext(ctx).Info("Starting async detector", "TypedName", d.typedName)
			ticker := time.NewTicker(d.interval)
			defer ticker.Stop()
			for {
				select {
				case <-d.done:
					return
				case <-ctx.Done():
					return
				case <-ticker.C: // refresh metrics periodically
					// TODO check dwell time. if we're on dwell time skip the call
					d.ratio = d.refreshSignal()
				}
			}
		}()
	})
}

func (d *ImbalanceDetector) SignalRatio() float64 {
	return d.ratio
}

func (d *ImbalanceDetector) refreshSignal() float64 {
	endpoints := d.ds.PodList(datastore.AllPodsPredicate)
	if len(endpoints) <= 1 { // if zero or one endpoints in the pool, the pool is balanced
		return 0
	}

	kvUtililzationValues := make([]float64, len(endpoints))
	assignedRequestsValues := make([]float64, len(endpoints))
	totalRequests := float64(0)

	for i, endpoint := range endpoints {
		metrics := endpoint.GetMetrics()
		kvUtililzationValues[i] = metrics.KVCacheUsagePercent
		assignedRequestsValues[i] = float64(metrics.RunningRequestsSize) + float64(metrics.WaitingQueueSize)
		totalRequests += assignedRequestsValues[i]
	}

	// TODO check only if there is some minimal number of requests to avoid noise to prevent false imbalance when few requests exist

	cvKvUtil := d.coefficientOfVariation(kvUtililzationValues)
	cvAssignedRequests := d.coefficientOfVariation(assignedRequestsValues)

	// TODO emit both CVs as metrics

	return weightKVUtil*d.normalizeCV(cvKvUtil) + weightAssignedRequests*d.normalizeCV(cvAssignedRequests)
}

func (d *ImbalanceDetector) coefficientOfVariation(data []float64) float64 {
	mean, variance := stat.MeanVariance(data, nil)
	std := math.Sqrt(variance)
	return (std / mean)
}

func (d *ImbalanceDetector) normalizeCV(cv float64) float64 {
	// TODO normalize
	return cv
}
