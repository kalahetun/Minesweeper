# Quick Start: CLI Examples & Validation Scripts

**Feature**: 009-cli-examples-update  
**Date**: 2025-12-10

## Prerequisites

确保以下环境已就绪：

```bash
# 1. kubectl 可用
kubectl version --client

# 2. k3s/k8s 集群运行中
kubectl get nodes

# 3. Istio 已安装
kubectl get pods -n istio-system

# 4. BOIFI 组件已部署
kubectl get pods -n boifi           # Control Plane
kubectl get wasmplugin -n demo      # WasmPlugin

# 5. 测试服务已部署
kubectl get pods -n demo
```

---

## Quick Start: 基础故障注入

### Step 1: 查看示例策略

```bash
cd executor/cli/examples

# 查看 abort 策略示例
cat basic/abort-policy.yaml
```

```yaml
# 示例内容预览
metadata:
  name: "abort-policy"
spec:
  selector:
    service: frontend      # 仅影响 frontend 服务
    namespace: demo
  rules:
    - match:
        method:
          exact: "GET"
        path:
          prefix: "/"
      fault:
        percentage: 100
        abort:
          httpStatus: 503
```

### Step 2: 应用策略

```bash
# 使用 hfi-cli 应用策略
./hfi-cli policy apply -f basic/abort-policy.yaml

# 或使用 curl 直接调用 API
kubectl port-forward svc/hfi-control-plane 8080:8080 -n boifi &
curl -X POST http://localhost:8080/v1/policies \
  -H "Content-Type: application/json" \
  -d @basic/abort-policy.yaml
```

### Step 3: 验证故障注入

```bash
# 等待策略传播（约 35 秒）
sleep 35

# 获取测试 Pod
POD=$(kubectl get pod -n demo -l app=frontend -o jsonpath='{.items[0].metadata.name}')

# 发送测试请求
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s -o /dev/null -w "%{http_code}\n" http://frontend.demo.svc.cluster.local/

# 预期输出: 503
```

### Step 4: 清理策略

```bash
./hfi-cli policy delete abort-policy

# 或
curl -X DELETE http://localhost:8080/v1/policies/abort-policy
```

---

## Quick Start: 运行验证脚本

### 基础验证

```bash
cd executor/cli/examples/scripts

# 运行基础故障注入验证
./validate-basic.sh

# 可选：指定命名空间
NAMESPACE=my-app CONTROL_PLANE_NS=boifi ./validate-basic.sh
```

### 服务选择器验证

```bash
# 确保有两个服务可用
kubectl get svc -n demo

# 运行选择器验证
./validate-selector.sh

# 可选：指定服务
SERVICE_A=frontend SERVICE_B=productcatalog ./validate-selector.sh
```

---

## Quick Start: 微服务场景示例

### Online Boutique 场景

```bash
cd executor/cli/examples/scenarios/online-boutique

# 查看场景说明
cat ../README.md

# 应用 frontend 故障
./hfi-cli policy apply -f frontend-abort.yaml

# 应用 checkout 延迟
./hfi-cli policy apply -f checkout-delay.yaml

# 验证效果
# 1. 访问 frontend → 503 错误
# 2. 结账流程 → 明显变慢
```

---

## 目录结构速览

```
examples/
├── README.md              ← 完整文档
├── basic/                 ← 从这里开始
│   ├── abort-policy.yaml
│   ├── delay-policy.yaml
│   └── percentage-policy.yaml
├── advanced/              ← 进阶学习
│   ├── header-policy.yaml
│   ├── time-limited-policy.yaml
│   └── service-targeted-policy.yaml
├── scenarios/             ← 真实场景
│   └── online-boutique/
└── scripts/               ← 自动化验证
    ├── validate-basic.sh
    └── validate-selector.sh
```

---

## 常见问题

### Q: 策略应用后没有效果？

```bash
# 1. 检查策略是否被接受
./hfi-cli policy list

# 2. 检查 WasmPlugin 是否加载
kubectl logs -n demo $POD -c istio-proxy | grep wasm

# 3. 检查 selector 是否匹配
# 策略的 selector.service 应与 Pod 的 app 标签匹配
kubectl get pod $POD -n demo --show-labels
```

### Q: 验证脚本超时？

```bash
# 增加等待时间（默认 35 秒）
PROPAGATION_WAIT=60 ./validate-basic.sh
```

### Q: 如何只测试 delay 不测试 abort？

```bash
# 直接使用单独的策略文件
./hfi-cli policy apply -f basic/delay-policy.yaml

# 手动验证
time kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s http://frontend.demo.svc.cluster.local/
```

---

## 下一步

1. 阅读 [README.md](../README.md) 了解完整策略格式
2. 查看 [advanced/](../advanced/) 目录了解高级用法
3. 运行 [scenarios/](../scenarios/) 中的真实场景测试
4. 查看 [../../k8s/tests/](../../k8s/tests/) 了解更多 E2E 测试

---

## 验证清单

在完成 Quick Start 后，确认以下项目：

- [ ] 能够查看和理解策略文件格式
- [ ] 能够使用 hfi-cli 或 curl 应用策略
- [ ] 能够验证 abort 故障（返回 503）
- [ ] 能够验证 delay 故障（响应变慢）
- [ ] 能够运行 validate-basic.sh 并看到 PASS
- [ ] 能够清理测试策略
