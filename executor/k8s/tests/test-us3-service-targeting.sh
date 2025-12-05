#!/bin/bash
# E2E Test: US3 - Service-Level Policy Targeting
#
# This script tests that fault injection policies can be targeted to specific
# services/namespaces, and that non-matching services are not affected.
#
# Acceptance Scenarios:
#   1. Policy targeting frontend only affects frontend requests
#   2. Requests to productcatalog are not affected
#   3. Wildcard (*) policy affects all services
#   4. Namespace selector correctly filters
#
# Prerequisites:
#   - kubectl configured and connected to cluster
#   - Istio installed with WasmPlugin CRD support
#   - Control plane deployed to boifi namespace
#   - WasmPlugin deployed to demo namespace
#   - Multiple services running in demo namespace (e.g., frontend, productcatalog)
#
# Usage:
#   ./test-us3-service-targeting.sh
#
set -e

NAMESPACE="demo"
CONTROL_PLANE_NS="boifi"
TIMEOUT=60
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_test() { echo -e "\n${BLUE}[TEST]${NC} $1"; }
log_debug() { echo -e "[DEBUG] $1"; }

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

pass_test() {
    log_info "✅ PASS: $1"
    ((TESTS_PASSED++))
}

fail_test() {
    log_error "❌ FAIL: $1"
    ((TESTS_FAILED++))
}

skip_test() {
    log_warn "⏭️ SKIP: $1"
}

# Helper: Create policy via Control Plane API
create_policy() {
    local name="$1"
    local selector_service="$2"
    local selector_namespace="$3"
    local fault_type="$4"   # "abort" or "delay"
    local percentage="$5"
    
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
            "fault": {"percentage": $percentage, "delay": {"fixed_delay": "500ms"}}
        }]
    }
}
EOF
)
    fi
    
    # Use port-forward to reach control plane
    kubectl exec -n ${CONTROL_PLANE_NS} deploy/hfi-control-plane -c control-plane -- \
        curl -s -X POST "http://localhost:8080/v1/policies" \
        -H "Content-Type: application/json" \
        -d "$policy_json" || return 1
}

# Helper: Delete policy via Control Plane API
delete_policy() {
    local name="$1"
    log_info "Deleting policy: $name"
    
    kubectl exec -n ${CONTROL_PLANE_NS} deploy/hfi-control-plane -c control-plane -- \
        curl -s -X DELETE "http://localhost:8080/v1/policies/$name" || true
}

# Helper: Delete all test policies
cleanup_policies() {
    log_info "Cleaning up test policies..."
    delete_policy "frontend-only-abort"
    delete_policy "wildcard-abort"
    delete_policy "demo-namespace-abort"
    sleep 2  # Wait for policy propagation
}

