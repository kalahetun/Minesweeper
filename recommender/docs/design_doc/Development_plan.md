# 项目开发计划：BOIFI - 贝叶斯优化推荐器

项目代号: `Project BO` (Bayesian Optimizer - 推荐系统)
核心团队: 1-2 名工程师
技术栈: Python 3.8+, FastAPI, scikit-optimize, Pydantic, Docker, Kubernetes

---

## 概述与架构

BOIFI 推荐器是一个独立的智能决策系统，负责：
1. **智能提议** (Propose): 基于历史实验数据，使用贝叶斯优化算法提出下一个最有潜力的故障注入组合
2. **观测记录** (Record): 接收实验结果的严重性评分，更新模型，迭代优化
3. **流程编排** (Coordinate): 管理完整的优化会话生命周期，与 HFI 执行器无缝协作
4. **响应分析** (Analyze): 将执行器返回的原始观测数据转化为标准化的严重性评分

系统由三个核心模块组成：
- **Coordinator Service**: 任务生命周期管理与流程编排
- **Optimizer Core**: 贝叶斯优化算法实现
- **Response Analyzer**: 多维度评分与领域知识编码

---

## 第一阶段：环境与接口设计 (MVP-Setup) - "搭建骨架"

**目标**: 搭建 Python 开发环境，定义核心数据结构和接口，建立与执行器的通信协议。  
**预计时间**: 1.5 周

### 阶段任务分解

| 任务 ID | 任务模块 | 任务描述 | 关键技术点/产出 | 依赖任务 |
|:--|:--|:--|:--|:--|
| P-1 | 项目基础 | 初始化 Python 项目结构，配置 `pyproject.toml`, `requirements.txt`, 搭建 FastAPI 开发环境 | Poetry / pip, FastAPI, Uvicorn | - |
| P-2 | 数据模型 | 定义核心数据结构：`FaultPlan`, `RawObservation`, `SeverityScore`, `SessionState`, `OptimizationProgress` | Pydantic dataclass, Enum | P-1 |
| P-3 | 执行器客户端 | 实现 `ExecutorClient`，支持与 HFI Control Plane 的通信 | `httpx` 或 `requests`, 异步 HTTP 客户端 | P-1 |
| P-4 | API 服务框架 | 使用 FastAPI 搭建 REST API 骨架，定义 `/v1/optimization/sessions` 等端点 | FastAPI, Pydantic models | P-2, P-3 |
| P-5 | 会话管理器框架 | 实现 `SessionManager` 的基本骨架，支持会话的创建、查询、停止 | 线程安全字典、Lock 机制 | P-4 |
| T-1 | 集成测试环境 | 搭建 Docker Compose 环境：包含 Mock Executor 和 Recommender | Docker, docker-compose | P-4, P-5 |

### 第一阶段可交付成果

```
recommender/
├── pyproject.toml                          # Python 项目配置
├── requirements.txt                        # 依赖列表
├── Dockerfile                              # 推荐器容器镜像
├── docker-compose-dev.yaml                 # 开发环境编排
├── src/
│   └── boifi_recommender/
│       ├── __init__.py
│       ├── main.py                         # FastAPI 应用入口
│       ├── models/
│       │   ├── __init__.py
│       │   ├── fault_plan.py               # 故障计划数据模型
│       │   ├── observation.py              # 观测数据模型
│       │   ├── session.py                  # 会话状态模型
│       │   └── api_models.py               # Pydantic API 模型
│       ├── clients/
│       │   ├── __init__.py
│       │   └── executor_client.py          # 执行器客户端
│       ├── api/
│       │   ├── __init__.py
│       │   └── routes.py                   # API 路由定义
│       ├── services/
│       │   ├── __init__.py
│       │   └── session_manager.py          # 会话管理器
│       └── utils/
│           ├── __init__.py
│           └── logger.py                   # 日志工具
├── tests/
│   ├── __init__.py
│   ├── test_models.py                      # 数据模型测试
│   └── test_executor_client.py             # 客户端测试
└── README.md
```

---

## 第二阶段：核心优化器实现 (MVP-Optimizer) - "让它算"

**目标**: 实现贝叶斯优化的核心算法，包括搜索空间定义、代理模型训练和采集函数。  
**预计时间**: 2.5 周

### 阶段任务分解

| 任务 ID | 任务模块 | 任务描述 | 关键技术点/产出 | 依赖任务 |
|:--|:--|:--|:--|:--|
| O-1 | 搜索空间 | 实现 `SpaceConverter`: 将 YAML/JSON 配置转换为 scikit-optimize Dimension 对象 | scikit-optimize, 维度编码 | P-2, P-5 |
| O-2 | 空间验证 | 实现搜索空间的验证逻辑：维度唯一性、类型检查、条件关系验证 | 配置验证、错误处理 | O-1 |
| O-3 | 代理模型 | 实现 `ProxyModel`：基于随机森林的 surrogate model，支持预测和不确定性估计 | scikit-optimize (RandomForest), 模型训练 | O-1 |
| O-4 | 采集函数 | 实现 `AcquisitionFunction`: 基于 Expected Improvement (EI) 或 Upper Confidence Bound (UCB) | scikit-optimize 中的采集函数 | O-3 |
| O-5 | 优化器核心 | 实现 `OptimizerCore` 类：提供 `propose()` 和 `record()` 接口，管理观测历史 | 贝叶斯优化主逻辑、点选择策略 | O-2, O-3, O-4 |
| O-6 | 点选择策略 | 实现从 Pareto 前沿选择最优点的策略 (如：max severity score) | 多目标优化、Pareto 分析 | O-5 |
| T-2 | 单元测试 | 编写 `OptimizerCore` 的详尽单元测试，覆盖所有策略分支 | pytest, mock 库 | O-5, O-6 |

### 第二阶段核心代码结构

```
src/boifi_recommender/
├── optimizer/
│   ├── __init__.py
│   ├── core.py                             # OptimizerCore 类
│   ├── space_converter.py                  # SpaceConverter
│   ├── proxy_model.py                      # ProxyModel (RandomForest)
│   ├── acquisition.py                      # AcquisitionFunction
│   ├── point_selector.py                   # 点选择策略
│   └── validation.py                       # 空间验证逻辑
├── models/
│   ├── optimization.py                     # 优化相关的数据模型
│   └── space_config.py                     # 搜索空间配置模型
```

### 第二阶段性能指标

| 指标 | 目标值 |
|:--|:--|
| 单次 propose() 延迟 | < 50ms (冷启动) / < 20ms (热启动) |
| 单次 record() 与模型重训练 | < 200ms |
| 支持的搜索空间维度 | <= 20 |
| 模型精度 (R² 分数) | >= 0.85 (50 次迭代后) |

---

## 第三阶段：响应分析器 (MVP-Analyzer) - "让它评"

**目标**: 实现多维度评分系统，将原始观测数据转化为标准化严重性评分。  
**预计时间**: 2 周

### 阶段任务分解

| 任务 ID | 任务模块 | 任务描述 | 关键技术点/产出 | 依赖任务 |
|:--|:--|:--|:--|:--|
| A-1 | Bug 评分器 | 实现 `BugScorer`：基于 HTTP 状态码、日志、错误率的评分逻辑 | 规则引擎、优先级匹配 | P-2 |
| A-2 | 性能评分器 | 实现 `PerformanceScorer`：基于延迟指标的非线性评分函数 | 数学函数（分段函数、插值） | P-2 |
| A-3 | 结构分析器 | 实现 `StructureScorer`：通过 Trace 比对检测异常模式 | Trace 解析、编辑距离、Span 分析 | P-2 |
| A-4 | 分析器服务 | 实现 `AnalyzerService`：整合三个 Scorer，计算加权聚合评分 | 加权平均、Fail-Safe 原则 | A-1, A-2, A-3 |
| A-5 | 配置管理 | 实现 `AnalyzerConfig`：管理权重、阈值等可配置参数 | 配置文件加载、验证 | A-4 |
| T-3 | 单元与集成测试 | 编写 Analyzer 的详尽测试，覆盖 3 个维度的多种场景 | pytest, fixture, mock | A-1 到 A-5 |

