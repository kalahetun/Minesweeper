# Feature Specification: BOIFI Recommender System

**Feature Branch**: `002-boifi-recommender`  
**Created**: 2025-11-14  
**Status**: Draft  
**Input**: Intelligent Bayesian optimization recommender system for autonomous chaos testing

## User Scenarios & Testing *(mandatory)*

### User Story 1 - SRE Initiates Autonomous Chaos Testing Campaign (Priority: P1)

An SRE engineer wants to launch a systematic, intelligent chaos testing campaign without manual intervention. Rather than manually designing individual fault scenarios, they want the system to automatically suggest promising fault combinations based on a search space configuration (service name, fault types, duration ranges, etc.).

**Why this priority**: This is the core value proposition of the Recommender system - enabling intelligent, autonomous testing that discovers critical faults efficiently. Without this capability, the recommender serves no purpose.

**Independent Test**: An SRE can completely test this feature end-to-end by: defining a search space in YAML, starting an optimization session via POST /v1/optimization/sessions, monitoring progress via GET /v1/optimization/sessions/{id}, and obtaining the best fault combination found, without relying on any other feature.

**Acceptance Scenarios**:

1. **Given** the system is initialized, **When** an SRE posts a valid optimization session request with search space configuration (service name, fault parameters, trial count, budget), **Then** the system returns a session ID and begins autonomous optimization with status PENDING→RUNNING.
2. **Given** an active optimization session, **When** the Recommender has completed a fault injection trial and received severity score, **Then** the system automatically proposes the next most promising fault combination within <50ms.
3. **Given** an active optimization session running 20 trials, **When** each trial completes, **Then** the system's next proposed fault gets progressively better (measured by increasing severity scores in later trials vs. early trials).
4. **Given** an active optimization session, **When** a trial completes with a new best severity score, **Then** the system records and tracks the best fault combination found so far, accessible via session status query.

---

### User Story 2 - Response Analysis Produces Standardized Severity Scores (Priority: P1)

The Response Analyzer must convert raw, heterogeneous observations from the Executor (HTTP status codes, latency, distributed traces, logs) into standardized, normalized severity scores [0.0, 10.0] that quantify how severely the injected fault impacts the target service. This scored feedback is essential for the Bayesian optimizer to learn meaningful patterns.

**Why this priority**: Without accurate severity quantification, the Bayesian optimizer has no signal to guide its search. This is the bridge between execution results and intelligent decision-making.

**Independent Test**: The Response Analyzer can be tested independently by: providing various raw observation payloads (different status codes, latency values, trace data), receiving normalized severity scores, and verifying scores are properly calculated according to the weighted scoring function (Bug Scorer + Performance Scorer + Structure Scorer).

**Acceptance Scenarios**:

1. **Given** raw observation with HTTP 5xx status code, **When** analyzed, **Then** severity score is ≥ 8.0 (indicating severe application fault).
2. **Given** raw observation with normal HTTP 200 + baseline latency (200ms), **When** analyzed, **Then** severity score is ≤ 2.0 (indicating minimal impact).
3. **Given** raw observation with latency increased from baseline 200ms to 1000ms (threshold), **When** analyzed, **Then** severity score is approximately 9.0 (high performance degradation).
4. **Given** raw observation with ERROR logs but HTTP 200, **When** analyzed, **Then** severity score includes bug dimension contribution (~6.0) indicating detected issue despite success response.
5. **Given** raw observation with trace showing 50% more spans than baseline (indicating retries/cascading calls), **When** analyzed, **Then** severity score reflects structural change (≥3.0).

---

### User Story 3 - Real-Time Optimization Progress Monitoring (Priority: P1)

An SRE or automated system needs to monitor the progress of an ongoing optimization session in real-time. They need to see current iteration count, best fault found, best severity score achieved, session status, and understand when the optimization will be complete.

**Why this priority**: Visibility into running optimization is critical for operations - teams need to know if testing is progressing, when it will finish, and what critical faults have been discovered so far.

