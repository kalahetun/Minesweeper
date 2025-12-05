"""Bayesian optimizer core components"""

from typing import List, Dict, Any, Tuple, Optional
import numpy as np
from abc import ABC, abstractmethod
import logging

logger = logging.getLogger(__name__)


class SpaceConverter:
    """Convert SearchSpaceConfig to scikit-optimize Dimension objects"""

    @staticmethod
    def convert(search_space_config: Dict[str, Any]) -> List[Any]:
        """Convert search space config to skopt Dimensions"""
        from skopt.space import Categorical, Integer, Real

        dimensions = []
        for dim_config in search_space_config.get("dimensions", []):
            dim_type = dim_config.get("type")
            name = dim_config.get("name")

            if dim_type == "categorical":
                values = dim_config.get("values", [])
                dimensions.append(Categorical(values, name=name))

            elif dim_type == "integer":
                bounds = dim_config.get("bounds", [])
                dimensions.append(Integer(bounds[0], bounds[1], name=name))

            elif dim_type == "real":
                bounds = dim_config.get("bounds", [])
                dimensions.append(Real(bounds[0], bounds[1], name=name))

        return dimensions

    @staticmethod
    def point_to_dict(dimensions_config: List[Dict], point: List[Any]) -> Dict[str, Any]:
        """Convert skopt point to fault plan dictionary"""
        result = {}
        for i, dim_config in enumerate(dimensions_config):
            result[dim_config["name"]] = point[i]
        return result

    @staticmethod
    def validate_bounds(dimensions_config: List[Dict]) -> bool:
        """Validate dimension bounds"""
        for dim in dimensions_config:
            dim_type = dim.get("type")

            if dim_type == "categorical":
                values = dim.get("values", [])
                if len(values) != len(set(values)):
                    return False

            elif dim_type in ["integer", "real"]:
                bounds = dim.get("bounds", [])
                if len(bounds) != 2 or bounds[0] >= bounds[1]:
                    return False

        return True


class ProxyModel:
    """Random Forest surrogate model for Bayesian optimization"""

    def __init__(self, n_estimators: int = 100, random_state: int = 42):
        from sklearn.ensemble import RandomForestRegressor

        self.model = RandomForestRegressor(
            n_estimators=n_estimators,
            random_state=random_state,
            n_jobs=-1,
        )
        self.is_fitted = False

    def fit(self, X: np.ndarray, y: np.ndarray) -> None:
        """Fit the surrogate model"""
        if len(X) == 0:
            return

        self.model.fit(X, y)
        self.is_fitted = True

    def predict(self, X: np.ndarray) -> np.ndarray:
        """Predict mean values"""
        if not self.is_fitted:
            return np.zeros(len(X))
        return self.model.predict(X)

    def predict_with_uncertainty(self, X: np.ndarray) -> Tuple[np.ndarray, np.ndarray]:
        """Predict with uncertainty estimates (from ensemble variance)"""
        if not self.is_fitted:
            return np.zeros(len(X)), np.ones(len(X))

        # Get predictions from individual trees
        predictions = np.array([tree.predict(X) for tree in self.model.estimators_])
        mean = np.mean(predictions, axis=0)
        std = np.std(predictions, axis=0)

        return mean, std


class AcquisitionFunction:
    """Expected Improvement acquisition function"""

    @staticmethod
    def expected_improvement(
        predictions: np.ndarray,
        uncertainties: np.ndarray,
        best_value: float,
        xi: float = 0.01,
    ) -> np.ndarray:
        """
        Calculate Expected Improvement

        Args:
            predictions: Mean predictions from surrogate model
            uncertainties: Prediction uncertainties
            best_value: Best observed value so far
            xi: Exploration trade-off parameter

        Returns:
            EI scores for each candidate point
        """
        improvement = predictions - best_value - xi

        # Avoid division by zero
        with np.errstate(divide="warn"):
            Z = improvement / (uncertainties + 1e-9)

        from scipy.stats import norm

        ei = improvement * norm.cdf(Z) + uncertainties * norm.pdf(Z)
        ei[uncertainties == 0.0] = 0.0

        return ei


