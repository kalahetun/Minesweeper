# Feature Specification: Service Discovery for BOIFI

**Feature Branch**: `006-service-discovery`  
**Created**: 2025-11-28  
**Status**: Draft  
**Input**: 实现一个名为 service discovery 的 Go 语言微服务，用于自动周期性探测 Kubernetes 微服务环境并产出结构化的服务地图

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 自动发现 Kubernetes 服务与 API 端点 (Priority: P1)

BOIFI 系统的其他组件（如 Recommender、Request Generator）需要了解目标 Kubernetes 集群中有哪些微服务及其 API 端点，以便进行故障注入测试。Service Discovery 服务会自动连接 Kubernetes API，解析 Istio VirtualService 资源，提取出所有可被注入故障的服务和 API 路径信息。

**Why this priority**: 这是 Service Discovery 的核心价值——没有服务和 API 信息，BOIFI 系统无法知道"在哪里"注入故障。这是所有后续功能的基础。

**Independent Test**: 可以通过部署一个包含 Istio VirtualService 的 Kubernetes 集群，启动 Service Discovery 服务，然后验证其输出的服务列表与集群实际配置是否匹配。

**Acceptance Scenarios**:

1. **Given** Kubernetes 集群中存在多个 Istio VirtualService 资源, **When** Service Discovery 执行发现流程, **Then** 输出的服务地图包含所有 VirtualService 中定义的服务名称
2. **Given** 一个 VirtualService 定义了多条 HTTP 路由规则（包含 prefix、exact、regex 匹配）, **When** Service Discovery 解析该资源, **Then** 每种匹配类型的 API 路径都被正确提取并记录
3. **Given** VirtualService 中某个路由未指定 HTTP 方法, **When** Service Discovery 解析该路由, **Then** 该 API 默认关联所有 HTTP 方法
4. **Given** Kubernetes API Server 连接配置正确, **When** Service Discovery 启动, **Then** 成功建立连接并能列出 VirtualService 资源

---

### User Story 2 - 构建服务级调用拓扑图 (Priority: P1)

系统需要了解微服务之间的实际调用关系，以便 Recommender 能够智能选择故障注入点（例如，注入关键依赖链上的服务会产生更大影响）。Service Discovery 通过查询 Jaeger 分布式追踪数据，分析 trace/span 信息，构建**服务级别**的有向调用图（不构建 API 级别的拓扑，因为过于复杂且不必要）。

**Why this priority**: 服务级拓扑是理解系统架构的关键，它使 BOIFI 能够进行更智能的故障注入决策。服务级别的粒度既足够指导决策，又避免了 API 级拓扑的复杂性。

**Independent Test**: 可以通过向一个已配置 Jaeger 追踪的微服务系统发送请求，生成 traces，然后验证 Service Discovery 输出的拓扑图是否准确反映了服务间的实际调用关系。

**Acceptance Scenarios**:

1. **Given** Jaeger 中存储了包含多个服务调用的 trace 数据, **When** Service Discovery 查询并解析这些 traces, **Then** 输出的拓扑图包含所有实际发生的**服务间**调用关系
2. **Given** 服务 A 调用服务 B 发生了 100 次, **When** Service Discovery 构建拓扑, **Then** A→B 的边包含调用次数统计（count=100）
3. **Given** trace 中存在来自 istio-ingressgateway 的入口流量, **When** Service Discovery 解析该 trace, **Then** 正确处理入口 span 而不产生错误
4. **Given** Jaeger Query API 地址配置正确, **When** Service Discovery 发起查询, **Then** 成功获取指定时间范围内的 trace 数据

---

### User Story 3 - 发布服务地图到共享存储 (Priority: P1)

BOIFI 系统的多个组件需要访问同一份服务地图数据。Service Discovery 将发现结果序列化为 JSON 并发布到 Redis，其他组件可以通过 Redis 读取最新的服务地图，实现数据共享。

**Why this priority**: 发布机制是使服务地图"可用"的关键环节，没有它，发现的数据无法被其他组件消费。

**Independent Test**: 可以运行 Service Discovery 完成一次发现流程，然后从 Redis 读取指定 key，验证存储的 JSON 结构和内容是否正确。

**Acceptance Scenarios**:

1. **Given** Service Discovery 完成一次完整的发现流程, **When** 准备发布结果, **Then** 服务地图被序列化为符合 ServiceMap 结构的 JSON 格式
2. **Given** Redis 服务可用, **When** Service Discovery 执行发布, **Then** JSON 数据成功存储到预定义的 Redis key（如 `boifi:service-map`）
3. **Given** 其他组件订阅了 Redis channel, **When** 新的服务地图发布, **Then** Redis channel 收到更新通知消息
4. **Given** 发布成功, **When** 其他组件读取 Redis key, **Then** 获取到的 JSON 可被正确反序列化为 ServiceMap 结构

---

### User Story 4 - 周期性自动执行发现流程 (Priority: P2)

Kubernetes 集群中的服务和调用关系会随时间变化（新服务部署、路由调整等）。Service Discovery 需要周期性地重新执行发现流程，保持服务地图的时效性，无需人工干预。

**Why this priority**: 自动化周期执行是运维友好性的核心，虽然手动触发也能工作，但周期性执行才能保证服务地图的持续准确。

**Independent Test**: 配置较短的执行周期（如 1 分钟），观察服务日志和 Redis 中的时间戳，验证是否按预期间隔更新。

**Acceptance Scenarios**:

