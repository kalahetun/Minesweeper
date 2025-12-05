# Implementation Plan: BOIFI Recommender System

**Branch**: `002-boifi-recommender` | **Date**: 2025-11-14 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-boifi-recommender/spec.md`

**Note**: This is the implementation plan for the Bayesian Optimizer (BO) recommender system, responsible for intelligent chaos testing decision-making.

## Summary

The BOIFI Recommender System is an intelligent decision engine that uses Bayesian optimization to autonomously suggest fault injection combinations. Core responsibilities:

1. **Optimization Coordinator**: Manages session lifecycle, orchestrates propose→execute→analyze→record loop
2. **Bayesian Optimizer Core**: Uses scikit-optimize (Random Forest + Expected Improvement) to propose next promising faults
3. **Response Analyzer**: Converts raw observations (HTTP status, latency, traces) into standardized [0-10] severity scores

Key technical approach:
- Python 3.8+ with FastAPI for REST API
- scikit-optimize for Bayesian optimization algorithm
- Modular design: Coordinator, Optimizer, Analyzer services independently testable
- Resilience: circuit breaker, exponential backoff, fail-safe scoring

## Technical Context

**Language/Version**: Python 3.8+ (aligns with constitution tech stack for Recommender)

**Primary Dependencies**: 
- FastAPI (REST API framework, async-first)
- scikit-optimize (Bayesian optimization with Random Forest surrogate)
- httpx (async HTTP client for Executor communication)
- Pydantic (data validation, type hints)
- pytest (testing framework)

**Storage**: 
- Phase 1: In-memory dictionary (SessionManager) + optional JSON file persistence
- Phase 2+: Redis/PostgreSQL for distributed session management (out of scope for this spec)

**Testing**: pytest with fixtures, mock, unittest.mock for Executor client simulation

**Target Platform**: Linux server (container-deployable via Docker)

**Project Type**: Single backend service (Python package structure)

**Performance Goals**:
- Single iteration <600ms (propose 20ms + execute 500ms + analyze 50ms + record 30ms)
- API latency <100ms (GET /sessions)
- Model training <200ms (Random Forest refit after each trial)
- Analyzer scoring <100ms (all three dimensions)

**Constraints**:
- Concurrent sessions: 10+ on single instance
- Memory: <500MB per session
- Executor availability: handles temporary unavailability with retries + circuit breaker
- No external dependencies for core optimization algorithm (scikit-optimize bundled)

**Scale/Scope**: 
- Single optimization session: 20-200 trials typical
- Fault space dimensions: max 20 (manageable by Random Forest)
- Supported concurrent users: 1-10 SREs running independent optimization campaigns

## Constitution Check

**GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.**

| Principle | Requirement | Plan Status | Notes |
|-----------|-------------|------------|-------|
| **I. Separation of Concerns** | Recommender decoupled from Executor via well-defined client interface; Coordinator/Optimizer/Analyzer independently testable | ✅ PASS | ExecutorClient abstraction; three independent modules in plan |
| **II. Declarative Configuration** | SearchSpaceConfig defined as YAML/JSON with validation; no command-line scripting | ✅ PASS | FR-002 specifies dimension validation; Space Converter in design |
| **III. Dynamic & Real-Time** | Session API supports graceful stop, real-time progress monitoring; no service restart needed | ✅ PASS | FR-008, FR-009 enable live session updates; no restart required |
| **IV. Test-Driven Development** | All modules (Optimizer, Analyzer, Coordinator) must have unit/integration tests; >70% coverage | ✅ PASS | Phase 1 design includes test structure; contracts define testable interfaces |
| **V. Performance-First** | <600ms per iteration; model training <200ms; analyzer <100ms; no hot-path allocations | ✅ PASS | NFRs 001-005 establish performance budgets; scikit-optimize chosen for efficiency |
| **VI. Fault Tolerance & Reliability** | Fail-safe Response Analyzer (missing data → defaults); circuit breaker for Executor; graceful degradation | ✅ PASS | FR-007 (fail-safe scoring), FR-011 (circuit breaker), SC-005 (resilience) |
| **VII. Simplicity & Minimalism** | Use scikit-optimize (not BoTorch); in-memory storage Phase 1; REST/JSON (not gRPC); only specified features | ✅ PASS | Tech stack chosen for simplicity; out-of-scope prevents feature creep |
| **VIII. Temporal Control & Lifecycle** | Session auto-expiration; graceful termination; result persistence | ✅ PASS | FR-009 (graceful stop), optimization loop completion tracked, results persisted |

**Gate Decision**: ✅ PASS - No violations. Recommender design aligns with constitution principles.

## Project Structure

### Documentation (this feature)

```text
specs/002-boifi-recommender/
├── spec.md              # Feature specification (requirements & user stories)
├── plan.md              # This file (implementation planning)
├── research.md          # Phase 0: Research findings on unknowns (TO BE GENERATED)
├── data-model.md        # Phase 1: Entity definitions & data structures (TO BE GENERATED)
├── quickstart.md        # Phase 1: Quick start guide for developers (TO BE GENERATED)
├── contracts/           # Phase 1: API contracts & schemas (TO BE GENERATED)
│   ├── sessions_api.yaml
│   ├── data_models.json
│   └── error_responses.json
├── checklists/requirements.md  # Quality validation (COMPLETED)
└── tasks.md             # Phase 2: Development task breakdown (TO BE GENERATED)
```

### Source Code (recommender project)

```text
recommender/
├── pyproject.toml              # Python project metadata & dependencies
├── requirements.txt            # Python package list
├── Dockerfile                  # Container image definition
├── docker-compose-dev.yaml     # Local development environment
│
├── src/boifi_recommender/
│   ├── __init__.py
│   ├── main.py                 # FastAPI application entry point
│   │
│   ├── models/                 # Data models (Pydantic)
│   │   ├── __init__.py
│   │   ├── fault_plan.py       # FaultPlan, SearchSpaceConfig, Dimension
│   │   ├── observation.py      # RawObservation, SeverityScore
│   │   ├── session.py          # OptimizationSession, SessionStatus
│   │   └── api_models.py       # API request/response models
│   │
│   ├── coordinator/            # Session orchestration & main loop
│   │   ├── __init__.py
│   │   ├── worker.py           # OptimizationWorker (propose→execute→analyze→record)
│   │   └── main_loop.py        # Loop logic & error handling
│   │
│   ├── optimizer/              # Bayesian optimization core
│   │   ├── __init__.py
│   │   ├── core.py             # OptimizerCore (interface & scikit-optimize wrapper)
│   │   ├── space_converter.py  # Convert YAML config → Dimensions
│   │   ├── proxy_model.py      # Random Forest surrogate wrapper
│   │   ├── acquisition.py      # Expected Improvement acquisition function
│   │   └── point_selector.py   # Selection strategy from acquisition results
│   │
│   ├── analyzer/               # Response analysis & scoring
│   │   ├── __init__.py
│   │   ├── service.py          # AnalyzerService (aggregation logic)
│   │   ├── config.py           # AnalyzerConfig (weights, thresholds)
│   │   └── scorers/
│   │       ├── __init__.py
│   │       ├── base.py         # IScorer interface
│   │       ├── bug_scorer.py   # HTTP status, error logs, error rate
│   │       ├── performance_scorer.py  # Latency degradation
│   │       └── structure_scorer.py    # Trace analysis
│   │
│   ├── clients/                # External service clients
│   │   ├── __init__.py
│   │   └── executor_client.py  # HTTP client for Executor API
│   │
│   ├── services/               # High-level business logic
│   │   ├── __init__.py
│   │   ├── session_manager.py  # SessionManager (create/get/stop sessions)
│   │   └── persistence.py      # Result storage & retrieval
│   │
│   ├── api/                    # REST API routes
│   │   ├── __init__.py
│   │   ├── routes.py           # API endpoints (POST/GET /sessions, /sessions/{id}/stop)
│   │   └── middleware.py       # CORS, logging, error handling
│   │
│   ├── utils/                  # Utilities & helpers
│   │   ├── __init__.py
│   │   ├── logger.py           # Structured logging setup
│   │   └── exceptions.py       # Custom exception hierarchy
│   │
│   └── config.py               # Configuration management (env vars, defaults)
│
├── tests/
│   ├── __init__.py
│   ├── conftest.py             # pytest fixtures & configuration
│   │
│   ├── unit/                   # Unit tests (single module)
│   │   ├── test_space_converter.py
│   │   ├── test_bug_scorer.py
│   │   ├── test_performance_scorer.py
│   │   ├── test_structure_scorer.py
│   │   ├── test_analyzer_service.py
│   │   ├── test_optimizer_core.py
│   │   └── test_session_manager.py
│   │
│   ├── integration/            # Integration tests (multiple modules)
│   │   ├── test_optimization_loop.py     # Full propose→execute→analyze→record cycle
│   │   ├── test_session_lifecycle.py     # Create→run→stop→retrieve
│   │   ├── test_executor_client.py       # Mock Executor communication
│   │   └── test_api_endpoints.py         # FastAPI endpoint contracts
│   │
│   └── contract/               # Contract tests (external systems)
│       └── test_executor_api_contract.py # Verify Executor API assumptions