### 第三阶段核心代码结构

```
src/boifi_recommender/
├── analyzer/
│   ├── __init__.py
│   ├── service.py                          # AnalyzerService 主类
│   ├── scorers/
│   │   ├── __init__.py
│   │   ├── base.py                         # Scorer 基类接口
│   │   ├── bug_scorer.py                   # BugScorer
│   │   ├── performance_scorer.py           # PerformanceScorer
│   │   └── structure_scorer.py             # StructureScorer
│   ├── trace_analyzer.py                   # Trace 分析逻辑
│   └── config.py                           # AnalyzerConfig
├── models/
│   ├── analyzer.py                         # 分析相关的数据模型
│   └── scoring.py                          # 评分结果模型
```

### 第三阶段性能指标

| 指标 | 目标值 |
|:--|:--|
| 单次评分 (calculate_severity) 延迟 | < 100ms (包含 Trace 解析) |
| 三个 Scorer 的覆盖范围 | Bug, 性能, 结构 各一个 |
| 评分结果范围 | [0.0, 10.0] 标准化 |
| Trace 分析覆盖 | Span 数变化、编辑距离、ERROR 检测、性能瓶颈 |

---

## 第四阶段：协调器与主循环 (MVP-Coordinator) - "让它跑"

**目标**: 实现完整的优化会话管理和主循环，支持端到端的闭环优化过程。  
**预计时间**: 2 周

### 阶段任务分解

| 任务 ID | 任务模块 | 任务描述 | 关键技术点/产出 | 依赖任务 |
|:--|:--|:--|:--|:--|
| C-1 | 优化工作者 | 实现 `OptimizationWorker`：主循环逻辑 (propose -> execute -> analyze -> record) | 异步编程、错误重试机制 | O-5, A-4, P-3 |
| C-2 | 工作者线程管理 | 完善 `SessionManager`：支持为每个会话启动独立 Worker 线程 | 线程池、生命周期管理 | P-5, C-1 |
| C-3 | 优雅停止机制 | 实现会话的优雅停止：完成当前实验后停止，保存最终结果 | Context 机制、信号处理 | C-1, C-2 |
| C-4 | 结果持久化 | 实现会话结果的保存与加载：JSON/YAML 格式 | 序列化、文件 I/O | C-1, C-2 |
| C-5 | API 完整化 | 完成所有 REST API 端点：POST /sessions, GET /sessions/{id}, POST /sessions/{id}/stop 等 | FastAPI, 状态序列化 | P-4, C-2, C-4 |
| C-6 | 错误处理与恢复 | 实现整个循环的容错机制：缺失数据、网络错误、超时等 | Try-except, Fallback 策略, Fail-Safe | C-1 到 C-5 |
| T-4 | 端到端测试 | 在完整 Docker 环境中进行端到端测试：启动会话、执行多个迭代、验证结果 | Integration test, mock executor | C-1 到 C-6 |

### 第四阶段核心代码结构

```
src/boifi_recommender/
├── coordinator/
│   ├── __init__.py
│   ├── worker.py                           # OptimizationWorker
│   ├── main_loop.py                        # 主循环逻辑
│   └── error_handler.py                    # 错误处理与重试
├── services/
│   ├── __init__.py
│   ├── session_manager.py                  # SessionManager (完整版)
│   └── persistence.py                      # 结果持久化
├── api/
│   ├── __init__.py
│   ├── routes.py                           # API 路由 (完整版)
│   └── middleware.py                       # CORS, 日志等中间件
```

### 第四阶段关键流程图

```
优化会话主循环：

┌─────────────────────────────────────────────────────┐
│ 1. 用户通过 API 启动优化会话                         │
│    POST /v1/optimization/sessions                   │
│    {search_space_config, total_trials, ...}         │
└────────────┬────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────┐
│ 2. SessionManager 创建会话，启动 OptimizationWorker │
└────────────┬────────────────────────────────────────┘
             │
             ▼
┌────────────────────────────────────────────────────┐
│ 3. Worker 启动主循环 (for trial in range(trials): │
│                                                   │
│    a) OptimizerCore.propose()                     │
│       → 返回下一个故障组合 x_next               │
│                                                   │
│    b) ExecutorClient.apply_policy(x_next)        │
│       → 在 HFI 执行器上执行故障注入             │
│                                                   │
│    c) ExecutorClient.get_observation()           │
│       → 获取执行结果（状态码、延迟、Trace）    │
│                                                   │
│    d) AnalyzerService.calculate_severity(obs)    │
│       → 计算严重性评分 y_next                   │
│                                                   │
│    e) OptimizerCore.record((x_next, y_next))     │
│       → 更新模型，为下一次迭代做准备            │
│                                                   │
│    f) SessionManager.update_progress()           │
│       → 更新迭代次数、最佳结果等               │
│ )                                                │
└────────────┬────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────┐
│ 4. 主循环结束，Worker 停止                          │
│    SessionManager.finalize_session()                │
│    保存最终结果和分析报告                          │
└────────────┬────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────┐
│ 5. 用户查询结果                                     │
│    GET /v1/optimization/sessions/{session_id}     │
│    返回 BestResult, OptimizationProgress 等        │
└─────────────────────────────────────────────────────┘
```

### 第四阶段性能指标

| 指标 | 目标值 |
|:--|:--|
| 单次优化循环总延迟 | < 600ms (propose 20ms + execute 500ms + analyze 50ms + record 30ms) |
| 并发会话数 | >= 10 (单节点) |
| 会话存活时间 | 可配置，默认 24 小时 |
| API 响应延迟 (非主循环) | < 100ms |

---

## 第五阶段：部署与优化 (MVP-Deployment) - "让它稳"

**目标**: 完成部署配置、性能优化、监控和文档。  
**预计时间**: 1.5 周

### 阶段任务分解

| 任务 ID | 任务模块 | 任务描述 | 关键技术点/产出 | 依赖任务 |
|:--|:--|:--|:--|:--|
| D-1 | 容器化 | 完善 Dockerfile，优化镜像大小，配置多阶段构建 | Docker best practices, 轻量级基础镜像 | P-1 |
| D-2 | Kubernetes 配置 | 编写 K8s 部署清单：Deployment, Service, ConfigMap, 资源限制 | K8s YAML, 资源管理 | D-1 |
| D-3 | 监控与日志 | 集成 Prometheus metrics 和 ELK 日志聚合 | prometheus_client, 结构化日志 | P-4 |
| D-4 | 性能优化 | 优化 scikit-optimize 的参数，减少模型训练时间 | 并行计算、缓存 | O-5, A-4 |
| D-5 | 安全加固 | 实现 API 认证、请求速率限制、输入验证 | JWT/OAuth2, 速率限制器 | C-5 |
| D-6 | 文档完善 | 编写 API 文档、部署指南、使用示例、故障排查指南 | Swagger/OpenAPI, markdown | 所有阶段 |
| D-7 | 集成测试与验证 | 在完整的 K8s 环境中进行压力测试、可靠性测试 | locust, k6, chaos engineering | D-1 到 D-6 |

