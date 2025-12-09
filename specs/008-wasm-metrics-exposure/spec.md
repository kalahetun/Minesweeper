# Feature Specification: Wasm Metrics Exposure

**Feature Branch**: `008-wasm-metrics-exposure`  
**Created**: 2025-12-09  
**Status**: Complete  
**Input**: User description: "Expose Wasm custom Prometheus metrics using combined approach: wasmcustom prefix naming and EnvoyFilter configuration"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Platform Operator Monitors Fault Injection Metrics (Priority: P1)

Platform operators need to monitor fault injection behavior in production to understand system resilience and validate chaos engineering experiments. They must be able to view metrics showing how many faults were injected, types of faults, and their impact on service latency.

**Why this priority**: This is the core value proposition - without visible metrics, operators cannot validate fault injection is working or measure its impact on system behavior.

**Independent Test**: Deploy Wasm plugin with updated metric names, verify metrics appear in Envoy stats endpoint (`/stats/prometheus`), and confirm Prometheus can scrape them.

**Acceptance Scenarios**:

1. **Given** a Wasm plugin with custom metrics deployed to an Istio service mesh, **When** fault injection policies trigger abort faults, **Then** the `wasmcustom.hfi_faults_aborts_total` counter increments
2. **Given** metrics are exposed via Envoy proxy, **When** operator queries Prometheus, **Then** all three HFI metrics (`aborts_total`, `delays_total`, `delay_duration_milliseconds`) are available with correct values
3. **Given** no fault injection policies are active, **When** operator checks metrics, **Then** counters remain at zero (baseline verification)

---

### User Story 2 - Operator Validates Metrics Configuration (Priority: P2)

Platform operators deploying the HFI system to new clusters need to verify that metrics are properly configured and exposed before running production chaos experiments.

**Why this priority**: Essential for deployment validation, but secondary to actually having metrics work in existing deployments.

**Independent Test**: Apply EnvoyFilter configuration, restart pods, and verify Envoy configuration includes the stats matcher rules via `/config_dump` endpoint.

**Acceptance Scenarios**:

1. **Given** EnvoyFilter is applied to a namespace, **When** operator inspects pod Envoy configuration, **Then** the stats_matcher includes `wasmcustom.*` prefix patterns
2. **Given** a fresh deployment without EnvoyFilter, **When** operator checks Envoy stats, **Then** metrics are still visible (due to wasmcustom prefix convention)
3. **Given** EnvoyFilter is deleted, **When** pods are restarted, **Then** metrics remain visible (validating that naming convention alone is sufficient)

---

### User Story 3 - Operator Troubleshoots Missing Metrics (Priority: P3)

[Describe this user journey in plain language]

**Why this priority**: Important for operational excellence, but lower priority than basic functionality.

**Independent Test**: Follow documented troubleshooting steps (check Envoy stats, verify EnvoyFilter, validate Prometheus scrape config) and identify root cause of missing metrics.

**Acceptance Scenarios**:

1. **Given** metrics are missing from Prometheus, **When** operator curls Envoy stats endpoint directly, **Then** can determine if metrics exist in Envoy but Prometheus isn't scraping
2. **Given** metrics don't appear in Envoy stats, **When** operator checks EnvoyFilter status, **Then** can identify configuration issues
3. **Given** old metric names (without wasmcustom prefix), **When** operator checks documentation, **Then** finds migration guide explaining naming change

---

### Edge Cases

- What happens when EnvoyFilter is applied to wrong namespace or with wrong workload selector?
  - Metrics should still appear due to wasmcustom prefix convention
- What happens when Envoy stats buffer is full?
  - Standard Envoy behavior applies - oldest stats are dropped (documented in troubleshooting)
- What happens during rolling update when half the pods have old code and half have new?
  - New pods expose wasmcustom.* metrics, old pods may expose hfi.faults.* (requires EnvoyFilter)
  - Both metric sets appear during transition period
- What happens if operator applies both EnvoyFilter variants (global + namespace-specific)?
  - More specific namespace filter takes precedence (standard Envoy behavior)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST rename Prometheus metrics in Wasm plugin code to use `wasmcustom.hfi_faults_*` prefix instead of `hfi.faults.*`
- **FR-002**: System MUST update metric increment calls to use new metric names consistently
- **FR-003**: System MUST provide EnvoyFilter configuration that explicitly includes wasmcustom metrics in stats_matcher
- **FR-004**: System MUST support both global (istio-system) and namespace-specific EnvoyFilter deployments
- **FR-005**: Documentation MUST explain the dual approach (naming + EnvoyFilter) and why both are used
- **FR-006**: System MUST maintain metric semantics (counter vs histogram) when renaming
- **FR-007**: System MUST expose metrics via standard Envoy endpoints (`/stats` and `/stats/prometheus`)
- **FR-008**: Metrics MUST be scrapable by Prometheus without additional Prometheus configuration
- **FR-009**: System MUST provide verification commands for operators to validate metrics exposure
- **FR-010**: Documentation MUST include troubleshooting guide for missing metrics

