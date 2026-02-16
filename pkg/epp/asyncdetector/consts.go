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

const (
	kvUtilNormalizedCVMetricKey          = "kv-utilization"
	effectiveKvUtilNormalizedCVMetricKey = "effective-kv-utilization"
	assignedRequestNormalizedCVMetricKey = "assigned-requests"
	minTotalRequests                     = 5   // avoid false positives from extremely low traffic, at least 5 requests to trigger ratio calculation
	signalSmoothingRatio                 = 0.1 //  0.1 weight for previous signal and 0.9 to the new one, for smoothing the signal
)
