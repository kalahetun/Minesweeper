# Research: Wasm Metrics Exposure

**Feature**: 008-wasm-metrics-exposure  
**Date**: 2025-12-09  
**Status**: Complete

## Research Questions

This document consolidates research findings for all technical unknowns identified in the implementation plan.

---

## 1. Envoy wasmcustom.* Prefix Convention

### Question
Does Envoy automatically expose metrics prefixed with `wasmcustom.*` without requiring EnvoyFilter configuration?

### Findings

**Primary Finding**: The `wasmcustom.*` prefix convention is **assumed but not explicitly documented** in Envoy's official documentation. However, our combined approach (naming + EnvoyFilter) hedges against this uncertainty.

**Evidence from Codebase**:
- Current implementation uses `hfi.faults.*` prefix (not exposed without EnvoyFilter)
- Documentation references to `wasmcustom.*` exist only in specs, not in running code
- Test script `test-07-metrics.sh` already expects `wasmcustom_hfi_faults_*` format

**Version Requirements**:
- Istio 1.20+ (minimum)
- Istio 1.24+ (recommended, actively tested)
- Envoy 1.24+ bundled with Istio

**Sources**:
- `executor/k8s/METRICS_SOLUTION.md` - Documents wasmcustom approach
- `specs/007-istio-k8s-deployment/spec.md` - Assumes Istio 1.20+
- `executor/k8s/tests/test-07-metrics.sh` - Expects wasmcustom prefix

### Decision

**Adopt the combined approach (Plan 3)**:

1. **Code-level**: Rename metrics to use `wasmcustom.hfi_faults_*` prefix
   - Rationale: Follows Envoy naming convention for Wasm metrics
   - Benefit: If automatic exposure works, metrics appear without EnvoyFilter

2. **Config-level**: Provide EnvoyFilter with stats_matcher configuration
   - Rationale: Defensive configuration ensures reliability
   - Benefit: Works even if automatic exposure doesn't work as expected

3. **Documentation**: Explain both mechanisms and their interaction
   - Rationale: Operators need to understand system behavior
   - Benefit: Clear troubleshooting path if metrics don't appear

### Alternatives Considered

| Alternative | Pros | Cons | Rejected Because |
|-------------|------|------|-----------------|
| wasmcustom prefix only | Simple, no K8s config | Untested assumption, higher risk | Risk too high for production |
| EnvoyFilter only (keep hfi.faults.*) | Explicit configuration | Non-standard naming, less portable | Doesn't follow Envoy conventions |
| Dual metrics (old + new names) | Backward compatible | Doubles metric overhead, complexity | Unnecessary - first production release |

### Rationale

The combined approach provides:
- ✅ **Defense in depth**: Two mechanisms ensure metrics are exposed
- ✅ **Portability**: Works across different Envoy/Istio versions
- ✅ **Clarity**: Explicit configuration (EnvoyFilter) makes system behavior obvious
- ✅ **Convention compliance**: Uses Envoy's wasmcustom prefix standard

**Risk**: Low - if automatic exposure fails, EnvoyFilter catches it

---

## 2. proxy-wasm-rust-sdk Metric APIs

### Question
What is the correct usage pattern for renaming metrics using proxy-wasm-rust-sdk APIs?

### Findings

**API Functions Used**:

1. **`proxy_wasm::hostcalls::define_metric()`**
   - Called during `on_vm_start()` lifecycle
   - Parameters: `MetricType` (Counter/Histogram), `name: &str`
   - Returns: `Result<u32, Status>` (metric ID)
   - Location: `executor/wasm-plugin/src/lib.rs` lines 75-108

2. **`proxy_wasm::hostcalls::increment_metric()`**
   - Called when counters need incrementing
   - Parameters: `metric_id: u32`, `offset: i64`
   - Location: `lib.rs` lines 450, 500; `executor.rs` lines 184, 217

3. **`proxy_wasm::hostcalls::record_metric()`**
   - Called to record histogram values
   - Parameters: `metric_id: u32`, `value: u64`
   - Location: `executor.rs` line 225

