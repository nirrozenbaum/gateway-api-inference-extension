/*
Copyright 2026 The Kubernetes Authors.

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

package framework

func NewInferenceRequest() *InferenceRequest {
	return &InferenceRequest{
		Headers:        map[string]string{},
		Body:           make(map[string]any),
		mutatedHeaders: make(map[string]string),
	}
}

type InferenceRequest struct {
	// original request
	Headers map[string]string
	Body    map[string]any

	// mutations
	mutatedHeaders map[string]string
}

func (r *InferenceRequest) SetHeader(key string, value string) {
	if old, ok := r.Headers[key]; !ok || old != value { // if we add or replace a header
		r.Headers[key] = value
		r.mutatedHeaders[key] = value
	}
}

func (r *InferenceRequest) MutatedHeaders() map[string]string {
	return r.mutatedHeaders
}

func NewInferenceResponse() *InferenceResponse {
	return &InferenceResponse{
		Headers: make(map[string]string),
		Body:    make(map[string]any),
	}
}

type InferenceResponse struct {
	Headers map[string]string
	Body    map[string]any
}
