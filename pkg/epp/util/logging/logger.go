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

package logging

import (
	"context"

	"github.com/go-logr/logr"
	uberzap "go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// NewTestLogger creates a new Zap logger using the dev mode.
func NewTestLogger() logr.Logger {
	return zap.New(zap.UseDevMode(true), zap.RawZapOpts(uberzap.AddCaller()))
}

// NewTestLoggerIntoContext creates a new Zap logger using the dev mode and inserts it into the given context.
func NewTestLoggerIntoContext(ctx context.Context) context.Context {
	return log.IntoContext(ctx, zap.New(zap.UseDevMode(true), zap.RawZapOpts(uberzap.AddCaller())))
}