**Independent Test**: By querying GET /v1/optimization/sessions/{session_id} at various points during execution, the system returns: session status (PENDING/RUNNING/STOPPING/COMPLETED/FAILED), current trial count (incrementing), best severity score (increasing or stable), best fault configuration, and estimated completion time.

**Acceptance Scenarios**:

1. **Given** an optimization session in RUNNING state with 5 completed trials, **When** status is queried, **Then** response includes trials_completed=5, current_best_score (max score seen), best_fault (fault config that achieved it), and estimated_remaining_time.
2. **Given** an optimization session with 100 requested trials completing 50 trials, **When** queried, **Then** progress shows 50% completion and estimated remaining time is reasonable (within project performance budgets).
3. **Given** an optimization session with multiple trials, **When** best severity score is updated, **Then** the GET response reflects the new best_score and best_fault_config without requiring session restart.

---

### User Story 4 - Graceful Session Termination and Result Persistence (Priority: P2)

An SRE may need to stop an optimization session early (due to system constraints, schedule change, or sufficient fault discovery). When stopped, the system should complete the current trial gracefully, persist all results found so far, and provide access to the best fault combination discovered.

**Why this priority**: P2 because optimization can proceed to completion (P1), but operational flexibility is important for production environments where testing windows may be limited.

**Independent Test**: By starting a session, allowing several trials to complete, calling POST /v1/optimization/sessions/{id}/stop, verifying status transitions to STOPPING then COMPLETED, and retrieving final results via GET, all discovered faults remain available.

**Acceptance Scenarios**:

1. **Given** an optimization session in RUNNING state, **When** a stop request is posted, **Then** status transitions to STOPPING and the system completes the current trial before stopping.
2. **Given** a stopped session, **When** status is queried, **Then** status shows COMPLETED (not RUNNING), all results from completed trials are available, and best_fault remains accessible.
3. **Given** a stopped session with 20 completed trials, **When** results are retrieved, **Then** all 20 trial results are persisted and can be exported or analyzed.

---

### User Story 5 - Executor Integration for Autonomous Loop Closure (Priority: P2)

The Recommender's Coordinator Service must reliably communicate with the HFI Executor to: submit proposed fault configurations for execution, wait for execution to complete, retrieve raw observation data, and handle network failures gracefully.

**Why this priority**: P2 because integration with Executor is critical for closed-loop operation, but can be tested separately from the recommendation algorithm itself.

**Independent Test**: By mocking the Executor service and verifying that: fault plans are correctly submitted, execution results are retrieved, and failures (timeouts, connection errors) are handled with appropriate retries and circuit breaker logic.

**Acceptance Scenarios**:

1. **Given** a proposed fault plan (service, fault type, duration, etc.), **When** submitted to Executor, **Then** the Executor receives a properly formatted request and returns raw observation within timeout window.
2. **Given** a temporary network issue during Executor communication, **When** the system attempts to execute a fault, **Then** automatic retry with exponential backoff occurs (3-5 attempts) before failing.
3. **Given** Executor service is temporarily unavailable, **When** multiple consecutive execution attempts fail, **Then** circuit breaker opens and subsequent attempts fail fast without waiting for timeout.

---

### Edge Cases

- What happens when the Executor is completely unavailable for the entire optimization session? (System should gracefully fail and report error in session status)
- How does the system handle malformed or incomplete observations from Executor? (Fail-safe scoring: use default values for missing dimensions, log warning, continue)
- What if the Bayesian optimizer runs out of promising fault combinations to explore? (System should transition to exploitation phase - recommending variations of best-found fault)
- How does the system handle concurrent requests to the same session? (Thread-safe SessionManager using locks to prevent race conditions)
- What happens when a single trial takes longer than expected due to slow Executor? (Timeout mechanism with configurable limits; trial fails gracefully and system continues)
- How does the system handle session restart requests? (New session ID, clean history, fresh optimization starting from first trial)

## Requirements *(mandatory)*

### Functional Requirements

