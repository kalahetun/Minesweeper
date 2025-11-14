# Task List: Executor 项目测试重构与架构优化

**Feature**: Executor 项目测试重构与架构优化  
**Branch**: `001-boifi-executor`  
**Generated**: 2025-11-14  
**Spec**: `/specs/001-boifi-executor/spec.md`  
**Plan**: `/specs/001-boifi-executor/plan.md`  

---

## Executive Summary

| Metric | Value |
|--------|-------|
| **总任务数** | 68 |
| **核心阶段** | 6 (Setup + Foundational + 4 User Stories) |
| **并行机会** | 32 个任务可并行执行 |
| **MVP 推荐范围** | Phase 2 + Phase 3 (US1: Manual Chaos Testing) |
| **预计工作量** | 6-8 周（完整），2-3 周（MVP） |

### User Stories 优先级与依赖

```
Phase 1: Setup & Foundational (必须完成，阻塞所有故事)
    ↓
Phase 3: US1 - SRE Manual Chaos Testing (P1) ← MVP 核心
    ├─→ Phase 4: US2 - Policy Lifecycle Management (P1)
    │   ├─→ Phase 5: US3 - High-Performance Plugin Execution (P1)
    │       ├─→ Phase 6: US4 - Recommender Integration (P2)
    │       └─→ Phase 7: US5 - Cloud-Native Deployment (P2)
    └─→ [并行] Phase 8: Polish & Cross-Cutting Concerns

**独立可测的用户故事**: 每个故事可独立实现和验证
- US1 通过: 启动 CP+Plugin，CLI 应用策略，发送测试请求，验证故障注入 ✓
- US2 通过: 执行 CLI policy CRUD，验证持久化 ✓
- US3 通过: 加载 10 个策略，1000req/sec，测量 <1ms 延迟 ✓
- US4 通过: Recommender API 调用，验证存储和分发 ✓
- US5 通过: Docker-compose 启动，Kubernetes 部署验证 ✓
```

---

## Phase 1: 项目初始化与基础设施 (Setup)

**目标**: 建立测试目录结构、工具链和共享基础设施

**可并行任务**: T001-T012（大部分独立）

- [ ] T001 在 `/executor/cli/` 下创建 `tests/` 目录结构 (unit/, integration/, fixtures/)
- [ ] T002 [P] 在 `/executor/control-plane/` 下创建 `tests/` 目录结构 (unit/, integration/, e2e/, benchmarks/, fixtures/)
- [ ] T003 [P] 在 `/executor/wasm-plugin/` 下创建 `tests/` 目录结构 (unit/, integration/, e2e/, benchmarks/, fixtures/)
- [ ] T004 为 Control Plane 创建 Makefile 目标: `make test`, `make test-coverage`, `make test-integ`, `make bench`, `make test-all` 到 `/executor/control-plane/Makefile`
- [ ] T005 [P] 为 CLI 创建 Makefile 目标: `make test`, `make test-coverage` 到 `/executor/cli/Makefile`
- [ ] T006 [P] 为 Wasm Plugin 创建 Makefile 目标: `make test`, `make test-coverage`, `make bench` 到 `/executor/wasm-plugin/Makefile`
- [ ] T007 创建 Control Plane 测试夹具模块 `/executor/control-plane/tests/fixtures/policies.go` 包含预定义 Policy 对象
- [ ] T008 [P] 创建 Wasm Plugin 测试夹具模块 `/executor/wasm-plugin/tests/fixtures/policies.rs` 包含预定义 Policy 对象
- [ ] T009 [P] 创建 CLI 测试夹具目录 `/executor/cli/tests/fixtures/sample_policies/` 包含 YAML 测试文件
- [ ] T010 生成测试架构文档 `/specs/001-boifi-executor/test-architecture.md` 说明目录结构、命名约定和运行规范
- [ ] T011 [P] 生成快速启动指南 `/specs/001-boifi-executor/quickstart.md` 包含测试运行示例
- [ ] T012 生成测试覆盖率基线报告 `/specs/001-boifi-executor/research.md` 分析当前覆盖率缺口

**验收标准**:
- ✓ 三个组件都有标准化的 tests/ 目录结构
- ✓ Makefile 支持标准化测试命令
- ✓ 测试夹具可被所有单元测试引用
- ✓ 文档清晰可执行

---

## Phase 2: 测试基础设施与迁移 (Foundational)

