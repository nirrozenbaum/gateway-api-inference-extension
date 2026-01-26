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
	ImbalanceDetectorType                = "imbalance-detector"
	signalSmoothingRatio                 = 0.1 //  0.1 weight for previous signal and 0.9 to the new one, for smoothing the signal
	minTotalRequests                     = 5   // avoid false positives from extremely low traffic, at least 5 requests to trigger ratio calculation
	kvUtilNormalizedCVMetricKey          = "kv-utilization"
	effectiveKvUtilNormalizedCVMetricKey = "effective-kv-utilization"
	assignedRequestNormalizedCVMetricKey = "assigned-requests"
	imbalanceFormulaKVFactor             = 0.6
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
	lock      sync.RWMutex
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

func (d *ImbalanceDetector) Start(ctx context.Context) error {
	log.FromContext(ctx).Info("Starting async detector", "TypedName", d.typedName)
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C: // refresh signal periodically
			signalRatio := d.refreshSignal()
			d.lock.Lock()
			d.ratio = signalSmoothingRatio*d.ratio + (1-signalSmoothingRatio)*signalRatio
			d.lock.Unlock()
		}
	}
}

// SignalRatio represent the imbalance state and is bound to the range [0,1].
// SignalRatio = 0 means that the pool is perfectly balanced.
// SignalRatio = 1 means the pool is maximally imbalanced.
// the Signal is monotonic: more imbalance results in larger signal
func (d *ImbalanceDetector) SignalRatio() float64 {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.ratio
}

func (d *ImbalanceDetector) refreshSignal() float64 {
	endpoints := d.ds.PodList(datastore.AllPodsPredicate)
	if len(endpoints) <= 1 { // if zero or one endpoints in the pool, the pool is balanced
		return 0
	}

	kvUtilizationPerEndpoint := make([]float64, len(endpoints))
	// requestsPerEndpoint represents the irreversible load pressure already assigned to an endpoint and cannot be taken back.
	// for the purpose of this we summarize (running + waiting requests), since we cannot take back requests from the waiting queue.
	requestsPerEndpoint := make([]float64, len(endpoints))
	totalRequests := 0.0

	for i, endpoint := range endpoints {
		metrics := endpoint.GetMetrics()
		kvUtilizationPerEndpoint[i] = metrics.KVCacheUsagePercent
		requestsPerEndpoint[i] = float64(metrics.RunningRequestsSize) + float64(metrics.WaitingQueueSize)
		totalRequests += requestsPerEndpoint[i]
	}

	// gate with minimal number of requests to avoid noise and prevent false imbalance signal when only few requests exist.
	// useful during cold start or getting out of idle state.
	if totalRequests < minTotalRequests {
		return 0 // considered balanced
	}

	normalizedRequestsPerEndpoint := make([]float64, len(endpoints))
	for i := range endpoints {
		// normalizedRequestsPerEndpoint is within [0,1] range and is representing the number of requests
		// assigned to the endpoint relative to total num of requests. this allows us to detect imbalance using CV.
		normalizedRequestsPerEndpoint[i] = requestsPerEndpoint[i] / totalRequests
	}

	normalizedCvKvUtilization := normalizedCoefficientOfVariation(kvUtilizationPerEndpoint)
	normalizedCvAssignedRequests := normalizedCoefficientOfVariation(normalizedRequestsPerEndpoint)

	// final formula properties:
	// Requests dominate (irreversible load), KV still matters but only when meaningful, no masking of request imbalance by KV
	signal := math.Max(normalizedCvAssignedRequests, imbalanceFormulaKVFactor*normalizedCvKvUtilization)

	metrics.RecordImbalanceNormalizedCV(kvUtilNormalizedCVMetricKey, normalizedCvKvUtilization)
	metrics.RecordImbalanceNormalizedCV(effectiveKvUtilNormalizedCVMetricKey, imbalanceFormulaKVFactor*normalizedCvKvUtilization)
	metrics.RecordImbalanceNormalizedCV(assignedRequestNormalizedCVMetricKey, normalizedCvAssignedRequests)
	metrics.RecordImbalanceSignal(signal)

	return signal
}

// NormalizedCoefficientOfVariation calculates the coefficient of variation (CV)
// and normalizes it by dividing by sqrt(n-1), which is the maximum possible CV
// for non-negative data with a fixed mean (e.g. proportions or bounded utilization).
// This normalization makes the result comparable across different N values
// and bounds the output to [0,1] under those assumptions.
func normalizedCoefficientOfVariation(data []float64) float64 {
	cv := coefficientOfVariation(data)

	// math.Sqrt(float64(len(endpoints)-1) is mathematically the max CV for len(endpoints)=N.
	// since N may change, we normalize the CV by dividing in the max possible CV.
	// this normalization allows to compare between CVs when N changes (otherwise the scale is different).
	normalizedCV := cv / math.Sqrt(float64(len(data)-1))

	// safe guard just to make sure normalized CV never exceeds 1.
	return math.Min(1.0, normalizedCV)
}

// CoefficientOfVariation returns the statistics calculation of CV on the given input.
// CV is a statistical measure showing relative variability, calculated as the standard deviation divided
// by the mean.
func coefficientOfVariation(data []float64) float64 {
	mean, variance := stat.MeanVariance(data, nil)
	if mean <= 0 {
		return 0
	}
	std := math.Sqrt(variance)
	return (std / mean)
}
