# API 参考文档

本文档提供 HFI (HTTP Fault Injection) 系统的完整 API 参考，包括故障注入策略的资源规范和 Control Plane REST API 的详细说明。

# 字段详细说明

## Metadata 字段

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--|||--||
| `metadata.name` | string | ✅ | - | 策略的唯一名称，必须符合 Kubernetes 命名规范 |
| `metadata.namespace` | string | ❌ | "default" | 策略所属的命名空间 |
| `metadata.labels` | map[string]string | ❌ | {} | 用于分组和选择的标签 |
| `metadata.annotations` | map[string]string | ❌ | {} | 附加的元数据信息 |

## Spec 核心字段

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--|||--||
| `spec.priority` | integer | ❌ | 0 | 策略优先级，数字越大优先级越高 (0-1000) |
| `spec.enabled` | boolean | ❌ | true | 策略是否启用 |

## Match 匹配字段

### HTTP 方法匹配

| 字段名 | 类型 | 必需 | 说明 |
|--||||
| `spec.match.method.exact` | string | ❌ | 精确匹配 HTTP 方法 (GET, POST, PUT, DELETE 等) |
| `spec.match.method.prefix` | string | ❌ | 前缀匹配 HTTP 方法 |
| `spec.match.method.regex` | string | ❌ | 正则表达式匹配 HTTP 方法 |

### URL 路径匹配

| 字段名 | 类型 | 必需 | 说明 |
|--||||
| `spec.match.path.exact` | string | ❌ | 精确匹配 URL 路径 |
| `spec.match.path.prefix` | string | ❌ | 前缀匹配 URL 路径，最常用的匹配方式 |
| `spec.match.path.regex` | string | ❌ | 正则表达式匹配 URL 路径 |

### 请求头匹配

| 字段名 | 类型 | 必需 | 说明 |
|--||||
| `spec.match.headers[].name` | string | ✅ | 头部名称 (不区分大小写) |
| `spec.match.headers[].value.exact` | string | ❌ | 精确匹配头部值 |
| `spec.match.headers[].value.prefix` | string | ❌ | 前缀匹配头部值 |
| `spec.match.headers[].value.regex` | string | ❌ | 正则表达式匹配头部值 |
| `spec.match.headers[].present` | boolean | ❌ | 仅检查头部是否存在，忽略值 |
| `spec.match.headers[].invert` | boolean | ❌ | 反向匹配，当不满足条件时匹配 |

### 查询参数匹配

| 字段名 | 类型 | 必需 | 说明 |
|--||||
| `spec.match.queryParams[].name` | string | ✅ | 查询参数名称 |
| `spec.match.queryParams[].value.exact` | string | ❌ | 精确匹配参数值 |
| `spec.match.queryParams[].value.prefix` | string | ❌ | 前缀匹配参数值 |
| `spec.match.queryParams[].value.regex` | string | ❌ | 正则表达式匹配参数值 |
| `spec.match.queryParams[].present` | boolean | ❌ | 仅检查参数是否存在 |

### 其他匹配字段

| 字段名 | 类型 | 必需 | 说明 |
|--||||
| `spec.match.body.regex` | string | ❌ | 正则表达式匹配请求体内容 (谨慎使用) |
| `spec.match.sourceIP[]` | string | ❌ | 源 IP 地址或 CIDR 范围 |

## Fault 故障注入字段

### 延迟故障

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--|||--||
| `spec.fault.delay.percentage` | float | ✅ | - | 受影响的请求百分比 (0.0-100.0) |
| `spec.fault.delay.fixedDelay` | string | ❌ | - | 固定延迟时间 (如 "1s", "500ms") |
| `spec.fault.delay.randomDelay.min` | string | ❌ | - | 随机延迟最小值 |
| `spec.fault.delay.randomDelay.max` | string | ❌ | - | 随机延迟最大值 |
| `spec.fault.delay.normalDelay.mean` | string | ❌ | - | 正态分布延迟均值 |
| `spec.fault.delay.normalDelay.stddev` | string | ❌ | - | 正态分布延迟标准差 |

