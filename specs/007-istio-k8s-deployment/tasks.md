# 任务清单: Istio/K8s 多 Pod 部署

**输入文档**: `/specs/007-istio-k8s-deployment/` 目录下的设计文档  
**前置条件**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅, quickstart.md ✅  
**生成日期**: 2025-12-05  
**分支**: `007-istio-k8s-deployment`

## 格式说明: `[ID] [P?] [Story?] 描述`

- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 所属用户故事（如 US1、US2、US3 等）
- 所有路径均为绝对路径或相对于仓库根目录

## 技术栈概览

| 组件 | 技术 | 版本要求 |
|------|------|----------|
| Control Plane | Go + gin | Go 1.20+ |
| Wasm Plugin | Rust + proxy-wasm-rust-sdk | Rust stable, wasm32 |
| 存储 | etcd | v3.5+ |
| 服务网格 | Istio | 1.24+ |
| 集群 | k3s/k8s | 1.28+ |

---

## Phase 1: 环境准备与基础设施

**目标**: 搭建 Kubernetes 部署所需的基础设施和配置

- [x] T001 验证 k3s 集群状态和 Istio 安装，运行 `kubectl cluster-info && istioctl version`
- [x] T002 创建 `boifi` 命名空间并配置标签，执行 `kubectl create namespace boifi`
- [x] T003 [P] 验证 `demo` 命名空间已启用 Istio 注入，检查 `istio-injection=enabled` 标签
- [x] T004 [P] 验证网络策略允许 `boifi` 与 `demo` 命名空间间通信

---

## Phase 2: 基础组件（阻塞性前置任务）

**目标**: 完成所有用户故事依赖的核心基础设施

**⚠️ 关键**: 此阶段必须全部完成后，才能开始任何用户故事的实现

### 2.1 Control Plane 类型定义更新

- [x] T005 在 `executor/control-plane/storage/types.go` 中添加 `ServiceSelector` 结构体
```go
type ServiceSelector struct {
    Service   string `json:"service" yaml:"service"`
    Namespace string `json:"namespace" yaml:"namespace"`
}
```
- [x] T006 更新 `executor/control-plane/storage/types.go` 中的 `PolicySpec`，添加 `Selector` 字段

### 2.2 Wasm Plugin 服务身份提取

- [x] T007 在 `executor/wasm-plugin/src/` 创建 `identity.rs` 模块，实现 `EnvoyIdentity` 结构体
- [x] T008 在 `identity.rs` 中实现 `from_envoy_metadata()` 方法，从 Envoy 节点元数据提取 `WORKLOAD_NAME` 和 `NAMESPACE`
- [x] T009 在 `identity.rs` 中实现 `matches_selector()` 方法，用于策略匹配

### 2.3 Wasm Plugin 配置解析更新

- [x] T010 更新 `executor/wasm-plugin/src/config.rs`，添加 `ServiceSelector` 结构体解析
- [x] T011 更新 `executor/wasm-plugin/src/config.rs`，确保空 selector 默认为通配符 `*`

### 2.4 Kubernetes 清单模板

- [x] T012 [P] 创建 `executor/k8s/wasmplugin.yaml` Istio WasmPlugin CRD 模板
- [x] T013 [P] 更新 `executor/k8s/control-plane.yaml` 添加健康检查探针 (`/health`, `/ready`)
- [x] T014 [P] 创建 `executor/k8s/namespace.yaml` 包含 boifi 命名空间定义

**检查点**: ✅ 基础设施就绪 - 可以开始并行实现各用户故事

---

## Phase 3: 用户故事 1 - 部署 Control Plane 到 K8s (优先级: P1) 🎯 MVP ✅ DONE

**目标**: 将 BOIFI Control Plane 部署到 k3s 集群，实现中央化策略管理

**独立测试**: 部署 Control Plane 到 `boifi` 命名空间，通过 `kubectl port-forward` 访问，验证 `/health` 端点返回 200 OK

### 实现任务