**目标**: 迁移现有测试，建立覆盖率基线和性能基准基础

**阻塞**: 所有用户故事测试依赖此阶段完成  
**可并行任务**: T013-T030（大部分独立）

### Control Plane 测试迁移

- [ ] T013 迁移 Control Plane 单元测试: `service/*_test.go` → `/executor/control-plane/tests/unit/service_test.go`
- [ ] T014 [P] 迁移 Control Plane 存储测试: `storage/*_test.go` → `/executor/control-plane/tests/unit/storage_test.go`
- [ ] T015 [P] 迁移 Control Plane 集成测试: `integration_test.go` → `/executor/control-plane/tests/integration/integration_test.go`
- [ ] T016 更新 Control Plane 测试 import 路径（因目录重组）在 `/executor/control-plane/tests/`

### Wasm Plugin 测试迁移

- [ ] T017 [P] 整合 Wasm Plugin 单元测试: `test_w5_unit.rs`, `test_basic.rs` → `/executor/wasm-plugin/tests/unit/core_test.rs`
- [ ] T018 [P] 整合 Wasm Plugin 集成测试: `int_1_*.rs`, `int_2_*.rs` → `/executor/wasm-plugin/tests/integration/rules_test.rs`
- [ ] T019 [P] 整合 Wasm Plugin E2E 测试: `int_3_*.rs`, `test_w5_integration.rs` → `/executor/wasm-plugin/tests/e2e/e2e_test.rs`
- [ ] T020 从 src/ 中移除旧的 `test_*.rs` 和 `int_*.rs` 文件
- [ ] T021 更新 Wasm Plugin Cargo.toml 指向新的测试目录结构

### 初始覆盖率报告生成

- [ ] T022 运行 Control Plane 覆盖率测试: `make test-coverage` 在 `/executor/control-plane/` 生成报告
- [ ] T023 [P] 运行 Wasm Plugin 覆盖率测试: `make test-coverage` 在 `/executor/wasm-plugin/` 生成报告
- [ ] T024 [P] 运行 CLI 覆盖率测试: `make test-coverage` 在 `/executor/cli/` 生成报告
- [ ] T025 汇总覆盖率结果到 `/specs/001-boifi-executor/research.md` 标记优先补充的模块

### 性能基准框架建立

- [ ] T026 为 Wasm Plugin Cargo.toml 添加 criterion 基准测试依赖 `[dev-dependencies] criterion`
- [ ] T027 [P] 为 Go 项目添加基准测试框架 (testing.B) 到 `/executor/control-plane/Makefile`
- [ ] T028 创建 Wasm Plugin 基准测试骨架 `/executor/wasm-plugin/tests/benchmarks/` 结构

### CI/CD 集成准备

- [ ] T029 验证三个组件的测试均可独立执行 (make test 成功)
- [ ] T030 [P] 创建根层 Makefile 支持 `make test-all` (运行三个组件的测试)

**验收标准**:
- ✓ 所有现有测试成功迁移且通过
- ✓ 覆盖率基线已建立（标记缺口）
- ✓ 基准测试框架可运行
- ✓ 三个组件都通过 make test-all

---

## Phase 3: User Story 1 - SRE 手动混沌测试 (P1)

**目标**: 实现端到端的策略应用和故障注入验证流程

**依赖**: Phase 1 & 2 完成  
**独立测试**: 启动 CP+Plugin → CLI 应用策略 → 发送请求 → 验证故障  
**成功标准**: SC-001, SC-002, SC-003, SC-004, SC-007, SC-012

### Control Plane - 策略管理基础

- [ ] T031 [P] 创建 Control Plane API 集成测试 `/executor/control-plane/tests/integration/api_test.go` 验证 POST /v1/policies
- [ ] T032 补充 Validator 单元测试 `/executor/control-plane/tests/unit/validator_test.go` 覆盖策略验证规则 (缺失的必需字段、无效 JSON、等)
- [ ] T033 [US1] 创建 Policy Service 集成测试 `/executor/control-plane/tests/integration/policy_service_test.go` 验证 CRUD 操作
- [ ] T034 [US1] 补充 ExpirationRegistry 并发测试 `/executor/control-plane/tests/integration/expiration_test.go` (策略自动过期)

### Wasm Plugin - 匹配与执行核心

