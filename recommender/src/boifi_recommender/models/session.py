"""Optimization session and trial models"""

from typing import List, Optional, Dict, Any, Literal
from pydantic import BaseModel, Field
from datetime import datetime
from enum import Enum
from uuid import uuid4


class SessionStatus(str, Enum):
    """Session lifecycle states"""

    PENDING = "PENDING"
    RUNNING = "RUNNING"
    STOPPING = "STOPPING"
    COMPLETED = "COMPLETED"
    FAILED = "FAILED"


class Trial(BaseModel):
    """Single trial in optimization session"""

    trial_id: int = Field(..., ge=0)
    fault_plan: Dict[str, Any] = Field(...)
    observation: Optional[Dict[str, Any]] = Field(default=None)
    severity_score: Optional[float] = Field(default=None, ge=0.0, le=10.0)
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    status: str = Field(default="completed")

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return self.model_dump()

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "Trial":
        """Create from dictionary"""
        return cls(**data)


class BestResult(BaseModel):
    """Best result found during optimization"""

    fault_plan: Dict[str, Any] = Field(...)
    severity_score: float = Field(..., ge=0.0, le=10.0)
    trial_id: int = Field(..., ge=0)
    timestamp: datetime = Field(default_factory=datetime.utcnow)

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return self.model_dump()

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "BestResult":
        """Create from dictionary"""
        return cls(**data)


class OptimizationSession(BaseModel):
    """Optimization session"""

    id: str = Field(default_factory=lambda: str(uuid4()))
    service_name: str = Field(..., min_length=1, max_length=255)
    status: SessionStatus = Field(default=SessionStatus.PENDING)
    search_space_config: Dict[str, Any] = Field(...)
    max_trials: int = Field(default=100, ge=1)
    trials: List[Trial] = Field(default_factory=list)
    best_result: Optional[BestResult] = Field(default=None)
    created_at: datetime = Field(default_factory=datetime.utcnow)
    updated_at: datetime = Field(default_factory=datetime.utcnow)
    started_at: Optional[datetime] = Field(default=None)
    completed_at: Optional[datetime] = Field(default=None)

    class Config:
        use_enum_values = True

    @property
    def trials_completed(self) -> int:
        """Get number of completed trials"""
        return len(self.trials)

    @property
    def best_score(self) -> float:
        """Get best score so far"""
        if self.best_result is None:
            return 0.0
        return self.best_result.severity_score

    @property
    def progress_percent(self) -> float:
        """Get progress percentage [0-100]"""
        if self.max_trials == 0:
            return 0.0
        return min(100.0, (self.trials_completed / self.max_trials) * 100.0)

    def add_trial(self, trial: Trial) -> None:
        """Add completed trial"""
        self.trials.append(trial)
        self.updated_at = datetime.utcnow()

        # Update best result
        if trial.severity_score is not None:
            if self.best_result is None or trial.severity_score > self.best_result.severity_score:
                self.best_result = BestResult(
                    fault_plan=trial.fault_plan,
                    severity_score=trial.severity_score,
                    trial_id=trial.trial_id,
                )

    def is_complete(self) -> bool:
        """Check if session is complete"""
        return self.status in [SessionStatus.COMPLETED, SessionStatus.FAILED]

    def transition_to_running(self) -> None:
        """Transition PENDING → RUNNING"""
        if self.status != SessionStatus.PENDING:
            raise ValueError(f"Cannot transition from {self.status} to RUNNING")
        self.status = SessionStatus.RUNNING
        self.started_at = datetime.utcnow()
        self.updated_at = datetime.utcnow()

    def transition_to_stopping(self) -> None:
        """Transition RUNNING → STOPPING"""
        if self.status != SessionStatus.RUNNING:
            raise ValueError(f"Cannot transition from {self.status} to STOPPING")
        self.status = SessionStatus.STOPPING
        self.updated_at = datetime.utcnow()

    def transition_to_completed(self) -> None:
        """Transition RUNNING/STOPPING → COMPLETED"""
        if self.status not in [SessionStatus.RUNNING, SessionStatus.STOPPING]:
            raise ValueError(f"Cannot transition from {self.status} to COMPLETED")
        self.status = SessionStatus.COMPLETED
        self.completed_at = datetime.utcnow()
        self.updated_at = datetime.utcnow()

    def transition_to_failed(self, reason: str = "Unknown error") -> None:
        """Transition any → FAILED"""
        self.status = SessionStatus.FAILED
        self.completed_at = datetime.utcnow()
        self.updated_at = datetime.utcnow()

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return self.model_dump(mode="json")

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "OptimizationSession":
        """Create from dictionary"""
        return cls(**data)
