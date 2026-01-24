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
	"math"

	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/asyncdetector"
	framework "sigs.k8s.io/gateway-api-inference-extension/pkg/epp/framework/interface/scheduling"
)

// Note: distribution and affinity are intentionally asymmetric, where affinity max weight is 0.7
// and distribution max weight is 0.8. At equilibrium (r ≈ 0.46), weights are ~50/50.
// At r = 0.5, distribution already slightly dominates. the higher the imbalance ratio is, the more
// weight we give to the distribution scorers.
const (
	distributionWeightMin = 0.3
	distributionWeightMax = 0.8
	sigmoidS0             = 0.5 // s0 is midpoint, the signal r where sigmoid(r) = 0.5
	sigmoidK              = 10  // represents the slope/steepness of the sigmoid function. higher k is more aggressive.
)

// NewAdaptiveConfigurator creates a new AdaptiveConfigurator object and returns its pointer.
func NewAdaptiveConfigurator(imbalanceDetector *asyncdetector.ImbalanceDetector, config *SchedulerConfig) *AdaptiveConfigurator {
	profileScorersByCategory := make(map[string]map[framework.ScorerCategory][]*framework.WeightedScorer)
	profileSumOfWeightsByCategory := make(map[string]map[framework.ScorerCategory]float64)
	for name, profile := range config.profiles {
		activeScorers := profile.GetScorers()
		scorersByCategory := map[framework.ScorerCategory][]*framework.WeightedScorer{
			framework.Affinity:     {},
			framework.Distribution: {},
		}
		sumOfWeightsByCategory := map[framework.ScorerCategory]float64{
			framework.Affinity:     0.0,
			framework.Distribution: 0.0,
		}
		for _, scorer := range activeScorers {
			if scorer.Category() == framework.Balance {
				continue // don't adapt weights of scorers from type Balance
			}
			weight := float64(scorer.Weight()) // TODO should be remove later when weight is changed to float
			sumOfWeightsByCategory[scorer.Category()] += weight
			scorersByCategory[scorer.Category()] = append(scorersByCategory[scorer.Category()], scorer)
		}

		// Remove categories with total weight 0.
		// if that was added to configuration we assume that was intentional and that this category shouldn't be handled
		// by the adaptive configurator.
		// this assumption simplifies later calculations where we might divide by zero if we don't make this assumption.
		for category, categoryWeight := range sumOfWeightsByCategory {
			if categoryWeight <= 0 {
				delete(sumOfWeightsByCategory, category)
				delete(scorersByCategory, category)
			}
		}

		profileScorersByCategory[name] = scorersByCategory
		profileSumOfWeightsByCategory[name] = sumOfWeightsByCategory
	}

	adaptiveConfigurator := &AdaptiveConfigurator{
		imbalanceDetector:             imbalanceDetector,
		profiles:                      config.profiles,
		profileScorersByCategory:      profileScorersByCategory,
		profileSumOfWeightsByCategory: profileSumOfWeightsByCategory,
	}
	// we tune the starting point to 70/30 ratio between affinity/distribution scorers
	adaptiveConfigurator.adaptWeights(distributionWeightMin, 1-distributionWeightMin)

	return adaptiveConfigurator
}

type AdaptiveConfigurator struct {
	imbalanceDetector             *asyncdetector.ImbalanceDetector
	profiles                      map[string]*framework.SchedulerProfile
	profileScorersByCategory      map[string]map[framework.ScorerCategory][]*framework.WeightedScorer
	profileSumOfWeightsByCategory map[string]map[framework.ScorerCategory]float64
}

// TODO should run periodically
func (c *AdaptiveConfigurator) Run() {
	distributionWeight, affinityWeight := c.calculateDesiredWeights()
	c.adaptWeights(distributionWeight, affinityWeight)
}

// calculateDesiredWeights returns the desired weights (total budget) for distribution and affinity categories.
func (c *AdaptiveConfigurator) calculateDesiredWeights() (float64, float64) {
	imbalanceRatio := c.imbalanceDetector.SignalRatio()
	sigmoidValue := sigmoid(imbalanceRatio, sigmoidS0, sigmoidK)

	// weight(D) = wDmin + (sigmoid(ratio) * (wDmax - wDmin))
	distributionWeight := distributionWeightMin + sigmoidValue*(distributionWeightMax-distributionWeightMin)
	return distributionWeight, 1 - distributionWeight // weight(D)+weight(A) = 1
}

func (c *AdaptiveConfigurator) adaptWeights(distributionWeight float64, affinityWeight float64) {
	for name, profile := range c.profiles {
		profileScorersByCategory := c.profileScorersByCategory[name]
		profileSumOfWeightsByCategory := c.profileSumOfWeightsByCategory[name]

		// Calculate scaling factors safely
		distributionSum, distributionOk := profileSumOfWeightsByCategory[framework.Distribution]
		distributionScorersFactor := 0.0
		if distributionOk && distributionSum > 0 {
			distributionScorersFactor = distributionWeight / distributionSum
		}

		affinitySum, affinityOk := profileSumOfWeightsByCategory[framework.Affinity]
		affinityScorersFactor := 0.0
		if affinityOk && affinitySum > 0 {
			affinityScorersFactor = affinityWeight / affinitySum
		}

		// TODO // Lock profile while updating scorers weights, to avoid inconsistencies when handling a request.
		if profile != nil {
			// TODO profile lock. no need for the if, it's just to avoid compile errors
		}
		c.applyScorerWeights(profileScorersByCategory[framework.Distribution], distributionScorersFactor)
		c.applyScorerWeights(profileScorersByCategory[framework.Affinity], affinityScorersFactor)
		// TODO unlock at this point, no need to wait
		// Update sum of weights after scaling
		if distributionOk {
			profileSumOfWeightsByCategory[framework.Distribution] = distributionWeight
		}
		if affinityOk {
			profileSumOfWeightsByCategory[framework.Affinity] = affinityWeight
		}
	}
}

func (c *AdaptiveConfigurator) applyScorerWeights(scorers []*framework.WeightedScorer, categoryWeightFactor float64) {
	if categoryWeightFactor <= 0 {
		return // we might have a case where scorer is applied with no weight, don't adapt its weight.
	}

	for _, scorer := range scorers {
		newWeight := float64(scorer.Weight()) * categoryWeightFactor // TODO remove after weight changes to float
		// TODO update scorer with new weight
		if newWeight > 0 { // a placeholder so it compiles, no need for the if
		}
	}
}

// Sigmoid is the "logistic function". generalized sigmoid function with s0 and k where s0 represents the
// midpoint (the signal r where sigmoid(r) = 0.5) and k represents the slope/steepness of the sigmoid function.
// Higher k means more aggressive slope/steepness.
// - input is expected to be in [0,1]
// - s0 must be in [0,1] range and represents the midpoint
// - k must be > 0; larger values increase steepness
func sigmoid(input float64, s0 float64, k float64) float64 {
	return 1.0 / (1.0 + math.Exp(-k*(input-s0)))
}