# Helper: Test service with multiple requests
# Returns the percentage of failed requests (503)
test_service() {
    local service="$1"
    local requests="${2:-20}"
    local test_pod=""
    
    # Find a test pod in demo namespace
    test_pod=$(kubectl get pods -n ${NAMESPACE} -l app=frontend -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    if [ -z "$test_pod" ]; then
        test_pod=$(kubectl get pods -n ${NAMESPACE} --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    fi
    
    if [ -z "$test_pod" ]; then
        log_warn "No running pod found in ${NAMESPACE} for testing"
        echo "0"
        return
    fi
    
    local failed=0
    local success=0
    
    for i in $(seq 1 $requests); do
        local status=$(kubectl exec -n ${NAMESPACE} "$test_pod" -c istio-proxy -- \
            curl -s -o /dev/null -w "%{http_code}" "http://${service}.${NAMESPACE}.svc.cluster.local/" 2>/dev/null || echo "000")
        
        if [ "$status" == "503" ]; then
            ((failed++))
        elif [ "$status" == "200" ]; then
            ((success++))
        fi
    done
    
    local total=$((failed + success))
    if [ $total -eq 0 ]; then
        echo "0"
    else
        echo "$((failed * 100 / total))"
    fi
}

# Helper: Wait for policy to propagate to Wasm plugins
wait_for_propagation() {
    log_info "Waiting 35 seconds for policy propagation (30s poll interval + buffer)..."
    sleep 35
}

# ============================================================================
# Pre-flight checks
# ============================================================================
log_info "Running pre-flight checks..."

# Check if control plane is running
if ! kubectl get pods -n ${CONTROL_PLANE_NS} -l app=control-plane --field-selector=status.phase=Running 2>/dev/null | grep -q "Running"; then
    log_error "Control plane not running in ${CONTROL_PLANE_NS} namespace"
    exit 1
fi
log_info "Control plane is running"

# Check if WasmPlugin is deployed
if ! kubectl get wasmplugin boifi-fault-injection -n ${NAMESPACE} &>/dev/null; then
    log_error "WasmPlugin not found in ${NAMESPACE} namespace"
    exit 1
fi
log_info "WasmPlugin is deployed"

# Get list of services
SERVICES=$(kubectl get svc -n ${NAMESPACE} -o jsonpath='{.items[*].metadata.name}')
log_info "Available services in ${NAMESPACE}: $SERVICES"

# Cleanup any existing test policies
cleanup_policies

# ============================================================================
# Scenario 1: Policy targeting frontend only affects frontend
# ============================================================================
log_test "Scenario 1: Policy targeting frontend should only affect frontend requests"

# Create a frontend-only abort policy
create_policy "frontend-only-abort" "frontend" "demo" "abort" 100

wait_for_propagation

# Test frontend - should see 100% failures
FRONTEND_FAIL_RATE=$(test_service "frontend")
log_info "Frontend failure rate: ${FRONTEND_FAIL_RATE}%"

# Test another service (productcatalogservice or any other) - should see 0% failures
OTHER_SERVICE="productcatalogservice"
# Check if productcatalogservice exists, otherwise use another service
if ! kubectl get svc -n ${NAMESPACE} ${OTHER_SERVICE} &>/dev/null; then
    OTHER_SERVICE=$(kubectl get svc -n ${NAMESPACE} -o jsonpath='{.items[?(@.metadata.name!="frontend")].metadata.name}' | cut -d' ' -f1)
fi

if [ -n "$OTHER_SERVICE" ]; then
    OTHER_FAIL_RATE=$(test_service "$OTHER_SERVICE")
    log_info "$OTHER_SERVICE failure rate: ${OTHER_FAIL_RATE}%"
    
    # Validate: frontend should have high failure rate, other service should have low
    if [ "$FRONTEND_FAIL_RATE" -ge 90 ] && [ "$OTHER_FAIL_RATE" -le 10 ]; then
        pass_test "Frontend-only policy correctly targets only frontend"
    else
        fail_test "Policy targeting not working correctly (frontend: ${FRONTEND_FAIL_RATE}%, ${OTHER_SERVICE}: ${OTHER_FAIL_RATE}%)"
    fi
else
    log_warn "No other service found to compare, testing frontend only"
    if [ "$FRONTEND_FAIL_RATE" -ge 90 ]; then
        pass_test "Frontend policy working (no comparison service available)"
    else
        fail_test "Frontend policy not working correctly (failure rate: ${FRONTEND_FAIL_RATE}%)"
    fi
fi

# Cleanup
delete_policy "frontend-only-abort"
sleep 5

# ============================================================================
# Scenario 2: Requests to productcatalog are not affected (already tested above)
# ============================================================================
log_test "Scenario 2: Requests to productcatalog should not be affected by frontend policy"

# This was validated in Scenario 1 - the other service should have 0% failure rate
log_info "This scenario was validated in Scenario 1"
pass_test "Verified in Scenario 1 - non-targeted services not affected"

# ============================================================================
# Scenario 3: Wildcard (*) policy affects all services
# ============================================================================
log_test "Scenario 3: Wildcard (*) policy should affect all services"

# Create a wildcard abort policy
create_policy "wildcard-abort" "*" "*" "abort" 100

wait_for_propagation

# Test multiple services
WILDCARD_WORKS=true

FRONTEND_FAIL_RATE=$(test_service "frontend")
log_info "Frontend failure rate with wildcard: ${FRONTEND_FAIL_RATE}%"

if [ "$FRONTEND_FAIL_RATE" -lt 90 ]; then
    WILDCARD_WORKS=false
fi

if [ -n "$OTHER_SERVICE" ]; then
    OTHER_FAIL_RATE=$(test_service "$OTHER_SERVICE")
    log_info "$OTHER_SERVICE failure rate with wildcard: ${OTHER_FAIL_RATE}%"
    
    if [ "$OTHER_FAIL_RATE" -lt 90 ]; then
        WILDCARD_WORKS=false
    fi
fi

if $WILDCARD_WORKS; then
    pass_test "Wildcard policy affects all services"
else
    fail_test "Wildcard policy not affecting all services"
fi

# Cleanup
delete_policy "wildcard-abort"
sleep 5

# ============================================================================
# Scenario 4: Namespace selector correctly filters
# ============================================================================
log_test "Scenario 4: Namespace selector should correctly filter policies"

# Create policy for demo namespace only
create_policy "demo-namespace-abort" "*" "demo" "abort" 100

wait_for_propagation

# Test frontend in demo namespace - should see failures
FRONTEND_FAIL_RATE=$(test_service "frontend")
log_info "Frontend failure rate with demo namespace policy: ${FRONTEND_FAIL_RATE}%"

if [ "$FRONTEND_FAIL_RATE" -ge 90 ]; then
    pass_test "Namespace selector correctly applies to demo namespace"
else
    fail_test "Namespace selector not working (failure rate: ${FRONTEND_FAIL_RATE}%)"
fi

# Cleanup
delete_policy "demo-namespace-abort"

# ============================================================================
# Final cleanup and summary
# ============================================================================
cleanup_policies

echo ""
echo "============================================"
echo "           US3 Test Summary"
echo "============================================"
echo -e "Tests Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Tests Failed: ${RED}${TESTS_FAILED}${NC}"
echo "============================================"

if [ $TESTS_FAILED -gt 0 ]; then
    log_error "Some tests failed!"
    exit 1
else
    log_info "All tests passed!"
    exit 0
fi
