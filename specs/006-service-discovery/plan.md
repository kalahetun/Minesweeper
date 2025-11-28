# Implementation Plan: Service Discovery for BOIFI

**Branch**: `006-service-discovery` | **Date**: 2025-11-28 | **Spec**: `/specs/006-service-discovery/spec.md`
**Input**: Feature specification from `/specs/006-service-discovery/spec.md`

## Summary

Service Discovery 是一个 Go 语言微服务，用于自动周期性探测 Kubernetes 微服务环境并产出结构化的**服务级别服务地图 (Service Map)**。核心功能包括：
1. 通过 Kubernetes API 解析 Istio VirtualService 发现服务和 API 端点
2. 通过 Jaeger Query API 分析 traces 构建服务级调用拓扑（非 API 级别）
3. 将服务地图发布到 Redis 供 BOIFI 其他组件消费

**设计决策**：拓扑构建仅到服务级别，不构建 API 级别的调用图，因为后者复杂度高且对故障注入决策无额外价值。

## Technical Context

**Language/Version**: Go 1.21+  
**Primary Dependencies**: 
- `k8s.io/client-go`: Kubernetes API 客户端
- `istio.io/client-go`: Istio CRD 客户端 (VirtualService)
- `github.com/redis/go-redis/v9`: Redis 客户端
- `net/http` + `encoding/json`: Jaeger HTTP API 和 JSON 处理
- `github.com/spf13/cobra` / `github.com/spf13/viper`: CLI 和配置管理

**Storage**: Redis (服务地图存储和发布通知)  
**Testing**: Go `testing` 标准库 + `testify` 断言库 + `gomock` mock 框架  
**Target Platform**: Linux/Kubernetes (容器化部署)  
**Project Type**: 单体微服务 (single project)  
**Performance Goals**: 
- 单次发现流程 < 30 秒 (100 服务规模)
- 周期性执行，默认间隔 5 分钟
**Constraints**: 
- 容错运行：Jaeger 或 OpenAPI 不可用时不崩溃
- Redis 发布失败时有重试逻辑
**Scale/Scope**: 
- 支持 100+ 服务的 Kubernetes 集群
- 支持处理 1000+ traces 的拓扑构建

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### 宪法对齐度检查

✅ **I. 关注点分离** - 符合
- Service Discovery 是独立的微服务，职责单一（发现和发布）
- 内部模块分离：K8s 发现器、Jaeger 拓扑构建器、Redis 发布器

✅ **II. 声明式配置** - 符合
- 所有外部依赖通过配置文件/环境变量声明
- 输出的 ServiceMap 是结构化的 JSON 格式

✅ **III. 动态性与实时性** - 符合
- 周期性发现确保数据时效性
- Redis Pub/Sub 通知其他组件有更新

✅ **IV. 测试驱动 (强制)** - 必须遵守
- 单元测试覆盖：K8s 解析、Jaeger 解析、Redis 发布
- 集成测试：端到端发现流程
- 目标覆盖率：核心路径 > 90%

✅ **V. 性能优先** - 符合
- 服务级拓扑（非 API 级）降低计算复杂度
- 并行获取 OpenAPI 规范（可选功能）

✅ **VI. 容错与可靠性** - 符合
- Jaeger/OpenAPI 不可用时降级运行
- Redis 连接失败时重试
- 结构化日志记录所有关键操作

✅ **VII. 简洁性** - 符合
- 使用 SSE-free 设计（主动推送到 Redis，非长连接）
- 单一职责，不引入额外复杂性

✅ **VIII. 时间控制** - 符合
- 可配置的发现周期（默认 5 分钟）
- 时间戳记录每次发现时间

**初步评估**: ✅ 通过宪法检查，可进入 Phase 0

## Project Structure

### Documentation (this feature)

```text
specs/006-service-discovery/
├── plan.md              # 本文件
├── research.md          # Phase 0: 技术研究
├── data-model.md        # Phase 1: 数据模型定义
├── quickstart.md        # Phase 1: 快速启动指南
├── contracts/           # Phase 1: API 契约
│   └── service-map-schema.json
└── tasks.md             # Phase 2: 任务分解
```

### Source Code (repository root)

```text
service-discovery/
├── cmd/
│   └── main.go                 # 入口点
├── internal/
│   ├── config/
│   │   └── config.go           # 配置加载
│   ├── discovery/
│   │   ├── kubernetes.go       # K8s/Istio VirtualService 发现
│   │   ├── jaeger.go           # Jaeger trace 拓扑构建
│   │   └── openapi.go          # OpenAPI 规范获取 (可选)
│   ├── publisher/
│   │   └── redis.go            # Redis 发布器
│   ├── scheduler/
│   │   └── ticker.go           # 周期性执行调度器
│   └── types/
│       └── servicemap.go       # ServiceMap 数据结构
├── pkg/
│   └── logger/
│       └── logger.go           # 结构化日志
├── tests/
│   ├── unit/
│   │   ├── kubernetes_test.go
│   │   ├── jaeger_test.go
│   │   └── redis_test.go
│   └── integration/
│       └── discovery_test.go
├── Dockerfile
├── go.mod
├── go.sum
└── Makefile
```

**Structure Decision**: 采用标准 Go 项目布局，`internal/` 包含私有模块，`pkg/` 包含可复用的公共库。

## Complexity Tracking

> 无宪法违反项，无需记录复杂性理由。
