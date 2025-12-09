#!/bin/bash
# Run All E2E Tests for BOIFI Executor
#
# This script runs all E2E test scripts in sequence and reports results.
#
# Prerequisites:
#   - kubectl configured for cluster access
#   - Control Plane deployed in boifi namespace
#   - Wasm Plugin deployed via WasmPlugin CRD
#   - Demo namespace with Istio injection enabled
#
# Usage:
#   ./run-all-tests.sh [--verbose] [--stop-on-failure]
#
# Options:
#   --verbose         : Show detailed output from each test
#   --stop-on-failure : Stop execution on first test failure
#
# Exit Codes:
#   0 - All tests passed
#   1 - One or more tests failed
#   2 - Environment not ready / prerequisites missing

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERBOSE=false
STOP_ON_FAILURE=false
FAILED_TESTS=()
PASSED_TESTS=()
SKIPPED_TESTS=()

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --verbose)
            VERBOSE=true
            shift
            ;;
        --stop-on-failure)
            STOP_ON_FAILURE=true
            shift
            ;;
        -h|--help)
            grep "^#" "$0" | tail -n +2 | sed 's/^# //' | sed 's/^#//'
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 2
            ;;
    esac
done

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_section() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

# Check prerequisites
check_prerequisites() {
    log_section "Checking Prerequisites"
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl not found"
        exit 2
    fi
    log_info "✓ kubectl found"
    
    # Check cluster connection
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 2
    fi
    log_info "✓ Cluster connection OK"
    
    # Check boifi namespace
    if ! kubectl get namespace boifi &> /dev/null; then
        log_error "Namespace 'boifi' not found"
        exit 2
    fi
    log_info "✓ Namespace 'boifi' exists"
    
    # Check demo namespace
    if ! kubectl get namespace demo &> /dev/null; then
        log_warn "Namespace 'demo' not found (some tests may be skipped)"
    else
        log_info "✓ Namespace 'demo' exists"
    fi
    
    # Check Control Plane deployment
    if ! kubectl get deployment hfi-control-plane -n boifi &> /dev/null; then
        log_error "Control Plane deployment not found in boifi namespace"
        exit 2
    fi
    log_info "✓ Control Plane deployed"
    
    # Check Control Plane pods ready
    local ready_pods=$(kubectl get pods -n boifi -l app=control-plane -o jsonpath='{.items[?(@.status.phase=="Running")].metadata.name}' | wc -w)
    if [ "$ready_pods" -eq 0 ]; then
        log_error "No Control Plane pods are running"
        exit 2
    fi
    log_info "✓ Control Plane pods running ($ready_pods replicas)"
    
    echo ""
    log_info "All prerequisites met"
}

# Run a single test script
run_test() {
    local test_script=$1
    local test_name=$(basename "$test_script" .sh)
    
    log_section "Running Test: $test_name"
    
    if [ ! -f "$test_script" ]; then
        log_error "Test script not found: $test_script"
        SKIPPED_TESTS+=("$test_name")
        return 1
    fi
    
    if [ ! -x "$test_script" ]; then
        log_warn "Test script not executable, setting +x: $test_script"
        chmod +x "$test_script"
    fi
    
    # Run test with timeout
    local start_time=$(date +%s)
    
    if [ "$VERBOSE" = true ]; then
        if bash "$test_script"; then
            local end_time=$(date +%s)
            local duration=$((end_time - start_time))
            log_info "✓ Test PASSED in ${duration}s: $test_name"
            PASSED_TESTS+=("$test_name")
            return 0
        else
            local end_time=$(date +%s)
            local duration=$((end_time - start_time))
            log_error "✗ Test FAILED after ${duration}s: $test_name"
            FAILED_TESTS+=("$test_name")
            return 1
        fi
    else
        # Capture output and only show on failure
        local output_file=$(mktemp)
        if bash "$test_script" > "$output_file" 2>&1; then
            local end_time=$(date +%s)
            local duration=$((end_time - start_time))
            log_info "✓ Test PASSED in ${duration}s: $test_name"
            PASSED_TESTS+=("$test_name")
            rm -f "$output_file"
            return 0
        else
            local end_time=$(date +%s)
            local duration=$((end_time - start_time))
            log_error "✗ Test FAILED after ${duration}s: $test_name"
            log_error "Output:"
            cat "$output_file"
            rm -f "$output_file"
            FAILED_TESTS+=("$test_name")
            return 1
        fi
    fi
}

# Print summary
print_summary() {
    log_section "Test Summary"
    
    local total=$((${#PASSED_TESTS[@]} + ${#FAILED_TESTS[@]} + ${#SKIPPED_TESTS[@]}))
    
    echo ""
    echo -e "${BLUE}Total Tests:${NC} $total"
    echo -e "${GREEN}Passed:${NC}      ${#PASSED_TESTS[@]}"
    echo -e "${RED}Failed:${NC}      ${#FAILED_TESTS[@]}"
    echo -e "${YELLOW}Skipped:${NC}     ${#SKIPPED_TESTS[@]}"
    echo ""
    
    if [ ${#PASSED_TESTS[@]} -gt 0 ]; then
        echo -e "${GREEN}Passed Tests:${NC}"
        for test in "${PASSED_TESTS[@]}"; do
            echo "  ✓ $test"
        done
        echo ""
    fi
    
    if [ ${#FAILED_TESTS[@]} -gt 0 ]; then
        echo -e "${RED}Failed Tests:${NC}"
        for test in "${FAILED_TESTS[@]}"; do
            echo "  ✗ $test"
        done
        echo ""
    fi
    
    if [ ${#SKIPPED_TESTS[@]} -gt 0 ]; then
        echo -e "${YELLOW}Skipped Tests:${NC}"
        for test in "${SKIPPED_TESTS[@]}"; do
            echo "  - $test"
        done
        echo ""
    fi
    
    # Final status
    if [ ${#FAILED_TESTS[@]} -eq 0 ]; then
        log_section "✓ ALL TESTS PASSED"
        return 0
    else
        log_section "✗ SOME TESTS FAILED"
        return 1
    fi
}

# Main execution
main() {
    local start_time=$(date +%s)
    
    log_section "BOIFI Executor E2E Test Suite"
    echo "Started at: $(date)"
    echo ""
    
    # Check prerequisites
    check_prerequisites
    
    # Define test scripts in execution order
    local test_scripts=(
        "$SCRIPT_DIR/test-us1-control-plane.sh"
        "$SCRIPT_DIR/test-us2-wasmplugin.sh"
        "$SCRIPT_DIR/test-us3-service-targeting.sh"
        "$SCRIPT_DIR/test-us4-pod-identity.sh"
        "$SCRIPT_DIR/test-us5-multi-pod.sh"
        "$SCRIPT_DIR/test-us6-observability.sh"
        "$SCRIPT_DIR/test-metrics.sh"
    )
    
    # Run each test
    for test_script in "${test_scripts[@]}"; do
        if [ -f "$test_script" ]; then
            if ! run_test "$test_script"; then
                if [ "$STOP_ON_FAILURE" = true ]; then
                    log_error "Stopping execution due to test failure"
                    break
                fi
            fi
        else
            local test_name=$(basename "$test_script" .sh)
            log_warn "Test script not found: $test_script"
            SKIPPED_TESTS+=("$test_name")
        fi
        echo ""
    done
    
    # Print summary
    print_summary
    local exit_code=$?
    
    local end_time=$(date +%s)
    local total_duration=$((end_time - start_time))
    echo ""
    echo "Completed at: $(date)"
    echo "Total duration: ${total_duration}s"
    
    exit $exit_code
}

# Run main function
main "$@"
