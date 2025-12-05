"""Observation and scoring models"""

from typing import List, Optional, Dict, Any
from pydantic import BaseModel, Field, field_validator
from datetime import datetime


class Span(BaseModel):
    """OpenTelemetry span"""

    traceID: str = Field(..., alias="trace_id")
    spanID: str = Field(..., alias="span_id")
    operationName: str = Field(..., alias="operation_name")
    duration: int = Field(..., ge=0)  # Duration in microseconds
    status: str = Field(default="ok")
    tags: Dict[str, Any] = Field(default_factory=dict)

    class Config:
        populate_by_name = True  # Allow both snake_case and camelCase


class TraceData(BaseModel):
    """Distributed trace data"""

    traceID: str = Field(..., alias="trace_id")
    spans: List[Span] = Field(default_factory=list)

    class Config:
        populate_by_name = True

    def get_span_count(self) -> int:
        """Get number of spans"""
        return len(self.spans)

    def get_error_span_count(self) -> int:
        """Get number of error spans"""
        return sum(1 for span in self.spans if span.status not in ["ok", "unset"])

    def get_total_duration_us(self) -> int:
        """Get total trace duration in microseconds"""
        if not self.spans:
            return 0
        max_duration = max(span.duration for span in self.spans)
        return max_duration


class RawObservation(BaseModel):
    """Raw observation from executor"""

    status_code: Optional[int] = Field(default=None, ge=100, le=599)
    latency_ms: Optional[float] = Field(default=None, ge=0.0)
    error_rate: Optional[float] = Field(default=None, ge=0.0, le=1.0)
    error_logs: List[str] = Field(default_factory=list)
    trace_data: Optional[TraceData] = Field(default=None)
    timestamp: datetime = Field(default_factory=datetime.utcnow)

    @field_validator("error_logs")
    @classmethod
    def validate_error_logs(cls, v):
        """Ensure at least one field is provided"""
        # At least one optional field should be provided (validated in root_validator)
        return v

    def __init__(self, **data):
        """Custom validation: at least one field required"""
        super().__init__(**data)
        # Check that at least one data field is not None
        if all(
            [
                self.status_code is None,
                self.latency_ms is None,
                self.error_rate is None,
                len(self.error_logs) == 0,
                self.trace_data is None,
            ]
        ):
            raise ValueError(
                "At least one of status_code, latency_ms, error_rate, error_logs, or trace_data must be provided"
            )

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return self.model_dump(exclude_none=False)

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RawObservation":
        """Create from dictionary"""
        return cls(**data)


class SeverityScore(BaseModel):
    """Standardized severity score [0-10]"""

    total_score: float = Field(..., ge=0.0, le=10.0)
    bug_score: float = Field(default=0.0, ge=0.0, le=10.0)
    perf_score: float = Field(default=0.0, ge=0.0, le=10.0)
    struct_score: float = Field(default=0.0, ge=0.0, le=10.0)
    components: Dict[str, Any] = Field(default_factory=dict)
    timestamp: datetime = Field(default_factory=datetime.utcnow)

    @field_validator("total_score")
    @classmethod
    def validate_aggregation(cls, v, info):
        """Validate that total score is aggregate of components"""
        # Allow total score to be set independently, will be recalculated during analysis
        return v

    def get_aggregated_score(self) -> float:
        """Calculate aggregated score from components"""
        return (self.bug_score + self.perf_score + self.struct_score) / 3.0

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return self.model_dump()

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "SeverityScore":
        """Create from dictionary"""
        return cls(**data)
