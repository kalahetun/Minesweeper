# CLI使用最佳实践指南

## 🎯 概述

本指南提供`hfi-cli`命令行工具的最佳使用实践，帮助您高效管理故障注入策略。

## 📋 基础使用模式

### 日常工作流程

#### 1. 健康检查
```bash
# 启动时首先检查系统状态
./hfi-cli version
./hfi-cli status  # 如果有此命令

# 或者使用全局标志检查连接
./hfi-cli policy list --verbose
```

#### 2. 策略开发周期
```bash
# 开发 → 验证 → 应用 → 测试 → 清理
./hfi-cli policy apply -f dev-policy.yaml
curl http://localhost:18000/test  # 测试效果
./hfi-cli policy list             # 确认策略
./hfi-cli policy delete dev-policy
```

### 建议的项目结构
```
your-project/
├── policies/
│   ├── production/
│   │   ├── network-latency.yaml
│   │   └── service-unavailable.yaml
│   ├── staging/
│   │   ├── chaos-monkey.yaml
│   │   └── load-test.yaml
│   └── development/
│       ├── debug-delay.yaml
│       └── test-abort.yaml
├── scripts/
│   ├── apply-prod-policies.sh
│   ├── cleanup-test-policies.sh
│   └── validate-policies.sh
└── docs/
    ├── policy-catalog.md
    └── runbooks/
```

## 🏗️ 策略编写最佳实践

### 命名约定
```yaml
# ✅ 好的命名：描述性且结构化
metadata:
  name: "api-gateway-latency-50p"     # 服务-故障类型-参数
  name: "payment-service-503-error"   # 服务-状态码
  name: "auth-timeout-2s-debug"       # 服务-延迟-用途

# ❌ 避免的命名：模糊或随意
metadata:
  name: "test"
  name: "policy1"
  name: "temp-debug-thing"
```

### 渐进式故障注入
```bash
# 第1步：低概率测试
cat > low-risk-test.yaml << EOF
metadata:
  name: "payment-503-test-5p"
spec:
  rules:
    - match:
        path:
          prefix: "/api/payment"
      fault:
        percentage: 5        # 开始时使用低概率
        abort:
          httpStatus: 503
EOF

./hfi-cli policy apply -f low-risk-test.yaml

# 第2步：监控影响
# 查看日志、监控指标、用户反馈

# 第3步：逐步增加强度
# 修改percentage: 5 → 10 → 25 → 50
```

### 环境隔离
```bash
# 使用不同的API端点
export HFI_ENDPOINT="http://dev-control-plane:8080"
./hfi-cli policy apply -f dev-policy.yaml

export HFI_ENDPOINT="http://staging-control-plane:8080"
./hfi-cli policy apply -f staging-policy.yaml

# 或在策略名称中包含环境信息
metadata:
  name: "dev-auth-latency-1s"
  name: "staging-payment-503-error"
```

## 🔧 常用命令组合

### 故障注入测试套件
```bash
#!/bin/bash
# test-fault-injection.sh

set -e

CLI="./hfi-cli"
BASE_URL="http://localhost:18000"

echo "🧪 开始故障注入测试套件"

# 1. 清理环境
echo "清理现有策略..."
$CLI policy list --output json | jq -r '.[].metadata.name' | xargs -I {} $CLI policy delete {}

# 2. 测试网络延迟
echo "测试网络延迟 (2秒)..."
cat > /tmp/delay-test.yaml << EOF
metadata:
  name: "test-delay-2s"
spec:
  rules:
    - match:
        path:
          exact: "/api/slow"
      fault:
        percentage: 100
        delay:
          fixed_delay: "2s"
EOF

$CLI policy apply -f /tmp/delay-test.yaml
echo "策略已应用，测试延迟效果..."
time curl -s "$BASE_URL/api/slow" > /dev/null
$CLI policy delete test-delay-2s

# 3. 测试服务不可用
echo "测试服务不可用 (503错误)..."
cat > /tmp/abort-test.yaml << EOF
metadata:
  name: "test-abort-503"
spec:
  rules:
    - match:
        path:
          exact: "/api/fail"
      fault:
        percentage: 100
        abort:
          httpStatus: 503
EOF

$CLI policy apply -f /tmp/abort-test.yaml
response=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/fail")
if [ "$response" = "503" ]; then
    echo "✅ 503错误测试成功"
else
    echo "❌ 503错误测试失败，收到状态码: $response"
fi
$CLI policy delete test-abort-503

# 4. 清理
rm -f /tmp/delay-test.yaml /tmp/abort-test.yaml
echo "🎉 测试套件完成"
```

