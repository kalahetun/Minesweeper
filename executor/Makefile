# HFI Makefile - 统一构建脚本

.PHONY: help build-all build-bin-control-plane build-bin-wasm-plugin build-bin-cli build-bin-clean build-docker build-k3s clean test test-go test-rust integration-test coverage verify setup fmt lint deps security run-local-docker stop-local-docker deploy-all deploy-namespace deploy-wasm-server deploy-control deploy-plugin deploy-envoyfilter undeploy redeploy status-k3s update-wasm-plugin logs-wasm-plugin logs-control test-k3s test-k8s-all

# 默认目标
help:
	@echo "BOIFI Fault Injection System - Makefile 帮助"
	@echo ""
	@echo "🏗️  构建目标:"
	@echo "  make build-all              - 构建所有组件 (控制平面 + CLI + Wasm插件)"
	@echo "  make build-bin-control-plane  - 构建控制平面"
	@echo "  make build-bin-wasm-plugin    - 构建 Wasm 插件"
	@echo "  make build-bin-cli            - 构建 CLI 工具"
	@echo "  make build-bin-clean          - 清理构建产物"
	@echo "  make build-docker             - 构建 Docker 镜像"
	@echo "  make build-k3s                - 构建镜像并导入到 k3s containerd"
	@echo ""
	@echo "🧪 测试目标:"
	@echo "  make test                   - 运行所有测试 (Go + Rust)"
	@echo "  make test-go                - 运行 Go 测试"
	@echo "  make test-rust              - 运行 Rust 测试"
	@echo "  make integration-test       - 运行集成测试"
	@echo "  make coverage               - 生成代码覆盖率"
	@echo "  make test-k3s               - 运行 K8s 端到端测试"
	@echo "  make test-k8s-all           - 运行所有 K8s 测试"
	@echo ""
	@echo "☸️  Kubernetes 部署:"
	@echo "  make deploy-all         - 完整 K8s 部署 (命名空间 + 所有组件)"
	@echo "  make deploy-namespace       - 创建命名空间"
	@echo "  make deploy-wasm-server     - 部署 Wasm 插件服务器"
	@echo "  make deploy-control     - 部署控制平面"
	@echo "  make deploy-plugin      - 部署 WasmPlugin CRD"
	@echo "  make deploy-envoyfilter - 部署 EnvoyFilter"
	@echo "  make undeploy               - 卸载所有 K8s 资源"
	@echo "  make redeploy               - 快速重新部署 (卸载 + 部署)"
	@echo "  make status-k3s             - 检查 K8s 部署状态"
	@echo "  make update-wasm-plugin     - 更新 Wasm 插件并重启"
	@echo ""
	@echo "📊 监控与调试:"
	@echo "  make logs-wasm-plugin       - 查看 Wasm 插件日志"
	@echo "  make logs-control       - 查看控制平面日志"
	@echo ""
	@echo "🐳 本地开发环境:"
	@echo "  make run-local-docker       - 本地 Docker 环境启动"
	@echo "  make stop-local-docker      - 本地 Docker 环境停止"
	@echo ""
	@echo "🔧 开发工具:"
	@echo "  make verify                 - 验证 (构建+测试)"
	@echo "  make setup                  - 检查开发环境依赖"
	@echo "  make fmt                    - 代码格式化"
	@echo "  make lint                   - 代码检查"
	@echo "  make deps                   - 更新依赖"
	@echo "  make security               - 安全扫描"
	@echo "  make coverage               - 测试覆盖率"
	@echo ""
	@echo "📚 帮助:"
	@echo "  make help                   - 显示此帮助信息"
	@echo ""

# 构建所有组件
build-all: build-bin-control-plane build-bin-wasm-plugin build-bin-cli
	@echo "✅ 所有组件构建完成"

# 构建控制平面
build-bin-control-plane:
	@echo "🏗️ 构建控制平面..."
	@cd control-plane && \
		CGO_ENABLED=0 go build -ldflags="-w -s" -o hfi-control-plane .
	@echo "✅ 控制平面构建完成: control-plane/hfi-control-plane"

