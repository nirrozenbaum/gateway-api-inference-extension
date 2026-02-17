#!/bin/bash

TOKEN=$(kubectl get secret inference-gateway-sa-metrics-reader-secret  -o jsonpath='{.secrets[0].name}' -o jsonpath='{.data.token}' | base64 --decode)

while true; do
  echo "======"
  curl -s -H "Authorization: Bearer $TOKEN" localhost:9090/metrics | grep adaptive | grep -v '^# ' | grep -v 'Handling'
  sleep 1
done