### 错误注入故障

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--|||--||
| `spec.fault.abort.percentage` | float | ✅ | - | 受影响的请求百分比 (0.0-100.0) |
| `spec.fault.abort.httpStatus` | integer | ✅ | - | HTTP 状态码 (400-599) |
| `spec.fault.abort.body` | string | ❌ | "" | 响应体内容 |
| `spec.fault.abort.headers[].name` | string | ❌ | - | 添加的响应头名称 |
| `spec.fault.abort.headers[].value` | string | ❌ | - | 添加的响应头值 |

### 限流故障

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--|||--||
| `spec.fault.rateLimit.percentage` | float | ✅ | - | 受影响的请求百分比 |
| `spec.fault.rateLimit.requestsPerSecond` | integer | ✅ | - | 每秒允许的请求数 |
| `spec.fault.rateLimit.burstSize` | integer | ❌ | requestsPerSecond | 突发请求容量 |
| `spec.fault.rateLimit.rejectStatus` | integer | ❌ | 429 | 超出限制时的 HTTP 状态码 |
| `spec.fault.rateLimit.rejectBody` | string | ❌ | "Rate limit exceeded" | 超出限制时的响应体 |

### 响应修改

| 字段名 | 类型 | 必需 | 说明 |
|--||||
| `spec.fault.responseModification.headers.add[].name` | string | ✅ | 要添加的响应头名称 |
| `spec.fault.responseModification.headers.add[].value` | string | ✅ | 要添加的响应头值 |
| `spec.fault.responseModification.headers.remove[]` | string | ❌ | 要删除的响应头名称 |
| `spec.fault.responseModification.headers.modify[].name` | string | ✅ | 要修改的响应头名称 |
| `spec.fault.responseModification.headers.modify[].value` | string | ✅ | 要修改的响应头新值 |
| `spec.fault.responseModification.body.replace.pattern` | string | ✅ | 要替换的正则表达式模式 |
| `spec.fault.responseModification.body.replace.replacement` | string | ✅ | 替换的内容 |

## Advanced 高级配置字段

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--|||--||
| `spec.advanced.sampling.strategy` | string | ❌ | "probabilistic" | 采样策略 (probabilistic, deterministic) |
| `spec.advanced.sampling.rate` | float | ❌ | 1.0 | 采样率 (0.0-1.0) |
| `spec.advanced.conditions.timeWindow.start` | string | ❌ | - | 生效开始时间 (HH:MM 格式) |
| `spec.advanced.conditions.timeWindow.end` | string | ❌ | - | 生效结束时间 (HH:MM 格式) |
| `spec.advanced.conditions.timeWindow.timezone` | string | ❌ | "UTC" | 时区 |
| `spec.advanced.recovery.enabled` | boolean | ❌ | false | 是否启用故障恢复 |
| `spec.advanced.recovery.healthCheckPath` | string | ❌ | "/health" | 健康检查路径 |
| `spec.advanced.recovery.recoveryThreshold` | integer | ❌ | 3 | 恢复阈值 |
| `spec.advanced.recovery.checkInterval` | string | ❌ | "30s" | 检查间隔 |

## Observability 可观测性字段

| 字段名 | 类型 | 必需 | 默认值 | 说明 |
|--|||--||
| `spec.observability.metrics.labels` | map[string]string | ❌ | {} | 自定义指标标签 |
| `spec.observability.logging.level` | string | ❌ | "INFO" | 日志级别 (DEBUG, INFO, WARN, ERROR) |
| `spec.observability.logging.sampleRate` | float | ❌ | 0.01 | 日志采样率 |
| `spec.observability.tracing.enabled` | boolean | ❌ | false | 是否启用链路追踪 |
| `spec.observability.tracing.sampleRate` | float | ❌ | 0.1 | 追踪采样率 |

 🌐 Control Plane REST API

# API 端点概览

| Method | Path | 描述 | 认证 |
|--||||
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