### 第五阶段可交付成果

```
recommender/
├── Dockerfile                              # 生产级 Dockerfile
├── docker-compose-prod.yaml                # 生产环境编排
├── k8s/
│   ├── namespace.yaml
│   ├── deployment.yaml                     # Recommender Deployment
│   ├── service.yaml                        # Service 定义
│   ├── configmap.yaml                      # 配置 ConfigMap
│   ├── hpa.yaml                            # 水平自动扩展
│   └── monitoring.yaml                     # Prometheus ServiceMonitor
├── docs/
│   ├── API_REFERENCE.md                    # API 完整文档
│   ├── DEPLOYMENT.md                       # 部署指南
│   ├── TROUBLESHOOTING.md                  # 故障排查
│   ├── PERFORMANCE.md                      # 性能优化指南
│   └── USAGE_EXAMPLES.md                   # 使用示例
├── config/
│   ├── default.yaml                        # 默认配置
│   └── production.yaml                     # 生产配置
└── tests/
    ├── test_integration.py                 # 集成测试
    └── test_stress.py                      # 压力测试 (可选)
```

### 第五阶段性能指标

| 指标 | 目标值 |
|:--|:--|
| 容器启动时间 | < 5s |
| 内存使用 | 单会话 < 200MB, 10 并发会话 < 2GB |
| CPU 使用 | 单会话空闲时 < 5% CPU, 活跃时 < 50% |
| API 可用性 | >= 99.9% |

---

## 跨阶段的关键设计原则

### 1. **模块化与职责单一**
- 每个模块只负责一个清晰的业务能力（如 Optimizer 只负责算法，不关心执行细节）
- 使用接口定义模块边界，方便后续替换或扩展（如更换贝叶斯优化库）

### 2. **Fail-Safe 设计**
- 任何一个子模块的失败（如 Trace 解析异常）都不应导致整个循环崩溃
- 使用默认值、日志记录、继续执行的策略

### 3. **可配置性与可扩展性**
- 所有"魔法数字"（权重、阈值、参数）都提取到配置文件
- 新增评分维度、采集函数、点选择策略应该只需实现接口

### 4. **强类型与验证**
- 使用 Pydantic 进行数据验证，捕获配置错误于早期
- 利用 Python 类型提示，提升代码可读性和可维护性

### 5. **异步与并发**
- 使用异步 HTTP 客户端，避免阻塞
- 支持多个会话并发运行，相互独立

### 6. **可观测性**
- 结构化日志、指标导出
- 支持会话进度查询、结果追踪

---

## 开发流程与最佳实践

### 开发顺序建议
1. **优先级**: P-1 → P-2 → P-3 → P-4 → P-5 (基础完成)
2. **优先级**: O-1 → O-2 → O-3 → O-4 → O-5 (算法完成)
3. **优先级**: A-1 → A-2 → A-3 → A-4 (分析器完成)
4. **优先级**: C-1 → C-2 → C-3 → C-5 (协调器完成)
5. **并行**: T-1, T-2, T-3, T-4 (测试贯穿全过程)
6. **最后**: D-* (部署优化)

### 代码审查重点
- 数据模型是否准确反映业务逻辑
- 接口定义是否清晰、易于扩展
- 错误处理是否完整、日志是否充分
- 测试覆盖率是否达到 >= 80%

### 关键评审检查点
- **第一阶段末**: 数据模型和 API 接口定义是否清晰？
- **第二阶段末**: 优化器核心是否通过单元测试？ propose/record 接口是否稳定？
- **第三阶段末**: 分析器是否能准确评分？ 三个 Scorer 的逻辑是否正确？
- **第四阶段末**: 主循环是否能完整运行 N 次迭代？ 结果是否符合预期？
- **第五阶段末**: 系统是否能在 K8s 中稳定运行？ 是否支持并发会话？

---

## 时间规划总结

| 阶段 | 预计时间 | 关键产出 |
|:--|:--|:--|
| **第一阶段**: 环境与接口 | 1.5 周 | 项目骨架、数据模型、API 框架 |
| **第二阶段**: 核心优化器 | 2.5 周 | Bayesian Optimizer 实现、单元测试 |
| **第三阶段**: 响应分析器 | 2 周 | 多维度评分系统、集成测试 |
| **第四阶段**: 协调器与主循环 | 2 周 | 完整的优化闭环、端到端测试 |
| **第五阶段**: 部署与优化 | 1.5 周 | K8s 配置、文档、性能优化 |
| **总计** | **9.5 周** | 生产就绪的 BOIFI 推荐器 |

---

## 常见风险与缓解策略

### 风险 1: scikit-optimize 库的学习曲线
- **缓解**: 在第二阶段早期花时间研究库的 API，编写 demo
- **备选**: 如有必要，考虑使用 Optuna (更现代) 或自实现简单的 BO

### 风险 2: Trace 分析逻辑复杂
- **缓解**: 在第三阶段前，与团队讨论 Trace 格式，编写详细规范
- **备选**: 第一版可简化，只检测 Span 数变化，后续迭代添加更复杂的分析

### 风险 3: 并发会话管理的线程安全问题
- **缓解**: 在第四阶段早期进行并发测试，使用 Lock 保护共享状态
- **备选**: 考虑使用消息队列 (RabbitMQ/Kafka) 解耦会话管理

### 风险 4: 与执行器的网络通信不稳定
- **缓解**: 实现重试机制、超时机制、幂等性检查
- **备选**: 实现本地缓存，支持离线模式

---

## 阶段交付与验收

### 第一阶段验收标准
- [ ] 项目能在本地环境运行，无依赖错误
- [ ] 所有核心数据模型已定义，通过 Pydantic 验证
- [ ] ExecutorClient 能成功连接到 mock executor
- [ ] API 框架已搭建，至少 2 个端点可用
- [ ] SessionManager 能创建和查询会话

### 第二阶段验收标准
- [ ] OptimizerCore 的 propose/record 接口已实现
- [ ] SpaceConverter 能正确转换配置
- [ ] 单元测试覆盖率 >= 80%
- [ ] 性能指标：propose() < 50ms, record() < 200ms

### 第三阶段验收标准
- [ ] BugScorer, PerformanceScorer, StructureScorer 各已实现
- [ ] AnalyzerService 能完整计算评分
- [ ] 单元测试覆盖率 >= 80%
- [ ] 评分结果在 [0, 10] 范围内

### 第四阶段验收标准
- [ ] 主循环能完整运行，完成 N 次迭代
- [ ] 支持至少 3 个并发会话
- [ ] 支持优雅停止
- [ ] 端到端测试通过
- [ ] 错误处理完整（网络错误、超时、数据缺失）

### 第五阶段验收标准
- [ ] Docker 镜像能成功构建和运行
- [ ] K8s 配置能在真实集群上部署
- [ ] 监控指标导出正常
- [ ] 文档完整，包含 API、部署、故障排查
- [ ] 压力测试通过，性能指标达标

---

## 详细开发 Prompt 与指导

本部分为每个阶段的任务提供详细的 Prompt，用于指导 AI 代码生成或人工开发。

### 使用说明

1. **分步进行**: 严格按照任务 ID 的顺序，一次提出一个 Prompt。不要试图一次性完成所有任务。
2. **提供上下文**: 在开始第一个 Prompt (P-1) 之前，可以先给 AI 一个总体上下文，这将极大地提升后续生成代码的连贯性。
3. **代码审查**: AI 生成的代码是"草稿"，需要仔细审查、理解、重构并测试。
4. **增量开发**: 完成一个任务后，再进行下一个任务，确保依赖关系得到满足。

