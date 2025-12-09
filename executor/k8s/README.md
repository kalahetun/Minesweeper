# Kubernetes 部署指南

本目录包含 BOIFI 故障注入系统的 Kubernetes 部署清单。

## 🚀 快速开始

### 使用 Makefile 一键部署（推荐）

```bash
cd executor
make deploy-all
```

自动完成：
- ✅ 编译 Wasm 插件
- ✅ 创建命名空间 (boifi, demo)
- ✅ 部署所有组件（控制平面、Wasm 服务器、WasmPlugin、EnvoyFilter）
- ✅ 验证部署状态

### 常用命令

```bash
make help                    # 查看所有可用命令
make deploy-all          # 完整部署
make status-k8s              # 检查部署状态
make test-k8s                # 运行端到端测试
make undeploy            # 卸载所有组件
make logs-wasm-plugin        # 查看 Wasm 插件日志
make logs-control-plane      # 查看控制平面日志
make update-wasm-plugin      # 更新 Wasm 插件
```

---

## 📦 组件说明

### 核心组件

| 组件 | 文件 | 说明 |
|------|------|------|
| 控制平面 | `control-plane.yaml` | 故障策略管理服务（2 副本）+ etcd 存储 |
| Wasm 服务器 | `wasm-server.yaml` | 通过 HTTP 分发 Wasm 插件（nginx + hostPath） |
| WasmPlugin | `wasmplugin.yaml` | Istio WasmPlugin CRD，自动注入到 Envoy sidecar |
| EnvoyFilter | `envoyfilter-wasm-stats.yaml` | 配置 Envoy 统计匹配器，暴露指标 |
| 命名空间 | `namespace.yaml` | 创建 boifi 和 demo 命名空间 |

### 前置条件

- Kubernetes 1.24+（测试使用 k3s）
- Istio 1.24+
- kubectl 和 istioctl 已配置
- demo 命名空间启用 Istio 注入：`kubectl label namespace demo istio-injection=enabled`

## 🔧 架构说明

```
┌─────────────────┐
│   控制平面       │  管理故障策略（API: 8080）
│   + etcd        │
└────────┬────────┘
         │
    ┌────┴────┐
    │ Wasm    │  通过 HTTP 分发插件
    │ Server  │  (http://wasm-server.boifi.svc/plugin.wasm)
    └────┬────┘
         │
┌────────▼────────┐
│  WasmPlugin CRD │  Istio 自动注入到所有 sidecar
└────────┬────────┘
         │
┌────────▼────────┐
│  应用 Pod       │
│  ├─ app         │  业务容器
│  └─ istio-proxy │  Envoy sidecar（加载 Wasm 插件）
└─────────────────┘
```

## 🧪 测试验证

### 应用故障策略

```bash
# 端口转发到控制平面
kubectl port-forward -n boifi svc/hfi-control-plane 8080:8080 &

# 应用故障策略
cd executor/cli
./hfi-cli policy apply -f examples/abort-policy.yaml

# 查看策略列表
./hfi-cli policy list
```

### 验证指标暴露

```bash
# 获取应用 Pod
POD=$(kubectl get pod -n demo -l app=frontend -o jsonpath='{.items[0].metadata.name}')

# 查询 Wasm 插件指标
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s http://localhost:15020/stats/prometheus | grep wasmcustom_hfi_faults
```

**暴露的指标：**
- `wasmcustom_hfi_faults_aborts_total`: 中止故障计数
- `wasmcustom_hfi_faults_delays_total`: 延迟故障计数
- `wasmcustom_hfi_faults_delay_duration_milliseconds`: 延迟时长分布（Histogram）

### 端到端测试

```bash
# 运行完整测试套件
make test-k8s

# 运行所有测试
make test-k8s-all
```

## 🐛 故障排查

### 常见问题

**1. WasmPlugin 未加载**

```bash
# 检查 wasm-server 是否运行
kubectl get pods -n boifi -l app=wasm-server

# 检查插件是否可访问
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl -I http://wasm-server.boifi.svc.cluster.local/plugin.wasm

# 查看 Wasm 插件日志
kubectl logs -n demo $POD -c istio-proxy | grep -i wasm
```

**2. 指标未显示**

```bash
# 检查 EnvoyFilter 是否部署
kubectl get envoyfilter -n demo hfi-wasm-metrics

# 手动查询指标端点
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s http://localhost:15020/stats/prometheus | grep wasmcustom

# 注意：BOOTSTRAP 类型的 EnvoyFilter 需要重启 Pod 生效
kubectl rollout restart deployment -n demo
```

**3. 控制平面无法连接**

```bash
# 检查控制平面状态
kubectl get pods -n boifi -l app=hfi-control-plane

# 查看日志
kubectl logs -n boifi -l app=hfi-control-plane --tail=50

# 端口转发进行本地测试
make port-forward-control-plane
```

详细故障排查：参见 [METRICS_SOLUTION.md](METRICS_SOLUTION.md)

## 📝 常用命令速查

```bash
# 部署管理
make deploy-all         # 完整部署
make undeploy           # 完全卸载
make redeploy           # 重新部署
make status-k8s             # 检查状态

# 组件管理
make update-wasm-plugin     # 更新插件
make logs-wasm-plugin       # 插件日志
make logs-control-plane     # 控制平面日志

# 端口转发
make port-forward-control-plane  # 转发控制平面（8080）
make port-forward-wasm-server    # 转发 Wasm 服务器（8081）

# 测试
make test-k8s               # 运行端到端测试
make test-k8s-all           # 运行所有测试

# K8s 原生命令
kubectl get wasmplugin -n demo              # 查看 WasmPlugin
kubectl get envoyfilter -n demo             # 查看 EnvoyFilter
kubectl get pods -n boifi                   # 查看 boifi 命名空间 Pod
kubectl logs -n demo $POD -c istio-proxy    # 查看 sidecar 日志
```

## 📚 参考文档

- [Istio WasmPlugin 文档](https://istio.io/latest/docs/reference/config/proxy_extensions/wasm-plugin/)
- [Envoy Wasm 扩展](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/wasm_filter)
- [Feature 008 规范](../../specs/008-wasm-metrics-exposure/spec.md)
- [详细故障排查指南](METRICS_SOLUTION.md)