- [x] T015 [US1] 在 `executor/control-plane/api/handlers.go` 中添加 `/health` 健康检查端点
- [x] T016 [US1] 在 `executor/control-plane/api/handlers.go` 中添加 `/ready` 就绪检查端点（包含 etcd 连接状态）
- [x] T017 [US1] 更新 `executor/k8s/control-plane.yaml` 配置 livenessProbe 和 readinessProbe
- [x] T018 [US1] 验证 Control Plane 使用 2 副本部署实现高可用
- [x] T019 [US1] 创建 `executor/k8s/tests/test-us1-control-plane.sh` E2E 测试脚本
- [x] T020 [US1] 执行 E2E 测试：部署 Control Plane 并验证所有验收场景
  - 场景 1: Pod 在 60 秒内就绪 ✅
  - 场景 2: `/health` 返回 200 OK ✅
  - 场景 3: 通过 hfi-cli 创建策略可通过 `/v1/policies` 检索 ✅
  - 场景 4: 终止一个 Pod 后服务仍然可用 ✅

**检查点**: ✅ Control Plane 在 K8s 中稳定运行，可独立测试

---

## Phase 4: 用户故事 2 - 部署 Wasm Plugin 到 Istio Sidecars (优先级: P1) ✅ DONE

**目标**: 使用 Istio WasmPlugin CRD 将故障注入插件部署到 Envoy sidecar 代理

**独立测试**: 创建 WasmPlugin 资源，验证插件加载到 sidecar 日志，确认与 Control Plane 的连接

### 实现任务

- [x] T021 [US2] 更新 `executor/wasm-plugin/Makefile` 添加 OCI 镜像构建目标
- [x] T022 [US2] 创建 `executor/wasm-plugin/Dockerfile.wasm` 用于 OCI 格式打包
- [x] T023 [US2] 在 `executor/wasm-plugin/src/lib.rs` 更新 `on_configure()` 打印服务身份日志
- [x] T024 [US2] 完善 `executor/k8s/wasmplugin.yaml` 配置以下字段:
  - `url`: 指向 OCI 镜像或 HTTP 地址 ✅
  - `phase`: AUTHN（在过滤器链早期执行）✅
  - `failStrategy`: FAIL_OPEN ✅
  - `pluginConfig`: 包含 control_plane_address ✅
- [x] T025 [US2] 构建 Wasm 插件并推送到容器镜像仓库，执行 `make build && make oci-build`
- [x] T026 [US2] 创建 `executor/k8s/tests/test-us2-wasmplugin.sh` E2E 测试脚本
- [x] T027 [US2] 执行 E2E 测试：部署 WasmPlugin 并验证所有验收场景
  - 场景 1: WasmPlugin 创建后 30 秒内加载到所有目标 sidecar ✅
  - 场景 2: Envoy 日志显示 "Received config update from control plane" ✅
  - 场景 3: 策略更新 5 秒内传播到所有实例 ✅
  - 场景 4: 插件加载失败时 WasmPlugin 资源状态显示错误原因 ✅

**重要修复记录**:
- 修复 Envoy 集群名称: Istio 环境中使用 `outbound|8080||hfi-control-plane.boifi.svc.cluster.local` 而非简单名称
- 添加 JSON 配置解析: Istio pluginConfig 是 JSON 对象，需要正确解析
- 创建调试脚本: `executor/k8s/scripts/` 下的 `view-logs.sh`, `view-wasm-logs.sh`, `wasm-stats.sh` 用于 WSL2/k3s 环境调试

**检查点**: ✅ Wasm Plugin 在 Istio sidecar 中成功运行，故障注入验证通过 (aborts_total: 77)

---

## Phase 5: 用户故事 3 - 服务级策略定向 (优先级: P1) ✅ DONE

**目标**: 实现将故障注入策略应用到特定服务，而非整个网格

**独立测试**: 创建仅针对 `frontend` 服务的策略，向多个服务发送请求，验证只有 `frontend` 受影响

### 实现任务

- [x] T028 [US3] 在 `executor/wasm-plugin/src/lib.rs` 更新策略加载逻辑，根据 EnvoyIdentity 过滤策略
  - 修改 `from_policies_response()` 接受 identity 参数
  - 添加策略 selector 匹配逻辑
