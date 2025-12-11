# Data Model: CLI Examples Update

**Feature**: 009-cli-examples-update  
**Date**: 2025-12-10  
**Status**: Complete

## Overview

本功能主要涉及配置文件格式，不引入新的数据实体。以下记录现有数据模型的使用方式。

---

## Entity: FaultInjectionPolicy

策略文件的完整结构。

### Schema (YAML)

```yaml
# FaultInjectionPolicy - 故障注入策略
metadata:
  name: string                    # Required: 策略唯一标识符

spec:
  selector:                       # Optional: 服务选择器
    service: string               # 目标服务名，"*" 匹配所有
    namespace: string             # 目标命名空间，"*" 匹配所有
  
  rules:                          # Required: 规则列表
    - match:                      # Required: 匹配条件
        method:                   # Optional: HTTP 方法匹配
          exact: string           # 精确匹配
          prefix: string          # 前缀匹配
          regex: string           # 正则匹配
        path:                     # Optional: 路径匹配
          exact: string
          prefix: string
          regex: string
        headers:                  # Optional: 头部匹配列表
          - name: string          # Required: 头部名称
            exact: string         # 匹配方式（三选一）
            prefix: string
            regex: string
      
      fault:                      # Required: 故障配置
        percentage: integer       # Required: 0-100，触发概率
        start_delay_ms: integer   # Optional: 请求到达后延迟执行（毫秒）
        duration_seconds: integer # Optional: 策略有效期（秒），0=永久
        
        # 故障类型（至少选一）
        abort:                    # 立即返回错误
          httpStatus: integer     # HTTP 状态码 (400-599)
        
        delay:                    # 添加延迟
          fixed_delay: string     # 延迟时长，如 "500ms", "2s"
```

### Validation Rules

| 字段 | 规则 |
|------|------|
| `metadata.name` | 必填，非空字符串，符合 DNS 标签规范 |
| `spec.rules` | 必填，至少包含一个规则 |
| `fault.percentage` | 必填，范围 0-100 |
| `fault.abort.httpStatus` | 范围 400-599 |
| `fault.delay.fixed_delay` | 格式：数字+单位 (ms/s/m) |
| `selector.service` | 可选，默认 "*" |
| `selector.namespace` | 可选，默认 "*" |

---

## Entity: ServiceSelector

服务选择器，用于指定策略应用范围。

### Fields

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `service` | string | `"*"` | Kubernetes workload 名称 |
| `namespace` | string | `"*"` | Kubernetes namespace |

### Matching Logic

```
if policy.selector is null:
    return true  # 匹配所有（向后兼容）

if selector.service == "*" or selector.service == "":
    service_matches = true
else:
    service_matches = (envoy.workload_name == selector.service)

if selector.namespace == "*" or selector.namespace == "":
    namespace_matches = true
else:
    namespace_matches = (envoy.namespace == selector.namespace)

return service_matches AND namespace_matches
```

### Examples

| Selector | 匹配范围 |
|----------|---------|
| `null` (省略) | 所有服务 |
| `{service: "*", namespace: "*"}` | 所有服务 |
| `{service: "frontend", namespace: "*"}` | 所有命名空间的 frontend |
| `{service: "*", namespace: "demo"}` | demo 命名空间的所有服务 |
| `{service: "frontend", namespace: "demo"}` | demo 命名空间的 frontend |

---

## File Structure Model

示例目录的组织结构。

```
examples/
├── README.md                     # 主文档
│
├── basic/                        # 基础示例
│   ├── abort-policy.yaml         # 503 错误注入
│   ├── delay-policy.yaml         # 延迟注入
│   └── percentage-policy.yaml    # 概率触发
│
├── advanced/                     # 高级示例
│   ├── header-policy.yaml        # 头部匹配
│   ├── time-limited-policy.yaml  # 自动过期
│   ├── late-stage-policy.yaml    # 延迟执行
│   └── service-targeted-policy.yaml  # 服务选择器
│
├── scenarios/                    # 场景示例
│   ├── README.md                 # 场景说明
│   └── online-boutique/          # Google Online Boutique
│       ├── frontend-abort.yaml
│       ├── checkout-delay.yaml
│       └── payment-cascading.yaml
│
└── scripts/                      # 验证脚本
    ├── common.sh                 # 共享函数
    ├── validate-basic.sh         # 基础验证
    └── validate-selector.sh      # 选择器验证
```

