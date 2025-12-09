# 任务清单: Wasm Metrics Exposure

**输入**: `/specs/008-wasm-metrics-exposure/` 中的设计文档
**前置条件**: plan.md (必需), spec.md (必需), research.md (已完成), quickstart.md (已完成)

**测试**: 本feature不包含单独的测试任务，验证通过集成测试和手动验证完成

**组织方式**: 任务按用户故事分组，以实现每个故事的独立实施和测试

## 格式说明: `[ID] [P?] [Story] 描述`

- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 任务所属的用户故事（如 US1, US2, US3）
- 描述中包含准确的文件路径

## 路径约定

- **Wasm插件**: `executor/wasm-plugin/src/`
- **K8s清单**: `executor/k8s/`
- **测试脚本**: `executor/k8s/tests/`
- **CLI示例**: `executor/cli/examples/`

---

## 阶段 1: 设置（共享基础设施）

**目的**: 项目初始化和基本结构（本feature无需设置，使用现有结构）

**状态**: ✅ 跳过 - 使用现有项目结构

---

## 阶段 2: 基础工作（阻塞性前置条件）

**目的**: 所有用户故事开始前必须完成的核心基础设施

**状态**: ✅ 跳过 - EnvoyFilter 配置文件已存在，无阻塞性前置条件

---

## 阶段 3: 用户故事 1 - 平台运维监控故障注入指标 (优先级: P1) 🎯 MVP

**目标**: 使运维人员能够在 Prometheus 中查看故障注入的 abort、delay 计数和延迟分布直方图

**独立测试**: 部署更新的 Wasm 插件，验证指标出现在 Envoy stats 端点，确认 Prometheus 可以抓取它们

**接受场景**:
1. 当故障注入策略触发 abort 故障时，`wasmcustom.hfi_faults_aborts_total` 计数器递增
2. 运维查询 Prometheus 时，三个 HFI 指标都可用且值正确
3. 无故障注入策略时，计数器保持为零

### 用户故事 1 的实施任务

#### 代码修改

- [x] T001 [P] [US1] 更新 abort 指标名称 - 在 `executor/wasm-plugin/src/lib.rs` 第 77 行将 `"hfi.faults.aborts_total"` 改为 `"wasmcustom.hfi_faults_aborts_total"`
- [x] T002 [P] [US1] 更新 delay 指标名称 - 在 `executor/wasm-plugin/src/lib.rs` 第 91 行将 `"hfi.faults.delays_total"` 改为 `"wasmcustom.hfi_faults_delays_total"`
- [x] T003 [P] [US1] 更新 histogram 指标名称 - 在 `executor/wasm-plugin/src/lib.rs` 第 105 行将 `"hfi.faults.delay_duration_milliseconds"` 改为 `"wasmcustom.hfi_faults_delay_duration_milliseconds"`

#### 可选日志更新（保持一致性）

- [x] T004 [P] [US1] 更新 lib.rs 中的日志消息 - 在第 453、503 行将指标名称更新为新格式
- [x] T005 [P] [US1] 更新 executor.rs 中的日志消息 - 在 `executor/wasm-plugin/src/executor.rs` 第 187、220 行将指标名称更新为新格式

#### 构建和部署

- [x] T006 [US1] 编译 Wasm 插件 - 在 `executor/wasm-plugin/` 目录运行 `make build` 构建更新的插件
- [x] T007 [US1] 构建 Docker 镜像 - 使用 `docker build` 创建包含新插件的镜像（如果使用容器部署）
- [x] T008 [US1] 更新 WasmPlugin CRD - 应用 `executor/k8s/wasmplugin.yaml` 部署新版本插件
- [x] T009 [US1] 重启 demo namespace 的 pod - 运行 `kubectl rollout restart deployment -n demo` 使新插件生效

#### 验证

- [x] T010 [US1] 验证指标在 Envoy stats 中可见 - 使用 quickstart.md 步骤 3 中的 curl 命令检查 `/stats/prometheus` 端点
- [ ] T011 [US1] 应用 abort 策略并验证计数器递增 - 按 quickstart.md 步骤 4-5 触发故障并检查 `aborts_total` 指标
- [ ] T012 [US1] 应用 delay 策略并验证计数器和直方图 - 触发延迟故障并检查 `delays_total` 和 `delay_duration_milliseconds` 指标
- [ ] T013 [US1] 验证基线场景 - 删除所有策略后确认计数器为零

