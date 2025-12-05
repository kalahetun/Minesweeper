"""Response analyzer - scoring system"""

from typing import Dict, Any, Optional
import logging

logger = logging.getLogger(__name__)


class BugScorer:
    """Score responses based on errors and failures"""

    @staticmethod
    def score(observation: Dict[str, Any]) -> float:
        """Calculate bug score [0-10]"""
        status_code = observation.get("status_code")
        error_rate = observation.get("error_rate", 0.0)
        error_logs = observation.get("error_logs", [])

        # 5xx errors get highest score
        if status_code and 500 <= status_code < 600:
            return 10.0

        # 4xx errors
        if status_code and 400 <= status_code < 500:
            return 8.0

        # Error logs
        if error_logs and len(error_logs) > 0:
            return 6.0

        # Error rate > 0
        if error_rate and error_rate > 0.0:
            return 3.0

        return 0.0


class PerformanceScorer:
    """Score responses based on latency degradation"""

    def __init__(self, baseline_ms: float = 100.0, threshold_ms: float = 500.0):
        self.baseline_ms = baseline_ms
        self.threshold_ms = threshold_ms

    def score(self, observation: Dict[str, Any]) -> float:
        """Calculate performance score [0-10]"""
        latency_ms = observation.get("latency_ms")

        if latency_ms is None:
            return 0.0

        # Linear interpolation between baseline and threshold
        if latency_ms <= self.baseline_ms:
            return 0.0

        if latency_ms >= self.threshold_ms:
            return 9.0

        # Linear: (latency - baseline) / (threshold - baseline) * 9.0
        ratio = (latency_ms - self.baseline_ms) / (
            self.threshold_ms - self.baseline_ms
        )
        return min(10.0, ratio * 9.0)


class StructureScorer:
    """Score responses based on trace structure changes"""

    @staticmethod
    def score(observation: Dict[str, Any]) -> float:
        """Calculate structure score [0-10]"""
        trace_data = observation.get("trace_data")

        if trace_data is None:
            return 0.0

        spans = trace_data.get("spans", [])
        span_count = len(spans)

        # Expected baseline: ~2 spans
        if span_count <= 2:
            span_score = 0.0
        elif span_count <= 5:
            span_score = 3.0
        else:
            span_score = 6.0

        # Check for error spans
        error_spans = [
            s for s in spans if s.get("status", "ok") not in ["ok", "unset"]
        ]
        error_score = 2.0 if len(error_spans) > 0 else 0.0

        # Edit distance (simplified - just span count diff)
        edit_distance_score = min(5.0, span_count * 0.5)

        # Max of all sub-scores
        return max(span_score, error_score, edit_distance_score)


class AnalyzerService:
    """Main response analysis service"""

    def __init__(
        self,
        baseline_ms: float = 100.0,
        threshold_ms: float = 500.0,
        bug_weight: float = 1.0,
        perf_weight: float = 1.0,
        struct_weight: float = 1.0,
    ):
        self.bug_scorer = BugScorer()
        self.perf_scorer = PerformanceScorer(baseline_ms, threshold_ms)
        self.struct_scorer = StructureScorer()

        # Normalize weights
        total_weight = bug_weight + perf_weight + struct_weight
        self.bug_weight = bug_weight / total_weight
        self.perf_weight = perf_weight / total_weight
        self.struct_weight = struct_weight / total_weight

    def calculate_severity(self, observation: Dict[str, Any]) -> Dict[str, Any]:
        """Calculate overall severity score"""
        # Get component scores
        bug_score = self.bug_scorer.score(observation)
        perf_score = self.perf_scorer.score(observation)
        struct_score = self.struct_scorer.score(observation)

        # Weighted aggregation
        total_score = (
            self.bug_weight * bug_score
            + self.perf_weight * perf_score
            + self.struct_weight * struct_score
        )

        # Clamp to [0, 10]
        total_score = max(0.0, min(10.0, total_score))

        return {
            "total_score": total_score,
            "bug_score": bug_score,
            "perf_score": perf_score,
            "struct_score": struct_score,
            "components": {
                "bug_weight": self.bug_weight,
                "perf_weight": self.perf_weight,
                "struct_weight": self.struct_weight,
            },
        }
