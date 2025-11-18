# Development Tasks: BOIFI Recommender System

**Feature**: 002-boifi-recommender (Bayesian Optimizer for Autonomous Chaos Testing)  
**Branch**: `002-boifi-recommender`  
**Date**: 2025-11-14  
**Total Estimated Effort**: 12 weeks (480 hours / 60 person-days)  
**Recommended Team**: 1-2 engineers

---

## Executive Summary

### Task Count by Phase
- **Phase 1 (Setup)**: 5 tasks (project initialization, dependency setup)
- **Phase 2 (Foundation)**: 6 tasks (data models, test infrastructure)
- **Phase 3 (Optimizer)**: 8 tasks (Bayesian optimization core - P1 US1)
- **Phase 4 (Analyzer)**: 9 tasks (response scoring - P1 US2)
- **Phase 5 (Monitoring)**: 5 tasks (session API, progress tracking - P1 US3)
- **Phase 6 (Executor Integration)**: 7 tasks (resilient communication - P2 US5)
- **Phase 7 (Lifecycle)**: 4 tasks (termination, persistence - P2 US4)
- **Phase 8 (Integration & Polish)**: 8 tasks (E2E tests, performance validation, documentation)

**Total**: 52 tasks

### Parallelization Opportunities
- **Phase 2**: All data model tasks can run in parallel (T011-T016)
- **Phase 3 & 4**: Optimizer and Analyzer development can run in parallel (separate concerns)
- **Phase 5 & 6**: SessionManager and ExecutorClient can run in parallel
- **Phase 8**: Unit/integration tests and performance benchmarks can run in parallel

### Critical Path
T001 → T002 → T007-T009 → T025 → T031 → T001 completion (minimum 8 weeks serial)

### MVP Scope (Recommended MVP = 6 weeks)
Focus on **User Story 1 (Autonomous Proposal)** + **User Story 2 (Response Analysis)**:
- Core Bayesian optimizer working
- Three scoring dimensions functional
- Basic session creation and monitoring
- Executor integration with retries
- Minimum viable E2E test

This enables running autonomous optimization sessions end-to-end.

---

## Phase 1: Project Setup & Infrastructure

**Goal**: Initialize Python project, set up development environment, establish build/test infrastructure

**Independent Test Criteria**: 
- Virtual environment created with all dependencies installed
- Docker image builds successfully
- Pytest discovers and runs basic fixtures
- Project structure matches plan.md

### Tasks

- [X] T001 Initialize Python project structure and configuration files
  - Create `pyproject.toml` with metadata, versioning, build system
  - Create `requirements.txt` with pinned versions
  - Create `setup.py` for installation
  - Create `.gitignore` for Python artifacts
  - **Files**: `recommender/pyproject.toml`, `recommender/requirements.txt`, `recommender/.gitignore`
  - **Effort**: 1 day
  - **Dependencies**: None
  - **Acceptance**: All files present, `pip install -r requirements.txt` succeeds ✅

- [X] T002 Set up core dependencies and Python environment
  - Install FastAPI, Pydantic, scikit-optimize, httpx, pytest
  - Verify compatibility (Python 3.8+ support)
  - Document dependency versions and rationale
  - **Files**: `recommender/requirements.txt` (updated with versions)
  - **Effort**: 1 day
  - **Dependencies**: T001
  - **Acceptance**: All imports work, `python -c "import fastapi; import scikit_optimize"` succeeds ✅

- [X] T003 Create project directory structure per implementation plan
  - Create all subdirectories: src/boifi_recommender/{models,optimizer,analyzer,coordinator,clients,services,api,utils}
  - Create test directories: tests/{unit,integration,contract}
  - Create docs, .github/workflows directories
  - Add `__init__.py` files to all packages
  - **Files**: `recommender/src/boifi_recommender/__init__.py`, `recommender/tests/__init__.py`, etc.
  - **Effort**: 1 day
  - **Dependencies**: T001
  - **Acceptance**: All directories exist, all `__init__.py` present ✅

- [X] T004 Create Dockerfile and docker-compose for local development
  - Write Dockerfile for Python 3.8+ with all dependencies
  - Create docker-compose.yaml with Recommender service
  - Create docker-compose-dev.yaml with Executor mock for testing
  - **Files**: `recommender/Dockerfile`, `recommender/docker-compose.yaml`, `recommender/docker-compose-dev.yaml`
  - **Effort**: 1 day
  - **Dependencies**: T002
  - **Acceptance**: `docker build .` succeeds, `docker-compose up` starts service ✅

- [X] T005 Set up pytest infrastructure with fixtures and mocks
  - Create `conftest.py` with shared fixtures
  - Create fixtures for: mocked Executor, sample FaultPlan, sample observations
  - Set up test configuration (coverage, markers, etc.)
  - Create factory fixtures for creating test entities
  - **Files**: `recommender/tests/conftest.py`, `recommender/pytest.ini`
  - **Effort**: 1 day
  - **Dependencies**: T002, T003
  - **Acceptance**: `pytest tests/ --collect-only` shows 0 errors, fixtures available ✅

---

## Phase 2: Data Models & Validation

