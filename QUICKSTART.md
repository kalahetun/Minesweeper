# HFI Quick Start Guide

## 简介

**HFI (HTTP Fault Injection)** 是一个基于 Kubernetes 和 Envoy 的云原生故障注入平台，用于进行混沌工程和弹性测试。它通过 WASM 插件在 Envoy 代理中实现细粒度的 HTTP 故障注入，支持延迟、中断、错误状态码等多种故障类型。

HFI 主要组件：
- **控制平面**：管理故障注入策略和配置
- **WASM 插件**：在 Envoy sidecar 中执行故障注入逻辑  
- **CLI 工具**：命令行界面，用于管理策略和监控

## 先决条件

在开始之前，请确保你的环境满足以下要求：

- **Kubernetes 集群**：
  - `kubectl` 已安装并配置好可访问集群
  - 推荐使用 `kind`、`minikube` 或 `k3s` 进行本地测试
  - Kubernetes 版本 >= 1.20

- **容器运行时**：
  - `docker` 已安装并运行
  - 能够构建和推送镜像到集群可访问的仓库

- **网络访问**：
  - 能够从本地访问 Kubernetes 集群服务（通过 port-forward）

### 验证环境

```bash
# 检查 kubectl 连接
kubectl cluster-info

# 检查节点状态  
kubectl get nodes

# 检查 Docker
docker version
```

## 步骤 1: 部署控制平面

首先部署 HFI 控制平面，包括 etcd 存储和控制平面服务：

```bash
# 应用控制平面配置
kubectl apply -f https://raw.githubusercontent.com/your-org/hfi/main/k8s/control-plane.yaml

# 等待控制平面启动
kubectl wait --for=condition=ready pod -l app=hfi-control-plane --timeout=120s
kubectl wait --for=condition=ready pod -l app=hfi-etcd --timeout=120s
```

**预期输出：**
```
deployment.apps/hfi-control-plane created
service/hfi-control-plane created
deployment.apps/hfi-etcd created
service/hfi-etcd created
pod/hfi-control-plane-xxx-xxx condition met
pod/hfi-etcd-xxx-xxx condition met
```

**验证部署：**
```bash
# 检查控制平面状态
kubectl get pods -l app=hfi-control-plane
kubectl get pods -l app=hfi-etcd

# 检查服务
kubectl get svc -l component=control-plane
```

## 步骤 2: 部署示例应用与 Envoy Sidecar

部署一个示例应用，配置了 Envoy sidecar 和 HFI WASM 插件：

```bash
# 部署示例应用
kubectl apply -f https://raw.githubusercontent.com/your-org/hfi/main/k8s/sample-app-with-proxy.yaml

# 等待应用启动
kubectl wait --for=condition=ready pod -l app=sample-app --timeout=120s
```

**预期输出：**
```
deployment.apps/sample-app-with-proxy created
service/sample-app created
pod/sample-app-with-proxy-xxx-xxx condition met
```

**验证部署：**
```bash
# 检查应用状态
kubectl get pods -l app=sample-app

# 检查服务
kubectl get svc sample-app

# 查看容器状态（应有 httpbin 和 envoy-proxy 两个容器）
kubectl describe pod -l app=sample-app
```

## 步骤 3: 安装 hfi-cli

下载并安装 HFI 命令行工具：

### Linux/macOS

```bash
# 下载最新版本（替换为实际版本号）
curl -LO https://github.com/your-org/hfi/releases/latest/download/hfi-cli-linux-amd64

# 添加执行权限
chmod +x hfi-cli-linux-amd64

# 移动到 PATH 目录
sudo mv hfi-cli-linux-amd64 /usr/local/bin/hfi-cli

# 验证安装
hfi-cli version
```

### 从源码构建

```bash
# 克隆仓库
git clone https://github.com/your-org/hfi.git
cd hfi/cli

# 构建 CLI
go build -o hfi-cli main.go

# 移动到 PATH（可选）
sudo mv hfi-cli /usr/local/bin/
```

**验证安装：**
```bash
hfi-cli --help
```

## 步骤 4: 注入你的第一个故障

### 4.1 创建故障注入策略

创建一个简单的延迟故障策略：

```yaml
# 保存为 policy.yaml
apiVersion: v1
kind: FaultInjectionPolicy
metadata:
  name: delay-demo
  version: "1.0.0"
spec:
  rules:
    - match:
        path:
          prefix: "/delay"
        method:
          exact: "GET"
      fault:
        delay:
          fixedDelay: "2s"
        percentage: 100
```

### 4.2 应用策略

```bash
# 配置 CLI 连接到控制平面
kubectl port-forward svc/hfi-control-plane 8080:8080 &

# 等待端口转发建立
sleep 3

# 应用故障注入策略
hfi-cli policy apply -f policy.yaml

# 验证策略已创建
hfi-cli policy list
```

**预期输出：**
```
Policy applied successfully: delay-demo
ID: delay-demo
Name: delay-demo
Version: 1.0.0
Rules: 1
```

## 步骤 5: 验证结果

