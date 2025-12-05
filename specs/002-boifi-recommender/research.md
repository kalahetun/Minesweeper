# Research Findings: BOIFI Recommender Technical Analysis

**Date**: 2025-11-14  
**Feature**: 002-boifi-recommender (Bayesian Optimizer)  
**Purpose**: Resolve technical unknowns and establish best practices for implementation

---

## 1. Executor API Compatibility

### Problem Statement
The specification assumes fault plan submission and observation retrieval from Executor, but the existing Executor project provides a Policy-based API (POST /v1/policies, DELETE /v1/policies). We need to clarify the integration pattern.

### Decision: Dual-Mode ExecutorClient

**Chosen Approach**: Implement `ExecutorClient` with two supported modes based on Executor availability:

#### Mode A: Policy-Based API (Current HFI Executor)
*Used when Executor only supports policy CRUD operations*

```
1. Recommender proposes FaultPlan (service, delay_ms, duration_sec, etc.)
2. ExecutorClient converts FaultPlan → FaultInjectionPolicy
3. POST /v1/policies to create the policy with unique ID
4. Recommender waits for fault_duration_seconds
5. GET /v1/policies/{id} or monitor logs to collect observations
6. DELETE /v1/policies/{id} to remove the fault
7. Parse observations from monitoring system or logs
```

**Challenges**:
- No explicit observation endpoint; must wait for policy duration to complete
- Observation collection requires external monitoring integration (logs, traces)
- Timing control less precise (policy duration-based vs request-level start_delay)

**Mitigation**:
- Wait deterministically (duration_ms + buffer) before attempting to collect observations
- Log all important events (policy create/delete) for debugging
- Document assumption that Executor logs contain trace/latency data

---

#### Mode B: Extended Executor API (Future Enhancement)
*If Executor adds dedicated fault execution endpoints*

```
1. Recommender proposes FaultPlan
2. POST /v1/faults/apply with FaultPlan → returns apply_id
3. GET /v1/faults/{apply_id}/status to poll execution status
4. GET /v1/faults/{apply_id}/metrics to retrieve RawObservation
```

**Advantages**:
- Precise execution control and observation retrieval
- Decoupled from policy lifecycle
- Clear synchronous execution model

**Implementation**: Make ExecutorClient abstract with two implementations:
- `PolicyBasedExecutorClient` (Mode A, works with current HFI)
- `ExtendedExecutorClient` (Mode B, future-proof)

**Recommendation**: Implement Mode A for immediate compatibility. Mode B can be added when Executor API expands.

---

### Rationale

| Aspect | Policy-Based (Mode A) | Extended (Mode B) | Decision |
|--------|----------------------|-------------------|----------|
| Current HFI Compatibility | ✅ Yes | ❌ No | Mode A for Phase 1 |
| Observation Clarity | ⚠️ Requires integration | ✅ Explicit endpoint | Mode B for Phase 2+ |
| Timing Control | ⚠️ Duration-based | ✅ Request-level | Mode A limitation accepted |
| Implementation Effort | 1-2 days | 2-3 days | Start with Mode A |

---

## 2. scikit-optimize Configuration

### Problem Statement
scikit-optimize provides Bayesian optimization, but requires tuning for fault space dimensionality (up to 20 dimensions per spec), and selection of surrogate model + acquisition function.

### Decision: Random Forest + Expected Improvement

**Surrogate Model Selection**:

| Model | Pros | Cons | Choice |
|-------|------|------|--------|
| **Gaussian Process (GP)** | Smooth approximations, uncertainty quantification | Scales poorly >10 dims, expensive covariance matrix | ❌ |
| **Random Forest** | Handles discrete + continuous, fast training, handles high dims | Less uncertainty estimate, discrete outputs | ✅ **Chosen** |
| **Gradient Boosting (GBDT)** | Very fast, handles mixed types | Requires careful tuning, less interpretable | ⚠️ Considered |

**Chosen**: Random Forest (via scikit-optimize `forest_minimize`)
- Efficient for up to 20 dimensions
- No hyperparameter tuning needed (sensible defaults)
- Handles both categorical and continuous parameters natively
- Model retraining <200ms target achievable

---

**Acquisition Function Selection**:

| Function | Behavior | Use Case | Choice |
|----------|----------|----------|--------|
| **Expected Improvement (EI)** | Balances exploration vs exploitation | General purpose, good default | ✅ **Chosen** |
| **Upper Confidence Bound (UCB)** | Optimistic exploration | Faster convergence, more exploration | ⚠️ Alternative |
| **Probability of Improvement (PI)** | Conservative, greedy | Exploitation-heavy | ❌ |

