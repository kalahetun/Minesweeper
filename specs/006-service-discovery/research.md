# Research: Service Discovery for BOIFI

**Branch**: `006-service-discovery` | **Date**: 2025-11-28

## 研究概要

本文档记录 Service Discovery 实现前的技术研究，解决所有技术选型和最佳实践问题。

---

## 1. Kubernetes/Istio VirtualService 访问

### 决策
使用 `istio.io/client-go` 库访问 Istio VirtualService CRD。

### 理由
- `istio.io/client-go` 是 Istio 官方提供的 Go 客户端，类型安全
- 提供对 `networking.istio.io/v1beta1` 和 `v1alpha3` 的完整支持
- 与标准 `k8s.io/client-go` 无缝集成

### 备选方案
| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| `istio.io/client-go` | 类型安全，官方支持 | 需要额外依赖 | ✅ 选择 |
| 动态客户端 + Unstructured | 无需 Istio 依赖 | 类型不安全，解析复杂 | ❌ 拒绝 |
| 直接调用 K8s REST API | 最轻量 | 无类型支持，易出错 | ❌ 拒绝 |

### 代码模式
```go
import (
    versionedclient "istio.io/client-go/pkg/clientset/versioned"
    networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
)

// 创建 Istio 客户端
istioClient, err := versionedclient.NewForConfig(kubeConfig)

// 列出所有命名空间的 VirtualService
vsList, err := istioClient.NetworkingV1beta1().VirtualServices("").List(ctx, metav1.ListOptions{})
```

---

## 2. Jaeger Query API 集成

### 决策
使用 Jaeger HTTP JSON API（`/api/traces`），而非 gRPC API。

### 理由
- HTTP API 更简单，无需额外的 protobuf 依赖
- Jaeger Query 默认暴露 HTTP 端口 (16686)
- 对于周期性批量查询场景，HTTP 性能足够

### 备选方案
| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| HTTP JSON API | 简单，无额外依赖 | 性能略低于 gRPC | ✅ 选择 |
| gRPC API | 高性能，类型安全 | 需要 protobuf，复杂 | ❌ 拒绝 |
| Jaeger UI API | 文档最全 | 非官方稳定 API | ❌ 拒绝 |

### API 端点
```
GET /api/services                    # 获取服务列表
GET /api/traces?service={name}&lookback=1h&limit=1000  # 获取 traces
GET /api/dependencies?endTs={ts}&lookback=1h           # 直接获取依赖图（优先使用）
```

### 设计决策：优先使用 Dependencies API
Jaeger 提供了 `/api/dependencies` 端点，直接返回服务间依赖关系，避免手动解析 traces：
```json
[
  {"parent": "service-a", "child": "service-b", "callCount": 150}
]
```
这大大简化了拓扑构建逻辑。如果 Dependencies API 不可用，回退到 traces 解析。

---

## 3. Redis 客户端与发布模式

### 决策
使用 `github.com/redis/go-redis/v9`，采用 SET + PUBLISH 模式。

### 理由
- go-redis 是 Go 生态最流行的 Redis 客户端，维护活跃
- SET 存储完整数据，PUBLISH 发送轻量通知
- 支持连接池、自动重连、管道等高级功能

### 发布模式
```go
// 1. 存储完整数据
err := rdb.Set(ctx, "boifi:service-map", jsonData, 0).Err()

// 2. 发送更新通知（可选）
err := rdb.Publish(ctx, "boifi:service-map:updates", "updated").Err()
```

### 重试策略
- 使用指数退避重试：初始 1s，最大 30s，最多 5 次
- 重试失败后记录错误日志，不崩溃

---

## 4. 配置管理

### 决策
使用 `spf13/viper` 支持多种配置源。

### 配置优先级（从高到低）
1. 命令行参数 (`--jaeger-url`)
2. 环境变量 (`BOIFI_JAEGER_URL`)
3. 配置文件 (`config.yaml`)
4. 默认值

