# Quickstart: Fix WASM Plugin Delay Fault Bug

**Feature**: 010-fix-wasm-delay-bug  
**Date**: 2025-12-11

## Overview

此功能修复了 WASM 插件 delay 故障注入失败的 Bug，并简化了配置格式。

## 配置格式变更

### 旧格式 (不再支持)

```yaml
fault:
  delay:
    fixed_delay: "500ms"  # ❌ 已弃用
```

### 新格式

```yaml
fault:
  delay:
    fixed_delay_ms: 500   # ✅ 直接使用毫秒整数
```

## 快速验证

### 1. 应用 Delay 策略

```bash
# 创建 delay 策略
cat <<EOF | hfi-cli policy apply -f -
metadata:
  name: delay-test
spec:
  selector:
    service: frontend
    namespace: demo
  rules:
    - match:
        path:
          prefix: /
      fault:
        percentage: 100
        delay:
          fixed_delay_ms: 500
EOF
```

### 2. 测试延迟效果

```bash
# 测量响应时间
time curl -s http://localhost:8081/ > /dev/null

# 期望输出: real 0m0.5xx s (约 500ms)
```

### 3. 验证指标

```bash
# 获取 frontend pod
POD=$(kubectl get pod -n demo -l app=frontend -o jsonpath='{.items[0].metadata.name}')

# 查询 delay 指标
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s http://localhost:15090/stats/prometheus | grep wasmcustom_hfi_faults_delays
```

### 4. 清理

```bash
hfi-cli policy delete delay-test
```

## 运行验证脚本

```bash
cd executor/cli/examples/scripts

# 基础验证 (包含 delay 测试)
./validate-basic.sh

# 期望输出:
# ✅ Test 1 PASSED: Abort fault working (Error rate: 100%)
# ✅ Test 2 PASSED: Delay fault working (Avg delay: 512ms)
```

## 迁移指南

如果您有使用旧格式的策略文件，请按以下方式更新：

| 旧值 | 新值 |
|------|------|
| `fixed_delay: "100ms"` | `fixed_delay_ms: 100` |
| `fixed_delay: "500ms"` | `fixed_delay_ms: 500` |
| `fixed_delay: "1s"` | `fixed_delay_ms: 1000` |
| `fixed_delay: "2s"` | `fixed_delay_ms: 2000` |

## 限制

- 最大延迟: 30,000ms (30 秒)
- 超过最大值的配置将自动 clamp 到 30,000ms

## 故障排除

### Delay 不生效

1. 检查策略已应用: `hfi-cli policy list`
2. 检查 selector 匹配: 确保 service/namespace 正确
3. 查看 Envoy 日志: `kubectl logs -n demo $POD -c istio-proxy | grep -i delay`

### 仍然看到 BadArgument 错误

确保使用的是修复后的 plugin.wasm 版本。重新构建并部署：

```bash
cd executor/wasm-plugin
make build
# 更新 WasmPlugin 资源指向新的 wasm 文件
```
