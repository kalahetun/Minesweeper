# Implementation Plan: Fix WASM Plugin Delay Fault Bug

**Branch**: `010-fix-wasm-delay-bug` | **Date**: 2025-12-11 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/010-fix-wasm-delay-bug/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

修复 WASM 插件 delay 故障注入失败的 Bug（`dispatch_http_call: BadArgument`），根因是调用了不存在的 `hfi_delay_cluster` 集群。解决方案是使用已有的控制平面集群实现延迟机制。同时将 `fixed_delay` 字段类型从 `String` 改为 `u64`（毫秒），删除 `parse_duration` 函数以简化代码。

## Technical Context

**Language/Version**: Rust 1.75+ (Wasm Plugin), Go 1.20+ (Control Plane, CLI)  
**Primary Dependencies**: proxy-wasm-rust-sdk, serde, gin (Control Plane API)  
**Storage**: N/A (策略存储在内存中，通过 Control Plane 分发)  
**Testing**: cargo test, go test, E2E 验证脚本 (validate-basic.sh)  
**Target Platform**: WASM (Envoy sidecar), Kubernetes + Istio  
**Project Type**: single - 单一 WASM 插件项目 + Control Plane  
**Performance Goals**: Wasm 插件开销 < 1ms，延迟精度 ±10%  
**Constraints**: 最大延迟 30,000ms，不引入新的外部依赖  
**Scale/Scope**: 影响 wasm-plugin, control-plane, cli, examples 4 个模块

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| 原则 | 状态 | 说明 |
|------|------|------|
| I. 关注点分离 | ✅ 通过 | 修改仅限于 Wasm 插件内部实现，不影响 Control Plane 接口 |
| II. 声明式配置 | ✅ 通过 | 配置格式从 `fixed_delay: "500ms"` 简化为 `fixed_delay_ms: 500`，仍是声明式 |
| III. 动态性与实时性 | ✅ 通过 | 不影响策略热更新机制 |
| IV. 测试驱动 | ✅ 通过 | 需更新现有测试，添加 delay 故障 E2E 验证 |
| V. 性能优先 | ✅ 通过 | 删除运行时字符串解析，简化代码路径 |
| VI. 容错与可靠性 | ✅ 通过 | 修复后 delay 故障失败应正确 fallback，不影响请求处理 |
| VII. 简洁性与最小化 | ✅ 通过 | 删除 ~50 行解析代码，简化配置格式 |
| VIII. 时间控制 | ✅ 通过 | 不影响 `start_delay_ms` 和 `duration_seconds` 逻辑 |

**GATE 结果**: ✅ 全部通过，可进入 Phase 0

## Project Structure

### Documentation (this feature)

```text
specs/010-fix-wasm-delay-bug/
├── plan.md              # This file
├── research.md          # Phase 0 output - 延迟实现方案研究
├── data-model.md        # Phase 1 output - DelayAction 数据模型
├── quickstart.md        # Phase 1 output - 快速开始指南
├── contracts/           # Phase 1 output - API 契约无变化
└── tasks.md             # Phase 2 output (by /speckit.tasks)
```

### Source Code (repository root)

```text
executor/wasm-plugin/
├── src/
│   ├── lib.rs           # 修改: 移除 hfi_delay_cluster 调用，使用新延迟机制
│   ├── config.rs        # 修改: DelayAction 字段类型变更，删除 parse_duration
│   ├── executor.rs      # 修改: execute_delay 函数更新
│   └── ...
└── tests/
    └── integration_tests.rs  # 更新: 新配置格式测试

executor/control-plane/
├── api/                 # 可能需要更新 Policy 验证逻辑
└── service/             # 可能需要更新策略处理

executor/cli/
├── types/               # 更新: Policy 结构体字段
└── examples/            # 更新: 所有 delay 策略示例文件
    ├── basic/
    ├── advanced/
    └── scenarios/
```

**Structure Decision**: 采用 Option 1 (Single project) 结构，修改集中在 `executor/wasm-plugin/` 和配置文件中。

## Complexity Tracking

> **无违规需要说明** - 所有 Constitution 检查项均通过
