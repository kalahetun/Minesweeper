"""Pytest configuration and fixtures"""

import json
import tempfile
from datetime import datetime, timedelta
from pathlib import Path
from typing import Dict, Any
from unittest.mock import Mock, AsyncMock

import pytest
from pydantic import BaseModel

# Configure pytest markers
def pytest_configure(config):
    config.addinivalue_line("markers", "unit: Unit tests")
    config.addinivalue_line("markers", "integration: Integration tests")
    config.addinivalue_line("markers", "contract: Contract tests")
    config.addinivalue_line("markers", "performance: Performance benchmarks")
    config.addinivalue_line("markers", "slow: Slow tests")


# ============================================================================
# SESSION FIXTURES
# ============================================================================

@pytest.fixture
def temp_session_dir():
    """Temporary directory for session storage"""
    with tempfile.TemporaryDirectory() as tmpdir:
        yield Path(tmpdir)


# ============================================================================
# DATA MODEL FIXTURES
# ============================================================================

@pytest.fixture
def sample_fault_plan() -> Dict[str, Any]:
    """Sample fault plan for testing"""
    return {
        "service": "payment-service",
        "fault_type": "delay",
        "duration_ms": 30000,
        "delay_ms": 500,
        "abort_probability": 0.0,
        "error_code": None,
    }


@pytest.fixture
def sample_observation() -> Dict[str, Any]:
    """Sample observation from executor"""
    return {
        "status_code": 200,
        "latency_ms": 150.0,
        "error_rate": 0.0,
        "error_logs": [],
        "trace_data": {
            "traceID": "trace-001",
            "spans": [
                {
                    "spanID": "span-001",
                    "traceID": "trace-001",
                    "operationName": "http.request",
                    "duration": 100000,  # microseconds
                    "status": "ok",
                    "tags": {},
                },
                {
                    "spanID": "span-002",
                    "traceID": "trace-001",
                    "operationName": "db.query",
                    "duration": 50000,
                    "status": "ok",
                    "tags": {},
                },
            ],
        },
    }


@pytest.fixture
def sample_search_space() -> Dict[str, Any]:
    """Sample search space configuration"""
    return {
        "name": "payment-service-faults",
        "dimensions": [
            {
                "name": "fault_type",
                "type": "categorical",
                "values": ["delay", "abort", "error_injection"],
                "default": "delay",
            },
            {
                "name": "delay_ms",
                "type": "integer",
                "bounds": [10, 1000],
                "default": 100,
            },
            {
                "name": "error_rate",
                "type": "real",
                "bounds": [0.0, 1.0],
                "default": 0.0,
            },
        ],
    }


@pytest.fixture
def sample_session_request() -> Dict[str, Any]:
    """Sample session creation request"""
    return {
        "service_name": "payment-service",
        "search_space_config": {
            "name": "payment-faults",
            "dimensions": [
                {
                    "name": "fault_type",
                    "type": "categorical",
                    "values": ["delay", "abort"],
                    "default": "delay",
                },
                {
                    "name": "delay_ms",
                    "type": "integer",
                    "bounds": [10, 500],
                    "default": 100,
                },
            ],
        },
        "max_trials": 20,
        "target_metrics": ["latency", "error_rate"],
    }


# ============================================================================
# MOCK FIXTURES
# ============================================================================

@pytest.fixture
def mock_executor_client():
    """Mock ExecutorClient"""
    client = AsyncMock()
    client.apply_policy = AsyncMock(
        return_value={
            "status_code": 200,
            "latency_ms": 150.0,
            "error_rate": 0.0,
            "error_logs": [],
            "trace_data": None,
        }
    )
    client.health_check = AsyncMock(return_value=True)
    return client


