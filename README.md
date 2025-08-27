# HFI (HTTP Fault Injection)

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](https://github.com/your-org/hfi)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.24+-blue)](https://golang.org)
[![Rust Version](https://img.shields.io/badge/rust-1.89+-orange)](https://rust-lang.org)

**HFI** 是一个基于 Kubernetes 和 Envoy 的云原生故障注入平台，专为混沌工程和弹性测试设计。它通过 WASM 插件在 Envoy 代理中实现细粒度的 HTTP 故障注入。

## ✨ 特性

- 🎯 **精确故障注入**: 支持基于路径、方法、头部的细粒度匹配
- ⚡ **低延迟**: 基于 WASM 的高性能故障注入引擎
- 📊 **实时指标**: 内置故障注入指标和监控
- 🔄 **动态配置**: 运行时热更新故障注入策略
- 🚀 **云原生**: 原生支持 Kubernetes 和 Istio/Envoy
- 🛡️ **生产就绪**: 完整的错误处理和故障恢复机制

## 🏗️ 架构

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│                 │    │                  │    │                 │
│   HFI CLI       │───▶│  Control Plane   │───▶│     etcd        │
│                 │    │                  │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                │ SSE Stream
                                ▼
                       ┌──────────────────┐
                       │                  │
                       │  Envoy + WASM    │
                       │     Plugin       │
                       │                  │
                       └──────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │                  │
                       │  Target Service  │
                       │                  │
                       └──────────────────┘
```

## 🚀 快速开始

### 先决条件

- Kubernetes 集群 (v1.20+)
- kubectl 配置好
- Docker (用于构建镜像)

### 1. 部署 HFI

```bash
# 部署控制平面
kubectl apply -f https://raw.githubusercontent.com/your-org/hfi/main/k8s/control-plane.yaml

# 部署示例应用
kubectl apply -f https://raw.githubusercontent.com/your-org/hfi/main/k8s/sample-app-with-proxy.yaml
```

### 2. 安装 CLI

```bash
# 下载最新版本
curl -LO https://github.com/your-org/hfi/releases/latest/download/hfi-cli-linux-amd64
chmod +x hfi-cli-linux-amd64
sudo mv hfi-cli-linux-amd64 /usr/local/bin/hfi-cli
```

### 3. 创建第一个故障注入

```bash
# 端口转发到控制平面
kubectl port-forward svc/hfi-control-plane 8080:8080 &

# 应用延迟故障
hfi-cli policy apply -f cli/examples/delay-policy.yaml

# 测试故障注入
kubectl port-forward svc/sample-app 8000:80 &
time curl http://localhost:8000/test  # 应该有 1 秒延迟
```

📖 **详细指南**: 查看 [QUICKSTART.md](QUICKSTART.md) 获取完整的入门教程。

## 📁 项目结构

```
.
├── cli/                    # CLI 工具
│   ├── examples/          # 策略示例文件
│   └── ...
├── control-plane/         # 控制平面 (Go)
│   ├── api/              # HTTP API 处理
│   ├── service/          # 业务逻辑
│   └── storage/          # 存储层
├── wasm-plugin/           # WASM 插件 (Rust)
│   └── src/              # 插件源码
├── k8s/                   # Kubernetes 部署文件
├── doc/                   # 设计文档
├── docs/                  # 用户文档
└── QUICKSTART.md          # 快速开始指南
```

## 🎯 故障类型

### 延迟故障
```yaml
fault:
  delay:
    fixed_delay: "1s"
  percentage: 50
```

### 中断故障
```yaml
fault:
  abort:
    httpStatus: 503
  percentage: 20
```

### 条件匹配
```yaml
match:
  method:
    exact: "GET"
  path:
    prefix: "/api"
  headers:
    - name: "x-user-type"
      exact: "premium"
```

## 📊 监控指标

HFI 提供以下内置指标：

- `hfi.faults.aborts_total` - 中断故障计数
- `hfi.faults.delays_total` - 延迟故障计数  
- `hfi.faults.delay_duration_milliseconds` - 延迟时长分布

通过 Envoy admin 接口访问：
```bash
curl http://localhost:19000/stats | grep hfi.faults
```

## 🛠️ 开发

### 构建控制平面

```bash
cd control-plane
go build -o hfi-control-plane .
```

### 构建 WASM 插件

```bash
cd wasm-plugin
cargo build --target wasm32-unknown-unknown --release
```

### 构建 CLI

```bash
cd cli
go build -o hfi-cli main.go
```

### 运行测试

```bash
# 控制平面测试
cd control-plane && go test ./...

# 集成测试 (需要 Docker)
docker-compose up -d
./scripts/integration-test.sh
```

## 📚 文档

- [快速开始](QUICKSTART.md) - 15分钟入门指南
- [架构设计](doc/Design.md) - 系统设计文档
- [API 参考](doc/API_REFERENCE.md) - REST API 文档
- [策略语法](doc/POLICY_SYNTAX.md) - 故障注入策略语法
- [运维指南](doc/OPERATIONS.md) - 生产环境运维
- [故障排除](docs/TROUBLESHOOTING.md) - 常见问题解决

## 🤝 贡献

我们欢迎贡献！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解如何参与项目。

### 开发流程

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 📄 许可证

本项目基于 MIT 许可证开源 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🏷️ 版本历史

- **v0.3.0** - 指标系统和错误处理增强
- **v0.2.0** - Kubernetes 集成和 CLI 工具
- **v0.1.0** - 基础故障注入功能

## 🆘 支持

- 📋 [GitHub Issues](https://github.com/your-org/hfi/issues) - 报告问题和请求功能
- 💬 [Discussions](https://github.com/your-org/hfi/discussions) - 社区讨论
- 📖 [文档](https://hfi.example.com/docs) - 完整文档
- 🎥 [视频教程](https://youtube.com/playlist?list=PLhfi-example) - 视频教程

## 🌟 致谢

感谢以下开源项目：

- [Envoy Proxy](https://envoyproxy.io) - 高性能代理
- [proxy-wasm](https://github.com/proxy-wasm) - WASM 扩展框架
- [etcd](https://etcd.io) - 分布式键值存储
- [Gin](https://gin-gonic.com) - Go HTTP 框架