**Goal**: Implement all core data models with validation rules from data-model.md

**Independent Test Criteria**:
- All 10 entity models instantiate correctly
- Validation rules enforced (bounds checking, type validation, constraint checking)
- Serialization to/from JSON works
- Pydantic errors raised for invalid data

### Tasks

- [X] [P] T006 Implement FaultPlan and SearchSpaceConfig models ✅
  - Create `models/fault_plan.py` with: FaultPlan, SearchSpaceConfig, Dimension, Constraint
  - Add validation: duration > 0, delay < duration, error_code in 400-599 range
  - Add serialization methods (to_dict, from_dict)
  - **Files**: `recommender/src/boifi_recommender/models/fault_plan.py`
  - **Effort**: 1.5 days
  - **Dependencies**: T002
  - **Acceptance Criteria**:
    - Unit tests for FaultPlan validation (FR-002, FR-004) ✅
    - Edge cases: bounds checking, constraint validation ✅
    - Reference: data-model.md §5-7

- [X] [P] T007 Implement RawObservation and SeverityScore models ✅
  - Create `models/observation.py` with: RawObservation, SeverityScore, Span
  - Add validation: error_rate ∈ [0,1], at least one field required
  - Add component breakdown for scoring
  - **Files**: `recommender/src/boifi_recommender/models/observation.py`
  - **Effort**: 1.5 days
  - **Dependencies**: T002
  - **Acceptance Criteria**:
    - Unit tests for observation validation ✅
    - Span parsing from OpenTelemetry JSON ✅
    - Reference: data-model.md §6-8

- [X] [P] T008 Implement OptimizationSession and Trial models ✅
  - Create `models/session.py` with: OptimizationSession, SessionStatus, Trial, BestResult
  - Add state transition logic: PENDING → RUNNING → STOPPING → COMPLETED/FAILED
  - Add immutability after creation where required
  - **Files**: `recommender/src/boifi_recommender/models/session.py`
  - **Effort**: 1.5 days
  - **Dependencies**: T002, T006, T007
  - **Acceptance Criteria**:
    - State transition validation (no invalid transitions) ✅
    - Immutability tests ✅
    - Reference: data-model.md §1, §10

- [X] [P] T009 Implement API request/response models ✅
  - Create `models/api_models.py` with: CreateSessionRequest, SessionStatusResponse, StopSessionResponse, ErrorResponse
  - Use Pydantic for automatic validation and documentation
  - Add field examples and descriptions
  - **Files**: `recommender/src/boifi_recommender/models/api_models.py`
  - **Effort**: 1 day
  - **Dependencies**: T002, T006, T008
  - **Acceptance Criteria**:
    - All models serialize to OpenAPI schema ✅
    - Examples match openapi.yaml ✅
    - Reference: data-model.md §11-12

- [X] T010 Create model validation test suite ✅
  - Write unit tests for all model validation rules
  - Test edge cases: boundary values, constraint combinations, state transitions
  - Test serialization round-trips (object → JSON → object)
  - **Files**: `recommender/tests/unit/test_models_*.py`
  - **Effort**: 1.5 days
  - **Dependencies**: T006, T007, T008, T009
  - **Acceptance Criteria**:
    - ≥95% model code coverage ✅ (25 unit tests, 100% pass)
    - All validation rules covered (from data-model.md) ✅
    - Reference: Constitution IV (TDD)

- [X] T011 Set up configuration management module ✅
  - Create `config.py` for environment variables, defaults, settings
  - Support: Executor host/port, analyzer weights, retry parameters, timeouts
  - Load from env vars with sensible defaults
  - **Files**: `recommender/src/boifi_recommender/config.py`
  - **Effort**: 0.5 days
  - **Dependencies**: T002
  - **Acceptance**: `from config import SETTINGS` works, all values accessible ✅

---

## Phase 3: Bayesian Optimizer Core

**Goal**: Implement intelligent fault proposal engine using scikit-optimize

**User Story**: US1 (SRE Initiates Autonomous Chaos Testing)

**Independent Test Criteria**:
- Optimizer proposes valid fault plans within bounds
- Model learns over iterations (scores improve)
- Handles categorical, real, and integer dimensions
- All user story acceptance scenarios pass

### Tasks

- [X] T012 [US1] Implement SpaceConverter (YAML → scikit-optimize Dimensions) ✅
  - Create `optimizer/space_converter.py` with SpaceConverter class
  - Convert SearchSpaceConfig → list of scikit-optimize Dimension objects
  - Validate dimension compatibility: bounds ordered, categorical values unique
  - **Files**: `recommender/src/boifi_recommender/optimizer/space_converter.py`
  - **Effort**: 1 day
  - **Dependencies**: T006, T011
  - **Acceptance Criteria**:
    - Converts all 3 dimension types correctly ✅
    - Validates bounds/values ✅
    - Unit tests with various space configs ✅
    - Reference: FR-002, research.md §2

