# Research: CLI Examples Update for Multi-Service Microservice System

**Feature**: 009-cli-examples-update  
**Date**: 2025-12-10  
**Status**: Complete

## Research Summary

本功能不涉及复杂的技术决策，主要是文件重组和脚本编写。以下记录关键决策。

---

## Decision 1: 策略示例目录结构

### Decision
采用三层目录结构：`basic/`、`advanced/`、`scenarios/`

### Rationale
- **渐进式学习**: 用户可以从基础示例开始，逐步深入
- **易于导航**: 清晰的分类减少查找时间
- **行业实践**: 类似 Kubernetes 示例仓库的组织方式

### Alternatives Considered
| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| 扁平结构（现状） | 简单 | 文件多时难以导航 | ❌ 不适合 |
| 按故障类型分类 | 逻辑清晰 | 用户场景跨多个目录 | ❌ 不便 |
| 三层结构 | 兼顾学习路径和组织性 | 需要迁移现有文件 | ✅ 采用 |

---

## Decision 2: 验证脚本实现方式

### Decision
使用 Bash 脚本 + kubectl exec，放置在 `examples/scripts/` 目录

### Rationale
- **零额外依赖**: 只需 kubectl、curl、jq（标准工具）
- **与示例紧密关联**: 放在 examples 目录下便于用户发现
- **复用现有模式**: 参考 `executor/k8s/tests/` 中已有的测试脚本风格

### Alternatives Considered
| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| Go 测试程序 | 类型安全，可复用 CLI | 需要编译，增加复杂性 | ❌ 过度工程化 |
| Python 脚本 | 可读性好，灵活 | 需要 Python 环境 | ❌ 增加依赖 |
| Bash 脚本 | 轻量，无依赖 | 复杂逻辑可读性差 | ✅ 采用 |

---

## Decision 3: 服务选择器默认值策略

### Decision
所有示例默认使用具体的 `selector` 值（如 `service: frontend, namespace: demo`），而非通配符

### Rationale
- **明确意图**: 用户清楚知道策略应用范围
- **安全默认**: 避免意外影响所有服务
- **教育目的**: 引导用户思考目标服务

### Alternatives Considered
| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| 默认通配符 | 向后兼容 | 可能意外影响所有服务 | ❌ 不安全 |
| 省略 selector | 简洁 | 新用户不知道字段存在 | ❌ 不教育 |
| 具体值 + 注释 | 明确，可教育 | 用户需要修改 | ✅ 采用 |

---

## Decision 4: 验证脚本的测试策略

### Decision
创建两个独立脚本：`validate-basic.sh`（基础故障验证）和 `validate-selector.sh`（选择器匹配验证）

### Rationale
- **关注点分离**: 每个脚本专注一个验证目标
- **独立运行**: 用户可以选择性运行
- **CI 友好**: 可以并行或按需执行

### 脚本设计

#### validate-basic.sh
```
功能：验证 abort 和 delay 故障注入基本功能
流程：
1. 前置检查（kubectl, Control Plane, WasmPlugin）
2. 创建 abort 策略 (100%)
3. 发送 10 个请求，验证 100% 返回 503
4. 清理策略
5. 创建 delay 策略 (500ms)
6. 发送请求，验证延迟 >= 400ms
7. 清理策略
8. 输出结果摘要
```

#### validate-selector.sh
```
功能：验证服务选择器精确匹配
前提：需要至少两个服务（如 frontend, productcatalog）
流程：
1. 前置检查
2. 创建针对 frontend 的策略
3. 测试 frontend 请求 → 应该被影响
4. 测试 productcatalog 请求 → 不应该被影响
5. 清理策略
6. 输出结果摘要
```

---

## Decision 5: Online Boutique 场景示例

### Decision
创建 3 个代表性策略示例，覆盖不同服务和故障类型

### Rationale
- **真实场景**: Online Boutique 是 Google 官方微服务示例，广泛使用
- **覆盖关键服务**: frontend（入口）、checkout（关键业务）、payment（外部依赖）
- **展示级联故障**: 通过 payment 服务故障演示上游影响

### 示例设计
| 文件 | 目标服务 | 故障类型 | 场景 |
|------|---------|---------|------|
| `frontend-abort.yaml` | frontend | 503 abort | 入口服务不可用 |
| `checkout-delay.yaml` | checkoutservice | 2s delay | 结账流程变慢 |
| `payment-cascading.yaml` | paymentservice | 503 abort | 支付失败导致级联 |

---

## Technical Notes

### Envoy Identity 元数据提取

Wasm 插件通过以下路径获取服务身份：
```
node.metadata.WORKLOAD_NAME → 服务名（如 "frontend"）
node.metadata.NAMESPACE → 命名空间（如 "demo"）
```

这些值由 Istio 注入到 Envoy 配置中，来源于 Pod 的标签和命名空间。

### 策略传播延迟

- Control Plane 默认 30 秒轮询间隔
- 验证脚本需要等待 ~35 秒让策略传播
- 可通过 `pluginConfig.poll_interval_ms` 调整（WasmPlugin CRD）

### 现有测试脚本参考

`executor/k8s/tests/test-us3-service-targeting.sh` 提供了完整的服务选择器测试实现，可以复用其：
- 前置检查逻辑
- 策略创建 helper 函数
- 结果统计方法

---

## Open Questions (Resolved)

| 问题 | 决策 | 理由 |
|------|------|------|
| 示例应该用 JSON 还是 YAML？ | YAML | 可读性更好，支持注释 |
| 验证脚本放哪个目录？ | `examples/scripts/` | 与示例紧密关联 |
| 是否需要 Docker Compose 验证？ | 不需要 | 专注 K8s/Istio 环境 |