# 构建 WASM 插件
build-bin-wasm-plugin:
	@echo "🏗️ 构建 WASM 插件..."
	@cd wasm-plugin && \
		cargo build --target wasm32-unknown-unknown --release
	@cp wasm-plugin/target/wasm32-unknown-unknown/release/hfi_wasm_plugin.wasm plugin.wasm
	@echo "✅ WASM 插件构建完成: plugin.wasm"

# 构建 CLI 工具
build-bin-cli:
	@echo "🏗️ 构建 CLI 工具..."
	@cd cli && \
		CGO_ENABLED=0 go build -ldflags="-w -s" -o hfi-cli .
	@echo "✅ CLI 工具构建完成: cli/hfi-cli"

# 清理构建产物
build-bin-clean:
	@echo "🧹 清理构建产物..."
	@rm -f control-plane/hfi-control-plane
	@rm -f cli/hfi-cli cli/hfi-cli-*
	@rm -f plugin.wasm
	@cd wasm-plugin && cargo clean
	@cd control-plane && go clean
	@cd cli && go clean
	@echo "✅ 清理完成"

# 运行所有测试
test: test-go test-rust
	@echo "✅ 所有测试完成"

# 运行 Go 测试
test-go:
	@echo "🧪 运行 Go 测试..."
	@cd control-plane && go test -v ./...
	@cd cli && go test -v ./...
	@echo "✅ Go 测试完成"

# 运行 Rust 测试
test-rust:
	@echo "🧪 运行 Rust 测试..."
	@cd wasm-plugin && cargo test
	@echo "✅ Rust 测试完成"

# 代码覆盖率
coverage:
	@echo "📊 生成代码覆盖率报告..."
	@cd control-plane && \
		go test -coverprofile=coverage.out ./... && \
		go tool cover -html=coverage.out -o coverage.html
	@cd cli && \
		go test -coverprofile=coverage.out ./... && \
		go tool cover -html=coverage.out -o coverage.html
	@echo "✅ 覆盖率报告生成完成"

# 启动本地开发环境
run-local-docker:
	@echo "🚀 启动本地开发环境..."
	@cd docker && docker-compose up -d
	@echo "等待服务启动..."
	@echo ""
	@echo "服务地址:"
	@echo "  控制平面: http://localhost:8080"
	@echo "  Envoy 代理: http://localhost:18000"
	@echo "  Envoy 管理: http://localhost:19000"
	@echo ""
	@echo "使用 'make stop-local-docker' 停止环境"

# 停止本地开发环境
stop-local-docker:
	@echo "🛑 停止本地开发环境..."
	@cd docker && docker-compose down -v
	@echo "✅ 本地环境已停止"

# 验证构建和测试
verify: build-bin-clean build-all test
	@echo "🎉 验证完成 - 所有组件构建成功，测试通过"

# 安装开发依赖
setup:
	@echo "🔧 安装开发依赖..."
	@echo "检查 Go 环境..."
	@go version || (echo "❌ Go 未安装" && exit 1)
	@echo "检查 Rust 环境..."
	@rustc --version || (echo "❌ Rust 未安装" && exit 1)
	@echo "检查 WASM 目标..."
	@rustup target list --installed | grep wasm32-unknown-unknown || rustup target add wasm32-unknown-unknown
	@echo "检查 Docker 环境..."
	@docker --version || (echo "❌ Docker 未安装" && exit 1)
	@echo "✅ 开发环境检查完成"

# 代码格式化
fmt:
	@echo "🎨 格式化代码..."
	@cd control-plane && go fmt ./...
	@cd cli && go fmt ./...
	@cd wasm-plugin && cargo fmt
	@echo "✅ 代码格式化完成"

# 代码检查
lint:
	@echo "🔍 代码检查..."
	@cd control-plane && go vet ./...
	@cd cli && go vet ./...
	@cd wasm-plugin && cargo clippy -- -D warnings
	@echo "✅ 代码检查完成"

