# HFI 本地开发环境搭建指南

本文档详细介绍如何搭建 HFI (HTTP Fault Injection) 项目的本地开发环境，包括工具安装、代码构建、本地运行和测试。

## 📋 先决条件

### 核心开发工具

| 工具 | 推荐版本 | 用途 | 安装说明 |
|--|-|--|-|
| Go | 1.24+ | Control Plane & CLI 开发 | [官方安装指南](https://golang.org/doc/install) |
| Rust | 1.89+ | WASM 插件开发 | [官方安装指南](https://rustup.rs/) |
| Docker | 20.10+ | 容器化构建和运行 | [官方安装指南](https://docs.docker.com/get-docker/) |
| Docker Compose | 2.0+ | 本地多服务编排 | 通常随 Docker 一起安装 |

### Rust 特殊配置

安装 Rust 后，需要添加 WASM 编译目标：

```bash
# 添加 WASM 编译目标
rustup target add wasm32-unknown-unknown

# 安装 WASM 优化工具 (可选，用于生产构建)
cargo install wasm-opt --locked
```

### Kubernetes 工具 (可选)

| 工具 | 推荐版本 | 用途 | 安装说明 |
|--|-|--|-|
| kubectl | 1.28+ | Kubernetes 集群管理 | [官方安装指南](https://kubernetes.io/docs/tasks/tools/) |
| kind | 0.20+ | 本地 Kubernetes 集群 | [官方安装指南](https://kind.sigs.k8s.io/docs/user/quick-start/) |
| minikube | 1.31+ | 本地 Kubernetes 集群 | [官方安装指南](https://minikube.sigs.k8s.io/docs/start/) |

### 验证安装

```bash
# 验证 Go 环境
go version
# 期望输出: go version go1.24.x ...

# 验证 Rust 环境
rustc --version
cargo --version
rustup target list --installed | grep wasm32
# 期望输出: wasm32-unknown-unknown (installed)

# 验证 Docker 环境
docker --version
docker-compose --version
# 或者对于新版本
docker compose version

# 验证 Kubernetes 工具 (可选)
kubectl version --client
kind version
```

## � 快速开始

### 30 秒快速验证

使用项目提供的 Makefile 可以快速验证开发环境：

```bash
# 1. 检查开发环境
make setup

# 2. 构建所有组件
make build-all

# 3. 运行测试
make test

# 4. 启动本地环境
make run-local

# 5. 验证系统运行
curl http://localhost:8080/v1/health

# 6. 停止环境
make stop-local
```

### 一键完整验证

```bash
# 一键完成：清理 + 构建 + 测试
make verify
```

这个命令会：
1. 清理之前的构建产物
2. 构建所有组件
3. 运行完整的测试套件
4. 输出验证结果

## �📁 代码获取与项目结构

### 克隆代码

```bash
# 克隆项目代码
git clone https://github.com/your-org/hfi.git
cd hfi

# 查看项目结构
tree -L 2
```

### 项目目录结构

```
hfi/
├── cli/                    # CLI 工具源码
│   ├── cmd/               # Cobra 命令实现
│   ├── client/            # API 客户端
│   ├── types/             # 类型定义
│   ├── examples/          # 策略示例文件
│   └── main.go           # CLI 入口
├── control-plane/         # 控制平面源码
│   ├── api/              # HTTP API 处理器
│   ├── service/          # 业务逻辑服务
│   ├── storage/          # 存储抽象层
│   ├── middleware/       # HTTP 中间件
│   ├── logger/           # 日志组件
│   └── main.go          # 控制平面入口
├── wasm-plugin/          # WASM 插件源码 (Rust)
│   ├── src/             # Rust 源码
│   │   ├── lib.rs       # 插件主逻辑
│   │   ├── config.rs    # 配置管理
│   │   ├── matcher.rs   # 请求匹配
│   │   └── executor.rs  # 故障执行
│   ├── Cargo.toml       # Rust 项目配置
│   └── Cargo.lock       # 依赖锁定文件
├── k8s/                  # Kubernetes 部署文件
├── docs/                 # 文档目录
├── docker-compose.yaml   # 本地开发环境
├── Dockerfile.*          # 各组件构建文件
└── README.md            # 项目主页
```

### 目录职责说明

- `cli/`: 用户交互的命令行工具，使用 Go + Cobra 框架
- `control-plane/`: 系统大脑，管理策略和配置分发，使用 Go + Gin
- `wasm-plugin/`: 数据平面执行组件，嵌入到 Envoy 中，使用 Rust
- `k8s/`: 生产环境部署配置，包含所有 Kubernetes 清单文件
- `docs/`: 完整的项目文档，包括用户指南和开发者文档

## 🔨 构建指南

### 构建控制平面

```bash
cd control-plane

# 开发构建 (快速，包含调试信息)
go build -o hfi-control-plane .

# 生产构建 (优化，去除调试信息)
CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o hfi-control-plane .

# 验证构建
./hfi-control-plane --help
```

构建选项说明：
- `CGO_ENABLED=0`: 禁用 CGO，生成静态链接二进制文件
- `GOOS=linux`: 指定目标操作系统 (适用于 Docker 部署)
- `-ldflags="-w -s"`: 去除调试信息和符号表，减小文件大小

### 构建 WASM 插件

```bash
cd wasm-plugin

# 开发构建 (包含调试信息)
cargo build --target wasm32-unknown-unknown

# 生产构建 (优化)
cargo build --target wasm32-unknown-unknown --release

# 使用 wasm-opt 进一步优化 (可选)
wasm-opt -Oz --enable-bulk-memory \
  target/wasm32-unknown-unknown/release/hfi_wasm_plugin.wasm \
  -o optimized_plugin.wasm

# 查看构建产物
ls -la target/wasm32-unknown-unknown/release/hfi_wasm_plugin.wasm
```

构建优化说明：
- `--release`: 启用所有编译器优化
- `wasm-opt -Oz`: 进一步优化 WASM 文件大小 (可选)
- `--enable-bulk-memory`: 启用批量内存操作，提升性能

### 构建 CLI 工具

```bash
cd cli

# 开发构建
go build -o hfi-cli .

# 跨平台构建
# Linux
GOOS=linux GOARCH=amd64 go build -o hfi-cli-linux-amd64 .
# macOS
GOOS=darwin GOARCH=amd64 go build -o hfi-cli-darwin-amd64 .
# Windows
GOOS=windows GOARCH=amd64 go build -o hfi-cli-windows-amd64.exe .

# 验证构建
./hfi-cli --help
```

### 使用 Makefile 构建（推荐）

项目提供了统一的 Makefile 来简化构建过程：

```bash
# 查看所有可用目标
make help

# 构建所有组件
make build-all

# 构建单个组件
make build-control-plane
make build-wasm-plugin
make build-cli

# 跨平台构建 CLI
make build-cli-cross

# 运行测试
make test
make test-go      # 仅运行 Go 测试
make test-rust    # 仅运行 Rust 测试

# 代码质量检查
make fmt          # 格式化代码
make lint         # 代码检查
make coverage     # 生成覆盖率报告

# 环境管理
make setup        # 检查开发环境
make deps         # 更新依赖
make clean        # 清理构建产物

# 一键验证
make verify       # 清理 + 构建 + 测试
```

### 统一构建脚本

为了简化构建过程，可以使用项目根目录的构建脚本：

```bash
# 使用 Makefile (推荐)
make build-all

# 或者使用 Shell 脚本 (适用于不支持 Make 的环境)
./scripts/build.sh build-all

# 仅构建特定组件
make build-control-plane
make build-wasm-plugin
make build-cli

# 或者
./scripts/build.sh build-control-plane
./scripts/build.sh build-wasm-plugin
./scripts/build.sh build-cli
```

## 🏠 本地运行 (非 K8s)

### 使用 Docker Compose（推荐）

#### 使用 Makefile 快速启动

```bash
# 启动本地开发环境
make run-local

# 停止本地环境
make stop-local
```

#### 手动使用 Docker Compose

项目提供了完整的本地开发环境配置：

```bash
# 启动完整的本地环境
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f control-plane
docker-compose logs -f envoy
```

服务说明：
- 控制平面: `http://localhost:8080` - API 服务器
- Envoy 代理: `http://localhost:18000` - 代理入口
- Envoy 管理: `http://localhost:19000` - 管理接口
- etcd: `http://localhost:2379` - 存储后端
- 测试后端: `http://localhost:8081` - 简单的测试目标

### 手动启动服务

如果需要更细粒度的控制，可以手动启动各个服务：

#### 1. 启动 etcd

```bash
# 使用 Docker 启动 etcd
docker run -d \
  --name hfi-etcd \
  -p 2379:2379 \
  quay.io/coreos/etcd:v3.5.9 \
  etcd \
  --listen-client-urls http://0.0.0.0:2379 \
  --advertise-client-urls http://localhost:2379
```

#### 2. 启动控制平面

```bash
# 设置环境变量
export STORAGE_BACKEND=etcd
export ETCD_ENDPOINTS=http://localhost:2379
export LOG_LEVEL=debug

# 启动控制平面
cd control-plane
./hfi-control-plane
```

#### 3. 启动 Envoy (带 WASM 插件)

```bash
# 确保 WASM 插件已构建
cd wasm-plugin
cargo build --target wasm32-unknown-unknown --release
cp target/wasm32-unknown-unknown/release/hfi_wasm_plugin.wasm ../plugin.wasm

# 启动 Envoy
docker run -d \
  --name hfi-envoy \
  -p 18000:18000 \
  -p 19000:19000 \
  -v $(pwd)/envoy.yaml:/etc/envoy/envoy.yaml \
  -v $(pwd)/plugin.wasm:/etc/envoy/plugin.wasm \
  envoyproxy/envoy:v1.28.0 \
  envoy -c /etc/envoy/envoy.yaml --log-level info
```

#### 4. 启动测试后端

```bash
# 简单的测试后端
docker run -d \
  --name hfi-backend \
  -p 8081:80 \
  nginx:alpine
```

### 验证本地环境

```bash
# 验证控制平面
curl http://localhost:8080/v1/health

# 验证 Envoy 代理
curl http://localhost:18000/

# 验证 Envoy 管理接口
curl http://localhost:19000/stats

# 使用 CLI 测试
cd cli
./hfi-cli policy apply -f examples/delay-policy.yaml
./hfi-cli policy list
```

## 🧪 测试指南

### 使用 Makefile 测试（推荐）

```bash
# 运行所有测试
make test

# 运行特定语言的测试
make test-go       # 仅运行 Go 测试
make test-rust     # 仅运行 Rust 测试

# 生成覆盖率报告
make coverage

# 运行集成测试
make integration-test

# 一键验证 (构建 + 测试)
make verify
```

### Go 单元测试

```bash
# 运行所有 Go 测试
go test ./...

# 运行特定模块测试
cd control-plane
go test ./service/...
go test ./storage/...

# 运行测试并显示覆盖率
go test -cover ./...

# 生成详细的覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Rust 单元测试

```bash
cd wasm-plugin

# 运行所有 Rust 测试
cargo test

# 运行特定测试
cargo test test_matcher
cargo test test_executor

# 运行测试并显示详细输出
cargo test -- --nocapture

# 运行性能测试 (如果有)
cargo test --release bench_
```

### 集成测试

```bash
# 使用 Makefile (推荐)
make integration-test

# 手动运行
# 启动测试环境
docker-compose -f docker-compose.test.yaml up -d

# 等待服务启动
sleep 10

# 运行集成测试
cd control-plane
go test -tags=integration ./integration_test.go

# 或者使用测试脚本
./scripts/integration-test.sh

# 清理测试环境
docker-compose -f docker-compose.test.yaml down -v
```

### 端到端测试

```bash
# 使用完整的本地环境进行端到端测试
docker-compose up -d

# 等待所有服务就绪
./scripts/wait-for-services.sh

# 运行端到端测试场景
./scripts/e2e-test.sh

# 测试场景包括:
# 1. 策略创建和应用
# 2. 故障注入验证
# 3. 配置热更新
# 4. 指标收集验证
```

## 🛠️ 开发工作流

### 日常开发流程

```bash
# 1. 拉取最新代码
git pull origin main

# 2. 启动开发环境
docker-compose up -d etcd

# 3. 开发和测试
# 修改代码 -> 运行测试 -> 构建验证

# 4. 提交代码
git add .
git commit -m "feat: add new feature"
git push origin feature-branch
```

### 调试技巧

#### 控制平面调试

```bash
# 启用详细日志
export LOG_LEVEL=debug

# 使用 Go 调试器 (Delve)
dlv debug ./main.go -- --port=8080
```

#### WASM 插件调试

```bash
# 查看 Envoy 日志中的 WASM 输出
docker logs -f hfi-envoy 2>&1 | grep wasm

# 使用 Rust 日志 (在代码中添加)
log::info!("Debug message: {:?}", variable);
```

### 性能分析

```bash
# Go 性能分析
go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=.
go tool pprof cpu.prof

# Rust 性能分析
cargo test --release -- --nocapture
```

## 🔧 常见问题和解决方案

### 构建问题

问题: `cargo build` 失败，提示找不到 `wasm32-unknown-unknown` target
```bash
# 解决方案
rustup target add wasm32-unknown-unknown
```

问题: Go 构建时出现模块依赖错误
```bash
# 解决方案
go mod tidy
go mod vendor  # 可选，用于离线构建
```

### 运行时问题

问题: 控制平面无法连接到 etcd
```bash
# 检查 etcd 是否运行
docker ps | grep etcd

# 检查网络连接
telnet localhost 2379
```

问题: Envoy 无法加载 WASM 插件
```bash
# 检查 WASM 文件是否存在
ls -la plugin.wasm

# 检查 Envoy 配置
docker exec hfi-envoy cat /etc/envoy/envoy.yaml
```

### 开发环境重置

```bash
# 完全重置开发环境
docker-compose down -v
docker system prune -f
docker volume prune -f

# 重新构建和启动
docker-compose build --no-cache
docker-compose up -d
```

