"""Unit tests for data models"""

import pytest
from datetime import datetime
from pydantic import ValidationError

from boifi_recommender.models.fault_plan import (
    FaultPlan,
    SearchSpaceConfig,
    CategoricalDimension,
    IntegerDimension,
    RealDimension,
    FaultType,
)
from boifi_recommender.models.observation import (
    RawObservation,
    SeverityScore,
    Span,
    TraceData,
)
from boifi_recommender.models.session import (
    OptimizationSession,
    Trial,
    BestResult,
    SessionStatus,
)


class TestFaultPlan:
    """Tests for FaultPlan model"""

    def test_valid_delay_fault(self, sample_fault_plan):
        """Test creating valid delay fault"""
        plan = FaultPlan(**sample_fault_plan)
        assert plan.service == "payment-service"
        assert plan.fault_type == FaultType.DELAY
        assert plan.duration_ms == 30000
        assert plan.delay_ms == 500

    def test_delay_must_be_less_than_duration(self):
        """Test that delay must be less than duration"""
        with pytest.raises(ValidationError) as exc_info:
            FaultPlan(
                service="test",
                fault_type="delay",
                duration_ms=1000,
                delay_ms=1500,
            )
        assert "less than duration" in str(exc_info.value)

    def test_error_code_must_be_4xx_or_5xx(self):
        """Test error code validation"""
        with pytest.raises(ValidationError):
            FaultPlan(
                service="test",
                fault_type="error_injection",
                duration_ms=30000,
                error_code=200,  # Invalid: 200 is not 4xx or 5xx
            )

    def test_duration_must_be_positive(self):
        """Test duration must be positive"""
        with pytest.raises(ValidationError):
            FaultPlan(
                service="test",
                fault_type="delay",
                duration_ms=0,
            )

    def test_abort_probability_in_valid_range(self):
        """Test abort probability must be [0, 1]"""
        plan = FaultPlan(
            service="test",
            fault_type="abort",
            duration_ms=30000,
            abort_probability=0.5,
        )
        assert plan.abort_probability == 0.5

        with pytest.raises(ValidationError):
            FaultPlan(
                service="test",
                fault_type="abort",
                duration_ms=30000,
                abort_probability=1.5,
            )

    def test_serialization(self, sample_fault_plan):
        """Test FaultPlan serialization"""
        plan = FaultPlan(**sample_fault_plan)
        plan_dict = plan.to_dict()
        assert plan_dict["service"] == "payment-service"
        assert plan_dict["fault_type"] == "delay"

        # Deserialize
        plan2 = FaultPlan.from_dict(plan_dict)
        assert plan2.service == plan.service


class TestSearchSpaceConfig:
    """Tests for SearchSpaceConfig model"""

    def test_valid_search_space(self, sample_search_space):
        """Test creating valid search space"""
        config = SearchSpaceConfig(**sample_search_space)
        assert config.name == "payment-service-faults"
        assert len(config.dimensions) == 3

    def test_dimension_names_must_be_unique(self):
        """Test that dimension names are unique"""
        with pytest.raises(ValidationError) as exc_info:
            SearchSpaceConfig(
                name="test",
                dimensions=[
                    CategoricalDimension(
                        name="fault_type",
                        values=["delay", "abort"],
                        default="delay",
                    ),
                    CategoricalDimension(
                        name="fault_type",  # Duplicate name
                        values=["error", "timeout"],
                        default="error",
                    ),
                ],
            )
        assert "unique" in str(exc_info.value).lower()

    def test_get_dimension_by_name(self, sample_search_space):
        """Test getting dimension by name"""
        config = SearchSpaceConfig(**sample_search_space)
        dim = config.get_dimension_by_name("fault_type")
        assert dim is not None
        assert isinstance(dim, CategoricalDimension)

    def test_get_nonexistent_dimension(self, sample_search_space):
        """Test getting nonexistent dimension"""
        config = SearchSpaceConfig(**sample_search_space)
        dim = config.get_dimension_by_name("nonexistent")
        assert dim is None


class TestDimensions:
    """Tests for dimension models"""

    def test_categorical_dimension_validation(self):
        """Test categorical dimension validation"""
        dim = CategoricalDimension(
            name="fault_type",
            values=["delay", "abort", "error"],
            default="delay",
        )
        assert dim.default == "delay"

        # Default must be in values
        with pytest.raises(ValidationError):
            CategoricalDimension(
                name="fault_type",
                values=["delay", "abort"],
                default="invalid",
            )

    def test_integer_dimension_bounds(self):
        """Test integer dimension bounds validation"""
        dim = IntegerDimension(
            name="delay_ms",
            bounds=(10, 1000),
            default=500,
        )
        assert dim.default == 500

        # Bounds must be ordered
        with pytest.raises(ValidationError):
            IntegerDimension(
                name="delay_ms",
                bounds=(1000, 10),  # Wrong order
                default=500,
            )

        # Default must be in bounds
        with pytest.raises(ValidationError):
            IntegerDimension(
                name="delay_ms",
                bounds=(10, 1000),
                default=2000,  # Out of bounds
            )

    def test_real_dimension_bounds(self):
        """Test real dimension bounds validation"""
        dim = RealDimension(
            name="error_rate",
            bounds=(0.0, 1.0),
            default=0.5,
        )
        assert dim.default == 0.5


