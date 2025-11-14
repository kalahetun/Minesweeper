# Data Model: BOIFI Recommender System

**Date**: 2025-11-14  
**Feature**: 002-boifi-recommender  
**Purpose**: Define all entities, relationships, and validation rules for the recommender system

---

## Entity Relationship Diagram

```
OptimizationSession (1) ──has many─→ (many) Trial
OptimizationSession (1) ──uses─→ (1) SearchSpaceConfig
OptimizationSession (1) ──has─→ (1) BestResult

Trial (1) ──proposes─→ (1) FaultPlan
Trial (1) ──receives─→ (1) RawObservation
Trial (1) ──computed─→ (1) SeverityScore

SearchSpaceConfig (1) ──contains─→ (many) Dimension
SearchSpaceConfig (1) ──enforces─→ (many) Constraint

RawObservation (1) ──contains─→ (many) Span
RawObservation (1) ──contains─→ (many) LogEntry

SeverityScore (1) ──aggregates─→ {BugScore, PerformanceScore, StructureScore}
```

---

## Core Entities

### 1. OptimizationSession

**Purpose**: Represents a single optimization campaign from start to completion.

**Fields**:

| Field | Type | Validation | Notes |
|-------|------|-----------|-------|
| `session_id` | UUID string | Required, unique, generated | e.g., "550e8400-e29b-41d4-a716-446655440000" |
| `status` | Enum | PENDING \| RUNNING \| STOPPING \| COMPLETED \| FAILED | Immutable once set (one-way transitions) |
| `created_at` | ISO8601 timestamp | Required, auto-set at creation | UTC timezone |
| `started_at` | ISO8601 timestamp | Set when status→RUNNING | null until session starts |
| `completed_at` | ISO8601 timestamp | Set when status→COMPLETED/FAILED | null while running |
| `parameters` | object | Required, immutable after creation | Defines search space, trial limits, budget |
| `results` | object | Initially empty, filled as trials complete | Trial history, best fault found |
| `service_name` | string | Required, 1-63 chars, alphanumeric + hyphens | Target service for fault injection |

**State Transitions**:

```
PENDING ──start()──> RUNNING ──complete()──> COMPLETED
                        ↓
                     stop() or timeout
                        ↓
                     STOPPING ──> COMPLETED

PENDING ──error()──> FAILED (any time)
RUNNING ──error()──> FAILED
```

**Serialization Format** (JSON):

```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "RUNNING",
  "created_at": "2025-11-14T10:30:00Z",
  "started_at": "2025-11-14T10:30:30Z",
  "completed_at": null,
  "service_name": "payment-service",
  "parameters": { /* SearchSpaceConfig */ },
  "results": { /* Trial results */ }
}
```

---

### 2. SearchSpaceConfig

**Purpose**: Defines the fault parameter space to explore during optimization.

**Fields**:

| Field | Type | Validation | Notes |
|-------|------|-----------|-------|
| `name` | string | 1-128 chars | e.g., "Payment Service Fault Space" |
| `description` | string | Optional, max 512 chars | Human-readable explanation |
| `dimensions` | array[Dimension] | Required, 1-20 items | Fault parameters to optimize over |
| `constraints` | array[Constraint] | Optional | Conditional rules on dimensions |

