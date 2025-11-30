# Tasks: Service Discovery for BOIFI

**Input**: Design documents from `/specs/006-service-discovery/`  
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅  

**Tests**: 根据宪法 (IV. 测试驱动) 要求，包含单元测试任务。

**Organization**: 任务按用户故事分组，支持独立实现和测试。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 任务所属用户故事（US1, US2, US3, US4, US5）
- 描述中包含确切的文件路径

## Path Conventions

项目结构基于 plan.md:
```
service-discovery/
├── cmd/main.go
├── internal/{config,discovery,publisher,scheduler,types}/
├── pkg/logger/
└── tests/{unit,integration}/
```

---

## Phase 1: Setup (项目初始化) ✅ COMPLETED

**Purpose**: 创建项目结构和基础依赖配置

- [x] T001 创建项目目录结构: `service-discovery/cmd/`, `service-discovery/internal/`, `service-discovery/pkg/`, `service-discovery/tests/`
- [x] T002 初始化 Go module: `service-discovery/go.mod`
- [x] T003 [P] 创建 Makefile 构建脚本: `service-discovery/Makefile`
- [x] T004 [P] 创建 Dockerfile: `service-discovery/Dockerfile`
- [x] T005 [P] 创建示例配置文件: `service-discovery/config.example.yaml`

---

## Phase 2: Foundational (基础设施) ✅ COMPLETED

**Purpose**: 所有用户故事依赖的核心基础设施

**⚠️ CRITICAL**: 此阶段完成前，不能开始任何用户故事

- [x] T006 定义 ServiceMap 数据结构: `service-discovery/internal/types/servicemap.go`
- [x] T007 [P] 定义配置数据结构: `service-discovery/internal/config/types.go`
- [x] T008 [P] 实现配置加载 (viper): `service-discovery/internal/config/config.go`
- [x] T009 [P] 实现结构化日志 (slog): `service-discovery/pkg/logger/logger.go`
- [x] T010 创建主程序入口框架: `service-discovery/cmd/main.go`
- [x] T011 [P] 单元测试: 配置加载测试: `service-discovery/tests/unit/config_test.go`

**Checkpoint**: ✅ 基础设施就绪 - 可以开始用户故事实现

---

## Phase 3: User Story 1 - 自动发现 K8s 服务与 API 端点 (Priority: P1) 🎯 MVP ✅ COMPLETED

**Goal**: 连接 Kubernetes API，解析 Istio VirtualService，提取服务和 API 端点信息

**Independent Test**: 使用 fake K8s clientset 模拟 VirtualService，验证解析输出正确

### Tests for User Story 1

- [x] T012 [P] [US1] 单元测试: VirtualService 解析测试: `service-discovery/tests/unit/kubernetes_test.go`

### Implementation for User Story 1

- [x] T013 [US1] 实现 K8s 客户端初始化: `service-discovery/internal/discovery/kubernetes.go` (NewKubernetesDiscoverer 函数)
- [x] T014 [US1] 实现 VirtualService 列表获取: `service-discovery/internal/discovery/kubernetes.go` (ListVirtualServices 方法)
- [x] T015 [US1] 实现 VirtualService 解析逻辑 (exact/prefix/regex): `service-discovery/internal/discovery/kubernetes.go` (ParseVirtualService 方法)
- [x] T016 [US1] 实现 HTTP 方法提取 (默认为 *): `service-discovery/internal/discovery/kubernetes.go` (extractHTTPMethods 函数)
- [x] T017 [US1] 实现服务信息聚合: `service-discovery/internal/discovery/kubernetes.go` (AggregateServices 方法)
- [x] T018 [US1] 添加 K8s 连接错误处理和日志: `service-discovery/internal/discovery/kubernetes.go`

**Checkpoint**: ✅ US1 完成 - 可以独立测试 K8s 服务发现功能

---

## Phase 4: User Story 2 - 构建服务级调用拓扑图 (Priority: P1)

**Goal**: 通过 Jaeger Dependencies API 构建服务间调用关系图

**Independent Test**: 使用 httptest 模拟 Jaeger API，验证拓扑构建正确

### Tests for User Story 2

- [ ] T019 [P] [US2] 单元测试: Jaeger Dependencies API 解析测试: `service-discovery/tests/unit/jaeger_test.go`

### Implementation for User Story 2

- [ ] T020 [US2] 定义 Jaeger API 响应结构: `service-discovery/internal/discovery/jaeger.go` (JaegerDependency struct)
- [ ] T021 [US2] 实现 Jaeger 客户端初始化: `service-discovery/internal/discovery/jaeger.go` (NewJaegerClient 函数)
- [ ] T022 [US2] 实现 Dependencies API 调用: `service-discovery/internal/discovery/jaeger.go` (FetchDependencies 方法)
- [ ] T023 [US2] 实现依赖数据到 ServiceEdge 的转换: `service-discovery/internal/discovery/jaeger.go` (BuildTopology 方法)
- [ ] T024 [US2] 添加 Jaeger 不可用时的降级处理: `service-discovery/internal/discovery/jaeger.go`
- [ ] T025 [US2] 添加 Jaeger 连接错误处理和日志: `service-discovery/internal/discovery/jaeger.go`