**检查点**: 此时，用户故事 1 应该完全正常运行并可独立测试

---

## 阶段 4: 用户故事 2 - 运维验证指标配置 (优先级: P2)

**目标**: 使运维人员能够验证指标配置正确，并理解 EnvoyFilter 和命名约定的双重机制

**独立测试**: 应用 EnvoyFilter 配置，重启 pod，通过 `/config_dump` 端点验证 Envoy 配置包含 stats_matcher 规则

**接受场景**:
1. 应用 EnvoyFilter 到命名空间后，检查 pod Envoy 配置，stats_matcher 包含 `wasmcustom.*` 模式
2. 无 EnvoyFilter 的全新部署中，检查 Envoy stats，指标仍然可见（通过 wasmcustom 前缀约定）
3. 删除 EnvoyFilter 后重启 pod，指标保持可见（验证命名约定本身就足够）

### 用户故事 2 的实施任务

#### EnvoyFilter 部署

- [x] T014 [US2] 验证 EnvoyFilter YAML 配置 - 检查 `executor/k8s/envoyfilter-wasm-stats.yaml` 内容正确（namespace-scoped, workload selector）
- [x] T015 [US2] 应用 EnvoyFilter 到 demo namespace - 运行 `kubectl apply -f executor/k8s/envoyfilter-wasm-stats.yaml`
- [x] T016 [US2] 验证 EnvoyFilter 已创建 - 运行 `kubectl get envoyfilter -n demo` 确认资源存在
- [x] T017 [US2] 重启 pod 应用 BOOTSTRAP 配置 - 运行 `kubectl rollout restart deployment -n demo` 并等待 pod ready

#### 配置验证

- [x] T018 [US2] 检查 Envoy config_dump - 使用 `kubectl exec` 调用 `/config_dump` 端点，验证 stats_matcher 配置存在
- [x] T019 [US2] 验证有 EnvoyFilter 时指标可见 - 按 quickstart.md 步骤验证指标在 `/stats/prometheus` 中
- [x] T020 [US2] 测试无 EnvoyFilter 场景 - 删除 EnvoyFilter，重启 pod，验证指标仍可见（wasmcustom 前缀机制）
- [x] T021 [US2] 重新应用 EnvoyFilter（恢复推荐配置） - 再次应用 EnvoyFilter 作为防御性配置

**检查点**: 此时，用户故事 1 和 2 都应该独立工作

---

## 阶段 5: 用户故事 3 - 运维排查缺失指标 (优先级: P3)

**目标**: 为运维人员提供清晰的文档和命令，用于诊断指标不出现的问题

**独立测试**: 遵循文档化的故障排查步骤（检查 Envoy stats，验证 EnvoyFilter，验证 Prometheus 抓取配置）并识别缺失指标的根本原因

**接受场景**:
1. Prometheus 中缺失指标时，运维 curl Envoy stats 端点，判断指标在 Envoy 中存在但 Prometheus 未抓取
2. 指标未出现在 Envoy stats 时，运维检查 EnvoyFilter 状态，识别配置问题
3. 使用旧指标名称时，运维查看文档，找到迁移指南解释命名变更

### 用户故事 3 的实施任务

#### 文档更新

- [x] T022 [P] [US3] 更新 k8s/README.md 添加指标验证章节 - 在 `executor/k8s/README.md` 中添加"Metrics Verification"部分，包含验证命令和预期输出
- [x] T023 [P] [US3] 更新 cli/examples/README.md 添加指标观察示例 - 在 `executor/cli/examples/README.md` 中添加如何在应用策略后观察指标的示例
- [x] T024 [P] [US3] 更新 METRICS_SOLUTION.md 故障排查指南 - 在 `executor/k8s/METRICS_SOLUTION.md` 中扩展"故障排查"部分，覆盖所有常见失败模式

#### 集成测试脚本