- [ ] T035 [P] 补充 Matcher 单元测试 `/executor/wasm-plugin/tests/unit/matcher_test.rs` 覆盖正则表达式、路径前缀、头部匹配的边界情况
- [ ] T036 补充 Executor 单元测试 `/executor/wasm-plugin/tests/unit/executor_test.rs` 覆盖 Abort 和 Delay 故障类型的原子性
- [ ] T037 [US1] 创建 Wasm Plugin 集成测试 `/executor/wasm-plugin/tests/integration/stateful_test.rs` 验证请求隔离（无状态泄露）

### CLI - 策略应用

- [ ] T038 [P] 创建 CLI 单元测试 `/executor/cli/tests/unit/client_test.go` 验证 HTTP 通信和错误处理
- [ ] T039 创建 CLI 命令测试 `/executor/cli/tests/unit/cmd_test.go` 验证 `policy apply` 命令解析
- [ ] T040 [US1] 创建 CLI 集成测试 `/executor/cli/tests/integration/apply_test.go` 验证端到端应用流程

### E2E 测试 - 完整流程

- [ ] T041 创建 US1 E2E 测试 `/executor/control-plane/tests/e2e/e2e_manual_chaos_test.go`
  - 场景 1: SRE 应用 abort 50% 策略 → 验证 1 秒内分发 → 发送请求 → 验证故障注入
  - 场景 2: 更新策略从 abort → delay → 验证实时切换
  - 场景 3: 多策略匹配 → 验证第一条规则生效
  - 场景 4: duration_seconds=60 → 验证 60 秒后自动删除

- [ ] T042 创建分布式 E2E 测试 `/executor/wasm-plugin/tests/e2e/distribution_test.rs`
  - Control Plane → Plugin SSE → 规则更新 → 请求处理 → 验证故障

### 文档与运行验证

- [ ] T043 [US1] 更新快速启动指南 `/specs/001-boifi-executor/quickstart.md` 包含 US1 运行步骤
- [ ] T044 创建 US1 独立运行脚本 `/executor/test-us1.sh` 验证整个流程可重复

**验收标准 (Phase 3)**:
- ✓ Policy CRUD 所有 API 端点都有集成测试
- ✓ Matcher 和 Executor 原子性已验证
- ✓ E2E 测试覆盖 4 个接受场景
- ✓ 故障注入准确性 > 99.9%
- ✓ 策略分发延迟 < 1 秒
- ✓ Control Plane API 响应 < 100ms

---

## Phase 4: User Story 2 - 实时策略生命周期管理 (P1)

**目标**: 完整的策略 CRUD 操作和时间控制

**依赖**: Phase 3 完成  
**独立测试**: CLI policy apply/get/delete/list → 验证持久化和实时响应  
**成功标准**: SC-001, SC-007, SC-009, SC-010, SC-011, SC-014

### Policy Lifecycle 完整测试

- [ ] T045 [P] 创建 Policy 生命周期集成测试 `/executor/control-plane/tests/integration/lifecycle_test.go`
  - Create: 应用新策略 → 验证创建成功
  - Read: Get 单个策略 → 验证详情完整
  - Update: 更新策略 → 验证规则变化
  - Delete: 删除策略 → 验证移除

- [ ] T046 补充时间控制测试 `/executor/control-plane/tests/unit/time_control_test.go`
  - start_delay_ms: 验证延迟激活
  - duration_seconds: 验证精度 ±50ms (请求 >500ms)

- [ ] T047 [US2] 创建 CLI 命令完整测试 `/executor/cli/tests/integration/lifecycle_test.go`
  - `policy apply -f policy.yaml` → 验证创建
  - `policy get <name>` → 验证详情
  - `policy list` → 验证列表和表格格式
  - `policy delete <name>` → 验证删除

### Temporal Control 验证

- [ ] T048 创建 Wasm Plugin 时间控制测试 `/executor/wasm-plugin/tests/integration/temporal_test.rs`
  - start_delay_ms > request_duration: 验证不注入故障
  - duration_seconds 过期: 验证规则过期时不应用

- [ ] T049 补充过期机制测试 `/executor/control-plane/tests/integration/expiration_test.go` 验证自动删除精度

### 错误处理与验证

- [ ] T050 [P] 补充 API 错误处理测试 `/executor/control-plane/tests/unit/api_errors_test.go`
  - 缺失必需字段 → 400 Bad Request
  - 无效 JSON → 400 Bad Request
  - 重复名称 → 409 Conflict 或 Update
  - 非法正则表达式 → 400 Bad Request

