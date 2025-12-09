# BOIFI Service Discovery

Service Discovery 是 BOIFI（面向混沌工程的智能故障注入系统）的服务发现组件，负责从 Kubernetes/Istio 和 Jaeger 收集服务拓扑信息，并发布到 Redis 供其他组件消费。

## 功能特性

- **Kubernetes 服务发现**: 解析 Istio VirtualService 获取服务和 API 端点信息
- **Jaeger 拓扑构建**: 通过 Jaeger Dependencies API 构建服务间调用关系图
- **Redis 发布**: 将 ServiceMap 以 JSON 格式存储到 Redis，并通过 Pub/Sub 通知更新
- **周期执行**: 可配置的定时发现周期
- **优雅降级**: Jaeger 不可用时继续工作，仅输出警告日志
- **OpenAPI 增强**: 可选的 OpenAPI 规范获取，补充 API 详细信息

## 快速开始

### 前置条件

- Go 1.21+
- Kubernetes 集群（带 Istio）
- Jaeger（可选，用于拓扑构建）
- Redis

### 构建

```bash
make build
```

### 配置

复制示例配置文件并修改：

```bash
cp config.example.yaml config.yaml
```

配置项说明：

```yaml
kubernetes:
  kubeconfig: ""  # 空字符串使用 in-cluster 配置

jaeger:
  url: "http://jaeger-query:16686"
  lookback: 1h      # 查询最近 1 小时的依赖数据
  timeout: 30s

redis:
  address: "redis:6379"
  password: ""
  db: 0
  key: "boifi:service-map"
  channel: "boifi:service-map:updates"

discovery:
  interval: 5m  # 发现周期

log:
  level: info   # debug, info, warn, error
  format: json  # json, text
```

### 运行

**周期执行模式（默认）**:

```bash
./bin/service-discovery -c config.yaml
```

**单次执行模式**:

```bash
./bin/service-discovery -c config.yaml --once
```

**使用环境变量**:

```bash
export REDIS_ADDRESS=redis:6379
export JAEGER_URL=http://jaeger:16686
./bin/service-discovery
```

### Docker

```bash
docker build -t boifi/service-discovery .
docker run -v $(pwd)/config.yaml:/app/config.yaml boifi/service-discovery
```

## 数据模型

### ServiceMap

```json
{
  "services": {
    "user-service": {
      "name": "user-service",
      "namespace": "default",
      "apis": [
        {"path": "/users", "method": "GET", "matchType": "exact"},
        {"path": "/users", "method": "POST", "matchType": "exact"}
      ]
    }
  },
  "edges": [
    {"source": "frontend", "target": "user-service", "callCount": 1000}
  ],
  "timestamp": "2024-01-01T00:00:00Z",
  "metadata": {
    "jaegerLookback": "1h",
    "discoveryInterval": "5m",
    "stale": false
  }
}
```

## Redis 使用

### 读取 ServiceMap

```bash
redis-cli GET boifi:service-map
```

### 订阅更新通知

```bash
redis-cli SUBSCRIBE boifi:service-map:updates
```

## 开发

### 运行测试

```bash
make test
```

### 运行 lint

```bash
make lint
```

### 项目结构

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