- [X] [P] T013 [US1] Implement ProxyModel (Random Forest surrogate) ✅
  - Create `optimizer/proxy_model.py` with ProxyModel wrapper
  - Encapsulates scikit-optimize's RandomForestRegressor
  - Methods: fit(X, y), predict(X), predict_with_uncertainty(X)
  - Handles training <200ms per research.md
  - **Files**: `recommender/src/boifi_recommender/optimizer/proxy_model.py`
  - **Effort**: 1.5 days
  - **Dependencies**: T002
  - **Acceptance Criteria**:
    - Training time <200ms on sample data ✅
    - Handles 20-dimensional space ✅
    - Unit tests: fitting, prediction accuracy ✅
    - Reference: research.md §2, NFR-005

- [X] [P] T014 [US1] Implement AcquisitionFunction (Expected Improvement) ✅
  - Create `optimizer/acquisition.py` with AcquisitionFunction
  - Implement Expected Improvement (EI) function
  - Methods: compute(predictions, uncertainties) → scores for candidates
  - **Files**: `recommender/src/boifi_recommender/optimizer/acquisition.py`
  - **Effort**: 1 day
  - **Dependencies**: T013
  - **Acceptance Criteria**:
    - EI correctly balances exploration vs exploitation ✅
    - Unit tests: known EI properties (maximizes at uncertain high points) ✅
    - Reference: research.md §2

- [X] [P] T015 [US1] Implement point selection strategy ✅
  - Create `optimizer/point_selector.py` with PointSelector
  - Select next point: maximize EI among candidate set
  - Handle categorical variables: enumerate all combinations
  - **Files**: `recommender/src/boifi_recommender/optimizer/point_selector.py`
  - **Effort**: 1 day
  - **Dependencies**: T013, T014
  - **Acceptance**: Selects valid points, no out-of-bounds proposals ✅

- [X] T016 [US1] Implement OptimizerCore (main Bayesian optimizer) ✅
  - Create `optimizer/core.py` with OptimizerCore class
  - Methods: propose() → FaultPlan, record(plan, score) → None, get_best() → {plan, score}
  - Manage observation history, retrain model after each record()
  - **Files**: `recommender/src/boifi_recommender/optimizer/core.py`
  - **Effort**: 1.5 days
  - **Dependencies**: T012, T013, T014, T015
  - **Acceptance Criteria**:
    - propose() returns valid FaultPlan ✅
    - record() updates model (retrains <200ms) ✅
    - get_best() returns best observed fault ✅
    - Model learns: later trials have higher average scores ✅
    - Unit tests: proposal validity, learning progress ✅
    - Reference: FR-003, FR-004, SC-003

- [X] T017 [US1] Create comprehensive optimizer unit tests ✅
  - Test suite: `tests/unit/test_optimizer_*.py`
  - Test each component: SpaceConverter, ProxyModel, AcquisitionFunction, PointSelector, OptimizerCore
  - Test scenarios: 2D space, 20D space, mixed types
  - Test learning: verify scores improve over 20+ iterations
  - **Files**: `recommender/tests/unit/test_optimizer_*.py`
  - **Effort**: 2 days
  - **Dependencies**: T012-T016
  - **Acceptance Criteria**:
    - ≥90% optimizer code coverage ✅
    - All FR-002 through FR-004 covered ✅
    - Performance tests verify <20ms propose() ✅
    - Reference: Constitution IV (TDD), NFR-001

- [X] T018 [US1] Create integration test: optimizer + model training loop ✅
  - Test full optimization loop: propose → record (20 iterations)
  - Verify model converges to high-scoring region
  - Test with mock observations
  - **Files**: `recommender/tests/integration/test_optimization_loop.py`
  - **Effort**: 1 day
  - **Dependencies**: T016, T017
  - **Acceptance Criteria**:
    - 20-iteration optimization completes ✅
    - Final best_score > initial random scores ✅
    - Reference: SC-003

---

## Phase 4: Response Analyzer (Scoring)

**Goal**: Convert raw observations into standardized [0-10] severity scores

**User Story**: US2 (Response Analysis Produces Standardized Severity Scores)

**Independent Test Criteria**:
- All three scoring dimensions work independently
- Aggregation formula produces [0-10] scores
- Fail-safe defaults for missing data
- All user story acceptance scenarios pass

### Tasks

- [X] [P] T019 [US2] Implement BugScorer ✅
  - Create `analyzer/scorers/bug_scorer.py` with BugScorer class
  - Scoring rules: 5xx≤10, 4xx≤08, ERROR log≤06, error_rate>0≤03, else≤00
  - Implements IScorer interface
  - Default: 0.0 if data missing
  - **Files**: `recommender/src/boifi_recommender/analyzer/scorers/bug_scorer.py`
  - **Effort**: 1 day
  - **Dependencies**: T007
  - **Acceptance Criteria**:
    - Unit tests for all 5 scoring rules ✅
    - Tests for missing data (default to 0.0) ✅
    - Reference: data-model.md §8, research.md §4

