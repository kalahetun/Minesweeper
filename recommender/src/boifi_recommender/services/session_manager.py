"""Session management service"""

import json
import logging
from typing import Dict, List, Optional, Any
from pathlib import Path
from threading import RLock
from uuid import uuid4
from datetime import datetime

from boifi_recommender.models.session import (
    OptimizationSession,
    SessionStatus,
)

logger = logging.getLogger(__name__)


class SessionManager:
    """Manage optimization sessions with persistence"""

    def __init__(self, storage_path: str = ".sessions"):
        self.storage_path = Path(storage_path)
        self.storage_path.mkdir(parents=True, exist_ok=True)
        self._sessions: Dict[str, OptimizationSession] = {}
        self._lock = RLock()
        self._load_all_sessions()

    def _load_all_sessions(self) -> None:
        """Load all sessions from disk"""
        with self._lock:
            for session_file in self.storage_path.glob("*.json"):
                try:
                    with open(session_file) as f:
                        data = json.load(f)
                        session = OptimizationSession.from_dict(data)
                        self._sessions[session.id] = session
                except Exception as e:
                    logger.warning(f"Failed to load session from {session_file}: {e}")

    def _save_session(self, session: OptimizationSession) -> None:
        """Save session to disk"""
        session_file = self.storage_path / f"{session.id}.json"
        try:
            with open(session_file, "w") as f:
                json.dump(session.to_dict(), f, indent=2, default=str)
        except Exception as e:
            logger.error(f"Failed to save session {session.id}: {e}")

    def create_session(
        self,
        service_name: str,
        search_space_config: Dict[str, Any],
        max_trials: int = 100,
    ) -> str:
        """Create new optimization session"""
        with self._lock:
            session = OptimizationSession(
                service_name=service_name,
                search_space_config=search_space_config,
                max_trials=max_trials,
            )
            self._sessions[session.id] = session
            self._save_session(session)
            logger.info(f"Created session {session.id} for {service_name}")
            return session.id

    def get_session(self, session_id: str) -> Optional[OptimizationSession]:
        """Get session by ID"""
        with self._lock:
            return self._sessions.get(session_id)

    def list_sessions(self) -> List[OptimizationSession]:
        """List all sessions"""
        with self._lock:
            return list(self._sessions.values())

    def update_session(self, session: OptimizationSession) -> None:
        """Update session"""
        with self._lock:
            self._sessions[session.id] = session
            self._save_session(session)

    def stop_session(self, session_id: str) -> Optional[OptimizationSession]:
        """Stop session"""
        with self._lock:
            session = self._sessions.get(session_id)
            if session:
                if session.status == SessionStatus.RUNNING:
                    session.transition_to_stopping()
                    session.transition_to_completed()
                    self._save_session(session)
                    logger.info(f"Stopped session {session_id}")
            return session

    def delete_session(self, session_id: str) -> bool:
        """Delete session"""
        with self._lock:
            if session_id in self._sessions:
                del self._sessions[session_id]
                session_file = self.storage_path / f"{session_id}.json"
                try:
                    session_file.unlink()
                except Exception as e:
                    logger.warning(f"Failed to delete session file: {e}")
                return True
            return False
