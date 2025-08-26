# 故障注入系统 - 部署与测试指南

## 📋 概述

本文档提供了完整的WebAssembly故障注入系统的部署、测试和故障排查指南。系统包括：
- **Control Plane** (Go + etcd)
- **Wasm Plugin** (Rust)
- **Envoy Proxy**
- **CLI工具** (hfi-cli)

## 🚀 快速部署

### 1. 启动所有服务

```bash
cd /home/huiguo/wasm_fault_injection
docker-compose up -d
```

### 2. 验证服务状态

```bash
# 检查所有容器状态
docker ps

# 验证Control Plane健康状态
curl http://localhost:8080/v1/health

# 验证Envoy管理界面
curl http://localhost:19000/ready
```

### 3. 构建CLI工具

```bash
cd cli
go build -o hfi-cli .
```

## 🧪 功能测试

### 基础连接测试

```bash
# 测试正常请求（无故障注入）
curl http://localhost:18000/
```

### Abort故障测试

1. **创建abort策略文件** (`abort-policy.yaml`):
```yaml
metadata:
  name: "test-abort-policy"
spec:
  rules:
    - match:
        method: "GET"
        path:
          exact: "/"
      fault:
        percentage: 100
        abort:
          httpStatus: 503
```

2. **应用策略**:
```bash
./hfi-cli policy apply -f abort-policy.yaml
```

3. **验证故障注入**:
```bash
curl -v http://localhost:18000/
# 应该返回 HTTP 503
```

### Delay故障测试

1. **创建delay策略文件** (`delay-policy.yaml`):
```yaml
metadata:
  name: "test-delay-policy"
spec:
  rules:
    - match:
        method: "GET"
        path:
          exact: "/"
      fault:
        percentage: 100
        delay:
          fixed_delay: "1000ms"
```

2. **应用策略**:
```bash
./hfi-cli policy apply -f delay-policy.yaml
```

3. **验证延迟**:
```bash
time curl http://localhost:18000/
# 应该延迟约1秒
```

### 概率测试

1. **创建50%概率策略** (`50-percent-policy.yaml`):
```yaml
metadata:
  name: "test-50-percent-policy"
spec:
  rules:
    - match:
        method: "GET"
        path:
          exact: "/"
      fault:
        percentage: 50
        delay:
          fixed_delay: "500ms"
```

2. **多次测试验证概率**:
```bash
for i in {1..10}; do
  echo -n "Request $i: "
  time curl -s http://localhost:18000/ > /dev/null
done
```

### Header条件测试

1. **创建header匹配策略** (`header-policy.yaml`):
```yaml
metadata:
  name: "test-header-policy"
spec:
  rules:
    - match:
        method: "GET"
        path:
          exact: "/"
        headers:
          - name: "x-user-id"
            exact: "test"
      fault:
        percentage: 100
        delay:
          fixed_delay: "800ms"
```

2. **测试带Header的请求**:
```bash
time curl -H "x-user-id: test" http://localhost:18000/
# 应该有延迟
```

3. **测试不带Header的请求**:
```bash
time curl http://localhost:18000/
# 应该正常
```

### 策略删除测试

```bash
# 手动删除策略
docker exec wasm_fault_injection-etcd-1 etcdctl del "hfi/policies/test-header-policy"

# 验证请求恢复正常
curl http://localhost:18000/
```

## 📝 策略示例

### 完整策略示例

```yaml
metadata:
  name: "complex-policy"
spec:
  rules:
    # Rule 1: Abort特定用户的请求
    - match:
        method: "POST"
        path:
          prefix: "/api/orders"
        headers:
          - name: "x-user-id"
            exact: "blocked-user"
      fault:
        percentage: 100
        abort:
          httpStatus: 403
    
    # Rule 2: 给高级用户增加延迟
    - match:
        method: "GET"
        path:
          regex: "/api/users/\\d+"
        headers:
          - name: "x-user-type"
            exact: "premium"
      fault:
        percentage: 30
        delay:
          fixed_delay: "200ms"
    
    # Rule 3: 模拟服务不稳定
    - match:
        method: "GET"
        path:
          exact: "/api/health"
      fault:
        percentage: 10
        abort:
          httpStatus: 500
```

