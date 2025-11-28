# Quick Start: Service Discovery for BOIFI

**Branch**: `006-service-discovery` | **Date**: 2025-11-28

## 概述

Service Discovery 是一个 Go 微服务，用于自动发现 Kubernetes 集群中的服务和 API 端点，构建服务级拓扑图，并发布到 Redis 供 BOIFI 其他组件使用。

---

## 前置条件

### 必需
- Go 1.21+
- Kubernetes 集群 (已安装 Istio)
- Redis 服务
- Jaeger Query 服务

### 可选
- Docker (用于容器化部署)
- kubectl (用于本地开发测试)

---

## 快速开始

### 1. 克隆并进入项目目录

```bash
cd /home/huiguo/wasm_fault_injection
git checkout 006-service-discovery
```

### 2. 构建

```bash
cd service-discovery
go mod tidy
go build -o bin/service-discovery ./cmd
```

### 3. 配置

创建 `config.yaml`:

```yaml
kubernetes:
  kubeconfig: ~/.kube/config  # 本地开发使用；部署时留空使用 in-cluster

jaeger:
  url: "http://localhost:16686"
  lookback: "1h"

redis:
  address: "localhost:6379"
  key: "boifi:service-map"
  channel: "boifi:service-map:updates"

discovery:
  interval: "5m"

log:
  level: "info"
  format: "json"
```

### 4. 运行

```bash
# 使用配置文件
./bin/service-discovery --config config.yaml

# 或使用环境变量
export BOIFI_JAEGER_URL=http://localhost:16686
export BOIFI_REDIS_ADDR=localhost:6379
./bin/service-discovery

# 单次运行（不周期执行）
./bin/service-discovery --once
```

---

## 验证

### 检查 Redis 中的服务地图

```bash
# 获取服务地图
redis-cli GET boifi:service-map | jq .

# 订阅更新通知
redis-cli SUBSCRIBE boifi:service-map:updates
```

### 预期输出

```json
{
  "timestamp": "2025-11-28T10:30:00Z",
  "services": {
    "productpage": {
      "name": "productpage",
      "namespace": "default",
      "apis": [
        {"method": "GET", "path": "/productpage", "match_type": "exact"}
      ],
      "source": "virtualservice"
    }
  },
  "topology": [
    {"source": "productpage", "target": "reviews", "call_count": 100}
  ],
  "metadata": {
    "discovery_interval": "5m",
    "jaeger_lookback": "1h",
    "stale": false
  }
}
```

---

## Docker 部署

### 构建镜像

```bash
docker build -t boifi/service-discovery:latest -f Dockerfile .
```

### 运行容器

```bash
docker run -d \
  --name service-discovery \
  -e BOIFI_JAEGER_URL=http://jaeger-query:16686 \
  -e BOIFI_REDIS_ADDR=redis:6379 \
  boifi/service-discovery:latest
```

---

## Kubernetes 部署

### 部署清单示例

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: service-discovery
  namespace: boifi
spec:
  replicas: 1
  selector:
    matchLabels:
      app: service-discovery
  template:
    metadata:
      labels:
        app: service-discovery
    spec:
      serviceAccountName: service-discovery
      containers:
        - name: service-discovery
          image: boifi/service-discovery:latest
          env:
            - name: BOIFI_JAEGER_URL
              value: "http://jaeger-query.observability:16686"
            - name: BOIFI_REDIS_ADDR
              value: "redis.boifi:6379"
            - name: BOIFI_DISCOVERY_INTERVAL
              value: "5m"
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: service-discovery
  namespace: boifi
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: service-discovery
rules:
  - apiGroups: ["networking.istio.io"]
    resources: ["virtualservices"]
    verbs: ["list", "get", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: service-discovery
subjects:
  - kind: ServiceAccount
    name: service-discovery
    namespace: boifi
roleRef:
  kind: ClusterRole
  name: service-discovery
  apiGroup: rbac.authorization.k8s.io
```

---

## 开发指南

### 运行测试

```bash
# 单元测试
go test ./internal/... -v

# 集成测试（需要环境）
go test ./tests/integration/... -v

# 覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### 代码结构

```
service-discovery/
├── cmd/main.go              # 入口点
├── internal/
│   ├── config/              # 配置加载
│   ├── discovery/           # 核心发现逻辑
│   │   ├── kubernetes.go    # VirtualService 解析
│   │   ├── jaeger.go        # 拓扑构建
│   │   └── openapi.go       # OpenAPI 增强
│   ├── publisher/           # Redis 发布
│   ├── scheduler/           # 周期调度
│   └── types/               # 数据结构
├── tests/
│   ├── unit/
│   └── integration/
└── Makefile
```

---

## 常见问题

### Q: 服务地图为空？
A: 检查：
1. Kubernetes 集群中是否有 VirtualService 资源
2. Service Discovery 是否有权限访问 `networking.istio.io` API

### Q: 拓扑图为空？
A: 检查：
1. Jaeger 是否可达
2. Jaeger 中是否有近期的 trace 数据
3. 尝试增加 `jaeger.lookback` 配置

### Q: Redis 发布失败？
A: 检查：
1. Redis 地址是否正确
2. 网络连接是否正常
3. 查看日志中的错误信息

---

## 下一步

- 查看 [data-model.md](./data-model.md) 了解数据结构
- 查看 [research.md](./research.md) 了解技术选型
- 查看 [plan.md](./plan.md) 了解实现计划
