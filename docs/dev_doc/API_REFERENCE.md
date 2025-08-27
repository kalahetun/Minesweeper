# API 参考文档

本文档提供 HFI (HTTP Fault Injection) 系统的完整 API 参考，包括故障注入策略的资源规范和 Control Plane REST API 的详细说明。

## 📋 目录

- [FaultInjectionPolicy 资源规范](#faultinjectionpolicy-资源规范)
- [Control Plane REST API](#control-plane-rest-api)
- [错误码参考](#错误码参考)
- [使用示例](#使用示例)
- [最佳实践](#最佳实践)

## 📄 FaultInjectionPolicy 资源规范

### 完整示例

以下是一个包含所有可能字段的完整 `FaultInjectionPolicy` 示例：

```yaml
apiVersion: hfi.io/v1
kind: FaultInjectionPolicy
metadata:
  name: demo-fault-policy
  namespace: default
  labels:
    app: demo-service
    env: staging
  annotations:
    description: "演示延迟和错误注入策略"
    owner: "platform-team"
spec:
  # 策略优先级
  priority: 100
  
  # 生效条件
  enabled: true
  
  # 请求匹配规则
  match:
    # HTTP 方法匹配
    method:
      exact: "POST"  # 精确匹配
      # prefix: "P"  # 前缀匹配
      # regex: "POST|PUT"  # 正则匹配
    
    # URL 路径匹配
    path:
      prefix: "/api/v1/"  # 前缀匹配
      # exact: "/api/v1/users"  # 精确匹配
      # regex: "^/api/v[0-9]+/.*"  # 正则匹配
    
    # 请求头匹配
    headers:
      - name: "user-agent"
        value:
          prefix: "mobile"  # 匹配移动端请求
      - name: "x-request-id"
        value:
          regex: "^req-[0-9a-f]{8}-.*"  # 匹配特定格式的请求ID
      - name: "authorization"
        present: true  # 检查头部是否存在
    
    # 查询参数匹配
    queryParams:
      - name: "version"
        value:
          exact: "beta"
      - name: "debug"
        present: true
    
    # 请求体匹配 (慎用，影响性能)
    body:
      regex: '"userId":\\s*"[0-9]+"'  # 匹配包含数字用户ID的JSON
    
    # 源IP匹配
    sourceIP:
      - "192.168.1.0/24"
      - "10.0.0.100"
  
  # 故障注入配置
  fault:
    # 延迟故障
    delay:
      percentage: 50.0  # 50% 的请求受影响
      fixedDelay: "2s"  # 固定延迟2秒
      # 或使用随机延迟
      # randomDelay:
      #   min: "100ms"
      #   max: "5s"
      # 或使用正态分布延迟
      # normalDelay:
      #   mean: "1s"
      #   stddev: "200ms"
    
    # 错误注入故障
    abort:
      percentage: 10.0  # 10% 的请求受影响
      httpStatus: 503   # 返回 503 Service Unavailable
      body: |
        {
          "error": "Service temporarily unavailable",
          "code": "SERVICE_UNAVAILABLE",
          "retryAfter": 30
        }
      headers:
        - name: "retry-after"
          value: "30"
        - name: "x-fault-type"
          value: "abort"
    
    # 限流故障
    rateLimit:
      percentage: 100.0  # 所有匹配的请求都检查限流
      requestsPerSecond: 100  # 每秒最多100个请求
      burstSize: 200  # 突发容量200个请求
      rejectStatus: 429  # 超出限制时返回429
      rejectBody: |
        {
          "error": "Rate limit exceeded",
          "limit": 100,
          "retryAfter": 1
        }
    
    # 响应修改
    responseModification:
      # 修改响应头
      headers:
        add:
          - name: "x-processed-by"
            value: "hfi-proxy"
          - name: "x-policy-name"
            value: "demo-fault-policy"
        remove:
          - "server"
          - "x-powered-by"
        modify:
          - name: "cache-control"
            value: "no-cache, no-store"
      
      # 修改响应体 (仅限小响应体)
      body:
        replace:
          pattern: '"success":\\s*true'
          replacement: '"success": false, "injected": true'
  
  # 高级配置
  advanced:
    # 采样配置
    sampling:
      strategy: "probabilistic"  # 概率采样
      rate: 0.1  # 10% 采样率
    
    # 条件执行
    conditions:
      # 仅在特定时间窗口生效
      timeWindow:
        start: "09:00"
        end: "17:00"
        timezone: "Asia/Shanghai"
      
      # 负载条件
      loadCondition:
        metric: "cpu_usage"
        threshold: 80.0  # CPU使用率超过80%时生效
        operator: "gt"  # greater than
    
    # 故障恢复
    recovery:
      enabled: true
      healthCheckPath: "/health"
      recoveryThreshold: 3  # 连续3次健康检查通过后恢复
      checkInterval: "30s"
  
  # 监控和日志
  observability:
    # 指标标签
    metrics:
      labels:
        service: "demo-service"
        version: "v1.2.3"
    
    # 日志配置
    logging:
      level: "INFO"  # DEBUG, INFO, WARN, ERROR
      sampleRate: 0.01  # 1% 的请求记录详细日志
    
    # 追踪配置
    tracing:
      enabled: true
      sampleRate: 0.1
      spanTags:
        - key: "fault.type"
          value: "delay"
```

### 字段详细说明

#### Metadata 字段

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `metadata.name` | string | ✅ | - | 策略的唯一名称，必须符合 Kubernetes 命名规范 |
| `metadata.namespace` | string | ❌ | "default" | 策略所属的命名空间 |
| `metadata.labels` | map[string]string | ❌ | {} | 用于分组和选择的标签 |
| `metadata.annotations` | map[string]string | ❌ | {} | 附加的元数据信息 |

#### Spec 核心字段

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `spec.priority` | integer | ❌ | 0 | 策略优先级，数字越大优先级越高 (0-1000) |
| `spec.enabled` | boolean | ❌ | true | 策略是否启用 |

#### Match 匹配字段

##### HTTP 方法匹配

| 字段名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `spec.match.method.exact` | string | ❌ | 精确匹配 HTTP 方法 (GET, POST, PUT, DELETE 等) |
| `spec.match.method.prefix` | string | ❌ | 前缀匹配 HTTP 方法 |
| `spec.match.method.regex` | string | ❌ | 正则表达式匹配 HTTP 方法 |

##### URL 路径匹配

| 字段名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `spec.match.path.exact` | string | ❌ | 精确匹配 URL 路径 |
| `spec.match.path.prefix` | string | ❌ | 前缀匹配 URL 路径，最常用的匹配方式 |
| `spec.match.path.regex` | string | ❌ | 正则表达式匹配 URL 路径 |

##### 请求头匹配

| 字段名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `spec.match.headers[].name` | string | ✅ | 头部名称 (不区分大小写) |
| `spec.match.headers[].value.exact` | string | ❌ | 精确匹配头部值 |
| `spec.match.headers[].value.prefix` | string | ❌ | 前缀匹配头部值 |
| `spec.match.headers[].value.regex` | string | ❌ | 正则表达式匹配头部值 |
| `spec.match.headers[].present` | boolean | ❌ | 仅检查头部是否存在，忽略值 |
| `spec.match.headers[].invert` | boolean | ❌ | 反向匹配，当不满足条件时匹配 |

##### 查询参数匹配

| 字段名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `spec.match.queryParams[].name` | string | ✅ | 查询参数名称 |
| `spec.match.queryParams[].value.exact` | string | ❌ | 精确匹配参数值 |
| `spec.match.queryParams[].value.prefix` | string | ❌ | 前缀匹配参数值 |
| `spec.match.queryParams[].value.regex` | string | ❌ | 正则表达式匹配参数值 |
| `spec.match.queryParams[].present` | boolean | ❌ | 仅检查参数是否存在 |

##### 其他匹配字段

| 字段名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `spec.match.body.regex` | string | ❌ | 正则表达式匹配请求体内容 (谨慎使用) |
| `spec.match.sourceIP[]` | string | ❌ | 源 IP 地址或 CIDR 范围 |

#### Fault 故障注入字段

##### 延迟故障

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `spec.fault.delay.percentage` | float | ✅ | - | 受影响的请求百分比 (0.0-100.0) |
| `spec.fault.delay.fixedDelay` | string | ❌ | - | 固定延迟时间 (如 "1s", "500ms") |
| `spec.fault.delay.randomDelay.min` | string | ❌ | - | 随机延迟最小值 |
| `spec.fault.delay.randomDelay.max` | string | ❌ | - | 随机延迟最大值 |
| `spec.fault.delay.normalDelay.mean` | string | ❌ | - | 正态分布延迟均值 |
| `spec.fault.delay.normalDelay.stddev` | string | ❌ | - | 正态分布延迟标准差 |

##### 错误注入故障

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `spec.fault.abort.percentage` | float | ✅ | - | 受影响的请求百分比 (0.0-100.0) |
| `spec.fault.abort.httpStatus` | integer | ✅ | - | HTTP 状态码 (400-599) |
| `spec.fault.abort.body` | string | ❌ | "" | 响应体内容 |
| `spec.fault.abort.headers[].name` | string | ❌ | - | 添加的响应头名称 |
| `spec.fault.abort.headers[].value` | string | ❌ | - | 添加的响应头值 |

##### 限流故障

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `spec.fault.rateLimit.percentage` | float | ✅ | - | 受影响的请求百分比 |
| `spec.fault.rateLimit.requestsPerSecond` | integer | ✅ | - | 每秒允许的请求数 |
| `spec.fault.rateLimit.burstSize` | integer | ❌ | requestsPerSecond | 突发请求容量 |
| `spec.fault.rateLimit.rejectStatus` | integer | ❌ | 429 | 超出限制时的 HTTP 状态码 |
| `spec.fault.rateLimit.rejectBody` | string | ❌ | "Rate limit exceeded" | 超出限制时的响应体 |

##### 响应修改

| 字段名 | 类型 | 必需 | 说明 |
|--------|------|------|------|
| `spec.fault.responseModification.headers.add[].name` | string | ✅ | 要添加的响应头名称 |
| `spec.fault.responseModification.headers.add[].value` | string | ✅ | 要添加的响应头值 |
| `spec.fault.responseModification.headers.remove[]` | string | ❌ | 要删除的响应头名称 |
| `spec.fault.responseModification.headers.modify[].name` | string | ✅ | 要修改的响应头名称 |
| `spec.fault.responseModification.headers.modify[].value` | string | ✅ | 要修改的响应头新值 |
| `spec.fault.responseModification.body.replace.pattern` | string | ✅ | 要替换的正则表达式模式 |
| `spec.fault.responseModification.body.replace.replacement` | string | ✅ | 替换的内容 |

#### Advanced 高级配置字段

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `spec.advanced.sampling.strategy` | string | ❌ | "probabilistic" | 采样策略 (probabilistic, deterministic) |
| `spec.advanced.sampling.rate` | float | ❌ | 1.0 | 采样率 (0.0-1.0) |
| `spec.advanced.conditions.timeWindow.start` | string | ❌ | - | 生效开始时间 (HH:MM 格式) |
| `spec.advanced.conditions.timeWindow.end` | string | ❌ | - | 生效结束时间 (HH:MM 格式) |
| `spec.advanced.conditions.timeWindow.timezone` | string | ❌ | "UTC" | 时区 |
| `spec.advanced.recovery.enabled` | boolean | ❌ | false | 是否启用故障恢复 |
| `spec.advanced.recovery.healthCheckPath` | string | ❌ | "/health" | 健康检查路径 |
| `spec.advanced.recovery.recoveryThreshold` | integer | ❌ | 3 | 恢复阈值 |
| `spec.advanced.recovery.checkInterval` | string | ❌ | "30s" | 检查间隔 |

#### Observability 可观测性字段

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `spec.observability.metrics.labels` | map[string]string | ❌ | {} | 自定义指标标签 |
| `spec.observability.logging.level` | string | ❌ | "INFO" | 日志级别 (DEBUG, INFO, WARN, ERROR) |
| `spec.observability.logging.sampleRate` | float | ❌ | 0.01 | 日志采样率 |
| `spec.observability.tracing.enabled` | boolean | ❌ | false | 是否启用链路追踪 |
| `spec.observability.tracing.sampleRate` | float | ❌ | 0.1 | 追踪采样率 |

## 🌐 Control Plane REST API

### API 端点概览

| Method | Path | 描述 | 认证 |
|--------|------|------|------|
| GET | `/v1/health` | 健康检查 | ❌ |
| GET | `/v1/metrics` | Prometheus 指标 | ❌ |
| GET | `/v1/policies` | 获取策略列表 | ✅ |
| POST | `/v1/policies` | 创建新策略 | ✅ |
| GET | `/v1/policies/{id}` | 获取指定策略 | ✅ |
| PUT | `/v1/policies/{id}` | 更新指定策略 | ✅ |
| DELETE | `/v1/policies/{id}` | 删除指定策略 | ✅ |
| GET | `/v1/policies/{id}/status` | 获取策略状态 | ✅ |
| POST | `/v1/policies/{id}/enable` | 启用策略 | ✅ |
| POST | `/v1/policies/{id}/disable` | 禁用策略 | ✅ |
| GET | `/v1/config/stream` | SSE 配置流 | ✅ |
| GET | `/v1/stats` | 系统统计信息 | ✅ |

### 详细 API 规范

#### 健康检查

**GET /v1/health**

获取系统健康状态。

**请求参数**: 无

**成功响应**: `200 OK`
```json
{
  "status": "healthy",
  "timestamp": "2025-08-27T10:30:00Z",
  "version": "1.0.0",
  "components": {
    "storage": "healthy",
    "distributor": "healthy",
    "metrics": "healthy"
  },
  "uptime": "72h30m15s"
}
```

**错误响应**: 
- `503 Service Unavailable`: 系统不健康

---

#### 获取指标

**GET /v1/metrics**

获取 Prometheus 格式的指标数据。

**请求参数**: 无

**成功响应**: `200 OK`
```
# HELP hfi_policies_total Total number of fault injection policies
# TYPE hfi_policies_total gauge
hfi_policies_total 5

# HELP hfi_requests_total Total number of processed requests
# TYPE hfi_requests_total counter
hfi_requests_total{policy="demo-policy",fault_type="delay"} 1234
```

---

#### 获取策略列表

**GET /v1/policies**

获取所有故障注入策略的列表。

**查询参数**:
| 参数 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `limit` | integer | ❌ | 50 | 返回的最大策略数量 (1-1000) |
| `offset` | integer | ❌ | 0 | 偏移量，用于分页 |
| `namespace` | string | ❌ | "" | 过滤指定命名空间的策略 |
| `enabled` | boolean | ❌ | - | 过滤启用/禁用的策略 |
| `labels` | string | ❌ | "" | 标签选择器 (key=value,key2=value2) |
| `sort` | string | ❌ | "name" | 排序字段 (name, priority, created) |
| `order` | string | ❌ | "asc" | 排序顺序 (asc, desc) |

**成功响应**: `200 OK`
```json
{
  "policies": [
    {
      "metadata": {
        "name": "demo-policy",
        "namespace": "default",
        "uid": "550e8400-e29b-41d4-a716-446655440000",
        "created": "2025-08-27T10:00:00Z",
        "updated": "2025-08-27T10:15:00Z"
      },
      "spec": {
        "priority": 100,
        "enabled": true,
        "match": {
          "path": {
            "prefix": "/api/"
          }
        },
        "fault": {
          "delay": {
            "percentage": 10.0,
            "fixedDelay": "1s"
          }
        }
      },
      "status": {
        "phase": "Active",
        "conditions": [
          {
            "type": "Ready",
            "status": "True",
            "lastTransitionTime": "2025-08-27T10:00:00Z"
          }
        ],
        "appliedGeneration": 1,
        "observedGeneration": 1
      }
    }
  ],
  "pagination": {
    "total": 1,
    "limit": 50,
    "offset": 0,
    "hasMore": false
  }
}
```

**错误响应**:
- `400 Bad Request`: 无效的查询参数
- `401 Unauthorized`: 认证失败
- `500 Internal Server Error`: 服务器内部错误

---

#### 创建策略

**POST /v1/policies**

创建新的故障注入策略。

**请求头**:
```
Content-Type: application/json
Authorization: Bearer <token>
```

**请求体**: `FaultInjectionPolicy` JSON 对象
```json
{
  "metadata": {
    "name": "new-policy",
    "namespace": "default",
    "labels": {
      "app": "test-service"
    }
  },
  "spec": {
    "priority": 100,
    "enabled": true,
    "match": {
      "path": {
        "prefix": "/api/test/"
      }
    },
    "fault": {
      "delay": {
        "percentage": 20.0,
        "fixedDelay": "500ms"
      }
    }
  }
}
```

**成功响应**: `201 Created`
```json
{
  "metadata": {
    "name": "new-policy",
    "namespace": "default",
    "uid": "550e8400-e29b-41d4-a716-446655440001",
    "created": "2025-08-27T10:30:00Z",
    "updated": "2025-08-27T10:30:00Z"
  },
  "spec": {
    // ... 完整的策略规范
  },
  "status": {
    "phase": "Pending",
    "conditions": [
      {
        "type": "Ready",
        "status": "False",
        "reason": "Creating",
        "message": "Policy is being created",
        "lastTransitionTime": "2025-08-27T10:30:00Z"
      }
    ]
  }
}
```

**错误响应**:
- `400 Bad Request`: 请求体格式错误或验证失败
- `401 Unauthorized`: 认证失败
- `409 Conflict`: 策略名称已存在
- `422 Unprocessable Entity`: 策略配置无效
- `500 Internal Server Error`: 服务器内部错误

---

#### 获取指定策略

**GET /v1/policies/{id}**

获取指定 ID 的策略详细信息。

**路径参数**:
- `id` (string): 策略的名称或 UID

**成功响应**: `200 OK`
```json
{
  "metadata": {
    "name": "demo-policy",
    "namespace": "default",
    "uid": "550e8400-e29b-41d4-a716-446655440000",
    "created": "2025-08-27T10:00:00Z",
    "updated": "2025-08-27T10:15:00Z"
  },
  "spec": {
    // ... 完整的策略规范
  },
  "status": {
    "phase": "Active",
    "conditions": [
      {
        "type": "Ready",
        "status": "True",
        "lastTransitionTime": "2025-08-27T10:00:00Z"
      }
    ],
    "appliedGeneration": 1,
    "observedGeneration": 1,
    "metrics": {
      "totalRequests": 1000,
      "faultedRequests": 100,
      "lastApplied": "2025-08-27T10:29:30Z"
    }
  }
}
```

**错误响应**:
- `401 Unauthorized`: 认证失败
- `404 Not Found`: 策略不存在
- `500 Internal Server Error`: 服务器内部错误

---

#### 更新策略

**PUT /v1/policies/{id}**

更新现有的故障注入策略。

**路径参数**:
- `id` (string): 策略的名称或 UID

**请求头**:
```
Content-Type: application/json
Authorization: Bearer <token>
If-Match: "1"  # 可选，用于乐观锁
```

**请求体**: 更新后的 `FaultInjectionPolicy` JSON 对象

**成功响应**: `200 OK`
```json
{
  "metadata": {
    "name": "demo-policy",
    "namespace": "default",
    "uid": "550e8400-e29b-41d4-a716-446655440000",
    "created": "2025-08-27T10:00:00Z",
    "updated": "2025-08-27T10:45:00Z"
  },
  "spec": {
    // ... 更新后的策略规范
  },
  "status": {
    "phase": "Updating",
    "conditions": [
      {
        "type": "Ready",
        "status": "False",
        "reason": "Updating",
        "message": "Policy is being updated",
        "lastTransitionTime": "2025-08-27T10:45:00Z"
      }
    ]
  }
}
```

**错误响应**:
- `400 Bad Request`: 请求体格式错误或验证失败
- `401 Unauthorized`: 认证失败
- `404 Not Found`: 策略不存在
- `409 Conflict`: 版本冲突 (如果使用了 If-Match)
- `422 Unprocessable Entity`: 策略配置无效
- `500 Internal Server Error`: 服务器内部错误

---

#### 删除策略

**DELETE /v1/policies/{id}**

删除指定的故障注入策略。

**路径参数**:
- `id` (string): 策略的名称或 UID

**查询参数**:
| 参数 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `gracePeriod` | integer | ❌ | 30 | 优雅删除期间 (秒) |
| `force` | boolean | ❌ | false | 是否强制立即删除 |

**成功响应**: `204 No Content`

**错误响应**:
- `401 Unauthorized`: 认证失败
- `404 Not Found`: 策略不存在
- `409 Conflict`: 策略正在被使用，无法删除
- `500 Internal Server Error`: 服务器内部错误

---

#### 获取策略状态

**GET /v1/policies/{id}/status**

获取策略的运行状态和统计信息。

**路径参数**:
- `id` (string): 策略的名称或 UID

**成功响应**: `200 OK`
```json
{
  "name": "demo-policy",
  "namespace": "default",
  "status": {
    "phase": "Active",
    "enabled": true,
    "lastApplied": "2025-08-27T10:29:30Z",
    "conditions": [
      {
        "type": "Ready",
        "status": "True",
        "reason": "PolicyActive",
        "message": "Policy is active and processing requests",
        "lastTransitionTime": "2025-08-27T10:00:00Z"
      }
    ]
  },
  "metrics": {
    "totalRequests": 1000,
    "matchedRequests": 200,
    "faultedRequests": 100,
    "delayedRequests": 60,
    "abortedRequests": 40,
    "rateLimitedRequests": 0,
    "averageDelay": "520ms",
    "errorRate": 0.04,
    "lastHourStats": {
      "requests": 150,
      "faults": 15
    }
  },
  "distribution": {
    "byDatacenter": {
      "us-east-1": 600,
      "us-west-2": 400
    },
    "byUserAgent": {
      "mobile": 300,
      "desktop": 700
    }
  }
}
```

**错误响应**:
- `401 Unauthorized`: 认证失败
- `404 Not Found`: 策略不存在
- `500 Internal Server Error`: 服务器内部错误

---

#### 启用策略

**POST /v1/policies/{id}/enable**

启用指定的故障注入策略。

**路径参数**:
- `id` (string): 策略的名称或 UID

**成功响应**: `200 OK`
```json
{
  "message": "Policy enabled successfully",
  "timestamp": "2025-08-27T10:50:00Z"
}
```

**错误响应**:
- `401 Unauthorized`: 认证失败
- `404 Not Found`: 策略不存在
- `409 Conflict`: 策略已经启用
- `500 Internal Server Error`: 服务器内部错误

---

#### 禁用策略

**POST /v1/policies/{id}/disable**

禁用指定的故障注入策略。

**路径参数**:
- `id` (string): 策略的名称或 UID

**请求体** (可选):
```json
{
  "reason": "Maintenance window",
  "gracePeriod": 60
}
```

**成功响应**: `200 OK`
```json
{
  "message": "Policy disabled successfully",
  "timestamp": "2025-08-27T10:50:00Z"
}
```

**错误响应**:
- `401 Unauthorized`: 认证失败
- `404 Not Found`: 策略不存在
- `409 Conflict`: 策略已经禁用
- `500 Internal Server Error`: 服务器内部错误

---

#### 配置流 (SSE)

**GET /v1/config/stream**

建立 Server-Sent Events 连接，实时接收配置更新。

**请求头**:
```
Accept: text/event-stream
Cache-Control: no-cache
Authorization: Bearer <token>
```

**查询参数**:
| 参数 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `namespace` | string | ❌ | "" | 订阅指定命名空间的配置 |
| `lastEventId` | string | ❌ | "" | 上次接收的事件 ID，用于断线重连 |

**成功响应**: `200 OK`
```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

data: {"type":"connected","timestamp":"2025-08-27T10:55:00Z","version":"1.0.0"}

id: 1
event: policy-created
data: {"policy":{"metadata":{"name":"new-policy"},"spec":{...}}}

id: 2
event: policy-updated
data: {"policy":{"metadata":{"name":"demo-policy"},"spec":{...}}}

id: 3
event: policy-deleted
data: {"name":"old-policy","namespace":"default"}

id: 4
event: config-compiled
data: {"version":"v1.2.3","timestamp":"2025-08-27T10:56:00Z","policies":5}

: heartbeat
```

**事件类型**:
- `connected`: 连接建立
- `policy-created`: 策略创建
- `policy-updated`: 策略更新
- `policy-deleted`: 策略删除
- `config-compiled`: 配置编译完成
- `heartbeat`: 心跳 (注释形式)

**错误响应**:
- `401 Unauthorized`: 认证失败
- `406 Not Acceptable`: 不支持的 Accept 头
- `500 Internal Server Error`: 服务器内部错误

---

#### 系统统计

**GET /v1/stats**

获取系统整体统计信息。

**查询参数**:
| 参数 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `window` | string | ❌ | "1h" | 统计时间窗口 (1m, 5m, 1h, 24h) |
| `granularity` | string | ❌ | "1m" | 数据粒度 (1m, 5m, 15m, 1h) |

**成功响应**: `200 OK`
```json
{
  "timestamp": "2025-08-27T11:00:00Z",
  "window": "1h",
  "system": {
    "uptime": "72h30m45s",
    "version": "1.0.0",
    "totalPolicies": 5,
    "activePolicies": 4,
    "totalRequests": 50000,
    "faultedRequests": 2500,
    "errorRate": 0.05
  },
  "performance": {
    "averageLatency": "12ms",
    "p95Latency": "45ms",
    "p99Latency": "120ms",
    "throughput": 850.5,
    "cpuUsage": 25.6,
    "memoryUsage": 512.3
  },
  "faults": {
    "delays": {
      "count": 1500,
      "averageDuration": "520ms"
    },
    "aborts": {
      "count": 800,
      "statusCodes": {
        "500": 400,
        "503": 300,
        "429": 100
      }
    },
    "rateLimits": {
      "count": 200,
      "rejectedRequests": 180
    }
  },
  "timeSeries": [
    {
      "timestamp": "2025-08-27T10:00:00Z",
      "requests": 833,
      "faults": 42,
      "errors": 5
    },
    {
      "timestamp": "2025-08-27T10:01:00Z",
      "requests": 847,
      "faults": 43,
      "errors": 3
    }
  ]
}
```

**错误响应**:
- `400 Bad Request`: 无效的查询参数
- `401 Unauthorized`: 认证失败
- `500 Internal Server Error`: 服务器内部错误

## ❌ 错误码参考

### HTTP 状态码

| 状态码 | 说明 | 常见原因 |
|--------|------|----------|
| 400 Bad Request | 请求格式错误 | JSON 格式错误、字段验证失败、参数类型错误 |
| 401 Unauthorized | 认证失败 | Token 无效、Token 过期、缺少认证头 |
| 403 Forbidden | 权限不足 | 没有操作权限、命名空间访问限制 |
| 404 Not Found | 资源不存在 | 策略不存在、API 端点不存在 |
| 409 Conflict | 资源冲突 | 策略名称重复、版本冲突、状态冲突 |
| 422 Unprocessable Entity | 业务逻辑错误 | 策略配置无效、依赖关系错误 |
| 429 Too Many Requests | 请求频率限制 | API 调用频率过高 |
| 500 Internal Server Error | 服务器内部错误 | 数据库错误、网络错误、Bug |
| 503 Service Unavailable | 服务不可用 | 系统维护、组件故障、过载保护 |

### 业务错误码

```json
{
  "error": {
    "code": "POLICY_VALIDATION_FAILED",
    "message": "Policy validation failed: invalid delay percentage",
    "details": {
      "field": "spec.fault.delay.percentage",
      "value": "150.0",
      "constraint": "must be between 0.0 and 100.0"
    },
    "requestId": "req-123e4567-e89b-12d3-a456-426614174000"
  }
}
```

常见业务错误码:

| 错误码 | 说明 | HTTP 状态 |
|--------|------|-----------|
| `POLICY_NOT_FOUND` | 策略不存在 | 404 |
| `POLICY_ALREADY_EXISTS` | 策略已存在 | 409 |
| `POLICY_VALIDATION_FAILED` | 策略验证失败 | 422 |
| `INVALID_REQUEST_FORMAT` | 请求格式无效 | 400 |
| `AUTHENTICATION_REQUIRED` | 需要认证 | 401 |
| `PERMISSION_DENIED` | 权限被拒绝 | 403 |
| `RATE_LIMIT_EXCEEDED` | 速率限制超出 | 429 |
| `STORAGE_ERROR` | 存储错误 | 500 |
| `CONFIGURATION_ERROR` | 配置错误 | 500 |
| `SERVICE_UNAVAILABLE` | 服务不可用 | 503 |

## 📝 使用示例

### 创建简单延迟策略

```bash
curl -X POST http://localhost:8080/v1/policies \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "metadata": {
      "name": "api-delay",
      "namespace": "production"
    },
    "spec": {
      "priority": 100,
      "match": {
        "path": {
          "prefix": "/api/v1/"
        }
      },
      "fault": {
        "delay": {
          "percentage": 10.0,
          "fixedDelay": "500ms"
        }
      }
    }
  }'
```

### 监听配置更新

```bash
curl -N -H "Accept: text/event-stream" \
  -H "Authorization: Bearer your-token" \
  http://localhost:8080/v1/config/stream
```

### 获取策略统计

```bash
curl -H "Authorization: Bearer your-token" \
  "http://localhost:8080/v1/policies/api-delay/status"
```

## 💡 最佳实践

### 1. 策略设计原则

- **渐进式部署**: 从低百分比开始，逐步增加故障注入比例
- **合理优先级**: 使用优先级避免策略冲突
- **精确匹配**: 使用具体的匹配条件，避免影响意外的请求
- **监控告警**: 配置适当的监控和告警

### 2. API 调用最佳实践

- **幂等性**: 使用 PUT 进行更新操作，确保幂等性
- **版本控制**: 使用 If-Match 头进行乐观锁控制
- **错误处理**: 实现适当的重试和错误处理逻辑
- **限流保护**: 实现客户端限流，避免过载服务器

### 3. 安全考虑

- **认证授权**: 所有 API 调用都应进行适当的认证和授权
- **输入验证**: 在客户端和服务端都进行输入验证
- **敏感信息**: 避免在日志中记录敏感信息
- **网络安全**: 使用 HTTPS 进行 API 调用

---

**相关文档**:
- [快速开始指南](../QUICKSTART.md)
- [系统架构文档](ARCHITECTURE.md)
- [Control Plane 深度解析](CONTROL_PLANE_DEEP_DIVE.md)
- [CLI 工具文档](../cli/README.md)
