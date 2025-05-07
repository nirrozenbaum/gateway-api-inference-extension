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

package filter

import (
	pluginutil "sigs.k8s.io/gateway-api-inference-extension/pkg/epp/scheduling/plugins/util"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/scheduling/types"
)

// ModelFilter is filtering pods by model.
// It checks which models are available on a pod and filter out pods that doesn't have the requested model.
// For using ModelFilter, it's mandatory to enable ModelsScraper in PodInfoFactory.
type ModelFilter struct{}

func (f *ModelFilter) Name() string {
	return "model"
}

func (f *ModelFilter) Filter(ctx *types.SchedulingContext, pods []types.Pod) []types.Pod {
	filteredPods := []types.Pod{}

	targetModel := ctx.Req.ResolvedTargetModel
	for _, pod := range pods {
		if podModels := pluginutil.GetModelsFromPodInfo(pod); podModels != nil {
			if pluginutil.Contains(podModels.ModelIDs, targetModel) {
				filteredPods = append(filteredPods, pod)
			}
		}
	}

	return filteredPods
}