**Validation Rules**:
- Dimension names must be unique within config
- At least one dimension required (can't have empty space)
- No circular dependencies in constraints

**Serialization Example** (YAML input format):

```yaml
name: Payment Service Fault Space
description: Explore delay and error injection combinations
dimensions:
  - name: delay_ms
    type: integer
    bounds: [100, 5000]
  - name: error_code
    type: categorical
    values: [500, 502, 503]
  - name: abort_probability
    type: real
    bounds: [0.0, 1.0]
constraints:
  - rule: "if error_code is 500 then abort_probability <= 0.5"
```

---

### 3. Dimension

**Purpose**: Represents one parameter in the search space.

**Fields**:

| Field | Type | Validation | Notes |
|-------|------|-----------|-------|
| `name` | string | 1-64 chars, alphanumeric + underscores | Unique within SearchSpaceConfig |
| `type` | Enum | categorical \| real \| integer | Determines value type |
| `bounds` | array[min, max] | For real/integer only | Inclusive [min, max] |
| `values` | array[any] | For categorical only | List of allowed values |
| `default` | any | Optional | Default value if not specified |

**Examples**:

```json
{
  "name": "delay_ms",
  "type": "integer",
  "bounds": [0, 5000],
  "default": 100
}
```

```json
{
  "name": "error_code",
  "type": "categorical",
  "values": [500, 502, 503, 504],
  "default": 500
}
```

```json
{
  "name": "abort_probability",
  "type": "real",
  "bounds": [0.0, 1.0],
  "default": 0.5
}
```

**Validation Rules**:
- For `real`: bounds[0] < bounds[1], both finite numbers
- For `integer`: bounds[0] < bounds[1], both ≥ 0
- For `categorical`: values array non-empty, all values of same type
- `default` must be within bounds/values if specified

---

### 4. Constraint

**Purpose**: Define conditional rules that must hold across dimensions.

**Format**: Conditional logic `if condition then constraint`

**Examples**:

```
if error_code is 500 then delay_ms >= 1000
if abort_probability > 0.8 then error_code not in [502, 503]
if delay_ms < 100 then error_code must be 200
```

**Implementation**: Stored as predicate functions, evaluated before each fault proposal.

---

### 5. FaultPlan

**Purpose**: A specific fault configuration to execute, derived from Dimension values.

**Fields**:

| Field | Type | Validation | Notes |
|-------|------|-----------|-------|
| `service` | string | Required | Target service name |
| `fault_type` | Enum | delay \| abort \| error_injection | Type of fault |
| `duration_ms` | integer | 1-60000 | How long fault is active |
| `delay_ms` | integer | 0-10000 | Network latency added |
| `error_code` | integer | 100-599 | HTTP error code (for error_injection) |
| `abort_probability` | float | [0.0, 1.0] | Probability to abort (for abort type) |
| `match_conditions` | object | Optional | Headers/paths to match |
| `start_delay_ms` | integer | 0-10000 | Delay before injecting fault |
| `proposal_id` | string | Auto-generated | Links to OptimizerCore proposal |

**Serialization Example**:

```json
{
  "service": "payment-service",
  "fault_type": "delay",
  "duration_ms": 30000,
  "delay_ms": 2500,
  "error_code": null,
  "abort_probability": null,
  "match_conditions": {
    "headers": {"x-user-type": "premium"}
  },
  "start_delay_ms": 200,
  "proposal_id": "prop-12345"
}
```

**Validation Rules**:
- `duration_ms` > 0
- If `fault_type == delay`: delay_ms must be > 0
- If `fault_type == abort`: abort_probability must be > 0
- If `fault_type == error_injection`: error_code must be in 400-599 range
- `start_delay_ms` < `duration_ms`

---

### 6. RawObservation

**Purpose**: Raw data returned from Executor after fault execution.

**Fields**:

| Field | Type | Validation | Notes |
|-------|------|-----------|-------|
| `status_code` | integer | 100-599, optional | HTTP response code |
| `latency_ms` | float | ≥ 0, optional | Response time in milliseconds |
| `error_rate` | float | [0.0, 1.0], optional | Fraction of failed requests |
| `headers` | object | Optional | Response headers (subset) |
| `logs` | array[string] | Optional | Application/system logs |
| `trace_data` | array[Span] | Optional | Distributed trace information |
| `timestamp` | ISO8601 | Required | When observation was collected |

**Serialization Example**:

```json
{
  "status_code": 500,
  "latency_ms": 2450.5,
  "error_rate": 0.15,
  "headers": {
    "content-type": "application/json"
  },
  "logs": [
    "ERROR: database connection timeout",
    "WARN: retrying payment request"
  ],
  "trace_data": [
    {
      "traceID": "abc123",
      "spanID": "def456",
      "operationName": "POST /payments",
      "startTime": 1234567890000,
      "duration": 2450,
      "status": "ERROR",
      "tags": {"http.status_code": 500}
    }
  ],
  "timestamp": "2025-11-14T10:35:45Z"
}
```

**Validation Rules**:
- At least one of {status_code, latency_ms, logs, trace_data} required (not all null)
- If error_rate present: must be [0.0, 1.0]
- latency_ms must be non-negative
- timestamp must be ISO8601 formatted

---

### 7. Span

**Purpose**: Single span from distributed trace (OpenTelemetry format).

**Fields**:

| Field | Type | Notes |
|-------|------|-------|
| `traceID` | UUID string | Identifies the full trace |
| `spanID` | UUID string | Identifies this span |
| `parentSpanID` | UUID string | Links to parent span (null if root) |
| `operationName` | string | e.g., "POST /api/payments" |
| `startTime` | milliseconds | Unix timestamp |
| `duration` | milliseconds | Duration of this span |
| `status` | "OK" \| "ERROR" | Span outcome |
| `tags` | object | Metadata (e.g., {"http.status_code": 200}) |
| `logs` | array[{timestamp, message}] | Span events |

---

### 8. SeverityScore

**Purpose**: Quantified impact of a fault injection, computed from RawObservation.

**Fields**:

| Field | Type | Validation | Notes |
|-------|------|-----------|-------|
| `trial_id` | integer | Required | References which trial |
| `total_score` | float | [0.0, 10.0] | Aggregated severity |
| `bug_score` | float | [0.0, 10.0] | HTTP/error dimension |
| `performance_score` | float | [0.0, 10.0] | Latency dimension |
| `structure_score` | float | [0.0, 10.0] | Trace dimension |
| `components` | object | | Breakdown of scoring |
| `timestamp` | ISO8601 | | When score was computed |

**Serialization Example**:

```json
{
  "trial_id": 15,
  "total_score": 7.3,
  "bug_score": 10.0,
  "performance_score": 10.0,
  "structure_score": 2.0,
  "components": {
    "bug": {
      "matched_condition": "HTTP 5xx",
      "value": 500,
      "score": 10.0
    },
    "performance": {
      "baseline_ms": 200,
      "threshold_ms": 1000,
      "current_ms": 2000,
      "score": 10.0
    },
    "structure": {
      "error_span_count": 1,
      "score": 2.0
    }
  },
  "timestamp": "2025-11-14T10:35:50Z"
}
```

**Calculation**:
```
total_score = (bug_score + performance_score + structure_score) / 3
```

---

### 9. Trial

**Purpose**: Single optimization iteration with its fault, execution, and result.

**Fields**:

| Field | Type | Notes |
|-------|------|-------|
| `trial_id` | integer | Sequential ID within session |
| `fault_plan` | FaultPlan | Proposed fault configuration |
| `raw_observation` | RawObservation | Executor's response |
| `severity_score` | SeverityScore | Analyzer's computation |
| `timestamp` | ISO8601 | When trial was executed |
| `duration_sec` | float | Execution time |
| `status` | "SUCCESS" \| "FAILED" | Trial outcome |

---

### 10. BestResult

**Purpose**: Tracks the best fault found so far in optimization.

**Fields**:

| Field | Type | Notes |
|-------|------|-------|
| `fault_plan` | FaultPlan | Best fault configuration |
| `severity_score` | float | Best score achieved |
| `trial_id` | integer | Which trial found this |
| `timestamp` | ISO8601 | When found |

---

## API Request/Response Models

### Create Session Request

```json
{
  "service_name": "payment-service",
  "search_space": {
    "name": "Payment Fault Space",
    "dimensions": [
      {"name": "delay_ms", "type": "integer", "bounds": [100, 5000]},
      {"name": "error_code", "type": "categorical", "values": [500, 502, 503]}
    ]
  },
  "max_trials": 50,
  "time_budget_sec": 3600
}
```

**Validation**:
- service_name: 1-63 chars, alphanumeric + hyphens
- max_trials: 1-1000
- time_budget_sec: 60-86400 (1 min - 24 hrs)

---

### Session Response

```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "RUNNING",
  "created_at": "2025-11-14T10:30:00Z",
  "service_name": "payment-service",
  "progress": {
    "trials_completed": 15,
    "max_trials": 50,
    "progress_percent": 30
  },
  "best_result": {
    "fault_plan": {...},
    "severity_score": 7.8,
    "trial_id": 12
  },
  "estimated_remaining_sec": 1200
}
```

---

### Session Status Response

```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "RUNNING",
  "trials_completed": 15,
  "max_trials": 50,
  "best_score": 7.8,
  "best_fault": {
    "service": "payment-service",
    "fault_type": "delay",
    "duration_ms": 30000,
    "delay_ms": 2500
  },
  "worst_score": 0.5,
  "average_score": 4.2,
  "created_at": "2025-11-14T10:30:00Z",
  "started_at": "2025-11-14T10:30:30Z",
  "estimated_completion_time": "2025-11-14T11:50:00Z"
}
```

---

### Error Response

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Search space must contain at least one dimension",
    "details": {
      "field": "search_space.dimensions",
      "constraint": "non-empty array"
    }
  }
}
```

---

## Validation Rules Summary

| Entity | Critical Rules | Constraints |
|--------|----------------|-------------|
| **OptimizationSession** | Immutable after creation, one-way status transitions | Max 10 concurrent sessions |
| **SearchSpaceConfig** | 1-20 dimensions, unique dimension names | No circular constraint dependencies |
| **Dimension** | Type-specific bounds/values, default in range | Bounds must be ordered (min < max) |
| **FaultPlan** | Fault type determines required fields, duration > 0 | duration_ms < start_delay_ms forbidden |
| **RawObservation** | At least one data field required (not all null) | error_rate ∈ [0, 1], latency_ms ≥ 0 |
| **SeverityScore** | Aggregation formula: (bug + perf + struct) / 3 | All component scores ∈ [0, 10] |
| **Trial** | Immutable once recorded | Status consistency with score presence |

---

## Type Definitions (Pydantic/TypeScript)

### Python Example

```python
from dataclasses import dataclass
from enum import Enum
from typing import Optional, List, Dict, Any
from datetime import datetime
from uuid import UUID

