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

// NewAdaptiveConfigurator creates a new AdaptiveConfigurator object and returns its pointer.
func NewAdaptiveConfigurator(imbalanceDetector *asyncdetector.ImbalanceDetector, schedulerConfig *SchedulerConfig,
	config *AdaptiveConfiguratorConfig) *AdaptiveConfigurator {
	profilesData := make(map[string]*ProfileData)
	for name, profile := range schedulerConfig.profiles {
		profilesData[name] = createProfileData(profile)
	}

	adaptiveConfigurator := &AdaptiveConfigurator{
		imbalanceDetector:     imbalanceDetector,
		profiles:              schedulerConfig.profiles,
		profilesData:          profilesData,
		distributionMinWeight: config.distributionMinWeight,
		distributionMaxWeight: config.distributionMaxWeight,
		sigmoidS0:             config.sigmoidS0,
		sigmoidK:              config.sigmoidK,
	}
	// we tune the starting point to minimal distribution, more weight for affinity scorers
	adaptiveConfigurator.adaptWeights(config.distributionMinWeight, 1-config.distributionMinWeight)

	return adaptiveConfigurator
}

type AdaptiveConfigurator struct {
	imbalanceDetector     *asyncdetector.ImbalanceDetector
	profiles              map[string]*framework.SchedulerProfile // keep the profiles for locking
	profilesData          map[string]*ProfileData
	distributionMinWeight float64
	distributionMaxWeight float64
	sigmoidS0             float64
	sigmoidK              float64
}

// TODO should run periodically
func (c *AdaptiveConfigurator) Run() {
	distributionWeight, affinityWeight := c.calculateDesiredWeights()
	c.adaptWeights(distributionWeight, affinityWeight)
}

// calculateDesiredWeights returns the desired weights (total budget) for distribution and affinity categories.
func (c *AdaptiveConfigurator) calculateDesiredWeights() (float64, float64) {
	imbalanceRatio := c.imbalanceDetector.SignalRatio()
	sigmoidValue := sigmoid(imbalanceRatio, c.sigmoidS0, c.sigmoidK)

	// weight(D) = wDmin + (sigmoid(ratio) * (wDmax - wDmin))
	distributionWeight := c.distributionMinWeight + sigmoidValue*(c.distributionMaxWeight-c.distributionMinWeight)
	return distributionWeight, 1 - distributionWeight // weight(D)+weight(A) = 1
}

func (c *AdaptiveConfigurator) adaptWeights(distributionWeight float64, affinityWeight float64) {
	for name, profile := range c.profiles {
		scorersByCategory := c.profilesData[name].scorersByCategory
		sumOfOriginalWeightsByCategory := c.profilesData[name].sumOfOriginalWeightsByCategory

		// Calculate scaling factors safely
		distributionSum, distributionOk := sumOfOriginalWeightsByCategory[framework.Distribution]
		distributionScorersFactor := 0.0
		if distributionOk && distributionSum > 0 {
			distributionScorersFactor = distributionWeight / distributionSum
		}

		affinitySum, affinityOk := sumOfOriginalWeightsByCategory[framework.Affinity]
		affinityScorersFactor := 0.0
		if affinityOk && affinitySum > 0 {
			affinityScorersFactor = affinityWeight / affinitySum
		}

		// TODO // Lock profile while updating scorers weights, to avoid inconsistencies when handling a request.
		if profile != nil {
			// TODO profile lock. no need for the if, it's just to avoid compile errors
		}
		c.applyScorerWeights(c.profilesData[name], scorersByCategory[framework.Distribution], distributionScorersFactor)
		c.applyScorerWeights(c.profilesData[name], scorersByCategory[framework.Affinity], affinityScorersFactor)
		// TODO unlock profile here
	}
}

func (c *AdaptiveConfigurator) applyScorerWeights(profileData *ProfileData, scorers []*framework.WeightedScorer, categoryWeightFactor float64) {
	if categoryWeightFactor <= 0 {
		return // we might have a case where scorers are applied with no weight, don't adapt.
	}

	for _, scorer := range scorers {
		originalWeight := profileData.originalScorersWeights[scorer]
		newWeight := originalWeight * categoryWeightFactor
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