├── docs/
│   ├── DEVELOPMENT.md          # Setup, dependencies, running locally
│   ├── API_REFERENCE.md        # Endpoint documentation
│   └── ARCHITECTURE.md         # System design rationale

└── README.md
```

**Structure Decision**: Single backend service (Option 1) - Python monolithic structure. Recommender is self-contained optimization service; no separate frontend/mobile. Modular internal structure (Coordinator/Optimizer/Analyzer) enables parallel development & testing without cross-service complexity.

## Implementation Phases

### Phase 0: Research & Clarifications
**Deliverable**: `research.md`

Research tasks (if any unknowns exist):
- Executor API compatibility: Policy-based vs extended API mode → resolves FR-010, FR-011 retry/circuit-breaker mapping
- scikit-optimize configuration: Expected Improvement parameters, Random Forest tuning for fault space dimensionality
- Trace analysis: OTEL span format, latency extraction from distributed tracing systems

### Phase 1: Design & Contracts
**Deliverables**: `data-model.md`, `contracts/`, `quickstart.md`

Outputs:
1. **data-model.md**: Entity definitions
   - FaultPlan, RawObservation, SeverityScore, OptimizationSession
   - SearchSpaceConfig, Dimension, validation rules
   - State transitions for sessions (PENDING→RUNNING→STOPPING→COMPLETED/FAILED)

2. **contracts/sessions_api.yaml**: OpenAPI schema
   - POST /v1/optimization/sessions → SessionResponse
   - GET /v1/optimization/sessions/{id} → SessionStatusResponse
   - POST /v1/optimization/sessions/{id}/stop → StopResponse
   - Error responses: 400, 401, 404, 500 with standard error envelope

3. **contracts/executor_client_interface.yaml**: ExecutorClient contract
   - apply_policy(FaultPlan) → RawObservation
   - health_check() → bool
   - Retry behavior, timeout values, circuit breaker thresholds

4. **quickstart.md**: Getting started guide
   - Install dependencies (pip install -r requirements.txt)
   - Run tests (pytest)
   - Start server (uvicorn main:app)
   - Example optimization session (curl POST /sessions with sample config)

### Phase 2: Tasks Breakdown (via /speckit.tasks)
**Deliverable**: `tasks.md`

Will decompose Phase 1 design into specific, assignable development tasks:
- Task groups: Coordinator (4 tasks), Optimizer (5 tasks), Analyzer (5 tasks), Clients (2 tasks), Testing (4 tasks)
- Estimated effort per task (1-3 days)
- Dependencies between tasks (e.g., data models before services)
- Success criteria per task (test coverage, contract adherence)
