# Wasm Plugin Metrics 暴露方案

## 问题总结

**Istio Telemetry API 不能实现 Wasm 自定义 Metrics**

原因：
- Telemetry API 只支持 Istio 预定义的标准指标（REQUEST_COUNT, REQUEST_DURATION 等）
- Wasm 插件通过 `proxy_wasm::define_metric()` 定义的自定义指标属于 Envoy 层
- 两者是独立的系统，Telemetry API 无法配置 Wasm metrics

## ✅ 可行方案对比

### 方案 1: EnvoyFilter + Stats Matcher（推荐）

**优点：**
- ✅ 不需要修改 Wasm 代码，只需要 K8s 配置
- ✅ 可以精确控制哪些指标暴露
- ✅ 支持全局或按命名空间配置
- ✅ 配置灵活，可以随时调整

**实施步骤：**

```bash
# 1. 应用 EnvoyFilter
kubectl apply -f executor/k8s/envoyfilter-wasm-stats.yaml

# 2. 重启 Pod 使配置生效（或等待自动重启）
kubectl rollout restart deployment -n demo

# 3. 验证指标
kubectl exec -n demo <pod-name> -c istio-proxy -- \
  curl -s http://localhost:15000/stats | grep "hfi.faults"
  
# 预期输出：
# hfi.faults.aborts_total: 123
# hfi.faults.delays_total: 45
# hfi.faults.delay_duration_milliseconds: P50=100 P95=500 P99=1000
```

**配置文件：** `executor/k8s/envoyfilter-wasm-stats.yaml`

---

### 方案 2: 修改指标命名使用 wasmcustom 前缀

**原理：** Envoy 默认会暴露 `wasmcustom.*` 前缀的指标

**优点：**
- ✅ 不需要额外的 K8s 配置
- ✅ 符合 Envoy 的命名约定
- ✅ 在所有 Envoy 版本中通用

**缺点：**
- ❌ 需要修改 Wasm 代码并重新编译
- ❌ 需要重新构建和部署 Wasm 插件镜像

**代码修改：**

```rust
// 在 executor/wasm-plugin/src/lib.rs 的 define_metrics() 方法中
// 修改指标名称：

// 修改前：
"hfi.faults.aborts_total"
"hfi.faults.delays_total"
"hfi.faults.delay_duration_milliseconds"

// 修改后：
"wasmcustom.hfi_faults_aborts_total"
"wasmcustom.hfi_faults_delays_total"
"wasmcustom.hfi_faults_delay_duration_milliseconds"
```

**实施步骤：**

```bash
# 1. 修改代码中的指标名称
# 2. 重新编译 Wasm 插件
cd executor/wasm-plugin
make build

# 3. 重新构建 Docker 镜像
docker build -t your-registry/hfi-wasm-plugin:v2 .

# 4. 更新 WasmPlugin CRD
kubectl set image wasmplugin/boifi-fault-injection \
  -n demo \
  --patch '{"spec":{"url":"oci://your-registry/hfi-wasm-plugin:v2"}}'

# 5. 验证（注意指标名称变化）
kubectl exec -n demo <pod-name> -c istio-proxy -- \
  curl -s http://localhost:15000/stats | grep "wasmcustom.hfi"
```

---

### 方案 3: 组合方案（最佳实践）

同时使用方案 1 和方案 2，获得最佳兼容性：

1. **代码层面**：使用 `wasmcustom.` 前缀（符合 Envoy 约定）
2. **配置层面**：添加 EnvoyFilter 确保指标被暴露（防御性配置）

**好处：**
- 在任何 Envoy/Istio 版本都能工作
- 即使 EnvoyFilter 未部署也能看到指标
- 配置明确，易于维护

---

## 推荐方案

**当前阶段：方案 1（EnvoyFilter）**

原因：
1. 无需修改代码，快速验证
2. 已实现的代码不需要重新编译
3. 可以在不重新构建镜像的情况下启用指标

**长期规划：方案 3（组合）**

在下一次 Wasm 插件更新时：
1. 重命名指标使用 `wasmcustom.` 前缀
2. 保留 EnvoyFilter 配置作为明确声明
3. 更新文档说明指标命名约定

---

## 验证步骤

### 1. 应用 EnvoyFilter

```bash
kubectl apply -f executor/k8s/envoyfilter-wasm-stats.yaml
```

### 2. 重启目标 Pod

```bash
# 方式 1: 滚动重启
kubectl rollout restart deployment -n demo

# 方式 2: 删除单个 Pod 让它重建
kubectl delete pod -n demo <pod-name>
```

### 3. 等待 Pod 就绪

```bash
kubectl wait --for=condition=ready pod -l app=frontend -n demo --timeout=60s
```

### 4. 触发一些故障注入

```bash
# 确保有活跃的策略
./cli/hfi-cli policy list

# 发送测试请求
for i in {1..20}; do
  kubectl run curl-test-$i -n demo --image=curlimages/curl --rm -i --restart=Never -- \
    curl -s http://frontend/
done
```

### 5. 检查 Envoy Stats

```bash
# 方式 1: 直接查看 stats
kubectl exec -n demo <frontend-pod> -c istio-proxy -- \
  curl -s http://localhost:15000/stats | grep "hfi.faults"

# 方式 2: 查看 Prometheus 格式
kubectl exec -n demo <frontend-pod> -c istio-proxy -- \
  curl -s http://localhost:15090/stats/prometheus | grep "hfi_faults"
```

### 6. 预期输出

```prometheus
# 成功时应该看到：
hfi_faults_aborts_total{} 15
hfi_faults_delays_total{} 5
hfi_faults_delay_duration_milliseconds_bucket{le="100"} 2
hfi_faults_delay_duration_milliseconds_bucket{le="500"} 5
hfi_faults_delay_duration_milliseconds_sum 2500
hfi_faults_delay_duration_milliseconds_count 5
```

---

## 故障排查

### 问题 1: 应用 EnvoyFilter 后仍然看不到指标

**原因：** Pod 没有重启，旧的 Envoy 配置仍在使用

**解决：**
```bash
kubectl rollout restart deployment/<deployment-name> -n demo
```

### 问题 2: 指标显示为 0

**原因：** 没有实际触发故障注入

**解决：**
1. 检查策略是否存在：`./cli/hfi-cli policy list`
2. 检查策略是否匹配请求
3. 发送测试流量触发故障注入

### 问题 3: EnvoyFilter 应用失败

**原因：** YAML 格式错误或 Istio 版本不兼容

**解决：**
```bash
# 检查 EnvoyFilter 状态
kubectl describe envoyfilter -n istio-system wasm-stats-inclusion

# 查看 istiod 日志
kubectl logs -n istio-system deployment/istiod | grep -i "envoyfilter\|error"
```

---

## 总结

| 特性 | Telemetry API | EnvoyFilter | 修改代码命名 |
|------|--------------|-------------|-------------|
| **支持 Wasm 自定义指标** | ❌ 不支持 | ✅ 支持 | ✅ 支持 |
| **需要修改代码** | N/A | ❌ 不需要 | ✅ 需要 |
| **需要重新部署** | N/A | ✅ 需要（重启Pod） | ✅ 需要（重建镜像） |
| **配置复杂度** | N/A | 中等 | 低 |
| **推荐使用** | ❌ 不适用 | ✅ 推荐 | ⚠️ 长期方案 |

**最终答案：Istio Telemetry API 不能实现，请使用 EnvoyFilter 方案。**