**Chosen**: Expected Improvement (EI)
- Good balance between exploring new regions and exploiting promising areas
- Standard in Bayesian optimization community
- Works well with Random Forest surrogate
- Theory: EI(x) = E[max(f(x) - f_best, 0)]

---

**Configuration Parameters**:

```python
# From scikit-optimize.forest_minimize documentation
optimizer_config = {
    'base_estimator': 'RF',          # Random Forest
    'acq_func': 'EI',                # Expected Improvement
    'acq_optimizer': 'auto',         # Automatic selection (Powell)
    'initial_point_generator': 'lhs', # Latin Hypercube Sampling for initial points
    'n_initial_points': 5,           # Start with 5 random trials before fitting model
    'n_random_starts': 3,            # Additional random trials during optimization
    'random_state': 42,              # Reproducibility (optional, can be None for true randomness)
}

# For 20-dimensional space, expected:
# - Model training time: ~150-200ms per trial
# - Proposal time: ~20-50ms
# - Achieved performance target: <600ms per iteration ✅
```

---

### Rationale

**Why scikit-optimize over alternatives?**

| Framework | Language | Ease of Use | Setup Complexity | Choice |
|-----------|----------|------------|------------------|--------|
| **scikit-optimize** | Python | Very easy, dict-based API | Minimal | ✅ **Chosen** |
| **BoTorch** (PyTorch) | Python | More complex, GPU-heavy | Significant | ❌ Overkill for MVP |
| **Ray Tune** | Python | Powerful but heavyweight | Integration heavy | ❌ Overkill |
| **Hyperopt** | Python | Good, but older | Moderate | ⚠️ Alternative, but less active |

**Chosen**: scikit-optimize
- Minimal dependencies (scikit-learn, numpy)
- Perfect for 20-dimensional fault space
- No GPU needed (runs on CPU efficiently)
- Active maintenance, good documentation
- Aligns with constitution principle VII (Simplicity & Minimalism)

---

## 3. Distributed Trace Analysis

### Problem Statement
Response Analyzer must detect structural anomalies (retries, cascading failures, circuit breaker activation) from distributed traces. We need to clarify:
- Trace format (OpenTelemetry, Jaeger, custom?)
- Latency extraction methodology
- Span comparison algorithm (edit distance, clustering?)

### Decision: OpenTelemetry (OTEL) Span Format

**Assumption**: Executor returns traces in standard OpenTelemetry format (Jaeger JSON export).

**Standard OTEL Span Format**:
```json
{
  "traceID": "uuid",
  "spans": [
    {
      "spanID": "uuid",
      "parentSpanID": "uuid",
      "operationName": "GET /api/users",
      "startTime": 1234567890123,
      "duration": 45,
      "status": { "code": "OK" | "ERROR" },
      "tags": {
        "http.status_code": 200,
        "span.kind": "server",
        "component": "http"
      },
      "logs": [
        {"timestamp": 1234567890130, "message": "request complete"}
      ]
    }
  ]
}
```

---

**Latency Extraction**:

1. **Span Duration**: Direct measurement from span start/end timestamps
2. **Critical Path Latency**: Sum of spans on critical path (depth-first, longest branch)
3. **Aggregation**:
   - Baseline latency = sum of all span durations in healthy trace
   - Current latency = sum of all span durations in fault-injected trace
   - Degradation = (current - baseline) / baseline

**Formula** (used in PerformanceScorer):
```
latency_ms = critical_path_duration_ms
degradation_ratio = (latency_ms - baseline_ms) / (threshold_ms - baseline_ms)
score_perf = min(10.0, 9.0 * degradation_ratio)
```

---

**Structural Anomaly Detection**:

Using edit distance (Levenshtein) on span operation sequences:

```python
def detect_structural_change(current_trace, baseline_trace):
    """
    Compare span sequences to detect anomalies
    """
    current_ops = [span['operationName'] for span in current_trace['spans']]
    baseline_ops = [span['operationName'] for span in baseline_trace['spans']]
    
    edit_distance = levenshtein_distance(current_ops, baseline_ops)
    
    # Anomaly thresholds
    if edit_distance > 2:
        return "MAJOR_CHANGE"  # Score 5.0
    elif len(current_ops) > len(baseline_ops) * 1.5:
        return "RETRY_OR_CASCADE"  # Score 3.0
    elif any(span['status']['code'] == 'ERROR' for span in current_trace['spans']):
        return "ERROR_SPAN"  # Score 2.0
    else:
        return "NO_CHANGE"  # Score 0.0
```

---

