# BOIFI 项目宪法 (Constitution)

## 核心原则

### I. 关注点分离 (Separation of Concerns)
系统必须严格遵循关注点分离原则，将复杂问题分解为独立的、单一职责的模块。具体要求：
- **决策与执行分离**: Recommender (决策大脑) 与 Executor (执行手脚) 通过明确定义的接口完全解耦
- **控制平面与数据平面分离**: Control Plane (管理中心，无状态) 与 Data Plane (Wasm 插件，极致性能) 职责清晰
- **API 与流式通信分离**: RESTful 接口处理策略管理，SSE/gRPC 处理配置实时分发
- **各服务独立性**: Coordinator, Optimizer Core, Response Analyzer 等各子模块应独立可测、可扩展

### II. 声明式配置 (Declarative Configuration)
所有策略和规范必须使用声明式方式定义，而非命令式脚本。具体要求：
- **Policy Schema**: FaultInjectionPolicy 必须清晰定义所有字段 (metadata, spec.rules, match, fault)
- **Space Config**: 故障空间维度必须通过结构化的 YAML/JSON 配置定义
- **验证先行**: 配置必须在解析时进行全量校验，拒绝不合规的定义
- **向前兼容**: 配置格式升级必须保持向后兼容，避免已有策略失效

### III. 动态性与实时性 (Dynamic & Real-Time)
系统必须支持在不重启任何组件的情况下，实时更新和应用规则。具体要求：
- **无缝策略更新**: Control Plane 中策略变更应通过 SSE/Watch 实时推送到 Data Plane，延迟 < 1 秒
- **热更新配置**: Wasm 插件必须使用并发安全的机制 (atomic.Value, RwLock) 接收新规则，不中断请求处理
- **幂等操作**: 所有 API 操作必须是幂等的，支持重试而无副作用
- **版本管理**: 每次规则集更新必须包含版本号，便于追踪和调试

### IV. 测试驱动 (Test-Driven Development - 强制要求)
所有功能实现必须遵循 TDD 流程，这是非可协商的。具体要求：
- **单元测试优先**: 每个模块 (Matcher, Executor, Analyzer, Optimizer) 必须有详尽的单元测试，覆盖核心逻辑和边界情况
- **集成测试覆盖**: API Handler → Service → DAL, Coordinator → Executor 等跨组件的交互必须有集成测试
- **端到端验证**: 从策略应用到故障注入的完整流程必须有 E2E 测试
- **性能基准测试**: 关键热路径 (Matcher, Executor) 必须有基准测试，验证性能目标 (< 1ms 延迟)
- **测试覆盖率**: 代码覆盖率必须 > 70%，关键路径 > 90%

### V. 性能优先 (Performance-First Design)
系统设计和实现必须优先考虑性能，尤其是 Data Plane。具体要求：
- **Wasm 插件开销**: 在不注入故障时，插件对请求的额外延迟必须 < 1ms
- **热路径优化**: Matcher (请求匹配) 和 Executor (故障执行) 是热路径，不能进行不必要的内存分配或锁竞争
- **规则预处理**: 正则表达式、时间解析等必须在规则更新时进行，不在请求处理时进行
- **配置高效传输**: CompiledRuleSet 的大小应保持在可接受范围 (< 1MB)，避免频繁序列化开销

### VI. 容错与可靠性 (Fault Tolerance & Reliability)
系统必须能优雅地处理各种故障场景。具体要求：
- **Fail-Safe Response Analyzer**: 缺失 Trace 或 HTTP 状态码不应导致整个优化循环失败，应返回默认评分值
- **重连与重试**: Wasm 插件断连应自动重连，Executor Client 请求失败应有指数退避重试策略
- **策略回滚**: 若新规则导致异常 (如规则解析失败)，应回退到上一个已知的良好规则集
- **可观测性**: 所有关键路径 (Policy CRUD, 规则分发, 故障注入) 必须有结构化日志和 Trace 支持

### VII. 简洁性与最小化原则 (Simplicity & Minimalism)
设计和实现必须优先选择简单方案，避免过度工程化。具体要求：
- **初期存储选择**: 阶段 1-2 使用内存存储或单节点 etcd，不提前引入分布式复杂性
- **协议选择**: 使用 SSE (而非复杂的 gRPC) 进行配置推送
- **库依赖**: 优先使用成熟、轻量的库 (scikit-optimize 而非复杂的 BoTorch)
- **功能范围**: YAGNI 原则，只实现规范中明确定义的功能，不推测未来需求

