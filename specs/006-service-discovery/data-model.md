# Data Model: Service Discovery for BOIFI

**Branch**: `006-service-discovery` | **Date**: 2025-11-28

## 概述

本文档定义 Service Discovery 的核心数据模型，包括输入数据（VirtualService、Jaeger Trace）和输出数据（ServiceMap）。

---

## 1. 输出数据模型 - ServiceMap

### ServiceMap (服务地图)

服务地图是 Service Discovery 的核心输出，包含服务列表、API 端点和服务级拓扑。

```go
// ServiceMap 服务地图 - 顶层结构
type ServiceMap struct {
    // 发现时间戳 (RFC3339 格式)
    Timestamp time.Time `json:"timestamp"`
    
    // 服务列表及其 API 端点
    // Key: 服务名 (如 "productpage", "reviews")
    Services map[string]ServiceInfo `json:"services"`
    
    // 服务级调用拓扑 (有向边列表)
    Topology []ServiceEdge `json:"topology"`
    
    // 元数据
    Metadata MapMetadata `json:"metadata"`
}

// ServiceInfo 单个服务的信息
type ServiceInfo struct {
    // 服务名称
    Name string `json:"name"`
    
    // 服务所在命名空间
    Namespace string `json:"namespace"`
    
    // API 端点列表 (从 VirtualService 提取)
    APIs []APIEndpoint `json:"apis"`
    
    // 数据来源
    Source string `json:"source"` // "virtualservice", "openapi", "merged"
}

// APIEndpoint 单个 API 端点
type APIEndpoint struct {
    // HTTP 方法 (GET, POST, PUT, DELETE, PATCH, *)
    Method string `json:"method"`
    
    // 路径模式
    Path string `json:"path"`
    
    // 匹配类型 (exact, prefix, regex)
    MatchType string `json:"match_type"`
}

// ServiceEdge 服务间调用关系
type ServiceEdge struct {
    // 调用方服务名
    Source string `json:"source"`
    
    // 被调用方服务名
    Target string `json:"target"`
    
    // 统计周期内的调用次数
    CallCount int `json:"call_count"`
}

// MapMetadata 服务地图元数据
type MapMetadata struct {
    // 发现周期配置
    DiscoveryInterval string `json:"discovery_interval"`
    
    // Jaeger 查询时间范围
    JaegerLookback string `json:"jaeger_lookback"`
    
    // 数据是否过期（来自缓存）
    Stale bool `json:"stale"`
    
    // 发现过程中的错误（如有）
    Errors []string `json:"errors,omitempty"`
}
```

### JSON 示例

```json
{
  "timestamp": "2025-11-28T10:30:00Z",
  "services": {
    "productpage": {
      "name": "productpage",
      "namespace": "default",
      "apis": [
        {"method": "GET", "path": "/productpage", "match_type": "exact"},
        {"method": "GET", "path": "/api/v1/products", "match_type": "prefix"}
      ],
      "source": "virtualservice"
    },
    "reviews": {
      "name": "reviews",
      "namespace": "default",
      "apis": [
        {"method": "*", "path": "/reviews", "match_type": "prefix"}
      ],
      "source": "virtualservice"
    },
    "ratings": {
      "name": "ratings",
      "namespace": "default",
      "apis": [
        {"method": "GET", "path": "/ratings/.*", "match_type": "regex"}
      ],
      "source": "virtualservice"
    }
  },
  "topology": [
    {"source": "istio-ingressgateway", "target": "productpage", "call_count": 1000},
    {"source": "productpage", "target": "reviews", "call_count": 980},
    {"source": "productpage", "target": "details", "call_count": 990},
    {"source": "reviews", "target": "ratings", "call_count": 950}
  ],
  "metadata": {
    "discovery_interval": "5m",
    "jaeger_lookback": "1h",
    "stale": false
  }
}
```

---

## 2. 输入数据模型 - Istio VirtualService

### VirtualService 结构 (简化)

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: bookinfo
  namespace: default
spec:
  hosts:
    - "*"
  gateways:
    - bookinfo-gateway
  http:
    - match:
        - uri:
            exact: /productpage
        - uri:
            prefix: /static
        - uri:
            regex: /api/v1/.*
      route:
        - destination:
            host: productpage
            port:
              number: 9080