- [X] [P] T020 [US2] Implement PerformanceScorer ✅
  - Create `analyzer/scorers/performance_scorer.py` with PerformanceScorer
  - Formula: (latency - baseline) / (threshold - baseline) * 9.0, clamped to [0,10]
  - Parameters from AnalyzerConfig: baseline_ms, threshold_ms
  - Default: 0.0 if latency missing
  - **Files**: `recommender/src/boifi_recommender/analyzer/scorers/performance_scorer.py`
  - **Effort**: 1 day
  - **Dependencies**: T007, T011
  - **Acceptance Criteria**:
    - Unit tests: baseline (0), 50% degradation (4.5), threshold (9), exceeds (10) ✅
    - Tests for missing data ✅
    - Reference: data-model.md §8, research.md §4

- [X] [P] T021 [US2] Implement StructureScorer ✅
  - Create `analyzer/scorers/structure_scorer.py` with StructureScorer
  - Implement trace analysis: edit distance, span count, error detection
  - Scoring: max(span_increase≤03, edit_distance≤05, error_spans≤02)
  - Default: 0.0 if trace missing
  - **Files**: `recommender/src/boifi_recommender/analyzer/scorers/structure_scorer.py`
  - **Effort**: 1.5 days
  - **Dependencies**: T007, T011
  - **Acceptance Criteria**:
    - Unit tests for each sub-score condition ✅
    - Edit distance calculation tests ✅
    - Tests for missing trace data ✅
    - Reference: data-model.md §8, research.md §3

- [X] T022 [US2] Create IScorer interface and base implementation ✅
  - Create `analyzer/scorers/base.py` with IScorer abstract base class
  - Define interface: score(observation, config) → float
  - Create test helper: validate all scorers implement interface
  - **Files**: `recommender/src/boifi_recommender/analyzer/scorers/base.py`
  - **Effort**: 0.5 days
  - **Dependencies**: T007
  - **Acceptance**: All three scorers pass interface contract ✅

- [X] T023 [US2] Implement AnalyzerService (aggregation) ✅
  - Create `analyzer/service.py` with AnalyzerService class
  - Method: calculate_severity(observation, config) → SeverityScore
  - Aggregate three scorers: (bug + perf + struct) / 3
  - Configure weights from AnalyzerConfig
  - Implement fail-safe: missing data → 0.0 for that dimension + warning log
  - **Files**: `recommender/src/boifi_recommender/analyzer/service.py`
  - **Effort**: 1 day
  - **Dependencies**: T019, T020, T021, T022
  - **Acceptance Criteria**:
    - Aggregation formula correct ✅
    - Fail-safe for missing dimensions ✅
    - <100ms calculation time (including all scorers) ✅
    - Unit tests: various observation combinations ✅
    - Reference: FR-005, FR-006, FR-007, NFR-004

- [X] T024 [US2] Create AnalyzerConfig for weights and thresholds ✅
  - Create `analyzer/config.py` with AnalyzerConfig dataclass
  - Parameters: weights (bug, perf, struct), baselines, thresholds
  - Load from env vars or defaults
  - Validate: weights positive, baselines < thresholds
  - **Files**: `recommender/src/boifi_recommender/analyzer/config.py`
  - **Effort**: 0.5 days
  - **Dependencies**: T011
  - **Acceptance**: Configuration loads correctly, validation works ✅

- [X] T025 [US2] Create comprehensive analyzer unit tests ✅
  - Test suite: `tests/unit/test_analyzer_*.py`
  - Test all three scorers independently: edge cases, boundary values
  - Test aggregation: weighted average, clamping to [0,10]
  - Test fail-safe: each missing data type → default
  - **Files**: `recommender/tests/unit/test_analyzer_*.py`
  - **Effort**: 1.5 days
  - **Dependencies**: T019-T024
  - **Acceptance Criteria**:
    - ≥90% analyzer code coverage ✅
    - All acceptance scenarios from spec.md US2 covered ✅
    - Performance test: <100ms on full trace ✅
    - Reference: Constitution IV (TDD)

---

## Phase 5: Session Management & REST API

**Goal**: Implement user-facing API for creating and monitoring optimization sessions

**User Story**: US3 (Real-Time Optimization Progress Monitoring)

**Independent Test Criteria**:
- Session CRUD operations work (create, get, stop)
- API returns correct status and progress
- Session persists across requests
- All user story acceptance scenarios pass

### Tasks

- [X] T026 [US3] Implement SessionManager (in-memory + JSON persistence) ✅
  - Create `services/session_manager.py` with SessionManager class
  - Methods: create_session(), get_session(), list_sessions(), stop_session()
  - Persist to JSON files in .sessions/ directory
  - Thread-safe using locks (RLock for concurrent access)
  - **Files**: `recommender/src/boifi_recommender/services/session_manager.py`
  - **Effort**: 1.5 days
  - **Dependencies**: T008, T011
  - **Acceptance Criteria**:
    - Session creation returns unique UUID ✅
    - get_session() returns correct state ✅
    - Persistence: session survives service restart ✅
    - Thread-safe: concurrent gets/updates don't corrupt ✅
    - Unit tests: CRUD operations, threading, persistence ✅
    - Reference: FR-008, FR-009, data-model.md §1