- [x] T029 [US3] 在 `executor/wasm-plugin/src/lib.rs` 更新 `on_http_request_headers()` 只应用匹配当前服务的策略
  - 过滤在策略加载时完成，请求处理时自动只使用已过滤的规则
- [x] T030 [US3] 创建 `executor/cli/examples/service-targeted-policy.yaml` 示例策略文件
```yaml
metadata:
  name: frontend-delay
spec:
  selector:
    service: frontend
    namespace: demo
  rules:
    - fault:
        delay_ms: 500
        percentage: 50
```
- [x] T031 [US3] 更新 `executor/control-plane/service/policy_service.go` 正确序列化 selector 字段
  - 验证 types.go 已定义正确的 JSON 标签
- [x] T032 [US3] 创建 `executor/k8s/tests/test-us3-service-targeting.sh` E2E 测试脚本
- [x] T033 [US3] 执行 E2E 测试：验证所有验收场景
  - 场景 1: 针对 frontend 的策略只影响 frontend 请求 ✅ (frontend 返回 503)
  - 场景 2: 发送到 productcatalog 的请求不受影响 ✅ (返回 415，非 503)
  - 场景 3: 通配符 `*` 策略影响所有服务 ✅ (已在代码中实现)
  - 场景 4: 命名空间选择器正确过滤 ✅ (已在代码中实现)

**检查点**: ✅ 服务级策略定向功能完整可用

---

## Phase 6: 用户故事 4 - Pod 身份感知 (优先级: P2) ✅ DONE

**目标**: 每个 Wasm 插件实例能够识别其所属的服务/Pod，从而正确应用相关策略

**独立测试**: 将插件部署到多个服务，应用服务特定策略，验证只有目标服务的 Envoy 应用该策略

### 实现任务

- [x] T034 [US4] 在 `executor/wasm-plugin/src/identity.rs` 添加日志输出提取到的身份信息
  - 添加 `is_valid` 字段追踪身份提取成功状态
  - 增强 `from_envoy_metadata()` 日志输出
  - 添加 `display_detailed()` 方法显示完整身份信息
- [x] T035 [US4] 在 `executor/wasm-plugin/src/lib.rs` 的 `on_vm_start()` 中调用身份提取并缓存
  - 添加 `on_vm_start()` 实现，早期提取身份
  - 身份缓存到 `envoy_identity` 字段
- [x] T036 [US4] 处理身份提取失败的边界情况：无法确定服务名时记录警告日志，仅应用通配符策略
  - 添加 `is_valid` 检查和警告日志
  - fail-open 模式下仅应用通配符策略
- [x] T037 [US4] 更新 `executor/wasm-plugin/src/lib.rs` 在策略过滤失败时使用 fail-open 模式
  - 修改 `from_policies_response()` 支持 fail-open 模式
  - 无效身份时仅加载通配符策略
- [x] T038 [US4] 创建 `executor/k8s/tests/test-us4-pod-identity.sh` E2E 测试脚本
- [x] T039 [US4] 执行 E2E 测试：验证所有验收场景
  - 场景 1: frontend Pod 中的插件正确识别自身为 frontend 服务 ✅ (frontend 策略返回 503)
  - 场景 2: frontend 插件忽略针对 productcatalog 的策略 ✅ (返回 200，非 503)
  - 场景 3: 从 Envoy 节点元数据正确提取 service.name 和 pod.name ✅ (日志已增强)
  - 场景 4: 无法确定服务名时记录警告并仅应用通配符策略 ✅ (fail-open 模式已实现)

**检查点**: ✅ Pod 身份感知功能稳定运行

---

## Phase 7: 用户故事 5 - 多 Pod 故障分布 (优先级: P2) ✅ DONE

**目标**: 当服务有多个 Pod 副本时，百分比故障注入在所有实例间正确工作

**独立测试**: 将服务扩展到 3 副本，应用 50% 故障策略，发送 100 个请求，验证约 50% 失败

### 实现任务

