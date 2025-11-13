# Feature Specification: Complete Wasm-based Injection Executor System

**Feature Branch**: `001-boifi-executor`  
**Created**: 2025-11-13  
**Status**: Draft  
**Input**: Complete implementation of Control Plane, Wasm Plugin, and CLI for dynamic fault injection

## User Scenarios & Testing *(mandatory)*

<!--
  IMPORTANT: User stories should be PRIORITIZED as user journeys ordered by importance.
  Each user story/journey must be INDEPENDENTLY TESTABLE - meaning if you implement just ONE of them,
  you should still have a viable MVP (Minimum Viable Product) that delivers value.
  
  Assign priorities (P1, P2, P3, etc.) to each story, where P1 is the most critical.
  Think of each story as a standalone slice of functionality that can be:
  - Developed independently
  - Tested independently
  - Deployed independently
  - Demonstrated to users independently
-->

### User Story 1 - SRE Manual Chaos Testing (Priority: P1)

An SRE engineer needs to inject faults into microservices to test resilience without modifying application code or restarting services. The engineer should be able to define fault injection policies (abort specific requests, add latency) based on HTTP matching criteria (path, method, headers) and apply them in real-time to control where injection happens.

**Why this priority**: This is the core MVP functionality - enables operators to perform chaos engineering experiments immediately without infrastructure changes.

**Independent Test**: Can be fully tested by: (1) starting Control Plane and Wasm Plugin infrastructure, (2) submitting a policy via CLI, (3) verifying faults are injected in real-time by sending test requests through the proxy.

**Acceptance Scenarios**:

1. **Given** Control Plane and Wasm Plugin are running, **When** SRE applies a policy that aborts 50% of requests to `/payment/checkout`, **Then** the plugin receives the policy within 1 second and subsequent requests matching the criteria are aborted with configured status code
2. **Given** an active abort policy, **When** SRE updates the policy to add a 200ms delay instead, **Then** the plugin receives the update within 1 second and requests are delayed, not aborted
3. **Given** multiple policies are active, **When** a request matches multiple policies, **Then** the plugin applies the first matching rule deterministically
4. **Given** a policy with `duration_seconds=60`, **When** 60 seconds elapse, **Then** the policy is automatically deleted and faults stop being injected

---

### User Story 2 - Real-Time Policy Lifecycle Management (Priority: P1)

Operators need full control over policy lifecycle (create, read, update, delete) via command-line tool with immediate propagation to data plane. Policies must support advanced matching (HTTP method, URL path with regex/prefix, request headers) and temporal control (start_delay_ms for staged injection, duration_seconds for auto-expiration).

**Why this priority**: Required for MVP to be operationally useful - without this, faults can't be managed dynamically.

**Independent Test**: Can be fully tested by: (1) applying various policies via CLI, (2) listing/getting policies to verify metadata, (3) deleting policies and confirming they're removed.

**Acceptance Scenarios**:

1. **Given** a policy YAML file is prepared, **When** SRE runs `hfi-cli policy apply -f policy.yaml`, **Then** the policy is created with unique name and confirmed with success message
2. **Given** a policy exists, **When** SRE runs `hfi-cli policy get <policy-name>`, **Then** full policy details are returned including all match conditions and fault specs
3. **Given** multiple policies exist, **When** SRE runs `hfi-cli policy list`, **Then** all policies are displayed in table format with name, rule count, and status
4. **Given** a policy is no longer needed, **When** SRE runs `hfi-cli policy delete <policy-name>`, **Then** the policy is removed and no longer appears in list
5. **Given** a policy specifies `start_delay_ms=200`, **When** a matching request is processed, **Then** fault is applied 200ms after request headers are received
6. **Given** a policy is applied with `duration_seconds=300`, **When** 300 seconds pass without manual deletion, **Then** the policy is automatically deleted by the system

---

### User Story 3 - High-Performance Plugin Execution (Priority: P1)

The Wasm plugin must execute request matching and fault injection with minimal overhead to avoid impacting legitimate traffic. All rule matching and fault decisions must complete in sub-millisecond timeframes to maintain SLO compliance.

**Why this priority**: Data plane performance directly impacts application health - excessive overhead would defeat the purpose of testing resilience.

**Independent Test**: Can be fully tested by: (1) loading plugin with policies, (2) sending high volume of requests (1000+ req/sec), (3) measuring end-to-end latency delta and confirming < 1ms additional overhead.

**Acceptance Scenarios**:

1. **Given** plugin is loaded with 10 active policies, **When** 1000 requests/sec flow through, **Then** plugin overhead (measured as p99 latency delta) is < 1ms per request
2. **Given** a request matches a fault rule, **When** the plugin executes the fault (abort or delay), **Then** execution is atomic and no request-specific state is leaked to subsequent requests
3. **Given** plugin receives new policy update via SSE, **When** update is applied, **Then** existing in-flight requests use old rules, new requests use new rules (no torn reads)

