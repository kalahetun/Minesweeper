# HFI Makefile - 统一构建脚本

.PHONY: help build-all build-control-plane build-wasm-plugin build-cli clean test test-go test-rust run-local stop-local

# 默认目标
help:
	@echo "HFI 项目构建工具"
	@echo ""
	@echo "可用目标:"
	@echo "  build-all           构建所有组件"
	@echo "  build-control-plane 构建控制平面"
	@echo "  build-wasm-plugin   构建 WASM 插件"
	@echo "  build-cli          构建 CLI 工具"
	@echo "  clean              清理构建产物"
	@echo "  test               运行所有测试"
	@echo "  test-go            运行 Go 测试"
	@echo "  test-rust          运行 Rust 测试"
	@echo "  run-local          启动本地开发环境"
	@echo "  stop-local         停止本地开发环境"
	@echo "  verify             验证构建和测试"

# 构建所有组件
build-all: build-control-plane build-wasm-plugin build-cli
	@echo "✅ 所有组件构建完成"

# 构建控制平面
build-control-plane:
	@echo "🏗️ 构建控制平面..."
	@cd control-plane && \
		CGO_ENABLED=0 go build -ldflags="-w -s" -o hfi-control-plane .
	@echo "✅ 控制平面构建完成: control-plane/hfi-control-plane"

# 构建 WASM 插件
build-wasm-plugin:
	@echo "🏗️ 构建 WASM 插件..."
	@cd wasm-plugin && \
		cargo build --target wasm32-unknown-unknown --release
	@cp wasm-plugin/target/wasm32-unknown-unknown/release/hfi_wasm_plugin.wasm plugin.wasm
	@echo "✅ WASM 插件构建完成: plugin.wasm"

# 构建 CLI 工具
build-cli:
	@echo "🏗️ 构建 CLI 工具..."
	@cd cli && \
		CGO_ENABLED=0 go build -ldflags="-w -s" -o hfi-cli .
	@echo "✅ CLI 工具构建完成: cli/hfi-cli"

# 跨平台构建 CLI
build-cli-cross:
	@echo "🏗️ 跨平台构建 CLI 工具..."
	@cd cli && \
		GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o hfi-cli-linux-amd64 . && \
		GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o hfi-cli-darwin-amd64 . && \
		GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o hfi-cli-windows-amd64.exe .
	@echo "✅ 跨平台 CLI 构建完成"

# 清理构建产物
clean:
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
run-local:
	@echo "🚀 启动本地开发环境..."
	@docker-compose up -d
	@echo "等待服务启动..."
	@sleep 10
	@echo "✅ 本地环境启动完成"
	@echo ""
	@echo "服务地址:"
	@echo "  控制平面: http://localhost:8080"
	@echo "  Envoy 代理: http://localhost:18000"
	@echo "  Envoy 管理: http://localhost:19000"
	@echo ""
	@echo "使用 'make stop-local' 停止环境"

# 停止本地开发环境
stop-local:
	@echo "🛑 停止本地开发环境..."
	@docker-compose down -v
	@echo "✅ 本地环境已停止"

# 验证构建和测试
verify: clean build-all test
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
docker-build:
	@echo "🐳 构建 Docker 镜像..."
	@docker build -f Dockerfile.controlplane -t hfi/control-plane:latest .
	@docker build -f Dockerfile.wasm -t hfi/wasm-plugin:latest .
	@echo "✅ Docker 镜像构建完成"

# 集成测试
integration-test: run-local
	@echo "🔬 运行集成测试..."
	@sleep 5  # 等待服务完全启动
	@cd control-plane && go test -tags=integration ./integration_test.go
	@make stop-local
	@echo "✅ 集成测试完成"