---

### 总体上下文 Prompt (在开始 P-1 之前)

> [System]
> 我正在开发一个名为 `BOIFI` (Bayesian Optimizer for Intelligent Fault Injection) 的智能韧性测试推荐系统。
> 
> **核心思想**:
> 1. 用户通过 REST API 启动一个"优化会话"，指定故障注入的搜索空间（如延迟范围、Abort 百分比等）。
> 2. 推荐器使用贝叶斯优化算法，基于历史实验数据，智能地提议下一个最有潜力的故障注入组合。
> 3. 推荐器与 HFI (Holistic Fault Injector) 执行器通信，执行故障注入。
> 4. 推荐器从执行器获取原始观测数据（延迟、HTTP 状态码、分布式 Trace），并通过多维度评分算法转化为严重性评分。
> 5. 推荐器将这个评分反馈给贝叶斯优化模型，迭代优化。
>
> **系统架构** (三个核心模块):
> - **Coordinator Service**: REST API + 会话生命周期管理
> - **Optimizer Core**: 贝叶斯优化算法 (基于 scikit-optimize)
> - **Response Analyzer**: 多维度评分系统 (Bug, 性能, 结构)
>
> **技术栈**: Python 3.8+, FastAPI, scikit-optimize, Pydantic, Docker, Kubernetes
>
> 接下来，我将分步请求你为我生成各个阶段的代码。请确保代码风格符合 Python 最佳实践，类型提示完整，文档清晰。

---

### 第一阶段 Prompt

#### 任务 P-1: 项目基础 - 初始化 Python 项目结构

> [User]
> 任务 P-1: 初始化 BOIFI 推荐器的 Python 项目结构和开发环境。
>
> 要求:
> 1. 创建一个标准的 Python 项目结构，使用 Poetry 或 pip + setup.py。
> 2. 创建 `pyproject.toml` 文件，包含:
>    - 项目基本信息: name = "boifi-recommender", version = "0.1.0"
>    - 核心依赖: FastAPI, Uvicorn, scikit-optimize, Pydantic, httpx, pyyaml
>    - 开发依赖: pytest, pytest-asyncio, black, flake8, mypy
> 3. 创建 `requirements.txt` 作为备选，列出所有依赖。
> 4. 创建项目目录结构:
>    ```
>    recommender/
>    ├── pyproject.toml
>    ├── requirements.txt
>    ├── README.md
>    ├── src/
>    │   └── boifi_recommender/
>    │       └── __init__.py
>    ├── tests/
>    │   └── __init__.py
>    └── docker/
>        └── Dockerfile (简单骨架)
>    ```
> 5. 初始化 Git 仓库，创建 `.gitignore`，排除 `__pycache__`, `.pytest_cache`, `*.pyc` 等。
> 6. 创建一个 `src/boifi_recommender/main.py` 文件，包含一个简单的 FastAPI "Hello World" 应用。
>    - 添加一个 `GET /health` 端点，返回 `{"status": "ok"}`。
>    - 应用应该监听在 `0.0.0.0:8000`。
> 7. 创建一个 `Dockerfile`，用于将 Python 应用容器化。
>    - 使用官方 Python 镜像 (`python:3.9-slim`)。
>    - 复制项目代码，安装依赖。
>    - 暴露端口 8000。
>    - 默认命令: `uvicorn boifi_recommender.main:app --host 0.0.0.0 --port 8000`。

#### 任务 P-2: 数据模型 - 定义核心数据结构

> [User]
> 任务 P-2: 定义 BOIFI 推荐器的核心数据模型。
>
> 要求:
> 1. 创建 `src/boifi_recommender/models/` 包，包含以下模块:
> 2. 在 `models/fault_plan.py` 中定义:
>    - `FaultType` 枚举: DELAY, ABORT, ERROR_INJECTION, RATE_LIMIT (对应 HFI 支持的故障类型)
>    - `FaultPlan` 数据类 (Pydantic BaseModel 或 dataclass + Pydantic):
>      - `fault_type: FaultType`
>      - `target_path: str` (HTTP path prefix or regex)
>      - `target_method: Optional[str]` (HTTP method, default None = all methods)
>      - `percentage: float` (应用故障的百分比, 0-100)
>      - 根据 fault_type 的不同参数:
>        - `delay_ms: Optional[int]` (for DELAY)
>        - `abort_status_code: Optional[int]` (for ABORT, default 500)
>        - `error_message: Optional[str]` (for ERROR_INJECTION)
>      - 实现 `__repr__` 方法用于日志输出
>      - 实现验证: percentage 必须在 [0, 100], delay_ms > 0 等
> 3. 在 `models/observation.py` 中定义:
>    - `RawObservation` 数据类:
>      - `status_code: int` (HTTP 状态码)
>      - `latency_ms: float` (请求延迟，毫秒)
>      - `error_rate: float` (错误率, 0.0-1.0)
>      - `logs: List[str]` (应用日志片段)
>      - `trace_data: Optional[str]` (分布式 Trace JSON 字符串)
>      - `timestamp: datetime` (观测时间)
> 4. 在 `models/scoring.py` 中定义:
>    - `SeverityScore` 数据类:
>      - `total_score: float` (最终评分, 0-10)
>      - `bug_score: float` (Bug 维度评分)
>      - `performance_score: float` (性能维度评分)
>      - `structure_score: float` (结构维度评分)
>      - `breakdown: Dict[str, float]` (详细分项，用于调试)
> 5. 在 `models/session.py` 中定义:
>    - `SessionStatus` 枚举: PENDING, RUNNING, STOPPING, COMPLETED, FAILED
>    - `BestResult` 数据类:
>      - `fault_plan: FaultPlan` (目前最佳的故障组合)
>      - `severity_score: float` (对应的严重性评分)
>      - `iteration: int` (在第几次迭代时找到)
>    - `OptimizationProgress` 数据类:
>      - `current_iteration: int`
>      - `total_iterations: int`
>      - `best_result: Optional[BestResult]`
>      - `history: List[Tuple[FaultPlan, SeverityScore]]` (完整历史)
>    - `SessionState` 数据类:
>      - `session_id: str` (UUID)
>      - `status: SessionStatus`
>      - `created_at: datetime`
>      - `progress: OptimizationProgress`
>      - `error_message: Optional[str]`
> 6. 在 `models/api_models.py` 中定义 REST API 的请求/响应模型 (Pydantic):
>    - `CreateSessionRequest`: 包含 search_space_config (dict), total_trials, initial_points, executor_endpoint
>    - `SessionStatusResponse`: 包含 session_id, status, progress, best_result
>    - `CreateSessionResponse`: 包含 session_id, status (202 Accepted)

#### 任务 P-3: 执行器客户端 - 实现与 HFI 的通信

> [User]
> 任务 P-3: 实现与 HFI Control Plane 的通信客户端。
>
> 要求:
> 1. 创建 `src/boifi_recommender/clients/` 包。
> 2. 在 `clients/executor_client.py` 中定义:
>    - `ExecutorClient` 类:
>      - 构造函数: `__init__(self, control_plane_addr: str, timeout: int = 30)`
>      - 使用 `httpx` 或 `requests` 库
>      - 实现异步方法 (使用 `async/await`)，以避免阻塞主线程
>    - 实现以下方法:
>      - `async apply_policy(fault_plan: FaultPlan) -> bool`: 向 HFI Control Plane POST 一个策略
>        - 将 FaultPlan 转换为 JSON (遵循 HFI 的 API 格式)
>        - POST 到 `/v1/policies` (或类似端点)
>        - 返回是否成功
>      - `async get_observation() -> RawObservation`: 从 HFI Control Plane 获取观测数据
>        - GET 请求到 `/v1/observations/latest` (或类似端点)
>        - 解析响应为 RawObservation
>      - `async health_check() -> bool`: 检查 Control Plane 是否可达
> 3. 实现错误处理:
>    - 连接错误、超时、HTTP 错误应该被捕获并转换为自定义异常，如 `ExecutorConnectionError`, `ExecutorTimeoutError`
>    - 在日志中记录所有错误