1. **Given** 配置的执行周期为 5 分钟, **When** Service Discovery 启动, **Then** 每隔 5 分钟自动执行一次完整的发现和发布流程
2. **Given** 上一次发现流程尚未完成, **When** 到达下一个周期触发时间, **Then** 等待当前流程完成后再开始新的流程（或跳过本次）
3. **Given** 服务运行中, **When** 检查服务地图的时间戳, **Then** 时间戳与最近一次执行时间一致

---

### User Story 5 - OpenAPI 规范增强 (Priority: P3)

VirtualService 中的路由信息可能比较粗略（如只有 prefix），服务本身暴露的 OpenAPI/Swagger 规范包含更详细的 API 定义（具体路径、参数、请求体等）。Service Discovery 尝试获取每个服务的 OpenAPI 规范来补充 API 信息。

**Why this priority**: 这是一个增强功能，核心发现流程不依赖它。即使 OpenAPI 获取失败，系统仍能使用 VirtualService 数据工作。

**Independent Test**: 部署一个提供 OpenAPI 端点的服务，验证 Service Discovery 能获取并解析其 API 定义，且这些 API 出现在服务地图中。

**Acceptance Scenarios**:

1. **Given** 某服务在 `/swagger.json` 提供 OpenAPI 规范, **When** Service Discovery 尝试获取, **Then** 成功下载并解析出详细的 API 列表
2. **Given** 某服务不提供 OpenAPI 端点, **When** Service Discovery 尝试访问失败, **Then** 记录警告日志并继续使用 VirtualService 中的信息
3. **Given** OpenAPI 规范与 VirtualService 中有重叠的 API 定义, **When** 聚合信息, **Then** OpenAPI 中的详细信息优先级更高

---

### Edge Cases

- 如果 Kubernetes API Server 不可达，服务如何处理？（记录错误，使用上一次缓存的数据或等待重试）
- 如果 Jaeger 服务暂时不可用，服务是否继续运行？（是，记录错误，拓扑信息留空或使用缓存）
- 如果 Redis 连接失败，发布如何处理？（重试若干次，记录错误，服务不崩溃）
- 如果 VirtualService 资源格式不符合预期，如何处理？（跳过该资源，记录警告）
- 如果查询到的 traces 为空，拓扑图应该是什么状态？（空列表，不是错误）
- 如果配置的周期间隔小于发现流程执行时间，如何处理？（跳过本次或排队等待）

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统必须能够连接 Kubernetes API Server 并列出所有命名空间下的 Istio VirtualService 资源
- **FR-002**: 系统必须解析 VirtualService 的 HTTP 路由规则，提取服务名、API 路径（支持 prefix/exact/regex）和 HTTP 方法
- **FR-003**: 系统必须连接 Jaeger Query API 并查询指定时间范围内的 trace 数据
- **FR-004**: 系统必须解析 trace 中的 span 数据，提取服务调用关系并统计调用次数
- **FR-005**: 系统必须将发现结果序列化为 JSON 格式的 ServiceMap 结构
- **FR-006**: 系统必须将 ServiceMap 发布到 Redis 的指定 key
- **FR-007**: 系统应该在发布后向 Redis channel 发送更新通知
- **FR-008**: 系统必须按照配置的时间间隔周期性执行发现流程
- **FR-009**: 系统必须支持通过命令行参数或环境变量配置所有外部依赖地址
- **FR-010**: 系统应该尝试获取服务的 OpenAPI 规范以补充 API 信息
- **FR-011**: 系统必须在关键操作（开始发现、完成发现、发布成功、发生错误）时输出结构化日志
- **FR-012**: 系统必须在某个数据源（OpenAPI、Jaeger）不可用时继续使用其他数据源完成任务
- **FR-013**: 系统必须在 Redis 连接失败时实现重试逻辑

### Key Entities

- **ServiceMap**: 服务地图的顶层结构，包含时间戳、服务列表和拓扑信息
- **APIDetails**: 单个服务的 API 详情，包含该服务暴露的所有 API 端点列表
- **ServiceEdge**: 服务拓扑中的一条边，表示从源服务到目标服务的调用关系及调用次数
- **VirtualService**: Istio 的路由配置资源，是服务和 API 发现的主要数据来源
- **Trace/Span**: Jaeger 中的分布式追踪数据，是构建服务拓扑的主要数据来源

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 服务地图能够在 30 秒内完成一次完整的发现流程（100 个服务规模的集群）
- **SC-002**: 服务发现准确率达到 100%——所有 VirtualService 中定义的服务都被正确识别
- **SC-003**: 拓扑构建覆盖率达到 95%——Jaeger 中存在的服务调用关系能被正确提取
- **SC-004**: 系统能够在数据源部分不可用（如 Jaeger 宕机）时继续运行，不发生崩溃
- **SC-005**: 服务地图更新延迟不超过配置的周期间隔加上 30 秒的执行时间
- **SC-006**: Redis 发布成功率在 Redis 可用时达到 99.9%
- **SC-007**: 其他 BOIFI 组件能够从 Redis 读取并正确解析服务地图数据

## Assumptions

- Kubernetes 集群已部署并配置了 Istio 服务网格
- Istio VirtualService 是定义服务路由的主要方式
- Jaeger 已部署并收集集群中的分布式追踪数据
- Redis 作为 BOIFI 系统的共享数据存储已部署可用
- Service Discovery 服务运行在 Kubernetes 集群内部或具有访问集群 API 的权限
- 服务间使用标准 HTTP/gRPC 协议通信，可被 Jaeger 追踪

## Dependencies

- Kubernetes API Server（必需）
- Istio VirtualService CRD（必需）
- Jaeger Query API（必需，但可降级运行）
- Redis Server（必需）
- 目标服务的 OpenAPI 端点（可选）
