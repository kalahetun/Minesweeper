# Quickstart: Wasm Metrics Exposure

**Feature**: 008-wasm-metrics-exposure  
**Audience**: Platform operators deploying and verifying HFI metrics  
**Time**: 5-10 minutes

## Prerequisites

- Kubernetes cluster with Istio 1.20+ installed
- HFI system deployed (Feature 007 complete)
- kubectl access to demo namespace
- Basic understanding of Prometheus metrics

## Overview

This feature exposes three custom Prometheus metrics from the HFI Wasm plugin:

| Metric Name | Type | Purpose |
|-------------|------|---------|
| `wasmcustom.hfi_faults_aborts_total` | Counter | Count of abort faults injected |
| `wasmcustom.hfi_faults_delays_total` | Counter | Count of delay faults injected |
| `wasmcustom.hfi_faults_delay_duration_milliseconds` | Histogram | Distribution of delay durations |

**Dual Mechanism**: Metrics are exposed via both naming convention (wasmcustom prefix) and EnvoyFilter configuration for maximum reliability.

---

## Step 1: Deploy Updated Wasm Plugin

The Wasm plugin with updated metric names is included in the standard HFI deployment.

```bash
# Verify WasmPlugin is deployed
kubectl get wasmplugin -n demo

# Expected output:
# NAME                  AGE
# boifi-fault-injection 1d
```

If not deployed, follow the main HFI deployment guide in `executor/k8s/README.md`.

---

## Step 2: Apply EnvoyFilter (Optional but Recommended)

The EnvoyFilter ensures metrics are exposed even if the automatic wasmcustom prefix mechanism doesn't work.

```bash
# Apply EnvoyFilter
kubectl apply -f executor/k8s/envoyfilter-wasm-stats.yaml

# Verify EnvoyFilter is created
kubectl get envoyfilter -n demo

# Expected output:
# NAME              AGE
# hfi-wasm-metrics  10s
```

**Note**: EnvoyFilter changes require pod restart to take effect (BOOTSTRAP patch).

```bash
# Restart pods to apply EnvoyFilter
kubectl rollout restart deployment -n demo

# Wait for pods to be ready
kubectl wait --for=condition=ready pod -l hfi-enabled=true -n demo --timeout=60s
```

---

## Step 3: Verify Metrics in Envoy Stats

Check that metrics appear in Envoy's stats endpoint.

```bash
# Get a pod name
POD=$(kubectl get pod -n demo -l app=frontend -o jsonpath='{.items[0].metadata.name}')

# Query Envoy stats endpoint
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s http://localhost:15090/stats/prometheus | grep wasmcustom_hfi_faults

# Expected output (before any faults):
# wasmcustom_hfi_faults_aborts_total 0
# wasmcustom_hfi_faults_delays_total 0
# wasmcustom_hfi_faults_delay_duration_milliseconds_bucket{le="50"} 0
# wasmcustom_hfi_faults_delay_duration_milliseconds_bucket{le="100"} 0
# wasmcustom_hfi_faults_delay_duration_milliseconds_bucket{le="250"} 0
# wasmcustom_hfi_faults_delay_duration_milliseconds_bucket{le="500"} 0
# wasmcustom_hfi_faults_delay_duration_milliseconds_bucket{le="1000"} 0
# wasmcustom_hfi_faults_delay_duration_milliseconds_bucket{le="+Inf"} 0
# wasmcustom_hfi_faults_delay_duration_milliseconds_sum 0
# wasmcustom_hfi_faults_delay_duration_milliseconds_count 0
```

**Troubleshooting**: If metrics don't appear:
1. Check EnvoyFilter status: `kubectl describe envoyfilter hfi-wasm-metrics -n demo`
2. Verify pod restarted after EnvoyFilter: `kubectl get pod -n demo -o wide`
3. Check Envoy logs: `kubectl logs -n demo $POD -c istio-proxy | grep -i metric`

---

## Step 4: Trigger Fault Injection

Apply a fault injection policy to generate metric data.

```bash
# Apply abort policy (50% failure rate)
cd executor/cli
./hfi-cli policy apply examples/abort-policy.yaml

# Verify policy is active
./hfi-cli policy list

# Expected output:
# ID                                   NAME          SERVICE   PROBABILITY  FAULT
# abort-policy-xxx                     abort-demo    frontend  0.50         abort(status=503)
```

Generate traffic to trigger faults:

```bash
# Send 20 requests (expect ~10 aborts)
for i in {1..20}; do
  kubectl run curl-test-$i -n demo --image=curlimages/curl --rm -i --restart=Never -- \
    curl -s -o /dev/null -w "%{http_code}\n" http://frontend/
done

# You should see mix of 200 and 503 responses
```

---

## Step 5: Verify Metrics Updated

