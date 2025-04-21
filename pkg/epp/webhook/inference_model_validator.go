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

package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/gateway-api-inference-extension/api/v1alpha2"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/datastore"
)

// InferenceModelValidator validates the uniqueness of .spec.modelName field of InferenceModel objects
// that are associated with an InferencePool.
type InferenceModelValidator struct {
	Datastore datastore.Datastore
	PoolName  string
}

// validate verify uniqueness of InferenceModel.spec.modelName within a pool.
func (v *InferenceModelValidator) validate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	logger := log.FromContext(ctx)
	infModel, ok := obj.(*v1alpha2.InferenceModel)
	if !ok {
		return nil, fmt.Errorf("Expected InferenceModel object but got a %T", obj)
	}

	logger.Info("Validating InferenceModel spec.modelName")
	// namespaced objects are filtered in server side by the controller-manager, no need to check for namespace
	if infModel.Spec.PoolRef.Name != v1alpha2.ObjectName(v.PoolName) {
		return nil, nil // skip validation, the object is not associated with the pool
	}

	modelName := infModel.Spec.ModelName
	if existingModel := v.Datastore.ModelGet(modelName); existingModel != nil {
		return nil, fmt.Errorf("InferenceModel with ModelName '%s' already exist in the InferencePool", modelName)
	}

	return nil, nil
}

func (v *InferenceModelValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, obj)
}

func (v *InferenceModelValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return v.validate(ctx, newObj)
}

func (v *InferenceModelValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil // no validation on delete
}

func (v *InferenceModelValidator) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha2.InferenceModel{}).
		WithValidator(v).
		Complete()
}
