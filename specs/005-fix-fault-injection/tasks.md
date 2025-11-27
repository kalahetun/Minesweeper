# Tasks: Fix Fault Injection

**Feature**: Fix Fault Injection
**Status**: In Progress

## Phase 1: Setup & Configuration

**Goal**: Ensure the development environment is ready for debugging and testing the Wasm plugin and Control Plane.

- [ ] T001 Verify local development environment (Go, Rust, Docker)
- [ ] T002 Verify `hfi-cli` build and basic functionality in `executor/cli`
- [ ] T003 Verify Wasm plugin build process in `executor/wasm-plugin`
- [ ] T004 Verify Control Plane build process in `executor/control-plane`
- [ ] T005 Verify Docker Compose environment startup in `executor/docker`

## Phase 2: Foundational Fixes (Blocking)

**Goal**: Fix the core issue where `abort-policy` returns 200 OK, and ensure basic policy serialization works.

- [ ] T006 [P] Update `FaultInjectionPolicy` struct in `executor/control-plane/api` to match Data Model (add `start_delay_ms`, `duration_seconds`)
- [ ] T007 [P] Update `FaultInjectionPolicy` struct in `executor/wasm-plugin/src/config.rs` to match Data Model
- [ ] T008 [P] Implement correct JSON serialization of policies in `executor/control-plane/distributor.go`
- [ ] T009 [P] Implement correct JSON deserialization of policies in `executor/wasm-plugin/src/lib.rs` (or `config.rs`)
- [ ] T010 [P] Implement `service_name` extraction from Envoy node metadata in `executor/wasm-plugin/src/lib.rs`

## Phase 3: User Story 1 - Fix Abort Injection (P1)

**Goal**: Ensure `abort-policy` correctly returns the specified HTTP status code.
**Independent Test**: Apply `abort-policy` (503), curl returns 503.

- [ ] T011 [US1] Debug and fix `send_http_response` usage in `executor/wasm-plugin/src/lib.rs` for Abort faults
- [ ] T012 [US1] Ensure `Action::Pause` is returned after sending abort response in `executor/wasm-plugin/src/lib.rs`
- [ ] T013 [US1] Add unit test for Abort injection logic in `executor/wasm-plugin/src/lib.rs`
- [ ] T014 [US1] Verify fix with `hfi-cli` and `curl` (manual verification step)

## Phase 4: User Story 2 - Verify Delay Injection (P1)

**Goal**: Ensure `delay-policy` correctly introduces latency without blocking Envoy threads.
**Independent Test**: Apply `delay-policy` (2s), curl takes >2s.

- [ ] T015 [US2] Implement non-blocking delay using `set_tick_period_milliseconds` and `Action::Pause` in `executor/wasm-plugin/src/lib.rs`
- [ ] T016 [US2] Implement `on_tick` handler in `executor/wasm-plugin/src/lib.rs` to resume request after delay
- [ ] T017 [US2] Add unit test for Delay injection logic in `executor/wasm-plugin/src/lib.rs`
- [ ] T018 [US2] Verify fix with `hfi-cli` and `curl` (manual verification step)

## Phase 5: User Story 3 - Verify Percentage & Matching (P2)

**Goal**: Ensure policies respect percentage and header matching rules.
**Independent Test**: 50% policy affects ~50% requests; Header policy only affects matching requests.

- [ ] T019 [US3] Implement/Verify percentage logic using pseudo-random number generator in `executor/wasm-plugin/src/lib.rs`
- [ ] T020 [US3] Implement/Verify header matching logic (Exact, Prefix, Regex) in `executor/wasm-plugin/src/lib.rs`
- [ ] T021 [US3] Implement/Verify `x-boifi-request-id` header matching support in `executor/wasm-plugin/src/lib.rs`
- [ ] T022 [US3] Add unit tests for Matcher and Percentage logic in `executor/wasm-plugin/src/lib.rs`

## Phase 6: User Story 4 - Verify Policy Expiration (P2)

**Goal**: Ensure policies automatically expire after `duration_seconds`.
**Independent Test**: Policy with 5s duration stops working after 6s.

- [ ] T023 [US4] Implement expiration check logic in `executor/wasm-plugin/src/lib.rs` using `proxy_wasm::types::SystemTime`
- [ ] T024 [US4] Implement background cleanup of expired policies (optional, or just ignore them) in `executor/wasm-plugin/src/lib.rs`
- [ ] T025 [US4] Add unit test for Expiration logic in `executor/wasm-plugin/src/lib.rs`

## Phase 7: User Story 5 - Verify Start Delay Execution (P3)

**Goal**: Ensure `start_delay_ms` delays the *injection* of the fault.
**Independent Test**: Policy with `start_delay_ms: 200` waits 200ms before injecting fault.

- [ ] T026 [US5] Implement `start_delay_ms` logic using `set_tick_period_milliseconds` (similar to Delay fault) in `executor/wasm-plugin/src/lib.rs`
- [ ] T027 [US5] Handle state transition from "Waiting for Start Delay" to "Injecting Fault" in `on_tick` in `executor/wasm-plugin/src/lib.rs`
- [ ] T028 [US5] Add unit test for Start Delay logic in `executor/wasm-plugin/src/lib.rs`

## Phase 8: Polish & Observability

**Goal**: Add metrics and ensure robustness.

- [ ] T029 [P] Implement Prometheus metrics emission (`boifi_fault_injected_total`) in `executor/wasm-plugin/src/lib.rs`
- [ ] T030 [P] Implement Fail-Open logic (ensure plugin never crashes request on error) in `executor/wasm-plugin/src/lib.rs`
- [ ] T031 [P] Update `executor/cli/examples/*.yaml` to include new fields (`start_delay_ms`, `duration_seconds`) for testing

## Dependencies

1. **Setup** (T001-T005) must be done first.
2. **Foundational Fixes** (T006-T010) are required for all User Stories.
3. **US1 (Abort)** (T011-T014) is the highest priority fix.
4. **US2 (Delay)** (T015-T018) can be done in parallel with US1 if resources allow, but relies on T006-T010.
5. **US3 (Matching)** (T019-T022) is independent of US1/US2 logic but relies on T006-T010.
6. **US4 (Expiration)** (T023-T025) relies on T006-T010.
7. **US5 (Start Delay)** (T026-T028) shares logic with US2 (Delay) and should probably follow it.

## Implementation Strategy

1. **Fix the Critical Bug**: Focus on T011-T012 (Abort) first to unblock the user.
2. **Enable Features**: Implement T006-T010 to support new fields.
3. **Iterate**: Implement US2, US3, US4, US5 in order.
4. **Verify**: Use `hfi-cli` and `curl` after each phase.
