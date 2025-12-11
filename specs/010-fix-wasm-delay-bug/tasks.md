# Tasks: Fix WASM Plugin Delay Fault Bug

**Input**: Design documents from `/specs/010-fix-wasm-delay-bug/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅

**Tests**: 包含 E2E 验证（使用 validate-basic.sh），按 Constitution IV (测试驱动) 要求。

**Organization**: 任务按用户故事组织，支持独立实现和测试。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 任务所属用户故事 (US1, US2, US3)
- 包含确切文件路径

---

## Phase 1: Setup (准备工作)

**Purpose**: 确认当前状态，准备开发环境

- [ ] T001 确认分支为 `010-fix-wasm-delay-bug` 并同步最新代码
- [ ] T002 [P] 阅读并理解当前 delay 实现在 `executor/wasm-plugin/src/lib.rs`
- [ ] T003 [P] 阅读并理解 `executor/wasm-plugin/src/config.rs` 中 DelayAction 结构

---

## Phase 2: Foundational (基础变更)

**Purpose**: 核心数据结构变更，所有用户故事都依赖此阶段

**⚠️ CRITICAL**: 用户故事实现必须等此阶段完成

- [ ] T004 修改 `DelayAction` 结构体：将 `fixed_delay: String` 改为 `fixed_delay_ms: u64` 在 `executor/wasm-plugin/src/config.rs`
- [ ] T005 删除 `parsed_duration_ms` 字段从 `DelayAction` 在 `executor/wasm-plugin/src/config.rs`
- [ ] T006 删除 `parse_duration` 函数在 `executor/wasm-plugin/src/config.rs`
- [ ] T007 删除 `test_parse_duration` 测试在 `executor/wasm-plugin/src/config.rs`
- [ ] T008 添加 `MAX_DELAY_MS` 常量 (30000) 在 `executor/wasm-plugin/src/config.rs`
- [ ] T009 更新预处理逻辑：移除 delay duration 解析代码在 `executor/wasm-plugin/src/config.rs`
- [ ] T010 运行 `cargo check` 确认编译通过在 `executor/wasm-plugin/`

**Checkpoint**: 数据结构变更完成，编译通过

---

## Phase 3: User Story 1 - Delay Fault Injection Works (Priority: P1) 🎯 MVP

**Goal**: 修复 delay 故障注入失败的 Bug，使用有效集群实现延迟

**Independent Test**: 应用 500ms delay 策略后，请求响应时间增加约 500ms

### Implementation for User Story 1

- [ ] T011 [US1] 修复 `dispatch_http_call` 调用：将 `"hfi_delay_cluster"` 替换为 `CONTROL_PLANE_CLUSTER` 在 `executor/wasm-plugin/src/lib.rs` (约第 728 行)
- [ ] T012 [US1] 更新 delay 读取逻辑：直接使用 `delay.fixed_delay_ms` 替代 `delay.parsed_duration_ms` 在 `executor/wasm-plugin/src/lib.rs`
- [ ] T013 [US1] 添加最大延迟限制 (clamp to MAX_DELAY_MS) 在 delay 执行前 `executor/wasm-plugin/src/lib.rs`
- [ ] T014 [US1] 更新 `execute_delay` 函数签名和实现在 `executor/wasm-plugin/src/executor.rs`
- [ ] T015 [US1] 处理零延迟情况 (`fixed_delay_ms == 0` 跳过故障注入) 在 `executor/wasm-plugin/src/lib.rs`
- [ ] T016 [US1] 运行 `cargo build --target wasm32-unknown-unknown --release` 构建新 plugin.wasm
- [ ] T017 [US1] 复制新 plugin.wasm 到测试目录 `/tmp/wasm-plugin/`
- [ ] T018 [US1] 重启 WasmPlugin 使新代码生效 (kubectl rollout restart)

**Checkpoint**: Delay 故障注入功能恢复正常

---

## Phase 4: User Story 2 - Simplified Configuration Format (Priority: P2)

**Goal**: 更新所有配置文件使用新的 `fixed_delay_ms` 格式

**Independent Test**: 使用新格式配置文件应用策略成功，validate-basic.sh 通过

### Implementation for User Story 2

- [ ] T019 [US2] 更新 `executor/cli/examples/basic/delay-policy.yaml`：`fixed_delay: "1000ms"` → `fixed_delay_ms: 1000`
- [ ] T020 [P] [US2] 更新 `executor/cli/examples/basic/percentage-policy.yaml`：`fixed_delay: "500ms"` → `fixed_delay_ms: 500`
- [ ] T021 [P] [US2] 更新 `executor/cli/examples/advanced/header-policy.yaml`：`fixed_delay: "800ms"` → `fixed_delay_ms: 800`
- [ ] T022 [P] [US2] 更新 `executor/cli/examples/advanced/time-limited-policy.yaml`：`fixed_delay: "500ms"` → `fixed_delay_ms: 500`
- [ ] T023 [P] [US2] 更新 `executor/cli/examples/advanced/late-stage-policy.yaml`：所有 fixed_delay 字段
- [ ] T024 [P] [US2] 更新 `executor/cli/examples/advanced/service-targeted-policy.yaml`：所有 fixed_delay 字段 (3处)
- [ ] T025 [US2] 更新 CLI types：`executor/cli/types/policy.go` 中 `DelayAction` 结构体
- [ ] T026 [US2] 更新 Control Plane types：`executor/control-plane/api/types.go` 中 `DelayAction` 结构体 (如果存在)
- [ ] T027 [US2] 更新 README 文档示例在 `executor/cli/examples/README.md`
- [ ] T028 [US2] 更新 validate-basic.sh 中的 delay 策略格式在 `executor/cli/examples/scripts/validate-basic.sh`

**Checkpoint**: 所有配置文件使用新格式，CLI 和 Control Plane 类型同步

---

## Phase 5: User Story 3 - Metrics Correctly Recorded (Priority: P2)

**Goal**: 确保 delay 故障指标正确记录

**Independent Test**: 查询 Envoy stats 端点验证 `wasmcustom_hfi_faults_delays_total` 计数器递增

### Implementation for User Story 3

- [ ] T029 [US3] 验证 `execute_delay` 中 metrics 记录逻辑正确在 `executor/wasm-plugin/src/executor.rs`
- [ ] T030 [US3] 确保 delay 成功时递增 `delays_total` 计数器在 `executor/wasm-plugin/src/lib.rs`
- [ ] T031 [US3] 确保 histogram 记录使用正确的 duration 值在 `executor/wasm-plugin/src/executor.rs`
- [ ] T032 [US3] 手动验证：应用 delay 策略，发送请求，查询 `/stats/prometheus` 确认指标递增

**Checkpoint**: Metrics 正确记录

---

## Phase 6: E2E 验证 & Polish

**Purpose**: 端到端验证，确保所有功能正常工作

- [ ] T033 运行 `cargo test` 确保所有单元测试通过在 `executor/wasm-plugin/`
- [ ] T034 运行 `cargo clippy` 确保无 lint 警告在 `executor/wasm-plugin/`
- [ ] T035 运行 `./validate-basic.sh` E2E 验证在 `executor/cli/examples/scripts/`
- [ ] T036 验证 delay 测试通过：响应时间增加约 500ms (±10%)
- [ ] T037 检查 Envoy 日志无 "dispatch_http_call: BadArgument" 错误
- [ ] T038 更新 quickstart.md 验证步骤在 `specs/010-fix-wasm-delay-bug/quickstart.md`
- [ ] T039 [P] 代码清理：移除未使用的 imports 在 `executor/wasm-plugin/src/config.rs`

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1 (Setup)
    │
    ▼
Phase 2 (Foundational) ← BLOCKS all user stories
    │
    ├──────────────────┬──────────────────┐
    ▼                  ▼                  ▼
Phase 3 (US1)     Phase 4 (US2)     Phase 5 (US3)
    │                  │                  │
    └──────────────────┴──────────────────┘
                       │
                       ▼
              Phase 6 (E2E & Polish)
```

