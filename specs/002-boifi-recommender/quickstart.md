# Quick Start Guide: BOIFI Recommender Development

**Date**: 2025-11-14  
**Feature**: 002-boifi-recommender  
**Purpose**: Get developers up and running with local development environment

---

## Prerequisites

- Python 3.8+ (verify with `python --version`)
- pip or poetry (verify with `pip --version`)
- Git (verify with `git --version`)
- Docker (optional, for containerized testing)
- curl or HTTPie (for manual API testing)

---

## Installation

### Option A: Using pip (Simple)

```bash
# 1. Clone and navigate to project
git clone <repo-url>
cd wasm_fault_injection/recommender

# 2. Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# 3. Install dependencies
pip install -r requirements.txt

# 4. Verify installation
python -c "import fastapi; import scikit_optimize; print('✅ Dependencies installed')"
```

### Option B: Using Poetry (Recommended)

```bash
# 1. Install poetry (if not already installed)
curl -sSL https://install.python-poetry.org | python3 -

# 2. Clone and navigate
git clone <repo-url>
cd wasm_fault_injection/recommender

# 3. Install dependencies
poetry install

# 4. Activate virtual environment
poetry shell
```

---

## Project Structure Overview

```
recommender/
├── src/boifi_recommender/      # Main application code
│   ├── models/                 # Data models (Pydantic)
│   ├── optimizer/              # Bayesian optimization logic
│   ├── analyzer/               # Response analysis & scoring
│   ├── coordinator/            # Session orchestration
│   ├── clients/                # Executor HTTP client
│   ├── api/                    # FastAPI routes
│   └── main.py                 # Application entry point
│
├── tests/                      # Test suite
│   ├── unit/                   # Single-module tests
│   ├── integration/            # Multi-module tests
│   └── contract/               # External system contracts
│
├── requirements.txt            # Python dependencies
├── pyproject.toml              # Project metadata
├── Dockerfile                  # Container image
└── README.md
```

---

## Running Locally

### Start the Development Server

```bash
# Terminal 1: Start FastAPI application
cd recommender
source venv/bin/activate
python -m uvicorn src.boifi_recommender.main:app --reload --host 0.0.0.0 --port 8000

# Expected output:
# INFO:     Uvicorn running on http://0.0.0.0:8000
# INFO:     Application startup complete
```

The server is now accessible at `http://localhost:8000`

---

### Running Tests

```bash
# Run all tests
pytest

# Run tests with coverage
pytest --cov=src/boifi_recommender --cov-report=html

# Run specific test file
pytest tests/unit/test_optimizer_core.py

# Run tests in watch mode (auto-rerun on file changes)
ptw  # requires pytest-watch: pip install pytest-watch
```

---

## Example: Create Your First Optimization Session

### 1. Define Search Space (YAML)

Create a file `example_search_space.yaml`:

```yaml
name: "Payment Service Fault Space"
description: "Explore delay and error injection combinations"
dimensions:
  - name: delay_ms
    type: integer
    bounds: [100, 5000]
    default: 1000
  - name: error_code
    type: categorical
    values: [500, 502, 503, 504]
    default: 500
  - name: request_timeout_sec
    type: real
    bounds: [0.5, 10.0]
    default: 5.0
```

### 2. Create Optimization Session

```bash
# Using curl
curl -X POST http://localhost:8000/v1/optimization/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "payment-service",
    "search_space": {
      "name": "Payment Service Fault Space",
      "dimensions": [
        {
          "name": "delay_ms",
          "type": "integer",
          "bounds": [100, 5000]
        },
        {
          "name": "error_code",
          "type": "categorical",
          "values": [500, 502, 503]
        }
      ]
    },
    "max_trials": 10,
    "time_budget_sec": 1800
  }'

# Response (example):
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "PENDING",
  "created_at": "2025-11-14T10:30:00Z"
}
```

Save the `session_id` for the next steps.

### 3. Monitor Progress

```bash
# Poll session status
SESSION_ID="550e8400-e29b-41d4-a716-446655440000"

curl http://localhost:8000/v1/optimization/sessions/$SESSION_ID

# Response (example):
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "RUNNING",
  "trials_completed": 5,
  "max_trials": 10,
  "best_score": 7.8,
  "best_fault": {
    "service": "payment-service",
    "fault_type": "delay",
    "delay_ms": 2500
  },
  "estimated_completion_time": "2025-11-14T10:50:00Z"
}
```

### 4. Stop Session (Optional)

```bash
curl -X POST http://localhost:8000/v1/optimization/sessions/$SESSION_ID/stop

# Response:
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "STOPPING"
}
```

---

## Development Workflow

### 1. Implement a New Feature

```bash
# Create a feature branch
git checkout -b feature/my-feature

# Create test file first (TDD)
touch tests/unit/test_my_feature.py
# Write failing test
vim tests/unit/test_my_feature.py

# Run tests to confirm failure
pytest tests/unit/test_my_feature.py -v

# Implement feature in src/
vim src/boifi_recommender/my_feature.py

# Run tests again - should pass
pytest tests/unit/test_my_feature.py -v

# Run all tests to ensure no regressions
pytest
```

