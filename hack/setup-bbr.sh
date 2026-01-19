#!/bin/bash

# Copyright 2025 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

SCRIPT_ROOT=$(dirname "${BASH_SOURCE}")

## cd to GIE root directory
cd ${SCRIPT_ROOT}/..

## Setup vllm-sim
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api-inference-extension/raw/main/config/manifests/bbr/sim-deployment.yaml

## Deploy Inference Pool and EPP
IGW_CHART_VERSION=v0
GATEWAY_PROVIDER=istio
helm install vllm-deepseek-r1 \
--dependency-update \
--set inferencePool.modelServers.matchLabels.app=vllm-deepseek-r1 \
--set provider.name=$GATEWAY_PROVIDER \
--set inferenceExtension.flags.v=3 \
--set experimentalHttpRoute.enabled=true \
--set experimentalHttpRoute.baseModel=deepseek/vllm-deepseek-r1 \
--version $IGW_CHART_VERSION \
oci://us-central1-docker.pkg.dev/k8s-staging-images/gateway-api-inference-extension/charts/inferencepool

## Deploy BBR
helm install bbr \
--set provider.name=$GATEWAY_PROVIDER \
--version $IGW_CHART_VERSION \
oci://us-central1-docker.pkg.dev/k8s-staging-images/gateway-api-inference-extension/charts/body-based-routing

## appy configmap for adapters
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api-inference-extension/raw/main/config/manifests/bbr/configmap.yaml

## Upgrade Inference Pool and EPP
helm upgrade vllm-llama3-8b-instruct oci://us-central1-docker.pkg.dev/k8s-staging-images/gateway-api-inference-extension/charts/inferencepool \
--dependency-update \
--set inferencePool.modelServers.matchLabels.app=vllm-llama3-8b-instruct \
--set provider.name=$GATEWAY_PROVIDER \
--set inferenceExtension.flags.v=3 \
--set experimentalHttpRoute.enabled=true \
--set experimentalHttpRoute.baseModel=meta-llama/Llama-3.1-8B-Instruct \
--version $IGW_CHART_VERSION
