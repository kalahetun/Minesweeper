# Data Model: Fix WASM Plugin Delay Fault Bug

**Feature**: 010-fix-wasm-delay-bug  
**Date**: 2025-12-11  
**Status**: Complete

## Entity Changes

### 1. DelayAction (WASM Plugin - Rust)

**Before** (当前):
```rust
#[derive(Debug, Clone, Deserialize)]
pub struct DelayAction {
    #[serde(rename = "fixed_delay")]
    pub fixed_delay: String,           // e.g., "500ms", "1s"
    #[serde(skip)]
    pub parsed_duration_ms: Option<u64>, // 运行时解析结果
}
```

**After** (修改后):
```rust
#[derive(Debug, Clone, Deserialize)]
pub struct DelayAction {
    #[serde(rename = "fixed_delay_ms")]
    pub fixed_delay_ms: u64,           // 毫秒，直接使用
}
```

**Changes**:
- 删除 `fixed_delay: String` 字段
- 添加 `fixed_delay_ms: u64` 字段
- 删除 `parsed_duration_ms` 字段（不再需要运行时解析）

**Validation Rules**:
- `fixed_delay_ms >= 0` (u64 自动满足)
- `fixed_delay_ms <= 30000` (运行时 clamp)
- `fixed_delay_ms == 0` 等同于无延迟故障

---

### 2. DelayAction (Control Plane - Go)

**Before**:
```go
type DelayAction struct {
    FixedDelay string `json:"fixed_delay" yaml:"fixed_delay"`
}
```

**After**:
```go
type DelayAction struct {
    FixedDelayMs uint64 `json:"fixed_delay_ms" yaml:"fixed_delay_ms"`
}
```

---

### 3. DelayAction (CLI - Go)

**Before**:
```go
type DelayAction struct {
    FixedDelay string `yaml:"fixed_delay"`
}
```

**After**:
```go
type DelayAction struct {
    FixedDelayMs uint64 `yaml:"fixed_delay_ms"`
}
```

---

## Configuration Format Changes

### Policy YAML Schema

**Before**:
```yaml
fault:
  delay:
    fixed_delay: "500ms"    # String, 需要解析
```

**After**:
```yaml
fault:
  delay:
    fixed_delay_ms: 500     # Integer (milliseconds)
```

### Supported Values

| 旧格式 | 新格式 |
|--------|--------|
| `"100ms"` | `100` |
| `"500ms"` | `500` |
| `"1s"` | `1000` |
| `"2s"` | `2000` |
| `"30s"` | `30000` |

---

## Deleted Code

### parse_duration Function (config.rs)

将被完全删除的函数:

```rust
/// Parse duration string (e.g., "2s", "100ms") to milliseconds
fn parse_duration(duration_str: &str) -> Option<u64> {
    let duration_str = duration_str.trim().to_lowercase();

    if duration_str.ends_with("ms") {
        if let Ok(ms) = duration_str[..duration_str.len() - 2].parse::<u64>() {
            return Some(ms);
        }
    } else if duration_str.ends_with('s') {
        if let Ok(s) = duration_str[..duration_str.len() - 1].parse::<u64>() {
            return Some(s * 1000);
        }
    } else if duration_str.ends_with('m') {
        if let Ok(m) = duration_str[..duration_str.len() - 1].parse::<u64>() {
            return Some(m * 60 * 1000);
        }
    }

    // Try parsing as plain number (assume milliseconds)
    if let Ok(ms) = duration_str.parse::<u64>() {
        return Some(ms);
    }

    log::warn!("Failed to parse duration: {}", duration_str);
    None
}
```

### test_parse_duration Test

```rust
#[test]
fn test_parse_duration() {
    assert_eq!(parse_duration("100ms"), Some(100));
    assert_eq!(parse_duration("2s"), Some(2000));
    assert_eq!(parse_duration("1m"), Some(60000));
    assert_eq!(parse_duration("500"), Some(500));
    assert_eq!(parse_duration("invalid"), None);
}
```

---

## Constants

### New Constants (WASM Plugin)

```rust
/// Maximum allowed delay in milliseconds (30 seconds)
pub const MAX_DELAY_MS: u64 = 30_000;
```

---

## State Transitions

N/A - 此功能不涉及状态机变更。

---

## Relationships

```
Policy YAML
    │
    ▼
┌─────────────┐
│ Control     │ ──── JSON ────▶ ┌─────────────┐
│ Plane       │                 │ WASM Plugin │
└─────────────┘                 └─────────────┘
    │                                 │
    │ DelayAction {                   │ DelayAction {
    │   fixed_delay_ms: u64           │   fixed_delay_ms: u64
    │ }                               │ }
    ▼                                 ▼
┌─────────────┐               ┌─────────────────┐
│ CLI         │               │ Fault Execution │
│ types.go    │               │ (executor.rs)   │
└─────────────┘               └─────────────────┘
```