---

### User Story 4 - Automated Integration with Recommender (Priority: P2)

The Recommender system (upstream of Executor) should be able to submit fault injection plans programmatically to the Control Plane API and receive confirmation, enabling closed-loop optimization experiments.

**Why this priority**: Enables intelligent recommendation workflow - not required for manual MVP but critical for full system integration.

**Independent Test**: Can be fully tested by: (1) submitting a FaultPlan via API, (2) verifying it's stored and distributed, (3) confirming it's auto-deleted after specified duration.

**Acceptance Scenarios**:

1. **Given** Recommender sends a POST request to `/v1/policies` with a FaultPlan, **When** the request is valid, **Then** Control Plane returns 201 with policy name and stores the policy
2. **Given** a policy created by Recommender with `duration_seconds=30`, **When** 30 seconds elapse, **Then** the policy is automatically removed without manual intervention
3. **Given** Recommender needs to query current policies, **When** Recommender calls `GET /v1/policies`, **Then** a list of all active policies is returned in JSON format

---

### User Story 5 - Cloud-Native Deployment (Priority: P2)

All components (Control Plane, Wasm Plugin, CLI) must be deployable in Kubernetes/Docker environments using standard configurations with no special privileges or custom modifications required.

**Why this priority**: Required for production deployment but can be achieved incrementally post-MVP.

**Independent Test**: Can be fully tested by: (1) deploying Control Plane container, (2) deploying Envoy with Wasm plugin sidecar, (3) verifying health checks pass and policies propagate.

**Acceptance Scenarios**:

1. **Given** Docker images are built, **When** containers are started with docker-compose, **Then** all services start without errors and logs show healthy initialization
2. **Given** Kubernetes manifests are provided, **When** deployed to a cluster, **Then** Control Plane pod is ready and Envoy sidecar successfully loads Wasm plugin
3. **Given** multiple Wasm plugin instances are running, **When** a policy is applied, **Then** all instances receive the policy within 1 second

---

### Edge Cases

- **Concurrent policy updates**: What happens when two different operators submit conflicting policies simultaneously?
  - Expected: First write wins, subsequent writes either overwrite or fail depending on conflict resolution strategy
  
- **Network partition between plugin and Control Plane**: How does plugin behave if it loses connection to Control Plane?
  - Expected: Plugin continues using last-known rule set (fail-safe), reconnects automatically with exponential backoff, no requests are blocked
  
- **Malformed policy**: What happens when a policy JSON has invalid syntax or missing required fields?
  - Expected: Control Plane rejects with clear error message (400 Bad Request), policy is not stored
  
- **Request matches multiple policies**: What happens when an incoming request matches rules in multiple policies?
  - Expected: First matching policy is applied deterministically (defined by policy ordering/priority)
  
- **Temporal control edge case**: What happens when start_delay_ms exceeds the total request duration?
  - Expected: Fault is not applied to that request (delay occurs after request completes), logged as skipped
  
- **Plugin rule cache exhaustion**: What happens if rule set grows too large for plugin memory?
  - Expected: Control Plane warns in logs, but all rules are still distributed; plugin memory usage should be monitored

## Requirements *(mandatory)*

### Functional Requirements

#### Control Plane - Policy Management API (7 requirements)

- **FR-001**: Control Plane MUST provide HTTP REST API endpoint `POST /v1/policies` to accept fault injection policy definitions in JSON format with fields: metadata.name, spec.rules[], spec.rules[].match (conditions), spec.rules[].fault (abort/delay), spec.start_delay_ms, spec.duration_seconds
- **FR-002**: Control Plane MUST support policy matching conditions: HTTP method (exact match), URL path (exact/prefix/regex), request headers (exact/contains/regex match)
- **FR-003**: Control Plane MUST support two fault types: **Abort** (return HTTP status code) and **Delay** (add milliseconds latency before allowing request to proceed)
- **FR-004**: Control Plane MUST provide HTTP REST API endpoint `GET /v1/policies` to list all active policies with pagination (20 policies per page default)
- **FR-005**: Control Plane MUST provide HTTP REST API endpoint `GET /v1/policies/{name}` to retrieve full details of a specific policy
- **FR-006**: Control Plane MUST provide HTTP REST API endpoint `DELETE /v1/policies/{name}` to remove a policy immediately, ceasing fault injection
- **FR-007**: Control Plane MUST validate all incoming policies and return 400 Bad Request with descriptive error message if validation fails (missing required fields, invalid JSON, syntax errors)