- [ ] T051 创建 CLI 错误提示测试 `/executor/cli/tests/unit/error_messages_test.go`
  - 验证错误消息可操作（指导用户）

### 文档与运行验证

- [ ] T052 [US2] 更新快速启动指南 `/specs/001-boifi-executor/quickstart.md` 包含 US2 CRUD 示例
- [ ] T053 创建 US2 独立运行脚本 `/executor/test-us2.sh`

**验收标准 (Phase 4)**:
- ✓ Policy CRUD 覆盖率 > 90%
- ✓ 时间控制精度 ±50ms
- ✓ 所有错误情况都有验证和清晰提示
- ✓ 并发 10 个策略操作无冲突
- ✓ CLI 命令响应 < 2 秒

---

## Phase 5: User Story 3 - 高性能插件执行 (P1)

**目标**: 验证 <1ms 延迟目标，建立性能基准

**依赖**: Phase 3 & 4 完成  
**独立测试**: 加载 10 策略 → 1000 req/sec → 测量 p99 延迟  
**成功标准**: SC-003, SC-004, SC-006, SC-010

### 性能基准测试建立

- [ ] T054 [P] 创建 Wasm Plugin Matcher 性能基准 `/executor/wasm-plugin/tests/benchmarks/matcher_bench.rs` (criterion)
  - 单规则匹配: 基准数据
  - 10 规则匹配: 基准数据
  - 正则表达式匹配: 基准数据
  - 目标: < 0.5ms

- [ ] T055 创建 Wasm Plugin Executor 性能基准 `/executor/wasm-plugin/tests/benchmarks/executor_bench.rs`
  - Abort 执行: 基准数据
  - Delay 执行: 基准数据
  - 目标: < 0.3ms

- [ ] T056 [P] 创建规则编译性能基准 `/executor/wasm-plugin/tests/benchmarks/compilation_bench.rs`
  - 编译 100 规则: 基准数据

- [ ] T057 创建 Control Plane Policy Service 性能基准 `/executor/control-plane/tests/benchmarks/policy_service_bench_test.go`
  - Create/Update/Delete: 基准数据
  - List 100 策略: 基准数据
  - 并发 10 更新: 基准数据
  - 目标: < 50ms

### 并发与原子性验证

- [ ] T058 [US3] 创建 Wasm Plugin 并发规则测试 `/executor/wasm-plugin/tests/integration/concurrent_rules_test.rs`
  - 多个规则更新: 验证原子性
  - 并发请求处理: 验证无状态泄露

- [ ] T059 创建规则缓存一致性测试 `/executor/wasm-plugin/tests/integration/cache_consistency_test.rs`
  - SSE 更新 → 旧请求使用旧规则
  - 新请求使用新规则
  - 无撕裂读

### 高并发负载测试

- [ ] T060 [US3] 创建高并发测试 `/executor/wasm-plugin/tests/e2e/load_test.rs`
  - 10 活跃策略
  - 1000 req/sec
  - 测量 p99 延迟
  - 验证 < 1ms 开销

- [ ] T061 创建内存泄漏测试 `/executor/control-plane/tests/e2e/memory_stability_test.go`
  - 24 小时运行
  - 10 并发连接
  - 验证稳定内存使用

### 性能报告与基准历史

- [ ] T062 [US3] 生成性能基准报告 `/specs/001-boifi-executor/performance-baseline.md`
  - Matcher: X us/op (< 0.5ms)
  - Executor: X us/op (< 0.3ms)
  - Policy Service: X ms/op (< 50ms)

- [ ] T063 创建性能趋势跟踪脚本 `/executor/scripts/bench-compare.sh` 用于检测回归

**验收标准 (Phase 5)**:
- ✓ Matcher 延迟 < 0.5ms
- ✓ Executor 延迟 < 0.3ms
- ✓ 1000 req/sec 下 p99 延迟 < 1ms
- ✓ 10 并发连接 24h 无内存泄漏
- ✓ 性能基准已建立和记录

---

## Phase 6: User Story 4 - Recommender 自动化集成 (P2)

**目标**: Recommender 能通过 API 编程方式提交故障注入计划

**依赖**: Phase 3 & 4 完成  
**独立测试**: Recommender POST /v1/policies → 验证存储和分发  
**成功标准**: SC-001, SC-002, SC-008, SC-009

### Recommender API 支持

