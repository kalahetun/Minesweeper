# Wasm Metrics Exposure Solution

## Overview

This document explains how the BOIFI Wasm plugin exposes Prometheus metrics and provides troubleshooting guidance for operators.

## Solution Architecture

### Combined Approach

The plugin uses a **combined strategy** for reliable metric exposure:

1. **wasmcustom.* Prefix Naming Convention** (Primary)
   - Envoy automatically exposes metrics with `wasmcustom.` prefix
   - No additional configuration required
   - Works out-of-the-box in Istio deployments

2. **EnvoyFilter Configuration** (Defensive)
   - Optional stats_matcher configuration
   - Provides redundancy in case prefix convention changes
   - Applied at namespace level (BOOTSTRAP patch)

### Exposed Metrics

| Metric Name | Type | Description | Buckets/Labels |
|-------------|------|-------------|----------------|
| `wasmcustom_hfi_faults_aborts_total` | Counter | Total abort faults injected | None |
| `wasmcustom_hfi_faults_delays_total` | Counter | Total delay faults injected | None |
| `wasmcustom_hfi_faults_delay_duration_milliseconds` | Histogram | Delay duration distribution | 0.5, 1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000, 60000, 300000, 600000, 1800000, 3600000, +Inf |

## Verification Steps

### 1. Check Metrics in Envoy Stats

```bash
# Get a pod with Istio sidecar
POD=$(kubectl get pod -n demo -l app=frontend -o jsonpath='{.items[0].metadata.name}')

# Query Envoy stats endpoint
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s http://localhost:15090/stats/prometheus | grep wasmcustom_hfi_faults

# Expected output (before any faults):
# TYPE wasmcustom_hfi_faults_aborts_total counter
wasmcustom_hfi_faults_aborts_total{} 0
# TYPE wasmcustom_hfi_faults_delays_total counter
wasmcustom_hfi_faults_delays_total{} 0
# TYPE wasmcustom_hfi_faults_delay_duration_milliseconds histogram
wasmcustom_hfi_faults_delay_duration_milliseconds_bucket{le="0.5"} 0
wasmcustom_hfi_faults_delay_duration_milliseconds_bucket{le="1"} 0
...
wasmcustom_hfi_faults_delay_duration_milliseconds_sum{} 0
wasmcustom_hfi_faults_delay_duration_milliseconds_count{} 0
```

### 2. Verify EnvoyFilter (Optional)

```bash
# Check if EnvoyFilter exists
kubectl get envoyfilter hfi-wasm-metrics -n demo

# Inspect configuration
kubectl describe envoyfilter hfi-wasm-metrics -n demo

# Check Envoy config_dump for stats_matcher
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s localhost:15000/config_dump | \
  jq '.configs[] | select(.["@type"] == "type.googleapis.com/envoy.admin.v3.BootstrapConfigDump") | 
      .bootstrap.stats_config.stats_matcher.inclusion_list.patterns'

# Expected: Should contain {"prefix": "wasm"} or {"prefix": "wasmcustom.hfi_faults"}
```

### 3. Verify Wasm Plugin Loaded

```bash
# Check istio-proxy logs for Wasm initialization
kubectl logs -n demo $POD -c istio-proxy | grep -i wasm

# Look for messages like:
# wasm vm created
# wasm filter created
# Defined metric: wasmcustom.hfi_faults_aborts_total
```

## Troubleshooting Guide

### Problem 1: Metrics Not Visible in Envoy Stats

**Symptoms:**
```bash
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s http://localhost:15090/stats/prometheus | grep wasmcustom_hfi_faults
# Returns: (empty, no output)
```

**Diagnosis Steps:**

1. **Check if Wasm plugin is loaded**
   ```bash
   kubectl logs -n demo $POD -c istio-proxy --tail=100 | grep -E "(wasm|hfi)"
   ```
   
   - **If no wasm logs**: WasmPlugin CRD not applied or not targeting this pod
     - Solution: Apply `kubectl apply -f executor/k8s/wasmplugin.yaml`
     - Verify selector matches pod labels: `kubectl get pod -n demo $POD --show-labels`
   
   - **If "failed to load wasm"**: Plugin binary not accessible
     - Check wasm-server service: `kubectl get svc -n boifi wasm-server`
     - Verify plugin file exists: `kubectl exec -n boifi <wasm-server-pod> -- ls -lh /usr/share/nginx/html/plugin.wasm`