**Checkpoint**: US2 完成 - 可以独立测试 Jaeger 拓扑构建功能

---

## Phase 5: User Story 3 - 发布服务地图到共享存储 (Priority: P1)

**Goal**: 将 ServiceMap 序列化为 JSON 并发布到 Redis

**Independent Test**: 使用 miniredis 内存 Redis，验证 SET 和 PUBLISH 操作正确

### Tests for User Story 3

- [ ] T026 [P] [US3] 单元测试: Redis 发布测试: `service-discovery/tests/unit/redis_test.go`

### Implementation for User Story 3

- [ ] T027 [US3] 实现 Redis 客户端初始化: `service-discovery/internal/publisher/redis.go` (NewRedisPublisher 函数)
- [ ] T028 [US3] 实现 ServiceMap JSON 序列化: `service-discovery/internal/publisher/redis.go` (SerializeServiceMap 方法)
- [ ] T029 [US3] 实现 Redis SET 操作: `service-discovery/internal/publisher/redis.go` (PublishServiceMap 方法)
- [ ] T030 [US3] 实现 Redis PUBLISH 通知: `service-discovery/internal/publisher/redis.go` (NotifyUpdate 方法)
- [ ] T031 [US3] 实现指数退避重试逻辑: `service-discovery/internal/publisher/redis.go` (retryWithBackoff 函数)
- [ ] T032 [US3] 添加 Redis 错误处理和日志: `service-discovery/internal/publisher/redis.go`

**Checkpoint**: US3 完成 - 可以独立测试 Redis 发布功能

---

## Phase 6: User Story 4 - 周期性自动执行发现流程 (Priority: P2)

**Goal**: 实现定时器控制的周期性发现和发布流程

**Independent Test**: 设置短周期（如 1 秒），验证多次执行和时间戳更新

### Tests for User Story 4

- [ ] T033 [P] [US4] 单元测试: 调度器测试: `service-discovery/tests/unit/scheduler_test.go`

### Implementation for User Story 4

- [ ] T034 [US4] 实现调度器结构: `service-discovery/internal/scheduler/ticker.go` (Scheduler struct)
- [ ] T035 [US4] 实现发现流程编排: `service-discovery/internal/scheduler/ticker.go` (RunDiscovery 方法 - 调用 K8s + Jaeger + Redis)
- [ ] T036 [US4] 实现周期性 Ticker: `service-discovery/internal/scheduler/ticker.go` (Start 方法)
- [ ] T037 [US4] 实现优雅停止: `service-discovery/internal/scheduler/ticker.go` (Stop 方法)
- [ ] T038 [US4] 实现任务重叠防护 (跳过或等待): `service-discovery/internal/scheduler/ticker.go`
- [ ] T039 [US4] 实现内存缓存 (上次成功的 ServiceMap): `service-discovery/internal/scheduler/ticker.go` (lastSuccessfulMap 字段)
- [ ] T040 [US4] 完善主程序: 集成调度器和信号处理: `service-discovery/cmd/main.go`

**Checkpoint**: US4 完成 - 可以独立测试周期性执行功能

---

## Phase 7: User Story 5 - OpenAPI 规范增强 (Priority: P3)

**Goal**: 尝试获取服务的 OpenAPI 规范补充 API 信息

**Independent Test**: 使用 httptest 模拟 OpenAPI 端点，验证 API 信息被正确合并

### Tests for User Story 5

- [ ] T041 [P] [US5] 单元测试: OpenAPI 获取和解析测试: `service-discovery/tests/unit/openapi_test.go`

### Implementation for User Story 5

- [ ] T042 [US5] 实现 OpenAPI 获取器结构: `service-discovery/internal/discovery/openapi.go` (OpenAPIFetcher struct)
- [ ] T043 [US5] 实现 OpenAPI 端点探测: `service-discovery/internal/discovery/openapi.go` (FetchOpenAPI 方法 - 尝试多个路径)
- [ ] T044 [US5] 实现 OpenAPI JSON 解析: `service-discovery/internal/discovery/openapi.go` (ParseOpenAPISpec 方法)
- [ ] T045 [US5] 实现 API 信息合并 (OpenAPI 优先): `service-discovery/internal/discovery/openapi.go` (MergeAPIs 方法)
- [ ] T046 [US5] 添加 OpenAPI 获取失败的降级处理 (仅 DEBUG 日志): `service-discovery/internal/discovery/openapi.go`
- [ ] T047 [US5] 集成 OpenAPI 增强到调度器流程: `service-discovery/internal/scheduler/ticker.go` (修改 RunDiscovery)

