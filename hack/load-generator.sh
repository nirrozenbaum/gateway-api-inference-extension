#!/usr/bin/env bash

set -euo pipefail

running=true

trap 'echo "Received SIGTERM, shutting down..."; running=false' SIGTERM SIGINT
IP=localhost
PORT=30080

while $running; do
  curl -i ${IP}:${PORT}/v1/completions -H 'Content-Type: application/json' -d '{
    "model": "food-review-1",
    "prompt": "Write as if you were a critic: San Francisco",
    "max_tokens": 1000,
    "temperature": 0
    }'
done

echo "Cleanup done. Exiting."