### Trace Format Compatibility

| Format | Status | How to Handle |
|--------|--------|---------------|
| **OpenTelemetry JSON** | Standard, recommended | Parse directly |
| **Jaeger JSON Export** | Compatible with OTEL | Same parser |
| **Custom JSON** | Possible if documented | Map fields to OTEL schema |
| **gRPC Proto** | Possible but complex | Not planned for Phase 1 |

**Recommendation**: Design StructureScorer to accept OTEL-compatible JSON. Document assumption that Executor exports traces in this format. If not, provide transformation layer.

---

## 4. Response Analyzer Scoring Formulas

### Problem Statement
Spec defines three scoring dimensions (Bug, Performance, Structure) but lacks precise mathematical formulas. Need to establish reproducible, testable scoring logic.

### Decision: Weighted Formulas with Fail-Safe Defaults

---

#### Dimension 1: Bug Scorer [0-10]

**Scoring Rules** (prioritized order - first match wins):

| Condition | Score | Rationale |
|-----------|-------|-----------|
| HTTP 5xx (500-599) | 10.0 | Server error, critical failure |
| HTTP 4xx (400-499) | 8.0 | Client error, but might indicate app logic issue |
| ERROR in logs (case-insensitive) | 6.0 | Application detected an error |
| error_rate > 0 | 3.0 | Some requests failed, but not all |
| All else | 0.0 | No bug symptoms detected |

**Default if data missing**: 0.0

---

#### Dimension 2: Performance Scorer [0-10]

**Parameters**:
- `baseline_ms` = normal service latency (e.g., 200ms) - from config
- `threshold_ms` = maximum acceptable latency (e.g., 1000ms) - from config
- `current_ms` = observed latency from this fault trial

**Formula**:
```
if current_ms > threshold_ms:
    score = 10.0  (exceeds acceptable limit)
elif current_ms >= baseline_ms:
    ratio = (current_ms - baseline_ms) / (threshold_ms - baseline_ms)
    score = 9.0 * ratio  (linear interpolation 0-9 in valid range)
else:
    score = 0.0  (faster than baseline, impossible but defensive)

score = min(10.0, max(0.0, score))  (clamp to [0, 10])
```

**Default if latency missing**: 0.0

**Interpretation**:
- Score 0: baseline latency (no degradation)
- Score 4.5: 50% of degradation budget consumed
- Score 9: at threshold
- Score 10: exceeds threshold

---

#### Dimension 3: Structure Scorer [0-10]

**Span Count Check**:
```
span_increase = (current_span_count - baseline_span_count) / baseline_span_count
if span_increase > 0.5:  (50% increase = possible retries/cascades)
    score = 3.0
```

**Edit Distance Check**:
```
edit_dist = levenshtein(current_ops, baseline_ops)
if edit_dist > 2:  (significant sequence change)
    score = 5.0
```

**Error Span Check**:
```
error_spans = count(span where status == 'ERROR')
if error_spans > 0:
    score = 2.0
```

**Aggregation**:
```
structure_score = max(all triggered sub-scores)
```

**Default if trace missing**: 0.0

---

#### Final Severity Score Aggregation

**Weighted Average**:
```
config.weights = {
    'bug': 1.0,           # Equally weighted
    'performance': 1.0,
    'structure': 1.0,
}

severity_score = (
    config.weights['bug'] * bug_score +
    config.weights['performance'] * perf_score +
    config.weights['structure'] * struct_score
) / sum(config.weights.values())

# Result: [0.0, 10.0]
```

**Example Calculations**:

*Trial 1*: HTTP 200, latency 600ms, no trace issues
- Bug: 0.0
- Perf: 9.0 * (600-200)/(1000-200) = 4.5
- Struct: 0.0
- **Total**: (0 + 4.5 + 0) / 3 = **1.5** (mild impact)

*Trial 2*: HTTP 500, latency 2000ms, error spans present
- Bug: 10.0
- Perf: 10.0 (exceeds threshold)
- Struct: 2.0 (error spans)
- **Total**: (10 + 10 + 2) / 3 = **7.3** (severe impact)

---

### Rationale for Fail-Safe Defaults

**Constitution Principle VI (Fault Tolerance)** requires fail-safe scoring when data is incomplete. If Executor doesn't return latency or traces:

- Missing `latency_ms` → assume 0.0 for performance (conservative)
- Missing `trace_data` → assume 0.0 for structure (conservative)
- Missing `logs` → assume 0.0 for bug score (conservative)

This ensures that incomplete observations don't crash the optimization loop; they simply contribute less signal to the model.

---

## 5. Executor Client Resilience

