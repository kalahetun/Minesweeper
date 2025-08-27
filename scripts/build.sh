#!/bin/bash

# HFI 项目统一构建脚本
# 此脚本提供了 Makefile 的替代方案，适用于不支持 Make 的环境

set -e  # 遇到错误时退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        log_error "$1 未安装或不在 PATH 中"
        return 1
    fi
    return 0
}

# 检查开发环境
check_environment() {
    log_info "检查开发环境..."
    
    # 检查 Go
    if check_command go; then
        GO_VERSION=$(go version | cut -d' ' -f3)
        log_success "Go 已安装: $GO_VERSION"
    else
        log_error "请安装 Go 1.24+"
        exit 1
    fi
    
    # 检查 Rust
    if check_command rustc; then
        RUST_VERSION=$(rustc --version | cut -d' ' -f2)
        log_success "Rust 已安装: $RUST_VERSION"
        
        # 检查 WASM 目标
        if rustup target list --installed | grep -q wasm32-unknown-unknown; then
            log_success "WASM 目标已安装"
        else
            log_warning "WASM 目标未安装，正在安装..."
            rustup target add wasm32-unknown-unknown
        fi
    else
        log_error "请安装 Rust 1.89+"
        exit 1
    fi
    
    # 检查 Docker
    if check_command docker; then
        DOCKER_VERSION=$(docker --version | cut -d' ' -f3 | tr -d ',')
        log_success "Docker 已安装: $DOCKER_VERSION"
    else
        log_warning "Docker 未安装，跳过容器相关功能"
    fi
    
    log_success "开发环境检查完成"
}

# 构建控制平面
build_control_plane() {
    log_info "构建控制平面..."
    cd control-plane
    
    CGO_ENABLED=0 go build -ldflags="-w -s" -o hfi-control-plane .
    
    if [ -f "hfi-control-plane" ]; then
        log_success "控制平面构建完成: control-plane/hfi-control-plane"
    else
        log_error "控制平面构建失败"
        exit 1
    fi
    
    cd ..
}

# 构建 WASM 插件
build_wasm_plugin() {
    log_info "构建 WASM 插件..."
    cd wasm-plugin
    
    cargo build --target wasm32-unknown-unknown --release
    
    if [ -f "target/wasm32-unknown-unknown/release/hfi_wasm_plugin.wasm" ]; then
        cp target/wasm32-unknown-unknown/release/hfi_wasm_plugin.wasm ../plugin.wasm
        log_success "WASM 插件构建完成: plugin.wasm"
    else
        log_error "WASM 插件构建失败"
        exit 1
    fi
    
    cd ..
}

# 构建 CLI 工具
build_cli() {
    log_info "构建 CLI 工具..."
    cd cli
    
    CGO_ENABLED=0 go build -ldflags="-w -s" -o hfi-cli .
    
    if [ -f "hfi-cli" ]; then
        log_success "CLI 工具构建完成: cli/hfi-cli"
    else
        log_error "CLI 工具构建失败"
        exit 1
    fi
    
    cd ..
}

# 运行 Go 测试
test_go() {
    log_info "运行 Go 测试..."
    
    cd control-plane
    go test -v ./...
    cd ..
    
    cd cli
    go test -v ./...
    cd ..
    
    log_success "Go 测试完成"
}

# 运行 Rust 测试
test_rust() {
    log_info "运行 Rust 测试..."
    
    cd wasm-plugin
    cargo test
    cd ..
    
    log_success "Rust 测试完成"
}

# 清理构建产物
clean() {
    log_info "清理构建产物..."
    
    rm -f control-plane/hfi-control-plane
    rm -f cli/hfi-cli
    rm -f plugin.wasm
    
    if [ -d "wasm-plugin/target" ]; then
        cd wasm-plugin
        cargo clean
        cd ..
    fi
    
    if [ -d "control-plane" ]; then
        cd control-plane
        go clean
        cd ..
    fi
    
    if [ -d "cli" ]; then
        cd cli
        go clean
        cd ..
    fi
    
    log_success "清理完成"
}

# 启动本地环境
run_local() {
    log_info "启动本地开发环境..."
    
    if ! check_command docker-compose && ! check_command docker; then
        log_error "Docker 或 Docker Compose 未安装"
        exit 1
    fi
    
    # 尝试使用新的 docker compose 命令
    if docker compose version &> /dev/null; then
        docker compose up -d
    elif docker-compose version &> /dev/null; then
        docker-compose up -d
    else
        log_error "Docker Compose 未安装"
        exit 1
    fi
    
    log_info "等待服务启动..."
    sleep 10
    
    log_success "本地环境启动完成"
    echo ""
    echo "服务地址:"
    echo "  控制平面: http://localhost:8080"
    echo "  Envoy 代理: http://localhost:18000"
    echo "  Envoy 管理: http://localhost:19000"
    echo ""
    echo "使用 '$0 stop' 停止环境"
}

# 停止本地环境
stop_local() {
    log_info "停止本地开发环境..."
    
    # 尝试使用新的 docker compose 命令
    if docker compose version &> /dev/null; then
        docker compose down -v
    elif docker-compose version &> /dev/null; then
        docker-compose down -v
    else
        log_error "Docker Compose 未安装"
        exit 1
    fi
    
    log_success "本地环境已停止"
}

# 显示帮助信息
show_help() {
    echo "HFI 项目构建脚本"
    echo ""
    echo "用法: $0 [命令]"
    echo ""
    echo "命令:"
    echo "  setup               检查开发环境"
    echo "  build-all           构建所有组件"
    echo "  build-control-plane 构建控制平面"
    echo "  build-wasm-plugin   构建 WASM 插件"
    echo "  build-cli          构建 CLI 工具"
    echo "  test               运行所有测试"
    echo "  test-go            运行 Go 测试"
    echo "  test-rust          运行 Rust 测试"
    echo "  clean              清理构建产物"
    echo "  run                启动本地环境"
    echo "  stop               停止本地环境"
    echo "  verify             清理 + 构建 + 测试"
    echo "  help               显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 setup           # 检查开发环境"
    echo "  $0 build-all       # 构建所有组件"
    echo "  $0 verify          # 完整验证"
}

# 主函数
main() {
    case "${1:-help}" in
        "setup")
            check_environment
            ;;
        "build-all")
            build_control_plane
            build_wasm_plugin
            build_cli
            log_success "所有组件构建完成"
            ;;
        "build-control-plane")
            build_control_plane
            ;;
        "build-wasm-plugin")
            build_wasm_plugin
            ;;
        "build-cli")
            build_cli
            ;;
        "test")
            test_go
            test_rust
            log_success "所有测试完成"
            ;;
        "test-go")
            test_go
            ;;
        "test-rust")
            test_rust
            ;;
        "clean")
            clean
            ;;
        "run")
            run_local
            ;;
        "stop")
            stop_local
            ;;
        "verify")
            clean
            check_environment
            build_control_plane
            build_wasm_plugin
            build_cli
            test_go
            test_rust
            log_success "验证完成 - 所有组件构建成功，测试通过"
            ;;
        "help"|*)
            show_help
            ;;
    esac
}

# 运行主函数
main "$@"
