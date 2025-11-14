"""Fault plan and search space configuration models"""

from typing import List, Optional, Dict, Any, Literal, Union
from pydantic import BaseModel, Field, field_validator
from enum import Enum


class FaultType(str, Enum):
    """Supported fault types"""

    DELAY = "delay"
    ABORT = "abort"
    ERROR_INJECTION = "error_injection"


class DimensionType(str, Enum):
    """Search space dimension types"""

    CATEGORICAL = "categorical"
    INTEGER = "integer"
    REAL = "real"


class CategoricalDimension(BaseModel):
    """Categorical dimension in search space"""

    name: str = Field(..., min_length=1, max_length=255)
    type: Literal["categorical"] = "categorical"
    values: List[str] = Field(..., min_items=2)
    default: str = Field(...)

    @field_validator("default")
    @classmethod
    def validate_default_in_values(cls, v, info):
        """Ensure default value is in values list"""
        if "values" in info.data and v not in info.data["values"]:
            raise ValueError(f"Default value '{v}' must be in values list")
        return v

    @field_validator("values")
    @classmethod
    def validate_unique_values(cls, v):
        """Ensure all values are unique"""
        if len(v) != len(set(v)):
            raise ValueError("Dimension values must be unique")
        return v


class IntegerDimension(BaseModel):
    """Integer dimension in search space"""

    name: str = Field(..., min_length=1, max_length=255)
    type: Literal["integer"] = "integer"
    bounds: tuple[int, int] = Field(...)
    default: int = Field(...)

    @field_validator("bounds")
    @classmethod
    def validate_bounds_ordered(cls, v):
        """Ensure bounds are ordered correctly"""
        if v[0] >= v[1]:
            raise ValueError(f"Lower bound {v[0]} must be less than upper bound {v[1]}")
        return v

    @field_validator("default")
    @classmethod
    def validate_default_in_bounds(cls, v, info):
        """Ensure default is within bounds"""
        if "bounds" in info.data:
            bounds = info.data["bounds"]
            if not (bounds[0] <= v <= bounds[1]):
                raise ValueError(
                    f"Default value {v} must be between {bounds[0]} and {bounds[1]}"
                )
        return v


class RealDimension(BaseModel):
    """Real (continuous) dimension in search space"""

    name: str = Field(..., min_length=1, max_length=255)
    type: Literal["real"] = "real"
    bounds: tuple[float, float] = Field(...)
    default: float = Field(...)

    @field_validator("bounds")
    @classmethod
    def validate_bounds_ordered(cls, v):
        """Ensure bounds are ordered correctly"""
        if v[0] >= v[1]:
            raise ValueError(f"Lower bound {v[0]} must be less than upper bound {v[1]}")
        return v

    @field_validator("default")
    @classmethod
    def validate_default_in_bounds(cls, v, info):
        """Ensure default is within bounds"""
        if "bounds" in info.data:
            bounds = info.data["bounds"]
            if not (bounds[0] <= v <= bounds[1]):
                raise ValueError(
                    f"Default value {v} must be between {bounds[0]} and {bounds[1]}"
                )
        return v


# Type union for all dimension types
Dimension = Union[CategoricalDimension, IntegerDimension, RealDimension]


class Constraint(BaseModel):
    """Constraint on dimensions (conditional rules)"""

    description: str = Field(..., min_length=1)
    # Constraints stored as arbitrary rules (validated at runtime)
    rules: Dict[str, Any] = Field(default_factory=dict)


class SearchSpaceConfig(BaseModel):
    """Search space configuration for optimization"""

    name: str = Field(..., min_length=1, max_length=255)
    dimensions: List[Dimension] = Field(..., min_items=1, max_items=20)
    constraints: List[Constraint] = Field(default_factory=list)

    @field_validator("dimensions")
    @classmethod
    def validate_unique_dimension_names(cls, v):
        """Ensure dimension names are unique"""
        names = [d.name for d in v]
        if len(names) != len(set(names)):
            raise ValueError("Dimension names must be unique")
        return v

    def get_dimension_by_name(self, name: str) -> Optional[Dimension]:
        """Get dimension by name"""
        for dim in self.dimensions:
            if dim.name == name:
                return dim
        return None

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return {
            "name": self.name,
            "dimensions": [d.model_dump() for d in self.dimensions],
            "constraints": [c.model_dump() for c in self.constraints],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "SearchSpaceConfig":
        """Create from dictionary"""
        return cls(**data)


class FaultPlan(BaseModel):
    """Fault plan to be executed"""

    service: str = Field(..., min_length=1, max_length=255)
    fault_type: FaultType = Field(...)
    duration_ms: int = Field(..., gt=0, le=3600000)  # Max 1 hour
    delay_ms: Optional[int] = Field(default=None, ge=0)
    abort_probability: Optional[float] = Field(default=0.0, ge=0.0, le=1.0)
    error_code: Optional[int] = Field(default=None)

    @field_validator("error_code")
    @classmethod
    def validate_error_code(cls, v):
        """Error code must be 4xx or 5xx"""
        if v is not None and not (400 <= v < 600):
            raise ValueError("Error code must be between 400 and 599")
        return v

    @field_validator("delay_ms")
    @classmethod
    def validate_delay(cls, v, info):
        """Delay must be less than duration"""
        if v is not None and "duration_ms" in info.data:
            if v >= info.data["duration_ms"]:
                raise ValueError(
                    f"Delay {v}ms must be less than duration {info.data['duration_ms']}ms"
                )
        return v

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return self.model_dump(exclude_none=True)

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "FaultPlan":
        """Create from dictionary"""
        return cls(**data)
