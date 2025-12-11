#!/bin/bash
# ============================================================================
# BOIFI Validation Scripts - Common Functions Library
# ============================================================================
#
# This script provides shared functions for all validation scripts.
# Source this file at the beginning of your validation script:
#   source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
#
# ============================================================================

set -e

# ============================================================================
# Configuration (can be overridden by environment variables)
# ============================================================================

NAMESPACE="${NAMESPACE:-demo}"
CONTROL_PLANE_NS="${CONTROL_PLANE_NS:-boifi}"
TARGET_SERVICE="${TARGET_SERVICE:-frontend}"
PROPAGATION_WAIT="${PROPAGATION_WAIT:-35}"  # seconds to wait for policy propagation
REQUEST_COUNT="${REQUEST_COUNT:-10}"         # number of requests for testing

# ============================================================================
# Colors for Output
# ============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ============================================================================
# Logging Functions
# ============================================================================

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_test() {
    echo -e "\n${BLUE}[TEST]${NC} $1"
}

log_debug() {
    if [ "${DEBUG:-false}" = "true" ]; then
        echo -e "${CYAN}[DEBUG]${NC} $1"
    fi
}

# ============================================================================
# Test Result Tracking
# ============================================================================

TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

pass_test() {
    log_info "✅ PASS: $1"
    ((TESTS_PASSED++)) || true
}

fail_test() {
    log_error "❌ FAIL: $1"
    ((TESTS_FAILED++)) || true
}

skip_test() {
    log_warn "⏭️ SKIP: $1"
    ((TESTS_SKIPPED++)) || true
}

# Print test summary and return appropriate exit code
print_summary() {
    local title="${1:-TEST SUMMARY}"
    echo ""
    echo "========================================"
    echo "$title"
    echo "========================================"
    echo "Total:   $((TESTS_PASSED + TESTS_FAILED + TESTS_SKIPPED))"
    echo "Passed:  $TESTS_PASSED"
    echo "Failed:  $TESTS_FAILED"
    echo "Skipped: $TESTS_SKIPPED"
    echo "========================================"
    
    if [ "$TESTS_FAILED" -gt 0 ]; then
        return 1
    fi
    return 0
}

# ============================================================================
# Pre-flight Check Functions
# ============================================================================

# Check if kubectl is available
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        return 1
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        return 1
    fi
    
    log_info "✅ kubectl available and cluster connected"
    return 0
}