**Checkpoint**: US5 完成 - 可以独立测试 OpenAPI 增强功能

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: 集成测试、文档和最终优化

- [ ] T048 [P] 集成测试: 端到端发现流程: `service-discovery/tests/integration/discovery_test.go`
- [ ] T049 [P] 添加 --once 参数支持单次执行: `service-discovery/cmd/main.go`
- [ ] T050 [P] 创建 README.md: `service-discovery/README.md`
- [ ] T051 [P] 创建 Kubernetes 部署清单: `service-discovery/k8s/deployment.yaml`
- [ ] T052 运行并验证 quickstart.md 中的所有步骤
- [ ] T053 代码审查和清理

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1 (Setup)
    │
    ▼
Phase 2 (Foundational) ──── BLOCKS ALL USER STORIES
    │
    ├─────────┬─────────┬─────────┬─────────┐
    ▼         ▼         ▼         ▼         ▼
Phase 3    Phase 4    Phase 5    Phase 6    Phase 7
(US1-P1)   (US2-P1)   (US3-P1)   (US4-P2)   (US5-P3)
    │         │         │         │         │
    └─────────┴─────────┴─────────┴─────────┘
                        │
                        ▼
                  Phase 8 (Polish)
```

### User Story Dependencies

| User Story | 依赖 | 可并行 |
|------------|------|--------|
| US1 (K8s Discovery) | Phase 2 完成 | ✅ 独立 |
| US2 (Jaeger Topology) | Phase 2 完成 | ✅ 独立 |
| US3 (Redis Publish) | Phase 2 完成 | ✅ 独立 |
| US4 (Scheduler) | US1 + US2 + US3 完成 | ❌ 需要前三者 |
| US5 (OpenAPI) | US1 完成 | ⚠️ 依赖 K8s 发现 |

### Within Each User Story

1. 测试先行 → 模型/结构 → 核心逻辑 → 错误处理 → 日志

### Parallel Opportunities

**Phase 2 内部并行**:
- T007, T008, T009, T011 可同时执行

**US1-US3 完全并行** (不同模块):
- T012-T018 (K8s)
- T019-T025 (Jaeger)  
- T026-T032 (Redis)

**Phase 8 内部并行**:
- T048, T049, T050, T051 可同时执行

---

## Parallel Example: Phase 2 + User Stories 1-3

```bash
# Phase 2 并行任务:
Task T007: "定义配置数据结构 in internal/config/types.go"
Task T008: "实现配置加载 in internal/config/config.go"
Task T009: "实现结构化日志 in pkg/logger/logger.go"

# US1-US3 并行 (Phase 2 完成后):
# 开发者 A - US1 K8s:
Task T012: "[US1] 单元测试: kubernetes_test.go"
Task T013-T018: "[US1] K8s 发现实现"

# 开发者 B - US2 Jaeger:
Task T019: "[US2] 单元测试: jaeger_test.go"
Task T020-T025: "[US2] Jaeger 拓扑构建实现"

# 开发者 C - US3 Redis:
Task T026: "[US3] 单元测试: redis_test.go"
Task T027-T032: "[US3] Redis 发布实现"
```

---

## Implementation Strategy

### MVP First (User Story 1-3)

1. ✅ Phase 1: Setup (T001-T005)
2. ✅ Phase 2: Foundational (T006-T011)
3. ✅ Phase 3: US1 - K8s Discovery (T012-T018)
4. ✅ Phase 4: US2 - Jaeger Topology (T019-T025)
5. ✅ Phase 5: US3 - Redis Publish (T026-T032)
6. **STOP**: 测试核心功能，可手动运行一次发现流程

### Incremental Delivery

| 阶段 | 交付物 | 价值 |
|------|--------|------|
| Setup + Foundational | 项目骨架 | 可编译运行 |
| + US1 | K8s 服务发现 | 可发现服务和 API |
| + US2 | Jaeger 拓扑 | 可构建调用图 |
| + US3 | Redis 发布 | 其他组件可消费数据 |
| + US4 | 周期执行 | 自动化运行 |
| + US5 | OpenAPI 增强 | 更详细的 API 信息 |

### Suggested MVP Scope

**推荐 MVP**: Phase 1-5 (Setup + Foundational + US1 + US2 + US3)
- 可手动触发一次完整的发现和发布流程
- 核心价值已交付：服务发现 + 拓扑构建 + Redis 发布

---

## Notes

- [P] 标记 = 不同文件，无依赖，可并行
- [US*] 标记 = 任务归属的用户故事，便于追踪
- 每个用户故事应可独立完成和测试
- 测试必须在实现前编写并确保失败
- 每个任务或逻辑组完成后提交代码
- 任何 Checkpoint 处都可停止验证功能
- 避免：模糊任务、同文件冲突、破坏独立性的跨故事依赖
