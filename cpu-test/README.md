## CPU based vLLM for inference 

### Deploy

1.  To see logs of the inference pod:
    ```bash
    kubectl logs -f $(kubectl get pods -l app=vllm-cpu-test -o jsonpath='{.items[*].metadata.name}')
    ```

### Test

1.  To test that things are working, start a new pod to be used as client:
    ```bash
    kubectl run -i --tty --rm test-cpu-vllm --image=badouralix/curl-jq --restart=Never -- /bin/sh
    ```

1.  To see the deployed models run the following:
    ```bash
    curl http://vllm-cpu-test.default.svc.cluster.local:5678/v1/models | jq
    ```

1.  To run `chat/completions` request run the following:
    ```bash
    curl http://vllm-cpu-test.default.svc.cluster.local:5678/v1/chat/completions -H "Content-Type: application/json" \
         -d '{
              "model": "tweet-summary-1",
              "messages": [
                {
                  "role": "system",
                  "content": "You are a helpful assistant."
                },
                {
                  "role": "user",
                  "content": "write a simple python random generator"
                }
              ]
            }' | jq
    ```

1. To run `completions` request run the following:
    ```bash
    curl http://vllm-cpu-test.default.svc.cluster.local:5678/v1/completions -H "Content-Type: application/json" \
         -d '{
              "model": "tweet-summary-1",
              "prompt": "Write as if you were a critic: San Francisco",
              "max_tokens": 100,
              "temperature": 0
            }' | jq
    ```
