# Wasm插件配置解析错误 - 完整排查指南

## 🔍 错误分析方法论

### 第一步：识别错误来源
Wasm配置解析错误通常出现在Envoy日志中，格式如下：
```
[timestamp][thread][warning][wasm] [source/extensions/common/wasm/context.cc:1174] 
wasm log hfi_root hfi_vm: Failed to parse configuration from control plane: [错误详情]
```

### 第二步：定位错误类型
根据错误信息分类：
1. **字段相关错误** (`missing field`, `unknown field`)
2. **类型相关错误** (`invalid type`, `expected`)
3. **格式相关错误** (`at line X column Y`)
4. **值相关错误** (`invalid value`)

## 📋 常见错误目录

### 1. 字段缺失错误

#### 1.1 缺少percentage字段
```
Error: missing field `percentage` at line 1 column 124
```

**问题分析**: 
- `percentage`字段必须在`fault`级别，不是在`abort`或`delay`内部
- 这是Wasm插件Rust结构体的要求

**错误示例**:
```yaml
# ❌ 错误：percentage在delay内部
fault:
  delay:
    percentage: 100
    fixed_delay: "1000ms"

# ❌ 错误：完全缺少percentage
fault:
  delay:
    fixed_delay: "1000ms"
```

**正确格式**:
```yaml
# ✅ 正确：percentage在fault级别
fault:
  percentage: 100
  delay:
    fixed_delay: "1000ms"
```

#### 1.2 缺少fixed_delay字段
```
Error: missing field `fixed_delay` at line 1 column 132
```

**问题分析**:
- 字段名必须是`fixed_delay`，不是`fixedDelayMs`
- 必须是字符串格式，包含时间单位

**错误示例**:
```yaml
# ❌ 错误字段名
delay:
  fixedDelayMs: 1000
  
# ❌ 缺少时间单位
delay:
  fixed_delay: 1000
```

**正确格式**:
```yaml
# ✅ 正确格式
delay:
  fixed_delay: "1000ms"
  # 或者
  fixed_delay: "1s"
  # 或者
  fixed_delay: "2.5s"
```

### 2. 类型错误

#### 2.1 路径类型错误
```
Error: invalid type: string "/", expected struct PathMatcherHelper
```

**问题分析**:
- 路径必须是对象，指定匹配类型
- 不能是简单字符串

**错误示例**:
```yaml
# ❌ 错误：直接使用字符串
match:
  path: "/"
  httpMethod: "GET"
```

**正确格式**:
```yaml
# ✅ 正确：使用对象格式
match:
  path:
    exact: "/"          # 精确匹配
  # 或者
  path:
    prefix: "/api/"     # 前缀匹配
  # 或者  
  path:
    regex: "/api/users/\\d+"  # 正则匹配
```

#### 2.2 Headers类型错误
```
Error: invalid type: map, expected a sequence at line 1 column 505
```

**问题分析**:
- headers必须是数组格式，不是键值对对象
- 每个header需要name字段和匹配条件

**错误示例**:
```yaml
# ❌ 错误：使用对象格式
headers:
  x-user-id:
    exact: "test"
  authorization:
    prefix: "Bearer "
```

**正确格式**:
```yaml
# ✅ 正确：使用数组格式
headers:
  - name: "x-user-id"
    exact: "test"
  - name: "authorization"
    prefix: "Bearer "
```

#### 2.3 HTTP方法字段错误
```
Error: unknown field `httpMethod`, expected `method`
```

**问题分析**:
- 字段名应该是`method`而不是`httpMethod`
- 需要使用StringMatcher格式

**错误示例**:
```yaml
# ❌ 错误字段名
match:
  httpMethod: "GET"
```

**正确格式**:
```yaml
# ✅ 正确格式
match:
  method:
    exact: "GET"
  # 或者简化写法（如果Wasm插件支持）
  method: "GET"
```

### 3. 值相关错误

#### 3.1 无效的HTTP状态码
```
Error: invalid value for httpStatus: 999
```

**问题分析**:
- HTTP状态码必须在有效范围内 (100-599)
- 常用错误状态码：400, 401, 403, 404, 500, 502, 503, 504

**错误示例**:
```yaml
# ❌ 无效状态码
abort:
  httpStatus: 999
```

**正确格式**:
```yaml
# ✅ 有效状态码
abort:
  httpStatus: 503
```

#### 3.2 无效的时间格式
```
Error: invalid duration format: "1000"
```

**问题分析**:
- 时间必须包含单位
- 支持的单位：ms, s, m, h

**错误示例**:
```yaml
# ❌ 缺少单位
delay:
  fixed_delay: "1000"
```

**正确格式**:
```yaml
# ✅ 包含单位
delay:
  fixed_delay: "1000ms"
  # 或者
  fixed_delay: "1s"
  # 或者
  fixed_delay: "1.5s"
```

#### 3.3 无效的百分比
```
Error: percentage must be between 0 and 100
```

**问题分析**:
- 百分比必须在0-100范围内
- 通常使用整数

**错误示例**:
```yaml
# ❌ 超出范围
fault:
  percentage: 150
```

**正确格式**:
```yaml
# ✅ 有效范围
fault:
  percentage: 80    # 80%的概率
```

## 🔧 系统性排查步骤

### 第1步：收集错误信息
```bash
# 获取最近的Wasm日志
docker logs wasm_fault_injection-envoy-1 --tail 50 | grep -E "(Failed to parse|error|warning)"

# 获取配置更新日志
docker logs wasm_fault_injection-envoy-1 --tail 50 | grep "Received config update"
```

