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
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"

	logutil "sigs.k8s.io/gateway-api-inference-extension/pkg/common/util/logging"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/datastore"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/framework/interface/plugin"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/metrics"
)

const (
	PressureDetectorType = "pressure-detector"
)

// compile-time type assertion
var _ AsyncDetector = &PressureDetector{}

// NewPressureDetector initializes a new PressureDetector and returns its pointer.
func NewPressureDetector(ds datastore.Datastore, interval time.Duration) *PressureDetector {
	return &PressureDetector{
		typedName: plugin.TypedName{Type: PressureDetectorType, Name: PressureDetectorType},
		ds:        ds,
		interval:  interval,
		ratio:     0,
		lock:      sync.RWMutex{},
	}
}

type PressureDetector struct {
	typedName plugin.TypedName
	ds        datastore.Datastore // temp, we should consume endpoints through datalayer endpoints data source (doesn't exist yet, Etai is working on)
	interval  time.Duration
	ratio     float64
	lock      sync.RWMutex
}

func (d *PressureDetector) TypedName() plugin.TypedName {
	return d.typedName
}

// Consumes returns the list of data that is consumed by the plugin.
func (d *PressureDetector) Consumes() map[string]any {
	return map[string]any{
		metrics.RunningRequestsSizeKey: int(0),
		metrics.WaitingQueueSizeKey:    int(0),
	}
}

func (d *PressureDetector) Start(ctx context.Context) error {
	log.FromContext(ctx).Info("Starting async detector", "TypedName", d.typedName)
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C: // refresh signal periodically
			signalRatio := d.refreshSignal(ctx)
			d.lock.Lock()
			d.ratio = signalSmoothingRatio*d.ratio + (1-signalSmoothingRatio)*signalRatio
			d.lock.Unlock()
		}
	}
}

// SignalRatio represent the pressure state and is bound to the range [0,1].
// Pressure is represented as (num_of_queued_requests)/(total_requests)
// SignalRatio = 0 means that the pool has no requests queued at all.
// SignalRatio = 1 means the pool is maximally pressured.
// the Signal is monotonic: more pressure results in larger signal
func (d *PressureDetector) SignalRatio() float64 {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.ratio
}

func (d *PressureDetector) refreshSignal(ctx context.Context) float64 {
	endpoints := d.ds.PodList(datastore.AllPodsPredicate)

	totalQueuedRequests := 0.0
	totalRequests := 0.0

	for _, endpoint := range endpoints {
		metrics := endpoint.GetMetrics()
		totalQueuedRequests += float64(metrics.WaitingQueueSize)
		totalRequests += float64(metrics.RunningRequestsSize) + float64(metrics.WaitingQueueSize)
	}

	// gate with minimal number of requests to avoid noise and prevent false pressure signal when only few requests exist.
	// useful during cold start or getting out of idle state.
	if totalRequests < minTotalRequests {
		log.FromContext(ctx).V(logutil.VERBOSE).Info("total requests lower than minimum", "count", totalRequests)
		return 0 // considered no pressure
	}

	signal := totalQueuedRequests / totalRequests

	metrics.RecordAsyncDetectorSignal(d.typedName.Type, d.typedName.Name, signal)
	metrics.RecordQueuedRequests(totalQueuedRequests)
	metrics.RecordRunningRequests(totalRequests)

	return signal
}
