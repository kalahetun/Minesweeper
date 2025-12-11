# Research: Fix WASM Plugin Delay Fault Bug

**Feature**: 010-fix-wasm-delay-bug  
**Date**: 2025-12-11  
**Status**: Complete

## Research Questions

### RQ-1: 如何在 proxy-wasm 中实现延迟而不依赖外部集群？

**背景**: 当前实现使用 `dispatch_http_call("hfi_delay_cluster", ...)` 尝试通过 HTTP 调用超时机制模拟延迟，但该集群不存在导致 `BadArgument` 错误。

**研究发现**:

1. **proxy-wasm SDK 限制**: SDK 没有原生的 "sleep" 或 "delay" API。可用的异步机制包括：
   - `dispatch_http_call` - 需要有效的 upstream 集群
   - `dispatch_grpc_call` - 同上
   - `set_tick_period` / `on_tick` - 仅适用于 RootContext，不适用于 HttpContext

2. **可行方案对比**:

   | 方案 | 可行性 | 复杂度 | 优缺点 |
   |------|--------|--------|--------|
   | A. 使用已有控制平面集群 | ✅ 高 | 低 | 复用现有集群，零基础设施变更 |
   | B. 创建 BlackHole 集群 | ⚠️ 中 | 中 | 需要修改 Envoy/Istio 配置 |
   | C. 使用 Envoy Lua Filter | ❌ 低 | 高 | 需要额外 filter 链配置 |
   | D. Pause + 外部 Timer | ❌ 低 | 高 | proxy-wasm 不支持 HttpContext timer |

3. **推荐方案**: **方案 A - 使用已有控制平面集群**

   ```rust
   // 当前（失败）:
   dispatch_http_call("hfi_delay_cluster", ...)  // ❌ 集群不存在
   
   // 修复后:
   dispatch_http_call(
       "outbound|8080||hfi-control-plane.boifi.svc.cluster.local",  // ✅ 已有集群
       vec![
           (":method", "GET"),
           (":path", "/__delay"),  // 控制平面可忽略的路径
           (":authority", "hfi-control-plane"),
       ],
       None,
       vec![],
       Duration::from_millis(delay_ms),  // 使用超时作为延迟
   )
   ```

**Decision**: 使用方案 A，复用 `CONTROL_PLANE_CLUSTER` 常量
**Rationale**: 零基础设施变更，最小代码改动，立即可用
**Alternatives Rejected**: 
- 方案 B 需要 Istio ServiceEntry 配置变更
- 方案 C/D 复杂度过高，不符合 Constitution VII (简洁性)

---

### RQ-2: 延迟实现的回调处理逻辑

**背景**: 使用 `dispatch_http_call` 实现延迟时，需要正确处理 `on_http_call_response` 回调。

**研究发现**:

1. **超时行为**: 当 `dispatch_http_call` 超时时，`on_http_call_response` 会被调用，`body_size = 0`，`trailers = 0`
2. **正常响应**: 如果集群实际返回响应（在超时前），同样会触发回调
3. **关键点**: 回调中需要调用 `resume_http_request()` 让请求继续

**当前代码分析**:

```rust
// lib.rs 中已有回调处理:
fn on_http_call_response(&mut self, _token_id: u32, _num_headers: usize, _body_size: usize, _num_trailers: usize) {
    if let Some(action) = self.pending_action.take() {
        match action {
            PendingAction::DelayFault => {
                // 延迟完成，恢复请求
                self.resume_http_request();
            }
            ...
        }
    }
}
```

**Decision**: 现有回调逻辑正确，无需修改
**Rationale**: 已正确处理 `PendingAction::DelayFault` 状态并调用 `resume_http_request()`

---

### RQ-3: `fixed_delay` 字段类型变更影响范围

**背景**: 将 `fixed_delay: String` 改为 `fixed_delay_ms: u64` 需要评估影响范围。

**研究发现**:

1. **WASM 插件端 (Rust)**:
   - `config.rs`: `DelayAction` 结构体字段变更
   - `config.rs`: 删除 `parse_duration` 函数和 `parsed_duration_ms` 字段
   - `lib.rs`: 更新 delay 读取逻辑
   - `executor.rs`: 更新 `execute_delay` 函数签名

2. **Control Plane 端 (Go)**:
   - `api/types.go`: Policy 结构体可能需要更新
   - `service/policy.go`: 验证逻辑可能需要调整

3. **CLI 端 (Go)**:
   - `types/policy.go`: DelayAction 结构体
   - 所有 YAML 示例文件

4. **需要更新的配置文件**:
   ```bash
   executor/cli/examples/basic/delay-policy.yaml
   executor/cli/examples/basic/percentage-policy.yaml
   executor/cli/examples/advanced/header-policy.yaml
   # 等等...
   ```

**Decision**: 同时更新所有组件以保持一致性
**Rationale**: Breaking change 需要同步更新，避免版本不兼容
**Migration**: 新格式 `fixed_delay_ms: 500` 替代 `fixed_delay: "500ms"`

---

### RQ-4: 最大延迟限制实现

**背景**: 规范要求限制最大延迟为 30,000ms。

**研究发现**:

1. **实现位置**: 应在 WASM 插件端验证，而非 Control Plane
2. **原因**: WASM 插件是执行层，应做最后防线验证
3. **实现方式**:

   ```rust
   const MAX_DELAY_MS: u64 = 30_000;
   
   let effective_delay = delay.fixed_delay_ms.min(MAX_DELAY_MS);
   if delay.fixed_delay_ms > MAX_DELAY_MS {
       warn!("Delay {} exceeds maximum {}, clamping", delay.fixed_delay_ms, MAX_DELAY_MS);
   }
   ```

**Decision**: 在 WASM 插件中实现 clamp 逻辑并记录警告日志
**Rationale**: 防御性编程，不拒绝请求但限制极端值

---

## Summary of Decisions

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 延迟实现机制 | 复用 CONTROL_PLANE_CLUSTER | 零基础设施变更 |
| 字段类型 | `fixed_delay_ms: u64` | 简化解析，消除运行时错误 |
| 最大延迟限制 | 30,000ms (clamp) | 防止资源耗尽 |
| 向后兼容 | 不支持旧格式 | Breaking change，简化代码 |

## References

- [proxy-wasm-rust-sdk dispatch_http_call](https://github.com/proxy-wasm/proxy-wasm-rust-sdk/blob/main/src/hostcalls.rs)
- [Envoy WASM ABI](https://github.com/proxy-wasm/spec)
- [BOIFI Constitution](../../.specify/memory/constitution.md)
