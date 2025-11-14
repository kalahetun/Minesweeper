"""Optimization worker - main loop"""

import asyncio
import logging
from typing import Optional

from boifi_recommender.models.session import OptimizationSession, SessionStatus, Trial
from boifi_recommender.optimizer.core import OptimizerCore
from boifi_recommender.analyzer.service import AnalyzerService
from boifi_recommender.services.session_manager import SessionManager
from boifi_recommender.clients.executor_client import ExecutorClient

logger = logging.getLogger(__name__)


class OptimizationWorker:
    """Main optimization worker that runs the feedback loop"""

    def __init__(
        self,
        session_id: str,
        session_manager: SessionManager,
        executor_client: ExecutorClient,
        analyzer: AnalyzerService,
        optimizer: OptimizerCore,
    ):
        self.session_id = session_id
        self.session_manager = session_manager
        self.executor_client = executor_client
        self.analyzer = analyzer
        self.optimizer = optimizer
        self.stop_flag = False

    async def run(self) -> None:
        """Run optimization loop"""
        session = self.session_manager.get_session(self.session_id)
        if not session:
            logger.error(f"Session {self.session_id} not found")
            return

        try:
            # Transition to RUNNING
            session.transition_to_running()
            self.session_manager.update_session(session)

            # Main loop
            for trial_id in range(session.max_trials):
                if self.stop_flag:
                    logger.info(f"Stop flag set, ending optimization")
                    break

                logger.info(f"Trial {trial_id + 1}/{session.max_trials}")

                # 1. Propose fault plan
                fault_plan = self.optimizer.propose()
                logger.debug(f"Proposed fault plan: {fault_plan}")

                # 2. Execute fault
                observation = await self.executor_client.apply_policy(fault_plan)
                if observation is None:
                    logger.warning("Executor returned None, retrying next trial")
                    continue

                # 3. Analyze response
                score_result = self.analyzer.calculate_severity(observation)
                severity_score = score_result.get("total_score", 0.0)
                logger.info(f"Severity score: {severity_score:.2f}")

                # 4. Record trial
                trial = Trial(
                    trial_id=trial_id,
                    fault_plan=fault_plan,
                    observation=observation,
                    severity_score=severity_score,
                )
                session.add_trial(trial)
                self.optimizer.record(fault_plan, severity_score)

                # Update session
                self.session_manager.update_session(session)

            # Transition to COMPLETED
            session.transition_to_completed()
            self.session_manager.update_session(session)
            logger.info(f"Optimization completed. Best score: {session.best_score:.2f}")

        except Exception as e:
            logger.error(f"Optimization failed: {e}", exc_info=True)
            session.transition_to_failed(reason=str(e))
            self.session_manager.update_session(session)

    def stop(self) -> None:
        """Signal to stop optimization"""
        logger.info("Stop requested")
        self.stop_flag = True
