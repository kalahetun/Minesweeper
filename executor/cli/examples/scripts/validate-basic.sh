#!/bin/bash
# ============================================================================
# BOIFI Basic Fault Injection Validation Script
# ============================================================================
#
# Purpose: Validate basic abort and delay fault injection functionality
#
# This script tests:
#   1. Abort fault injection (503 response)
#   2. Delay fault injection (latency increase)
#
# Prerequisites:
#   - kubectl configured and connected to cluster
#   - Control Plane deployed to boifi namespace
#   - WasmPlugin deployed to target namespace
#   - At least one service running in target namespace
#
# Usage:
#   ./validate-basic.sh
#
# Environment Variables:
#   NAMESPACE          Target namespace (default: demo)
#   CONTROL_PLANE_NS   Control plane namespace (default: boifi)
#   TARGET_SERVICE     Target service for testing (default: frontend)
#   PROPAGATION_WAIT   Seconds to wait for propagation (default: 35)
#   REQUEST_COUNT      Number of requests per test (default: 10)
#   DEBUG              Enable debug output (default: false)
#
# Exit Codes:
#   0  All tests passed
#   1  One or more tests failed
#   2  Pre-flight checks failed
#
# ============================================================================

set -e

# Get script directory and source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

# ============================================================================
# Script Configuration
# ============================================================================

# Policy names for this test
ABORT_POLICY_NAME="validate-basic-abort"
DELAY_POLICY_NAME="validate-basic-delay"

# Expected values
EXPECTED_ABORT_RATE=100    # 100% abort rate
EXPECTED_MIN_LATENCY=450   # Minimum latency in ms for delay test (500ms - tolerance)
DELAY_VALUE=500            # Delay value in ms

# ============================================================================
# Test Functions
# ============================================================================

test_abort_fault() {
    log_test "Abort Fault Injection (503 Response)"
    
    # Create abort policy
    add_cleanup_policy "$ABORT_POLICY_NAME"
    if ! create_policy "$ABORT_POLICY_NAME" "$TARGET_SERVICE" "$NAMESPACE" "abort" 100; then
        fail_test "Failed to create abort policy"
        return 1
    fi
    
    # Wait for propagation
    wait_for_propagation
    
    # Send requests and measure
    log_info "Sending ${REQUEST_COUNT} requests to ${TARGET_SERVICE}..."
    local result
    result=$(send_requests "$TARGET_SERVICE" "$REQUEST_COUNT")
    
    local error_rate
    error_rate=$(calc_error_rate "$result")
    
    log_info "Error rate: ${error_rate}% (expected: ~${EXPECTED_ABORT_RATE}%)"
    
    # Cleanup policy before checking result
    delete_policy "$ABORT_POLICY_NAME"
    
    # Check result
    if within_tolerance "$error_rate" "$EXPECTED_ABORT_RATE" 15; then
        pass_test "Abort fault injection working correctly (${error_rate}% errors)"
        return 0
    else
        fail_test "Abort fault injection: expected ~${EXPECTED_ABORT_RATE}% errors, got ${error_rate}%"
        return 1
    fi
}

test_delay_fault() {
    log_test "Delay Fault Injection (${DELAY_VALUE}ms Latency)"
    
    # First, measure baseline latency without any policy
    log_info "Measuring baseline latency..."
    local baseline_result
    baseline_result=$(send_requests "$TARGET_SERVICE" 5)
    local baseline_latency
    baseline_latency=$(calc_avg_latency "$baseline_result" 5)
    log_info "Baseline average latency: ${baseline_latency}ms"
    
    # Create delay policy
    add_cleanup_policy "$DELAY_POLICY_NAME"
    if ! create_policy "$DELAY_POLICY_NAME" "$TARGET_SERVICE" "$NAMESPACE" "delay" 100 "$DELAY_VALUE"; then
        fail_test "Failed to create delay policy"
        return 1
    fi
    
    # Wait for propagation
    wait_for_propagation
    
    # Send requests and measure
    log_info "Sending ${REQUEST_COUNT} requests to ${TARGET_SERVICE}..."
    local result
    result=$(send_requests "$TARGET_SERVICE" "$REQUEST_COUNT")
    
    local avg_latency
    avg_latency=$(calc_avg_latency "$result" "$REQUEST_COUNT")
    
    log_info "Average latency with delay: ${avg_latency}ms (expected: >= ${EXPECTED_MIN_LATENCY}ms)"
    
    # Cleanup policy before checking result
    delete_policy "$DELAY_POLICY_NAME"
    
    # Check result - latency should be at least baseline + delay - tolerance
    local expected_min=$((baseline_latency + EXPECTED_MIN_LATENCY))
    if [ "$avg_latency" -ge "$EXPECTED_MIN_LATENCY" ]; then
        pass_test "Delay fault injection working correctly (avg ${avg_latency}ms)"
        return 0
    else
        fail_test "Delay fault injection: expected >= ${EXPECTED_MIN_LATENCY}ms, got ${avg_latency}ms"
        return 1
    fi
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
    echo "============================================"
    echo "BOIFI Basic Fault Injection Validation"
    echo "============================================"
    echo ""
    echo "Configuration:"
    echo "  Namespace:        ${NAMESPACE}"
    echo "  Control Plane NS: ${CONTROL_PLANE_NS}"
    echo "  Target Service:   ${TARGET_SERVICE}"
    echo "  Propagation Wait: ${PROPAGATION_WAIT}s"
    echo "  Request Count:    ${REQUEST_COUNT}"
    echo ""
    
    # Register cleanup handler
    register_cleanup
    
    # Run pre-flight checks
    if ! run_preflight_checks; then
        log_error "Pre-flight checks failed"
        exit 2
    fi
    
    # Check target service exists
    if ! check_service_exists "$TARGET_SERVICE"; then
        log_error "Target service '$TARGET_SERVICE' not found"
        exit 2
    fi
    
    echo ""
    log_info "Starting validation tests..."
    echo ""
    
    # Run tests
    test_abort_fault || true
    echo ""
    test_delay_fault || true
    
    # Print summary and exit with appropriate code
    echo ""
    if print_summary "BASIC VALIDATION SUMMARY"; then
        log_info "🎉 All basic validation tests passed!"
        exit 0
    else
        log_error "Some tests failed. Check the output above for details."
        exit 1
    fi
}

# Run main function
main "$@"