- [X] T027 [US3] Implement session progress tracking ✅
  - Add progress fields to OptimizationSession: trials_completed, best_score, best_fault
  - Implement estimated_completion_time calculation
  - Update progress as trials complete
  - **Files**: `recommender/src/boifi_recommender/models/session.py` (extend), `services/session_manager.py` (extend)
  - **Effort**: 0.5 days
  - **Dependencies**: T026
  - **Acceptance Criteria**:
    - Progress updates in real-time ✅
    - ETA calculation reasonable ✅
    - Unit tests: progress math ✅
    - Reference: FR-008, SC-004

- [X] T028 [US3] Implement REST API routes for session management ✅
  - Create `api/routes.py` with FastAPI routes
  - Endpoints: POST /v1/optimization/sessions, GET /v1/optimization/sessions/{id}, POST /v1/optimization/sessions/{id}/stop
  - Use SessionManager for CRUD
  - Return SessionStatusResponse with current progress
  - **Files**: `recommender/src/boifi_recommender/api/routes.py`
  - **Effort**: 1 day
  - **Dependencies**: T026, T009
  - **Acceptance Criteria**:
    - All 3 endpoints implemented ✅
    - Responses match openapi.yaml schema ✅
    - Status codes correct (202, 200, 404, 409) ✅
    - Unit tests: happy path, error cases ✅
    - Reference: FR-008, FR-009, openapi.yaml

- [X] T029 [US3] Implement API middleware and error handling ✅
  - Create `api/middleware.py` with error handling, logging, CORS
  - Implement global exception handler for standard errors
  - Format error responses per ErrorResponse schema
  - Add request logging (all endpoints)
  - **Files**: `recommender/src/boifi_recommender/api/middleware.py`
  - **Effort**: 0.5 days
  - **Dependencies**: T028, T009
  - **Acceptance**: Error responses match schema, logging works ✅

- [X] T030 [US3] Create FastAPI application and main entry point ✅
  - Create `main.py` with FastAPI app initialization
  - Mount all routes from api/routes.py
  - Add middleware from api/middleware.py
  - Add /v1/health endpoint
  - **Files**: `recommender/src/boifi_recommender/main.py`
  - **Effort**: 0.5 days
  - **Dependencies**: T028, T029
  - **Acceptance**: `uvicorn main:app` starts successfully ✅

- [X] T031 [US3] Create API endpoint unit and integration tests ✅
  - Test suite: `tests/integration/test_api_endpoints.py`
  - Test all 3 endpoints: request/response schema validation
  - Test error cases: invalid input, session not found, already completed
  - Test status code correctness
  - **Files**: `recommender/tests/integration/test_api_endpoints.py`
  - **Effort**: 1 day
  - **Dependencies**: T028-T030
  - **Acceptance Criteria**:
    - All endpoints tested ✅
    - Responses match openapi.yaml ✅
    - Reference: FR-008, FR-009, Constitution IV (TDD)

---

## Phase 6: Executor Integration & Resilience

**Goal**: Reliably communicate with HFI Executor with automatic retry and circuit breaker

**User Story**: US5 (Executor Integration for Loop Closure)

**Independent Test Criteria**:
- ExecutorClient submits faults and collects observations
- Automatic retry with exponential backoff
- Circuit breaker opens/closes correctly
- All user story acceptance scenarios pass

### Tasks

- [X] T032 [US5] Implement ExecutorClient interface ✅
  - Create `clients/executor_client.py` with IExecutorClient abstract class
  - Define methods: apply_policy(fault_plan) → RawObservation, health_check() → bool
  - Create HttpExecutorClient implementation
  - **Files**: `recommender/src/boifi_recommender/clients/executor_client.py`
  - **Effort**: 1 day
  - **Dependencies**: T007, T011
  - **Acceptance Criteria**:
    - Interface clearly defined ✅
    - HttpExecutorClient implements all methods ✅
    - Unit tests: interface contract ✅
    - Reference: research.md §1, FR-010-FR-014

- [X] T033 [US5] Implement retry strategy with exponential backoff ✅
  - Create retry logic in HttpExecutorClient
  - Exponential backoff: 0.5s, 1.0s, 2.0s, 4.0s, 8.0s (max 5 attempts)
  - Add jitter: ±10% random variation
  - Log each retry attempt
  - **Files**: `recommender/src/boifi_recommender/clients/executor_client.py` (extend)
  - **Effort**: 1 day
  - **Dependencies**: T032
  - **Acceptance Criteria**:
    - Correct delay progression (0.5 * 2^n) ✅
    - Jitter applied ✅
    - Logging works ✅
    - Unit tests: retry timing, jitter distribution ✅
    - Reference: research.md §5, FR-010

- [X] T034 [US5] Implement circuit breaker pattern ✅
  - Create CircuitBreaker class in `clients/executor_client.py`
  - States: CLOSED (normal) → OPEN (too many failures) → HALF_OPEN (testing recovery)
  - Thresholds: 5 consecutive failures, recovery timeout 60s
  - Methods: can_attempt(), record_success(), record_failure()
  - **Files**: `recommender/src/boifi_recommender/clients/executor_client.py` (extend)
  - **Effort**: 1 day
  - **Dependencies**: T032
  - **Acceptance Criteria**:
    - State transitions correct ✅
    - Fails fast when OPEN (no retry) ✅
    - Tests recovery after timeout ✅
    - Unit tests: state machine, threshold behavior ✅
    - Reference: research.md §5, FR-011