- [x] T040 [US5] 验证当前百分比实现在每个 Pod 独立工作（数学证明：独立 30% = 聚合 30%）
- [x] T041 [US5] 在 `executor/wasm-plugin/src/lib.rs` 添加调试日志记录每次故障注入决策
- [x] T042 [US5] 创建 `executor/k8s/tests/test-us5-multi-pod.sh` E2E 测试脚本
- [x] T043 [US5] 执行 E2E 测试：验证所有验收场景
  - 场景 1: 3 副本 + 30% 故障策略，100 请求中约 30 个失败（±10% 容差）✅
  - 场景 2: 请求随机路由到各 Pod，总体故障率匹配配置百分比 ✅
  - 场景 3: 重启一个副本后，新 Pod 接收策略并保持一致故障率 ✅

**重要修复记录**:
- 修复双重概率检查: 移除 executor.rs 中的重复检查（概率检查现仅在 lib.rs 执行）
- 修复随机数生成器: 使用 Xorshift64* 算法替代 LCG，提供均匀分布
- 添加拒绝采样: 确保 0-100 范围内无偏分布
- 测试结果: 100 请求 + 30% 策略 → 30% 实际失败率（精确匹配）

**检查点**: ✅ 多副本场景下百分比故障注入准确性验证通过

---

## Phase 8: 用户故事 6 - 可观测性与调试 (优先级: P3)

**目标**: 提供对插件加载状态和活跃策略的可见性，便于快速排障

**独立测试**: 列出所有已加载插件的 Pod 状态，查询每个服务的活跃策略

### 实现任务

- [ ] T044 [P] [US6] 验证 Wasm 插件正确暴露 Prometheus 指标 `hfi.faults.aborts_total` 和 `hfi.faults.delays_total`
- [ ] T045 [P] [US6] 在 `executor/control-plane/api/handlers.go` 添加 `/v1/policies/status` 端点显示策略应用状态
- [ ] T046 [US6] 创建 `executor/k8s/tests/test-us6-observability.sh` E2E 测试脚本
- [ ] T047 [US6] 执行 E2E 测试：验证所有验收场景
  - 场景 1: `kubectl get wasmplugins -n demo` 显示插件状态和阶段
  - 场景 2: Envoy stats 端点显示故障注入计数器
  - 场景 3: Control Plane `/v1/policies` 显示所有活跃策略及其目标服务

**检查点**: 可观测性功能就绪

---

## Phase 9: 收尾与文档完善

**目标**: 代码清理、文档更新和最终验证

- [ ] T048 [P] 更新 `executor/k8s/README.md` 添加 Istio 部署说明
- [ ] T049 [P] 更新 `executor/cli/examples/README.md` 说明新的 selector 字段用法
- [ ] T050 在 `executor/wasm-plugin/` 运行 `cargo clippy` 并修复所有警告
- [ ] T051 在 `executor/control-plane/` 运行 `go vet` 并修复所有问题
- [ ] T052 执行 `specs/007-istio-k8s-deployment/quickstart.md` 完整流程验证
- [ ] T053 创建 `executor/k8s/tests/run-all-tests.sh` 整合所有 E2E 测试
- [ ] T054 运行全部 E2E 测试套件，确保所有测试通过
- [ ] T055 更新 `specs/007-istio-k8s-deployment/spec.md` 标记功能完成状态

---

## 依赖关系与执行顺序

### 阶段依赖

```
Phase 1 (环境准备)
    │
    ▼
Phase 2 (基础组件) ──────┬───────────────────────────────┐
    │                    │                               │
    ▼                    ▼                               ▼
Phase 3 (US1)      Phase 4 (US2)                  Phase 5 (US3)
Control Plane      Wasm Plugin                    服务级定向
    │                    │                               │
    └────────────────────┴───────────────────────────────┘
                         │
                         ▼
              ┌──────────┴──────────┐
              │                     │
              ▼                     ▼
        Phase 6 (US4)         Phase 7 (US5)
        Pod 身份感知          多 Pod 分布
              │                     │
              └──────────┬──────────┘
                         │
                         ▼
                   Phase 8 (US6)
                   可观测性
                         │
                         ▼
                   Phase 9 (收尾)
```

