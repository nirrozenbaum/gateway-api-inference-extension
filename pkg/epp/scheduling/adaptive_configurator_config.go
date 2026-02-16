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
	"fmt"
)

// NewAdaptiveConfiguratorConfig creates a new AdaptiveConfiguratorConfig object and returns its pointer.
func NewAdaptiveConfiguratorConfig(distributionMinWeight float64, distributionMaxWeight float64,
	conservativeMaxWeight float64, sigmoidS0 float64, sigmoidK float64) *AdaptiveConfiguratorConfig {
	// TODO: validate config invariants:
	// - 0 <= distributionMinWeight < distributionMaxWeight <= 1
	// - 0 < sigmoidS0 && sigmoidS0 < 1
	// - sigmoidK > 0
	return &AdaptiveConfiguratorConfig{
		distributionMinWeight: distributionMinWeight,
		distributionMaxWeight: distributionMaxWeight,
		conservativeMaxWeight: conservativeMaxWeight,
		sigmoidS0:             sigmoidS0,
		sigmoidK:              sigmoidK,
	}
}

// SchedulerConfig provides a configuration for the scheduler which influence routing decisions.
type AdaptiveConfiguratorConfig struct {
	distributionMinWeight float64
	distributionMaxWeight float64
	conservativeMaxWeight float64
	sigmoidS0             float64 // s0 is midpoint, the signal r where sigmoid(r) = 0.5
	sigmoidK              float64 // represents the slope/steepness of the sigmoid function. higher k is more aggressive.
}

func (c *AdaptiveConfiguratorConfig) String() string {
	return fmt.Sprintf(
		"{distributionMinWeight: %f, distributionMaxWeight: %f, conservativeMaxWeight: %f, sigmoidS0: %f, sigmoidK: %f}",
		c.distributionMinWeight,
		c.distributionMaxWeight,
		c.conservativeMaxWeight,
		c.sigmoidS0,
		c.sigmoidK,
	)
}
