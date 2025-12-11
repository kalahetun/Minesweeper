# Feature Specification: Fix WASM Plugin Delay Fault Bug

**Feature Branch**: `010-fix-wasm-delay-bug`  
**Created**: 2025-12-11  
**Status**: Draft  
**Input**: User description: "修复 WASM 插件 delay 故障 dispatch_http_call BadArgument 错误，并将 fixed_delay 类型从 String 改为 u64 (毫秒) 以简化代码"

## Problem Analysis

### Root Cause

WASM 插件的 delay 故障注入功能当前存在一个严重 Bug，导致所有 delay 故障都会失败并返回 `dispatch_http_call: BadArgument` 错误。

**根本原因分析**：

1. **虚假集群调用**：在 `lib.rs` 中，delay 实现试图使用 `dispatch_http_call` 调用一个名为 `"hfi_delay_cluster"` 的集群，利用超时机制模拟延迟。

2. **集群不存在**：`"hfi_delay_cluster"` 未在 Envoy 配置中定义，因此 `dispatch_http_call` 返回 `Status::BadArgument`。

3. **设计缺陷**：开发者试图通过 HTTP 调用的超时机制来实现延迟，但这需要一个有效的 Envoy 集群。proxy-wasm SDK 没有原生的 "sleep" API。

### Secondary Issue

`fixed_delay` 字段使用 `String` 类型（如 `"500ms"`、`"1s"`），需要运行时解析，增加了代码复杂性和潜在的解析错误。

## User Scenarios & Testing

### User Story 1 - Delay Fault Injection Works (Priority: P1)

作为一个混沌工程师，我希望配置的 delay 故障能够正确生效，使目标服务的响应时间增加指定的毫秒数。

**Why this priority**: Delay 故障是 BOIFI 核心功能之一，当前完全无法工作，阻塞了所有依赖延迟测试的用户场景。

**Independent Test**: 可通过 `validate-basic.sh` 脚本独立测试：应用 delay 策略后，请求响应时间应增加配置的延迟量。

**Acceptance Scenarios**:

1. **Given** 一个配置了 500ms delay 的策略已应用，**When** 用户发送 HTTP 请求到目标服务，**Then** 响应时间应增加约 500ms（±50ms 误差范围内）。

2. **Given** 一个 delay 策略已配置，**When** WasmPlugin 接收到匹配的请求，**Then** Envoy 日志中应显示 "Delay fault triggered" 而不是 "Failed to dispatch delay call"。

3. **Given** 一个 100% 概率的 delay 策略，**When** 发送 10 个请求，**Then** 所有 10 个请求的响应时间都应增加配置的延迟量。

---

### User Story 2 - Simplified Configuration Format (Priority: P2)

作为一个策略配置者，我希望使用简单的整数（毫秒）来配置延迟时间，而不是需要解析的字符串格式。

**Why this priority**: 简化配置格式减少用户错误，降低代码复杂性，但不如 Bug 修复紧急。

**Independent Test**: 可通过修改配置文件并应用策略来测试：使用 `fixed_delay_ms: 500` 而非 `fixed_delay: "500ms"`。

**Acceptance Scenarios**:

1. **Given** 一个使用新格式 `fixed_delay_ms: 500` 的策略，**When** 用户通过 CLI 应用策略，**Then** 策略应被成功接受和应用。

2. **Given** 所有示例配置文件已更新为新格式，**When** 用户运行 `validate-basic.sh`，**Then** 所有测试应通过。

3. **Given** 一个使用旧格式 `fixed_delay: "500ms"` 的策略，**When** 用户尝试应用，**Then** 系统应明确拒绝并提示使用新格式。

---

### User Story 3 - Metrics Correctly Recorded (Priority: P2)

作为一个运维人员，我希望 delay 故障的指标能够正确记录，以便通过 Prometheus 监控延迟注入效果。

**Why this priority**: 与 US2 同等重要，都是 delay 功能完整性的一部分。

**Independent Test**: 可通过查询 Envoy stats 端点独立验证。

**Acceptance Scenarios**:

1. **Given** delay 故障成功注入，**When** 查询 Envoy `/stats/prometheus` 端点，**Then** `wasmcustom_hfi_faults_delays_total` 计数器应递增。

2. **Given** 多个 delay 故障以不同延迟时间注入，**When** 查询 histogram 指标，**Then** `wasmcustom_hfi_faults_delay_duration_milliseconds` 应正确记录延迟分布。

---

### Edge Cases

- **大延迟值**：当配置 30 秒或更长的延迟时，系统应限制最大延迟为 30 秒
- **零延迟**：当 `fixed_delay_ms: 0` 时，应等同于无故障（不暂停请求）
- **并发请求**：多个并发请求同时触发 delay 时，每个请求应独立延迟

## Requirements

### Functional Requirements

- **FR-001**: WASM 插件 MUST 能够成功执行 delay 故障，暂停匹配请求指定的毫秒数
- **FR-002**: WASM 插件 MUST 使用有效的 Envoy 集群（或其他机制）来实现延迟，而非不存在的 `hfi_delay_cluster`
- **FR-003**: 系统 MUST 支持新的整数格式 `fixed_delay_ms`（毫秒）作为延迟配置
- **FR-004**: 系统 MUST 移除对字符串格式 `fixed_delay` 的支持，删除 `parse_duration` 函数
- **FR-005**: 所有示例策略文件 MUST 更新为使用新的 `fixed_delay_ms` 字段
- **FR-006**: 验证脚本 MUST 能够验证 delay 故障正确生效
- **FR-007**: Delay 故障执行时 MUST 正确递增 `wasmcustom_hfi_faults_delays_total` 指标
- **FR-008**: 系统 MUST 限制最大延迟时间为 30,000ms（30 秒）以防止资源耗尽

### Key Entities

- **DelayAction**: 延迟故障配置，核心属性从 `fixed_delay: String` 改为 `fixed_delay_ms: u64`
- **Fault**: 故障配置容器，包含可选的 `DelayAction`
- **Policy YAML**: 策略配置文件，需要更新字段格式

## Success Criteria

### Measurable Outcomes

- **SC-001**: `validate-basic.sh` 的 delay 测试通过率从 0% 提升到 100%
- **SC-002**: 配置 500ms delay 后，实际响应延迟在 450ms-550ms 范围内（±10% 误差）
- **SC-003**: 代码复杂度降低：删除 `parse_duration` 函数及其相关测试（约减少 50 行代码）
- **SC-004**: 所有现有单元测试和集成测试在修改后仍然通过
- **SC-005**: Envoy 日志中不再出现 "dispatch_http_call: BadArgument" 错误

## Assumptions

- Envoy 有可用的 upstream 集群可用于实现延迟机制（如 blackhole 集群或使用 `set_effective_context` + timer）
- 用户愿意接受配置格式的 breaking change
- 最大 30 秒延迟对所有用例足够
