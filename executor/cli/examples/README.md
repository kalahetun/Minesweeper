# Policy Examples

This directory contains example fault injection policies for BOIFI (Bayesian Optimized Intelligent Fault Injection).

## Available Examples

### Basic Fault Types

| File | Description | Match | Effect |
|------|-------------|-------|--------|
| `abort-policy.yaml` | Abort 故障 | `GET /` | 返回 HTTP 503 |
| `delay-policy.yaml` | Delay 故障 | `GET /` | 延迟 1 秒 |
| `percentage-policy.yaml` | 概率故障 | `GET /` | 50% 概率延迟 500ms |
| `header-policy.yaml` | Header 匹配 | `GET /` + `x-user-id: test` | 延迟 800ms |

### Advanced Time Control

| File | Description | start_delay_ms | duration_seconds |
|------|-------------|----------------|------------------|
| `immediate-fault-policy.yaml` | 立即+永久 | 0 | 0 (永久) |
| `time-limited-fault-policy.yaml` | 立即+自动过期 | 0 | 300 (5分钟) |
| `late-stage-fault-policy.yaml` | 延迟执行+永久 | 500 | 0 |
| `delayed-timed-fault-policy.yaml` | 延迟执行+自动过期 | 200 | 60 |

### Testing

| File | Description |
|------|-------------|
| `invalid-policy.yaml` | 无效策略示例（缺少 name 字段） |

## Time Control Fields

### `start_delay_ms` (请求级延迟)

请求到达后等待指定毫秒数再注入故障。模拟 "late-stage" 故障场景。

```yaml
fault:
  start_delay_ms: 500  # 请求到达 500ms 后才注入故障
  abort:
    httpStatus: 503
```

### `duration_seconds` (策略过期时间)

策略创建后的有效时间（秒）。过期后不再注入故障。

```yaml
fault:
  duration_seconds: 300  # 5 分钟后自动过期
  delay:
    fixed_delay: "500ms"
```

## Usage

```bash
# Apply a policy
hfi-cli policy apply -f examples/abort-policy.yaml

# List applied policies
hfi-cli policy list

# Get policy details
hfi-cli policy get <policy-name>

# Delete a policy
hfi-cli policy delete <policy-name>

# Apply with time override
hfi-cli policy apply -f examples/abort-policy.yaml --duration-seconds 60
hfi-cli policy apply -f examples/abort-policy.yaml --start-delay-ms 1000
```

## Testing

```bash
# Test abort policy
curl -w "\nStatus: %{http_code}\n" http://localhost:18000/

# Test delay policy (measure time)
time curl http://localhost:18000/

# Test header-based policy
curl -H "x-user-id: test" http://localhost:18000/

# Test percentage policy (run multiple times)
for i in {1..10}; do curl -s -o /dev/null -w "%{http_code}\n" http://localhost:18000/; done
```

## Policy Structure

```yaml
metadata:
  name: "policy-name"           # Required: unique policy name
spec:
  selector:                     # Optional: service targeting (Istio/K8s only)
    service: "frontend"         # Target specific service ("*" for all)
    namespace: "demo"           # Target specific namespace ("*" for all)
  rules:
    - match:
        method:
          exact: "GET"          # HTTP method
        path:
          exact: "/api/users"   # or prefix: "/api"
        headers:                # Optional: header matchers
          - name: "x-user-id"
            exact: "test"       # or prefix: "test-"
      fault:
        percentage: 100         # 0-100, probability of fault injection
        start_delay_ms: 0       # Optional: wait before injecting fault
        duration_seconds: 0     # Optional: policy expiration (0 = never)
        abort:                  # Fault type 1: return error
          httpStatus: 503
        delay:                  # Fault type 2: add latency
          fixed_delay: "1000ms"
```

## Service Selector (Istio/K8s Deployments)

### Target Specific Services

```yaml
metadata:
  name: checkout-abort
spec:
  selector:
    service: checkoutservice    # Only affects this service
    namespace: demo
  rules:
    - match:
        path:
          prefix: /hipstershop.CheckoutService/
      fault:
        percentage: 25
        abort:
          httpStatus: 503
```

### Apply to All Services (Wildcard)

```yaml
metadata:
  name: global-delay
spec:
  # Omit selector entirely, OR use wildcards:
  selector:
    service: "*"          # All services
    namespace: "*"        # All namespaces
  rules:
    - match:
        path:
          prefix: /
      fault:
        percentage: 10
        delay:
          fixed_delay: "200ms"
```

### Selector Behavior