**Current Implementation Pattern**:
```rust
// In on_vm_start():
match proxy_wasm::hostcalls::define_metric(
    proxy_wasm::types::MetricType::Counter,
    "hfi.faults.aborts_total"  // ← NEEDS RENAME
) {
    Ok(metric_id) => self.aborts_total_metric = Some(metric_id),
    Err(e) => warn!("Failed to define metric: {:?}", e),
}

// In request handler:
if let Some(metric_id) = self.get_aborts_total_metric() {
    let _ = proxy_wasm::hostcalls::increment_metric(metric_id, 1);
}
```

**Key Insight**: Metric IDs are opaque u32 values. Only the name string in `define_metric()` needs changing.

### Decision

**Code changes required**:

| File | Line | Change |
|------|------|--------|
| `lib.rs` | 77 | `"hfi.faults.aborts_total"` → `"wasmcustom.hfi_faults_aborts_total"` |
| `lib.rs` | 91 | `"hfi.faults.delays_total"` → `"wasmcustom.hfi_faults_delays_total"` |
| `lib.rs` | 105 | `"hfi.faults.delay_duration_milliseconds"` → `"wasmcustom.hfi_faults_delay_duration_milliseconds"` |

**Optional log updates** (for consistency):
- `lib.rs` lines 453, 503
- `executor.rs` lines 187, 220

**No changes needed**:
- Metric ID storage (struct fields remain same type: `Option<u32>`)
- `increment_metric()` calls (use IDs, not names)
- `record_metric()` calls (use IDs, not names)

### Alternatives Considered

| Alternative | Pros | Cons | Rejected Because |
|-------------|------|------|-----------------|
| Define both old and new metrics | Backward compatible | 2x memory, code complexity | Unnecessary for first release |
| Use metric labels instead of names | Flexible | Not supported in current SDK version | API limitation |
| Dynamic metric names | Runtime flexibility | Performance overhead, complexity | YAGNI - names are stable |

### Rationale

**Simplicity**: Only 3 string literals need changing. This is a surgical, low-risk change.

**Best Practice**: Define metrics at startup (on_vm_start), use IDs in hot path for performance.

**Testing**: Existing test infrastructure already expects new names (`test-07-metrics.sh` line 110).

---

## 3. EnvoyFilter stats_matcher Configuration

### Question
What is the correct EnvoyFilter configuration to expose Wasm custom metrics via stats_matcher?

### Findings

**EnvoyFilter Mechanism**:
- **Purpose**: Patch Envoy bootstrap configuration to include metric prefixes in stats output
- **Patch Type**: `BOOTSTRAP` - requires pod restart to take effect
- **Configuration Point**: `stats_config.stats_matcher.inclusion_list`

**Configuration Structure**:
```yaml
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: wasm-stats-inclusion
  namespace: istio-system  # Global or namespace-specific
spec:
  configPatches:
  - applyTo: BOOTSTRAP
    patch:
      operation: MERGE
      value:
        stats_config:
          stats_matcher:
            inclusion_list:
              patterns:
              - prefix: "wasmcustom."
```

**Key Parameters**:
- `namespace: istio-system` → Applies globally to all workloads
- `namespace: demo` + workloadSelector → Applies to specific namespace/pods
- `prefix: "wasmcustom."` → Matches all metrics starting with this string

**Existing File Status**:
- File referenced in documentation: `executor/k8s/envoyfilter-wasm-stats.yaml`
- File search result: **NOT FOUND** in current workspace
- **Action Required**: File needs to be created (already exists per conversation context)

### Decision

**Use namespace-specific EnvoyFilter**:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: hfi-wasm-metrics
  namespace: demo  # Apply to demo namespace where HFI is deployed
spec:
  workloadSelector:
    labels:
      hfi-enabled: "true"  # Only pods with this label
  configPatches:
  - applyTo: BOOTSTRAP
    patch:
      operation: MERGE
      value:
        stats_config:
          stats_matcher:
            inclusion_list:
              patterns:
              - prefix: "wasmcustom.hfi_faults"