- [X] T035 [US5] Implement health checking ✅
  - Add periodic health check: GET /v1/health to Executor
  - Run at startup and every 60s
  - Update circuit breaker on health check failure
  - Cache health status to avoid excessive calls
  - **Files**: `recommender/src/boifi_recommender/clients/executor_client.py` (extend)
  - **Effort**: 0.5 days
  - **Dependencies**: T034
  - **Acceptance**: Health checks run, circuit breaker reacts ✅

- [X] T036 [US5] Implement timeout handling ✅
  - Add connection timeout (5s) and read timeout (30s)
  - Total request timeout: 30s max
  - Timeout treated as transient error (eligible for retry)
  - Log timeout events
  - **Files**: `recommender/src/boifi_recommender/clients/executor_client.py` (extend)
  - **Effort**: 0.5 days
  - **Dependencies**: T032, T033
  - **Acceptance**: Timeouts trigger retries, logged correctly ✅

- [X] T037 [US5] Create ExecutorClient unit tests with mocks ✅
  - Test suite: `tests/unit/test_executor_client.py`
  - Mock Executor responses (success, failure, timeout)
  - Test retry behavior: correct delays, attempt count
  - Test circuit breaker: state transitions, fast-fail
  - Test health check: cache behavior
  - **Files**: `recommender/tests/unit/test_executor_client.py`
  - **Effort**: 1.5 days
  - **Dependencies**: T032-T036
  - **Acceptance Criteria**:
    - ≥90% client code coverage ✅
    - All FR-010 through FR-014 covered ✅
    - Reference: Constitution IV (TDD), Constitution VI (Fault Tolerance)

- [X] T038 [US5] Create contract test: verify Executor API assumptions ✅
  - Test suite: `tests/contract/test_executor_api_contract.py`
  - Assume Executor provides: POST /v1/policies, DELETE /v1/policies, GET /v1/health
  - Test request/response format matches expectations
  - Document assumptions about policy structure
  - **Files**: `recommender/tests/contract/test_executor_api_contract.py`
  - **Effort**: 0.5 days
  - **Dependencies**: T032, T037
  - **Acceptance**: Contract tests pass against real Executor (if available) ✅

---

## Phase 7: Optimization Worker & Session Lifecycle

**Goal**: Implement main optimization loop connecting all components; handle session termination and result persistence

**User Stories**: US4 (Graceful Termination), part of US1 (loop closure)

**Independent Test Criteria**:
- Optimization loop runs end-to-end
- Session can be stopped gracefully
- Results persist after stop
- All user story acceptance scenarios pass

### Tasks

- [X] T039 [US1+US4] Implement OptimizationWorker (main loop) ✅
  - Create `coordinator/worker.py` with OptimizationWorker class
  - Main loop: propose() → execute() → analyze() → record()
  - Handle per-iteration timeout (FR-012)
  - Catch and log all exceptions (fail-safe)
  - Transition session status: PENDING → RUNNING → COMPLETED/FAILED
  - **Files**: `recommender/src/boifi_recommender/coordinator/worker.py`
  - **Effort**: 2 days
  - **Dependencies**: T016, T023, T026, T032
  - **Acceptance Criteria**:
    - Loop structure matches design ✅
    - Session status updates correctly ✅
    - Exception handling prevents crash ✅
    - Unit tests: loop logic with mocks ✅
    - Reference: FR-001, FR-003-FR-009, plan.md Phase Implementation

- [X] T040 [US4] Implement graceful session termination ✅
  - Add stop_flag to OptimizationWorker
  - Stopping: complete current trial, don't start new ones
  - Transition status: RUNNING → STOPPING → COMPLETED
  - Persist final results
  - **Files**: `recommender/src/boifi_recommender/coordinator/worker.py` (extend), `services/session_manager.py` (extend)
  - **Effort**: 0.5 days
  - **Dependencies**: T039, T026
  - **Acceptance Criteria**:
    - stop() transitions correctly ✅
    - Current trial completes (not interrupted) ✅
    - Results available after stop ✅
    - Unit tests: stop sequence ✅
    - Reference: FR-009, US4 acceptance scenarios

- [X] T041 [US4] Implement session result persistence ✅
  - Create `services/persistence.py` with persistence logic
  - Save complete session (with all trial data) to JSON
  - Load session from JSON, resume from last completed trial
  - Handle file I/O errors gracefully
  - **Files**: `recommender/src/boifi_recommender/services/persistence.py`
  - **Effort**: 1 day
  - **Dependencies**: T008, T026
  - **Acceptance Criteria**:
    - Serialization round-trip (object → JSON → object) works ✅
    - Session survives service crash ✅
    - Resume works correctly ✅
    - Unit tests: persistence operations ✅
    - Reference: FR-009, SC-008

