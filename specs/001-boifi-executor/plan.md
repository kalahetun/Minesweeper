````markdown
# Implementation Plan: Executor 项目测试重构与架构优化

**Branch**: `001-boifi-executor` | **Date**: 2025-11-14 | **Spec**: `/specs/001-boifi-executor/spec.md`
**Input**: Feature specification from `/specs/001-boifi-executor/spec.md`

## Summary

executor 项目包含三个独立的组件（Control Plane、CLI、Wasm Plugin），各自有不同的编程语言（Go、Go、Rust）和测试框架。当前测试文件分散在各组件目录中，缺乏统一的组织结构。本次重构的目标是：

1. **统一测试架构**: 为三个组件建立清晰、一致的测试目录结构，包括单元测试、集成测试和端到端测试
2. **改进测试可维护性**: 按测试类型和功能模块组织，便于团队快速定位和维护测试代码
3. **强化测试覆盖率**: 按照宪法要求（核心路径 >90% 覆盖率），系统评估和补充缺失的测试
4. **建立测试规范**: 文档化测试命名约定、运行方式和CI集成规范
5. **添加性能基准**: 为关键热路径（Matcher、Executor、Policy Service）建立基准测试
6. **改进开发体验**: 提供清晰的快速启动指南和测试文档

## Technical Context

**Language/Version**: Go 1.21+ (Control Plane, CLI), Rust 1.75+ (Wasm Plugin)  
**Primary Dependencies**: 
- Control Plane: `gin`, `etcd`, `cobra`
- CLI: `cobra`, `yaml`
- Wasm Plugin: `proxy-wasm`, `serde`, `regex`

**Storage**: etcd (可选，当前支持内存存储)  
**Testing**: 
- Go: `testing` (标准库), `testify`, `gomock`
- Rust: `cargo test`, `criterion` (基准测试)

**Target Platform**: Linux/Kubernetes (container deployment)
**Project Type**: Multi-component system (Go backend + Go CLI + Rust WASM)  
**Performance Goals**: 
- Control Plane API: <100ms p99 响应时间
- Wasm Plugin: <1ms 每请求开销
- Policy 分发: <1s 端到端延迟

**Constraints**: 
- Wasm plugin 内存占用: <100MB
- Control Plane 可服务 10+ 并发连接
- 测试执行时间: 单个组件 <30s

**Scale/Scope**: 
- 17 个现有测试文件（分散在3个组件中）
- 总项目代码：约477MB（含编译产物）
- 目标：整合、标准化、补充缺失覆盖率

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### 宪法对齐度检查

✅ **I. 关注点分离** - 符合
- 三个组件（CP/CLI/Plugin）已清晰分离，各自独立可测
- 本重构将进一步强化每个组件的单元/集成/E2E 三层测试结构

✅ **II. 声明式配置** - 符合
- Policy YAML 配置已标准化，测试覆盖了多个场景
- 重构将补充配置验证和边界情况的测试

✅ **IV. 测试驱动（强制）** - **部分符合，需改进**
- 当前测试覆盖率未达标（< 70% 核心模块）
- **必须行动**: 补充 Matcher、TimeControl、PolicyService 的单元测试
- **必须行动**: 建立性能基准测试（Matcher, Executor）

⚠️ **V. 性能优先** - 部分符合，需基准测试
- 当前缺少基准测试验证 <1ms 延迟目标
- **必须行动**: 为 Matcher、Executor 添加 criterion 基准测试

✅ **VI. 容错与可靠性** - 符合
- 现有测试覆盖了网络分区、重连等场景
- 需强化：测试 fail-safe 路径

✅ **VII. 简洁性** - 符合
- 当前存储和通信设计足够简洁
- 重构应保持简洁，避免过度工程化

**初步评估**: 通过基本审查，但需在 Phase 0 深化测试覆盖率分析

## Project Structure

### Documentation (this feature)

```text
specs/001-boifi-executor/
├── plan.md                          # 本文件
├── research.md                      # Phase 0: 测试覆盖率分析、最佳实践研究
├── data-model.md                    # Phase 1: 测试架构数据模型（Test Pyramid, Coverage Matrix）
├── test-architecture.md             # Phase 1: 详细测试组织结构和运行指南
├── quickstart.md                    # Phase 1: 测试快速启动指南
└── contracts/
    ├── test-structure-api.md        # 三个组件的测试目录 API 规范
    └── benchmark-specification.md   # 性能基准测试规范
```

### Source Code - Current Structure