2. **Check metric names in plugin source code**
   ```bash
   # Verify metrics use wasmcustom prefix
   grep "define_metric" executor/wasm-plugin/src/lib.rs
   
   # Expected output:
   # "wasmcustom.hfi_faults_aborts_total"
   # "wasmcustom.hfi_faults_delays_total"
   # "wasmcustom.hfi_faults_delay_duration_milliseconds"
   ```
   
   - **If using old names** (e.g., "hfi.faults.*"): Plugin not updated
     - Solution: Rebuild plugin with new names, update wasm-server, restart pods

3. **Check Envoy stats_matcher configuration**
   ```bash
   kubectl exec -n demo $POD -c istio-proxy -- \
     curl -s localhost:15000/config_dump | \
     jq '.configs[] | select(.["@type"] == "type.googleapis.com/envoy.admin.v3.BootstrapConfigDump") | 
         .bootstrap.stats_config.stats_matcher.inclusion_list.patterns[] | select(.prefix | contains("wasm"))'
   ```
   
   - **If no "wasm" prefix found**: Istio default config issue (rare)
     - Solution: Apply EnvoyFilter: `kubectl apply -f executor/k8s/envoyfilter-wasm-stats.yaml`
     - **Important**: Restart pod after applying EnvoyFilter (BOOTSTRAP patch requires restart)
       ```bash
       kubectl delete pod -n demo $POD
       kubectl wait --for=condition=ready pod -l app=frontend -n demo --timeout=90s
       ```

### Problem 2: Metrics Not Incrementing

**Symptoms:**
```bash
# Metrics visible but always zero
wasmcustom_hfi_faults_aborts_total{} 0
wasmcustom_hfi_faults_delays_total{} 0
```

**Diagnosis Steps:**

1. **Verify policy is applied**
   ```bash
   cd executor/cli
   ./hfi-cli policy list --control-plane-addr http://hfi-control-plane.boifi.svc.cluster.local:8080
   ```
   
   - **If empty or error**: Control plane not accessible or no policies applied
     - Check control plane: `kubectl get pods -n boifi -l app=hfi-control-plane`
     - Apply a test policy: `./hfi-cli policy apply -f examples/abort-policy.yaml`

2. **Check policy selector matches pod**
   ```bash
   # Get pod labels
   kubectl get pod -n demo $POD --show-labels
   
   # Expected labels: app=frontend, version=v1, etc.
   ```
   
   - **If policy has selector** (service/namespace): Verify it matches pod labels
   - Example policy selector:
     ```yaml
     selector:
       service: frontend    # Must match app=frontend label
       namespace: demo      # Must match pod namespace
     ```

3. **Verify traffic reaches the pod**
   ```bash
   # Generate test traffic
   kubectl exec -n demo $POD -c server -- curl -v http://localhost:8080/

   # Check Envoy access logs
   kubectl logs -n demo $POD -c istio-proxy --tail=20
   ```
   
   - **If no traffic logs**: Requests not reaching Envoy proxy
     - Check service configuration: `kubectl get svc -n demo`
     - Verify pod port configuration

4. **Check policy match rules**
   ```bash
   # Review policy match criteria
   ./hfi-cli policy describe <policy-id>
   ```
   
   - **If path doesn't match**: Update policy path matcher
     - Example: Policy has `prefix: /api` but requests go to `/`
   - **If headers don't match**: Remove header matchers or adjust them

### Problem 3: Metrics Lost After Pod Restart

**Symptoms:**
- Metrics visible before pod restart
- After `kubectl rollout restart`, metrics revert to 0

**Expected Behavior:** This is **NORMAL**

**Explanation:**
- Metrics are **ephemeral**, stored in Envoy proxy memory
- Pod restart = new Envoy instance = counters reset to 0
- Prometheus should scrape and persist historical data before restart

**Solution (for historical data):**
- Ensure Prometheus is scraping pods regularly (typically every 15-30s)
- Query Prometheus for historical data:
  ```promql
  wasmcustom_hfi_faults_aborts_total[1h]  # Last hour of data
  ```

### Problem 4: Metrics Missing in Prometheus

**Symptoms:**
- Metrics visible in Envoy stats (`curl http://localhost:15090/stats/prometheus`)
- But not appearing in Prometheus UI

**Diagnosis Steps:**

1. **Check Prometheus scrape configuration**
   ```bash
   # Get Prometheus config
   kubectl get configmap -n prometheus prometheus-server -o yaml | grep -A 20 "kubernetes_sd_configs"
   ```
   
   - **If Istio metrics present but not wasm metrics**: Prometheus might be filtering metrics
   - Look for `metric_relabel_configs` that might drop wasmcustom.* metrics

