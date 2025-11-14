# Prompt for /speckit.tasks: BOIFI Recommender

## Context

You are generating a development task breakdown for the BOIFI Recommender System (Bayesian Optimizer for chaos testing). This feature has comprehensive specification (spec.md) and implementation plan (plan.md) already completed.

## Input Materials Available

- **spec.md**: 5 user stories (3 P1 + 2 P2), 15 functional requirements, 8 success criteria
- **plan.md**: Technical context, constitution check, project structure, implementation phases
- **research.md**: Technical decisions on executor API, optimizer algorithm, scoring formulas, resilience patterns
- **data-model.md**: 10 core entities with validation rules and relationships
- **quickstart.md**: Developer setup and usage guide
- **openapi.yaml**: REST API contract specification

## Task Decomposition Strategy

### Phase 1: Infrastructure & Data Models (Week 1-2)
Foundation that enables all subsequent work. Tasks should include:
- Project setup (Python environment, dependencies, Docker)
- Data model implementation (Pydantic models for all 10 entities)
- Basic test fixtures and factories

### Phase 2: Core Optimizer (Week 3-4)
Implement the intelligence engine independently. Tasks should include:
- SpaceConverter (YAML config → scikit-optimize dimensions)
- OptimizerCore (Bayesian optimization wrapper)
- ProxyModel and AcquisitionFunction

### Phase 3: Response Analysis (Week 5-6)
Implement the scoring engine independently. Tasks should include:
- Three independent scorers (Bug, Performance, Structure)
- AnalyzerService (aggregation logic)
- Comprehensive scoring validation tests

### Phase 4: Executor Integration (Week 7-8)
Implement communication with HFI Executor. Tasks should include:
- ExecutorClient (HTTP client with resilience)
- Circuit breaker implementation
- Retry and timeout handling

### Phase 5: Session Management & API (Week 9-10)
Implement user-facing functionality. Tasks should include:
- SessionManager (create/get/stop sessions)
- OptimizationWorker (main optimization loop)
- FastAPI routes and middleware

### Phase 6: Testing & Integration (Week 11-12)
Comprehensive testing to ensure quality. Tasks should include:
- Unit tests for each component
- Integration tests (multi-component flows)
- Contract tests with mocked Executor
- End-to-end optimization session test

## Task Attributes to Include

For each task:
1. **Title**: Clear, action-oriented (e.g., "Implement BugScorer with unit tests")
2. **Description**: What needs to be built, with reference to spec requirements
3. **Success Criteria**: How to verify the task is complete (testable outcomes)
4. **Estimated Effort**: 1-3 days (for team planning)
5. **Dependencies**: Which tasks must complete first
6. **Acceptance Criteria**: Reference to specific FRs or success criteria from spec

## Key Principles

- **Independence**: Each task should be independently testable and deployable
- **Test-Driven**: All tasks include unit/integration test creation as acceptance criteria
- **Constitution Alignment**: Refer to constitution principles (especially TDD, Performance-First, Fault Tolerance)
- **Clear Scope**: Avoid vague requirements; reference specific entities, methods, test cases
- **Team Parallelization**: Identify tasks that can run in parallel (e.g., BugScorer and PerformanceScorer)

## Output Format

Generate `tasks.md` with:
- Executive summary (estimated total effort, parallelizable task groups)
- Task list organized by phase
- For each task: ID, title, description, effort, dependencies, acceptance criteria, references to spec

## Do Not Include

- Implementation code or detailed algorithms (those come during actual development)
- Architecture diagrams (already in plan.md)
- Tutorial content (that's in quickstart.md)
- Repeated content from spec/plan/research (reference them instead)

---

**Ready to generate comprehensive task breakdown for the Recommender System.**