**FR-001**: System MUST accept optimization session requests via REST API (POST /v1/optimization/sessions) with parameters: service name, fault space configuration (dimensions: fault type, duration, other parameters), maximum trials, time budget, and return session ID for tracking.

**FR-002**: System MUST define and validate the search space with support for three dimension types: categorical (enumerated values), real (min/max continuous), and integer (min/max discrete) parameters.

**FR-003**: System MUST implement Bayesian optimization using scikit-optimize (Random Forest surrogate model + Expected Improvement acquisition function) to intelligently propose next fault combinations.

**FR-004**: System MUST maintain observation history (all executed faults + severity scores) and retrain the surrogate model after each trial completes in <200ms.

**FR-005**: System MUST implement Response Analyzer with three independent scoring dimensions: Bug Scorer (HTTP status, error logs, error rate), Performance Scorer (latency degradation), and Structure Scorer (distributed trace analysis).

**FR-006**: System MUST aggregate three scoring dimensions via weighted average (configurable weights) producing normalized severity score in range [0.0, 10.0].

**FR-007**: System MUST detect and handle missing observation data gracefully: if any observation field is missing, default to safe value (0.0 for that dimension) and log warning without failing the optimization loop.

**FR-008**: System MUST track and expose session state via REST API (GET /v1/optimization/sessions/{session_id}) including: session status (PENDING/RUNNING/STOPPING/COMPLETED/FAILED), trials completed/total, best severity score, best fault configuration, and estimated remaining time.

**FR-009**: System MUST implement graceful session termination via REST API (POST /v1/optimization/sessions/{session_id}/stop) that completes current trial, transitions status to COMPLETED, and persists all results.

**FR-010**: System MUST implement retry logic for Executor communication: exponential backoff (0.5s, 1s, 2s, 4s, 8s) with maximum 5 attempts before failing a trial.

**FR-011**: System MUST implement circuit breaker pattern for Executor communication: open after 5 consecutive failures, half-open every 60 seconds to test recovery, close on success.

**FR-012**: System MUST implement connection timeout (5 seconds) and read timeout (30 seconds) for Executor API calls to prevent hanging requests.

**FR-013**: System MUST support concurrent optimization sessions (minimum 10 concurrent) with thread-safe SessionManager using appropriate locking mechanisms.

**FR-014**: System MUST perform health checks on Executor service at startup and periodically (every 60 seconds) to detect availability before attempting fault injection.

**FR-015**: System MUST log all optimization events (session start/stop, trial start/complete, severity score, best fault update) with DEBUG level for detailed analysis and ERROR level for failures.

### Non-Functional Requirements

**NFR-001**: Single optimization loop iteration (propose + execute + analyze + record) MUST complete in <600ms (20ms propose, 500ms execute, 50ms analyze, 30ms record).

**NFR-002**: API response latency for GET /v1/optimization/sessions/{id} MUST be <100ms (not including background optimization worker time).

**NFR-003**: System MUST support 10+ concurrent optimization sessions on a single instance with <500MB memory per session.

**NFR-004**: Response Analyzer MUST calculate severity scores in <100ms even with complete trace data.

**NFR-005**: Model retraining (random forest fit) after each trial MUST complete in <200ms.

**NFR-006**: System MUST achieve 99.9% API availability (excluding planned maintenance).

### Key Entities

- **FaultPlan**: The proposed fault configuration to execute, including: service name, fault type (delay/abort/error_injection), duration_ms, error_code (if applicable), match conditions (header/path matching), and temporal parameters (start_delay_ms for staged injection).

- **RawObservation**: Raw data returned from Executor after executing a fault, containing: HTTP status code, latency_ms, error_rate (0-1), HTTP headers, application logs (list), distributed trace data (Span collection), and timestamp.

- **SeverityScore**: Quantified impact measurement, containing: total_score (0-10), bug_score (0-10), performance_score (0-10), structure_score (0-10), component breakdown, and timestamp.