class TestRawObservation:
    """Tests for RawObservation model"""

    def test_valid_observation(self, sample_observation):
        """Test creating valid observation"""
        obs = RawObservation(**sample_observation)
        assert obs.status_code == 200
        assert obs.latency_ms == 150.0

    def test_at_least_one_field_required(self):
        """Test that at least one field is required"""
        with pytest.raises(ValueError) as exc_info:
            RawObservation()
        assert "at least one" in str(exc_info.value).lower()

    def test_error_rate_in_valid_range(self):
        """Test error rate must be [0, 1]"""
        obs = RawObservation(error_rate=0.5)
        assert obs.error_rate == 0.5

        with pytest.raises(ValidationError):
            RawObservation(error_rate=1.5)

    def test_status_code_in_valid_range(self):
        """Test status code must be [100, 599]"""
        obs = RawObservation(status_code=200)
        assert obs.status_code == 200

        with pytest.raises(ValidationError):
            RawObservation(status_code=700)


class TestSeverityScore:
    """Tests for SeverityScore model"""

    def test_valid_score(self):
        """Test creating valid severity score"""
        score = SeverityScore(
            total_score=5.5,
            bug_score=3.0,
            perf_score=5.0,
            struct_score=8.0,
        )
        assert score.total_score == 5.5
        assert score.get_aggregated_score() == pytest.approx(5.333, abs=0.01)

    def test_score_range(self):
        """Test score must be [0, 10]"""
        with pytest.raises(ValidationError):
            SeverityScore(total_score=15.0)

        score = SeverityScore(total_score=0.0)
        assert score.total_score == 0.0

        score = SeverityScore(total_score=10.0)
        assert score.total_score == 10.0


class TestOptimizationSession:
    """Tests for OptimizationSession model"""

    def test_create_session(self, sample_session_request):
        """Test creating session"""
        session = OptimizationSession(
            service_name="test-service",
            search_space_config=sample_session_request["search_space_config"],
            max_trials=20,
        )
        assert session.status == SessionStatus.PENDING
        assert session.trials_completed == 0
        assert session.best_score == 0.0

    def test_state_transitions(self):
        """Test state transition logic"""
        session = OptimizationSession(
            service_name="test",
            search_space_config={},
        )

        # PENDING → RUNNING
        session.transition_to_running()
        assert session.status == SessionStatus.RUNNING
        assert session.started_at is not None

        # RUNNING → STOPPING
        session.transition_to_stopping()
        assert session.status == SessionStatus.STOPPING

        # STOPPING → COMPLETED
        session.transition_to_completed()
        assert session.status == SessionStatus.COMPLETED
        assert session.completed_at is not None

    def test_invalid_state_transition(self):
        """Test that invalid transitions raise errors"""
        session = OptimizationSession(
            service_name="test",
            search_space_config={},
        )

        # Cannot go from PENDING to STOPPING directly
        with pytest.raises(ValueError):
            session.transition_to_stopping()

    def test_add_trial(self):
        """Test adding trials updates best result"""
        session = OptimizationSession(
            service_name="test",
            search_space_config={},
            max_trials=10,
        )

        trial1 = Trial(trial_id=0, fault_plan={}, severity_score=5.0)
        session.add_trial(trial1)
        assert session.trials_completed == 1
        assert session.best_score == 5.0

        trial2 = Trial(trial_id=1, fault_plan={}, severity_score=7.0)
        session.add_trial(trial2)
        assert session.trials_completed == 2
        assert session.best_score == 7.0

    def test_progress_calculation(self):
        """Test progress percentage calculation"""
        session = OptimizationSession(
            service_name="test",
            search_space_config={},
            max_trials=10,
        )

        assert session.progress_percent == 0.0

        for i in range(10):
            trial = Trial(trial_id=i, fault_plan={}, severity_score=5.0)
            session.add_trial(trial)

        assert session.progress_percent == 100.0

    def test_serialization(self):
        """Test session serialization"""
        session = OptimizationSession(
            service_name="test",
            search_space_config={"name": "test-space"},
            max_trials=20,
        )

        session_dict = session.to_dict()
        assert session_dict["service_name"] == "test"

        # Deserialize
        session2 = OptimizationSession.from_dict(session_dict)
        assert session2.service_name == session.service_name
        assert session2.id == session.id