```

### 解析规则

| VirtualService 字段 | 提取目标 | 处理逻辑 |
|---------------------|----------|----------|
| `spec.http[].match[].uri.exact` | `APIEndpoint.Path` | 直接使用 |
| `spec.http[].match[].uri.prefix` | `APIEndpoint.Path` | 直接使用，标记 `match_type=prefix` |
| `spec.http[].match[].uri.regex` | `APIEndpoint.Path` | 直接使用，标记 `match_type=regex` |
| `spec.http[].match[].method` | `APIEndpoint.Method` | 如缺失，默认为 `*` |
| `spec.http[].route[].destination.host` | `ServiceInfo.Name` | 提取服务名 |
| `metadata.namespace` | `ServiceInfo.Namespace` | 提取命名空间 |

---

## 3. 输入数据模型 - Jaeger Dependencies

### Jaeger Dependencies API 响应

```json
[
  {
    "parent": "productpage",
    "child": "reviews",
    "callCount": 980
  },
  {
    "parent": "reviews",
    "child": "ratings",
    "callCount": 950
  }
]
```

### 映射规则

| Jaeger 字段 | ServiceEdge 字段 |
|-------------|------------------|
| `parent` | `Source` |
| `child` | `Target` |
| `callCount` | `CallCount` |

---

## 4. 配置数据模型

```go
// Config 服务配置
type Config struct {
    Kubernetes KubernetesConfig `yaml:"kubernetes"`
    Jaeger     JaegerConfig     `yaml:"jaeger"`
    Redis      RedisConfig      `yaml:"redis"`
    Discovery  DiscoveryConfig  `yaml:"discovery"`
    OpenAPI    OpenAPIConfig    `yaml:"openapi"`
    Log        LogConfig        `yaml:"log"`
}

type KubernetesConfig struct {
    Kubeconfig string `yaml:"kubeconfig"` // 空字符串表示使用 in-cluster 配置
}

type JaegerConfig struct {
    URL      string        `yaml:"url"`      // http://jaeger-query:16686
    Lookback time.Duration `yaml:"lookback"` // 1h
    Timeout  time.Duration `yaml:"timeout"`  // 30s
}

type RedisConfig struct {
    Address  string `yaml:"address"`  // redis:6379
    Password string `yaml:"password"` // 可选
    DB       int    `yaml:"db"`       // 0
    Key      string `yaml:"key"`      // boifi:service-map
    Channel  string `yaml:"channel"`  // boifi:service-map:updates
}

type DiscoveryConfig struct {
    Interval time.Duration `yaml:"interval"` // 5m
}

type OpenAPIConfig struct {
    Enabled bool     `yaml:"enabled"` // false
    Paths   []string `yaml:"paths"`   // ["/swagger.json", "/v3/api-docs"]
    Timeout time.Duration `yaml:"timeout"` // 5s
}

type LogConfig struct {
    Level  string `yaml:"level"`  // info
    Format string `yaml:"format"` // json
}
```

### 配置文件示例 (config.yaml)

```yaml
kubernetes:
  kubeconfig: ""  # 空表示 in-cluster

jaeger:
  url: "http://jaeger-query:16686"
  lookback: "1h"
  timeout: "30s"

redis:
  address: "redis:6379"
  password: ""
  db: 0
  key: "boifi:service-map"
  channel: "boifi:service-map:updates"

discovery:
  interval: "5m"

openapi:
  enabled: false
  paths:
    - "/swagger.json"
    - "/v3/api-docs"
    - "/openapi.json"
  timeout: "5s"

log:
  level: "info"
  format: "json"
```

---

## 5. 状态与生命周期

### ServiceMap 更新周期

```
┌─────────────────┐
│   Scheduler     │
│  (每 5 分钟)     │
└────────┬────────┘
         │ 触发
         ▼
┌─────────────────┐     ┌─────────────────┐
│  K8s Discovery  │────▶│ Jaeger Topology │
│ (VirtualService)│     │ (Dependencies)  │
└────────┬────────┘     └────────┬────────┘
         │                       │
         └───────────┬───────────┘
                     │ 聚合
                     ▼
              ┌─────────────┐
              │  ServiceMap │
              │ (内存组装)   │
              └──────┬──────┘
                     │ 序列化
                     ▼
              ┌─────────────┐
              │    Redis    │
              │ SET + PUB   │
              └─────────────┘
```

### 错误状态

当发现过程中出现错误时，ServiceMap 仍然会被生成，但会在 `metadata.errors` 中记录：

```json
{
  "metadata": {
    "stale": false,
    "errors": [
      "jaeger: connection timeout",
      "openapi: service 'reviews' returned 404"
    ]
  }
}
```

---

## 6. 验证规则

### ServiceMap 验证

| 字段 | 验证规则 |
|------|----------|
| `Timestamp` | 必须是有效的时间戳 |
| `Services` | 可以为空 map，但不能为 nil |
| `Services[].Name` | 非空字符串 |
| `Services[].APIs` | 可以为空数组 |
| `Services[].APIs[].Method` | 有效 HTTP 方法或 `*` |
| `Services[].APIs[].Path` | 非空字符串 |
| `Services[].APIs[].MatchType` | `exact` / `prefix` / `regex` 之一 |
| `Topology` | 可以为空数组 |
| `Topology[].Source` | 非空字符串 |
| `Topology[].Target` | 非空字符串 |
| `Topology[].CallCount` | >= 0 |

### 配置验证

| 字段 | 验证规则 |
|------|----------|
| `Jaeger.URL` | 有效的 HTTP URL |
| `Jaeger.Lookback` | > 0 |
| `Redis.Address` | 非空，格式 `host:port` |
| `Discovery.Interval` | >= 1 分钟 |