- [ ] T064 [P] 创建 Recommender API 集成测试 `/executor/control-plane/tests/integration/recommender_api_test.go`
  - POST /v1/policies (FaultPlan): 201 Created
  - 返回 policy name
  - 验证存储

- [ ] T065 补充 Recommender 场景 E2E 测试 `/executor/control-plane/tests/e2e/recommender_e2e_test.go`
  - Recommender POST FaultPlan → Control Plane 存储 → Plugin 接收 → 应用故障 → 自动过期

- [ ] T066 [US4] 创建 Recommender 集成文档 `/specs/001-boifi-executor/recommender-integration.md`
  - API 示例: 创建、查询、删除
  - 预期响应格式

### 持久化与恢复

- [ ] T067 创建持久化测试 `/executor/control-plane/tests/integration/persistence_test.go`
  - 策略保存到 etcd/内存
  - Control Plane 重启后数据恢复

- [ ] T068 [P] 补充存储层测试 `/executor/control-plane/tests/unit/storage_test.go` 覆盖边界情况

**验收标准 (Phase 6)**:
- ✓ Recommender API 响应 < 100ms
- ✓ 策略创建后 1 秒内分发
- ✓ 持久化工作正常
- ✓ 自动过期精度 ±5 秒

---

## Phase 7: User Story 5 - 云原生部署 (P2)

**目标**: Kubernetes 和 Docker Compose 部署验证

**依赖**: Phase 3 & 4 完成  
**独立测试**: Docker-compose up → 健康检查 → Kubernetes deploy → 验证分发  
**成功标准**: SC-002, SC-006, SC-012

### Docker 集成验证

- [ ] T069 [P] 创建 Docker Compose 集成测试 `/executor/docker/compose-test.sh`
  - docker-compose up
  - 等待服务就绪
  - 健康检查: GET /healthz
  - 验证日志正常

- [ ] T070 创建 Control Plane 容器镜像测试 `/executor/control-plane/tests/e2e/docker_test.go`
  - 构建镜像
  - 启动容器
  - API 响应

- [ ] T071 [P] 创建 Wasm Plugin 容器加载测试 `/executor/wasm-plugin/tests/e2e/envoy_test.rs`
  - Envoy with WASM sidecar
  - 插件加载验证
  - 通信测试

### Kubernetes 部署验证

- [ ] T072 [US5] 创建 Kubernetes 部署测试 `/executor/k8s/tests/deploy_test.sh`
  - kubectl apply -f control-plane.yaml
  - 等待 Pod ready
  - 验证 SSE 连接
  - 验证策略分发

- [ ] T073 创建多实例分发测试 `/executor/k8s/tests/multi_instance_test.sh`
  - 部署 3 个 Plugin 实例
  - 应用策略
  - 验证全部 3 个接收 (< 1 秒)

### 故障恢复与扩展

- [ ] T074 [P] 创建 Control Plane 故障转移测试 `/executor/k8s/tests/failover_test.sh`
  - Pod 重启
  - 数据恢复
  - 新连接建立

- [ ] T075 创建自动扩展测试 `/executor/k8s/tests/scaling_test.sh` (可选, Phase 8)
  - Plugin 扩展时策略同步

### 部署文档

- [ ] T076 [US5] 更新部署指南 `/executor/docs/dev_doc/DEPLOYMENT.md`
  - Docker Compose 部署步骤
  - Kubernetes 部署步骤
  - 健康检查和监控

**验收标准 (Phase 7)**:
- ✓ Docker-compose 启动无错误
- ✓ Kubernetes 部署成功，Pod ready
- ✓ 10 个 Plugin 并发连接，策略分发 < 1 秒
- ✓ 故障转移和恢复工作正常

---

## Phase 8: 完善 & 跨切面关注点 (Polish & Cross-Cutting)

**目标**: 测试覆盖率达标，性能基准稳定，文档完整

**并行任务**: T077-T103（全部独立）

### 测试覆盖率最终补充

- [ ] T077 [P] 补充 Control Plane 缺失覆盖: `distributor.go` 测试 `/executor/control-plane/tests/unit/distributor_test.go` (SSE 广播逻辑)
- [ ] T078 补充 CLI 缺失覆盖: `types/policy.go` YAML 解析测试 `/executor/cli/tests/unit/types_test.go`
- [ ] T079 [P] 补充 Wasm Plugin 缺失覆盖: `config.rs` 完整测试 `/executor/wasm-plugin/tests/unit/config_test.rs` (边界和无效输入)
- [ ] T080 补充 Reconnect 逻辑测试 `/executor/wasm-plugin/tests/integration/reconnect_test.rs` (指数退避、网络分区恢复)
- [ ] T081 [P] 补充 Panic Safety 测试 `/executor/wasm-plugin/tests/integration/panic_safety_test.rs` (恐慌恢复，无数据损坏)

