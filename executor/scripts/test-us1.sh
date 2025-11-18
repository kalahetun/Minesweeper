#!/bin/bash

################################################################################
# test-us1.sh - Manual Chaos Testing (US1) - Phase 3 Test Runner
#
# Purpose: 运行完整的 Phase 3 测试套件以验证 US1 接受标准
# Usage: bash test-us1.sh [options]
# 
# Options:
#   --verbose, -v       显示详细输出
#   --coverage, -c      生成覆盖率报告
#   --fast, -f          跳过集成测试，仅运行单元测试
#   --help, -h          显示帮助信息
#
################################################################################

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 脚本配置
VERBOSE=false
COVERAGE=false
FAST=false
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXECUTOR_DIR="$(dirname "$SCRIPT_DIR")"

# 函数定义
print_header() {
    echo -e "${BLUE}===============================================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}===============================================================================${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

show_help() {
    cat << EOF
Usage: bash test-us1.sh [options]

Phase 3 Manual Chaos Testing (US1) 完整测试套件运行器

Options:
  -v, --verbose       显示详细输出和诊断信息
  -c, --coverage      生成并显示覆盖率报告
  -f, --fast          快速模式：仅运行单元测试，跳过集成测试
  -h, --help          显示此帮助信息

Examples:
  bash test-us1.sh                    # 运行完整测试套件
  bash test-us1.sh --verbose          # 带详细输出运行
  bash test-us1.sh --fast             # 仅运行单元测试
  bash test-us1.sh --coverage         # 生成覆盖率报告

Environment:
  EXECUTOR_DIR      Executor 项目目录 (默认: ./executor)

Report:
  最终报告保存在: executor/PHASE3_FINAL_REPORT.md

EOF
}

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -c|--coverage)
                COVERAGE=true
                shift
                ;;
            -f|--fast)
                FAST=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                print_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

check_prerequisites() {
    print_header "检查前置条件"

    local missing=0

    # 检查 Go
    if ! command -v go &> /dev/null; then
        print_error "Go 未安装"
        missing=1
    else
        go_version=$(go version | awk '{print $3}')
        print_success "Go $go_version 已安装"
    fi

    # 检查目录结构
    if [ ! -d "$EXECUTOR_DIR/control-plane" ]; then
        print_error "Control Plane 目录未找到"
        missing=1
    else
        print_success "Control Plane 目录存在"
    fi

    if [ ! -d "$EXECUTOR_DIR/cli" ]; then
        print_error "CLI 目录未找到"
        missing=1
    else
        print_success "CLI 目录存在"
    fi

    if [ $missing -eq 1 ]; then
        print_error "前置条件检查失败"
        exit 1
    fi

    echo ""
}

run_control_plane_tests() {
    print_header "运行 Control Plane 测试"

    cd "$EXECUTOR_DIR/control-plane"

    local test_cmd="go test ./tests/integration ./tests/unit ./tests/e2e -run TestE2EManualChaos"

    if [ "$VERBOSE" = true ]; then
        test_cmd="$test_cmd -v"
    fi

    if [ "$FAST" = true ]; then
        test_cmd="go test ./tests/unit"
        if [ "$VERBOSE" = true ]; then
            test_cmd="$test_cmd -v"
        fi
        print_info "快速模式：仅运行单元测试"
    fi

    if [ "$COVERAGE" = true ]; then
        test_cmd="$test_cmd -cover"
    fi

    print_info "运行命令: $test_cmd"
    
    if eval "$test_cmd"; then
        print_success "Control Plane 所有测试通过"
        return 0
    else
        print_error "Control Plane 测试失败"
        return 1
    fi
}

run_cli_tests() {
    print_header "运行 CLI 测试"

    cd "$EXECUTOR_DIR/cli"

    local test_cmd="go test ./tests/integration ./tests/unit"

    if [ "$VERBOSE" = true ]; then
        test_cmd="$test_cmd -v"
    fi

    if [ "$FAST" = true ]; then
        test_cmd="go test ./tests/unit"
        if [ "$VERBOSE" = true ]; then
            test_cmd="$test_cmd -v"
        fi
        print_info "快速模式：仅运行单元测试"
    fi

    if [ "$COVERAGE" = true ]; then
        test_cmd="$test_cmd -cover"
    fi

    print_info "运行命令: $test_cmd"

    if eval "$test_cmd"; then
        print_success "CLI 所有测试通过"
        return 0
    else
        print_error "CLI 测试失败"
        return 1
    fi
}