#### 任务 P-4: API 服务框架 - 搭建 FastAPI 应用

> [User]
> 任务 P-4: 搭建 BOIFI 推荐器的 REST API 框架。
>
> 要求:
> 1. 修改 `src/boifi_recommender/main.py`:
>    - 导入 FastAPI, CORS 中间件, 日志模块
>    - 创建 FastAPI 应用实例
>    - 添加 CORS 中间件 (允许跨域请求，便于前端调用)
> 2. 在 `src/boifi_recommender/api/` 包中创建路由文件 `routes.py`:
>    - 定义路由组 `/v1/optimization`
>    - 实现以下端点 (暂时返回占位符或简单逻辑):
>      - `POST /v1/optimization/sessions`: 创建新的优化会话
>        - 请求体: `CreateSessionRequest`
>        - 响应 (202 Accepted): `CreateSessionResponse`
>      - `GET /v1/optimization/sessions/{session_id}`: 查询会话状态
>        - 响应 (200 OK): `SessionStatusResponse`
>      - `POST /v1/optimization/sessions/{session_id}/stop`: 停止会话
>        - 响应 (202 Accepted): `{status: "stopping"}`
>      - `GET /health`: 健康检查
> 3. 在路由处理器中，进行基本的输入验证和错误处理。
> 4. 为所有响应添加适当的 HTTP 状态码。

#### 任务 P-5: 会话管理器框架 - 实现会话生命周期管理

> [User]
> 任务 P-5: 实现会话管理器，支持会话的创建、查询和停止。
>
> 要求:
> 1. 创建 `src/boifi_recommender/services/` 包。
> 2. 在 `services/session_manager.py` 中定义:
>    - `SessionManager` 类 (单例模式):
>      - 使用线程安全的字典存储活跃会话: `Dict[str, SessionState]`
>      - 使用 `threading.Lock` 保护并发访问
>    - 实现以下方法:
>      - `create_session(config: CreateSessionRequest) -> str`: 创建新会话
>        - 生成唯一的 session_id (使用 uuid.uuid4())
>        - 初始化 SessionState，状态为 PENDING
>        - 返回 session_id
>      - `get_session(session_id: str) -> Optional[SessionState]`: 查询会话
>      - `list_sessions() -> List[SessionState]`: 列出所有会话
>      - `update_session(session_id: str, state: SessionState) -> bool`: 更新会话状态
>      - `stop_session(session_id: str) -> bool`: 请求停止会话
> 3. 实现线程安全:
>    - 所有方法都应该使用 Lock 保护对共享字典的访问
>    - 避免死锁 (持有 Lock 时不调用其他可能获取 Lock 的方法)

#### 任务 T-1: 集成测试环境 - 搭建 Docker Compose

> [User]
> 任务 T-1: 为开发和测试搭建一个 Docker Compose 环境。
>
> 要求:
> 1. 创建 `docker-compose-dev.yaml` 文件，包含:
>    - **recommender service**: 
>      - 使用之前创建的 Dockerfile
>      - 暴露端口 8000
>      - 环境变量: HFI_EXECUTOR_ADDR=http://mock-executor:9090
>    - **mock-executor service** (可选，用于模拟 HFI Control Plane):
>      - 一个简单的 Python Flask/FastAPI 应用
>      - 实现两个端点:
>        - `POST /v1/policies`: 接收策略，打印日志
>        - `GET /v1/observations/latest`: 返回模拟的 RawObservation
>      - 暴露端口 9090
> 2. 创建 `docker/mock_executor.py`:
>    - 一个简单的 Flask 应用，模拟 HFI Control Plane 的 API
>    - 接收策略时，打印策略内容
>    - 返回固定的模拟观测数据
> 3. 编写启动脚本 `scripts/start-dev.sh`:
>    - 使用 `docker-compose -f docker-compose-dev.yaml up` 启动所有服务
>    - 等待服务启动后，输出访问地址
>
> **测试步骤**:
> 1. 运行 `bash scripts/start-dev.sh`
> 2. 在另一个终端中，使用 `curl` 测试 API:
>    ```bash
>    # 创建会话
>    curl -X POST http://localhost:8000/v1/optimization/sessions \
>      -H "Content-Type: application/json" \
>      -d '{
>        "search_space_config": {...},
>        "total_trials": 100,
>        "initial_points": 10,
>        "executor_endpoint": "http://mock-executor:9090"
>      }'
>    
>    # 查询会话
>    curl http://localhost:8000/v1/optimization/sessions/{session_id}
>    ```
> 3. 检查是否收到正确的响应和日志输出。

---

### 第二阶段 Prompt

#### 任务 O-1: 搜索空间转换器 - 实现 SpaceConverter

> [User]
> 任务 O-1: 实现 `SpaceConverter`，将 YAML/JSON 搜索空间配置转换为 scikit-optimize 的 Dimension 对象。
>
> 要求:
> 1. 创建 `src/boifi_recommender/optimizer/` 包。
> 2. 在 `optimizer/space_converter.py` 中定义:
>    - `DimensionConfig` 数据类: 表示单个维度的配置
>      - `name: str`
>      - `type: str` (categorical, real, integer)
>      - `values: List[Any]` (for categorical)
>      - `min: float, max: float` (for real/integer)
>      - `log_scale: bool` (for real, optional)
>    - `SpaceConverter` 类:
>      - 方法 `from_dict(config_dict: dict) -> List[Dimension]`:
>        - 接收用户的搜索空间配置 (dict 形式)
>        - 遍历每个维度定义
>        - 根据类型调用相应的 scikit-optimize API:
>          - `Categorical(categories=...)` for categorical
>          - `Real(low, high, prior='uniform'|'log-uniform')` for real
>          - `Integer(low, high)` for integer
>        - 返回 Dimension 列表
>      - 方法 `to_dict(point: List[Any]) -> dict`:
>        - 将 scikit-optimize 返回的点 (list) 转换为可读的 dict 形式
>        - 例: [500, 0.1, "abort"] → {"delay_ms": 500, "cpu_load": 0.1, "fault_type": "abort"}
>
> **示例配置**:
> ```yaml
> search_space:
>   - name: delay_ms
>     type: integer
>     min: 0
>     max: 5000
>   - name: cpu_load
>     type: real
>     min: 0.0
>     max: 1.0
>   - name: fault_type
>     type: categorical
>     values: [abort, delay, error_injection]
> ```

#### 任务 O-2: 搜索空间验证 - 实现 SpaceValidator

> [User]
> 任务 O-2: 实现搜索空间的验证逻辑。
>
> 要求:
> 1. 在 `optimizer/validation.py` 中定义:
>    - `SpaceValidator` 类:
>      - 方法 `validate_dimensions(dimensions: List[Dimension]) -> None`:
>        - 检查维度名称是否唯一 (抛出 `ValueError` 如果重复)
>        - 检查维度类型是否有效
>        - 检查数值范围的有效性 (min < max)
>        - 检查 categorical 是否为空
>      - 方法 `validate_point(point: dict, config: dict) -> None`:
>        - 检查点的值是否在定义的范围内
>        - 检查必需的维度是否都存在
>      - 错误消息应该清晰，便于用户调试
> 2. 将验证集成到 `SpaceConverter.from_dict` 中，使得构造时就进行验证。

