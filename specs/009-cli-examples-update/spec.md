# Feature Specification: CLI Examples Update for Multi-Service Microservice System

**Feature Branch**: `009-cli-examples-update`  
**Created**: 2025-12-10  
**Status**: Draft  
**Input**: User description: "更新 CLI 示例以支持多服务微服务系统中的服务选择器，并添加 K8s/Istio 环境下的验证注入效果脚本"

## Background

BOIFI Executor 原本针对单个 Envoy 设计，现已扩展为支持多服务微服务系统。当前的关键变化：

1. **服务选择器 (Service Selector)**: Wasm 插件现在通过 `EnvoyIdentity` 从 Envoy 元数据中获取当前服务的身份信息（`WORKLOAD_NAME` 和 `NAMESPACE`），并使用 `ServiceSelector` 来决定策略是否应用于当前服务
2. **策略匹配逻辑**: 只有当策略的 `selector` 匹配当前服务身份时，才会执行故障注入；否则请求会被放行
3. **运行环境**: 系统运行在 k3s + Istio 环境中，Wasm 插件通过 Istio 的 `WasmPlugin` CRD 部署到 istio-proxy sidecar

现有的 `/executor/cli/examples/` 目录中的示例需要更新以反映这些变化，并需要添加用于验证注入效果的脚本。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 更新现有策略示例以包含服务选择器 (Priority: P1)

作为一名 SRE 工程师，我希望 CLI 示例中的策略文件都包含 `selector` 字段，以便我能够理解如何针对特定服务进行故障注入，而不会影响其他服务。

**Why this priority**: 这是基础功能，所有后续验证都依赖于正确的策略格式。如果示例不正确，用户将无法正确使用系统。

**Independent Test**: 可以通过将更新后的示例策略提交到 Control Plane API 并验证响应来独立测试。

**Acceptance Scenarios**:

1. **Given** 用户查看 `delay-policy.yaml` 示例，**When** 用户阅读文件内容，**Then** 文件包含 `spec.selector` 字段，指定目标服务和命名空间
2. **Given** 用户查看 `abort-policy.yaml` 示例，**When** 用户阅读文件内容，**Then** 文件包含 `spec.selector` 字段，并有注释说明通配符 `*` 的用法
3. **Given** 用户查看任意策略示例，**When** 用户使用 `hfi-cli` 提交策略，**Then** Control Plane 接受策略并返回成功响应

---

### User Story 2 - 创建 K8s 环境下的基础验证脚本 (Priority: P1)

作为一名开发者，我希望有一个脚本能够验证故障注入的端到端流程，包括策略创建、策略传播、故障触发和结果验证。

**Why this priority**: 验证脚本是确保系统正确工作的关键工具，对于开发和运维都至关重要。

**Independent Test**: 在已部署 BOIFI 组件的 k3s 集群上运行脚本，观察输出结果。

**Acceptance Scenarios**:

1. **Given** k3s 集群已部署 Control Plane 和 WasmPlugin，**When** 用户运行验证脚本，**Then** 脚本自动检查前置条件（Control Plane 运行中、WasmPlugin 已部署）
2. **Given** 脚本创建一个 abort 策略（503），**When** 向目标服务发送请求，**Then** 脚本报告收到 503 响应的百分比与策略的 `percentage` 字段一致（±10% 容差）
3. **Given** 脚本创建一个 delay 策略（500ms），**When** 向目标服务发送请求，**Then** 脚本报告请求延迟时间符合预期

---

### User Story 3 - 服务选择器精确匹配验证 (Priority: P2)

作为一名 SRE 工程师，我希望有脚本验证服务选择器的精确匹配功能，确保策略只影响目标服务，不影响其他服务。

**Why this priority**: 服务选择器是多服务环境下的核心差异化功能，必须被验证。

**Independent Test**: 创建针对特定服务的策略，同时测试目标服务和非目标服务的请求。

**Acceptance Scenarios**:

1. **Given** 存在 `frontend` 和 `productcatalog` 两个服务，**When** 创建仅针对 `frontend` 的 abort 策略，**Then** `frontend` 的请求返回 503，`productcatalog` 的请求正常返回 200
2. **Given** 创建通配符 `*` 策略，**When** 向任意服务发送请求，**Then** 所有服务都受到故障注入影响
3. **Given** 创建命名空间级别策略（`service: "*", namespace: "demo"`），**When** 向 demo 命名空间的服务发送请求，**Then** 该命名空间的所有服务都受影响

---

### User Story 4 - 更新 README 文档 (Priority: P2)