# Check if Control Plane is running
check_control_plane() {
    local cp_pod
    cp_pod=$(kubectl get pods -n "${CONTROL_PLANE_NS}" -l app=control-plane \
        --field-selector=status.phase=Running \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    
    if [ -z "$cp_pod" ]; then
        log_error "Control Plane not running in ${CONTROL_PLANE_NS} namespace"
        return 1
    fi
    
    log_info "✅ Control Plane running: $cp_pod"
    return 0
}

# Check if WasmPlugin is deployed
check_wasmplugin() {
    if ! kubectl get wasmplugin boifi-fault-injection -n "${NAMESPACE}" &>/dev/null; then
        log_error "WasmPlugin 'boifi-fault-injection' not found in ${NAMESPACE} namespace"
        return 1
    fi
    
    log_info "✅ WasmPlugin deployed in ${NAMESPACE}"
    return 0
}

# Check if a service exists
check_service_exists() {
    local service="$1"
    local ns="${2:-$NAMESPACE}"
    
    if ! kubectl get svc "$service" -n "$ns" &>/dev/null; then
        log_warn "Service '$service' not found in ${ns} namespace"
        return 1
    fi
    
    log_info "✅ Service '$service' exists in ${ns}"
    return 0
}

# Run all pre-flight checks
run_preflight_checks() {
    log_info "Running pre-flight checks..."
    
    check_kubectl || return 2
    check_control_plane || return 2
    check_wasmplugin || return 2
    
    log_info "All pre-flight checks passed"
    return 0
}

# ============================================================================
# Policy Management Functions
# ============================================================================

# Create a policy via Control Plane API
# Usage: create_policy "name" "service" "namespace" "abort|delay" percentage [delay_ms]
create_policy() {
    local name="$1"
    local selector_service="$2"
    local selector_namespace="$3"
    local fault_type="$4"
    local percentage="$5"
    local delay_ms="${6:-500}"
    
    log_info "Creating policy: $name (selector: $selector_service.$selector_namespace, $fault_type at $percentage%)"
    
    local policy_json=""
    if [ "$fault_type" == "abort" ]; then
        policy_json=$(cat <<EOF
{
    "metadata": {"name": "$name"},
    "spec": {
        "selector": {"service": "$selector_service", "namespace": "$selector_namespace"},
        "rules": [{
            "match": {"method": {"exact": "GET"}, "path": {"prefix": "/"}},
            "fault": {"percentage": $percentage, "abort": {"httpStatus": 503}}
        }]
    }
}
EOF
)
    else
        policy_json=$(cat <<EOF
{
    "metadata": {"name": "$name"},
    "spec": {
        "selector": {"service": "$selector_service", "namespace": "$selector_namespace"},
        "rules": [{
            "match": {"method": {"exact": "GET"}, "path": {"prefix": "/"}},
            "fault": {"percentage": $percentage, "delay": {"fixed_delay_ms": $delay_ms}}
        }]
    }
}
EOF
)
    fi
    
    # Use kubectl exec to reach control plane
    local result
    result=$(kubectl exec -n "${CONTROL_PLANE_NS}" deploy/hfi-control-plane -c control-plane -- \
        curl -s -X POST "http://localhost:8080/v1/policies" \
        -H "Content-Type: application/json" \
        -d "$policy_json" 2>&1) || {
        log_error "Failed to create policy: $result"
        return 1
    }
    
    log_debug "Policy creation response: $result"
    return 0
}

# Delete a policy via Control Plane API
delete_policy() {
    local name="$1"
    log_info "Deleting policy: $name"
    
    kubectl exec -n "${CONTROL_PLANE_NS}" deploy/hfi-control-plane -c control-plane -- \
        curl -s -X DELETE "http://localhost:8080/v1/policies/$name" &>/dev/null || true
}

# Wait for policy to propagate to Wasm plugins
wait_for_propagation() {
    local wait_time="${1:-$PROPAGATION_WAIT}"
    log_info "Waiting ${wait_time}s for policy propagation..."
    sleep "$wait_time"
}

# ============================================================================
# Request Testing Functions
# ============================================================================

# Find a test pod in the namespace
find_test_pod() {
    local ns="${1:-$NAMESPACE}"
    local label="${2:-app=frontend}"
    
    local pod
    pod=$(kubectl get pods -n "$ns" -l "$label" \
        --field-selector=status.phase=Running \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    
    if [ -z "$pod" ]; then
        # Try to find any running pod
        pod=$(kubectl get pods -n "$ns" \
            --field-selector=status.phase=Running \
            -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    fi
    
    echo "$pod"
}

# Send requests to a service and count responses
# Returns: "success_count:error_count:total_latency_ms"
send_requests() {
    local service="$1"
    local count="${2:-$REQUEST_COUNT}"
    local test_pod="${3:-$(find_test_pod)}"
    
    if [ -z "$test_pod" ]; then
        log_error "No test pod found for sending requests"
        echo "0:0:0"
        return 1
    fi
    
    local success=0
    local errors=0
    local total_latency=0
    
    log_debug "Sending $count requests to $service from pod $test_pod"
    
    for i in $(seq 1 "$count"); do
        local start_time
        local end_time
        local status
        
        start_time=$(date +%s%N)
        
        status=$(kubectl exec -n "${NAMESPACE}" "$test_pod" -c istio-proxy -- \
            curl -s -o /dev/null -w "%{http_code}" \
            "http://${service}.${NAMESPACE}.svc.cluster.local/" 2>/dev/null || echo "000")
        
        end_time=$(date +%s%N)
        
        local latency_ms=$(( (end_time - start_time) / 1000000 ))
        total_latency=$((total_latency + latency_ms))
        
        if [ "$status" == "503" ] || [ "$status" == "000" ]; then
            ((errors++)) || true
        else
            ((success++)) || true
        fi
        
        log_debug "Request $i: status=$status, latency=${latency_ms}ms"
    done
    
    echo "${success}:${errors}:${total_latency}"
}

# Calculate error rate percentage
calc_error_rate() {
    local result="$1"
    local success
    local errors
    
    success=$(echo "$result" | cut -d: -f1)
    errors=$(echo "$result" | cut -d: -f2)
    
    local total=$((success + errors))
    if [ "$total" -eq 0 ]; then
        echo "0"
        return
    fi
    
    echo $((errors * 100 / total))
}

# Calculate average latency in milliseconds
calc_avg_latency() {
    local result="$1"
    local count="${2:-$REQUEST_COUNT}"
    local total_latency
    
    total_latency=$(echo "$result" | cut -d: -f3)
    
    if [ "$count" -eq 0 ]; then
        echo "0"
        return
    fi
    
    echo $((total_latency / count))
}

# ============================================================================
# Cleanup Functions
# ============================================================================

# Cleanup function to be called on exit
cleanup() {
    log_info "Cleaning up test policies..."
    # Add policy names to clean up here
    for policy in "${CLEANUP_POLICIES[@]:-}"; do
        delete_policy "$policy" 2>/dev/null || true
    done
}

# Register cleanup on script exit
register_cleanup() {
    trap cleanup EXIT
}

# Add a policy to the cleanup list
add_cleanup_policy() {
    CLEANUP_POLICIES+=("$1")
}

# Initialize cleanup array
CLEANUP_POLICIES=()

# ============================================================================
# Utility Functions
# ============================================================================

# Check if a value is within tolerance of expected
# Usage: within_tolerance actual expected tolerance_percent
within_tolerance() {
    local actual="$1"
    local expected="$2"
    local tolerance="${3:-10}"
    
    local lower=$((expected - tolerance))
    local upper=$((expected + tolerance))
    
    if [ "$actual" -ge "$lower" ] && [ "$actual" -le "$upper" ]; then
        return 0
    fi
    return 1
}

# Print script usage
print_usage() {
    local script_name="${1:-$(basename "$0")}"
    echo "Usage: $script_name [OPTIONS]"
    echo ""
    echo "Environment Variables:"
    echo "  NAMESPACE          Target namespace (default: demo)"
    echo "  CONTROL_PLANE_NS   Control plane namespace (default: boifi)"
    echo "  TARGET_SERVICE     Target service for testing (default: frontend)"
    echo "  PROPAGATION_WAIT   Seconds to wait for propagation (default: 35)"
    echo "  REQUEST_COUNT      Number of requests per test (default: 10)"
    echo "  DEBUG              Enable debug output (default: false)"
    echo ""
    echo "Exit Codes:"
    echo "  0  All tests passed"
    echo "  1  One or more tests failed"
    echo "  2  Pre-flight checks failed"
}