#### 任务 O-3: 代理模型 - 实现 ProxyModel

> [User]
> 任务 O-3: 实现基于随机森林的代理模型。
>
> 要求:
> 1. 在 `optimizer/proxy_model.py` 中定义:
>    - `ProxyModel` 类:
>      - 构造函数: `__init__(self, n_estimators: int = 100, random_state: int = 42)`
>        - 使用 scikit-optimize 提供的 `RandomForestRegressor` 或 scikit-learn 的直接接口
>      - 方法 `fit(X: List[List[Any]], y: List[float]) -> None`:
>        - 训练模型
>        - X 是点的列表 (从 scikit-optimize Dimension 编码)
>        - y 是对应的观测值 (严重性评分)
>        - 如果 X 为空或样本数不足，记录警告
>      - 方法 `predict(X: List[List[Any]]) -> np.ndarray`:
>        - 预测给定点的目标函数值
>        - 返回预测值数组
>      - 方法 `predict_std(X: List[List[Any]]) -> np.ndarray`:
>        - 返回预测的不确定性 (标准差)
>        - 使用随机森林的树间方差来估计
>      - 属性 `is_fitted: bool`:
>        - 返回模型是否已训练
> 2. 实现错误处理:
>    - 训练数据格式错误时抛出异常
>    - 预测前检查模型是否已训练

#### 任务 O-4: 采集函数 - 实现 AcquisitionFunction

> [User]
> 任务 O-4: 实现采集函数，用于在搜索空间中选择最有潜力的点。
>
> 要求:
> 1. 在 `optimizer/acquisition.py` 中定义:
>    - `AcquisitionFunction` 抽象基类:
>      - 抽象方法 `evaluate(point: List[Any], model: ProxyModel, history_y: List[float]) -> float`:
>        - 给定一个点，评估其"潜力" (高值表示更有希望)
>        - history_y 是历史观测值列表 (用于计算当前最优值)
>    - `ExpectedImprovement` (EI) 实现:
>      - EI 公式: $EI(x) = E[max(f(x) - f(x_best), 0)]$
>      - 使用模型的预测均值和标准差
>      - 实现对标准正态分布的积分计算 (使用 scipy.stats.norm)
>      - 可选: 实现 `xi` 参数用于平衡探索和利用
>    - `UpperConfidenceBound` (UCB) 实现 (可选):
>      - UCB 公式: $UCB(x) = \mu(x) + \beta \cdot \sigma(x)$
>      - $\beta$ 可配置 (默认 1.96 对应 95% 置信区间)
> 2. 导出工厂函数:
>    - `get_acquisition_function(name: str, **kwargs) -> AcquisitionFunction`
>    - 支持 "ei", "ucb" 等

#### 任务 O-5: 优化器核心 - 实现 OptimizerCore

> [User]
> 任务 O-5: 实现贝叶斯优化的核心逻辑。
>
> 要求:
> 1. 在 `optimizer/core.py` 中定义:
>    - `OptimizerCore` 类:
>      - 构造函数: `__init__(self, space_config: dict, acquisition_func: str = "ei")`
>        - 初始化搜索空间 (使用 SpaceConverter)
>        - 初始化代理模型 (ProxyModel)
>        - 初始化采集函数
>      - 方法 `propose() -> dict`:
>        - 如果模型未训练 (初始化阶段):
>          - 从搜索空间中随机采样一个点
>          - 返回点的 dict 表示
>        - 如果模型已训练:
>          - 在搜索空间中网格采样或随机采样大量候选点 (如 1000 个)
>          - 使用采集函数评估每个候选点
>          - 选择采集函数值最高的点
>          - 返回该点的 dict 表示
>      - 方法 `record(point: dict, score: float) -> None`:
>        - 接收一个实验点和对应的观测值 (严重性评分)
>        - 将点和评分添加到历史数据中
>        - 使用新的历史数据重训练模型
>      - 属性 `history: List[Tuple[dict, float]]`:
>        - 返回完整的历史数据
>      - 属性 `best_point: Tuple[dict, float]`:
>        - 返回目前的最优点和其评分
> 2. 实现 memoization:
>    - 如果 propose() 建议的点已经在历史中被测试过，则随机选择第二好的点，避免重复

#### 任务 T-2: 优化器单元测试

> [User]
> 任务 T-2: 为优化器核心编写详尽的单元测试。
>
> 要求:
> 1. 创建 `tests/test_optimizer.py`:
>    - 测试 SpaceConverter:
>      - 测试各种维度类型的转换
>      - 测试配置验证
>    - 测试 ProxyModel:
>      - 测试训练和预测
>      - 测试不确定性估计
>    - 测试 OptimizerCore:
>      - 测试 propose() 在初始化和训练后的行为
>      - 测试 record() 后模型是否更新
>      - 测试历史追踪
>      - 测试避免重复采样
> 2. 使用 `pytest` 框架，至少 20+ 个测试用例。
> 3. 使用 Mock 库来模拟 ProxyModel 的预测，便于单元测试隔离。
> 4. 测试覆盖率目标: >= 80%

---

### 第三阶段 Prompt

#### 任务 A-1: Bug 评分器 - 实现 BugScorer

> [User]
> 任务 A-1: 实现 `BugScorer`，检测系统中的软件缺陷迹象。
>
> 要求:
> 1. 在 `src/boifi_recommender/analyzer/scorers/` 包中创建 `bug_scorer.py`:
> 2. 定义 `BugScorer` 类，继承自 `IScorer` 接口:
>    - 方法 `score(observation: RawObservation) -> float`:
>      - 优先级规则 (返回第一个匹配的规则的分数):
>        1. 如果 status_code >= 500: 返回 10.0 (服务器错误)
>        2. 如果 status_code >= 400: 返回 8.0 (客户端错误)
>        3. 如果日志中包含 "ERROR" 或 "FATAL": 返回 6.0
>        4. 如果 error_rate > 0: 返回 3.0 (检测到部分错误)
>        5. 否则: 返回 0.0
>      - 日志匹配应该是不区分大小写的
>      - 返回结果应该在 [0, 10] 范围内
> 3. 实现错误处理:
>    - 如果 observation 为 None，抛出 `ValueError`
>    - 如果缺少必需字段 (status_code), 返回默认值 0.0 并记录警告
> 4. 在日志中记录每次评分的理由，便于调试

#### 任务 A-2: 性能评分器 - 实现 PerformanceScorer

> [User]
> 任务 A-2: 实现 `PerformanceScorer`，衡量故障对响应时间的影响。
>
> 要求:
> 1. 在 `analyzer/scorers/` 中创建 `performance_scorer.py`:
> 2. 定义 `PerformanceScorer` 类:
>    - 构造函数: `__init__(self, baseline_ms: float = 200, threshold_ms: float = 1000)`
>      - baseline_ms: 正常情况下的预期延迟
>      - threshold_ms: 可接受的最大延迟
>    - 方法 `score(observation: RawObservation) -> float`:
>      - 根据公式:
>        ```
>        if latency > threshold:
>            score = 10.0
>        elif latency >= baseline:
>            ratio = (latency - baseline) / (threshold - baseline)
>            score = min(9.0 * ratio, 10.0)
>        else:
>            score = 0.0
>        ```
>      - 返回值始终在 [0, 10] 范围内
> 3. 实现验证:
>    - baseline_ms 和 threshold_ms 应该都 > 0
>    - baseline_ms < threshold_ms
> 4. 在日志中记录计算过程，便于调试

