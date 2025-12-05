"""Configuration management for BOIFI Recommender System"""

import os
from typing import Optional
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Application settings loaded from environment variables with defaults"""

    # Server configuration
    SERVER_HOST: str = "0.0.0.0"
    SERVER_PORT: int = 8000
    DEBUG: bool = False

    # Executor configuration
    EXECUTOR_HOST: str = "localhost"
    EXECUTOR_PORT: int = 8001
    EXECUTOR_TIMEOUT_SECONDS: int = 30

    # Optimizer configuration
    OPTIMIZER_MAX_TRIALS: int = 100
    OPTIMIZER_INITIAL_POINTS: int = 5
    OPTIMIZER_TIMEOUT_SECONDS: int = 600  # 10 minutes per iteration

    # Analyzer configuration
    ANALYZER_BASELINE_MS: float = 100.0  # Baseline latency for performance scoring
    ANALYZER_THRESHOLD_MS: float = 500.0  # Threshold latency (target: 9.0 score)
    ANALYZER_BUG_WEIGHT: float = 1.0
    ANALYZER_PERF_WEIGHT: float = 1.0
    ANALYZER_STRUCT_WEIGHT: float = 1.0

    # Retry configuration (exponential backoff)
    RETRY_BASE_DELAY_SECONDS: float = 0.5
    RETRY_MAX_DELAY_SECONDS: float = 8.0
    RETRY_MAX_ATTEMPTS: int = 5
    RETRY_JITTER_PERCENT: float = 10.0

    # Circuit breaker configuration
    CIRCUIT_BREAKER_FAILURE_THRESHOLD: int = 5
    CIRCUIT_BREAKER_RECOVERY_TIMEOUT_SECONDS: int = 60

    # Health check configuration
    HEALTH_CHECK_INTERVAL_SECONDS: int = 60

    # Session persistence
    SESSION_STORAGE_PATH: str = ".sessions"
    SESSION_AUTO_CLEANUP_DAYS: int = 7

    # Logging
    LOG_LEVEL: str = "INFO"
    LOG_FORMAT: str = "json"  # json or text

    class Config:
        env_file = ".env"
        case_sensitive = True

    @property
    def executor_base_url(self) -> str:
        """Get Executor base URL"""
        return f"http://{self.EXECUTOR_HOST}:{self.EXECUTOR_PORT}"

    @property
    def executor_timeout(self) -> float:
        """Get Executor timeout as float"""
        return float(self.EXECUTOR_TIMEOUT_SECONDS)


# Global settings instance
SETTINGS = Settings()