- [X] T042 [US4] Create integration test: complete optimization session end-to-end ✅
  - Test suite: `tests/integration/test_optimization_session_lifecycle.py`
  - Scenario 1: Full completion (max trials reached)
  - Scenario 2: Early stop (graceful termination)
  - Scenario 3: Error recovery (Executor temporarily unavailable)
  - Verify results persist and are retrievable
  - **Files**: `recommender/tests/integration/test_optimization_session_lifecycle.py`
  - **Effort**: 1.5 days
  - **Dependencies**: T039-T041
  - **Acceptance Criteria**:
    - All 3 scenarios pass ✅
    - Results match expectations ✅
    - Reference: SC-001, US1-US4 acceptance scenarios

---

## Phase 8: Integration Testing, Performance & Polish

**Goal**: Comprehensive validation, performance optimization, documentation, deployment readiness

**Independent Test Criteria**:
- All components work together correctly
- Performance meets all NFRs (<600ms/iteration, <100ms API, etc.)
- Code coverage ≥70% overall, ≥90% for critical modules
- Documentation complete and up-to-date

### Tasks

- [ ] T043 Create end-to-end integration test with mocked Executor
  - Test suite: `tests/integration/test_e2e_with_mock_executor.py`
  - Mock Executor: returns realistic observations
  - Run 20-trial optimization campaign
  - Verify: best_score increases, no crashes
  - **Files**: `recommender/tests/integration/test_e2e_with_mock_executor.py`
  - **Effort**: 1.5 days
  - **Dependencies**: T039-T042
  - **Acceptance**: E2E test passes, demonstrates full system working

- [ ] T044 [P] Create performance benchmarks for critical paths
  - Benchmark: OptimizerCore.propose() → <20ms
  - Benchmark: AnalyzerService.calculate_severity() → <100ms
  - Benchmark: full iteration (propose+execute+analyze+record) → <600ms
  - Run benchmarks in CI/CD pipeline
  - **Files**: `recommender/tests/performance/bench_*.py`, `.github/workflows/bench.yml`
  - **Effort**: 1 day
  - **Dependencies**: T016, T023, T039
  - **Acceptance Criteria**:
    - All benchmarks pass NFRs
    - Baseline metrics recorded
    - Reference: NFR-001 through NFR-005

- [ ] [P] T045 Write comprehensive documentation
  - **README.md**: Project overview, quick start, architecture summary
  - **docs/DEVELOPMENT.md**: Setup instructions, development workflow
  - **docs/API_REFERENCE.md**: REST API endpoints, examples, error codes
  - **docs/ARCHITECTURE.md**: Design rationale, module responsibilities
  - Update **quickstart.md** with any corrections
  - **Files**: `recommender/README.md`, `recommender/docs/*.md`
  - **Effort**: 1.5 days
  - **Dependencies**: T030, T041
  - **Acceptance**: All docs complete, examples runnable

- [ ] T046 Set up CI/CD pipeline
  - Create `.github/workflows/` files:
    - `test.yml`: Run pytest on every PR
    - `lint.yml`: Run flake8, black, mypy
    - `docker.yml`: Build Docker image
  - Configure code coverage reporting
  - **Files**: `.github/workflows/test.yml`, `.github/workflows/lint.yml`, `.github/workflows/docker.yml`
  - **Effort**: 1 day
  - **Dependencies**: T001-T005
  - **Acceptance**: CI/CD runs on PRs, all status checks required

- [ ] T047 Code quality and refactoring
  - Ensure ≥70% overall code coverage
  - Ensure ≥90% coverage for critical modules (optimizer, analyzer, coordinator)
  - Run static analysis (flake8, mypy)
  - Refactor for clarity, eliminate code duplication
  - **Files**: All source files
  - **Effort**: 1 day
  - **Dependencies**: T043, T044
  - **Acceptance**: Coverage targets met, linting passes

- [ ] T048 Create troubleshooting guide and FAQs
  - Document common issues and solutions
  - Include debugging techniques
  - Provide performance tuning guidance
  - **Files**: `recommender/docs/TROUBLESHOOTING.md`, `recommender/FAQ.md`
  - **Effort**: 0.5 days
  - **Dependencies**: T045
  - **Acceptance**: Guide covers all known issues

---

## Phase 9: Validation Against Constitution

**Goal**: Ensure implementation adheres to BOIFI project constitution

**Independent Test Criteria**:
- All 8 constitution principles verified in code
- Test coverage reflects TDD mandate (>70%)
- Performance meets targets
- Documentation complete

### Tasks

- [ ] T049 Validate Constitution Principle I (Separation of Concerns)
  - Verify: Recommender independently deployable (not coupled to Executor)
  - Verify: Coordinator, Optimizer, Analyzer can be tested separately
  - Document boundaries between modules
  - **Files**: Code review + `recommender/docs/ARCHITECTURE.md`
  - **Effort**: 0.5 days
  - **Dependencies**: All previous tasks
  - **Acceptance**: Design review confirms decoupling

- [ ] T050 Validate Constitution Principle IV (Test-Driven Development)
  - Verify: ≥70% code coverage
  - Verify: All critical paths tested
  - Verify: Integration tests cover major flows
  - **Files**: Coverage reports, test files
  - **Effort**: 0.5 days
  - **Dependencies**: T043, T047
  - **Acceptance**: Coverage thresholds met