#### 任务 A-3: 结构分析器 - 实现 StructureScorer

> [User]
> 任务 A-3: 实现 `StructureScorer`，通过分布式 Trace 分析检测系统异常模式。
>
> 要求:
> 1. 在 `analyzer/` 中创建 `trace_analyzer.py`:
>    - 定义 `TraceAnalyzer` 类:
>      - 方法 `parse_trace(trace_json: str) -> List[Span]`:
>        - 解析 Trace 的 JSON 字符串 (假设格式: `{"spans": [{"id": "...", "name": "...", "duration_ms": 123, "status": "success|error", ...}]}`)
>        - 返回 Span 列表
>      - 方法 `compare_traces(current: List[Span], baseline: List[Span]) -> dict`:
>        - 计算 Span 数变化百分比
>        - 计算操作序列的编辑距离 (Levenshtein distance)
>        - 检查是否存在 ERROR 状态的 Span
>        - 检查是否存在性能瓶颈 (单个 Span 延迟增加 > 5 倍)
>        - 返回 dict 包含所有指标
> 2. 在 `analyzer/scorers/` 中创建 `structure_scorer.py`:
>    - 定义 `StructureScorer` 类:
>      - 构造函数: `__init__(self, trace_analyzer: TraceAnalyzer = None)`
>      - 方法 `score(observation: RawObservation, baseline_observation: Optional[RawObservation] = None) -> float`:
>        - 如果没有 Trace 数据, 返回 0.0
>        - 否则调用 trace_analyzer.compare_traces()
>        - 根据检测到的异常指标返回评分:
>          - Span 数增加 > 50%: +3.0
>          - 编辑距离 > 2: +5.0
>          - 存在 ERROR Span: +2.0
>          - 性能瓶颈: +2.0
>        - 返回 min(10.0, max(0.0, sum))
> 3. 实现错误处理:
>    - Trace JSON 解析失败时, 返回 0.0 并记录错误
>    - 缺少 baseline 时, 仅基于当前 Trace 分析

#### 任务 A-4: 分析器服务 - 实现 AnalyzerService

> [User]
> 任务 A-4: 实现 `AnalyzerService`，整合所有评分器并计算最终的严重性评分。
>
> 要求:
> 1. 在 `analyzer/` 中创建 `service.py`:
> 2. 定义 `AnalyzerService` 类:
>    - 构造函数: `__init__(self, config: AnalyzerConfig)`
>      - 接收配置对象，包含权重、阈值等参数
>      - 初始化三个评分器 (BugScorer, PerformanceScorer, StructureScorer)
>    - 方法 `calculate_severity(observation: RawObservation, baseline: Optional[RawObservation] = None) -> SeverityScore`:
>      - 调用三个评分器:
>        - bug_score = BugScorer.score(observation)
>        - perf_score = PerformanceScorer.score(observation)
>        - struct_score = StructureScorer.score(observation, baseline)
>      - 计算加权平均:
>        ```
>        w_bug, w_perf, w_struct = config.weights['bug'], config.weights['perf'], config.weights['struct']
>        total_score = (w_bug * bug_score + w_perf * perf_score + w_struct * struct_score) / (w_bug + w_perf + w_struct)
>        ```
>      - 返回 SeverityScore 对象，包含总分和分项
>    - Fail-Safe 原则:
>      - 任何一个评分器的失败都不应导致整个调用失败
>      - 使用 try-except 捕获每个评分器的异常
>      - 记录详细错误日志
>      - 返回一个合理的默认分数
> 3. 实现日志:
>    - 记录每次评分的完整过程，便于调试

#### 任务 A-5: 分析器配置 - 实现 AnalyzerConfig

> [User]
> 任务 A-5: 实现分析器的配置管理。
>
> 要求:
> 1. 在 `analyzer/` 中创建 `config.py`:
> 2. 定义 `AnalyzerConfig` 数据类:
>    - `weights: Dict[str, float]` (默认: {"bug": 10, "perf": 2, "struct": 5})
>    - `baseline_latency_ms: float` (默认: 200)
>    - `threshold_latency_ms: float` (默认: 1000)
>    - `span_increase_threshold: float` (默认: 0.5 表示 50%)
>    - `edit_distance_threshold: int` (默认: 2)
>    - 实现 `from_yaml(path: str)` 类方法，从 YAML 文件加载配置
>    - 实现验证，确保权重都 > 0, 延迟阈值合理等

#### 任务 T-3: 分析器单元与集成测试

> [User]
> 任务 T-3: 为分析器编写详尽的测试。
>
> 要求:
> 1. 创建 `tests/test_analyzer.py`:
>    - 测试 BugScorer:
>      - 测试各种 HTTP 状态码
>      - 测试日志匹配
>      - 测试错误率检测
>    - 测试 PerformanceScorer:
>      - 测试边界情况 (baseline, threshold)
>      - 测试超阈值情况
>    - 测试 StructureScorer:
>      - 测试 Trace 解析
>      - 测试异常检测 (Span 数变化、编辑距离等)
>    - 测试 AnalyzerService:
>      - 测试加权平均计算
>      - 测试 Fail-Safe (某个评分器失败时的行为)
>      - 测试完整的评分过程
> 2. 使用 fixture 提供测试数据 (模拟 RawObservation, Trace 等)
> 3. 测试覆盖率目标: >= 80%

---

### 第四阶段 Prompt

#### 任务 C-1: 优化工作者 - 实现 OptimizationWorker

> [User]
> 任务 C-1: 实现 `OptimizationWorker`，负责执行优化主循环。
>
> 要求:
> 1. 在 `src/boifi_recommender/coordinator/` 包中创建 `worker.py`:
> 2. 定义 `OptimizationWorker` 类:
>    - 构造函数: `__init__(self, session_id: str, optimizer: OptimizerCore, executor_client: ExecutorClient, analyzer: AnalyzerService, session_manager: SessionManager, total_trials: int, initial_points: int)`
>    - 方法 `run(self)` (主循环):
>      ```
>      for iteration in range(total_trials):
>          # 1. 获取下一个故障组合
>          fault_plan = optimizer.propose()
>          
>          # 2. 在执行器上执行
>          success = executor_client.apply_policy(fault_plan)
>          if not success:
>              # 记录失败，继续下一次迭代
>              continue
>          
>          # 3. 获取观测数据
>          observation = executor_client.get_observation()
>          
>          # 4. 计算严重性评分
>          severity = analyzer.calculate_severity(observation)
>          
>          # 5. 反馈给优化器
>          optimizer.record(fault_plan, severity.total_score)
>          
>          # 6. 更新会话进度
>          session_manager.update_progress(session_id, iteration, fault_plan, severity)
>          
>          # 7. 检查是否被要求停止
>          if session_manager.should_stop(session_id):
>              break
>      ```
>    - 实现错误处理:
>      - 网络错误 → 重试 (最多 3 次)
>      - 超时 → 跳过当前迭代
>      - 数据缺失 → 使用默认值
>    - 实现日志:
>      - 记录每次迭代的详细信息
>      - 记录发现的最佳结果
> 3. 支持中断:
>    - 使用 `threading.Event` 来实现优雅停止
>    - 主循环检查事件，若收到停止信号，在当前迭代完成后退出

#### 任务 C-2: 工作者线程管理 - 完善 SessionManager

