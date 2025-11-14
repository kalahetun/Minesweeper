"""FastAPI application and main entry point"""

import logging
from datetime import datetime
from typing import Optional

from fastapi import FastAPI, HTTPException, status
from fastapi.responses import JSONResponse

from boifi_recommender.models.api_models import (
    CreateSessionRequest,
    SessionStatusResponse,
    StopSessionRequest,
    StopSessionResponse,
    HealthCheckResponse,
    ErrorResponse,
)
from boifi_recommender.models.session import SessionStatus
from boifi_recommender.services.session_manager import SessionManager
from boifi_recommender.config import SETTINGS

# Configure logging
logging.basicConfig(level=SETTINGS.LOG_LEVEL)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(
    title="BOIFI Recommender",
    description="Bayesian Optimizer for Intelligent Fault Injection",
    version="0.1.0",
)

# Global session manager
session_manager = SessionManager(storage_path=SETTINGS.SESSION_STORAGE_PATH)


# ============================================================================
# ERROR HANDLING
# ============================================================================


@app.exception_handler(Exception)
async def global_exception_handler(request, exc):
    """Global exception handler"""
    logger.error(f"Unhandled exception: {exc}", exc_info=True)
    return JSONResponse(
        status_code=500,
        content={
            "error": "INTERNAL_ERROR",
            "message": "An unexpected error occurred",
            "timestamp": datetime.utcnow().isoformat(),
        },
    )


# ============================================================================
# HEALTH CHECK
# ============================================================================


@app.get("/v1/health", response_model=HealthCheckResponse)
async def health_check():
    """Health check endpoint"""
    # TODO: Check executor availability
    return HealthCheckResponse(
        status="healthy",
        timestamp=datetime.utcnow().isoformat(),
        executor_available=True,  # Should be checked
        details={"sessions": len(session_manager.list_sessions())},
    )


# ============================================================================
# SESSION ENDPOINTS
# ============================================================================


@app.post("/v1/optimization/sessions", status_code=202)
async def create_session(request: CreateSessionRequest) -> SessionStatusResponse:
    """Create new optimization session"""
    try:
        session_id = session_manager.create_session(
            service_name=request.service_name,
            search_space_config=request.search_space_config,
            max_trials=request.max_trials,
        )

        session = session_manager.get_session(session_id)
        if not session:
            raise HTTPException(
                status_code=500, detail="Failed to create session"
            )

        return SessionStatusResponse(
            id=session.id,
            service_name=session.service_name,
            status=session.status.value,
            trials_completed=session.trials_completed,
            max_trials=session.max_trials,
            progress_percent=session.progress_percent,
            best_score=session.best_score,
            best_fault=session.best_result.fault_plan if session.best_result else None,
            created_at=session.created_at.isoformat(),
            updated_at=session.updated_at.isoformat(),
        )

    except Exception as e:
        logger.error(f"Failed to create session: {e}")
        raise HTTPException(
            status_code=500,
            detail="Failed to create optimization session",
        )


@app.get("/v1/optimization/sessions/{session_id}", response_model=SessionStatusResponse)
async def get_session(session_id: str) -> SessionStatusResponse:
    """Get session status"""
    session = session_manager.get_session(session_id)
    if not session:
        raise HTTPException(
            status_code=404,
            detail=f"Session {session_id} not found",
        )

    return SessionStatusResponse(
        id=session.id,
        service_name=session.service_name,
        status=session.status.value,
        trials_completed=session.trials_completed,
        max_trials=session.max_trials,
        progress_percent=session.progress_percent,
        best_score=session.best_score,
        best_fault=session.best_result.fault_plan if session.best_result else None,
        created_at=session.created_at.isoformat(),
        updated_at=session.updated_at.isoformat(),
    )


@app.post("/v1/optimization/sessions/{session_id}/stop", status_code=202, response_model=StopSessionResponse)
async def stop_session(
    session_id: str, request: Optional[StopSessionRequest] = None
) -> StopSessionResponse:
    """Stop optimization session"""
    session = session_manager.stop_session(session_id)
    if not session:
        raise HTTPException(
            status_code=404,
            detail=f"Session {session_id} not found",
        )

    return StopSessionResponse(
        id=session.id,
        status=session.status.value,
        message=f"Session {session_id} stopped",
    )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        app,
        host=SETTINGS.SERVER_HOST,
        port=SETTINGS.SERVER_PORT,
        log_level=SETTINGS.LOG_LEVEL.lower(),
    )
