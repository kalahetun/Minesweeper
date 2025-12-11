# Tasks: CLI Examples Update for Multi-Service Microservice System

**Input**: Design documents from `/specs/009-cli-examples-update/`  
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, quickstart.md ✅

**Tests**: 本功能的核心交付物就是验证脚本，因此"测试"即实现任务的一部分。

**Organization**: 任务按用户故事分组，支持独立实现和测试。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 任务所属的用户故事（如 US1, US2, US3）
- 描述中包含完整文件路径

## Path Conventions

- **策略示例**: `executor/cli/examples/`
- **验证脚本**: `executor/cli/examples/scripts/`
- **参考测试**: `executor/k8s/tests/`

---

## Phase 1: Setup (目录结构创建)

**Purpose**: 创建新的目录结构，为后续任务做准备

- [x] T001 创建目录结构 `executor/cli/examples/basic/`
- [x] T002 [P] 创建目录结构 `executor/cli/examples/advanced/`
- [x] T003 [P] 创建目录结构 `executor/cli/examples/scenarios/online-boutique/`
- [x] T004 [P] 创建目录结构 `executor/cli/examples/scripts/`

**Checkpoint**: ✅ 目录结构就绪，可以开始迁移和创建文件

---

## Phase 2: Foundational (共享脚本库)

**Purpose**: 创建验证脚本的共享函数库，被所有验证脚本依赖

- [x] T005 创建共享函数库 `executor/cli/examples/scripts/common.sh`，包含：
  - 日志输出函数（log_info, log_error, log_test）
  - 颜色定义
  - 前置检查函数（check_kubectl, check_control_plane, check_wasmplugin）
  - 策略创建/删除 helper 函数
  - 请求发送和结果统计函数
  - 清理函数

**Checkpoint**: ✅ 共享库就绪，验证脚本可以开始实现

---

## Phase 3: User Story 1 - 更新策略示例 (Priority: P1) 🎯 MVP

**Goal**: 为所有现有策略示例添加 `selector` 字段，并迁移到新目录结构

**Independent Test**: 使用 `hfi-cli policy apply` 或 curl 提交每个示例到 Control Plane，验证返回成功

### Implementation for User Story 1

#### Basic 策略（移动 + 更新）

- [x] T006 [P] [US1] 创建 `executor/cli/examples/basic/abort-policy.yaml`，添加 `selector: {service: frontend, namespace: demo}` 和详细注释
- [x] T007 [P] [US1] 创建 `executor/cli/examples/basic/delay-policy.yaml`，添加 `selector` 字段和详细注释
- [x] T008 [P] [US1] 创建 `executor/cli/examples/basic/percentage-policy.yaml`，添加 `selector` 字段和详细注释

#### Advanced 策略（移动 + 更新）

- [x] T009 [P] [US1] 创建 `executor/cli/examples/advanced/header-policy.yaml`，添加 `selector` 字段
- [x] T010 [P] [US1] 创建 `executor/cli/examples/advanced/time-limited-policy.yaml`，添加 `selector` 字段（演示 duration_seconds）
- [x] T011 [P] [US1] 创建 `executor/cli/examples/advanced/late-stage-policy.yaml`，添加 `selector` 字段（演示 start_delay_ms）
- [x] T012 [P] [US1] 移动现有 `executor/cli/examples/service-targeted-policy.yaml` 到 `executor/cli/examples/advanced/service-targeted-policy.yaml`

#### 清理旧文件

- [x] T013 [US1] 删除根目录下的旧策略文件（abort-policy.yaml, delay-policy.yaml 等），保留 README.md

**Checkpoint**: ✅ User Story 1 完成 - 所有策略示例已更新并包含 selector 字段

---

## Phase 4: User Story 2 - 基础验证脚本 (Priority: P1) 🎯 MVP

**Goal**: 创建验证 abort 和 delay 故障注入基本功能的脚本

**Independent Test**: 在 k3s 集群上运行 `./validate-basic.sh`，观察测试通过

### Implementation for User Story 2