### Key Entities

- **Prometheus Metric**: Counter or histogram tracking fault injection behavior
  - Attributes: name (with wasmcustom prefix), type (counter/histogram), labels (optional), value
  - Three specific metrics: aborts_total, delays_total, delay_duration_milliseconds
  
- **EnvoyFilter Resource**: Kubernetes CRD patching Envoy proxy configuration
  - Attributes: namespace (istio-system for global, demo for namespace-specific), workload selector, patch type (BOOTSTRAP), stats_matcher configuration
  
- **Wasm Plugin Metric Definition**: Code-level metric registration
  - Attributes: metric name, metric type (Counter/Histogram), metric ID (internal reference)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can view all three HFI metrics in Prometheus within 30 seconds of deploying a fault injection policy
- **SC-002**: Metrics appear in Envoy stats endpoint on 100% of pods with Wasm plugin deployed
- **SC-003**: Metric values accurately reflect fault injection activity (abort count matches policy execution count)
- **SC-004**: Metrics remain visible across pod restarts and rolling updates without operator intervention
- **SC-005**: Operators can validate metrics exposure using documented curl commands in under 2 minutes
- **SC-006**: Metrics work correctly in both scenarios: with EnvoyFilter applied and without (wasmcustom prefix only)

## Assumptions & Constraints

### Assumptions

1. Istio 1.20+ is deployed with Prometheus integration enabled
2. Istio proxy (Envoy) automatically exposes metrics with `wasmcustom.*` prefix by default
3. Prometheus is configured to scrape Envoy stats endpoints (standard Istio setup)
4. Operators have kubectl access to apply EnvoyFilter resources
5. Existing Wasm plugin code uses proxy_wasm::hostcalls::define_metric() correctly
6. Metric increment logic already exists and works (verified in code review)

### Constraints

1. Cannot use Istio Telemetry API (only supports predefined Istio metrics, not custom Wasm metrics)
2. Metric names must avoid Envoy reserved prefixes (cluster, listener, http, server)
3. EnvoyFilter applies at BOOTSTRAP lifecycle (requires pod restart to take effect)
4. Histogram buckets are fixed at plugin compile time (cannot be changed dynamically)
5. Metric names cannot be changed without rebuilding Wasm plugin (backward compatibility concern)

## Dependencies

### Internal Dependencies

- Wasm plugin source code (`executor/wasm-plugin/src/lib.rs`)
- Existing metrics implementation (define_metrics method, increment_metric calls)
- Kubernetes manifests directory (`executor/k8s/`)

### External Dependencies

- Istio service mesh (1.20+) with sidecar injection
- Envoy proxy stats subsystem
- Prometheus server (for metrics collection)
- kubectl CLI (for applying EnvoyFilter)

## Out of Scope

- Creating new metrics (only renaming existing ones)
- Prometheus dashboard creation (separate feature)
- Alerting rules based on metrics
- Metric aggregation or rollup logic
- Custom histogram bucket configuration
- Grafana integration
- Metric retention policies
- Multi-cluster metric federation
- Changing metric types (counter to gauge, etc.)
- Adding metric labels or dimensions
- Performance optimization of metric collection

## Risks & Mitigation

### Risk 1: Metric Name Change Breaks Existing Dashboards

**Impact**: Medium - existing Grafana dashboards or Prometheus queries will stop working

**Mitigation**: 
- Document old vs new metric names in migration guide
- Provide Prometheus recording rules to create aliases (if needed)
- This is first production release, so no existing dashboards in wild

### Risk 2: EnvoyFilter Configuration Error

**Impact**: Low - worst case, metrics still work due to wasmcustom prefix

**Mitigation**:
- Test both scenarios (with and without EnvoyFilter)
- Provide kubectl validation commands
- Include EnvoyFilter in E2E test suite

### Risk 3: Envoy Stats Buffer Overflow

**Impact**: Low - high-cardinality metrics could fill stats buffer

**Mitigation**:
- Keep metrics low-cardinality (no per-request labels)
- Document Envoy stats buffer limits
- Monitor stats memory usage in production

## Non-Functional Considerations

### Performance

- Metric collection overhead: Negligible (<1μs per increment)
- Stats endpoint query time: Must respond within 500ms even under load
- Memory overhead: ~100 bytes per metric definition

### Reliability

- Metrics must persist across Envoy hot restart
- Metric counters must never decrease (except on pod restart)
- Histogram values must accurately reflect observed latencies

### Observability

- EnvoyFilter status must be queryable via kubectl
- Metrics presence must be verifiable without Prometheus access
- Envoy config_dump must show stats_matcher configuration

### Documentation

- README must include metrics verification section
- METRICS_SOLUTION.md must document combined approach rationale
- Troubleshooting guide must cover common failure modes