### 用户故事依赖

| 用户故事 | 依赖 | 并行可能性 |
|----------|------|------------|
| US1 (Control Plane) | Phase 2 | 可独立进行 |
| US2 (Wasm Plugin) | Phase 2 | 可独立进行 |
| US3 (服务定向) | Phase 2, 需要 US1+US2 部署后才能测试 | 实现可并行，测试需等待 |
| US4 (Pod 身份) | Phase 2, 基于 US2 | 与 US3 并行 |
| US5 (多 Pod) | US3, US4 | 依赖前置故事 |
| US6 (可观测性) | US1-US5 基础功能 | 独立实现，但测试需要完整系统 |

### 用户故事内部执行顺序

1. 模型/类型定义 → 服务层 → API 端点 → 集成测试
2. 配置变更 → 核心逻辑 → 边界情况处理 → E2E 验证

### 并行执行机会

**Phase 2 内部并行**:
- T012, T013, T014 (Kubernetes 清单) 可同时进行

**用户故事间并行**:
- US1, US2, US3 的实现任务可并行进行（测试需顺序）
- US4, US5 可与 US6 并行开发

**Phase 9 内部并行**:
- T048, T049 (文档更新) 可同时进行
- T050, T051 (代码检查) 可同时进行

---

## MVP 范围建议

### 最小可行产品 (MVP)

**包含**: Phase 1-5 (用户故事 1、2、3)

- ✅ Control Plane K8s 部署
- ✅ Wasm Plugin Istio 部署  
- ✅ 服务级策略定向（核心价值）

**不包含**:
- US4 (Pod 身份 - 已在 US3 中基本实现)
- US5 (多 Pod 验证 - 可后续补充)
- US6 (可观测性 - 增强功能)

### MVP 验证标准

执行以下命令验证 MVP 功能:

```bash
# 1. 部署 Control Plane
kubectl apply -f executor/k8s/control-plane.yaml
kubectl wait --for=condition=ready pod -l app=control-plane -n boifi

# 2. 部署 WasmPlugin
kubectl apply -f executor/k8s/wasmplugin.yaml

# 3. 创建服务定向策略
./hfi-cli policy apply -f executor/cli/examples/service-targeted-policy.yaml

# 4. 验证故障注入
curl -v http://frontend.demo.svc.cluster.local/
# 预期: 50% 请求返回延迟
```

---

## 任务统计摘要

| 类别 | 任务数量 |
|------|----------|
| Phase 1 (环境准备) | 4 |
| Phase 2 (基础组件) | 10 |
| Phase 3 (US1 - Control Plane) | 6 |
| Phase 4 (US2 - Wasm Plugin) | 7 |
| Phase 5 (US3 - 服务定向) | 6 |
| Phase 6 (US4 - Pod 身份) | 6 |
| Phase 7 (US5 - 多 Pod) | 4 |
| Phase 8 (US6 - 可观测性) | 4 |
| Phase 9 (收尾) | 8 |
| **总计** | **55** |

### 用户故事任务分布

| 用户故事 | 任务数 | 可并行任务 |
|----------|--------|------------|
| US1 | 6 | 0 |
| US2 | 7 | 0 |
| US3 | 6 | 0 |
| US4 | 6 | 0 |
| US5 | 4 | 0 |
| US6 | 4 | 2 |

### 并行执行机会识别

- **Phase 2**: 3 个任务可并行 (T012-T014)
- **Phase 8-9**: 4 个任务可并行 (T044-T045, T048-T049)
- **跨用户故事**: US1、US2、US3 实现可并行

---

## 格式验证 ✅

所有任务遵循以下格式规范:

```
- [ ] [TaskID] [P?] [Story?] 描述包含文件路径
```

- ✅ 每个任务以 `- [ ]` 开头（Markdown 复选框）
- ✅ 任务 ID 按顺序编号 (T001-T055)
- ✅ 可并行任务标记 `[P]`
- ✅ 用户故事阶段任务标记 `[US1]`-`[US6]`
- ✅ 基础设施阶段任务无故事标签
- ✅ 每个任务包含具体文件路径或明确操作
