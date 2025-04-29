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

package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/backend"
)

const (
	modelsRequestRelativePath = "/v1/models" // this is a const according to OpenAI list models API schema.
)

var _ backend.Scraper = &ModelScraper{}

type ModelScraper struct{}

func (s *ModelScraper) Name() string {
	return "model"
}

// Scrape scrapes available models (LoRA and base) from a given pod based on OpenAI models API.
// returns back a slice of models IDs that are currently available on the given pod.
func (s *ModelScraper) Scrape(ctx context.Context, pod *backend.Pod) (backend.ScrapeResult, error) {
	url := fmt.Sprintf("http://%s/%s", pod.Address, modelsRequestRelativePath)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create models request - %w", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models from pod %s - %w", pod.NamespacedName, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from pod %s: %v", pod.NamespacedName, response.StatusCode)
	}

	return s.parseResponse(response.Body)
}

func (s *ModelScraper) parseResponse(responseBody io.Reader) (backend.ScrapeResult, error) {
	var modelsResponse ModelsResponse
	if err := json.NewDecoder(responseBody).Decode(&modelsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode models response body - %w", err)
	}

	modelIDs := make([]string, len(modelsResponse.Data))
	for i, modelInfo := range modelsResponse.Data {
		modelIDs[i] = modelInfo.ID
	}

	return ModelScrapeResult{ModelIDs: modelIDs}, nil
}

func (s *ModelScraper) ProcessResult(result backend.ScrapeResult) {

}