class PointSelector:
    """Select next point to evaluate based on acquisition function"""

    @staticmethod
    def select_next_point(
        space_dimensions: List[Any],
        surrogate_model: ProxyModel,
        best_value: float,
        n_candidates: int = 1000,
        random_state: int = 42,
    ) -> List[Any]:
        """
        Select next point using Expected Improvement

        Args:
            space_dimensions: skopt Dimension objects
            surrogate_model: Fitted surrogate model
            best_value: Best observed value so far
            n_candidates: Number of candidate points to generate
            random_state: Random seed

        Returns:
            Next point to evaluate
        """
        from skopt.utils import cartesian_product

        np.random.seed(random_state)

        # Generate candidate points
        candidates = []
        for dim in space_dimensions:
            if dim.name.endswith("0"):  # Hack to detect categorical
                try:
                    if hasattr(dim, "categories"):
                        candidates.append(dim.categories)
                except:
                    pass

        if not candidates:
            # Random sampling
            candidates = []
            for dim in space_dimensions:
                if hasattr(dim, "categories"):  # Categorical
                    candidates.append(
                        np.random.choice(dim.categories, size=n_candidates)
                    )
                else:  # Numerical
                    candidates.append(
                        np.random.uniform(dim.low, dim.high, size=n_candidates)
                    )
        else:
            # Cartesian product for categorical + random for numerical
            candidates = cartesian_product([dim.categories for dim in space_dimensions])

        if len(candidates) == 0:
            return None

        # Evaluate EI on candidates
        predictions, uncertainties = surrogate_model.predict_with_uncertainty(
            candidates
        )
        ei_scores = AcquisitionFunction.expected_improvement(
            predictions, uncertainties, best_value
        )

        # Select point with highest EI
        best_idx = np.argmax(ei_scores)
        return candidates[best_idx]


class OptimizerCore:
    """Main Bayesian optimizer"""

    def __init__(self, search_space_config: Dict[str, Any]):
        self.search_space_config = search_space_config
        self.dimensions = SpaceConverter.convert(search_space_config)
        self.surrogate_model = ProxyModel()
        self.observation_history_X = []
        self.observation_history_y = []
        self.best_value = 0.0
        self.best_point = None

    def propose(self) -> Dict[str, Any]:
        """Propose next fault plan to test"""
        if len(self.observation_history_y) == 0:
            # Initial random point
            import random

            point = []
            for dim in self.dimensions:
                if hasattr(dim, "categories"):  # Categorical
                    point.append(random.choice(dim.categories))
                else:  # Numerical
                    point.append(
                        dim.low + random.random() * (dim.high - dim.low)
                    )
            return SpaceConverter.point_to_dict(
                self.search_space_config["dimensions"], point
            )
        else:
            # Use surrogate model + EI
            next_point = PointSelector.select_next_point(
                self.dimensions, self.surrogate_model, self.best_value
            )

            if next_point is None:
                # Fallback to random
                import random

                next_point = [
                    random.choice(dim.categories)
                    if hasattr(dim, "categories")
                    else dim.low + random.random() * (dim.high - dim.low)
                    for dim in self.dimensions
                ]

            return SpaceConverter.point_to_dict(
                self.search_space_config["dimensions"], next_point
            )

    def record(self, fault_plan: Dict[str, Any], score: float) -> None:
        """Record trial result and update model"""
        # Convert plan to point
        point = []
        dims = self.search_space_config["dimensions"]
        for dim in dims:
            point.append(fault_plan.get(dim["name"]))

        self.observation_history_X.append(point)
        self.observation_history_y.append(score)

        # Update best
        if score > self.best_value:
            self.best_value = score
            self.best_point = fault_plan.copy()

        # Retrain model
        if len(self.observation_history_X) >= 2:
            X = np.array(self.observation_history_X)
            y = np.array(self.observation_history_y)
            self.surrogate_model.fit(X, y)

    def get_best(self) -> Dict[str, Any]:
        """Get best result so far"""
        return {
            "plan": self.best_point or {},
            "score": self.best_value,
        }