- **OptimizationSession**: Represents one optimization campaign, containing: session_id (UUID), status (PENDING/RUNNING/STOPPING/COMPLETED/FAILED), parameters (search space, trial limits, budget), results (all trials with faults and scores), best_fault (configuration achieving highest score), best_score (numeric), trials_completed (counter), created_at (timestamp), completed_at (timestamp or null if running).

- **SearchSpaceConfig**: Defines the fault parameter space, containing: dimensions (list of Dimension), constraints (optional conditional rules), and metadata (description, owner, created_date).

- **Dimension**: One parameter in the search space, containing: name (unique), type (categorical/real/integer), bounds (min/max for real/integer, values list for categorical), and default value.

## Success Criteria *(mandatory)*

- **SC-001**: A complete optimization session with 50 trials executes end-to-end without human intervention, and the final best fault configuration is more severe than a random baseline (selected fault has severity score in top 20% of all trials).

- **SC-002**: Response Analyzer correctly computes severity scores with weighted dimensions; independent unit tests verify scoring formulas match technical specification (bug/perf/structure dimensions, aggregation logic).

- **SC-003**: Bayesian optimizer demonstrates learning: comparing faults proposed in trials 1-10 vs. 40-50, latter group has higher average severity scores indicating the model learned to explore more promising regions.

- **SC-004**: Session status API returns accurate, up-to-date progress information with <100ms response latency; querying during active optimization reflects recent trial completions within 1 second.

- **SC-005**: Executor integration handles network failures gracefully: simulating Executor unavailability results in appropriate retries + circuit breaker activation, session completes with degraded results rather than crashing.

- **SC-006**: Concurrent sessions remain independent: running 5 simultaneous optimization sessions produces independent results; best fault found in session A does not appear in session B.

- **SC-007**: Performance targets met: single iteration completes <600ms, API responses <100ms, analyzer <100ms, model training <200ms.

- **SC-008**: Session termination persists all results: stopping a session and later retrieving results shows all completed trials; no data loss occurs.

## Assumptions

1. **Executor Compatibility**: The HFI Executor provides either: (a) policy-based API (POST /v1/policies, DELETE /v1/policies), or (b) extended API with /v1/faults/apply + status endpoints. Recommender implements appropriate client for whichever is available.

2. **Observation Data Format**: Executor returns observations in consistent JSON format with fields: status_code (integer), latency_ms (number), error_rate (number 0-1), logs (string list), trace (Span collection). Missing fields default to safe values (0.0 for numeric, empty list/null for collections).

3. **Stateless Scorer**: Response Analyzer is pure function; identical observations + config always produce identical severity scores. No state maintained across calls.

4. **Single-Objective Optimization**: Recommender optimizes for maximizing severity score (single metric). Multi-objective optimization (Pareto front) is out of scope.

5. **Deterministic Search Space**: Fault space dimensions are defined at session start and don't change during optimization; constraints are static.

6. **Sequential Trial Execution**: Recommender executes one fault trial at a time (serial execution) against Executor. Parallel execution of multiple faults against same service is out of scope.

## Out of Scope

- **Multi-Objective Optimization**: Optimizing for multiple conflicting metrics (e.g., maximize severity AND minimize execution time). Single objective (severity) only.

- **Adaptive Search Space**: Dynamically expanding or modifying fault space dimensions during optimization based on discovered patterns.

- **Fault Scheduling/Timing**: Recommender does not schedule when faults occur relative to load tests or application release cycles; external orchestration required.

- **Metrics Aggregation**: Exporting severity scores and optimization progress to external monitoring systems (Prometheus, Datadog, etc.). Session API provides data; external integration is consumer responsibility.

- **Multi-Service Orchestration**: Coordinating faults across multiple services simultaneously (e.g., cascading failure simulation). Each trial targets one service; orchestration of multi-service campaigns is out of scope.

- **Result Visualization/Reporting**: Dashboard or visualization of optimization progress and Pareto front. API provides raw data; visualization is consumer responsibility.

- **Version Management**: Tracking versions of search space configs or fault history across multiple optimization campaigns. Each session is independent; long-term archival is out of scope.
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