class SessionStatus(str, Enum):
    PENDING = "PENDING"
    RUNNING = "RUNNING"
    STOPPING = "STOPPING"
    COMPLETED = "COMPLETED"
    FAILED = "FAILED"

@dataclass
class Dimension:
    name: str
    type: str  # "categorical", "real", "integer"
    bounds: Optional[List[float]] = None
    values: Optional[List[Any]] = None
    default: Optional[Any] = None

@dataclass
class SearchSpaceConfig:
    name: str
    dimensions: List[Dimension]
    description: Optional[str] = None
    constraints: Optional[List[str]] = None

@dataclass
class FaultPlan:
    service: str
    fault_type: str  # "delay", "abort", "error_injection"
    duration_ms: int
    delay_ms: Optional[int] = None
    error_code: Optional[int] = None
    abort_probability: Optional[float] = None
    match_conditions: Optional[Dict[str, Any]] = None
    start_delay_ms: int = 0
    proposal_id: Optional[str] = None

@dataclass
class OptimizationSession:
    session_id: UUID
    status: SessionStatus
    service_name: str
    created_at: datetime
    parameters: SearchSpaceConfig
    started_at: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    results: Optional[Dict[str, Any]] = None
```

---

## Database Schema (Future, Phase 2+)

If migrating to PostgreSQL:

```sql
CREATE TABLE optimization_sessions (
    session_id UUID PRIMARY KEY,
    status VARCHAR(20) NOT NULL,
    service_name VARCHAR(63) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    search_space JSONB NOT NULL,
    results JSONB,
    created_by VARCHAR(128)
);