- [x] T014 [US2] 创建 `executor/cli/examples/scripts/validate-basic.sh`，实现：
  - 前置检查（调用 common.sh）
  - Abort 策略验证（创建策略 → 等待传播 → 发送请求 → 验证 503 → 清理）
  - Delay 策略验证（创建策略 → 等待传播 → 发送请求 → 验证延迟 → 清理）
  - 结果摘要输出
  - 正确的退出码（0=成功，1=失败，2=前置检查失败）
- [x] T015 [US2] 使脚本可执行 `chmod +x executor/cli/examples/scripts/validate-basic.sh`
- [ ] T016 [US2] 在 k3s 集群上测试脚本，确保端到端流程正常

**Checkpoint**: User Story 2 完成 - 基础验证脚本可用 ✅ (T016 待运行时测试)

---

## Phase 5: User Story 3 - 服务选择器验证脚本 (Priority: P2)

**Goal**: 创建验证服务选择器精确匹配功能的脚本

**Independent Test**: 在有多个服务的 k3s 集群上运行 `./validate-selector.sh`

### Implementation for User Story 3

- [ ] T017 [US3] 创建 `executor/cli/examples/scripts/validate-selector.sh`，实现：
  - 前置检查（确保至少两个服务可用）
  - 精确匹配测试（创建针对 SERVICE_A 的策略 → 验证 SERVICE_A 受影响 → 验证 SERVICE_B 不受影响）
  - 通配符测试（可选）
  - 结果摘要输出
  - 正确的退出码
- [ ] T018 [US3] 使脚本可执行 `chmod +x executor/cli/examples/scripts/validate-selector.sh`
- [ ] T019 [US3] 在 k3s 集群上测试脚本，验证选择器精确匹配

**Checkpoint**: User Story 3 完成 - 服务选择器验证脚本可用

---

## Phase 6: User Story 4 - 更新 README 文档 (Priority: P2)

**Goal**: 更新 README 文档，添加 Service Selector、Validation Scripts、Quick Start 章节

**Independent Test**: 新用户按照 README 完成首次故障注入验证

### Implementation for User Story 4

- [ ] T020 [US4] 更新 `executor/cli/examples/README.md`：
  - 更新目录结构说明（basic/, advanced/, scenarios/, scripts/）
  - 添加 "Service Selector" 章节，解释 selector 字段用法和匹配规则
  - 添加 "Validation Scripts" 章节，列出所有脚本及用途
  - 更新 "Quick Start" 章节，添加使用验证脚本的步骤
  - 更新策略文件引用路径

**Checkpoint**: User Story 4 完成 - README 文档已更新

---

## Phase 7: User Story 5 - 微服务场景示例 (Priority: P3)

**Goal**: 提供 Online Boutique 微服务场景的完整示例

**Independent Test**: 在部署了 Online Boutique 的集群上应用示例并验证

### Implementation for User Story 5

- [ ] T021 [P] [US5] 创建 `executor/cli/examples/scenarios/README.md`，说明场景示例的用途
- [ ] T022 [P] [US5] 创建 `executor/cli/examples/scenarios/online-boutique/frontend-abort.yaml`，模拟前端服务不可用
- [ ] T023 [P] [US5] 创建 `executor/cli/examples/scenarios/online-boutique/checkout-delay.yaml`，模拟结账服务延迟
- [ ] T024 [P] [US5] 创建 `executor/cli/examples/scenarios/online-boutique/payment-cascading.yaml`，模拟支付服务故障导致级联

**Checkpoint**: User Story 5 完成 - 微服务场景示例可用

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: 最终验证和清理

- [ ] T025 [P] 验证所有策略示例可被 Control Plane API 接受（SC-001）
- [ ] T026 [P] 验证验证脚本在 3 分钟内完成（SC-002）
- [ ] T027 运行 `quickstart.md` 中的步骤，确保文档准确
- [ ] T028 提交所有变更并创建 Pull Request

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1 (Setup) ─────────────────────────────────┐
                                                 │
Phase 2 (Foundational) ──────────────────────────┤
                                                 │
    ┌────────────────────────────────────────────┴───────────────────────────────────────────┐
    │                                                                                        │
    ▼                                ▼                               ▼                       ▼