#### Control Plane - Policy Distribution (5 requirements)

- **FR-008**: Control Plane MUST maintain an in-memory or etcd-backed store of all active policies, ensuring durability for at least single-node failure recovery
- **FR-009**: Control Plane MUST provide Server-Sent Events (SSE) streaming endpoint `GET /v1/config/stream` that allows Wasm plugins to establish persistent connections
- **FR-010**: When any policy changes (create/update/delete), Control Plane MUST compile all active policies into a single `CompiledRuleSet` and broadcast to all connected plugins within 1 second
- **FR-011**: Control Plane MUST support auto-expiration of policies: if `duration_seconds > 0`, the policy MUST be automatically deleted after the specified duration elapses (using internal timer/scheduler)
- **FR-012**: Control Plane MUST handle multiple concurrent plugin connections (at least 10 concurrent SSE subscribers) without degradation

#### Wasm Plugin - Core Functionality (10 requirements)

- **FR-013**: Wasm Plugin MUST establish and maintain SSE connection to Control Plane `/v1/config/stream` endpoint on startup and auto-reconnect with exponential backoff (max 5s delay) if connection is lost
- **FR-014**: Wasm Plugin MUST parse and deserialize incoming policy updates from SSE stream and safely update in-memory rule cache using thread-safe mechanisms (atomic operations or RW locks)
- **FR-015**: Wasm Plugin MUST intercept all HTTP requests flowing through Envoy and match them against current rule set before allowing them to proceed (request headers are available for matching)
- **FR-016**: For each intercepted request, Wasm Plugin MUST: (1) extract headers and determine metadata (method, path, headers), (2) iterate through rules in order, (3) stop at first match
- **FR-017**: When a rule matches and fault type is **Abort**, Wasm Plugin MUST immediately return configured HTTP status code (e.g., 500, 503) to client with optional error body, without forwarding to upstream
- **FR-018**: When a rule matches and fault type is **Delay**, Wasm Plugin MUST delay the request by configured milliseconds using Envoy timer APIs before allowing request to proceed to upstream service
- **FR-019**: Wasm Plugin MUST respect temporal control: if `start_delay_ms > 0`, fault is applied only after that many milliseconds have elapsed since request headers received (can skip fault if request ends before delay window)
- **FR-020**: Wasm Plugin MUST ensure fault injection is atomic: no partial fault state should be visible to subsequent requests, all state updates occur synchronously at decision point
- **FR-021**: Wasm Plugin MUST maintain < 1 millisecond overhead per request when processing high volume (1000+ req/sec) with 10 active policies
- **FR-022**: Wasm Plugin MUST handle rule set updates without blocking in-flight requests: existing requests use old rules, new requests use new rules (lock-free or RW-lock with read optimization)

#### CLI Tool - Policy Operations (4 requirements)

- **FR-023**: CLI MUST implement `hfi-cli policy apply -f <file>` command to read policy from YAML/JSON file and submit to Control Plane API, returning success/failure status
- **FR-024**: CLI MUST implement `hfi-cli policy get [name]` command: with name argument returns full policy details; without argument lists all policies in table format
- **FR-025**: CLI MUST implement `hfi-cli policy delete <name>` command to remove policy from Control Plane, with confirmation prompt before deletion
- **FR-026**: CLI MUST support global flags: `--control-plane-addr` (default localhost:8080), `--timeout` (default 10s), `--output` format (table/json/yaml, default table)

#### Observability & Reliability (5 requirements)

- **FR-027**: Control Plane MUST log all policy mutations (create, update, delete) with timestamp and actor information at INFO level; log all API errors at ERROR level
- **FR-028**: Wasm Plugin MUST log connection status changes, rule updates, and errors (never log individual request details unless ERROR level for data privacy)
- **FR-029**: Control Plane MUST respond to health check requests (e.g., `GET /healthz`) with 200 OK if operational, 503 if degraded (unable to serve requests)
- **FR-030**: Wasm Plugin MUST fail-safe: if unable to connect to Control Plane or rule cache is unavailable, MUST default to permissive behavior (allow all requests to proceed) rather than block
- **FR-031**: System MUST handle network partition gracefully: plugins continue using last-known rules, Control Plane remains responsive to new policy submissions, automatic recovery when connection restored

### Key Entities

- **FaultInjectionPolicy**: Represents a named collection of fault injection rules. Contains: metadata.name (unique identifier), spec.rules[] (list of rules), spec.start_delay_ms (when to apply faults), spec.duration_seconds (auto-expiration timer). Can be stored and retrieved as JSON/YAML.

- **InjectionRule**: A single rule within a policy. Contains: match conditions (method/path/header criteria), fault specification (abort status or delay ms), probability (percentage of matching requests to inject).