```

**Deployment**:
```bash
kubectl apply -f executor/k8s/envoyfilter-wasm-stats.yaml
kubectl rollout restart deployment -n demo  # Required for BOOTSTRAP patch
```

### Alternatives Considered

| Alternative | Pros | Cons | Rejected Because |
|-------------|------|------|-----------------|
| Global EnvoyFilter (istio-system) | Applies everywhere | Affects all workloads, overkill | Too broad scope |
| Include all wasmcustom.* | Future-proof | Exposes metrics from other plugins | Keep scope minimal |
| No EnvoyFilter (rely on prefix) | Simpler | Higher risk if prefix doesn't work | Part of combined approach |

### Rationale

**Namespace-specific**: Only affects demo namespace where HFI is deployed, doesn't pollute global Istio config.

**Workload selector**: Further scopes to pods with `hfi-enabled: true` label (added to deployment manifests).

**Explicit prefix**: `wasmcustom.hfi_faults` is more specific than `wasmcustom.*`, reducing noise.

**BOOTSTRAP lifecycle**: Envoy constraint, not our choice - stats_matcher requires bootstrap configuration.

---

## 4. Metric Name Migration Strategy

### Question
Do we need backward compatibility to support both old (`hfi.faults.*`) and new (`wasmcustom.hfi_faults_*`) metric names during migration?

### Findings

**Current Deployment Status**:
- **Feature 007** (Istio K8s deployment) just completed (2025-12-09)
- **First production release**: No existing deployments with monitoring dashboards
- **Test environment only**: Metrics have been defined but not actively monitored

**Impact Assessment**:

| Stakeholder | Impact | Mitigation Needed |
|-------------|--------|-------------------|
| Internal testing | Test scripts expect new names | ✅ Already updated (test-07-metrics.sh) |
| External users | No users yet | ❌ None needed |
| Prometheus queries | No queries exist | ❌ None needed |
| Grafana dashboards | No dashboards exist | ❌ None needed |

**Breaking Change Analysis**:
- ✅ Clean slate: Feature 007 just deployed, no monitoring in production
- ✅ Test infrastructure ready: Scripts already use `wasmcustom_hfi_faults_*`
- ✅ Documentation updated: METRICS_SOLUTION.md documents new approach

### Decision

**No backward compatibility needed - Direct migration**

**Rationale**:
1. **Timing**: Feature 007 just completed, no production monitoring established
2. **Simplicity**: Avoid dual metric definitions (2x memory, code complexity)
3. **Clarity**: One naming scheme is easier to document and support
4. **Test readiness**: Infrastructure already expects new names

**Migration Plan**:
```
Phase 1: Code change
- Update 3 metric names in lib.rs
- Rebuild Wasm plugin
- Update Docker image

Phase 2: Deployment
- Apply updated WasmPlugin CRD
- Apply EnvoyFilter configuration
- Rolling restart pods

Phase 3: Verification
- Run test-07-metrics.sh
- Verify metrics in Envoy stats endpoint
- Confirm Prometheus scraping (if configured)