2. **Verify Prometheus can reach pods**
   ```bash
   # Check Prometheus targets
   # Access Prometheus UI: http://<prometheus-url>/targets
   # Look for your namespace (demo) and verify targets are "UP"
   ```
   
   - **If targets DOWN**: Network policy or firewall blocking Prometheus
   - Check service monitor: `kubectl get servicemonitor -n demo`

3. **Check Prometheus scrape endpoint**
   ```bash
   # Prometheus should scrape port 15090 for Istio sidecars
   kubectl get pod -n demo $POD -o jsonpath='{.spec.containers[?(@.name=="istio-proxy")].ports}'
   
   # Expected: Should include port 15090
   ```

4. **Manually test scrape endpoint from Prometheus pod**
   ```bash
   PROM_POD=$(kubectl get pod -n prometheus -l app=prometheus -o jsonpath='{.items[0].metadata.name}')
   
   kubectl exec -n prometheus $PROM_POD -- \
     curl -s http://$POD.demo.svc.cluster.local:15090/stats/prometheus | grep wasmcustom
   ```
   
   - **If empty**: Networking issue between Prometheus and target pods
   - **If shows metrics**: Prometheus configuration issue (check relabel rules)

### Problem 5: Old Metric Names Used

**Symptoms:**
```bash
# Old metric names visible (without wasmcustom prefix)
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s localhost:15090/stats/prometheus | grep "hfi.faults"

# Output: hfi.faults.aborts_total (OLD NAME)
```

**Root Cause:** Old Wasm plugin version deployed

**Solution:**

1. **Rebuild Wasm plugin with new names**
   ```bash
   cd executor/wasm-plugin
   make build  # Compiles with updated metric names
   ```

2. **Update plugin binary in wasm-server**
   ```bash
   # Copy new binary to wasm-server hostPath
   sudo cp target/wasm32-unknown-unknown/release/hfi_wasm_plugin.wasm /tmp/wasm-plugin/plugin.wasm
   
   # Restart wasm-server to reload binary
   kubectl delete pod -n boifi -l app=wasm-server
   ```

3. **Restart application pods to load new plugin**
   ```bash
   kubectl delete pod -n demo $POD
   kubectl wait --for=condition=ready pod -l app=frontend -n demo --timeout=90s
   ```

4. **Verify new metrics**
   ```bash
   kubectl exec -n demo <new-pod> -c istio-proxy -- \
     curl -s localhost:15090/stats/prometheus | grep wasmcustom_hfi_faults
   ```

## Migration from Old Metric Names

If you have existing dashboards or alerts using old metric names:

### Old Names (Not Exposed by Default)
- `hfi.faults.aborts_total`
- `hfi.faults.delays_total`
- `hfi.faults.delay_duration_milliseconds`

### New Names (Exposed with wasmcustom Prefix)
- `wasmcustom_hfi_faults_aborts_total`
- `wasmcustom_hfi_faults_delays_total`
- `wasmcustom_hfi_faults_delay_duration_milliseconds`

### Update Prometheus Queries

**Before:**
```promql
rate(hfi_faults_aborts_total[1m])
```

**After:**
```promql
rate(wasmcustom_hfi_faults_aborts_total[1m])
```

**Note:** 
- No backward compatibility for metric names (first production release)
- Update all dashboards, alerts, and recording rules to use new names

## Quick Reference Commands

```bash
# Check metrics in pod
POD=$(kubectl get pod -n demo -l app=frontend -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n demo $POD -c istio-proxy -- curl -s http://localhost:15090/stats/prometheus | grep wasmcustom_hfi_faults

# Check EnvoyFilter
kubectl get envoyfilter hfi-wasm-metrics -n demo
kubectl describe envoyfilter hfi-wasm-metrics -n demo

# Check Wasm plugin status
kubectl logs -n demo $POD -c istio-proxy --tail=100 | grep -i wasm

# Check policy list
cd executor/cli
./hfi-cli policy list --control-plane-addr http://hfi-control-plane.boifi.svc.cluster.local:8080

# Generate test traffic
for i in {1..20}; do
  kubectl exec -n demo $POD -c server -- curl -s http://localhost:8080/ > /dev/null
  sleep 0.5
done

# Restart pod (for BOOTSTRAP changes)
kubectl delete pod -n demo $POD
kubectl wait --for=condition=ready pod -l app=frontend -n demo --timeout=90s
```

## Additional Resources

- [Envoy Wasm Stats Documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/wasm_filter#statistics)
- [Istio WasmPlugin API Reference](https://istio.io/latest/docs/reference/config/proxy_extensions/wasm-plugin/)
- [Feature Specification](../../specs/008-wasm-metrics-exposure/spec.md)
- [Quickstart Guide](../../specs/008-wasm-metrics-exposure/quickstart.md)
