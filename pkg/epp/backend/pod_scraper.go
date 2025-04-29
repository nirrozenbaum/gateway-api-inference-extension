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
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	logutil "sigs.k8s.io/gateway-api-inference-extension/pkg/epp/util/logging"
)

const (
	scrapeTimeout = 5 * time.Second
)

type PodScraper interface {
	Stop()
}

func newPodScraper(pod *corev1.Pod, scrapers map[Scraper]time.Duration) *podScraper {
	return &podScraper{
		pod:      toInternalPod(pod),
		scrapers: scrapers,
		once:     sync.Once{},
	}
}

type podScraper struct {
	pod      *Pod
	scrapers map[Scraper]time.Duration

	once          sync.Once // ensure the start function is called only once.
	cancelCtxFunc context.CancelFunc
	logger        logr.Logger
}

func (s *podScraper) start(ctx context.Context) {
	// The returned context's Done channel is closed when the returned cancel function is called or when the parent context's Done channel is closed, whichever happens first.
	newCtx, cancel := context.WithCancel(ctx)
	s.cancelCtxFunc = cancel
	s.logger = log.FromContext(ctx).WithValues("pod", s.pod.NamespacedName)

	s.once.Do(func() {
		for scraper, interval := range s.scrapers { // TODO should be consolidated into a single go routine for all scrapers
			go s.startScrapeLoop(newCtx, scraper, interval)
		}
	})
}

func (s *podScraper) startScrapeLoop(ctx context.Context, scraper Scraper, interval time.Duration) {
	s.logger.V(logutil.DEFAULT).Info("Starting scraper", "name", scraper.Name())
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C: // scrape periodically
			ctxWithTimeout, cancel := context.WithTimeout(ctx, scrapeTimeout)
			scrapeResult, err := scraper.Scrape(ctxWithTimeout, s.pod)
			cancel() // cancels only ctxWithTimeout, doesn't cancel ctx
			if err != nil {
				s.logger.V(logutil.TRACE).Error(err, "Failed to scrape", "name", scraper.Name())
			}
			scraper.ProcessResult(scrapeResult)
		}
	}
}

func (s *podScraper) Stop() {
	s.logger.V(logutil.DEFAULT).Info("Stopping scrape loop")
	s.cancelCtxFunc()
}

func toInternalPod(in *corev1.Pod) *Pod {
	return &Pod{
		NamespacedName: types.NamespacedName{
			Name:      in.Name,
			Namespace: in.Namespace,
		},
		Address: in.Status.PodIP,
	}
}