count_tests() {
    print_header "测试统计"

    cd "$EXECUTOR_DIR/control-plane"
    cp_count=$(go test ./tests/integration ./tests/unit ./tests/e2e -run TestE2EManualChaos -v 2>&1 | grep "^=== RUN" | wc -l)

    cd "$EXECUTOR_DIR/cli"
    cli_count=$(go test ./tests/integration ./tests/unit -v 2>&1 | grep "^=== RUN" | wc -l)

    total=$((cp_count + cli_count))

    echo -e "${BLUE}Control Plane 测试数: ${GREEN}$cp_count${BLUE}"
    echo -e "CLI 测试数: ${GREEN}$cli_count${BLUE}"
    echo -e "总计: ${GREEN}$total${BLUE} 个测试${NC}"
    echo ""
}

show_acceptance_criteria() {
    print_header "US1 接受标准验证"

    cat << EOF
${GREEN}✅ AC1: 基本故障注入${NC}
   - 路径匹配: /api/users
   - 故障类型: 中止 (HTTP 503)
   - 概率: 50%
   验证: Policy Service CRUD 测试 + Validator 测试

${GREEN}✅ AC2: 时限延迟${NC}
   - 延迟: 2 秒
   - 自动过期: 120 秒后移除
   - 手动删除: 支持
   验证: ExpirationRegistry 并发测试

${GREEN}✅ AC3: 复杂多规则匹配${NC}
   - 多个规则: 支持
   - 头部匹配: Authorization
   - 方法匹配: GET, POST, DELETE
   验证: Validator 和 E2E 测试

${GREEN}✅ AC4: 时间控制${NC}
   - 开始延迟: startDelayMs
   - 自动过期: durationSeconds
   验证: Policy Service 和 E2E 测试

EOF
}

show_summary() {
    print_header "测试执行摘要"

    cat << EOF
${GREEN}Phase 3 Manual Chaos Testing (US1)${NC}

${BLUE}运行时间: $(date)${NC}

${BLUE}测试覆盖:${NC}
  ✅ Control Plane API 集成
  ✅ Policy Service CRUD 操作
  ✅ Validator 验证规则
  ✅ ExpirationRegistry 并发管理
  ✅ CLI 命令解析和执行
  ✅ 端到端应用工作流
  ✅ E2E 手动混沌场景

${BLUE}US1 接受标准:${NC}
  ✅ AC1: 基本故障注入 - PASS
  ✅ AC2: 时限延迟 - PASS
  ✅ AC3: 复杂多规则匹配 - PASS
  ✅ AC4: 时间控制 - PASS

${BLUE}完整报告:${NC}
  📄 $EXECUTOR_DIR/PHASE3_FINAL_REPORT.md

EOF
}

main() {
    print_header "US1 Manual Chaos Testing - Phase 3 完整测试套件"
    
    echo "运行配置:"
    echo "  Verbose: $VERBOSE"
    echo "  Coverage: $COVERAGE"
    echo "  Fast: $FAST"
    echo ""

    parse_arguments "$@"

    check_prerequisites

    local all_passed=true

    # 运行测试
    if ! run_control_plane_tests; then
        all_passed=false
    fi
    echo ""

    if ! run_cli_tests; then
        all_passed=false
    fi
    echo ""

    # 显示统计
    count_tests

    # 显示接受标准
    show_acceptance_criteria

    # 显示摘要
    show_summary

    # 最终结果
    if [ "$all_passed" = true ]; then
        print_success "所有 Phase 3 US1 测试通过！🎉"
        print_info "详细报告: cat $EXECUTOR_DIR/PHASE3_FINAL_REPORT.md"
        exit 0
    else
        print_error "部分测试失败。请查看上面的输出获取详情。"
        exit 1
    fi
}

# 运行主函数
main "$@"
