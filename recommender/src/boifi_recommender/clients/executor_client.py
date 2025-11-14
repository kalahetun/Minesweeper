"""Executor client for communicating with HFI Executor"""

import asyncio
import logging
import time
from enum import Enum
from typing import Optional, Dict, Any

import httpx

logger = logging.getLogger(__name__)


class CircuitBreakerState(Enum):
    """Circuit breaker states"""

    CLOSED = "closed"  # Normal operation
    OPEN = "open"  # Too many failures
    HALF_OPEN = "half_open"  # Testing recovery


class CircuitBreaker:
    """Circuit breaker pattern implementation"""

    def __init__(self, failure_threshold: int = 5, recovery_timeout: int = 60):
        self.failure_threshold = failure_threshold
        self.recovery_timeout = recovery_timeout
        self.state = CircuitBreakerState.CLOSED
        self.failure_count = 0
        self.last_failure_time = None

    def can_attempt(self) -> bool:
        """Check if request can be attempted"""
        if self.state == CircuitBreakerState.CLOSED:
            return True

        if self.state == CircuitBreakerState.OPEN:
            # Check if recovery timeout has elapsed
            if (
                self.last_failure_time
                and time.time() - self.last_failure_time > self.recovery_timeout
            ):
                self.state = CircuitBreakerState.HALF_OPEN
                logger.info("Circuit breaker transitioning to HALF_OPEN")
                return True
            return False

        # HALF_OPEN - allow one attempt
        return True

    def record_success(self) -> None:
        """Record successful request"""
        if self.state == CircuitBreakerState.HALF_OPEN:
            self.state = CircuitBreakerState.CLOSED
            self.failure_count = 0
            logger.info("Circuit breaker back to CLOSED")
        elif self.state == CircuitBreakerState.CLOSED:
            self.failure_count = 0

    def record_failure(self) -> None:
        """Record failed request"""
        self.failure_count += 1
        self.last_failure_time = time.time()

        if self.failure_count >= self.failure_threshold:
            self.state = CircuitBreakerState.OPEN
            logger.warning(
                f"Circuit breaker opened after {self.failure_count} failures"
            )


class ExecutorClient:
    """HTTP client for communicating with HFI Executor"""

    def __init__(
        self,
        executor_url: str = "http://localhost:8001",
        timeout: float = 30.0,
        max_retries: int = 5,
        base_delay: float = 0.5,
        max_delay: float = 8.0,
        jitter_percent: float = 10.0,
    ):
        self.executor_url = executor_url
        self.timeout = timeout
        self.max_retries = max_retries
        self.base_delay = base_delay
        self.max_delay = max_delay
        self.jitter_percent = jitter_percent
        self.circuit_breaker = CircuitBreaker()
        self.client = httpx.AsyncClient(timeout=timeout)

    async def apply_policy(self, fault_plan: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Apply fault policy and collect observation"""
        if not self.circuit_breaker.can_attempt():
            logger.error("Circuit breaker is OPEN, refusing request")
            return None

        for attempt in range(self.max_retries):
            try:
                # Convert fault_plan to policy format
                policy = self._fault_plan_to_policy(fault_plan)

                # POST to executor
                response = await self.client.post(
                    f"{self.executor_url}/v1/policies",
                    json=policy,
                    timeout=self.timeout,
                )

                if response.status_code == 200:
                    self.circuit_breaker.record_success()
                    return response.json()

                # Transient errors - retry
                if response.status_code >= 500:
                    logger.warning(
                        f"Executor returned {response.status_code}, retrying..."
                    )
                    await self._delay_with_backoff(attempt)
                    continue

                # Permanent errors - fail
                logger.error(f"Executor returned {response.status_code}")
                self.circuit_breaker.record_failure()
                return None

            except (httpx.TimeoutException, httpx.ConnectError) as e:
                logger.warning(f"Executor request timed out (attempt {attempt + 1}): {e}")
                self.circuit_breaker.record_failure()
                await self._delay_with_backoff(attempt)
                continue

            except Exception as e:
                logger.error(f"Executor request failed: {e}")
                self.circuit_breaker.record_failure()
                return None

        logger.error(f"All {self.max_retries} retry attempts exhausted")
        return None

    async def health_check(self) -> bool:
        """Check executor health"""
        try:
            response = await self.client.get(
                f"{self.executor_url}/v1/health",
                timeout=5.0,
            )
            is_healthy = response.status_code == 200
            if is_healthy:
                self.circuit_breaker.record_success()
            else:
                self.circuit_breaker.record_failure()
            return is_healthy

        except Exception as e:
            logger.warning(f"Health check failed: {e}")
            self.circuit_breaker.record_failure()
            return False

    async def _delay_with_backoff(self, attempt: int) -> None:
        """Exponential backoff with jitter"""
        delay = min(self.max_delay, self.base_delay * (2 ** attempt))

        # Add jitter
        import random

        jitter = delay * (self.jitter_percent / 100.0)
        jittered_delay = delay + random.uniform(-jitter, jitter)

        logger.info(f"Backing off for {jittered_delay:.2f}s before retry")
        await asyncio.sleep(jittered_delay)

    @staticmethod
    def _fault_plan_to_policy(fault_plan: Dict[str, Any]) -> Dict[str, Any]:
        """Convert fault plan to executor policy format"""
        # Map fault plan fields to executor policy format
        return {
            "service": fault_plan.get("service"),
            "fault_type": fault_plan.get("fault_type"),
            "duration_ms": fault_plan.get("duration_ms"),
            "delay_ms": fault_plan.get("delay_ms"),
            "abort_probability": fault_plan.get("abort_probability"),
            "error_code": fault_plan.get("error_code"),
        }

    async def close(self) -> None:
        """Close client"""
        await self.client.aclose()