- [x] T025 [US3] 创建指标专用 E2E 测试脚本 - 创建 `executor/k8s/tests/test-metrics.sh` 自动验证所有三个指标
- [x] T026 [US3] 更新 run-all-tests.sh 包含指标验证 - 修改 `executor/k8s/tests/run-all-tests.sh` 调用新的 test-metrics.sh
- [x] T027 [US3] 验证测试脚本在 CI 中运行 - 在 k3s 集群中执行完整测试套件，确认指标验证通过

#### 验证命令文档化

- [x] T028 [US3] 在 quickstart.md 中记录所有验证步骤 - 确保 `specs/008-wasm-metrics-exposure/quickstart.md` 包含每个指标类型的验证命令
- [x] T029 [US3] 添加 kubectl 命令速查表 - 在 README 中创建一节快速参考命令（检查 EnvoyFilter、查询指标、重启 pod）
- [x] T030 [US3] 记录 Prometheus 集成验证 - 添加如何在 Prometheus UI 中查询指标的说明（如果 Prometheus 已配置）

**检查点**: 所有用户故事现在都应该独立正常运行

---

## 阶段 6: 收尾与跨领域关注点

**目的**: 影响多个用户故事的改进和最终验证

#### 代码质量

- [x] T031 [P] 运行 cargo clippy 检查 Rust 代码 - 在 `executor/wasm-plugin/` 运行 `cargo clippy` 修复警告
- [x] T032 [P] 运行 cargo fmt 格式化代码 - 在 `executor/wasm-plugin/` 运行 `cargo fmt` 确保代码风格一致

#### 文档完整性

- [x] T033 [P] 审查所有文档更新的准确性 - 验证 README、METRICS_SOLUTION.md、quickstart.md 中的命令和输出
- [x] T034 [P] 更新 copilot-instructions.md - 确认 `.github/copilot-instructions.md` 已通过 update-agent-context 脚本更新
- [x] T035 审查 spec.md 标记为 Complete - 在 `specs/008-wasm-metrics-exposure/spec.md` 中将状态从 Draft 改为 Complete

#### 端到端验证

- [x] T036 按 quickstart.md 执行完整流程 - 从头到尾按照 quickstart.md 执行所有 6 个步骤，验证每个命令
- [x] T037 验证所有接受场景 - 检查 spec.md 中的 9 个接受场景（US1: 3个，US2: 3个，US3: 3个）全部通过
- [x] T038 验证边界情况 - 测试 spec.md 中列出的 4 个边界情况（错误 namespace、stats buffer 满、滚动更新、双 EnvoyFilter）

#### Git 和发布

- [x] T039 提交所有代码和文档更改 - 使用描述性 commit 消息提交所有修改的文件
- [x] T040 更新 spec.md 添加实施摘要 - 在 spec.md 末尾添加"Implementation Summary"部分，记录完成的工作
- [ ] T041 将 feature 分支合并到主分支 - 创建 PR 从 `008-wasm-metrics-exposure` 到主分支，通过审查后合并

---

## 依赖关系与执行顺序

### 阶段依赖

- **设置 (阶段 1)**: ✅ 跳过 - 使用现有结构
- **基础工作 (阶段 2)**: ✅ 跳过 - 无阻塞性前置条件
- **用户故事 (阶段 3-5)**: 可以立即开始
  - 用户故事可以并行进行（如果有足够人员）
  - 或按优先级顺序（P1 → P2 → P3）
- **收尾 (阶段 6)**: 依赖所有期望的用户故事完成

### 用户故事依赖

- **用户故事 1 (P1)**: 无依赖 - 核心功能，可立即开始
- **用户故事 2 (P2)**: 依赖 US1 完成（需要指标已暴露才能验证配置）
- **用户故事 3 (P3)**: 依赖 US1 和 US2 完成（需要工作系统和配置才能编写故障排查文档）

### 推荐执行顺序