- **CompiledRuleSet**: Optimized, ready-to-execute representation of all active policies. Contains: version identifier (timestamp or hash), flattened list of rules pre-compiled for fast matching (regex pre-compiled, etc.), JSON-serializable for SSE transmission.

- **MatchCondition**: Specifies when a rule applies. Contains: HTTP method (GET, POST, etc.), URL path (exact/prefix/regex), request headers (key-value pairs with exact/contains/regex matching).

- **FaultSpec**: Describes the fault to inject. Contains: type (abort/delay), abort_code (HTTP status for abort), delay_ms (milliseconds for delay), percentage (0-100, probability).

## Success Criteria *(mandatory)*
- **[Entity 2]**: [What it represents, relationships to other entities]

## Success Criteria *(mandatory)*

<!--
  ACTION REQUIRED: Define measurable success criteria.
  These must be technology-agnostic and measurable.
-->

### Measurable Outcomes

- **SC-001**: [Measurable metric, e.g., "Users can complete account creation in under 2 minutes"]
- **SC-002**: [Measurable metric, e.g., "System handles 1000 concurrent users without degradation"]
- **SC-003**: [User satisfaction metric, e.g., "90% of users successfully complete primary task on first attempt"]
- **SC-004**: [Business metric, e.g., "Reduce support tickets related to [X] by 50%"]

- **SC-001**: Control Plane API responds to policy creation requests within 100ms (p99 latency)
- **SC-002**: Policy updates are propagated to all connected Wasm plugins within 1 second of creation/modification
- **SC-003**: Wasm Plugin adds < 1 millisecond overhead per request when 10 active policies are loaded and processing 1000 req/sec
- **SC-004**: Fault injection matches incoming requests with > 99.9% accuracy (correct rule selection, no false positives/negatives)
- **SC-005**: Policy auto-expiration (duration_seconds) is accurate within ±5 seconds (policy deleted no more than 5s after target time)
- **SC-006**: System supports at least 10 concurrent plugin connections with stable memory usage (no memory leaks over 24 hours)
- **SC-007**: CLI commands return success/failure status within 2 seconds for typical operations (apply, get, delete)
- **SC-008**: All policy data is persisted and survives Control Plane restart (when using persistent storage)
- **SC-009**: Error messages from API and CLI are actionable and guide users to resolution (e.g., "Policy 'payment' not found. Use 'policy list' to see available policies")
- **SC-010**: Policy matching supports at least 10 active rules and 20 concurrent requests without timeout
- **SC-011**: Help documentation is available for all CLI commands (e.g., `hfi-cli policy --help`)
- **SC-012**: Plugin gracefully handles missing or unavailable Control Plane (doesn't crash, logs error, allows requests to proceed)
- **SC-013**: Temporal control parameters (start_delay_ms, duration_seconds) are honored within ±50ms accuracy for requests lasting > 500ms
- **SC-014**: System prevents duplicate policy names (second submission with same name updates existing policy or returns 409 Conflict)
- **SC-015**: All code components (Control Plane, Plugin, CLI) have test coverage > 70% critical paths, > 85% for core business logic

## Assumptions

The following assumptions are made about the deployment and operational environment:

1. **Envoy Proxy Availability**: Envoy is deployed as sidecar proxy in Kubernetes or standalone, and Wasm plugin is properly loaded via Envoy configuration with correct Control Plane address
2. **Network Connectivity**: Control Plane and Wasm plugins can communicate via HTTP/SSE over standard network paths; no exotic routing or NAT traversal required
3. **Storage Backend**: For MVP, in-memory storage is acceptable; for production, etcd or Redis can be integrated with minimal API changes
4. **Request Metadata Availability**: HTTP request headers are available for matching; request body inspection is out of scope for MVP
5. **No Authentication**: MVP assumes trusted network; authentication/authorization can be added in Phase 2
6. **Single Data Center**: Deployment assumes single Kubernetes cluster or network domain; multi-region replication is Phase 3

## Out of Scope

The following features are explicitly out of scope for this specification and will be addressed in future phases:

- **Request body matching**: Policy conditions are limited to headers, method, and path
- **Response modification**: Faults limited to abort and delay; response body rewriting not supported
- **Metrics/tracing integration**: Injection events are logged but not exported to external monitoring systems
- **Policy versioning**: Only current policy state is maintained; historical versions not tracked
- **Rate limiting**: Percentage-based injection is the only throttling mechanism
- **Advanced scheduling**: Time-based activation (e.g., "inject only between 2-4 PM") is not supported
- **Cost estimation**: No integration with cost analyzers or budgeting systems

---

**Document Version**: 1.0  
**Last Updated**: 2025-11-13  
**Status**: Ready for Planning Phase
