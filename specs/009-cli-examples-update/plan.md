# Implementation Plan: CLI Examples Update for Multi-Service Microservice System

**Branch**: `009-cli-examples-update` | **Date**: 2025-12-10 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/009-cli-examples-update/spec.md`

## Summary

更新 CLI 策略示例以支持多服务微服务系统中的服务选择器 (`spec.selector`)，并创建 K8s/Istio 环境下的验证脚本，用于端到端测试故障注入效果。主要工作包括：
1. 为所有现有策略示例添加 `selector` 字段
2. 重组示例目录结构（basic/、advanced/、scenarios/）
3. 创建 E2E 验证脚本
4. 更新 README 文档

## Technical Context

**Language/Version**: Bash (验证脚本), YAML (策略文件), Markdown (文档)  
**Primary Dependencies**: kubectl, curl, jq (验证脚本依赖)  
**Storage**: N/A (文件系统，无数据库)  
**Testing**: Bash 脚本 + kubectl exec 进行 E2E 验证  
**Target Platform**: Linux (k3s + Istio 环境)  
**Project Type**: 文档/配置/脚本更新 (无核心代码变更)  
**Performance Goals**: 验证脚本在 3 分钟内完成完整测试流程  
**Constraints**: 脚本必须支持 CI/CD 无人值守运行，正确返回退出码  
**Scale/Scope**: ~10 个策略示例文件，2-3 个验证脚本

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| 原则 | 状态 | 说明 |
|------|------|------|
| I. 关注点分离 | ✅ PASS | 示例文件和验证脚本是独立的，不修改核心代码 |
| II. 声明式配置 | ✅ PASS | 策略文件使用 YAML 声明式格式，包含完整的 schema |
| III. 动态性与实时性 | ✅ PASS | 验证脚本测试策略的实时传播 |
| IV. 测试驱动 | ✅ PASS | 创建验证脚本本身就是测试工作 |
| V. 性能优先 | ✅ PASS | 不涉及热路径代码修改 |
| VI. 容错与可靠性 | ✅ PASS | 验证脚本包含前置检查和清理逻辑 |
| VII. 简洁性 | ✅ PASS | 使用 Bash + kubectl，不引入新依赖 |
| VIII. 时间控制 | ✅ PASS | 示例包含 time_control 字段演示 |

**Gate Result**: ✅ ALL PASS - 无违规，可继续

## Project Structure

### Documentation (this feature)

```text
specs/009-cli-examples-update/
├── plan.md              # 本文件
├── research.md          # Phase 0: 研究输出
├── data-model.md        # Phase 1: 数据模型
├── quickstart.md        # Phase 1: 快速开始指南
├── contracts/           # Phase 1: API 契约 (N/A - 无新 API)
└── tasks.md             # Phase 2: 任务列表
```

### Source Code (变更范围)

```text
executor/cli/examples/
├── README.md                     # 更新：添加 Service Selector 和 Validation Scripts 章节
├── basic/                        # 新建：基础策略示例
│   ├── abort-policy.yaml         # 移动+更新：添加 selector
│   ├── delay-policy.yaml         # 移动+更新：添加 selector
│   └── percentage-policy.yaml    # 移动+更新：添加 selector
├── advanced/                     # 新建：高级策略示例
│   ├── header-policy.yaml        # 移动+更新
│   ├── time-limited-policy.yaml  # 移动+更新
│   ├── late-stage-policy.yaml    # 移动+更新
│   └── service-targeted-policy.yaml  # 保留：已有 selector 示例
├── scenarios/                    # 新建：微服务场景示例
│   ├── online-boutique/          # 示例：Google Online Boutique
│   │   ├── frontend-abort.yaml
│   │   ├── checkout-delay.yaml
│   │   └── payment-cascading.yaml
│   └── README.md                 # 场景说明
└── scripts/                      # 新建：验证脚本
    ├── validate-basic.sh         # 基础验证（abort/delay）
    ├── validate-selector.sh      # 服务选择器验证
    └── common.sh                 # 共享函数库

executor/k8s/tests/               # 现有测试脚本位置（参考）
├── test-us3-service-targeting.sh # 参考：已有的服务选择器测试
└── ...
```

**Structure Decision**: 采用分层目录结构 (basic/advanced/scenarios/)，便于用户按复杂度渐进学习。验证脚本放在 examples/scripts/ 下，与示例紧密关联。

## Complexity Tracking

> 无违规需要说明