---

## Script Interface Model

验证脚本的输入输出接口。

### validate-basic.sh

**输入**:
```bash
# 环境变量（可选）
NAMESPACE="demo"           # 目标命名空间
CONTROL_PLANE_NS="boifi"   # Control Plane 命名空间
TARGET_SERVICE="frontend"  # 测试目标服务
```

**输出**:
```
[INFO] Running pre-flight checks...
[INFO] ✅ kubectl available
[INFO] ✅ Control Plane running
[INFO] ✅ WasmPlugin deployed

[TEST] Abort Fault Injection
[INFO] Creating policy: test-abort-policy
[INFO] Waiting for propagation (35s)...
[INFO] Sending 10 requests...
[INFO] ✅ PASS: 10/10 requests returned 503

[TEST] Delay Fault Injection
[INFO] Creating policy: test-delay-policy
[INFO] Waiting for propagation (35s)...
[INFO] Sending 5 requests...
[INFO] ✅ PASS: Average latency 512ms (expected >= 450ms)

[INFO] Cleaning up policies...

========================================
TEST SUMMARY
========================================
Total:  2
Passed: 2
Failed: 0
========================================
```

**退出码**:
- `0`: 所有测试通过
- `1`: 至少一个测试失败
- `2`: 前置检查失败

### validate-selector.sh

**输入**:
```bash
# 环境变量（可选）
NAMESPACE="demo"
CONTROL_PLANE_NS="boifi"
SERVICE_A="frontend"       # 目标服务（应被影响）
SERVICE_B="productcatalog" # 非目标服务（不应被影响）
```

**输出**:
```
[INFO] Running pre-flight checks...
[INFO] ✅ Service frontend found
[INFO] ✅ Service productcatalog found

[TEST] Service Selector - Targeted Policy
[INFO] Creating policy targeting frontend only...
[INFO] Waiting for propagation...

[TEST] Frontend requests (should be affected)
[INFO] ✅ PASS: 10/10 requests to frontend returned 503

[TEST] ProductCatalog requests (should NOT be affected)
[INFO] ✅ PASS: 10/10 requests to productcatalog returned 200

[INFO] Cleaning up...

========================================
SELECTOR TEST SUMMARY
========================================
Total:  2
Passed: 2
Failed: 0
========================================
```

---

## State Transitions

策略的生命周期状态。

```
                    ┌──────────────────┐
                    │                  │
    create_policy   │    ACTIVE        │
    ───────────────►│                  │
                    └────────┬─────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │                   │                   │
         ▼                   ▼                   ▼
    delete_policy      duration_seconds     update_policy
         │              expires              │
         │                   │               │
         ▼                   ▼               ▼
    ┌──────────┐      ┌──────────┐    ┌──────────────┐
    │ DELETED  │      │ EXPIRED  │    │ ACTIVE       │
    │ (removed)│      │ (auto-   │    │ (new version)│
    └──────────┘      │  deleted)│    └──────────────┘
                      └──────────┘
```

---

## Compatibility Notes

### 向后兼容

- 省略 `selector` 字段等同于 `{service: "*", namespace: "*"}`
- 现有不带 `selector` 的策略仍然有效
- CLI 和 Control Plane API 保持不变

### 版本信息

| 组件 | 最低版本 | 说明 |
|------|---------|------|
| Control Plane | v0.1.0 | 已支持 selector 字段 |
| Wasm Plugin | v0.1.0 | 已支持 EnvoyIdentity |
| hfi-cli | v0.1.0 | 无需更新 |
| Istio | 1.20+ | WasmPlugin CRD 支持 |
