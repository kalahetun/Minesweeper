# 部署指南 (DEPLOYMENT.md)

**版本**: v1.0  
**最后更新**: 2024-11-13  
**适用**: HFI v1.0+

---

## 目录

1. [前置条件](#前置条件)
2. [部署步骤](#部署步骤)
3. [配置参数](#配置参数)
4. [验证部署](#验证部署)
5. [故障排查](#故障排查)

---

## 前置条件

### 系统要求

```
✅ Kubernetes 1.19+ (任何发行版: EKS, GKE, AKS 等)
✅ Envoy 1.20+ (作为 sidecar 或网关)
✅ etcd 3.5+ (可选，默认使用内存存储)
✅ Docker 20.10+ (用于构建镜像)
✅ 4GB+ RAM (Control Plane 最小配置)
✅ 2+ CPU cores (生产环境建议)
```

### 开发者要求 (如需从源代码构建)

```
✅ Go 1.18+ (Control Plane)
✅ Rust 1.70+ (WASM Plugin)
✅ Cargo (Rust 包管理器)
```

### 网络要求

```
✅ Control Plane 可访问性: 需要对 WASM 插件暴露端口 8080
✅ WASM 插件连接: 需要能连接回 Control Plane SSE 端点
✅ etcd 连接: 若使用 etcd 存储，需要网络可达
```

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