### 边界情况和容错

- [ ] T082 创建网络分区模拟测试 `/executor/control-plane/tests/e2e/network_partition_test.go`
  - Plugin 无法连接 → fail-safe (允许请求)
  - 恢复后重新连接 → 规则同步

- [ ] T083 [P] 创建大规模规则集测试 `/executor/wasm-plugin/tests/e2e/large_ruleset_test.rs`
  - 加载 1000 个规则
  - 验证编译和执行正常
  - 内存占用 < 100MB

- [ ] T084 创建并发冲突测试 `/executor/control-plane/tests/integration/concurrent_conflicts_test.go`
  - 两个操作员同时创建同名策略
  - 验证冲突解决策略

- [ ] T085 [P] 创建无效策略拒绝测试 `/executor/control-plane/tests/unit/validation_errors_test.go`
  - 缺失字段
  - 无效正则表达式
  - 不合法的 HTTP 方法
  - 等等

### 可观测性和日志

- [ ] T086 创建日志验证测试 `/executor/control-plane/tests/unit/logging_test.go`
  - INFO 级别: 策略 mutations
  - ERROR 级别: API 错误
  - 验证时间戳和元数据

- [ ] T087 [P] 创建健康检查测试 `/executor/control-plane/tests/unit/health_test.go`
  - GET /healthz → 200 OK (operational)
  - GET /healthz → 503 (degraded, e.g., 存储不可用)

### CLI 完整性

- [ ] T088 补充 CLI help 文档测试 `/executor/cli/tests/unit/help_test.go`
  - `hfi-cli policy --help` 输出完整
  - 所有命令都有帮助文本

- [ ] T089 [P] 补充 CLI 全局标志测试 `/executor/cli/tests/unit/flags_test.go`
  - `--control-plane-addr`
  - `--timeout`
  - `--output` (table/json/yaml)

### 集成测试覆盖完整性

- [ ] T090 创建完整工作流 E2E `/executor/control-plane/tests/e2e/complete_workflow_test.go`
  - 启动 Control Plane
  - CLI 应用策略
  - Plugin 接收
  - 请求被故障注入
  - 策略更新
  - 请求使用新规则
  - 策略过期
  - 故障停止

- [ ] T091 [P] 创建黑盒集成测试 `/executor/tests/e2e/system_test.sh` (可选)
  - 三个组件完全独立启动
  - 通过公共接口交互
  - 验证端到端功能

### 性能基准稳定性

- [ ] T092 生成性能基准报告 `/specs/001-boifi-executor/performance-results.md`
  - 记录当前基准值
  - 建立告警阈值 (>5% 回归)

- [ ] T093 [P] 创建 CI 性能检查脚本 `/executor/scripts/ci-bench-check.sh`
  - 运行基准测试
  - 对比历史结果
  - 失败如果回归 > 5%

### 文档完整性

- [ ] T094 完成测试架构文档 `/specs/001-boifi-executor/test-architecture.md`
  - 命名约定
  - 目录结构说明
  - 运行方式

- [ ] T095 [P] 完成快速启动指南 `/specs/001-boifi-executor/quickstart.md`
  - 5 分钟快速启动
  - 运行每个 User Story 的步骤
  - 调试常见问题

- [ ] T096 生成测试覆盖率最终报告 `/specs/001-boifi-executor/coverage-final.md`
  - 所有组件覆盖率 > 70%
  - 核心路径 > 90%
  - 按模块详细列出

- [ ] T097 [P] 更新 ARCHITECTURE.md `/executor/docs/dev_doc/ARCHITECTURE.md`
  - 添加测试架构部分
  - 说明如何添加新测试

- [ ] T098 生成 TROUBLESHOOTING.md 测试章节 `/executor/docs/dev_doc/TROUBLESHOOTING.md`
  - 常见测试失败原因
  - 调试方法

### CI/CD 集成（最终）