> [User]
> 任务 C-2: 完善 `SessionManager`，支持为每个会话启动独立的 Worker 线程。
>
> 要求:
> 1. 修改 `services/session_manager.py`:
>    - 添加字段:
>      - `workers: Dict[str, OptimizationWorker]` (存储会话对应的 Worker)
>      - `worker_threads: Dict[str, threading.Thread]` (存储工作线程)
>      - `stop_events: Dict[str, threading.Event]` (用于停止信号)
>    - 修改 `create_session` 方法:
>      - 创建 OptimizationWorker 实例
>      - 创建一个新线程并启动 Worker.run()
>      - 记录线程和停止事件
>    - 添加方法 `should_stop(session_id: str) -> bool`:
>      - 检查对应会话的停止事件是否被设置
>    - 添加方法 `request_stop(session_id: str) -> None`:
>      - 设置停止事件，Worker 会在当前迭代后停止
>    - 添加方法 `wait_for_completion(session_id: str, timeout: int = None) -> bool`:
>      - 等待工作线程完成
>      - 支持超时

#### 任务 C-3: 优雅停止机制 - 实现 Stop Logic

> [User]
> 任务 C-3: 实现优雅停止机制，使得会话能够平稳地停止而不丢失数据。
>
> 要求:
> 1. 在 `coordinator/` 中创建 `stop_handler.py`:
>    - 定义 `StopHandler` 类:
>      - 方法 `finalize_session(session_id: str, session_state: SessionState) -> None`:
>        - 更新会话状态为 COMPLETED
>        - 保存最终结果
>        - 生成会话总结报告 (迭代次数、最佳结果、性能统计等)
> 2. 在 Worker 的主循环中，当检测到停止信号时:
>    - 完成当前迭代
>    - 调用 StopHandler.finalize_session()
>    - 优雅地退出

#### 任务 C-4: 结果持久化 - 实现数据保存与加载

> [User]
> 任务 C-4: 实现会话结果的持久化。
>
> 要求:
> 1. 在 `services/` 中创建 `persistence.py`:
>    - 定义 `PersistenceService` 类:
>      - 方法 `save_session(session_state: SessionState, output_dir: str = "./sessions") -> str`:
>        - 将会话状态序列化为 JSON
>        - 保存到文件 `{output_dir}/{session_id}/state.json`
>        - 同时保存完整的历史数据到 `{output_dir}/{session_id}/history.jsonl` (每行一个观测)
>        - 返回保存路径
>      - 方法 `load_session(session_id: str, input_dir: str = "./sessions") -> SessionState`:
>        - 从文件加载会话状态
>      - 方法 `generate_report(session_state: SessionState) -> str`:
>        - 生成一个 Markdown 格式的报告，包含:
>          - 优化过程概览 (总迭代数、耗时等)
>          - 最佳结果
>          - 评分历史曲线 (用 ASCII 艺术或参考数据)
>          - 建议的下一步行动

#### 任务 C-5: API 完整化 - 实现所有 REST 端点

> [User]
> 任务 C-5: 完成 REST API 的所有端点实现。
>
> 要求:
> 1. 修改 `api/routes.py`:
>    - `POST /v1/optimization/sessions`:
>      - 验证请求体
>      - 调用 SessionManager.create_session()
>      - 返回 202 Accepted + session_id
>    - `GET /v1/optimization/sessions/{session_id}`:
>      - 调用 SessionManager.get_session()
>      - 返回 SessionStatusResponse
>      - 如果会话不存在，返回 404
>    - `POST /v1/optimization/sessions/{session_id}/stop`:
>      - 调用 SessionManager.request_stop()
>      - 返回 202 Accepted
>    - `GET /v1/optimization/sessions`:
>      - 返回所有会话的列表 (支持分页)
>    - `GET /v1/optimization/sessions/{session_id}/report`:
>      - 生成并返回优化报告 (Markdown 或 JSON)
> 2. 实现错误处理和适当的 HTTP 状态码

#### 任务 C-6: 错误处理与恢复 - 实现容错机制

> [User]
> 任务 C-6: 实现整个优化循环的容错机制。
>
> 要求:
> 1. 在 `coordinator/` 中创建 `error_handler.py`:
>    - 定义 `ErrorHandler` 类，提供重试和降级策略:
>      - 方法 `retry_with_backoff(func, max_retries: int = 3, backoff_factor: float = 2.0)`:
>        - 实现指数退避重试
>        - 返回结果或抛出 `MaxRetriesExceededError`
>      - 方法 `handle_executor_error(error, fallback_value = None)`:
>        - 区分错误类型 (连接错误、超时、HTTP 错误等)
>        - 返回是否可恢复
>      - 方法 `handle_analyzer_error(error, fallback_score: float = 5.0)`:
>        - 如果分析器失败，返回中性评分
> 2. 在 Worker 中集成 ErrorHandler:
>    - 调用执行器时使用重试机制
>    - 调用分析器时使用 Fail-Safe 原则
> 3. 在会话状态中记录所有错误，用于事后分析

#### 任务 T-4: 端到端测试 - 完整流程验证

> [User]
> 任务 T-4: 编写端到端集成测试，验证完整的优化流程。
>
> 要求:
> 1. 创建 `tests/test_e2e.py`:
> 2. 编写测试用例:
>    - `test_complete_optimization_session()`:
>      - 启动一个优化会话 (总迭代数 = 5)
>      - 验证会话成功创建 (session_id 非空)
>      - 轮询查询会话状态，直到 COMPLETED
>      - 验证最终结果有效 (best_result 非空)
>      - 验证历史数据完整 (5 个迭代对应的数据都存在)
>    - `test_stop_session()`:
>      - 启动会话
>      - 中途发送停止请求
>      - 验证会话优雅停止
>    - `test_error_recovery()`:
>      - 模拟执行器返回错误
>      - 验证 Worker 正确重试
>    - `test_concurrent_sessions()`:
>      - 并发启动 3 个会话
>      - 验证它们独立运行，相互不影响
> 3. 使用 Mock 库模拟 ExecutorClient 和 AnalyzerService，便于测试隔离
> 4. 测试覆盖率目标: >= 75%

---

## 开发过程总结与最佳实践

### 代码审查清单

在提交代码前，确保满足以下条件：

**第一阶段**:
- [ ] 所有数据模型都通过 Pydantic 验证
- [ ] ExecutorClient 能成功连接到 mock executor
- [ ] FastAPI 应用能启动，至少 3 个端点可用
- [ ] docker-compose 配置正确，容器能启动

**第二阶段**:
- [ ] SpaceConverter 能正确处理各种维度类型
- [ ] ProxyModel 能训练和预测，不确定性估计合理
- [ ] OptimizerCore.propose() 和 record() 接口稳定
- [ ] 单元测试覆盖率 >= 80%, 所有测试通过

**第三阶段**:
- [ ] BugScorer, PerformanceScorer, StructureScorer 各自正确
- [ ] AnalyzerService 计算的评分在 [0, 10] 范围内
- [ ] Fail-Safe 机制正确：单个评分器失败不影响整体
- [ ] 单元测试覆盖率 >= 80%, 所有测试通过

**第四阶段**:
- [ ] Worker 能完整运行多个迭代
- [ ] SessionManager 能管理多个并发会话
- [ ] 所有 REST API 端点正确响应
- [ ] 错误处理完整，重试机制有效
- [ ] 端到端测试通过，并发测试通过

**第五阶段**:
- [ ] Docker 镜像大小合理 (< 500MB)
- [ ] K8s 配置通过验证，能部署到真实集群
- [ ] 监控指标正确导出 (Prometheus format)
- [ ] 文档完整，包含 API、部署、故障排查
- [ ] 压力测试通过，性能指标达标

---

**项目状态**: 待启动  
**最后更新**: 2024-11-13  
**维护者**: BOIFI 开发团队