# 更新依赖
deps:
	@echo "📦 更新依赖..."
	@cd control-plane && go mod tidy
	@cd cli && go mod tidy
	@cd wasm-plugin && cargo update
	@echo "✅ 依赖更新完成"

# 安全扫描
security:
	@echo "🔒 安全扫描..."
	@cd control-plane && go list -json -m all | nancy sleuth
	@cd cli && go list -json -m all | nancy sleuth
	@cd wasm-plugin && cargo audit
	@echo "✅ 安全扫描完成"

# Docker 镜像构建
build-docker:
	@echo "🐳 构建 Docker 镜像..."
	@docker build -f docker/Dockerfile.controlplane -t boifi/control-plane:latest .
	@docker build -f docker/Dockerfile.wasm -t boifi/wasm-plugin:latest .
	@echo "✅ Docker 镜像构建完成"

# 构建并导入控制平面镜像到 k3s containerd
build-k3s: build-docker
	@echo "🚀 导入控制平面镜像到 k3s containerd..."
	@which sudo > /dev/null || (echo "❌ sudo 不可用"; exit 1)
	@which ctr > /dev/null || (echo "❌ ctr 不可用"; exit 1)
	@docker save boifi/control-plane:latest | sudo ctr -a /run/k3s/containerd/containerd.sock -n k8s.io images import -

# 集成测试
integration-test: run-local-docker
	@echo "🔬 运行集成测试..."
	@sleep 5  # 等待服务完全启动
	@cd control-plane && go test -tags=integration ./integration_test.go
	@make stop-local-docker
	@echo "✅ 集成测试完成"

#==============================================================================
# Kubernetes 部署目标
#==============================================================================

# 部署到 Kubernetes - 完整流程
deploy-all: build-k3s deploy-namespace deploy-wasm-server deploy-control deploy-plugin deploy-envoyfilter
	@echo "✅ K8s 完整部署完成"
	@echo ""
	@echo "📊 部署状态:"
	@make status-k3s

# 创建命名空间
deploy-namespace:
	@echo "🔧 创建 K8s 命名空间..."
	@kubectl apply -f k8s/namespace.yaml
	@echo "✅ 命名空间创建完成"

# 部署 Wasm 插件服务器
deploy-wasm-server: build-bin-wasm-plugin
	@echo "📦 部署 Wasm 插件服务器..."
	@echo "  1. 复制 Wasm 插件到主机路径..."
	@sudo mkdir -p /tmp/wasm-plugin
	@sudo cp wasm-plugin/target/wasm32-unknown-unknown/release/hfi_wasm_plugin.wasm /tmp/wasm-plugin/plugin.wasm
	@sudo chmod 644 /tmp/wasm-plugin/plugin.wasm
	@echo "  2. 部署 wasm-server..."
	@kubectl apply -f k8s/wasm-server.yaml
	@echo "  3. 等待 wasm-server 就绪..."
	@kubectl wait --for=condition=ready pod -l app=wasm-server -n boifi --timeout=120s || true
	@echo "✅ Wasm 插件服务器部署完成"

# 部署控制平面
deploy-control:
	@echo "🎛️  部署控制平面..."
	@kubectl apply -f k8s/control-plane.yaml
	@echo "  等待控制平面就绪..."
	@kubectl wait --for=condition=ready pod -l app=control-plane -n boifi --timeout=120s || true
	@echo "✅ 控制平面部署完成"

# 部署 WasmPlugin CRD
deploy-plugin:
	@echo "🔌 部署 WasmPlugin CRD..."
	@kubectl apply -f k8s/wasmplugin.yaml
	@echo "  等待 WasmPlugin 生效 (5秒)..."
	@sleep 5
	@echo "✅ WasmPlugin 部署完成"