@pytest.fixture
def mock_session_manager():
    """Mock SessionManager"""
    manager = Mock()
    manager.create_session = Mock(return_value="session-uuid-001")
    manager.get_session = Mock(
        return_value={
            "id": "session-uuid-001",
            "service_name": "test-service",
            "status": "RUNNING",
            "trials_completed": 5,
            "best_score": 7.5,
        }
    )
    manager.list_sessions = Mock(return_value=[])
    manager.stop_session = Mock(return_value=None)
    return manager


@pytest.fixture
def mock_optimizer():
    """Mock OptimizerCore"""
    optimizer = Mock()
    optimizer.propose = Mock(
        return_value={
            "service": "test-service",
            "fault_type": "delay",
            "duration_ms": 30000,
            "delay_ms": 100,
        }
    )
    optimizer.record = Mock(return_value=None)
    optimizer.get_best = Mock(
        return_value={
            "plan": {
                "service": "test-service",
                "fault_type": "delay",
                "duration_ms": 30000,
                "delay_ms": 250,
            },
            "score": 8.5,
        }
    )
    return optimizer


@pytest.fixture
def mock_analyzer():
    """Mock AnalyzerService"""
    analyzer = Mock()
    analyzer.calculate_severity = Mock(
        return_value={
            "total_score": 5.0,
            "bug_score": 3.0,
            "perf_score": 5.0,
            "struct_score": 6.0,
        }
    )
    return analyzer


# ============================================================================
# FACTORY FIXTURES
# ============================================================================

class FaultPlanFactory:
    """Factory for creating fault plans for testing"""

    @staticmethod
    def create(
        service: str = "test-service",
        fault_type: str = "delay",
        duration_ms: int = 30000,
        delay_ms: int = 100,
        abort_probability: float = 0.0,
        error_code: int = None,
    ) -> Dict[str, Any]:
        return {
            "service": service,
            "fault_type": fault_type,
            "duration_ms": duration_ms,
            "delay_ms": delay_ms,
            "abort_probability": abort_probability,
            "error_code": error_code,
        }

    @staticmethod
    def create_batch(count: int = 5) -> list:
        plans = []
        for i in range(count):
            plans.append(
                FaultPlanFactory.create(
                    delay_ms=100 + (i * 50),
                    abort_probability=0.0 + (i * 0.1),
                )
            )
        return plans


class ObservationFactory:
    """Factory for creating observations for testing"""

    @staticmethod
    def create(
        status_code: int = 200,
        latency_ms: float = 150.0,
        error_rate: float = 0.0,
        error_logs: list = None,
        trace_data: Dict = None,
    ) -> Dict[str, Any]:
        return {
            "status_code": status_code,
            "latency_ms": latency_ms,
            "error_rate": error_rate,
            "error_logs": error_logs or [],
            "trace_data": trace_data,
        }

    @staticmethod
    def create_failure(status_code: int = 500, error_logs: list = None) -> Dict[str, Any]:
        return ObservationFactory.create(
            status_code=status_code,
            latency_ms=2000.0,
            error_rate=1.0,
            error_logs=error_logs or ["Internal server error"],
        )

    @staticmethod
    def create_slow(latency_ms: float = 1000.0) -> Dict[str, Any]:
        return ObservationFactory.create(latency_ms=latency_ms)


class SessionFactory:
    """Factory for creating sessions for testing"""

    @staticmethod
    def create(
        session_id: str = "session-001",
        service_name: str = "test-service",
        status: str = "RUNNING",
        trials_completed: int = 0,
        best_score: float = 0.0,
    ) -> Dict[str, Any]:
        return {
            "id": session_id,
            "service_name": service_name,
            "status": status,
            "trials_completed": trials_completed,
            "best_score": best_score,
            "created_at": datetime.utcnow().isoformat(),
            "updated_at": datetime.utcnow().isoformat(),
        }


@pytest.fixture
def fault_plan_factory():
    """Factory fixture for fault plans"""
    return FaultPlanFactory


@pytest.fixture
def observation_factory():
    """Factory fixture for observations"""
    return ObservationFactory


@pytest.fixture
def session_factory():
    """Factory fixture for sessions"""
    return SessionFactory
