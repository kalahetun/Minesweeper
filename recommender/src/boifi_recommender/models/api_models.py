"""API request/response models"""

from typing import List, Optional, Dict, Any, Literal
from pydantic import BaseModel, Field
from enum import Enum


class CreateSessionRequest(BaseModel):
    """Request to create optimization session"""

    service_name: str = Field(..., min_length=1, max_length=255, description="Target service name")
    search_space_config: Dict[str, Any] = Field(
        ..., description="Search space configuration (dimensions, constraints)"
    )
    max_trials: int = Field(default=100, ge=1, le=10000, description="Maximum trials to run")


class SessionStatusResponse(BaseModel):
    """Response with session status"""

    id: str = Field(..., description="Unique session ID")
    service_name: str = Field(..., description="Target service name")
    status: str = Field(..., description="Session status: PENDING, RUNNING, STOPPING, COMPLETED, FAILED")
    trials_completed: int = Field(..., ge=0, description="Number of completed trials")
    max_trials: int = Field(..., ge=1, description="Maximum trials")
    progress_percent: float = Field(..., ge=0.0, le=100.0, description="Progress percentage [0-100]")
    best_score: float = Field(..., ge=0.0, le=10.0, description="Best severity score found")
    best_fault: Optional[Dict[str, Any]] = Field(
        default=None, description="Fault plan with best score"
    )
    created_at: str = Field(..., description="Session creation timestamp (ISO 8601)")
    updated_at: str = Field(..., description="Last update timestamp (ISO 8601)")


class StopSessionRequest(BaseModel):
    """Request to stop session"""

    reason: Optional[str] = Field(
        default=None, max_length=1000, description="Reason for stopping"
    )


class StopSessionResponse(BaseModel):
    """Response when stopping session"""

    id: str = Field(..., description="Session ID")
    status: str = Field(..., description="New session status")
    message: str = Field(..., description="Status message")


class ErrorResponse(BaseModel):
    """Standardized error response"""

    error: str = Field(..., description="Error type/code")
    message: str = Field(..., description="Human-readable error message")
    details: Optional[Dict[str, Any]] = Field(default=None, description="Additional error details")
    timestamp: str = Field(..., description="Error timestamp (ISO 8601)")


class HealthCheckResponse(BaseModel):
    """Health check response"""

    status: str = Field(..., description="Health status: healthy, unhealthy, degraded")
    timestamp: str = Field(..., description="Check timestamp (ISO 8601)")
    executor_available: bool = Field(..., description="Whether Executor is available")
    details: Optional[Dict[str, Any]] = Field(default=None, description="Additional health details")
