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
	"context"
	"math"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"

	logutil "sigs.k8s.io/gateway-api-inference-extension/pkg/common/util/logging"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/asyncdetector"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/framework/interface/plugin"
	framework "sigs.k8s.io/gateway-api-inference-extension/pkg/epp/framework/interface/scheduling"
)

// NewAdaptiveConfigurator creates a new AdaptiveConfigurator object and returns its pointer.
func NewAdaptiveConfigurator(imbalanceDetector *asyncdetector.ImbalanceDetector,
	pressureDetector *asyncdetector.PressureDetector, interval time.Duration,
	schedulerConfig *SchedulerConfig, config *AdaptiveConfiguratorConfig) *AdaptiveConfigurator {
	profilesData := make(map[string]*ProfileData)
	for name, profile := range schedulerConfig.profiles {
		profilesData[name] = createProfileData(profile)
	}

	adaptiveConfigurator := &AdaptiveConfigurator{
		imbalanceDetector:     imbalanceDetector,
		pressureDetector:      pressureDetector,
		interval:              interval,
		profiles:              schedulerConfig.profiles,
		profilesData:          profilesData,
		distributionMinWeight: config.distributionMinWeight,
		distributionMaxWeight: config.distributionMaxWeight,
		conservativeMaxWeight: config.conservativeMaxWeight,
		sigmoidS0:             config.sigmoidS0,
		sigmoidK:              config.sigmoidK,
	}
	// we tune the starting point to minimal distribution, more weight for affinity scorers
	adaptiveConfigurator.adaptWeights(config.distributionMinWeight, 1-config.distributionMinWeight)

	return adaptiveConfigurator
}

type AdaptiveConfigurator struct {
	imbalanceDetector     *asyncdetector.ImbalanceDetector
	pressureDetector      *asyncdetector.PressureDetector
	interval              time.Duration
	profiles              map[string]*framework.SchedulerProfile // keep the profiles for locking
	profilesData          map[string]*ProfileData
	distributionMinWeight float64
	distributionMaxWeight float64
	conservativeMaxWeight float64
	sigmoidS0             float64
	sigmoidK              float64
}

func (c *AdaptiveConfigurator) Start(ctx context.Context) error {
	log.FromContext(ctx).Info("Starting adaptive configurator")
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C: // adapt scorers weights periodically
			c.adapt(ctx)
		}
	}
}

func (c *AdaptiveConfigurator) adapt(ctx context.Context) {
	distributionWeight, affinityWeight := c.calculateDesiredWeights()
	log.FromContext(ctx).V(logutil.VERBOSE).Info("adapting scorers weights", "distribution-weight", distributionWeight,
		"affinity-weight", affinityWeight)

	c.adaptWeights(distributionWeight, affinityWeight)
}

func (c *AdaptiveConfigurator) adaptWeights(distributionWeight float64, affinityWeight float64) {
	for name, profile := range c.profiles {
		profileData := c.profilesData[name]
		updatedWeightsMap := map[plugin.TypedName]float64{} // map from scorer TypedName to its new weight
		sumOfOriginalWeightsByCategory := profileData.sumOfOriginalWeightsByCategory

		// Calculate scaling factors safely
		distributionSum, distributionOk := sumOfOriginalWeightsByCategory[framework.Distribution]
		if distributionOk && distributionSum > 0 {
			distributionScorersFactor := distributionWeight / distributionSum
			distributionScorers := profileData.scorersByCategory[framework.Distribution]
			for _, scorer := range distributionScorers {
				originalWeight := profileData.originalScorersWeights[scorer]
				updatedWeightsMap[scorer.TypedName()] = originalWeight * distributionScorersFactor
			}
		}

		affinitySum, affinityOk := sumOfOriginalWeightsByCategory[framework.Affinity]
		if affinityOk && affinitySum > 0 {
			affinityScorersFactor := affinityWeight / affinitySum
			affinityScorers := profileData.scorersByCategory[framework.Affinity]
			for _, scorer := range affinityScorers {
				originalWeight := profileData.originalScorersWeights[scorer]
				updatedWeightsMap[scorer.TypedName()] = originalWeight * affinityScorersFactor
			}
		}
		// UpdateScorersWeights is internally synchronized by SchedulerProfile
		profile.UpdateScorersWeights(updatedWeightsMap)
	}
}

// calculateDesiredWeights returns the desired weights (total budget) for distribution and affinity categories.
func (c *AdaptiveConfigurator) calculateDesiredWeights() (float64, float64) {
	imbalanceRatio := c.imbalanceDetector.SignalRatio()
	sigmoidValue := sigmoid(imbalanceRatio, c.sigmoidS0, c.sigmoidK)

	pressureRatio := c.pressureDetector.SignalRatio()
	// weight(D) = wDmin + (sigmoid(ratio) * (wDmax - wDmin))
	dynamicMaxWeight := c.distributionMaxWeight - pressureRatio*(c.distributionMaxWeight-c.conservativeMaxWeight)
	distributionWeight := c.distributionMinWeight + sigmoidValue*(dynamicMaxWeight-c.distributionMinWeight)
	return distributionWeight, 1 - distributionWeight // weight(D)+weight(A) = 1
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
