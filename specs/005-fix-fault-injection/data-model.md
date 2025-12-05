# Data Model: Fix Fault Injection

## Entities

### FaultInjectionPolicy (Go & Rust)

The core configuration object.

```go
type FaultInjectionPolicy struct {
    Metadata PolicyMetadata `json:"metadata"`
    Spec     PolicySpec     `json:"spec"`
}

type PolicyMetadata struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace"` // Optional, for future multi-tenancy
    ID        string `json:"id"`        // Unique ID for metrics
}

type PolicySpec struct {
    Rules []PolicyRule `json:"rules"`
}

type PolicyRule struct {
    Match MatchCondition `json:"match"`
    Fault FaultAction    `json:"fault"`
}

type MatchCondition struct {
    Method  *StringMatch      `json:"method,omitempty"`
    Path    *StringMatch      `json:"path,omitempty"`
    Headers []HeaderMatch     `json:"headers,omitempty"`
}

type StringMatch struct {
    Exact  string `json:"exact,omitempty"`
    Prefix string `json:"prefix,omitempty"`
    Regex  string `json:"regex,omitempty"`
}

type HeaderMatch struct {
    Name  string      `json:"name"`
    Value StringMatch `json:"value"`
}

type FaultAction struct {
    Percentage      float32      `json:"percentage"` // 0.0 - 100.0
    DurationSeconds int64        `json:"duration_seconds,omitempty"` // 0 = forever
    StartDelayMs    int64        `json:"start_delay_ms,omitempty"`   // Delay before injection
    Abort           *AbortFault  `json:"abort,omitempty"`
    Delay           *DelayFault  `json:"delay,omitempty"`
}

type AbortFault struct {
    HttpStatus int `json:"http_status"`
}

type DelayFault struct {
    FixedDelayMs int64 `json:"fixed_delay_ms"`
}
```

### WasmConfig (Rust)

The configuration passed to the Wasm plugin.

```rust
#[derive(Deserialize, Clone, Debug)]
pub struct WasmConfig {
    pub policies: Vec<FaultInjectionPolicy>,
    pub service_name: Option<String>, // Local service identity
}
```

## Metrics

- `boifi_fault_injected_total`: Counter
  - Labels: `policy_id`, `service`, `fault_type` (abort/delay), `result` (success/skipped)

## State Transitions

1. **Policy Created**: Stored in Control Plane.
2. **Policy Pushed**: Serialized to JSON, sent to Envoy via xDS or SSE (simulated via file/API for now).
3. **Policy Active**: Wasm plugin parses config.
4. **Request Arrives**:
   - Match? -> No -> Pass.
   - Match? -> Yes -> Check Percentage.
   - Percentage? -> Pass -> Pass.
   - Percentage? -> Hit -> Check Expiration.
   - Expired? -> Pass.
   - Active? -> Apply Fault.
     - Abort: Send HTTP response immediately.
     - Delay: Pause request, wait `fixed_delay_ms`, resume.
     - Start Delay: Pause request, wait `start_delay_ms`, then Apply Fault (Abort or Delay).