### 策略验证脚本
```bash
#!/bin/bash
# validate-policy.sh

POLICY_FILE="$1"

if [ -z "$POLICY_FILE" ]; then
    echo "用法: $0 <policy-file.yaml>"
    exit 1
fi

echo "🔍 验证策略文件: $POLICY_FILE"

# 1. YAML语法检查
echo "检查YAML语法..."
python3 -c "import yaml; yaml.safe_load(open('$POLICY_FILE'))" || {
    echo "❌ YAML语法错误"
    exit 1
}

# 2. 必需字段检查
echo "检查必需字段..."
if ! grep -q "metadata:" "$POLICY_FILE"; then
    echo "❌ 缺少metadata字段"
    exit 1
fi

if ! grep -q "name:" "$POLICY_FILE"; then
    echo "❌ 缺少name字段"
    exit 1
fi

if ! grep -q "spec:" "$POLICY_FILE"; then
    echo "❌ 缺少spec字段"
    exit 1
fi

# 3. 常见错误检查
echo "检查常见配置错误..."

# 检查percentage位置
if grep -A5 "delay:" "$POLICY_FILE" | grep -q "percentage:"; then
    echo "⚠️  警告: percentage应该在fault级别，不是在delay内部"
fi

if grep -A5 "abort:" "$POLICY_FILE" | grep -q "percentage:"; then
    echo "⚠️  警告: percentage应该在fault级别，不是在abort内部"
fi

# 检查字段名称
if grep -q "fixedDelayMs:" "$POLICY_FILE"; then
    echo "❌ 错误: 应该使用'fixed_delay'而不是'fixedDelayMs'"
    exit 1
fi

if grep -q "httpMethod:" "$POLICY_FILE"; then
    echo "❌ 错误: 应该使用'method'而不是'httpMethod'"
    exit 1
fi

# 4. 模拟应用（如果支持dry-run）
echo "模拟应用策略..."
# ./hfi-cli policy apply -f "$POLICY_FILE" --dry-run

echo "✅ 策略文件验证通过"
```

### 批量操作
```bash
# 批量应用目录中的所有策略
find policies/production/ -name "*.yaml" -exec ./hfi-cli policy apply -f {} \;

# 批量删除匹配模式的策略
./hfi-cli policy list --output json | \
    jq -r '.[] | select(.metadata.name | startswith("test-")) | .metadata.name' | \
    xargs -I {} ./hfi-cli policy delete {}

# 策略备份
./hfi-cli policy list --output json > policies-backup-$(date +%Y%m%d).json
```

## 📊 监控和观察

### 故障注入效果验证
```bash
# 1. 基本功能测试
echo "测试正常路径（应该不受影响）:"
curl -w "响应时间: %{time_total}s, 状态码: %{http_code}\n" http://localhost:18000/health

echo "测试故障路径（应该受到影响）:"
curl -w "响应时间: %{time_total}s, 状态码: %{http_code}\n" http://localhost:18000/api/target

# 2. 概率验证（多次请求）
echo "测试50%概率故障注入:"
for i in {1..10}; do
    status=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:18000/api/random)
    echo "请求 $i: $status"
done

# 3. 延迟测试
echo "测试延迟注入:"
for i in {1..5}; do
    time curl -s http://localhost:18000/api/slow > /dev/null
done
```