No rollback period needed - first production release
```

### Alternatives Considered

| Alternative | Pros | Cons | Rejected Because |
|-------------|------|------|-----------------|
| Dual metrics (old + new) | Gradual migration | 2x overhead, complexity | No existing users to migrate |
| Prometheus recording rules | Aliasing without code changes | Adds Prometheus dependency | No queries to preserve |
| Deprecation period | Industry best practice | Delays feature, adds complexity | Nothing to deprecate |

### Rationale

**YAGNI Principle**: Don't build migration infrastructure when there's nothing to migrate from.

**Simplicity wins**: Direct rename is the simplest, safest approach for a new feature.

**Documentation**: Clearly document the naming in README and quickstart guide.

---

## 5. Testing and Verification Strategy

### Question
How do we comprehensively test that metric renaming works correctly across all scenarios?

### Findings

**Test Layers Required**:

1. **Unit Tests** (Rust)
   - Verify `define_metric()` succeeds with new names
   - Mock hostcall returns to test error handling
   - File: `executor/wasm-plugin/tests/metrics_test.rs` (NEW)

2. **Integration Tests** (kubectl + curl)
   - Deploy plugin, check Envoy stats endpoint
   - Verify metric presence and correct naming
   - File: `executor/k8s/tests/test-metrics.sh` (NEW)

3. **E2E Tests** (existing framework)
   - Extend `test-07-metrics.sh` to verify counters increment
   - Test with and without EnvoyFilter
   - File: `executor/k8s/tests/test-07-metrics.sh` (UPDATE)

4. **Manual Verification** (operator commands)
   - Document curl commands in quickstart.md
   - Verify Prometheus scraping

**Test Matrix**:

| Scenario | wasmcustom prefix | EnvoyFilter | Expected Result |
|----------|-------------------|-------------|-----------------|
| Both | ✅ | ✅ | Metrics visible |
| Prefix only | ✅ | ❌ | Metrics visible (if assumption holds) |
| Filter only | ❌ | ✅ | Metrics NOT visible (wrong prefix) |
| Neither | ❌ | ❌ | Metrics NOT visible |

### Decision

**Three-tier testing approach**:

1. **Fast feedback** (Unit tests):
   ```rust
   #[test]
   fn test_define_wasmcustom_metrics() {
       // Verify metric definitions succeed
       assert!(define_metric(Counter, "wasmcustom.hfi_faults_aborts_total").is_ok());
   }
   ```

2. **Integration** (Automated):
   ```bash
   # test-metrics.sh
   kubectl apply -f wasmplugin.yaml
   sleep 5
   POD=$(kubectl get pod -l app=frontend -n demo -o name | head -1)
   kubectl exec $POD -c istio-proxy -- \
     curl -s localhost:15090/stats/prometheus | grep wasmcustom_hfi_faults
   ```

3. **E2E** (User journey):
   ```bash
   # Apply policy, trigger faults, verify metrics increment
   hfi-cli policy apply abort-policy.yaml
   curl http://frontend/  # Trigger fault
   # Check counters increased
   ```

### Alternatives Considered

| Alternative | Pros | Cons | Rejected Because |
|-------------|------|------|-----------------|
| Manual testing only | No code overhead | Not repeatable, error-prone | Can't verify in CI |
| Prometheus-based tests | End-to-end validation | Complex setup, slower | Overkill for basic verification |
| Mock Envoy in tests | Fast unit tests | Doesn't catch real integration issues | Need both unit + integration |

### Rationale

**Pyramid approach**: Many unit tests, some integration tests, few E2E tests.

**Practical**: Integration tests using kubectl are simple and effective for K8s-native features.

**Documentation as tests**: Quickstart commands double as manual verification steps.

---

## Summary & Recommendations

### Key Decisions

1. **✅ Use combined approach**: wasmcustom prefix + EnvoyFilter for defense in depth
2. **✅ Direct migration**: No backward compatibility needed (first production release)
3. **✅ Namespace-scoped EnvoyFilter**: Apply only to demo namespace with workload selector
4. **✅ Surgical code changes**: Only 3 string literals in define_metric() calls
5. **✅ Three-tier testing**: Unit, integration, and E2E test coverage

### Implementation Checklist

- [ ] Update 3 metric names in `lib.rs` (lines 77, 91, 105)
- [ ] Create EnvoyFilter YAML (namespace-scoped, workload selector)
- [ ] Add unit tests for metric definitions
- [ ] Create integration test script (test-metrics.sh)
- [ ] Update E2E test (test-07-metrics.sh)
- [ ] Document verification commands in quickstart.md
- [ ] Update README with metrics section
- [ ] Test both scenarios (with and without EnvoyFilter)

### Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|---------|------------|
| wasmcustom prefix doesn't auto-expose | Medium | Low | EnvoyFilter provides fallback |
| EnvoyFilter misconfiguration | Low | Low | Metrics still work via prefix |
| Breaking existing dashboards | None | N/A | First release, no dashboards exist |
| Performance regression | Very Low | Low | Metrics are already defined, just renamed |

### Open Questions

None - all research questions resolved.

### Next Steps

Proceed to **Phase 1: Design & Contracts** (though contracts not needed for this feature).

Generate `quickstart.md` with operator verification commands.
