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

func newInferenceMessage() InferenceMessage {
	return InferenceMessage{
		Headers:        map[string]string{},
		Body:           make(map[string]any),
		mutatedHeaders: make(map[string]string),
	}
}

type InferenceMessage struct {
	// original request
	Headers map[string]string
	Body    map[string]any

	// mutations
	mutatedHeaders map[string]string
}

func (r *InferenceMessage) SetHeader(key string, value string) {
	if old, ok := r.Headers[key]; !ok || old != value { // if we add or replace a header
		r.Headers[key] = value
		r.mutatedHeaders[key] = value
	}
}

func (r *InferenceMessage) MutatedHeaders() map[string]string {
	return r.mutatedHeaders
}

type InferenceRequest struct {
	InferenceMessage
}

type InferenceResponse struct {
	InferenceMessage
}

// NewInferenceRequest returns a new request with initialized Headers, Body, and mutatedHeaders.
func NewInferenceRequest() *InferenceRequest {
	return &InferenceRequest{
		InferenceMessage: newInferenceMessage(),
	}
}

// NewInferenceResponse returns a new response with initialized Headers, Body, and mutatedHeaders.
func NewInferenceResponse() *InferenceResponse {
	return &InferenceResponse{
		InferenceMessage: newInferenceMessage(),
	}
}
