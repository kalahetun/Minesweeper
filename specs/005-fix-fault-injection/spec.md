# Feature Specification: Fix Fault Injection

**Feature Branch**: `005-fix-fault-injection`  
**Created**: 2025-11-26  
**Status**: Draft  
**Input**: User description: "The executor's wasmplugin is not working correctly. `abort-policy` was applied but `curl` returned 200 OK instead of 503. Need to check all injection functions (abort, delay, percentage, header matching) to see if it's a control-plane, wasm plugin, or config issue."

## Clarifications

### Session 2025-11-26

- Q: What happens when multiple policies match a single request? → A: **Priority-based Resolution**:
    1. **Abort over Delay**: If both Abort and Delay policies match, the Abort policy takes precedence (Delay is ignored).
    2. **Max Delay Wins**: If multiple Delay policies match, the one with the longest duration is applied.
    3. **Service Specificity**: Policies are distributed to all plugins, but plugins must only apply policies relevant to their specific service context (unless global).
- Q: How should the system handle invalid configurations or plugin crashes? → A: **Fail Open**: Log the error and allow the request to proceed normally without any fault injection.
- Q: How does the Wasm plugin identify which service it is controlling to apply the correct policy? → A: **Workload Labels/Metadata**: The plugin extracts the local service identity (e.g., `app` label or service name) from the Envoy node metadata or environment variables provided by the Istio proxy context.
- Q: How does the Recommender target specific requests for injection? → A: **HTTP Header**: The Request Generator adds a custom header (e.g., `x-boifi-request-id`) which the Wasm plugin checks against the policy's header matchers.
- Q: How should the system observe and verify that a fault was actually injected? → A: **Prometheus Metrics**: The Wasm plugin emits custom metrics (e.g., `boifi_fault_injected_total`) labeled by policy, service, and fault type.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Fix Abort Injection (Priority: P1)

As a developer or operator, I need the system to correctly return the specified HTTP error code when an abort policy is applied, so that I can verify my service's error handling logic. Currently, the system returns 200 OK instead of the expected error code.

**Why this priority**: This is the core functionality reported as broken. Without this, the fault injection system fails its primary purpose of simulating errors.

**Independent Test**: Can be fully tested by applying an `abort-policy` and sending a matching request. The response must have the status code defined in the policy.

**Acceptance Scenarios**:

1. **Given** a running BOIFI system with an `abort-policy` configured for 503 Service Unavailable, **When** I send a request matching the policy criteria, **Then** I receive an HTTP 503 response immediately.
2. **Given** a running BOIFI system with an `abort-policy` configured for 404 Not Found, **When** I send a request matching the policy criteria, **Then** I receive an HTTP 404 response.

---

### User Story 2 - Verify Delay Injection (Priority: P1)

As a developer, I need the system to introduce the specified latency when a delay policy is applied, so that I can test my service's timeout and latency handling.

**Why this priority**: Delay injection is the second most common fault type. If abort is broken, delay might be too. Verifying this ensures the fix covers the injection mechanism broadly.

**Independent Test**: Can be tested by applying a `delay-policy` (e.g., 2 seconds) and measuring the response time.

**Acceptance Scenarios**:

1. **Given** a `delay-policy` of 2000ms, **When** I send a matching request, **Then** the response time is at least 2000ms greater than the baseline.

---

### User Story 3 - Verify Percentage and Matching Logic (Priority: P2)

As a developer, I need the system to respect the configured injection percentage and header matching rules, so that I can target specific traffic segments and control the blast radius of experiments.

**Why this priority**: Precision is key for safe fault injection. If the system injects faults into 100% of traffic when 10% was requested, or ignores header constraints, it causes outages instead of experiments.

**Independent Test**: Send a batch of requests (e.g., 100) with a 50% policy and check the distribution. Send requests with and without matching headers.

**Acceptance Scenarios**:

1. **Given** a policy with `percentage: 50`, **When** I send 100 matching requests, **Then** approximately 50 requests trigger the fault (allowing for statistical variance).
2. **Given** a policy matching header `x-test-user: true`, **When** I send a request *without* this header, **Then** no fault is injected.
3. **Given** a policy matching header `x-test-user: true`, **When** I send a request *with* this header, **Then** the fault is injected.

---

### User Story 4 - Verify Policy Automatic Expiration (Priority: P2)

As an operator, I need to ensure that fault injection policies automatically expire after their configured duration, so that I don't accidentally leave faults running indefinitely if I forget to delete them or if the control plane becomes unreachable.

**Why this priority**: Prevents "zombie" faults from causing prolonged outages. Essential for safe automated testing.

**Independent Test**: Apply a policy with a short duration (e.g., 5 seconds). Verify faults are injected immediately, and then verify faults stop being injected after 5 seconds without manual intervention.

**Acceptance Scenarios**:

1. **Given** a policy configured with a 5-second duration, **When** 6 seconds have elapsed since application, **Then** requests matching the policy are no longer faulted (return 200 OK).
2. **Given** an expired policy, **When** I list active policies, **Then** it should either be removed or marked as expired/inactive.

