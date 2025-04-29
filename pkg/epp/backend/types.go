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

package backend

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/gateway-api-inference-extension/api/v1alpha2"
)

type Pod struct {
	NamespacedName types.NamespacedName
	Address        string
	Labels         map[string]string
}

func (p *Pod) String() string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%+v", *p)
}

func (p *Pod) Clone() *Pod {
	if p == nil {
		return nil
	}
	return &Pod{
		NamespacedName: types.NamespacedName{
			Name:      p.NamespacedName.Name,
			Namespace: p.NamespacedName.Namespace,
		},
		Address: p.Address,
		Labels:  p.Labels,
	}
}

type Datastore interface {
	PoolGet() (*v1alpha2.InferencePool, error)
	// PodMetrics operations
	// PodGetAll returns all pods and metrics, including fresh and stale.
	// PodGetAll() []PodMetrics
	// PodList(func(PodMetrics) bool) []PodMetrics
}

type Scraper interface {
	Name() string
	Scrape(ctx context.Context, pod *Pod) (ScrapeResult, error)
	ProcessResult(ScrapeResult)
}

type ScrapeResult interface{}