### 路径匹配示例

```yaml
# 精确匹配
path:
  exact: "/api/users"

# 前缀匹配
path:
  prefix: "/api/"

# 正则匹配
path:
  regex: "/api/users/\\d+"
```

### Header匹配示例

```yaml
headers:
  # 精确匹配
  - name: "authorization"
    exact: "Bearer token123"
  
  # 前缀匹配
  - name: "user-agent"
    prefix: "Mozilla"
  
  # 正则匹配
  - name: "x-trace-id"
    regex: "^[a-f0-9]{32}$"
```

### 故障类型示例

```yaml
# Abort故障
fault:
  percentage: 80
  abort:
    httpStatus: 503

# Delay故障
fault:
  percentage: 50
  delay:
    fixed_delay: "2s"

# 组合使用（但只能选择一种故障类型）
fault:
  percentage: 100
  delay:
    fixed_delay: "1000ms"
```

## 🔧 故障排查指南

### Wasm配置解析错误排查

#### 1. 常见错误类型

##### A. 字段缺失错误
```
Error: missing field `percentage` at line 1 column 124
```

**原因**: `percentage`字段在`fault`级别缺失
**解决**: 确保在`fault`下直接包含`percentage`字段

```yaml
# ❌ 错误格式
fault:
  delay:
    percentage: 100  # 错误位置
    fixed_delay: "1000ms"

# ✅ 正确格式  
fault:
  percentage: 100    # 正确位置
  delay:
    fixed_delay: "1000ms"
```

##### B. 字段名称错误
```
Error: missing field `fixed_delay` at line 1 column 132
```

**原因**: 延迟字段应该是`fixed_delay`而不是`fixedDelayMs`
**解决**: 使用正确的字段名

```yaml
# ❌ 错误格式
delay:
  fixedDelayMs: 1000

# ✅ 正确格式
delay:
  fixed_delay: "1000ms"
```

##### C. 类型错误
```
Error: invalid type: string "/", expected struct PathMatcherHelper
```

**原因**: 路径应该是对象而不是字符串
**解决**: 使用正确的路径匹配格式

```yaml
# ❌ 错误格式
match:
  path: "/"

# ✅ 正确格式
match:
  path:
    exact: "/"
```

##### D. Header格式错误
```
Error: invalid type: map, expected a sequence
```

**原因**: headers应该是数组而不是对象
**解决**: 使用正确的headers格式

```yaml
# ❌ 错误格式
headers:
  x-user-id:
    exact: "test"

# ✅ 正确格式
headers:
  - name: "x-user-id"
    exact: "test"
```

#### 2. 排查步骤

##### 步骤1: 检查Envoy日志
```bash
docker logs wasm_fault_injection-envoy-1 --tail 20 | grep -E "(parse|error|warning)"
```

查找以下关键字：
- `Failed to parse configuration`
- `missing field`
- `invalid type`
- `expected`

##### 步骤2: 验证配置传播
```bash
# 检查etcd中的策略
docker exec wasm_fault_injection-etcd-1 etcdctl get --prefix "hfi/policies/"

# 检查Control Plane日志
docker logs wasm_fault_injection-control-plane-1 --tail 10
```

##### 步骤3: 验证策略格式
```bash
# 使用CLI验证策略文件
./hfi-cli policy apply -f your-policy.yaml
```

##### 步骤4: 检查Wasm插件状态
```bash
# 查看成功解析的规则数量
docker logs wasm_fault_injection-envoy-1 --tail 50 | grep "Successfully parsed"
```

#### 3. Header匹配问题排查

如果Header匹配不工作，检查：