### 系统状态检查
```bash
# 检查所有组件状态
echo "=== 控制平面状态 ==="
curl -s http://localhost:8080/health || echo "控制平面不可达"

echo "=== Envoy代理状态 ==="
curl -s http://localhost:19000/ready || echo "Envoy不可达"

echo "=== 当前策略列表 ==="
./hfi-cli policy list

echo "=== 最近的Envoy日志 ==="
docker logs wasm_fault_injection-envoy-1 --tail 10 --since 5m
```

## 🎛️ 高级使用技巧

### 条件故障注入
```yaml
# 基于用户ID的故障注入
metadata:
  name: "user-specific-failure"
spec:
  rules:
    - match:
        headers:
          - name: "x-user-id"
            exact: "test-user-123"
        path:
          prefix: "/api/"
      fault:
        percentage: 100
        abort:
          httpStatus: 500

# 基于API版本的故障注入
metadata:
  name: "v1-api-deprecation"
spec:
  rules:
    - match:
        headers:
          - name: "x-version"
            exact: "v1"
      fault:
        percentage: 25
        delay:
          fixed_delay: "3s"
```

### 多规则策略
```yaml
metadata:
  name: "comprehensive-chaos"
spec:
  rules:
    # 规则1: 慢查询模拟
    - match:
        method:
          exact: "GET"
        path:
          prefix: "/api/search"
      fault:
        percentage: 30
        delay:
          fixed_delay: "5s"
    
    # 规则2: 写操作失败
    - match:
        method:
          exact: "POST"
        path:
          prefix: "/api/orders"
      fault:
        percentage: 10
        abort:
          httpStatus: 503
    
    # 规则3: 认证服务不稳定
    - match:
        path:
          exact: "/api/auth/login"
      fault:
        percentage: 15
        abort:
          httpStatus: 429
```

### 临时快速测试
```bash
# 创建临时策略文件的函数
create_temp_policy() {
    local name="$1"
    local path="$2"
    local fault_type="$3"
    local percentage="${4:-100}"
    
    cat > "/tmp/${name}.yaml" << EOF
metadata:
  name: "$name"
spec:
  rules:
    - match:
        path:
          exact: "$path"
      fault:
        percentage: $percentage
EOF

    case "$fault_type" in
        "delay")
            cat >> "/tmp/${name}.yaml" << EOF
        delay:
          fixed_delay: "2s"
EOF
            ;;
        "abort")
            cat >> "/tmp/${name}.yaml" << EOF
        abort:
          httpStatus: 503
EOF
            ;;
    esac
    
    echo "/tmp/${name}.yaml"
}

# 使用示例
policy_file=$(create_temp_policy "quick-test" "/test" "delay" 50)
./hfi-cli policy apply -f "$policy_file"
curl http://localhost:18000/test
./hfi-cli policy delete quick-test
rm "$policy_file"
```

## 🚨 故障排除清单

### CLI连接问题
```bash
# 检查网络连接
curl -v http://localhost:8080/health

# 检查端口占用
netstat -tlnp | grep 8080

# 检查控制平面服务
docker ps | grep control-plane
docker logs wasm_fault_injection-control-plane-1 --tail 20
```

### 策略不生效
```bash
# 1. 确认策略已应用
./hfi-cli policy list

# 2. 检查etcd存储
docker exec wasm_fault_injection-etcd-1 etcdctl get --prefix "hfi/policies/"

# 3. 检查Wasm插件日志
docker logs wasm_fault_injection-envoy-1 --tail 50 | grep -E "(wasm|Failed|Successfully)"

# 4. 验证请求路径匹配
curl -v http://localhost:18000/your-test-path
```

### 性能问题
```bash
# 监控资源使用
docker stats wasm_fault_injection-envoy-1

# 检查并发连接数
curl http://localhost:19000/stats | grep "^cluster.*cx_"

# 查看处理时间统计
curl http://localhost:19000/stats | grep "^http.*duration"
```

---

**记住**: 故障注入是强大的测试工具，但也有风险。始终在非生产环境中充分测试，并在生产环境中谨慎使用。