### 5.1 设置访问

```bash
# 转发示例应用端口
kubectl port-forward svc/sample-app 8000:80 &

# 等待端口转发建立
sleep 3
```

### 5.2 测试正常请求

```bash
# 测试正常请求（应该很快返回）
time curl -s http://localhost:8000/get
```

**预期输出：**
```json
{
  "args": {},
  "headers": {
    "Host": "localhost:8000",
    "User-Agent": "curl/7.81.0"
  },
  "origin": "127.0.0.1",
  "url": "http://localhost:8000/get"
}

real    0m0.156s
user    0m0.004s
sys     0m0.008s
```

### 5.3 测试故障注入

```bash
# 测试延迟故障（应该延迟 2 秒）
time curl -s http://localhost:8000/delay/test
```

**预期输出：**
```json
{
  "args": {},
  "headers": {
    "Host": "localhost:8000",
    "User-Agent": "curl/7.81.0"
  },
  "origin": "127.0.0.1",
  "url": "http://localhost:8000/delay/test"
}

real    0m2.234s  ← 注意这里有 2 秒延迟
user    0m0.003s
sys     0m0.009s
```

### 5.4 查看指标

```bash
# 访问 Envoy admin 接口查看故障注入指标
kubectl port-forward svc/sample-app 19000:19000 &
sleep 3

# 查看故障注入指标
curl -s http://localhost:19000/stats | grep hfi.faults
```

**预期输出：**
```
wasmcustom.hfi.faults.delays_total: 1
wasmcustom.hfi.faults.delay_duration_milliseconds: P50(nan,2000) P95(nan,2000) P99(nan,2000)
```

## 步骤 6: 探索更多故障类型

### 6.1 中断故障

```yaml
# 保存为 abort-policy.yaml
apiVersion: v1
kind: FaultInjectionPolicy
metadata:
  name: abort-demo
  version: "1.0.0"
spec:
  rules:
    - match:
        path:
          prefix: "/abort"
        method:
          exact: "GET"
      fault:
        abort:
          httpStatus: 503
        percentage: 100
```

```bash
# 应用中断策略
hfi-cli policy apply -f abort-policy.yaml

# 测试中断故障
curl -w "HTTP %{http_code}\n" http://localhost:8000/abort/test
```

**预期输出：**
```
Fault injection: Service unavailable
HTTP 503
```

### 6.2 基于头部的故障注入

```yaml
# 保存为 header-policy.yaml
apiVersion: v1
kind: FaultInjectionPolicy
metadata:
  name: header-demo
  version: "1.0.0"
spec:
  rules:
    - match:
        path:
          prefix: "/api"
        headers:
          - name: "x-user-type"
            exact: "test"
      fault:
        delay:
          fixedDelay: "1s"
        percentage: 50
```

```bash
# 应用头部策略
hfi-cli policy apply -f header-policy.yaml

# 测试带头部的请求
time curl -H "x-user-type: test" http://localhost:8000/api/users
```

## 清理

完成测试后，清理所有创建的资源：

```bash
# 停止端口转发
pkill -f "kubectl port-forward"

# 删除故障注入策略
hfi-cli policy delete delay-demo
hfi-cli policy delete abort-demo
hfi-cli policy delete header-demo

# 删除 Kubernetes 资源
kubectl delete -f https://raw.githubusercontent.com/your-org/hfi/main/k8s/sample-app-with-proxy.yaml
kubectl delete -f https://raw.githubusercontent.com/your-org/hfi/main/k8s/control-plane.yaml

# 验证清理
kubectl get pods
```

**预期输出：**
```
No resources found in default namespace.
```

## 下一步

恭喜！你已经成功运行了第一个故障注入实验。接下来你可以：

1. **深入了解策略语法**：查看 [策略参考文档](docs/policy-reference.md)
2. **集成监控**：设置 Prometheus 和 Grafana 来可视化故障注入指标
3. **自动化测试**：将故障注入集成到 CI/CD 流水线中
4. **生产环境**：在生产环境中进行受控的混沌工程实验

## 故障排除

### 常见问题

**问题 1：控制平面无法启动**
```bash
# 检查日志
kubectl logs deployment/hfi-control-plane
kubectl logs deployment/hfi-etcd

# 检查资源
kubectl describe pod -l app=hfi-control-plane
```

**问题 2：故障注入不生效**
```bash
# 检查 WASM 插件日志
kubectl logs deployment/sample-app-with-proxy -c envoy-proxy

# 检查策略是否正确应用
hfi-cli policy get delay-demo
```

**问题 3：网络连接问题**
```bash
# 检查服务状态
kubectl get svc
kubectl get endpoints

# 测试服务连通性
kubectl run debug --image=curlimages/curl:latest --rm -it -- sh
```

## 联系我们

- **GitHub Issues**: [https://github.com/your-org/hfi/issues](https://github.com/your-org/hfi/issues)
- **文档**: [https://hfi.example.com/docs](https://hfi.example.com/docs)
- **社区**: [https://discord.gg/hfi](https://discord.gg/hfi)
