# Implementation Plan: Wasm Metrics Exposure

**Branch**: `008-wasm-metrics-exposure` | **Date**: 2025-12-09 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/008-wasm-metrics-exposure/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

**Primary Requirement**: Expose Wasm plugin custom Prometheus metrics to enable platform operators to monitor fault injection behavior in production.

**Technical Approach (Combined Solution)**:
1. **Code-level**: Rename metrics from `hfi.faults.*` to `wasmcustom.hfi_faults_*` prefix to comply with Envoy's automatic exposure convention
2. **Config-level**: Provide EnvoyFilter with stats_matcher configuration as defensive mechanism
3. **Documentation**: Update deployment guides with verification commands and troubleshooting steps

**Value**: Operators can view abort counts, delay counts, and latency histograms in Prometheus to validate chaos experiments and measure resilience impact.

## Technical Context

**Language/Version**: Rust 1.75+ (Wasm plugin), YAML (Kubernetes manifests)
**Primary Dependencies**: 
  - `proxy-wasm-rust-sdk` 0.2+ (Wasm plugin framework)
  - `proxy_wasm::hostcalls` (metric definition and increment APIs)
  - Istio 1.20+ (service mesh with EnvoyFilter CRD support)
  - Envoy proxy stats subsystem (automatic wasmcustom.* prefix exposure)
**Storage**: N/A (metrics are ephemeral, stored in Envoy memory, scraped by Prometheus)
**Testing**: 
  - `cargo test` (Rust unit tests for metric definition logic)
  - Integration tests via kubectl + curl (verify metrics in Envoy stats endpoint)
  - E2E tests (verify Prometheus scraping and metric accuracy)
**Target Platform**: 
  - Wasm32-wasi (compiled plugin running in Envoy proxy)
  - Kubernetes cluster with Istio service mesh
**Project Type**: Embedded system (Wasm plugin) + Infrastructure (K8s manifests)
**Performance Goals**: 
  - Metric increment overhead: <1μs per call
  - Stats endpoint response time: <500ms under load
  - Zero impact on request latency when no faults injected
**Constraints**: 
  - Metric names cannot use Envoy reserved prefixes (cluster, listener, http, server)
  - Histogram buckets fixed at compile time (cannot change dynamically)
  - EnvoyFilter changes require pod restart (BOOTSTRAP patch lifecycle)
  - Metric definitions must happen in on_vm_start() (before any requests)
**Scale/Scope**: 
  - 3 metrics total (2 counters, 1 histogram)
  - ~10 lines of code changes in lib.rs
  - 1 new EnvoyFilter manifest (~40 lines YAML)
  - Documentation updates in 2-3 files

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. 关注点分离 (Separation of Concerns) ✅
- **PASS**: Metrics exposure is clearly separated - Wasm plugin defines and increments metrics (data plane), EnvoyFilter configures Envoy stats (infrastructure), documentation guides operators (usage)
- No violation: Each component has single responsibility

### II. 声明式配置 (Declarative Configuration) ✅
- **PASS**: EnvoyFilter is declarative YAML defining desired Envoy stats_matcher state
- Metric definitions are declarative (name, type) via `define_metric()` API
- No imperative scripting for configuration

### III. 动态性与实时性 (Dynamic & Real-Time) ✅
- **PASS**: Metrics reflect real-time fault injection activity (counters increment immediately)
- EnvoyFilter changes require pod restart (Envoy limitation, not our design choice)
- Acceptable: BOOTSTRAP lifecycle is Envoy's constraint for stats configuration

### IV. 测试驱动 (Test-Driven Development) ✅
- **PASS**: Plan includes unit tests (cargo test for metric logic), integration tests (kubectl + curl), E2E tests (Prometheus scraping validation)
- Test coverage for all 3 user stories defined
- Verification commands provided for operators

### V. 性能优先 (Performance-First Design) ✅
- **PASS**: 
  - Metric increment is O(1) operation via hostcall (<1μs overhead)
  - No memory allocations in hot path (metrics pre-defined in on_vm_start)
  - Stats endpoint already optimized by Envoy (not our concern)
- Performance goals explicitly defined: <1μs increment, <500ms stats query

### VI. 容错与可靠性 (Fault Tolerance & Reliability) ✅
- **PASS**:
  - Metric definition failures are logged but don't crash plugin (warn! instead of panic)
  - Metrics work with or without EnvoyFilter (dual approach ensures reliability)
  - Metric increment errors are handled gracefully (logged, not panicked)

### VII. 简洁性与最小化原则 (Simplicity & Minimalism) ✅
- **PASS**:
  - Minimal changes: rename 3 metric strings, no new dependencies
  - Simple EnvoyFilter config (7 lines of meaningful YAML)
  - No over-engineering: reusing Envoy's built-in stats system

### VIII. 时间控制与生命周期管理 (Temporal Control & Lifecycle) ✅
- **N/A**: This feature doesn't involve temporal policy lifecycle
- Metrics are ephemeral (reset on pod restart) which is standard Prometheus pattern

**Constitution Compliance**: ✅ ALL GATES PASSED - No violations, ready to proceed