Check that counters have incremented.

```bash
# Query metrics again
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s http://localhost:15090/stats/prometheus | grep wasmcustom_hfi_faults

# Expected output (after ~10 aborts):
# wasmcustom_hfi_faults_aborts_total 10
# wasmcustom_hfi_faults_delays_total 0
# wasmcustom_hfi_faults_delay_duration_milliseconds_count 0
```

**Apply delay policy to test delay metrics**:

```bash
# Apply delay policy (100ms delays)
./hfi-cli policy apply examples/delay-policy.yaml

# Generate traffic
for i in {1..10}; do
  kubectl run curl-test-delay-$i -n demo --image=curlimages/curl --rm -i --restart=Never -- \
    curl -s http://frontend/ > /dev/null
done

# Check delay metrics
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s http://localhost:15090/stats/prometheus | grep wasmcustom_hfi_faults_delay

# Expected output:
# wasmcustom_hfi_faults_delays_total 10
# wasmcustom_hfi_faults_delay_duration_milliseconds_bucket{le="100"} 10
# wasmcustom_hfi_faults_delay_duration_milliseconds_bucket{le="250"} 10
# wasmcustom_hfi_faults_delay_duration_milliseconds_sum 1000
# wasmcustom_hfi_faults_delay_duration_milliseconds_count 10
```

---

## Step 6: Verify Prometheus Scraping (Optional)

If Prometheus is configured to scrape Istio metrics:

```bash
# Port-forward to Prometheus
kubectl port-forward -n istio-system svc/prometheus 9090:9090 &

# Query metrics via PromQL
curl -s 'http://localhost:9090/api/v1/query?query=wasmcustom_hfi_faults_aborts_total' | jq

# Or open in browser: http://localhost:9090/graph
# Enter query: wasmcustom_hfi_faults_aborts_total
```

**Expected**: Metrics visible with pod labels (namespace, pod, app, etc.)

---

## Verification Checklist

After completing all steps, verify:

- [x] EnvoyFilter applied to demo namespace
- [x] Pods restarted after EnvoyFilter (check AGE column)
- [x] Metrics visible in Envoy stats endpoint (curl localhost:15090/stats/prometheus)
- [x] Abort counter increments when abort policy is active
- [x] Delay counter increments when delay policy is active
- [x] Histogram records delay durations correctly
- [x] Prometheus scrapes metrics (if configured)
- [x] Metric names use wasmcustom.hfi_faults_* format

---

## Common Issues

### Issue 1: Metrics show all zeros

**Cause**: No fault injection policies are active, or policies don't match traffic.

**Solution**:
```bash
# Check active policies
./hfi-cli policy list

# Verify policy matches service
./hfi-cli policy describe <policy-id>

# Check traffic is reaching the service
kubectl logs -n demo deployment/frontend | tail
```

### Issue 2: Metrics not visible at all

**Cause**: EnvoyFilter not applied or pods not restarted.

**Solution**:
```bash
# Check EnvoyFilter exists
kubectl get envoyfilter -n demo

# Force pod restart
kubectl rollout restart deployment -n demo
kubectl wait --for=condition=ready pod -l hfi-enabled=true -n demo --timeout=60s

# Re-check metrics
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s localhost:15090/stats/prometheus | grep wasmcustom
```

### Issue 3: Old metric names (hfi.faults.*) still appear

**Cause**: Old Wasm plugin version still deployed.

**Solution**:
```bash
# Check WasmPlugin image version
kubectl get wasmplugin boifi-fault-injection -n demo -o yaml | grep url

# Expected: Should reference image with wasmcustom metrics

# If old version, update WasmPlugin:
kubectl apply -f executor/k8s/plugin-multi-instance.yaml
kubectl rollout restart deployment -n demo
```

### Issue 4: Prometheus doesn't scrape metrics

**Cause**: Prometheus scrape config doesn't include Istio pods or wasmcustom prefix.

**Solution**:
```bash
# Check Prometheus targets
kubectl port-forward -n istio-system svc/prometheus 9090:9090 &
# Visit: http://localhost:9090/targets
# Look for istio-mesh or kubernetes-pods targets

# If missing, Prometheus config needs update (outside this feature's scope)
```

---

## Next Steps

- **Grafana Dashboard**: Create dashboards visualizing HFI metrics
- **Alerting**: Set up Prometheus alerts for unexpected fault injection rates
- **Production Deployment**: Follow production readiness checklist in main README

---

## Reference

- **Main Documentation**: `executor/k8s/README.md`
- **Policy Examples**: `executor/cli/examples/README.md`
- **Metrics Solution Details**: `executor/k8s/METRICS_SOLUTION.md`
- **E2E Tests**: `executor/k8s/tests/test-07-metrics.sh`