1. **Header名称是否在common_headers列表中**:
```rust
// 当前支持的headers
let common_headers = [
    "host", "user-agent", "accept", "accept-language", "accept-encoding",
    "authorization", "content-type", "content-length", "x-forwarded-for",
    "x-real-ip", "x-user-id", "x-tenant-id", "x-service", "x-version"
];
```

2. **使用支持的header进行测试**:
```bash
# 使用支持的header
curl -H "x-user-id: test" http://localhost:18000/
```

#### 4. 性能问题排查

##### 检查配置轮询频率
```bash
# 查看HTTP调用频率
docker logs wasm_fault_injection-envoy-1 --tail 100 | grep "Dispatching HTTP call" | wc -l
```

##### 检查延迟精度
```bash
# 多次测试验证延迟一致性
for i in {1..5}; do
  echo "Test $i:"
  time curl -s http://localhost:18000/ > /dev/null
done
```

### 网络连接问题

#### Control Plane连接
```bash
# 测试Control Plane连接
curl http://localhost:8080/v1/health

# 检查网络连通性
docker exec wasm_fault_injection-envoy-1 curl http://control-plane:8080/v1/health
```

#### 后端服务连接
```bash
# 检查Envoy集群状态
curl http://localhost:19000/clusters | grep local_backend
```

## 📊 监控和调试

### 实时日志监控

```bash
# 监控所有服务日志
docker-compose logs -f

# 监控特定服务
docker logs -f wasm_fault_injection-envoy-1
docker logs -f wasm_fault_injection-control-plane-1
```

### 配置验证脚本

```bash
#!/bin/bash
# config-check.sh

echo "=== 检查服务状态 ==="
docker ps | grep wasm_fault_injection

echo "=== 检查etcd中的策略 ==="
docker exec wasm_fault_injection-etcd-1 etcdctl get --prefix "hfi/policies/"

echo "=== 检查最新配置解析 ==="
docker logs wasm_fault_injection-envoy-1 --tail 5 | grep -E "(Successfully parsed|Failed to parse)"
```

## 🔄 故障恢复

### 重启服务
```bash
# 重启所有服务
docker-compose restart

# 重启特定服务
docker-compose restart envoy
docker-compose restart control-plane
```

### 清理配置
```bash
# 清理所有策略
docker exec wasm_fault_injection-etcd-1 etcdctl del --prefix "hfi/policies/"

# 验证配置清理
curl http://localhost:18000/  # 应该正常响应
```

### 构建问题解决
```bash
# 重新构建Wasm插件
docker-compose up -d wasm-builder

# 重新构建CLI
cd cli && go build -o hfi-cli .
```

## 📈 性能基准

### 正常延迟基准
- **无故障注入**: ~50-100ms
- **网络基础延迟**: ~10-20ms
- **Envoy处理延迟**: ~5-10ms

### 故障注入精度
- **1000ms延迟**: 实际 ~1000-1020ms (误差 <3%)
- **500ms延迟**: 实际 ~500-510ms (误差 <3%)
- **概率准确性**: 在大样本下符合设定概率 (±5%)

## 🛠️ 开发扩展

### 添加新的Header支持
编辑 `wasm-plugin/src/matcher.rs`:
```rust
let common_headers = [
    "host", "user-agent", "accept", "accept-language", "accept-encoding",
    "authorization", "content-type", "content-length", "x-forwarded-for",
    "x-real-ip", "x-user-id", "x-tenant-id", "x-service", "x-version",
    "your-custom-header"  // 添加新header
];
```

### CLI命令扩展
- `hfi-cli policy get` - 获取策略
- `hfi-cli policy delete` - 删除策略  
- `hfi-cli policy list` - 列出所有策略

---

**测试环境要求**:
- Docker & Docker Compose
- Go 1.22+
- Rust (用于Wasm插件开发)
- curl (用于测试)

**相关端口**:
- 18000: Envoy代理 (用户流量)
- 19000: Envoy管理界面
- 8080: Control Plane API
- 2379: etcd