### 2. Check Code Quality

```bash
# Format code
black src/boifi_recommender tests

# Lint (style checking)
flake8 src/boifi_recommender tests

# Type checking
mypy src/boifi_recommender

# All-in-one check
make lint  # if Makefile exists
```

### 3. Commit and Push

```bash
git add src/boifi_recommender tests
git commit -m "feat: add my feature with tests"
git push origin feature/my-feature

# Create pull request
# Ensure all tests pass in CI/CD
# Request review from team
```

---

## Key Components to Understand

### OptimizerCore (optimizer/core.py)

```python
from src.boifi_recommender.optimizer.core import OptimizerCore
from src.boifi_recommender.models.fault_plan import FaultPlan

# Initialize optimizer
optimizer = OptimizerCore(
    search_space=search_space_config,
    n_initial_points=5,  # Random trials before fitting model
    random_state=42
)

# Propose next fault
fault_plan = optimizer.propose()
print(f"Proposed fault: {fault_plan}")

# Record observation
severity_score = 7.5
optimizer.record(fault_plan, severity_score)

# Get best fault found so far
best = optimizer.get_best()
print(f"Best fault: {best['fault']}, score: {best['score']}")
```

### AnalyzerService (analyzer/service.py)

```python
from src.boifi_recommender.analyzer.service import AnalyzerService
from src.boifi_recommender.models.observation import RawObservation

analyzer = AnalyzerService()

# Create sample observation
observation = RawObservation(
    status_code=500,
    latency_ms=2500,
    error_rate=0.15,
    logs=["ERROR: database timeout"],
    trace_data=None
)

# Compute severity score
score = analyzer.calculate_severity(observation)
print(f"Severity score: {score}")  # Should be high (around 7-8)
```

### ExecutorClient (clients/executor_client.py)

```python
from src.boifi_recommender.clients.executor_client import HttpExecutorClient

# Initialize client
client = HttpExecutorClient(
    executor_host="http://localhost:8080",
    timeout_sec=30.0,
    max_retries=3
)

# Submit fault and collect observation
fault_plan = FaultPlan(
    service="payment-service",
    fault_type="delay",
    duration_ms=30000,
    delay_ms=2500
)

observation = client.apply_and_observe(fault_plan)
print(f"Observation: {observation}")
```

---

## Troubleshooting

### Issue: "ModuleNotFoundError: No module named 'boifi_recommender'"

**Solution**: Ensure virtual environment is activated and dependencies are installed:
```bash
source venv/bin/activate
pip install -r requirements.txt
```

### Issue: "Connection refused" when calling Executor

**Reason**: Executor service is not running

**Solution**: Start Executor separately or mock it in tests:
```python
from unittest.mock import Mock
client = Mock()
client.apply_and_observe.return_value = mock_observation
```

### Issue: Tests are slow

**Reason**: Optimizer is training real models

**Solution**: Use fixtures with pre-trained models or mock them:
```python
@pytest.fixture
def mock_optimizer():
    optimizer = Mock(spec=OptimizerCore)
    optimizer.propose.return_value = sample_fault_plan
    return optimizer
```

### Issue: "permission denied" when running scripts

**Solution**: Make script executable:
```bash
chmod +x script.sh
```

---

## Next Steps

1. **Understand Data Models**: Read `data-model.md` to understand entities
2. **Review API Contracts**: Check `contracts/sessions_api.yaml` for endpoint specifications
3. **Write a Unit Test**: Create a simple test in `tests/unit/` following existing patterns
4. **Implement a Scorer**: Add a new scoring dimension to `analyzer/scorers/`
5. **Create Integration Test**: Write an end-to-end test combining multiple components

---

## Common Commands

```bash
# Install additional dependencies
pip install <package-name>

# Update requirements file
pip freeze > requirements.txt

# Run specific test
pytest tests/unit/test_optimizer_core.py::test_propose_returns_valid_fault_plan -v

# Debug test execution
pytest tests/unit/test_analyzer_service.py -v -s  # -s shows print statements

# Generate test coverage report
pytest --cov=src/boifi_recommender --cov-report=html
# Then open htmlcov/index.html in browser

# Clean up bytecode and cache
find . -type d -name __pycache__ -exec rm -r {} +
find . -type f -name "*.pyc" -delete
```

---

## Resources

- **FastAPI Documentation**: https://fastapi.tiangolo.com/
- **pytest Documentation**: https://docs.pytest.org/
- **scikit-optimize Guide**: https://scikit-optimize.github.io/
- **OpenTelemetry Spec**: https://opentelemetry.io/docs/reference/specification/

---

## Getting Help

1. Check existing issues in the project
2. Review test examples in `tests/` directory
3. Read docstrings in source code
4. Ask in team Slack channel or discussion board

Happy coding! 🚀