### Problem Statement
Network failures, timeouts, and service degradation must be handled gracefully. Need to specify retry policy, circuit breaker, and health check strategy.

### Decision: Exponential Backoff + Circuit Breaker

---

**Retry Strategy**: Exponential Backoff with Jitter

```python
max_retries = 5
base_delay_sec = 0.5
max_delay_sec = 10.0

for attempt in range(1, max_retries + 1):
    try:
        result = executor_client.execute(fault_plan)
        return result
    except (TimeoutError, ConnectionError) as e:
        if attempt >= max_retries:
            raise ExecutionFailed(f"Max retries exceeded: {e}")
        
        # Exponential backoff with jitter
        delay = min(
            base_delay_sec * (2 ** (attempt - 1)),  # 0.5, 1.0, 2.0, 4.0, 8.0
            max_delay_sec
        )
        jitter = random.uniform(0, 0.1 * delay)  # 10% jitter
        wait_time = delay + jitter
        
        logger.warning(f"Attempt {attempt} failed, retrying in {wait_time:.2f}s")
        time.sleep(wait_time)
```

**Sequence**:
- Attempt 1 fails: wait 0.5-0.55s, retry
- Attempt 2 fails: wait 1.0-1.10s, retry
- Attempt 3 fails: wait 2.0-2.20s, retry
- Attempt 4 fails: wait 4.0-4.40s, retry
- Attempt 5 fails: wait 8.0-8.80s, retry
- Attempt 6 fails: raise ExecutionFailed

**Why exponential backoff?**
- Avoids thundering herd (not all retries at same time)
- Gives Executor service time to recover
- Matches industry best practices (AWS, GCP, Azure)

---

**Circuit Breaker Pattern**

```python
class CircuitBreaker:
    def __init__(self, failure_threshold=5, recovery_timeout_sec=60):
        self.state = "CLOSED"          # Normal operation
        self.failure_count = 0
        self.failure_threshold = failure_threshold
        self.recovery_timeout_sec = recovery_timeout_sec
        self.last_failure_time = None
    
    def can_attempt(self):
        """Check if request is allowed"""
        if self.state == "CLOSED":
            return True  # All requests allowed
        
        if self.state == "OPEN":
            # Try to transition to HALF_OPEN after timeout
            if time.time() - self.last_failure_time > self.recovery_timeout_sec:
                self.state = "HALF_OPEN"
                self.failure_count = 0
                return True  # Allow one test request
            else:
                return False  # Fast fail, don't retry
        
        if self.state == "HALF_OPEN":
            return True  # Allow test request
    
    def record_success(self):
        """Call after successful request"""
        if self.state == "HALF_OPEN":
            self.state = "CLOSED"
            self.failure_count = 0
        elif self.state == "CLOSED":
            self.failure_count = 0
    
    def record_failure(self):
        """Call after failed request"""
        self.failure_count += 1
        self.last_failure_time = time.time()
        
        if self.failure_count >= self.failure_threshold:
            self.state = "OPEN"
            logger.error(f"Circuit breaker OPEN after {self.failure_count} failures")
```

**State Transitions**:
```
CLOSED ──(5 failures)──> OPEN ──(timeout)──> HALF_OPEN ──(success)──> CLOSED
                                      ↑
                                   (failure)
```

**Benefits**:
- Fails fast when Executor is down (no wasted retries)
- Periodically tests recovery (every 60s)
- Protects both Executor and Recommender from cascading failures

---

**Health Check**

```python
def health_check(self):
    """Verify Executor is accessible"""
    try:
        response = httpx.get(
            f"{self.executor_host}/v1/health",
            timeout=5.0
        )
        return response.status_code == 200
    except Exception:
        return False

# Called at startup and periodically (every 60s)
if not self.health_check():
    logger.error("Executor health check failed")
    self.circuit_breaker.record_failure()
```

---

### Rationale

| Mechanism | Why Needed | How It Works |
|-----------|-----------|-------------|
| **Exponential Backoff** | Prevents retry storms, gives service recovery time | Double wait time each attempt (0.5s → 1s → 2s → ...) |
| **Jitter** | Prevents thundering herd (all clients retry simultaneously) | Add random 0-10% variation to delay |
| **Circuit Breaker** | Fails fast, protects system from cascading failures | Open after N failures, test recovery periodically |
| **Health Check** | Early detection of Executor unavailability | Lightweight ping to /health endpoint |

---

## 6. Session Persistence & Recovery

### Problem Statement
Specification requires result persistence (FR-009, SC-008), but doesn't define storage format or recovery behavior. Need to clarify session data structure and persistence mechanism.