### 配置项
| 配置项 | 环境变量 | 默认值 | 说明 |
|--------|----------|--------|------|
| `kubernetes.kubeconfig` | `KUBECONFIG` | in-cluster | K8s 配置路径 |
| `jaeger.url` | `BOIFI_JAEGER_URL` | `http://jaeger-query:16686` | Jaeger Query 地址 |
| `jaeger.lookback` | `BOIFI_JAEGER_LOOKBACK` | `1h` | 查询时间范围 |
| `redis.address` | `BOIFI_REDIS_ADDR` | `redis:6379` | Redis 地址 |
| `redis.key` | `BOIFI_REDIS_KEY` | `boifi:service-map` | 存储 key |
| `redis.channel` | `BOIFI_REDIS_CHANNEL` | `boifi:service-map:updates` | 通知 channel |
| `discovery.interval` | `BOIFI_DISCOVERY_INTERVAL` | `5m` | 发现周期 |
| `openapi.enabled` | `BOIFI_OPENAPI_ENABLED` | `false` | 是否获取 OpenAPI |
| `openapi.paths` | `BOIFI_OPENAPI_PATHS` | `/swagger.json,/v3/api-docs` | OpenAPI 路径列表 |

---

## 5. 服务级拓扑 vs API 级拓扑

### 决策
**仅构建服务级拓扑**，不构建 API 级别的调用图。

### 理由
1. **复杂度**：API 级拓扑需要解析每个 span 的 HTTP 路径，增加 10 倍以上的数据处理量
2. **存储开销**：API 级拓扑数据量巨大（N 个服务 × M 个 API × K 个调用方）
3. **实用性**：故障注入决策主要基于服务依赖，API 级细节对 Recommender 价值有限
4. **可行性**：Jaeger 的 `/api/dependencies` 直接提供服务级依赖

### 数据模型影响
- `ServiceMap.Topology` 只包含服务间的边 (`ServiceEdge`)
- `ServiceMap.Services` 包含每个服务的 API 端点列表（从 VirtualService 提取）
- 拓扑和 API 端点是正交的，不建立 API→API 的调用关系

---

## 6. 容错与降级策略

### 场景与处理
| 场景 | 处理策略 | 日志级别 |
|------|----------|----------|
| K8s API 不可达 | 重试 3 次后失败，使用上次缓存数据 | ERROR |
| Istio CRD 不存在 | 跳过 VirtualService 发现，服务列表为空 | WARN |
| Jaeger 不可用 | 跳过拓扑构建，拓扑为空数组 | WARN |
| Redis 连接失败 | 指数退避重试 5 次，最终记录错误 | ERROR |
| OpenAPI 获取失败 | 跳过该服务的 OpenAPI 增强 | DEBUG |
| VirtualService 格式异常 | 跳过该资源，继续处理其他 | WARN |

### 缓存机制
- 在内存中保留最后一次成功的 ServiceMap
- 当 K8s/Jaeger 完全不可用时，可选择发布缓存数据（带 `stale: true` 标记）

---

## 7. 结构化日志

### 决策
使用 `log/slog` (Go 1.21+ 标准库) 进行结构化日志。

### 理由
- Go 1.21 原生支持，无外部依赖
- 支持 JSON 输出，便于 ELK/Loki 采集
- 性能优秀，支持上下文传递

### 日志格式
```json
{
  "time": "2025-11-28T10:00:00Z",
  "level": "INFO",
  "msg": "discovery completed",
  "services_count": 15,
  "edges_count": 23,
  "duration_ms": 1250
}
```

---

## 8. 测试策略

### 单元测试
- **K8s 解析测试**：使用 fake clientset 模拟 VirtualService
- **Jaeger 解析测试**：使用 httptest 模拟 Jaeger API 响应
- **Redis 发布测试**：使用 miniredis 内存 Redis

### 集成测试
- 使用 Kind + Istio 部署测试集群
- 使用 Docker Compose 启动 Jaeger + Redis
- 端到端验证完整发现流程

### 覆盖率目标
- 核心路径（解析、发布）：> 90%
- 整体覆盖率：> 70%

---

## 研究结论

所有技术选型已确定，无未解决的 NEEDS CLARIFICATION 项。可进入 Phase 1 设计阶段。

| 研究项 | 决策 | 状态 |
|--------|------|------|
| Istio 客户端 | `istio.io/client-go` | ✅ 已确定 |
| Jaeger API | HTTP JSON + Dependencies API | ✅ 已确定 |
| Redis 客户端 | `go-redis/v9` | ✅ 已确定 |
| 配置管理 | `spf13/viper` | ✅ 已确定 |
| 拓扑粒度 | 服务级（非 API 级） | ✅ 已确定 |
| 日志库 | `log/slog` | ✅ 已确定 |