| Selector | Behavior |
|----------|----------|
| Omitted | Applies to ALL services (backward compatible) |
| `service: "frontend"` | Only applies to pods with `app=frontend` label |
| `namespace: "demo"` | Only applies to pods in `demo` namespace |
| Both specified | Must match BOTH service AND namespace |
| Wildcards (`*`) | Explicit "apply to all" (same as omitting) |

**Note**: Service selector uses Kubernetes workload labels (`app`, `version`) extracted from Envoy node metadata. Only works with Istio-injected pods.

## Best Practices

1. **Start small**: Begin with low percentages (10-20%) in production
2. **Use expiration**: Set `duration_seconds` for experimental policies
3. **Target carefully**: Use `selector` to avoid affecting unintended services
4. **Test locally**: Validate in Docker Compose before Kubernetes
5. **Monitor**: Check Envoy logs for fault injection confirmations
6. **Clean up**: Remove test policies after experiments

---

## Observing Metrics After Policy Application

### Quick Metrics Check

After applying a policy, you can observe the metrics to verify fault injection is working:

```bash
# 1. Apply a policy (e.g., abort policy)
./hfi-cli policy apply -f examples/abort-policy.yaml

# 2. Get a pod name with Istio sidecar (if using Kubernetes)
POD=$(kubectl get pod -n demo -l app=frontend -o jsonpath='{.items[0].metadata.name}')

# 3. Query Envoy stats for HFI metrics
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s http://localhost:15090/stats/prometheus | grep wasmcustom_hfi_faults

# Expected output:
# wasmcustom_hfi_faults_aborts_total 0      ← Before traffic
# wasmcustom_hfi_faults_delays_total 0
# wasmcustom_hfi_faults_delay_duration_milliseconds_count 0
```

### Generate Traffic and Observe Changes

```bash
# Generate 20 requests
for i in {1..20}; do
  kubectl exec -n demo $POD -c server -- curl -s http://localhost:8080/ > /dev/null
  sleep 0.5
done

# Check metrics again
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s http://localhost:15090/stats/prometheus | grep "wasmcustom_hfi_faults_aborts_total"

# Expected output (if abort-policy.yaml has 50% probability):
# wasmcustom_hfi_faults_aborts_total 10    ← Increased (roughly 50% of 20)
```

### Metric Examples by Policy Type

#### Abort Policy (`abort-policy.yaml`)

Increments `aborts_total` counter:

```bash
# Before: wasmcustom_hfi_faults_aborts_total 0
# After 20 requests at 50%: wasmcustom_hfi_faults_aborts_total 10
```

#### Delay Policy (`delay-policy.yaml`)

Increments `delays_total` counter and updates histogram:

```bash
# Before: 
# wasmcustom_hfi_faults_delays_total 0
# wasmcustom_hfi_faults_delay_duration_milliseconds_sum 0

# After 10 requests with 1000ms delay:
# wasmcustom_hfi_faults_delays_total 10
# wasmcustom_hfi_faults_delay_duration_milliseconds_sum 10000
# wasmcustom_hfi_faults_delay_duration_milliseconds_bucket{le="1000"} 10
```

### Available Metrics

| Metric | Type | Updated By |
|--------|------|------------|
| `wasmcustom_hfi_faults_aborts_total` | Counter | Any policy with `abort` fault |
| `wasmcustom_hfi_faults_delays_total` | Counter | Any policy with `delay` fault |
| `wasmcustom_hfi_faults_delay_duration_milliseconds` | Histogram | Delay faults (tracks duration distribution) |

### Prometheus Query Examples

If Prometheus is scraping your Istio sidecars:

```promql
# Rate of abort faults per second
rate(wasmcustom_hfi_faults_aborts_total[1m])

# Total delays in last 5 minutes
increase(wasmcustom_hfi_faults_delays_total[5m])

# 95th percentile of delay duration
histogram_quantile(0.95, 
  rate(wasmcustom_hfi_faults_delay_duration_milliseconds_bucket[5m]))

# Percentage of requests with faults (if you have total request count)
rate(wasmcustom_hfi_faults_aborts_total[1m]) / 
  rate(istio_requests_total{app="frontend"}[1m]) * 100
```

### Troubleshooting

**Metrics not appearing?**
1. Check Wasm plugin loaded: `kubectl logs -n demo $POD -c istio-proxy | grep wasm`
2. Verify policy applied: `./hfi-cli policy list`
3. Confirm traffic reaching pod: `kubectl logs -n demo $POD -c server`
4. See [../../k8s/METRICS_SOLUTION.md](../../k8s/METRICS_SOLUTION.md) for detailed troubleshooting

**Counters not incrementing?**
- Verify policy `selector` matches your pod (service/namespace)
- Check policy `match` rules align with your request (path, method, headers)
- Look for "fault injected" messages in Envoy logs