---

### User Story 5 - Verify Start Delay Execution (Priority: P3)

As a developer, I need the system to support a `start_delay_ms` configuration, which delays the *injection* of the fault by a specified time **after the request arrives**, so that I can simulate faults that occur mid-processing (e.g., after some initial computation or downstream calls).

**Clarification**: `start_delay_ms` is a **request-level delay**, meaning each individual request will wait for this duration before the fault is injected. This is different from:
- **Policy-level delay**: When the policy becomes active (handled by `duration_seconds` start time)
- **Delay fault**: The `delay` fault type which adds latency *before forwarding* the request to upstream

**Use Case**: Simulate "late-stage" failures where a service starts processing a request, does some work (database queries, API calls), and then fails. This is critical for testing partial failure handling, resource cleanup, and transaction rollback.

**Why this priority**: This enables more sophisticated failure scenarios, such as "late-stage" failures, which are critical for testing partial failure handling and resource cleanup.

**Independent Test**: Apply a policy with `start_delay_ms: 200` and an abort fault. Measure the time to first byte (TTFB). It should be at least 200ms.

**Acceptance Scenarios**:

1. **Given** a policy with `start_delay_ms: 200` and `abort: 503`, **When** I send a matching request, **Then** the system waits for approximately 200ms after the request arrives before returning the 503 error (TTFB ≈ 200ms).
2. **Given** a policy with `start_delay_ms: 0` (default), **When** I send a matching request, **Then** the fault is injected immediately without additional delay.
3. **Given** a policy with `start_delay_ms: 500` and `delay: 1000ms`, **When** I send a matching request, **Then** the system waits 500ms, then applies a 1000ms delay before forwarding (total latency ≈ 1500ms).

### Edge Cases

- **Conflict Resolution**:
    - **Abort vs Delay**: Abort policy always takes precedence over Delay policy for the same request.
    - **Multiple Delays**: The longest delay duration wins.
- **Failure Handling**:
    - **Invalid Config/Crash**: System MUST **Fail Open**. If the Wasm plugin cannot load configuration, encounters an invalid policy, or crashes, it MUST allow the request to proceed without modification and log the error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST intercept HTTP requests matching an active `FaultInjectionPolicy`.
- **FR-002**: When an `abort` fault is triggered, the system MUST return the HTTP status code specified in the policy and stop further processing of the request.
- **FR-003**: When a `delay` fault is triggered, the system MUST pause request processing for the duration specified in the policy before forwarding the request.
- **FR-004**: The system MUST respect the `percentage` field in the policy, injecting faults only into the specified proportion of matching requests.
- **FR-005**: The system MUST evaluate `headers` matching rules; if a request does not contain the specified headers with matching values, the fault MUST NOT be injected.
- **FR-006**: The Control Plane MUST correctly serialize the policy configuration into a format the Wasm plugin can understand.
- **FR-007**: The Wasm Plugin MUST correctly deserialize the configuration received from the Control Plane.
- **FR-008**: The Wasm Plugin MUST correctly identify the HTTP method and path of incoming requests for matching purposes.
- **FR-009**: The Wasm Plugin MUST identify its own service context (e.g., via Envoy node metadata or environment variables) and ONLY apply policies that match its service identity (e.g., `service_name` or `app` label).
- **FR-010**: The system MUST support matching requests based on custom HTTP headers (e.g., `x-boifi-request-id`) to allow precise targeting by the Request Generator.
- **FR-011**: The Wasm Plugin MUST emit Prometheus metrics (e.g., `boifi_fault_injected_total`) for each injected fault, including labels for `policy_id`, `service`, `fault_type`, and `result`.
- **FR-012**: The system MUST stop enforcing a policy once its configured expiration time or duration has passed.
- **FR-013**: The system MUST support a `start_delay_ms` parameter in the policy, causing the fault injection (abort or delay) to be postponed by the specified duration **after each individual request is received**. This is a per-request delay, not a policy activation delay.

### Success Criteria

- **SC-001**: `abort-policy` (e.g., 503) results in the correct HTTP status code for 100% of matching requests when percentage is 100.
- **SC-002**: `delay-policy` (e.g., 1s) results in response times increasing by at least the delay amount.
- **SC-003**: Policies with specific header matchers do not affect requests missing those headers.
- **SC-004**: Policies with `percentage` < 100 do not affect all requests.
- **SC-005**: Policies with a configured duration cease to be active after the duration elapses.
- **SC-006**: Policies with `start_delay_ms` introduce a per-request pre-injection latency corresponding to the configured value before the fault occurs (e.g., `start_delay_ms: 200` results in TTFB ≈ 200ms for abort faults).

### Key Entities

- **FaultInjectionPolicy**: The configuration object defining the match criteria (path, method, headers) and the fault to inject (abort, delay).
- **WasmConfig**: The internal configuration format passed from Control Plane to the Envoy Wasm Plugin.