### Decision: JSON File + In-Memory Cache

**Storage Strategy**:

```
recommender/
└── .sessions/           # Local session data (optional, development only)
    ├── session-uuid1.json
    ├── session-uuid2.json
    └── ...
```

**Session File Format** (JSON):

```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "COMPLETED",
  "created_at": "2025-11-14T10:30:00Z",
  "completed_at": "2025-11-14T11:15:30Z",
  "parameters": {
    "service": "payment-service",
    "max_trials": 50,
    "search_space": {
      "dimensions": [
        {
          "name": "delay_ms",
          "type": "integer",
          "bounds": [100, 5000]
        },
        {
          "name": "error_code",
          "type": "categorical",
          "values": [500, 502, 503]
        }
      ]
    }
  },
  "results": {
    "trials_completed": 50,
    "best_fault": {
      "delay_ms": 2500,
      "error_code": 502
    },
    "best_score": 8.7,
    "trial_history": [
      {
        "trial_id": 1,
        "fault_plan": {...},
        "severity_score": 3.2,
        "timestamp": "2025-11-14T10:31:00Z"
      },
      ...
    ]
  }
}
```

---

**Persistence Behavior**:

1. **Session Created**: Write session-uuid.json with initial state (PENDING)
2. **Trial Completed**: Update JSON with trial results in real-time
3. **Session Stopped**: Write final status (COMPLETED/FAILED/STOPPED)
4. **Recovery on Restart**: Load session-uuid.json, resume from last completed trial

---

**Implementation**:

```python
class SessionPersistence:
    def __init__(self, sessions_dir: Path = Path(".sessions")):
        self.sessions_dir = sessions_dir
        self.sessions_dir.mkdir(exist_ok=True)
    
    def save_session(self, session: OptimizationSession):
        """Persist session to JSON file"""
        session_file = self.sessions_dir / f"{session.session_id}.json"
        with open(session_file, 'w') as f:
            json.dump(session.to_dict(), f, indent=2)
    
    def load_session(self, session_id: str) -> OptimizationSession:
        """Load session from JSON file"""
        session_file = self.sessions_dir / f"{session_id}.json"
        if not session_file.exists():
            raise SessionNotFound(f"Session {session_id} not found")
        
        with open(session_file, 'r') as f:
            data = json.load(f)
        return OptimizationSession.from_dict(data)
    
    def list_sessions(self) -> List[str]:
        """List all session IDs"""
        return [f.stem for f in self.sessions_dir.glob("*.json")]
```

---

**Trade-offs**:

| Aspect | JSON File | In-Memory Only | Database |
|--------|-----------|----------------|----------|
| **Persistence** | ✅ Survives restart | ❌ Lost on restart | ✅ Survives restart |
| **Complexity** | ✅ Simple | ✅ Simple | ❌ Requires infrastructure |
| **Scalability** | ⚠️ ~1000 sessions | ⚠️ ~100 sessions | ✅ Unlimited |
| **Phase 1** | ✅ Perfect | ⚠️ Acceptable | ❌ Over-engineered |

**Chosen**: JSON File + In-Memory Cache
- Simple to implement (pure Python, no external DB)
- Works for Phase 1 scale (10 concurrent sessions)
- Easy to migrate to database later
- Aligns with Constitution VII (Simplicity)

---

## Summary of Decisions

| Topic | Decision | Rationale |
|-------|----------|-----------|
| **Executor Integration** | Policy-Based API (Mode A) + future Extended API support | Immediate compatibility, future-proof design |
| **Bayesian Optimizer** | scikit-optimize + Random Forest + EI | Efficient, simple, proven in similar domains |
| **Trace Analysis** | OpenTelemetry JSON format, edit distance for anomaly detection | Standard format, clear methodology |
| **Scoring Formulas** | Weighted three-dimension aggregation with fail-safe defaults | Constitutional fault tolerance, reproducible |
| **Executor Resilience** | Exponential backoff + circuit breaker + periodic health check | Industry best practice, protects system stability |
| **Session Persistence** | JSON files + in-memory cache | Simple Phase 1 solution, easy migration path |

---

## Next Steps

1. **Phase 1 Design** (generate data-model.md, contracts/)
   - Formalize entity definitions using these decisions
   - Create OpenAPI spec for session management API
   - Define data validation rules in Pydantic models

2. **Phase 2 Tasks** (via /speckit.tasks)
   - Break down each service (Coordinator, Optimizer, Analyzer, Client) into testable modules
   - Assign effort estimates based on complexity
   - Identify parallelizable tasks for efficient team allocation
