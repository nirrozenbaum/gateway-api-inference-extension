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
	framework "sigs.k8s.io/gateway-api-inference-extension/pkg/epp/framework/interface/scheduling"
)

// createProfileData gets a scheduling profile and returns pointer to ProfileData that is needed for adaptive configuration.
func createProfileData(profile *framework.SchedulerProfile) *ProfileData {
	activeScorers := profile.GetScorers()
	scorersByCategory := map[framework.ScorerCategory][]*framework.WeightedScorer{
		framework.Affinity:     {},
		framework.Distribution: {},
	}
	sumOfOriginalWeightsByCategory := map[framework.ScorerCategory]float64{
		framework.Affinity:     0.0,
		framework.Distribution: 0.0,
	}
	originalScorersWeights := make(map[*framework.WeightedScorer]float64)

	for _, scorer := range activeScorers {
		if scorer.Category() == framework.Balance {
			continue // don't adapt weights of scorers from type Balance
		}
		scorersByCategory[scorer.Category()] = append(scorersByCategory[scorer.Category()], scorer)
		weight := scorer.Weight()
		originalScorersWeights[scorer] = weight
		sumOfOriginalWeightsByCategory[scorer.Category()] += weight
	}
	// Remove categories with total weight 0.
	// assume 0 weight scorers setting was intentional and that this category shouldn't be handled by the adaptive configurator.
	// this assumption simplifies later calculations where we might divide by zero if we don't make this assumption.
	for category, categoryWeight := range sumOfOriginalWeightsByCategory {
		if categoryWeight <= 0 {
			delete(sumOfOriginalWeightsByCategory, category)
			delete(scorersByCategory, category)
		}
	}

	return &ProfileData{
		scorersByCategory:              scorersByCategory,
		originalScorersWeights:         originalScorersWeights,
		sumOfOriginalWeightsByCategory: sumOfOriginalWeightsByCategory,
	}
}

type ProfileData struct {
	scorersByCategory              map[framework.ScorerCategory][]*framework.WeightedScorer
	originalScorersWeights         map[*framework.WeightedScorer]float64
	sumOfOriginalWeightsByCategory map[framework.ScorerCategory]float64
}