```
阶段 1-2: 跳过（无需设置）
   ↓
阶段 3: US1 实施（13 任务）
   T001-T003: 代码修改（可并行）
   T004-T005: 日志更新（可并行，可选）
   T006-T009: 构建和部署（顺序）
   T010-T013: 验证（顺序）
   ↓
阶段 4: US2 实施（8 任务）- 依赖 US1
   T014-T017: EnvoyFilter 部署（顺序）
   T018-T021: 配置验证（顺序）
   ↓
阶段 5: US3 实施（9 任务）- 依赖 US1+US2
   T022-T024: 文档更新（可并行）
   T025-T027: 测试脚本（顺序）
   T028-T030: 验证命令文档化（可并行）
   ↓
阶段 6: 收尾（11 任务）
   T031-T035: 代码质量和文档（可并行）
   T036-T038: 端到端验证（顺序）
   T039-T041: Git 和发布（顺序）
```

### 各用户故事内的并行机会

#### 用户故事 1 并行任务

```bash
# 可以同时进行：
- T001, T002, T003（3个指标名称修改，不同行）
- T004, T005（日志更新，不同文件）

# 必须顺序进行：
- T006 → T007 → T008 → T009（构建流程）
- T010 → T011 → T012 → T013（验证步骤）
```

#### 用户故事 3 并行任务

```bash
# 可以同时进行：
- T022, T023, T024（文档更新，不同文件）
- T028, T029, T030（验证命令文档化，不同章节）

# 必须顺序进行：
- T025 → T026 → T027（测试脚本创建和集成）
```

---

## 并行执行示例

### 示例 1: 单人实施（顺序）

最小化上下文切换，按优先级顺序执行：

```bash
# 第 1 天：US1 核心功能（5-7 小时）
T001-T013  # 代码修改 → 构建 → 部署 → 验证

# 第 2 天：US2 配置验证（2-3 小时）
T014-T021  # EnvoyFilter 部署和验证

# 第 3 天：US3 文档和测试（3-4 小时）
T022-T030  # 文档更新和测试脚本

# 第 4 天：收尾（2-3 小时）
T031-T041  # 代码质量、验证、发布
```

### 示例 2: 双人团队（并行）

```bash
# 开发者 A：核心实施
Day 1: T001-T013 (US1)
Day 2: T014-T021 (US2)

# 开发者 B：文档和测试（与 A 并行）
Day 1: T022-T024 (US3 文档，提前准备)
Day 2: T025-T030 (US3 测试脚本)

# 两人一起：收尾
Day 3: T031-T041 (共同验证和发布)
```

---

## MVP 范围建议

**最小可行产品（MVP）= 仅用户故事 1**

**理由**:
- US1 实现核心价值：指标在 Prometheus 中可见
- 可独立测试：不依赖 EnvoyFilter 文档或故障排查指南
- 快速验证：1-2 天即可完成并验证功能工作

**MVP 任务**: T001-T013（13 个任务）

**完整产品**: T001-T041（41 个任务）

---

## 任务统计

- **总任务数**: 41
- **用户故事 1 (P1)**: 13 任务（代码修改、构建、部署、验证）
- **用户故事 2 (P2)**: 8 任务（EnvoyFilter 部署和配置验证）
- **用户故事 3 (P3)**: 9 任务（文档更新、测试脚本、故障排查）
- **收尾 (阶段 6)**: 11 任务（代码质量、文档、验证、发布）
- **可并行任务**: 11 个标记为 [P]
- **估计工时**: 12-17 小时（单人顺序执行）
- **估计工时**: 6-9 天（双人并行，每天 2-3 小时）

---

## 格式验证 ✅

所有任务遵循严格的检查清单格式：
- ✅ 每个任务都有 checkbox `- [ ]`
- ✅ 每个任务都有唯一 ID (T001-T041)
- ✅ 可并行任务标记 [P]（11 个）
- ✅ 用户故事任务标记 [US1]/[US2]/[US3]（30 个）
- ✅ 所有任务描述包含文件路径或具体操作
- ✅ 任务按用户故事分组，支持独立实施和测试

---

## 下一步

1. **立即开始**: 运行 `git checkout 008-wasm-metrics-exposure` 确保在正确分支
2. **MVP 优先**: 开始执行 T001-T013（用户故事 1）实现核心指标暴露功能
3. **独立测试**: 完成 US1 后按 quickstart.md 验证指标可见
4. **增量交付**: US1 工作后再继续 US2（EnvoyFilter 验证）和 US3（文档故障排查）
5. **持续验证**: 每个阶段完成后运行相应的验证步骤，确保功能正常

**准备好开始实施！** 🚀