- [ ] T051 Validate Constitution Principle V (Performance-First)
  - Verify: All NFRs met (iteration <600ms, API <100ms, etc.)
  - Run performance benchmarks
  - Identify any hot spots for optimization
  - **Files**: Benchmark results, performance analysis
  - **Effort**: 0.5 days
  - **Dependencies**: T044
  - **Acceptance**: All NFRs verified in benchmarks

- [ ] T052 Validate Constitution Principle VI (Fault Tolerance)
  - Verify: ExecutorClient retries on transient errors
  - Verify: Circuit breaker prevents cascading failures
  - Verify: AnalyzerService handles missing data (fail-safe)
  - Document resilience mechanisms
  - **Files**: Code review, resilience documentation
  - **Effort**: 0.5 days
  - **Dependencies**: T032-T041
  - **Acceptance**: Resilience features verified

---

## Summary Table

| Phase | Goal | Tasks | Dependencies | Est. Effort | Parallelizable |
|-------|------|-------|--------------|-------------|-----------------|
| 1 | Setup | T001-T005 | None | 5 days | Yes (all) |
| 2 | Data Models | T006-T011 | T001-T005 | 7.5 days | Yes (T006-T009 parallel) |
| 3 | Optimizer | T012-T018 | T001-T011 | 9 days | Yes (T013-T015 parallel) |
| 4 | Analyzer | T019-T025 | T001-T011 | 7 days | Yes (T019-T021 parallel) |
| 5 | Session API | T026-T031 | T001-T011 | 4.5 days | Mostly serial |
| 6 | Executor Client | T032-T038 | T001-T011 | 5.5 days | Yes (T033-T036 parallel) |
| 7 | Worker Loop | T039-T042 | All above | 5 days | Serial (dependencies) |
| 8 | Integration & Polish | T043-T048 | All above | 6.5 days | Yes (T044, T045 parallel) |
| 9 | Constitution Validation | T049-T052 | All above | 2 days | Yes (all parallel) |

**Total Estimated Effort**: 52 days (1 engineer) or 26 days (2 engineers with parallelization)

---

## Recommended MVP Scope (First 4 weeks)

**Goal**: Demonstrate autonomous optimization working end-to-end

**Included Tasks**:
- Phase 1: T001-T005 (Setup)
- Phase 2: T006-T010 (Data models + tests)
- Phase 3: T012-T018 (Optimizer core)
- Phase 4: T019-T025 (Analyzer)
- Phase 5: T026-T031 (Session API)
- Phase 6: T032-T037 (Executor client) - **critical for MVP**
- Phase 7: T039, T040 (Worker loop, termination)

**Not included**: Performance optimization, documentation depth, CI/CD, constitution validation

**MVP Deliverable**: Working optimization session via REST API, proposing intelligent faults, scoring them, and tracking progress

---

## Task Dependencies Graph

```
T001 (Setup)
  ├── T002 (Dependencies) → T004 (Docker), T005 (Test infra)
  ├── T003 (Structure)
  └── T011 (Config)

T006-T009 (Data Models) → T010 (Model tests)
  ├── T016 (OptimizerCore)
  ├── T026 (SessionManager)
  └── T009 (API models) → T028 (Routes)

T012 → T013 → T014 → T015 → T016 (Optimizer chain)
  └── T017 → T018 (Optimizer tests)

T019, T020, T021 (Scorers - parallel) → T023 (Analyzer) → T025 (Tests)

T026 (SessionManager) → T027 → T028 → T031
  └── T030 (Main.py)

T032 → T033 → T034 → T035, T036 (Client chain)
  └── T037 → T038 (Client tests)

T016, T023, T026, T032 → T039 (Worker)
  └── T040 → T041 → T042 (Termination tests)

All critical path → T043-T048 (Integration & Polish)
  └── T049-T052 (Constitution validation)
```

---

## Running Individual Tasks

Each task is designed to be **independently testable**. Example development flow:

```bash
# Task T006: Data models
cd recommender
git checkout -b task/T006-fault-plan-models
python -m pytest tests/unit/test_models_fault_plan.py -v
# ... implement T006 ...
pytest tests/unit/test_models_fault_plan.py -v  # ✅ Pass

# Task T012: SpaceConverter
git checkout -b task/T012-space-converter
pytest tests/unit/test_space_converter.py -v
# ... implement T012 ...
pytest tests/unit/test_space_converter.py -v  # ✅ Pass

# Commit and PR review
git push origin task/T006-fault-plan-models
# ... create PR, get reviewed ...
git merge main
```

---

## Next: /speckit.implement or Manual Development?

This tasks.md is designed for **manual development** with AI assistance. Each task includes enough context that an LLM (or human developer) can complete it given:
1. The task description
2. File paths
3. References to spec.md, data-model.md, research.md, openapi.yaml

To implement tasks with Copilot assistance:
```
"Help me implement T006 (Fault Plan models) for the recommender system. 
Reference: recommender/specs/002-boifi-recommender/data-model.md section 5-7"
```
