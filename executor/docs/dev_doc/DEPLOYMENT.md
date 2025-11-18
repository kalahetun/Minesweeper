# BOIFI Cloud-Native 部署指南

**版本**: v2.0 (Phase 7)  
**最后更新**: 2024-11-16  
**适用**: BOIFI v1.0+

---

## 目录

1. [Docker Compose 部署](#docker-compose-部署)
2. [k3s Kubernetes 部署](#k3s-kubernetes-部署)
3. [多实例部署](#多实例部署)
4. [故障转移和恢复](#故障转移和恢复)
5. [监控和故障排除](#监控和故障排除)
6. [性能指标](#性能指标)

---

## Docker Compose 部署

### 概述

Docker Compose 用于本地开发和测试。使用官方的 `docker-compose.yaml` 快速启动完整的 BOIFI 系统。

**组件**:
- Control Plane: REST API 服务器 (端口 8080)
- Backend: 故障注入执行引擎 (端口 5000)
- etcd: 策略持久化存储 (端口 2379)

### 前置要求

```bash
# 检查 Docker 版本
docker --version  # 需要 20.10 或更高

# 检查 Docker Compose 版本
docker-compose --version  # 需要 1.29 或更高
```

### 快速启动

```bash
# 进入 Docker 目录
cd /path/to/wasm_fault_injection/executor/docker

# 启动所有服务
docker-compose up -d

# 检查服务状态
docker-compose ps

# 查看日志
docker-compose logs -f control-plane
docker-compose logs -f backend
```

### 服务验证

#### 1. Control Plane 健康检查

```bash
# 检查服务是否就绪
curl -s http://localhost:8080/healthz
# 预期响应: 200 OK

# 列出所有策略
curl -s http://localhost:8080/v1/policies | jq '.'

# 创建测试策略
curl -X POST http://localhost:8080/v1/policies \
  -H 'Content-Type: application/json' \
  -d '{
    "metadata": {"name": "test-policy", "version": "1.0"},
    "spec": {
      "rules": [{
        "match": {"path": {"exact": "/api/test"}},
        "fault": {"percentage": 50, "abort": {"httpStatus": 500}}
      }]
    }
  }'

# 检索策略
curl -s http://localhost:8080/v1/policies/test-policy | jq '.'
```

#### 2. 后端服务验证

```bash
# 检查后端服务
curl -s http://localhost:5000/health
# 预期响应: 200 OK 或 {"status": "healthy"}
```

### 运行集成测试

项目提供自动化测试脚本用于验证 Docker Compose 部署：

```bash
# 运行 Docker Compose 集成测试
cd /path/to/wasm_fault_injection/executor/docker
bash compose-test.sh

# 脚本检查项：
# ✓ Docker 和 docker-compose 版本验证
# ✓ 服务启动和镜像拉取
# ✓ 健康检查 (/healthz 端点)
# ✓ 策略创建和检索 API 验证
# ✓ 容器日志聚合
```

### 停止和清理

```bash
# 停止所有服务（保持数据）
docker-compose down

# 停止并删除所有卷（完全清理）
docker-compose down -v

# 查看已停止的容器
docker ps -a

# 删除特定容器
docker rm <container_id>
```

---

## k3s Kubernetes 部署

### 概述

k3s 是一个轻量级 Kubernetes 发行版，适合生产部署和测试。与标准 Kubernetes 相比，开销更小（<5MB）。

**架构**:
- Control Plane Deployment: 1 副本（可扩展）
- Plugin Deployment: 多个副本（默认 3 个）
- etcd: 内部 K3s etcd 或外部 etcd
- Service 和 Ingress: 流量路由

### 前置要求

#### 1. 安装 k3s

```bash
# 在 Linux 上安装 k3s（单节点群集）
curl -sfL https://get.k3s.io | sh -

# 验证安装
sudo k3s --version

# 设置 kubeconfig
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
sudo chmod 644 /etc/rancher/k3s/k3s.yaml

# 或复制到用户主目录
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/k3s.yaml
sudo chown $USER:$USER ~/.kube/k3s.yaml
chmod 600 ~/.kube/k3s.yaml
```

#### 2. 验证 k3s 群集

```bash
# 检查群集信息
kubectl cluster-info

# 查看节点
kubectl get nodes

# 查看系统 Pods
kubectl get pods -n kube-system
```

#### 3. 构建容器镜像

```bash
# 构建 Control Plane 镜像（如果使用本地构建）
cd /path/to/wasm_fault_injection/executor/control-plane
docker build -f ../docker/Dockerfile.controlplane -t boifi/control-plane:latest .

# 构建 Wasm Plugin 镜像
cd /path/to/wasm_fault_injection/executor/wasm-plugin
docker build -f ../docker/Dockerfile.backend -t boifi/wasm-plugin:latest .

# 或使用预构建镜像
docker pull envoyproxy/envoy:v1.27-latest
```

### 部署步骤

#### 1. 创建命名空间

```bash
# 创建 boifi 命名空间
kubectl create namespace boifi

# 验证
kubectl get namespace boifi
```

#### 2. 部署 Control Plane

```bash
# 应用 Control Plane 配置
kubectl apply -f /path/to/wasm_fault_injection/executor/k8s/control-plane.yaml

# 验证部署
kubectl get deployment -n boifi
kubectl get pods -n boifi

# 等待 Pod 就绪（最多 300 秒）
kubectl wait --for=condition=ready pod \
  -l app=control-plane -n boifi --timeout=300s
```

#### 3. 检查 Control Plane 日志

```bash
# 实时查看日志
kubectl logs -n boifi -l app=control-plane -f

# 查看最后 100 行日志
kubectl logs -n boifi -l app=control-plane --tail=100

# 查看特定 Pod 日志
kubectl logs -n boifi <pod-name>
```

#### 4. 访问 Control Plane API

```bash
# 获取 Control Plane Service IP
kubectl get svc -n boifi control-plane

# 使用 LoadBalancer（如果可用）
CONTROL_PLANE_IP=$(kubectl get svc -n boifi control-plane \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# 或使用 ClusterIP（需要 port-forward）
kubectl port-forward -n boifi svc/control-plane 8080:8080 &

# 测试 API
curl -s http://localhost:8080/healthz
curl -s http://localhost:8080/v1/policies | jq '.'
```

### 运行 k3s 部署测试

项目提供自动化测试脚本用于验证 k3s 部署：

```bash
# 运行 k3s 部署测试
bash /path/to/wasm_fault_injection/executor/k8s/tests/deploy_test.sh

# 脚本检查项：
# ✓ kubectl 和 k3s 可用性验证
# ✓ kubeconfig 路径配置
# ✓ 命名空间创建
# ✓ 清单部署 (control-plane.yaml, envoy-config.yaml)
# ✓ Pod 就绪等待 (300s 超时)
# ✓ LoadBalancer/ClusterIP IP 发现
# ✓ SSE 端点连接性验证
# ✓ 策略创建和分发验证
# ✓ 集群诊断和日志聚合
```

---

## 多实例部署

### 概述

在生产环境中，通常需要多个 Plugin 副本以实现高可用性和负载均衡。此部分演示如何部署 3 个副本并验证策略分发。

**目标**:
- 部署 3 个 Plugin 实例
- 验证策略在 **< 1 秒** 内分发到所有实例
- 测试负载均衡和故障转移

### 多实例架构

```
Control Plane (1 副本)
    |
    | (etcd 存储)
    |
Service: control-plane
    |
    +-- Deployment: plugin-multi-instance (3 副本)
            |
            +-- Pod 1 (Plugin 实例 1)
            +-- Pod 2 (Plugin 实例 2)
            +-- Pod 3 (Plugin 实例 3)
```

### 部署步骤

#### 1. 验证 Control Plane 就绪

```bash
# 确认 Control Plane 正在运行
kubectl get pods -n boifi -l app=control-plane

# Pod 应显示 Running 和 Ready 状态
```

#### 2. 运行多实例测试脚本

```bash
# 运行多实例分发测试
bash /path/to/wasm_fault_injection/executor/k8s/tests/multi_instance_test.sh

# 脚本检查项：
# ✓ kubectl 和 k3s 可用性验证
# ✓ Plugin 清单生成 (3 副本)
# ✓ 多实例 Deployment 部署
# ✓ 所有副本就绪等待
# ✓ 策略创建时间戳记录
# ✓ 策略分发时间测量（纳秒精度）
# ✓ 所有实例连接性验证
# ✓ 诊断和日志聚合
```

#### 3. 手动验证多实例

```bash
# 查看 Plugin 副本
kubectl get pods -n boifi -l app=plugin

# 检查副本数量（应为 3）
kubectl get deployment -n boifi plugin-multi-instance -o jsonpath='{.spec.replicas}'

# 获取就绪副本数
kubectl get deployment -n boifi plugin-multi-instance -o jsonpath='{.status.readyReplicas}'
```

### 性能验证

```bash
# 创建策略并测量分发时间
START_TIME=$(date +%s%N)

# 创建策略
curl -X POST http://localhost:8080/v1/policies \
  -H 'Content-Type: application/json' \
  -d '{
    "metadata": {"name": "perf-test-policy", "version": "1.0"},
    "spec": {"rules": []}
  }'

# 验证所有实例接收到策略
sleep 0.5

END_TIME=$(date +%s%N)
DURATION_MS=$(( (END_TIME - START_TIME) / 1000000 ))

echo "策略分发耗时: ${DURATION_MS}ms"
# 应该 < 1000ms
```

---

## 故障转移和恢复

### 概述

故障转移测试验证 Control Plane Pod 重启后的恢复能力。包括数据恢复和新连接建立。

### 测试场景

1. **预故障**: 创建测试数据（3 个策略）
2. **模拟故障**: 删除 Control Plane Pod
3. **恢复**: Kubernetes 自动重启 Pod
4. **验证**: 确认数据恢复和连接正常

### 运行故障转移测试

```bash
# 执行完整的故障转移测试
bash /path/to/wasm_fault_injection/executor/k8s/tests/failover_test.sh

# 脚本检查项：
# ✓ Control Plane 初始运行状态验证
# ✓ 测试数据创建（3 个策略）
# ✓ 测试数据存在性验证
# ✓ Control Plane Pod 删除（模拟故障）
# ✓ Pod 恢复等待（最多 300 秒）
# ✓ 数据恢复验证
# ✓ 新连接测试（健康检查、策略创建）
# ✓ 故障转移日志和事件收集
```

### 手动故障转移测试

```bash
# 1. 创建测试数据
curl -X POST http://localhost:8080/v1/policies \
  -H 'Content-Type: application/json' \
  -d '{
    "metadata": {"name": "failover-test", "version": "1.0"},
    "spec": {"rules": []}
  }'

# 2. 验证数据存在
curl -s http://localhost:8080/v1/policies/failover-test | jq '.'

# 3. 获取 Pod 名称并删除
POD_NAME=$(kubectl get pods -n boifi -l app=control-plane -o jsonpath='{.items[0].metadata.name}')
kubectl delete pod -n boifi $POD_NAME

# 4. 监控恢复过程
kubectl get pods -n boifi -l app=control-plane -w

# 5. 验证数据恢复（等待 5-10 秒）
sleep 10
curl -s http://localhost:8080/v1/policies/failover-test | jq '.'

# 6. 测试新连接
curl -s http://localhost:8080/v1/policies | jq '.'
```

---

## 监控和故障排除

### 日志收集

#### Control Plane 日志

```bash
# 实时日志
kubectl logs -n boifi -l app=control-plane -f

# 带时间戳的日志
kubectl logs -n boifi -l app=control-plane --timestamps=true

# 前 50 行和后 50 行
kubectl logs -n boifi -l app=control-plane --tail=50

# 特定时间范围的日志（如果支持）
kubectl logs -n boifi -l app=control-plane --since=10m
```

#### Plugin 日志

```bash
# 所有 Plugin 副本日志
kubectl logs -n boifi -l app=plugin --all-containers=true -f

# 特定 Pod 日志
kubectl logs -n boifi <plugin-pod-name> -f
```

### 常见问题诊断

#### 1. Pod 不就绪

```bash
# 检查 Pod 状态详情
kubectl describe pod -n boifi <pod-name>

# 查看启动日志
kubectl logs -n boifi <pod-name> --previous

# 检查资源可用性
kubectl top nodes
kubectl top pods -n boifi
```

#### 2. API 端点无响应

```bash
# 检查 Service 和 Endpoints
kubectl get svc -n boifi
kubectl get endpoints -n boifi

# 在 Pod 内测试连接
kubectl exec -it -n boifi <pod-name> -- \
  curl http://localhost:8080/healthz

# 检查 NetworkPolicy（如果存在）
kubectl get networkpolicy -n boifi
```

#### 3. 策略分发失败

```bash
# 验证 Control Plane 日志中的分发事件
kubectl logs -n boifi -l app=control-plane | grep -i "distribute\|publish"

# 检查 etcd（如果使用外部 etcd）
kubectl exec -n boifi <etcd-pod> -- \
  etcdctl get --prefix /policies

# 验证 Plugin 连接
kubectl logs -n boifi -l app=plugin | grep -i "connect\|subscribe"
```

---

## 性能指标

### 基准测试

在单个 k3s 节点上的性能指标（3 个 Plugin 副本）：

| 指标 | 值 | 单位 |
|------|-----|------|
| 策略分发延迟 | < 1000 | ms |
| API 响应时间 (GET /v1/policies) | < 50 | ms |
| API 响应时间 (POST /v1/policies) | < 100 | ms |
| Pod 启动时间 | 15-30 | s |
| 数据恢复时间 | < 60 | s |
| 内存占用 (Control Plane) | 256-512 | Mi |
| 内存占用 (Plugin) | 128-256 | Mi |

### 扩展性建议

| 场景 | 配置 | 注意事项 |
|------|------|---------|
| 开发/测试 | 1 Control Plane + 1 Plugin | Docker Compose 足够 |
| 小型生产 | 1 Control Plane + 3 Plugins | k3s 单节点 |
| 中型生产 | 3 Control Plane + 5+ Plugins | k3s 多节点或 K8s |
| 大型生产 | HA Control Plane + 10+ Plugins | 完整 K8s 集群 |

---

## 部署步骤

### 步骤 1: 构建和推送镜像

#### 1.1 构建 Control Plane 镜像

```bash
cd /home/huiguo/wasm_fault_injection

# 构建 Docker 镜像
docker build -f executor/docker/Dockerfile.controlplane \
  -t <your-registry>/hfi-control-plane:v1.0 .

# 推送到镜像仓库
docker push <your-registry>/hfi-control-plane:v1.0
```

#### 1.2 构建 WASM 插件镜像

```bash
cd /home/huiguo/wasm_fault_injection

# 构建 WASM 插件
cargo build --manifest-path executor/wasm-plugin/Cargo.toml \
  --target wasm32-unknown-unknown --release

# 验证构建结果
ls -lh executor/wasm-plugin/target/wasm32-unknown-unknown/release/*.wasm

# 构建 Docker 镜像 (包含 WASM 文件)
docker build -f executor/docker/Dockerfile.backend \
  -t <your-registry>/hfi-wasm-plugin:v1.0 .

# 推送到镜像仓库
docker push <your-registry>/hfi-wasm-plugin:v1.0
```

### 步骤 2: 部署 Control Plane

#### 2.1 创建 Kubernetes 命名空间

```bash
kubectl create namespace hfi-system
```

#### 2.2 部署 Control Plane

```bash
# 修改镜像地址（替换为你的仓库）
kubectl set image deployment/hfi-control-plane \
  hfi-control-plane=<your-registry>/hfi-control-plane:v1.0 \
  -n hfi-system

# 或直接应用 YAML
kubectl apply -f executor/k8s/control-plane.yaml

# 验证部署
kubectl get pods -n hfi-system -l app=hfi-control-plane
kubectl logs -n hfi-system -l app=hfi-control-plane
```

#### 2.3 暴露 Control Plane 服务

```bash
# 查看当前服务
kubectl get svc -n hfi-system

# 创建端口转发 (开发环境)
kubectl port-forward -n hfi-system svc/hfi-control-plane 8080:8080

# 或创建 Ingress (生产环境)
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hfi-control-plane-ingress
  namespace: hfi-system
spec:
  rules:
  - host: hfi-api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: hfi-control-plane
            port:
              number: 8080
EOF
```

### 步骤 3: 配置 Envoy 加载 WASM 插件

#### 3.1 创建 Envoy 配置

编辑 `/tmp/envoy.yaml`:

```yaml
static_resources:
  listeners:
  - name: listener_0
    address:
      socket_address:
        address: 0.0.0.0
        port_value: 10000
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: ingress_http
          codec_type: AUTO
          route_config:
            name: local_route
            virtual_hosts:
            - name: backend
              domains: ["*"]
              routes:
              - match:
                  prefix: "/"
                route:
                  cluster: backend_cluster
          http_filters:
          - name: envoy.filters.http.wasm
            typed_config:
              "@type": type.googleapis.com/udpa.type.v1.TypedStruct
              type_url: type.googleapis.com/envoy.extensions.filters.http.wasm.v3.Wasm
              value:
                config:
                  name: fault_injector
                  root_id: fault_injector
                  vm_config:
                    runtime: envoy.wasm.runtime.v8
                    code:
                      local:
                        filename: /etc/envoy/plugins/fault-injector.wasm
                  configuration:
                    "@type": type.googleapis.com/google.protobuf.StringValue
                    value: |
                      {
                        "control_plane_addr": "http://hfi-control-plane:8080",
                        "retry_interval_sec": 5
                      }
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
  clusters:
  - name: backend_cluster
    connect_timeout: 0.25s
    type: STRICT_DNS
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: backend_cluster
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 127.0.0.1
                port_value: 8888
```

#### 3.2 启动 Envoy

```bash
# 使用 Docker 启动 Envoy
docker run -d --name envoy \
  -v /tmp/envoy.yaml:/etc/envoy/envoy.yaml \
  -v /path/to/wasm-plugin.wasm:/etc/envoy/plugins/fault-injector.wasm \
  -p 10000:10000 \
  envoyproxy/envoy:v1.24-latest

# 或在 Kubernetes 中作为 sidecar
# (参考 executor/k8s/sample-app-with-proxy.yaml)
```

### 步骤 4: 验证集群连接

```bash
# 测试 Control Plane API
curl http://localhost:8080/v1/health

# 预期响应
# {"status": "ok", "version": "1.0"}
```

---

## 配置参数

### Control Plane 配置

#### 环境变量

```bash
# 基础配置
export CONTROL_PLANE_PORT=8080
export CONTROL_PLANE_ADDR=0.0.0.0:8080

# 存储配置
export STORAGE_BACKEND=memory    # 或 etcd
export ETCD_ENDPOINTS=http://localhost:2379

# 日志配置
export LOG_LEVEL=info            # debug, info, warn, error
export LOG_FORMAT=json           # json 或 text

# 性能配置
export REQUEST_TIMEOUT=30s
export WATCH_BUFFER_SIZE=1000
export MAX_POLICIES=10000
```

#### Kubernetes ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: hfi-control-plane-config
  namespace: hfi-system
data:
  config.yaml: |
    server:
      port: 8080
      timeout: 30s
    
    storage:
      backend: memory  # 或 etcd
      etcd:
        endpoints:
          - http://etcd:2379
    
    logging:
      level: info
      format: json
    
    policies:
      max_count: 10000
      auto_expire_check_interval: 60s
```

### WASM 插件配置

#### 环境变量

```bash
export CONTROL_PLANE_ADDR=http://control-plane:8080
export RETRY_INTERVAL_SEC=5
export CACHE_SIZE=1000
export DEBUG=false
```

#### Envoy 配置片段

```yaml
configuration:
  "@type": type.googleapis.com/google.protobuf.StringValue
  value: |
    {
      "control_plane_addr": "http://control-plane:8080",
      "retry_interval_sec": 5,
      "cache_size": 1000,
      "debug": false
    }
```

---

## 验证部署

### 验证步骤

#### 1. 检查 Control Plane

```bash
# 检查 Pod 状态
kubectl get pods -n hfi-system -l app=hfi-control-plane

# 查看日志
kubectl logs -n hfi-system -l app=hfi-control-plane

# 健康检查
curl http://localhost:8080/v1/health
# 预期: {"status": "ok"}
```

#### 2. 检查 WASM 插件连接

```bash
# 查看 Envoy 日志
docker logs envoy | grep "fault-injector"

# 或 Kubernetes 环境
kubectl logs <envoy-pod> -c envoy | grep "fault-injector"

# 查看 Control Plane 日志中的连接信息
kubectl logs -n hfi-system -l app=hfi-control-plane | grep "connected"
```

#### 3. 创建测试策略

```bash
# 创建策略文件
cat > test-policy.yaml << 'EOF'
metadata:
  name: test-policy
spec:
  rules:
    - match:
        path:
          prefix: /api/test
      fault:
        percentage: 50
        delay:
          fixedDelay: "100ms"
EOF

# 应用策略
hfi-cli policy apply -f test-policy.yaml

# 验证策略
hfi-cli policy get test-policy
```

#### 4. 测试故障注入

```bash
# 启动测试客户端
for i in {1..10}; do
  curl http://localhost:10000/api/test
done

# 观察延迟和响应
# - 约 50% 的请求应该有额外的 100ms 延迟
```

---

## 故障排查

### 常见问题

#### 问题 1: Control Plane 无法启动

**症状**:
```
Failed to start: bind: address already in use
```

**原因**: 端口 8080 已被占用

**解决方案**:
```bash
# 方案 1: 杀死占用端口的进程
lsof -i :8080
kill -9 <PID>

# 方案 2: 更改端口
export CONTROL_PLANE_PORT=9090

# 方案 3: Kubernetes 中更改 Service 端口
kubectl edit svc hfi-control-plane -n hfi-system
# 修改 port: 8080 -> 9090
```

#### 问题 2: WASM 插件无法加载

**症状**:
```
Error loading WASM module: file not found
```

**原因**: WASM 文件路径不正确

**解决方案**:
```bash
# 检查文件是否存在
ls -l /etc/envoy/plugins/fault-injector.wasm

# 检查 Envoy 配置中的路径
grep "filename:" /tmp/envoy.yaml

# 重新构建并复制文件
cargo build --manifest-path executor/wasm-plugin/Cargo.toml \
  --target wasm32-unknown-unknown --release

cp executor/wasm-plugin/target/wasm32-unknown-unknown/release/*.wasm \
   /etc/envoy/plugins/fault-injector.wasm
```

#### 问题 3: 策略应用成功但不生效

**症状**:
```
hfi-cli policy apply -f policy.yaml  # 成功
# 但故障注入不生效
```

**原因**: 
1. WASM 插件未连接到 Control Plane
2. Match 条件不匹配
3. 时间窗口已过期

**解决方案**:
```bash
# 1. 检查连接状态
kubectl logs -n hfi-system -l app=hfi-control-plane | grep "connected"

# 2. 检查策略的 match 条件
hfi-cli policy describe test-policy

# 3. 检查时间字段
hfi-cli policy get test-policy -o json | grep -E "start_delay|duration"

# 4. 增加日志级别
export LOG_LEVEL=debug
kubectl rollout restart deployment/hfi-control-plane -n hfi-system
```

#### 问题 4: 性能下降明显

**症状**:
```
请求延迟从 1ms 增加到 100ms+
```

**原因**: 
1. 规则数过多
2. 正则表达式匹配复杂
3. Control Plane 资源不足

**解决方案**:
```bash
# 1. 检查规则数量
hfi-cli policy list | wc -l

# 2. 优化规则使用精确匹配
# 修改 policy.yaml，用精确匹配替代正则表达式

# 3. 按优先级排序规则
hfi-cli policy get -o json | jq '.[] | .spec.rules | length'

# 4. 扩容 Control Plane
kubectl scale deployment hfi-control-plane --replicas=3 -n hfi-system

# 5. 查看 CPU/内存使用
kubectl top pod -n hfi-system
```

### 日志分析

#### 查看相关日志

```bash
# Control Plane 日志
kubectl logs -n hfi-system -l app=hfi-control-plane -f

# WASM 插件日志
kubectl logs <envoy-pod> -c envoy -f | grep -i fault

# 系统事件
kubectl describe pod <pod-name> -n hfi-system

# etcd 日志 (如使用 etcd 存储)
kubectl logs -n kube-system -l component=etcd -f
```

#### 日志级别说明

```
debug  - 最详细，包含所有操作步骤
info   - 标准日志，记录重要事件
warn   - 警告，可能的问题
error  - 错误，需要注意
```

### 性能监控

#### 关键指标

```bash
# Control Plane 性能
kubectl top pod -n hfi-system -l app=hfi-control-plane

# WASM 插件延迟 (从 Envoy 指标)
curl localhost:9000/stats | grep wasm

# 查询 Control Plane 指标
curl http://localhost:8080/v1/metrics
```

#### Prometheus 集成

```yaml
apiVersion: v1
kind: Service
metadata:
  name: hfi-metrics
  namespace: hfi-system
  labels:
    app: hfi-control-plane
spec:
  ports:
  - name: metrics
    port: 9090
    targetPort: 9090
  selector:
    app: hfi-control-plane

---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: hfi-control-plane
  namespace: hfi-system
spec:
  selector:
    matchLabels:
      app: hfi-control-plane
  endpoints:
  - port: metrics
    interval: 30s
```

---

## 最佳实践

### 1. 存储选择

```
开发环境:
├─ 使用内存存储
├─ 快速启动和测试
└─ 数据丢失无关紧要

生产环境:
├─ 使用 etcd 存储
├─ 数据持久化
├─ 支持多副本
└─ 强一致性
```

### 2. 高可用配置

```bash
# 3 副本 Control Plane
kubectl scale deployment hfi-control-plane --replicas=3

# etcd 集群 (3 或 5 节点)
# 参考: https://etcd.io/docs/v3.5/op-guide/clustering/

# 负载均衡
# 使用 Service 或 Ingress 进行负载均衡
```

### 3. 安全最佳实践

```yaml
# RBAC
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: hfi-control-plane
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]

---
# NetworkPolicy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: hfi-network-policy
spec:
  podSelector:
    matchLabels:
      app: hfi-control-plane
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: envoy
```

---

## 卸载

```bash
# 删除所有资源
kubectl delete namespace hfi-system

# 或选择性删除
kubectl delete -f executor/k8s/control-plane.yaml
kubectl delete configmap hfi-control-plane-config -n hfi-system
```

---

**文档版本**: v1.0  
**最后更新**: 2024-11-13  
**维护者**: HFI 团队