## Project Structure

### Documentation (this feature)

```text
specs/008-wasm-metrics-exposure/
├── spec.md              # Feature specification (COMPLETE)
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output - WILL BE GENERATED
├── quickstart.md        # Phase 1 output - WILL BE GENERATED
├── checklists/
│   └── requirements.md  # Quality checklist (COMPLETE, APPROVED)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

Note: `contracts/` and `data-model.md` are not needed for this feature (no API changes or data entities).

### Source Code (repository root)

```text
executor/
├── wasm-plugin/
│   ├── src/
│   │   └── lib.rs                    # MODIFY: Rename metric strings (3 locations)
│   ├── Cargo.toml                    # No changes needed
│   ├── Makefile                      # No changes (existing build process)
│   └── tests/
│       └── metrics_test.rs           # NEW: Unit tests for metric definitions
├── k8s/
│   ├── envoyfilter-wasm-stats.yaml   # ALREADY EXISTS: Apply to cluster
│   ├── METRICS_SOLUTION.md           # ALREADY EXISTS: Reference documentation
│   ├── README.md                     # MODIFY: Add metrics verification section
│   └── tests/
│       ├── run-all-tests.sh          # MODIFY: Add metrics verification step
│       └── test-metrics.sh           # NEW: Metrics-specific E2E test
└── cli/
    └── examples/
        └── README.md                 # MODIFY: Add metrics observation examples
```

**Structure Decision**: This feature follows the existing repository structure. Changes are localized to:
1. **Wasm plugin code** (`executor/wasm-plugin/src/lib.rs`) - metric name updates
2. **K8s manifests** (`executor/k8s/`) - EnvoyFilter already created, documentation updates needed
3. **Testing** - new test scripts for metrics verification

No new directories or major structural changes required.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

**No Violations**: All constitution gates passed. No complexity justification needed.

---

## Phase 0: Research & Discovery

**Status**: ✅ COMPLETE

### Research Tasks

All research tasks completed. See [research.md](./research.md) for full findings.

**Key Decisions**:
1. ✅ Adopt combined approach (wasmcustom prefix + EnvoyFilter)
2. ✅ Direct migration (no backward compatibility needed)
3. ✅ Namespace-scoped EnvoyFilter configuration
4. ✅ Surgical code changes (3 string literals only)
5. ✅ Three-tier testing strategy

**All NEEDS CLARIFICATION items resolved** - Ready for Phase 1.

---

## Phase 1: Design & Contracts

**Status**: ✅ COMPLETE

**Note**: This feature does not require data-model.md or API contracts/ directory because:
- No new data entities (metrics are ephemeral Envoy stats)
- No API changes (existing Wasm plugin APIs unchanged)
- Configuration-only changes (metric names, EnvoyFilter YAML)

### Deliverables

1. **quickstart.md** - ✅ Created
   - Step-by-step operator guide
   - Verification commands for each metric type
   - Troubleshooting common issues
   - 6-step workflow with expected outputs

2. **Constitution Re-check** - ✅ Passed
   - All 8 constitutional principles validated
   - No violations introduced during design phase
   - Performance goals clearly defined and achievable
   - Testing strategy comprehensive (unit + integration + E2E)

### Architecture Summary

**Component Changes**:
- **Wasm Plugin** (`lib.rs`): 3 metric name string updates
- **K8s Manifests**: EnvoyFilter YAML (already created)
- **Documentation**: README updates, quickstart guide
- **Testing**: Unit tests, integration tests, E2E test updates

**Data Flow** (unchanged):
```
Request → Envoy → Wasm Plugin → increment_metric(id) → Envoy Stats
                                                          ↓
                                    Prometheus ← /stats/prometheus endpoint
```

**Metric Exposure Mechanism**:
```
define_metric("wasmcustom.hfi_faults_aborts_total")
              ↓
Envoy Naming Convention → Auto-expose (primary path)
              ↓
EnvoyFilter stats_matcher → Explicit inclusion (fallback)
              ↓
Prometheus scrape config → Collect metrics
```

---

## Phase 2: Task Breakdown

**Status**: Not started (run `/speckit.tasks` to generate)

This phase will break down implementation into concrete tasks covering:
- Code changes (lib.rs metric renaming)
- Testing (unit tests, integration tests, E2E updates)
- Documentation (README, examples)
- Deployment (EnvoyFilter application)
- Verification (operator validation steps)

**Note**: Phase 2 planning is done by the `/speckit.tasks` command, not `/speckit.plan`.

---

## Summary

**Planning Complete**: ✅ All prerequisites met for implementation phase

**Key Artifacts**:
- [x] Technical context defined
- [x] Constitution compliance verified (2 times)
- [x] Research completed (research.md)
- [x] Quickstart guide created (quickstart.md)
- [x] Implementation strategy validated

**Ready for**: `/speckit.tasks` command to generate detailed task breakdown

**Estimated Effort**:
- Code changes: 1-2 hours (3 lines + optional log updates)
- Testing: 2-3 hours (write tests, run verification)
- Documentation: 1-2 hours (README updates, review)
- Total: ~5-7 hours of development time
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