### VIII. 时间控制与生命周期管理 (Temporal Control & Lifecycle)
系统必须支持灵活的策略生命周期管理，支持持久化和临时性规则。具体要求：
- **duration_seconds 字段**: `duration_seconds = 0` (持久) vs `> 0` (临时)，由 Control Plane 自动处理过期删除
- **start_delay_ms 字段**: 支持请求到达后延迟执行故障，实现精确的故障注入时机控制
- **自动清理**: 过期的临时性 Policy 应由 Control Plane 的后台任务自动删除
- **时间准确性**: 时间控制精度应在秒级 (start_delay_ms 级别)

## 技术栈与约束

### 编程语言
- **Control Plane / CLI**: Go (1.20+) - 云原生首选，高并发，易部署
- **Wasm 插件**: Go (proxy-wasm-go-sdk) 或 Rust (proxy-wasm-rust-sdk) - 根据团队选择，性能优先
- **Recommender / Optimizer**: Python (3.8+) - 数据科学生态，scikit-optimize 库

### 存储与通信
- **Policy Store**: 阶段 1 使用内存 + 文件备份，阶段 2+ 升级至 etcd/Redis/PostgreSQL
- **Config Distribution**: Server-Sent Events (SSE) over HTTP - 轻量且足够
- **API 格式**: JSON for REST, Protocol Buffers (optional) for gRPC

### 部署与监控
- **容器化**: 所有组件必须提供 Dockerfile，支持 Docker 构建和运行
- **Kubernetes**: 提供 YAML 清单 (Deployment, Service, ConfigMap)，后期提供 Helm Chart
- **日志与追踪**: 结构化日志 (JSON), 支持接入 ELK/Loki; 可选的分布式追踪 (Jaeger/Zipkin)

## 开发工作流

### 代码审查与提交
- **PR 必需**:所有代码变更必须通过 Pull Request，至少 1 名审查者同意
- **CI/CD 强制**: 必须通过自动化 CI (编译、单元测试、lint) 才能合并
- **提交信息**: 使用标准的 Conventional Commits 格式 (feat:, fix:, docs:, test:, refactor: 等)

### 版本管理与发布
- **语义化版本**: MAJOR.MINOR.PATCH (e.g., 1.0.2)
  - MAJOR: 向后不兼容的 API 或功能删除
  - MINOR: 新增功能或原则调整
  - PATCH: Bug 修复、文档更新、澄清说明
- **发布标签**: 每个发布版本必须打 Git tag，标签格式为 `v1.0.0`
- **变更日志**: CHANGELOG.md 必须记录每个版本的所有变更

### 文档要求
- **设计文档**: 每个重要功能必须有设计文档 (设计目标、接口定义、数据流、技术选型)
- **API 文档**: 所有公开 API 必须有 OpenAPI (Swagger) 规范
- **代码注释**: 关键函数、复杂算法、易误点必须有清晰的注释
- **README**: 每个模块必须有 README，说明其职责、使用方式、开发指南

## 治理 (Governance)

### 宪法的地位与权重
- 本宪法定义了 BOIFI 项目的核心原则和约束，高于所有其他开发指南和最佳实践文档
- 任何代码、配置、流程必须遵守本宪法定义的原则
- 若发现违反，审查者有权拒绝合并或要求修改

### 宪法的修订
- **修订流程**: 任何人都可以提议修订，但必须在项目的主要贡献者中征求共识 (至少 2/3 同意)
- **修订记录**: 每次修订必须更新本宪法，在"最后修订日期"字段中记录时间，并在 CHANGELOG.md 中说明理由
- **向后兼容**: 修订必须尽量保持向后兼容，突破性变更必须有充分的文档和迁移指南

### 合规验证
- **CI 级别**: 单元测试、代码覆盖率、Lint 检查等在 CI 中自动验证
- **Code Review 级别**: 审查者必须验证 PR 是否遵守以上原则 (特别是 TDD、关注点分离)
- **定期审计**: 每个季度进行一次宪法合规审计，检查是否存在持续的违反情况

### 运行时开发指导
- 详见项目根目录的 `DEVELOPMENT.md` 和各模块的 `README.md`
- 实时开发问题可参考 `docs/dev_doc/` 下的各类详细指南

---

**版本**: 1.0.0 | **制定日期**: 2025-11-13 | **最后修订**: 2025-11-13
