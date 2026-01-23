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

package scheduling

import (
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/asyncdetector"
	framework "sigs.k8s.io/gateway-api-inference-extension/pkg/epp/framework/interface/scheduling"
)

type AdaptiveConfigurator struct {
	imbalanceDetector *asyncdetector.ImbalanceDetector
}

func (c *AdaptiveConfigurator) AdaptScorerWeights(profile *framework.SchedulerProfile) {
	imbalanceRatio := c.imbalanceDetector.SignalRatio()

	activeScorers := profile.GetScorers()
	scorersByCategory := map[framework.ScorerCategory][]*framework.WeightedScorer{
		framework.Affinity:     {},
		framework.Distribution: {},
		framework.Balance:      {},
	}

	sumOfWeightsByCategory := map[framework.ScorerCategory]int{
		framework.Affinity:     0,
		framework.Distribution: 0,
		framework.Balance:      0,
	}

	for _, scorer := range activeScorers {
		sumOfWeightsByCategory[scorer.Category()] += scorer.Weight()
		scorersByCategory[scorer.Category()] = append(scorersByCategory[scorer.Category()], scorer)
	}

	// TODO rebalance based on the ratio...
	if imbalanceRatio > 0.2 {

	}

}
