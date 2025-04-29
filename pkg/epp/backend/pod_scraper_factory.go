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
	"time"

	corev1 "k8s.io/api/core/v1"
)

func NewPodScraperFactory(scrapers map[Scraper]time.Duration) *PodScraperFactory {
	return &PodScraperFactory{
		scrapers: scrapers,
	}
}

type PodScraperFactory struct {
	scrapers map[Scraper]time.Duration // map from scraper to its refresh interval. different scrapers may different different intervals
}

func (f *PodScraperFactory) NewPodScraper(ctx context.Context, pod *corev1.Pod, ds Datastore) *podScraper {
	podScraper := newPodScraper(pod, f.scrapers)
	podScraper.start(ctx)
	return podScraper
	// pm := &podMetrics{
	// 	pmc:      f.pmc,
	// 	ds:       ds,
	// 	interval: f.refreshMetricsInterval,
	// 	once:     sync.Once{},
	// 	done:     make(chan struct{}),
	// 	logger:   log.FromContext(parentCtx).WithValues("pod", pod.NamespacedName),
	// }
	// pm.pod.Store(pod)
	// pm.metrics.Store(newMetrics())

	// pm.startRefreshLoop(parentCtx)
	// return pm
}