作为一名新用户，我希望 README 文档清晰说明新的策略格式和验证脚本的使用方法。

**Why this priority**: 良好的文档可以减少用户的学习成本和支持负担。

**Independent Test**: 新用户按照 README 的说明能够成功创建和验证策略。

**Acceptance Scenarios**:

1. **Given** 用户阅读 README，**When** 查看 "Service Selector" 章节，**Then** 章节清晰解释 `selector` 字段的用法和匹配规则
2. **Given** 用户阅读 README，**When** 查看 "Validation Scripts" 章节，**Then** 章节列出所有可用脚本及其用途
3. **Given** 用户按照 README 的 Quick Start 步骤操作，**When** 完成所有步骤，**Then** 用户成功验证了一次故障注入

---

### User Story 5 - 提供完整的微服务场景示例 (Priority: P3)

作为一名 SRE 工程师，我希望有一套完整的示例，展示在真实微服务场景（如 Online Boutique）中如何使用 BOIFI 进行故障注入测试。

**Why this priority**: 完整的场景示例有助于用户理解系统在实际环境中的应用，但不是核心功能。

**Independent Test**: 示例文件可以直接在部署了 Online Boutique 的集群上应用和验证。

**Acceptance Scenarios**:

1. **Given** 用户查看微服务场景示例目录，**When** 阅读示例内容，**Then** 示例包含针对不同服务（frontend, checkout, payment）的策略
2. **Given** 用户应用场景示例，**When** 运行验证脚本，**Then** 所有策略都按预期生效

---

### Edge Cases

- 当 Envoy 无法获取服务身份元数据时，系统应使用通配符匹配（fail-open 模式）
- 当策略没有 `selector` 字段时，应向后兼容，默认匹配所有服务
- 当目标服务不存在时，策略创建应成功，但不会有实际效果
- 当 Control Plane 不可达时，Wasm 插件应使用最后已知的策略配置

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 所有策略示例文件 MUST 包含 `spec.selector` 字段，明确指定目标服务和命名空间
- **FR-002**: 示例文件 MUST 包含注释，解释每个字段的含义和可选值
- **FR-003**: 验证脚本 MUST 在运行前检查前置条件（kubectl 可用、Control Plane 运行中、WasmPlugin 已部署）
- **FR-004**: 验证脚本 MUST 支持指定目标命名空间（默认 `demo`）和 Control Plane 命名空间（默认 `boifi`）
- **FR-005**: 验证脚本 MUST 在测试完成后清理创建的策略
- **FR-006**: 验证脚本 MUST 输出清晰的测试结果摘要（通过/失败/跳过）
- **FR-007**: README MUST 包含 "Service Selector"、"Validation Scripts"、"Quick Start" 三个章节
- **FR-008**: 示例目录 MUST 按功能分类组织（basic/、advanced/、scenarios/）

### Key Entities

- **FaultInjectionPolicy**: 故障注入策略，包含 metadata（名称）、spec（选择器和规则）
- **ServiceSelector**: 服务选择器，包含 service（服务名）和 namespace（命名空间），支持通配符 `*`
- **Rule**: 规则，包含 match（匹配条件）和 fault（故障动作）
- **ValidationScript**: 验证脚本，用于在 K8s 环境中测试故障注入效果

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% 的策略示例文件包含有效的 `selector` 字段，并能被 Control Plane 成功接受
- **SC-002**: 验证脚本能在 3 分钟内完成端到端测试流程（策略创建→传播→验证→清理）
- **SC-003**: 服务选择器精确匹配测试的准确率达到 100%（目标服务被影响，非目标服务不受影响）
- **SC-004**: 新用户按照 README 说明可以在 10 分钟内完成首次故障注入验证
- **SC-005**: 验证脚本在 CI/CD 环境中可以无人值守运行，返回正确的退出码（0=成功，非0=失败）

## Assumptions

- k3s 集群已安装并配置了 Istio
- demo 命名空间已启用 istio-injection
- BOIFI Control Plane 已部署到 boifi 命名空间
- WasmPlugin CRD 已部署到 demo 命名空间
- 用户有 kubectl 访问集群的权限
- 测试服务（如 frontend, productcatalog）已部署在 demo 命名空间

## Out of Scope

- Wasm 插件本身的代码修改（插件已支持 ServiceSelector）
- Control Plane 的代码修改（API 已支持 selector 字段）
- 新的故障类型（abort/delay 已足够）
- 自动化部署脚本（已在 007-istio-k8s-deployment 中完成）