- [ ] T099 创建根目录 CI 脚本 `/executor/.github/workflows/test.yml` (如果使用 GitHub Actions)
  - 分层测试执行
  - 快速反馈 (unit < 1min)
  - 完整验证 (all < 5min)

- [ ] T100 [P] 配置覆盖率报告上传 `/executor/scripts/upload-coverage.sh`
  - 生成覆盖率 badge
  - 追踪历史趋势

### 验收和最终验证

- [ ] T101 执行完整测试套件: `make test-all` 在 `/executor/` 全通过
- [ ] T102 [P] 验证所有文档已生成和完整: 检查 `/specs/001-boifi-executor/` 和 `/executor/docs/dev_doc/`
- [ ] T103 最终覆盖率检查: 确认 >= 70% 全局，>= 90% 核心模块

**验收标准 (Phase 8)**:
- ✓ 全局测试覆盖率 >= 70%，核心路径 >= 90%
- ✓ 所有 5 个 User Stories 有 E2E 测试
- ✓ 性能基准已建立并可在 CI 中验证
- ✓ 文档完整，新成员可快速上手
- ✓ CI/CD 能检测性能和覆盖率回归

---

## 任务执行策略

### MVP 快速路径 (2-3 周)
**范围**: Phase 1 + Phase 2 + Phase 3 (US1 only)

```
Week 1:
  - Phase 1: 测试目录 + Makefile (T001-T012) - 2-3 天
  - Phase 2: 测试迁移 + 覆盖率基线 (T013-T030) - 3-4 天

Week 2-3:
  - Phase 3: US1 完整测试 (T031-T044) - 5-6 天
  - 文档和验证 - 2-3 天
```

**MVP 交付物**:
- ✓ 标准化的三层测试结构
- ✓ 基线覆盖率报告
- ✓ SRE 手动混沌测试完整可用
- ✓ 快速启动指南

### 完整实现路径 (6-8 周)
**范围**: 所有 Phase 1-8, 所有 User Stories

```
Weeks 1-2:  Phase 1 + 2 (基础设施) [并行: T001-T030]
Weeks 2-3:  Phase 3 + 4 (US1 + US2) [顺序: 依赖关系]
Weeks 4-5:  Phase 5 (US3 性能) [并行: T054-T063]
Weeks 5-6:  Phase 6 + 7 (US4 + US5) [并行: 独立]
Weeks 6-8:  Phase 8 (完善) [并行: T077-T103]
```

### 并行执行机会

**高度并行的阶段**:
- Phase 1: T001-T012 全部独立，可 6 人同时进行
- Phase 2: T013-T030 70% 可并行
- Phase 5: T054-T063 基准测试全部独立
- Phase 8: T077-T103 85% 可并行

**关键路径** (完整实现):
```
T001 → T010 → T013-T030 → T031-T044 → T045-T053 → T054-T063 → ... → T103
~33 天（周期制约）
```

---

## 任务标签说明

- **[P]**: 该任务可与其他相同 phase 的任务并行执行（不同文件/无依赖）
- **[US1/US2/...]**: 该任务属于特定 User Story 实现阶段

---

## 成功定义

### MVP 成功
- [ ] Phase 3 所有任务完成
- [ ] SRE 能通过 CLI 应用故障策略
- [ ] 请求实时被正确的故障注入
- [ ] 覆盖率基线已建立

### 完整成功
- [ ] 所有 5 个 User Stories 完整实现
- [ ] 全局测试覆盖率 >= 70%，核心 >= 90%
- [ ] 性能基准已验证 (<1ms, <50ms 等)
- [ ] Kubernetes 部署可验证
- [ ] 文档完整，可自动化维护

---

## 附录：任务依赖图

```
Phase 1 (Setup)
    ↓
Phase 2 (Foundational)
    ├→ Phase 3 (US1: Manual Chaos)
    │   ├→ Phase 4 (US2: Lifecycle)
    │   │   ├→ Phase 5 (US3: Performance) ↔ [并行]
    │   │   ├→ Phase 6 (US4: Recommender) ↔ [并行]
    │   │   └→ Phase 7 (US5: K8s) ↔ [并行]
    │   └→ Phase 8 (Polish) [汇聚所有分支]
```

---

**总结**:
- **总任务**: 103 个
- **可并行**: 32 个（31%）
- **关键路径**: ~33 天（完整）
- **MVP 路径**: ~10 天
- **预计总投入**: 60-70 人天（完整），15-20 人天（MVP）