Phase 3 (US1)                   Phase 4 (US2)                   Phase 5 (US3)           Phase 6 (US4)
策略示例更新                      基础验证脚本                     选择器验证脚本          README 更新
    │                                │                               │                       │
    │                                │                               │                       │
    │                                ▼                               │                       │
    │                           Phase 7 (US5)                        │                       │
    │                         微服务场景示例                           │                       │
    │                                │                               │                       │
    └────────────────────────────────┴───────────────────────────────┴───────────────────────┘
                                                 │
                                                 ▼
                                        Phase 8 (Polish)
```

### User Story Dependencies

| User Story | 依赖 | 说明 |
|------------|------|------|
| US1 (策略示例) | Phase 1, 2 | 独立，无其他故事依赖 |
| US2 (基础验证) | Phase 1, 2, common.sh | 依赖共享库 |
| US3 (选择器验证) | Phase 1, 2, common.sh | 依赖共享库 |
| US4 (README) | 建议等 US1, US2 完成 | 需要引用新文件路径 |
| US5 (场景示例) | Phase 1 | 独立，可与其他故事并行 |

### Parallel Opportunities

**Phase 1 内部并行**:
```
T001, T002, T003, T004 可同时执行
```

**Phase 3 (US1) 内部并行**:
```
T006, T007, T008, T009, T010, T011, T012 可同时执行（不同文件）
```

**Phase 7 (US5) 内部并行**:
```
T021, T022, T023, T024 可同时执行（不同文件）
```

**跨用户故事并行**（Foundational 完成后）:
```
US1 (策略示例) ←→ US2 (基础验证) ←→ US3 (选择器验证) ←→ US5 (场景示例)
可由不同开发者并行进行
```

---

## Parallel Example: User Story 1

```bash
# 并行创建所有 basic 策略:
T006: 创建 basic/abort-policy.yaml
T007: 创建 basic/delay-policy.yaml
T008: 创建 basic/percentage-policy.yaml

# 并行创建所有 advanced 策略:
T009: 创建 advanced/header-policy.yaml
T010: 创建 advanced/time-limited-policy.yaml
T011: 创建 advanced/late-stage-policy.yaml
T012: 移动 advanced/service-targeted-policy.yaml
```

---

## Implementation Strategy

### MVP First (User Story 1 + 2)

1. ✅ Complete Phase 1: Setup（目录结构）
2. ✅ Complete Phase 2: Foundational（common.sh）
3. ✅ Complete Phase 3: User Story 1（策略示例更新）
4. ✅ Complete Phase 4: User Story 2（基础验证脚本）
5. **STOP and VALIDATE**: 运行 `validate-basic.sh` 验证端到端流程
6. 可以发布 MVP 版本

### Incremental Delivery

| 阶段 | 交付物 | 价值 |
|------|--------|------|
| MVP | US1 + US2 | 用户可以使用更新的示例并验证基础功能 |
| +US3 | 选择器验证脚本 | 验证多服务场景的精确匹配 |
| +US4 | README 更新 | 完整的用户文档 |
| +US5 | 场景示例 | 真实场景参考 |

### 建议执行顺序

1. **第一批**（并行）: T001-T004（目录结构）
2. **第二批**: T005（common.sh）
3. **第三批**（并行）: T006-T012（所有策略文件）
4. **第四批**: T013（清理旧文件）
5. **第五批**: T014-T016（validate-basic.sh）
6. **第六批**: T017-T019（validate-selector.sh）
7. **第七批**: T020（README 更新）
8. **第八批**（并行）: T021-T024（场景示例）
9. **第九批**: T025-T028（最终验证）

---

## Notes

- 所有脚本必须设置 `set -e` 确保错误时立即退出
- 验证脚本等待策略传播时间约 35 秒（30秒轮询 + 缓冲）
- 参考 `executor/k8s/tests/test-us3-service-targeting.sh` 获取现有模式
- 策略文件使用 YAML 格式，包含详细注释
- 提交时使用 Conventional Commits 格式（如 `feat: update CLI examples with service selector`）
