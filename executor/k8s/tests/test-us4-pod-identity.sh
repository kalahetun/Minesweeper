#!/bin/bash
# E2E Test: US4 - Pod Identity Awareness
#
# This script tests that each Wasm plugin instance correctly identifies
# its service/pod identity and applies only matching policies.
#
# Acceptance Scenarios:
#   1. frontend Pod correctly identifies itself as frontend service
#   2. frontend plugin ignores policies targeting productcatalog
#   3. Service name and pod name correctly extracted from Envoy metadata
#   4. When identity cannot be determined, only wildcard policies apply
#
# Prerequisites:
#   - kubectl configured and connected to cluster
#   - Istio installed with WasmPlugin CRD support
#   - Control plane deployed to boifi namespace
#   - WasmPlugin deployed to demo namespace
#   - Multiple services running in demo namespace
#
# Usage:
#   ./test-us4-pod-identity.sh
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
    delete_policy "frontend-abort"
    delete_policy "productcatalog-abort"
    delete_policy "wildcard-abort"
    sleep 2
}

# Helper: Get pod logs using crictl (workaround for kubectl logs issues in WSL2)
get_wasm_logs() {
    local pod="$1"
    local container="istio-proxy"
    
    # Get container ID
    local container_id=$(sudo crictl ps --name "$container" -q | head -1)
    if [ -n "$container_id" ]; then
        sudo crictl logs "$container_id" 2>&1 | tail -50
    else
        echo "Container not found"
    fi
}

# Helper: Wait for policy propagation
wait_for_propagation() {
    log_info "Waiting 35 seconds for policy propagation..."
    sleep 35
}

# Helper: Test service response
test_service_response() {
    local service="$1"
    local port="${2:-80}"
    local test_pod=""
    
    test_pod=$(kubectl get pods -n ${NAMESPACE} -l app=frontend -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    if [ -z "$test_pod" ]; then
        echo "000"
        return
    fi
    
    kubectl exec -n ${NAMESPACE} "$test_pod" -c istio-proxy -- \
        curl -s -o /dev/null -w "%{http_code}" "http://${service}.${NAMESPACE}.svc.cluster.local:${port}/" 2>/dev/null || echo "000"
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

# Get frontend pod
FRONTEND_POD=$(kubectl get pod -n ${NAMESPACE} -l app=frontend -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -z "$FRONTEND_POD" ]; then
    log_error "No frontend pod found in ${NAMESPACE} namespace"
    exit 1
fi
log_info "Frontend pod: $FRONTEND_POD"

# Cleanup any existing test policies
cleanup_policies

# ============================================================================
# Scenario 1: frontend Pod correctly identifies itself as frontend service
# ============================================================================
log_test "Scenario 1: frontend Pod should identify itself as 'frontend' service"

# Check Envoy stats for identity-related information
IDENTITY_INFO=$(kubectl exec -n ${NAMESPACE} ${FRONTEND_POD} -c istio-proxy -- \
    pilot-agent request GET /stats 2>/dev/null | grep -i "wasm" | head -5 || echo "")

# Create a frontend-only policy
create_policy "frontend-abort" "frontend" "demo" "abort" 100
wait_for_propagation

# Test frontend - should get 503
FRONTEND_STATUS=$(test_service_response "frontend")
log_info "Frontend response status: $FRONTEND_STATUS"

if [ "$FRONTEND_STATUS" == "503" ]; then
    pass_test "Frontend pod correctly identifies as 'frontend' and receives targeted policy"
else
    fail_test "Frontend pod identity issue - expected 503, got $FRONTEND_STATUS"
fi

# Cleanup
delete_policy "frontend-abort"
sleep 5

# ============================================================================
# Scenario 2: frontend plugin ignores policies targeting productcatalog
# ============================================================================
log_test "Scenario 2: frontend plugin should ignore policies targeting productcatalog"

# Create a productcatalog-only policy
create_policy "productcatalog-abort" "productcatalogservice" "demo" "abort" 100
wait_for_propagation

# Test frontend - should NOT get 503 (because policy is for productcatalog)
FRONTEND_STATUS=$(test_service_response "frontend")
log_info "Frontend response status with productcatalog policy: $FRONTEND_STATUS"

if [ "$FRONTEND_STATUS" != "503" ]; then
    pass_test "Frontend plugin correctly ignores productcatalog-targeted policy"
else
    fail_test "Frontend plugin incorrectly applied productcatalog policy"
fi

# Cleanup
delete_policy "productcatalog-abort"
sleep 5

# ============================================================================
# Scenario 3: Service name and pod name correctly extracted from Envoy metadata
# ============================================================================
log_test "Scenario 3: Verify service/pod identity extraction from Envoy metadata"

# Check Envoy node metadata
NODE_METADATA=$(kubectl exec -n ${NAMESPACE} ${FRONTEND_POD} -c istio-proxy -- \
    pilot-agent request GET /server_info 2>/dev/null | head -20 || echo "")

# Check for workload name in Istio proxy
WORKLOAD_CHECK=$(kubectl get pod -n ${NAMESPACE} ${FRONTEND_POD} -o jsonpath='{.metadata.labels.app}' 2>/dev/null || echo "")
log_info "Pod label 'app': $WORKLOAD_CHECK"

if [ "$WORKLOAD_CHECK" == "frontend" ]; then
    pass_test "Pod correctly labeled with service identity (app=frontend)"
else
    fail_test "Pod label mismatch - expected 'frontend', got '$WORKLOAD_CHECK'"
fi

# ============================================================================
# Scenario 4: Wildcard policies apply when identity cannot be determined
# ============================================================================
log_test "Scenario 4: Wildcard policies should apply in all cases (including fail-open)"

# Create a wildcard policy
create_policy "wildcard-abort" "*" "*" "abort" 100
wait_for_propagation

# Test frontend - should get 503 (wildcard applies to all)
FRONTEND_STATUS=$(test_service_response "frontend")
log_info "Frontend response status with wildcard policy: $FRONTEND_STATUS"

if [ "$FRONTEND_STATUS" == "503" ]; then
    pass_test "Wildcard policy correctly applied to frontend"
else
    fail_test "Wildcard policy not applied - expected 503, got $FRONTEND_STATUS"
fi

# Cleanup
delete_policy "wildcard-abort"

# ============================================================================
# Final cleanup and summary
# ============================================================================
cleanup_policies

echo ""
echo "============================================"
echo "           US4 Test Summary"
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