### 第2步：验证策略文件格式
```bash
# 使用CLI验证（会进行基础验证）
./hfi-cli policy apply -f your-policy.yaml --dry-run  # 如果支持

# 手动验证YAML格式
python3 -c "import yaml; yaml.safe_load(open('your-policy.yaml'))"
```

### 第3步：逐步测试配置
```bash
# 1. 测试最简单的配置
cat > minimal-test.yaml << EOF
metadata:
  name: "minimal-test"
spec:
  rules:
    - match:
        method:
          exact: "GET"
        path:
          exact: "/test"
      fault:
        percentage: 100
        abort:
          httpStatus: 503
EOF

./hfi-cli policy apply -f minimal-test.yaml
```

### 第4步：检查配置传播
```bash
# 检查etcd存储
docker exec wasm_fault_injection-etcd-1 etcdctl get "hfi/policies/minimal-test"

# 检查Control Plane处理
docker logs wasm_fault_injection-control-plane-1 --tail 20
```

### 第5步：分析Wasm解析结果
```bash
# 查看解析成功的消息
docker logs wasm_fault_injection-envoy-1 --tail 20 | grep "Successfully parsed"

# 查看规则加载情况
docker logs wasm_fault_injection-envoy-1 --tail 20 | grep "Rule [0-9]"
```

## 🏗️ Wasm插件结构理解

### Rust结构体定义
理解错误需要了解Wasm插件的数据结构：

```rust
// config.rs 中的关键结构
pub struct Fault {
    pub abort: Option<AbortAction>,
    pub delay: Option<DelayAction>,
    pub percentage: u32,  // ← 必须在这个级别
}

pub struct DelayAction {
    #[serde(rename = "fixed_delay")]  // ← 字段名映射
    pub fixed_delay: String,
}

pub struct MatchCondition {
    pub path: Option<PathMatcher>,
    pub method: Option<StringMatcher>,
    pub headers: Option<Vec<HeaderMatcher>>,  // ← 数组，不是map
}

pub struct HeaderMatcher {
    pub name: String,        // ← header名称
    pub exact: Option<String>,
    pub prefix: Option<String>,
    pub regex: Option<String>,
}
```

### 字段映射关系
| YAML字段 | Rust字段 | 说明 |
|---------|----------|------|
| `fixed_delay` | `fixed_delay` | 时间字符串 |
| `httpStatus` | `http_status` | HTTP状态码 |
| `httpMethod` | ❌ 错误 | 应该用`method` |
| `headers` | `headers` | 必须是数组 |
| `percentage` | `percentage` | 在`fault`级别 |

## 🎯 Header匹配特殊问题

### 支持的Header列表
Wasm插件只能匹配预定义的headers：

```rust
let common_headers = [
    "host", "user-agent", "accept", "accept-language", "accept-encoding",
    "authorization", "content-type", "content-length", "x-forwarded-for",
    "x-real-ip", "x-user-id", "x-tenant-id", "x-service", "x-version"
];
```

### Header匹配问题排查
```bash
# 1. 使用支持的header测试
curl -H "x-user-id: test" http://localhost:18000/

# 2. 检查header是否被正确提取
docker logs wasm_fault_injection-envoy-1 --tail 50 | grep -E "(header|Header)"

# 3. 验证header名称大小写
# HTTP header通常是case-insensitive，但配置中要保持一致
```

## 🚨 紧急修复手册

### 配置解析完全失败
```bash
# 1. 立即清理有问题的策略
docker exec wasm_fault_injection-etcd-1 etcdctl del --prefix "hfi/policies/"

# 2. 验证服务恢复
curl http://localhost:18000/  # 应该正常响应

# 3. 重新应用已知正确的配置
./hfi-cli policy apply -f working-policy.yaml
```

### Wasm插件停止响应
```bash
# 1. 重启Envoy
docker-compose restart envoy

# 2. 检查插件加载
docker logs wasm_fault_injection-envoy-1 | grep -E "(wasm|plugin)"

# 3. 如果需要，重新构建插件
docker-compose up -d wasm-builder
docker-compose restart envoy
```

### 调试模式启用
```bash
# 增加Envoy日志详细度（在envoy.yaml中）
# 添加：--log-level debug

# 或者实时调整
curl -X POST "http://localhost:19000/logging?level=debug"
```

## 📚 配置模板库

### 模板1: 简单Abort
```yaml
metadata:
  name: "simple-abort"
spec:
  rules:
    - match:
        path:
          exact: "/test"
      fault:
        percentage: 100
        abort:
          httpStatus: 503
```

### 模板2: 条件延迟
```yaml
metadata:
  name: "conditional-delay"
spec:
  rules:
    - match:
        method:
          exact: "POST"
        path:
          prefix: "/api/"
        headers:
          - name: "x-user-id"
            exact: "test-user"
      fault:
        percentage: 50
        delay:
          fixed_delay: "2s"
```

### 模板3: 复杂匹配
```yaml
metadata:
  name: "complex-matching"
spec:
  rules:
    - match:
        method:
          exact: "GET"
        path:
          regex: "/api/users/\\d+"
        headers:
          - name: "authorization"
            prefix: "Bearer "
          - name: "x-version"
            exact: "v1"
      fault:
        percentage: 25
        abort:
          httpStatus: 429
```

---

**记住**: 大多数配置错误都是由于字段名称、类型或结构不匹配造成的。仔细对照Rust结构体定义通常能快速发现问题。
