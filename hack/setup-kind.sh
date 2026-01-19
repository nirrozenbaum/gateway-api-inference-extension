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

KUBERNETES_VERSION=v1.27.3
KIND_NODE_IMAGE=kindest/node:${KUBERNETES_VERSION}

# Set a default CLUSTER_NAME if not provided
CLUSTER_NAME="test"

# Set the host port to map to the Gateway's inbound port (30080)
GATEWAY_HOST_PORT=30080

IGW_CHART_VERSION=v0

kind create cluster --image ${KIND_NODE_IMAGE} --name "${CLUSTER_NAME}" --config - << EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30080
    hostPort: ${GATEWAY_HOST_PORT}
    protocol: TCP
EOF

set -x

# Hotfix for https://github.com/kubernetes-sigs/kind/issues/3880
CONTAINER_NAME="${CLUSTER_NAME}-control-plane"
docker exec -it ${CONTAINER_NAME} /bin/bash -c "sysctl net.ipv4.conf.all.arp_ignore=0"

kubectl apply -f - << EOF
apiVersion: v1
kind: Service
metadata:
  annotations:
    networking.istio.io/service-type: NodePort
  labels:
    gateway.istio.io/managed: istio.io-gateway-controller
    gateway.networking.k8s.io/gateway-name: inference-gateway
    istio.io/enable-inference-extproc: "true"
  name: inference-gateway-istio-nodeport
spec:
  type: NodePort
  selector:
    gateway.networking.k8s.io/gateway-name: inference-gateway
  ports:
  - appProtocol: tcp
    name: status-port
    port: 15021
    protocol: TCP
    targetPort: 15021
    nodePort: 32021
  - appProtocol: http
    name: default
    port: 80
    protocol: TCP
    targetPort: 80
    nodePort: 30080
EOF

## Setup vllm-sim
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api-inference-extension/refs/tags/v1.2.0/config/manifests/vllm/sim-deployment.yaml

## Deploy GIE CRDs
kubectl apply -k https://github.com/kubernetes-sigs/gateway-api-inference-extension/config/crd

## Deploy Gateway API CRDs
kubectl apply --server-side -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.0/standard-install.yaml

## Install Istio on macOS
ISTIO_VERSION=1.28.1
curl -L https://istio.io/downloadIstio | ISTIO_VERSION=${ISTIO_VERSION} sh -
./istio-$ISTIO_VERSION/bin/istioctl install \
   --set values.pilot.env.ENABLE_GATEWAY_API_INFERENCE_EXTENSION=true

rm -r ./istio-$ISTIO_VERSION

## Deploy an Inference Gateway
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api-inference-extension/raw/main/config/manifests/gateway/istio/gateway.yaml

## Deploy Inference Pool and EPP
export GATEWAY_PROVIDER=istio
helm install vllm-llama3-8b-instruct \
--dependency-update \
--set inferencePool.modelServers.matchLabels.app=vllm-llama3-8b-instruct \
--set provider.name=$GATEWAY_PROVIDER \
--set inferenceExtension.flags.v=3 \
--set experimentalHttpRoute.enabled=true \
--version $IGW_CHART_VERSION \
oci://us-central1-docker.pkg.dev/k8s-staging-images/gateway-api-inference-extension/charts/inferencepool
