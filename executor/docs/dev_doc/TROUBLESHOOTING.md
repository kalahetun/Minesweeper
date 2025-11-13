# 故障排查指南 (TROUBLESHOOTING.md)

**版本**: v1.0  
**最后更新**: 2024-11-13

---

## 目录

1. [常见问题](#常见问题)
2. [调试技巧](#调试技巧)
3. [监控指标](#监控指标)
4. [日志分析](#日志分析)

---

## 常见问题

### 问题 1: 策略应用成功但不生效

**症状**:
```bash
$ hfi-cli policy apply -f my-policy.yaml
faultinjectionpolicy.hfi.dev "my-policy" applied

# 但实际请求没有被注入故障
$ curl http://service:8080/api/v1/test
# 返回 200，没有延迟或 abort
```

**常见原因**:

| 原因 | 检查方法 | 解决方案 |
|------|---------|---------|
| WASM 插件未连接 | `kubectl logs hfi-control-plane \| grep connected` | 重启 Envoy |
| Match 条件不匹配 | `hfi-cli policy describe my-policy` | 检查路径、方法等 |
| 时间窗口已过期 | `hfi-cli policy get my-policy -o json` | 重新应用策略 |
| 百分比为 0% | `hfi-cli policy get my-policy -o json \| grep percentage` | 增加百分比 |
| 缓存未更新 | 等待 5-10 秒 | 或重启 WASM 插件 |

**诊断步骤**:

```bash
# 1. 验证策略格式
hfi-cli policy describe my-policy
# 查看 Match Conditions 和 Fault Actions 是否正确

# 2. 查看 WASM 插件日志
kubectl logs <envoy-pod> -c envoy | grep -i "my-policy"

# 3. 检查 Control Plane 连接状态
kubectl logs -n hfi-system -l app=hfi-control-plane | tail -20

# 4. 查看插件版本
kubectl logs <envoy-pod> -c envoy | grep "Plugin version"

# 5. 测试请求匹配
curl -v http://service:8080/api/v1/test 2>&1 | grep -i "X-"

# 6. 生成诊断日志
LOG_LEVEL=debug kubectl rollout restart deploy/hfi-control-plane -n hfi-system
```

**完整恢复方案**:

```bash
#!/bin/bash
# troubleshoot-policy.sh

POLICY_NAME=$1

if [ -z "$POLICY_NAME" ]; then
  echo "Usage: $0 <policy-name>"
  exit 1
fi

echo "=== 1. 检查策略存在性 ==="
hfi-cli policy get "$POLICY_NAME" && echo "✅ 策略存在" || echo "❌ 策略不存在"

echo -e "\n=== 2. 检查策略详情 ==="
hfi-cli policy describe "$POLICY_NAME"

echo -e "\n=== 3. 检查 Control Plane 状态 ==="
curl -s http://localhost:8080/v1/health | jq . && echo "✅ Control Plane 正常" || echo "❌ Control Plane 异常"

echo -e "\n=== 4. 检查 WASM 插件连接 ==="
kubectl logs -n hfi-system -l app=hfi-control-plane | grep "SSE client connected" | tail -1

echo -e "\n=== 5. 重启 Envoy (如需要) ==="
read -p "重启 Envoy? (y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  kubectl rollout restart deployment/envoy
  kubectl wait --for=condition=Ready pod -l app=envoy --timeout=60s
  echo "✅ Envoy 已重启"
fi

echo -e "\n=== 6. 测试故障注入 ==="
for i in {1..5}; do
  echo "请求 $i:"
  time curl -s http://service:8080/api/v1/test > /dev/null
done
```

---

### 问题 2: Control Plane 响应缓慢

**症状**:
```bash
$ time curl http://localhost:8080/v1/policies
real    0m2.345s

# 原本应该 < 100ms
```

**常见原因**:

| 原因 | 症状 | 解决方案 |
|------|------|---------|
| 规则数过多 | 列表返回 > 10000 条 | 删除不需要的规则 |
| 复杂的正则表达式 | CPU 占用率高 | 使用精确匹配 |
| etcd 性能不足 | etcd 响应缓慢 | 升级 etcd 或切换内存存储 |
| 网络延迟 | ping 延迟高 | 检查网络连接 |
| 资源不足 | 内存/CPU 充满 | 增加副本或资源限制 |

**诊断和优化**:

```bash
# 1. 检查规则数量
echo "总规则数:"
hfi-cli policy list | tail -1 | awk '{print $NF}'

# 2. 分析每个策略的规则数
echo "规则数分布:"
hfi-cli policy get -o json | jq '.[] | {name: .metadata.name, rules: (.spec.rules | length)}'

# 3. 查看 CPU 和内存使用
kubectl top pod -n hfi-system -l app=hfi-control-plane

# 4. 测试单个策略获取速度
time hfi-cli policy get largest-policy

# 5. etcd 性能检查 (如使用 etcd)
kubectl exec -it <etcd-pod> -- etcdctl endpoint health
kubectl exec -it <etcd-pod> -- etcdctl check perf

# 6. 优化方案
# 方案 A: 删除不需要的旧策略
hfi-cli policy delete old-policy-1
hfi-cli policy delete old-policy-2

# 方案 B: 优化正则表达式
# 修改策略，用精确匹配替代正则:
# "path": {"regex": "^/api/v1/.*"} 
# 改为:
# "path": {"prefix": "/api/v1/"}

# 方案 C: 增加 Control Plane 副本
kubectl scale deployment hfi-control-plane --replicas=3 -n hfi-system

# 方案 D: 启用内存缓存 (WASM 插件)
# 在 Envoy 配置中增加 cache_size 参数
```

---

### 问题 3: WASM 插件频繁断连

**症状**:
```
Envoy 日志中频繁出现:
[2024-11-13 10:15:23] Error: SSE connection closed
[2024-11-13 10:15:28] Reconnecting to Control Plane...
[2024-11-13 10:15:33] SSE connected
[2024-11-13 10:15:45] Error: SSE connection closed
```

**常见原因**:

| 原因 | 检查方法 | 解决方案 |
|------|---------|---------|
| 网络不稳定 | `ping control-plane` | 检查网络路由 |
| Control Plane 资源不足 | `kubectl top pod` | 增加资源限制 |
| 防火墙阻止 | `nmap control-plane` | 配置防火墙规则 |
| SSE 超时设置过短 | 检查 Envoy 配置 | 增加 timeout 值 |
| WASM 插件版本不匹配 | 检查日志中的版本号 | 重新构建 WASM |

**诊断和修复**:

```bash
# 1. 检查网络连通性
kubectl exec <envoy-pod> -- ping control-plane

# 2. 查看 DNS 解析
kubectl exec <envoy-pod> -- nslookup hfi-control-plane.hfi-system.svc.cluster.local

# 3. 检查防火墙规则
kubectl get networkpolicy -A

# 4. 增加日志级别
kubectl set env deployment/hfi-control-plane LOG_LEVEL=debug -n hfi-system

# 5. 查看最近的连接断开原因
kubectl logs -n hfi-system -l app=hfi-control-plane | grep -i "disconnect\|closed\|error" | tail -10

# 6. 增加 SSE 超时和重连间隔
# 修改 Envoy 配置:
configuration:
  value: |
    {
      "control_plane_addr": "http://control-plane:8080",
      "retry_interval_sec": 10,  # 增加到 10 秒
      "connection_timeout_sec": 30
    }

# 7. 重启 Envoy
kubectl rollout restart deployment/envoy
```

---

### 问题 4: 内存占用不断增长

**症状**:
```bash
$ kubectl top pod -n hfi-system -l app=hfi-control-plane
NAME                               CPU(m)  MEMORY(Mi)
hfi-control-plane-5d4c8f...       50m     256Mi        # 原始
hfi-control-plane-5d4c8f...       52m     512Mi        # 30 分钟后
hfi-control-plane-5d4c8f...       55m     1024Mi       # 1 小时后
```

**常见原因**:

| 原因 | 检查方法 | 解决方案 |
|------|---------|---------|
| 内存泄漏 | 查看日志中的 WARN/ERROR | 升级到最新版本 |
| 缓存未清理 | 检查配置中的 cache_ttl | 设置合理的 TTL |
| 事件流堆积 | 检查 SSE 客户端数 | 减少连接或增加副本 |
| 策略数过多 | `hfi-cli policy list \| wc -l` | 清理旧策略 |

**诊断和修复**:

```bash
# 1. 启用内存分析 (Golang pprof)
kubectl port-forward -n hfi-system <pod-name> 6060:6060

# 在另一个终端访问:
go tool pprof http://localhost:6060/debug/pprof/heap

# 2. 查看 goroutine 数量 (可能泄漏)
curl http://localhost:6060/debug/pprof/goroutine?debug=1 | grep "goroutine profile"

# 3. 清理旧策略
# 列出所有策略及其创建时间
hfi-cli policy get -o json | jq '.[] | {name: .metadata.name, createdAt: .metadata.createdAt}'

# 删除 7 天前的策略
hfi-cli policy list | awk '{print $1}' | while read policy; do
  age=$(( $(date +%s) - $(stat -c %Y <(hfi-cli policy get "$policy" -o json)) ))
  if [ $age -gt 604800 ]; then  # 7 天
    echo "删除旧策略: $policy"
    hfi-cli policy delete "$policy"
  fi
done

# 4. 配置内存限制和 GC
kubectl set resources deployment hfi-control-plane \
  --limits=memory=512Mi,cpu=500m \
  --requests=memory=256Mi,cpu=250m \
  -n hfi-system

# 5. 启用垃圾回收 (如需要)
kubectl set env deployment/hfi-control-plane \
  GOGC=80 \
  -n hfi-system
```

---

## 调试技巧

### 1. 启用详细日志

#### Control Plane

```bash
# 方式 1: 环境变量
kubectl set env deployment/hfi-control-plane \
  LOG_LEVEL=debug \
  LOG_FORMAT=json \
  -n hfi-system

# 方式 2: 查看详细日志
kubectl logs -n hfi-system -l app=hfi-control-plane -f --tail=100

# 方式 3: 结构化日志查询
kubectl logs -n hfi-system -l app=hfi-control-plane -f | jq '.level, .msg, .error'
```

#### WASM 插件

```bash
# Envoy 日志
kubectl logs <envoy-pod> -c envoy -f | grep -E "fault|wasm"

# 增加 Envoy 日志级别
kubectl set env deployment/envoy \
  --limit-overrides=LOGLEVEL=debug \
  -n default
```

### 2. 手动测试

#### 测试策略应用

```bash
# 创建测试策略
cat > test-policy.yaml << 'EOF'
metadata:
  name: debug-policy
spec:
  rules:
    - match:
        path:
          prefix: /debug
      fault:
        percentage: 100  # 100% 注入，确保生效
        delay:
          fixedDelay: "500ms"  # 明显的延迟
EOF

# 应用
hfi-cli policy apply -f test-policy.yaml

# 测试
time curl http://service:8080/debug
# 应该看到 ~500ms 延迟

# 清理
hfi-cli policy delete debug-policy
```

#### 生成高负载测试

```bash
# 使用 Apache Bench
ab -n 1000 -c 10 http://service:8080/api/v1/test

# 或使用 hey (更详细的输出)
go install github.com/rakyll/hey@latest
hey -n 10000 -c 50 http://service:8080/api/v1/test

# 或使用 wrk (高性能)
wrk -t 4 -c 100 -d 30s http://service:8080/api/v1/test
```

### 3. 检查点列表

```bash
# 部署检查清单
□ Control Plane Pod 运行中
  kubectl get pods -n hfi-system

□ WASM 插件 Pod 运行中
  kubectl get pods -l app=envoy

□ Control Plane 可访问
  curl http://localhost:8080/v1/health

□ WASM 插件已连接
  kubectl logs -n hfi-system -l app=hfi-control-plane | grep "connected"

□ 策略已创建
  hfi-cli policy list

□ 策略匹配条件正确
  hfi-cli policy describe <policy-name>

□ 网络连接正常
  kubectl exec <pod> -- ping control-plane

□ 无 OOM 错误
  kubectl describe pod <pod> | grep -i oom

□ 日志无 ERROR
  kubectl logs -n hfi-system -l app=hfi-control-plane | grep ERROR
```

---

## 监控指标

### 关键指标

```
系统级别:
├── control_plane_up (1=运行, 0=停止)
├── control_plane_request_latency_ms (P50, P95, P99)
├── control_plane_request_error_rate (%)
│
WASM 插件:
├── wasm_plugin_connected (1=连接, 0=断开)
├── wasm_plugin_connection_failures_total (计数)
├── wasm_plugin_policy_cache_size (策略数)
├── wasm_plugin_cache_hit_rate (%)
│
策略:
├── policy_count_total (总数)
├── policy_rules_total (总规则数)
├── policy_update_latency_ms (更新延迟)
│
故障注入:
├── rules_matched_total (匹配的规则)
├── faults_injected_total (注入的故障)
├── fault_injection_latency_ms (故障注入延迟)
├── fault_injection_error_rate (%)
│
系统资源:
├── control_plane_cpu_usage (%)
├── control_plane_memory_usage (MB)
├── wasm_plugin_memory_usage (MB)
└── goroutine_count (Golang 线程数)
```

### Prometheus 查询示例

```promql
# Control Plane 响应延迟 P99
histogram_quantile(0.99, rate(control_plane_request_duration_seconds_bucket[5m]))

# 故障注入成功率
rate(faults_injected_total[5m]) / rate(rules_matched_total[5m])

# WASM 插件连接状态
wasm_plugin_connected

# 内存使用趋势
rate(process_resident_memory_bytes[5m])

# 错误率
rate(control_plane_errors_total[5m])
```

### 告警规则

```yaml
groups:
- name: hfi_alerts
  rules:
  - alert: ControlPlaneDown
    expr: control_plane_up == 0
    for: 1m
    annotations:
      summary: "Control Plane 异常"
      
  - alert: HighFaultInjectionLatency
    expr: histogram_quantile(0.99, fault_injection_latency_ms) > 10
    for: 5m
    annotations:
      summary: "故障注入延迟过高 (P99 > 10ms)"
      
  - alert: WasmPluginDisconnected
    expr: wasm_plugin_connected == 0
    for: 2m
    annotations:
      summary: "WASM 插件断开连接"
      
  - alert: HighMemoryUsage
    expr: process_resident_memory_bytes > 512 * 1024 * 1024
    for: 5m
    annotations:
      summary: "内存占用过高"
```

---

## 日志分析

### 日志级别

```
DEBUG   - 最详细，包括所有操作步骤、变量值等
INFO    - 标准信息，记录重要事件
WARN    - 警告，表示可能的问题
ERROR   - 错误，需要立即关注
FATAL   - 致命错误，导致程序退出
```

### 常见日志模式

#### 正常启动

```
[INFO] Starting HFI Control Plane v1.0
[INFO] Listening on 0.0.0.0:8080
[INFO] Storage backend: memory
[INFO] Ready to accept connections
```

#### 正常连接

```
[INFO] New SSE client connected: 192.168.1.10:54321
[INFO] Broadcasting 5 policies to client
[INFO] Client disconnected after 3600s
```

#### 错误情况

```
[ERROR] Failed to apply policy: validation failed - percentage must be 0-100
[ERROR] SSE client closed unexpectedly: connection reset by peer
[ERROR] Storage backend unavailable: etcd connection timeout
```

### 日志搜索模式

```bash
# 查找连接日志
kubectl logs -n hfi-system -l app=hfi-control-plane | grep -E "connected|disconnected"

# 查找错误
kubectl logs -n hfi-system -l app=hfi-control-plane | grep -i error

# 查找特定策略的日志
kubectl logs -n hfi-system -l app=hfi-control-plane | grep "my-policy"

# 统计错误类型
kubectl logs -n hfi-system -l app=hfi-control-plane | grep ERROR | awk -F: '{print $NF}' | sort | uniq -c

# 查看时间段内的日志
kubectl logs -n hfi-system -l app=hfi-control-plane --since=1h --until=30m

# 实时日志流 + 搜索
kubectl logs -n hfi-system -l app=hfi-control-plane -f | grep -E "ERROR|WARN"
```

---

## 获取帮助

### 收集诊断信息

```bash
#!/bin/bash
# collect-diagnostics.sh

echo "=== HFI 诊断信息收集 ==="

echo "1. 版本信息"
hfi-cli version

echo -e "\n2. Control Plane 状态"
kubectl get pods -n hfi-system -l app=hfi-control-plane -o wide

echo -e "\n3. Control Plane 日志 (最后 50 行)"
kubectl logs -n hfi-system -l app=hfi-control-plane --tail=50

echo -e "\n4. 策略列表"
hfi-cli policy list

echo -e "\n5. 系统资源使用"
kubectl top nodes
kubectl top pods -n hfi-system

echo -e "\n6. 网络测试"
kubectl run -it --rm debug --image=alpine --restart=Never -- sh -c 'ping -c 3 hfi-control-plane.hfi-system'

echo -e "\n=== 诊断完成 ==="
```

### 提交 Issue 时包含的信息

```
- HFI 版本: `hfi-cli version`
- Kubernetes 版本: `kubectl version`
- 错误日志: Control Plane + WASM 插件日志 (附件)
- 策略定义: `hfi-cli policy get <policy-name> -o yaml`
- 环境信息: 诊断脚本输出 (附件)
- 重现步骤: 详细的步骤说明
- 预期行为: 应该发生什么
- 实际行为: 实际发生了什么
```

---

**文档版本**: v1.0  
**最后更新**: 2024-11-13  
**维护者**: HFI 团队
