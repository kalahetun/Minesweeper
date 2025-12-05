# Implementation Plan: Istio/K8s Multi-Pod Deployment

**Branch**: `007-istio-k8s-deployment` | **Date**: 2025-12-05 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/007-istio-k8s-deployment/spec.md`

## Summary

Deploy BOIFI executor to k3s cluster with Istio service mesh. The primary requirement is enabling fault injection across multiple microservices while supporting service-level policy targeting. Key technical challenges:

1. Use Istio `WasmPlugin` CRD to deploy plugin to sidecar Envoys
2. Extract service identity from Envoy node metadata (`WORKLOAD_NAME`, `NAMESPACE`)
3. Implement service selector matching in Wasm plugin
4. Ensure percentage-based fault injection works correctly across multiple pod replicas

## Technical Context

**Language/Version**: Go 1.20+ (Control Plane), Rust (Wasm Plugin)  
**Primary Dependencies**: Istio 1.24+, proxy-wasm-rust-sdk, gin (Control Plane API)  
**Storage**: etcd (via existing hfi-etcd deployment)  
**Testing**: kubectl + curl for E2E, cargo test for unit tests  
**Target Platform**: k3s with Istio service mesh, WASM (wasm32-unknown-unknown)  
**Project Type**: Kubernetes deployment with existing codebase  
**Performance Goals**: Plugin adds <1ms latency, policy updates propagate <30s  
**Constraints**: Must work with Istio sidecar injection, fail-open on errors  
**Scale/Scope**: 12+ microservices in demo namespace

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Separation of Concerns | ✅ PASS | Control Plane (policy mgmt) vs Wasm Plugin (execution) already separated |
| II. Declarative Configuration | ✅ PASS | Policies defined via YAML, WasmPlugin CRD is declarative |
| III. Dynamic & Real-Time | ✅ PASS | Policies pushed via SSE, no restart needed |
| IV. Test-Driven Development | ⚠️ PENDING | E2E tests needed for Istio deployment |
| V. Performance-First Design | ✅ PASS | Existing plugin already meets <1ms target |
| VI. Fault Tolerance | ✅ PASS | Fail-open implemented, reconnect logic exists |
| VII. Simplicity & Minimalism | ✅ PASS | Reusing existing components, minimal new code |
| VIII. Time Control | ✅ PASS | duration_seconds and start_delay_ms already implemented |

**Gate Result**: PASS - No blocking violations. TDD requirement will be addressed in tasks.

## Project Structure

### Documentation (this feature)

```text
specs/007-istio-k8s-deployment/
├── plan.md              # This file
├── research.md          # Phase 0 output - Istio WasmPlugin research
├── data-model.md        # Phase 1 output - ServiceSelector entity
├── quickstart.md        # Phase 1 output - Deployment guide
├── contracts/           # Phase 1 output - Updated policy API
└── tasks.md             # Phase 2 output
```

### Source Code (repository root)

```text
executor/
├── k8s/
│   ├── control-plane.yaml       # Existing - may need updates
│   ├── wasmplugin.yaml          # NEW - Istio WasmPlugin CRD
│   ├── tests/
│   │   └── istio-e2e-test.sh    # NEW - E2E test script
│   └── README.md                 # Update with Istio instructions
├── wasm-plugin/
│   └── src/
│       ├── lib.rs               # UPDATE - Add service identity extraction
│       ├── config.rs            # UPDATE - Add ServiceSelector parsing
│       └── matcher.rs           # UPDATE - Add service matching logic
└── control-plane/
    └── api/
        └── types.go             # UPDATE - Add selector fields to Policy
```

**Structure Decision**: Extend existing executor structure. New Istio-specific files go in `executor/k8s/`. Wasm plugin modifications are minimal - primarily adding service identity awareness.

## Complexity Tracking

> No violations requiring justification - using existing architecture.

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| Deployment method | Istio WasmPlugin CRD | Standard Istio approach, well-documented |
| Service identity | Envoy node metadata | Already available, no additional infra needed |
| Policy distribution | Existing SSE mechanism | Already implemented and working |