```text
executor/
├── cli/
│   ├── main.go, client/, cmd/, types/
│   ├── examples/ (YAML 策略示例)
│   └── [无测试目录，tests 散在 cmd 中]
│
├── control-plane/
│   ├── main.go, distributor.go
│   ├── api/, logger/, middleware/, service/, storage/
│   ├── service/*_test.go (6 个测试文件混入业务代码目录)
│   ├── storage/*_test.go (3 个测试文件混入存储目录)
│   ├── integration_test.go (顶级集成测试)
│   └── [缺少统一的 tests/ 目录结构]
│
└── wasm-plugin/
    ├── src/
    │   ├── core/, bin/
    │   ├── lib.rs, config.rs, executor.rs, matcher.rs, ...
    │   ├── int_1_comprehensive_unit_tests.rs (混入src)
    │   ├── int_2_multi_rules_tests.rs
    │   ├── int_3_end_to_end_tests.rs
    │   ├── test_*.rs (9 个测试文件散布)
    │   └── [缺少 tests/ 目录和性能基准]
    ├── Cargo.toml
    └── target/ (编译产物)
```

### Source Code - Target Structure (Phase 1 输出)

```text
executor/
├── cli/
│   ├── src/
│   │   ├── main.go, client/, cmd/, types/
│   │   └── examples/
│   ├── tests/
│   │   ├── unit/
│   │   │   ├── cmd_test.go (命令行解析)
│   │   │   ├── client_test.go (HTTP 客户端)
│   │   │   └── types_test.go (类型验证)
│   │   ├── integration/
│   │   │   ├── cli_control_plane_test.go (CLI vs Control Plane API)
│   │   │   └── policy_lifecycle_test.go (完整策略CRUD)
│   │   └── fixtures/
│   │       └── sample_policies/ (测试用 YAML 文件)
│   └── Makefile (test, test-coverage, test-all)
│
├── control-plane/
│   ├── src/
│   │   ├── main.go, distributor.go
│   │   ├── api/, logger/, middleware/, service/, storage/
│   │   └── models/ (新增，集中业务模型定义)
│   ├── tests/
│   │   ├── unit/
│   │   │   ├── api_test.go
│   │   │   ├── service_test.go
│   │   │   ├── storage_test.go
│   │   │   ├── validator_test.go
│   │   │   └── expiration_registry_test.go
│   │   ├── integration/
│   │   │   ├── policy_lifecycle_test.go
│   │   │   ├── sse_distribution_test.go
│   │   │   ├── concurrent_access_test.go
│   │   │   └── failover_test.go
│   │   ├── e2e/
│   │   │   ├── full_workflow_test.go
│   │   │   └── plugin_integration_test.go
│   │   ├── benchmarks/
│   │   │   ├── policy_service_bench_test.go
│   │   │   └── matcher_bench_test.go
│   │   └── fixtures/
│   │       ├── policies.go (预定义测试 Policy 对象)
│   │       └── test_data.yaml
│   └── Makefile (test, test-coverage, bench)
│
└── wasm-plugin/
    ├── src/
    │   ├── lib.rs, main.rs
    │   ├── core/, bin/
    │   ├── config.rs, executor.rs, matcher.rs, metrics.rs, ...
    │   └── [仅 library 代码，无测试]
    ├── tests/
    │   ├── unit/
    │   │   ├── matcher_test.rs (正则/路径匹配)
    │   │   ├── executor_test.rs (故障执行)
    │   │   ├── config_test.rs (配置解析)
    │   │   ├── time_control_test.rs
    │   │   └── reconnect_test.rs
    │   ├── integration/
    │   │   ├── multi_rules_test.rs (多规则交互)
    │   │   ├── concurrent_rules_test.rs (并发安全)
    │   │   └── panic_safety_test.rs (恐慌恢复)
    │   ├── e2e/
    │   │   ├── full_injection_test.rs (完整故障注入流程)
    │   │   └── policy_update_test.rs (实时策略更新)
    │   ├── benchmarks/
    │   │   ├── matcher_bench.rs (criterion)
    │   │   ├── executor_bench.rs
    │   │   └── rule_compilation_bench.rs
    │   └── fixtures/
    │       └── policies.rs (预定义测试 Policy)
    ├── Cargo.toml (添加 [dev-dependencies] benches)
    └── Makefile (test, test-coverage, bench)
```

**结构决策**: 采用 **三层三级制** 的统一测试架构
- **三层**: 单元测试 (unit) → 集成测试 (integration) → 端到端测试 (e2e)
- **三级**: 代码级 (src/*_test) / 组件级 (tests/) / 系统级 (e2e/)
- **优势**: 
  - 符合宪法的分层测试要求（IV.测试驱动）
  - 便于团队理解和维护
  - 性能基准集中管理
  - 支持渐进式执行（快速反馈 vs 全量验证）

## Complexity Tracking

> **Constitution Violations Justification**

无违反。重构旨在更好地遵守宪法要求，特别是：
- **IV. 测试驱动**: 通过标准化结构和补充缺失测试，达到 >70% 覆盖率
- **V. 性能优先**: 通过基准测试验证 <1ms 延迟目标
- **VI. 容错**: 强化网络分区、恐慌恢复等场景测试

## Next Steps (Phase 0)

1. 分析当前测试覆盖率缺口（research.md）
2. 列举缺失的关键测试场景
3. 评估现有测试可复用性
4. 制定迁移策略（避免代码重复）

````
