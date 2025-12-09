# BOIFI Service Discovery

Service Discovery 是 BOIFI（面向混沌工程的智能故障注入系统）的服务发现组件，负责从 Kubernetes/Istio 和 Jaeger 收集服务拓扑信息，并发布到 Redis 供其他组件消费。

## 功能特性

- **自动服务发现**: 自动发现所有 Kubernetes Services 和 Istio VirtualServices
- **API 调用详情**: 从 Jaeger traces 提取具体 API 路径、方法和调用次数
- **服务拓扑构建**: 通过 Jaeger Dependencies API 构建服务间调用关系图
- **Redis 发布**: 将 ServiceMap 以 JSON 格式存储到 Redis，并通过 Pub/Sub 通知更新
- **周期执行**: 可配置的定时发现周期（默认 5 分钟）
- **优雅降级**: Jaeger 不可用时继续工作，仅输出警告日志

## 快速开始

### 前置条件

- Kubernetes 集群（k3s/k8s）
- Istio 服务网格
- Jaeger（用于拓扑构建和 API 提取）
- Redis（在 `boifi` 命名空间）
- Docker
- Make

### 部署流程

**1. 配置服务地址**

编辑 `k8s/deployment.yaml` 中的 ConfigMap，确认 Jaeger 和 Redis 地址：

```yaml
data:
  config.yaml: |
    jaeger:
      url: "http://tracing.istio-system.svc.cluster.local:80/jaeger"
      lookback: 1h
      timeout: 30s
    
    redis:
      address: "redis.boifi.svc.cluster.local:6379"
      password: ""
      db: 0
```

**2. 构建并部署**

```bash
# 一键部署（构建镜像 + 导入 k3s + 部署到 Kubernetes）
make deploy
```

这将执行以下步骤：
1. 构建 Docker 镜像
2. 导入镜像到 k3s
3. 应用 Kubernetes 清单
4. 等待 Deployment 就绪

**3. 验证部署**

```bash
# 查看部署状态
make status

# 查看日志
make logs

# 检查 Redis 中的服务地图数据
make check-redis
```

## 数据模型

### ServiceMap

ServiceMap 包含服务列表、API 端点和服务间调用拓扑：

```json
{
  "timestamp": "2025-12-09T06:17:19Z",
  "services": {
    "frontend": {
      "name": "frontend",
      "namespace": "demo",
      "apis": [
        {"method": "*", "path": "/", "match_type": "prefix"}
      ],
      "source": "merged"
    },
    "productcatalogservice": {
      "name": "productcatalogservice",
      "namespace": "demo",
      "apis": [
        {"method": "*", "path": "/hipstershop.ProductCatalogService", "match_type": "prefix"}
      ],
      "source": "merged"
    }
  },
  "topology": [
    {
      "source": "frontend.demo",
      "target": "productcatalogservice.demo",
      "call_count": 402,
      "apis": [
        {
          "path": "/hipstershop.ProductCatalogService/GetProduct",
          "method": "POST",
          "call_count": 2762
        },
        {
          "path": "/hipstershop.ProductCatalogService/ListProducts",
          "method": "POST",
          "call_count": 19
        }
      ]
    }
  ],
  "metadata": {
    "discovery_interval": "5m0s",
    "jaeger_lookback": "1h",
    "stale": false
  }
}
```

**服务来源标记**：
- `virtualservice`: 从 Istio VirtualService 发现
- `kubeservice`: 从 Kubernetes Service 发现
- `merged`: 同时存在 VirtualService 和 K8s Service

## Makefile 命令

### 部署相关

```bash
make deploy          # 构建镜像、导入 k3s、部署到 Kubernetes
make undeploy        # 删除 Kubernetes 部署
make restart         # 重启 Deployment
```

### 构建相关

```bash
make build           # 构建本地二进制文件（用于开发测试）
make docker-build    # 构建 Docker 镜像
make k3s-import      # 导入镜像到 k3s
```

### 监控相关

```bash
make status          # 查看部署状态和 Pod 信息
make logs            # 实时查看日志
make check-redis     # 检查 Redis 中的服务地图数据
```

### 开发相关

```bash
make test            # 运行所有测试
make test-unit       # 运行单元测试
make test-cover      # 运行测试并生成覆盖率报告
make fmt             # 格式化代码
make lint            # 运行代码检查
make clean           # 清理构建产物
```

## Redis 使用

### 读取 ServiceMap

```bash
# 在集群内
kubectl exec -n boifi deploy/redis -- redis-cli GET boifi:service-map | jq '.'

# 查看统计信息
make check-redis
```

### 订阅更新通知

```bash
kubectl exec -n boifi deploy/redis -- redis-cli SUBSCRIBE boifi:service-map:updates
```

## 配置说明

### Deployment 配置 (`k8s/deployment.yaml`)

主要配置项：

```yaml
# 服务配置
jaeger:
  url: "http://tracing.istio-system.svc.cluster.local:80/jaeger"
  lookback: 1h          # 查询最近多长时间的 trace 数据
  timeout: 30s

redis:
  address: "redis.boifi.svc.cluster.local:6379"
  key: "boifi:service-map"
  channel: "boifi:service-map:updates"

discovery:
  interval: 5m          # 发现周期
```

### 命令行选项

Service Discovery 容器支持以下命令行选项（在 Deployment args 中配置）：

```yaml
args:
  - "--config"
  - "/etc/service-discovery/config.yaml"
  - "--full-discovery=true"    # 自动发现所有 K8s Services（默认 true）
  - "--extract-apis=true"      # 从 Jaeger 提取 API 调用信息（默认 true）
  - "--log-level=info"         # 日志级别：debug, info, warn, error
  - "--log-format=json"        # 日志格式：json, text
```

**选项说明**：
- `--full-discovery`: 启用后会发现所有 K8s Services，而不仅是 VirtualServices
- `--extract-apis`: 启用后会从 Jaeger traces 中提取具体的 API 路径和调用统计
- `--once`: 单次执行后退出（用于测试）

## 开发调试

### 本地构建测试

```bash
# 构建二进制
make build

# 本地运行（需要配置 kubeconfig 和端口转发）
./bin/service-discovery --config config-test.yaml --once
```

### 修改后重新部署

```bash
# 重新构建并部署
make deploy

# 或分步执行
make docker-build
make k3s-import
make restart
```

```
service-discovery/
├── cmd/main.go                 # 程序入口
├── internal/
│   ├── config/                 # 配置管理
│   ├── discovery/              # 服务发现
│   │   ├── kubernetes.go       # K8s/Istio 发现
│   │   ├── jaeger.go           # Jaeger 拓扑
│   │   └── openapi.go          # OpenAPI 增强
│   ├── publisher/              # 发布器
│   │   └── redis.go            # Redis 发布
│   ├── scheduler/              # 调度器
│   │   └── ticker.go           # 周期调度
│   └── types/                  # 数据类型
│       └── servicemap.go       # ServiceMap 定义
├── pkg/logger/                 # 日志工具
├── tests/
│   ├── unit/                   # 单元测试
│   └── integration/            # 集成测试
├── k8s/                        # Kubernetes 部署清单
├── Makefile
├── Dockerfile
└── config.example.yaml
```

## 许可证

MIT License