# 部署 EnvoyFilter (可选)
deploy-envoyfilter:
	@echo "🔧 部署 EnvoyFilter (统计匹配器)..."
	@kubectl apply -f k8s/envoyfilter-wasm-stats.yaml
	@echo "✅ EnvoyFilter 部署完成"

# 卸载所有 K8s 资源
undeploy:
	@echo "🗑️  卸载 K8s 资源..."
	@kubectl delete -f k8s/envoyfilter-wasm-stats.yaml --ignore-not-found=true
	@kubectl delete -f k8s/wasmplugin.yaml --ignore-not-found=true
	@kubectl delete -f k8s/control-plane.yaml --ignore-not-found=true
	@kubectl delete -f k8s/wasm-server.yaml --ignore-not-found=true
	@kubectl delete -f k8s/namespace.yaml --ignore-not-found=true
	@echo "✅ K8s 资源卸载完成"

# 检查 K8s 部署状态
status-k3s:
	@echo "📊 检查 K8s 部署状态..."
	@echo ""
	@echo "=== Namespaces ==="
	@kubectl get ns boifi demo 2>/dev/null || echo "  ⚠️  命名空间不存在"
	@echo ""
	@echo "=== Pods in boifi namespace ==="
	@kubectl get pods -n boifi 2>/dev/null || echo "  ⚠️  无 pods"
	@echo ""
	@echo "=== Services in boifi namespace ==="
	@kubectl get svc -n boifi 2>/dev/null || echo "  ⚠️  无 services"
	@echo ""
	@echo "=== WasmPlugin ==="
	@kubectl get wasmplugin -A 2>/dev/null || echo "  ⚠️  WasmPlugin CRD 未安装或无实例"
	@echo ""
	@echo "=== EnvoyFilter ==="
	@kubectl get envoyfilter -A 2>/dev/null || echo "  ⚠️  无 EnvoyFilter"
	@echo ""

# 更新 Wasm 插件 (重新编译并更新)
update-wasm-plugin: build-bin-wasm-plugin
	@echo "🔄 更新 Wasm 插件..."
	@echo "  1. 复制新版本插件到主机路径..."
	@sudo cp wasm-plugin/target/wasm32-unknown-unknown/release/hfi_wasm_plugin.wasm /tmp/wasm-plugin/plugin.wasm
	@sudo chmod 644 /tmp/wasm-plugin/plugin.wasm
	@echo "  2. 重启 wasm-server 使其生效..."
	@kubectl rollout restart deployment/wasm-server -n boifi
	@kubectl wait --for=condition=ready pod -l app=wasm-server -n boifi --timeout=120s || true
	@echo "  3. 触发 Istio sidecar 重新加载插件..."
	@kubectl get pods -n demo -l app --no-headers | awk '{print $$1}' | xargs -I {} kubectl exec {} -n demo -c istio-proxy -- curl -X POST http://localhost:15000/reset_counters 2>/dev/null || true
	@echo "✅ Wasm 插件更新完成"

# 查看 Wasm 插件日志
logs-wasm-plugin:
	@echo "📝 查看 Wasm 插件日志 (从 demo namespace 的第一个 pod)..."
	@POD=$$(kubectl get pods -n demo -l app --no-headers | head -1 | awk '{print $$1}'); \
	if [ -n "$$POD" ]; then \
		echo "  Pod: $$POD"; \
		kubectl logs -n demo $$POD -c istio-proxy --tail=50; \
	else \
		echo "  ⚠️  demo namespace 中没有找到应用 pod"; \
	fi

# 查看控制平面日志
logs-control:
	@echo "📝 查看控制平面日志..."
	@kubectl logs -n boifi -l app=control-plane --tail=50

# 运行所有 K8s 测试
test-k8s-all:
	@echo "🧪 运行所有 K8s 测试..."
	@bash k8s/tests/run-all-tests.sh
	@echo "✅ 所有 K8s 测试完成"

# K8s 快速重新部署 (卸载 + 部署)
redeploy: undeploy deploy-all
	@echo "✅ K8s 快速重新部署完成"