### User Story Dependencies

| Story | 依赖 | 可并行 |
|-------|------|--------|
| US1 (P1) | Phase 2 完成 | 否，先完成此 MVP |
| US2 (P2) | Phase 2 完成 + US1 (plugin.wasm 构建后) | 可与 US3 并行 |
| US3 (P2) | Phase 2 完成 + US1 | 可与 US2 并行 |

### Within Each Phase

- T004-T009 必须顺序执行（数据结构依赖）
- T019-T024 可并行执行（不同文件）
- T025-T027 依赖 T019-T024 完成

---

## Parallel Opportunities

### Phase 2 内部

```bash
# 无并行机会 - 数据结构变更有顺序依赖
```

### Phase 4 内部 (US2)

```bash
# 可并行更新所有 YAML 文件:
T020, T021, T022, T023, T024 可同时执行
```

### User Story 并行

```bash
# US1 完成后，US2 和 US3 可并行:
Developer A: T019-T028 (US2 配置更新)
Developer B: T029-T032 (US3 Metrics 验证)
```

---

## Implementation Strategy

### MVP First (仅 User Story 1)

1. ✅ Complete Phase 1: Setup
2. ✅ Complete Phase 2: Foundational (数据结构变更)
3. ✅ Complete Phase 3: User Story 1 (核心 Bug 修复)
4. **STOP and VALIDATE**: 手动测试 delay 故障注入
5. 如果 MVP 通过，可立即部署修复

### Incremental Delivery

1. Setup + Foundational → 代码编译通过
2. User Story 1 → Delay 故障可用 → **MVP Ready!**
3. User Story 2 → 配置格式统一 → 用户体验改进
4. User Story 3 → Metrics 验证 → 可观测性完整
5. Phase 6 → E2E 验证 → 发布就绪

---

## Notes

- 此功能是 **Bug 修复 + 配置简化**，影响范围明确
- US1 是 MVP，优先完成以恢复核心功能
- US2 和 US3 可并行，优先级相同
- 所有 YAML 文件更新可批量并行完成
- 使用 `validate-basic.sh` 作为最终验收标准