CREATE TABLE trials (
    trial_id INTEGER NOT NULL,
    session_id UUID NOT NULL REFERENCES optimization_sessions,
    fault_plan JSONB NOT NULL,
    raw_observation JSONB,
    severity_score JSONB,
    timestamp TIMESTAMP NOT NULL,
    status VARCHAR(20) NOT NULL,
    PRIMARY KEY (session_id, trial_id)
);

CREATE INDEX idx_sessions_service ON optimization_sessions(service_name);
CREATE INDEX idx_sessions_status ON optimization_sessions(status);
CREATE INDEX idx_trials_session ON trials(session_id);
```

---

## Validation Implementation

All data models implement validation at construction time:

1. **Pydantic Models** (Python):
   - Automatic type checking
   - Custom validators for cross-field rules
   - Clear error messages on validation failure

2. **API Endpoint Validation**:
   - Request body validation (Pydantic)
   - Response serialization (Pydantic)
   - HTTP error codes for validation failures (400 Bad Request)

3. **Business Logic Validation**:
   - Constraint evaluation (SearchSpaceConfig)
   - Fault plan legality checks (dimension values within bounds)
   - Session state transition rules

---

## Migration Path

Phase 1 → Phase 2:
- Keep JSON serialization format unchanged
- Add database layer beneath SessionManager
- Maintain same API contracts
- Enable distributed session management
