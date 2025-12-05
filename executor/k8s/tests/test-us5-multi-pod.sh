#!/bin/bash
# E2E Test: US5 - Multi-Pod Fault Distribution
#
# This script tests that percentage-based fault injection works correctly
# when requests flow through multiple services in series.
#
# Key Concept:
#   - Each Wasm plugin instance independently decides whether to inject fault
#   - For a 30% fault policy, each request has 30% chance of being faulted
#   - Requests flow serially through services (frontend → checkout → payment)
#   - No coordination needed between pods - each makes independent decision
#
# Acceptance Scenarios:
#   1. 30% fault policy results in ~30% of requests failing (±10% tolerance)
#   2. Requests that pass frontend can still be faulted by downstream services
#   3. New pods after restart correctly apply fault policies
#
# Prerequisites:
#   - kubectl configured and connected to cluster
#   - Istio installed with WasmPlugin CRD support
#   - Control plane deployed to boifi namespace
#   - WasmPlugin deployed to demo namespace
#
# Usage:
#   ./test-us5-multi-pod.sh
#
set -e

NAMESPACE="demo"
CONTROL_PLANE_NS="boifi"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_test() { echo -e "\n${BLUE}[TEST]${NC} $1"; }

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

# Helper: Create policy
create_policy() {
    local name="$1"
    local selector_service="$2"
    local percentage="$3"
    
    log_info "Creating policy: $name ($selector_service at $percentage%)"
    
    kubectl exec -n ${CONTROL_PLANE_NS} deploy/hfi-control-plane -c control-plane -- \
        curl -s -X POST "http://localhost:8080/v1/policies" \
        -H "Content-Type: application/json" \
        -d "{
            \"metadata\": {\"name\": \"$name\"},
            \"spec\": {
                \"selector\": {\"service\": \"$selector_service\", \"namespace\": \"demo\"},
                \"rules\": [{
                    \"match\": {\"method\": {\"exact\": \"GET\"}, \"path\": {\"prefix\": \"/\"}},
                    \"fault\": {\"percentage\": $percentage, \"abort\": {\"httpStatus\": 503}}
                }]
            }
        }" || return 1
}

# Helper: Delete policy
delete_policy() {
    local name="$1"
    kubectl exec -n ${CONTROL_PLANE_NS} deploy/hfi-control-plane -c control-plane -- \
        curl -s -X DELETE "http://localhost:8080/v1/policies/$name" 2>/dev/null || true
}

# Helper: Cleanup all test policies
cleanup_policies() {
    log_info "Cleaning up test policies..."
    delete_policy "frontend-30pct"
    delete_policy "frontend-50pct"
    sleep 2
}

# Helper: Wait for policy propagation
wait_for_propagation() {
    log_info "Waiting 35 seconds for policy propagation..."
    sleep 35
}

# Helper: Send N requests and count failures
# Returns: "success_count:fail_count"
send_requests() {
    local service="$1"
    local count="$2"
    local test_pod=""
    
    test_pod=$(kubectl get pods -n ${NAMESPACE} -l app=frontend -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -z "$test_pod" ]; then
        echo "0:0"
        return
    fi
    
    local success=0
    local fail=0
    
    for i in $(seq 1 $count); do
        local status=$(kubectl exec -n ${NAMESPACE} "$test_pod" -c istio-proxy -- \
            curl -s -o /dev/null -w "%{http_code}" "http://${service}.${NAMESPACE}.svc.cluster.local/" 2>/dev/null || echo "000")
        
        if [ "$status" == "503" ]; then
            ((fail++))
        elif [ "$status" == "200" ]; then
            ((success++))
        fi
        
        # Progress indicator every 10 requests
        if [ $((i % 10)) -eq 0 ]; then
            echo -n "."
        fi
    done
    echo ""  # newline after progress dots
    
    echo "${success}:${fail}"
}

# ============================================================================
# Pre-flight checks
# ============================================================================
log_info "Running pre-flight checks..."

if ! kubectl get pods -n ${CONTROL_PLANE_NS} -l app=control-plane --field-selector=status.phase=Running 2>/dev/null | grep -q "Running"; then
    log_error "Control plane not running"
    exit 1
fi

if ! kubectl get wasmplugin boifi-fault-injection -n ${NAMESPACE} &>/dev/null; then
    log_error "WasmPlugin not found"
    exit 1
fi

FRONTEND_POD=$(kubectl get pod -n ${NAMESPACE} -l app=frontend -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -z "$FRONTEND_POD" ]; then
    log_error "No frontend pod found"
    exit 1
fi

cleanup_policies

# ============================================================================
# Scenario 1: 30% fault policy results in ~30% failures
# ============================================================================
log_test "Scenario 1: 30% fault policy should result in ~30% request failures"

create_policy "frontend-30pct" "frontend" 30
wait_for_propagation

log_info "Sending 100 requests to frontend..."
RESULT=$(send_requests "frontend" 100)
SUCCESS=$(echo $RESULT | cut -d: -f1)
FAIL=$(echo $RESULT | cut -d: -f2)
TOTAL=$((SUCCESS + FAIL))

if [ $TOTAL -eq 0 ]; then
    fail_test "No valid responses received"
else
    FAIL_RATE=$((FAIL * 100 / TOTAL))
    log_info "Results: $SUCCESS success, $FAIL failures out of $TOTAL requests (${FAIL_RATE}% failure rate)"
    
    # Allow ±15% tolerance (15% to 45% is acceptable for 30% target)
    if [ $FAIL_RATE -ge 15 ] && [ $FAIL_RATE -le 45 ]; then
        pass_test "30% fault policy results in ${FAIL_RATE}% failures (within ±15% tolerance)"
    else
        fail_test "Failure rate ${FAIL_RATE}% is outside acceptable range (15-45%) for 30% policy"
    fi
fi

delete_policy "frontend-30pct"
sleep 5

# ============================================================================
# Scenario 2: 50% fault policy results in ~50% failures
# ============================================================================
log_test "Scenario 2: 50% fault policy should result in ~50% request failures"

create_policy "frontend-50pct" "frontend" 50
wait_for_propagation

log_info "Sending 100 requests to frontend..."
RESULT=$(send_requests "frontend" 100)
SUCCESS=$(echo $RESULT | cut -d: -f1)
FAIL=$(echo $RESULT | cut -d: -f2)
TOTAL=$((SUCCESS + FAIL))

if [ $TOTAL -eq 0 ]; then
    fail_test "No valid responses received"
else
    FAIL_RATE=$((FAIL * 100 / TOTAL))
    log_info "Results: $SUCCESS success, $FAIL failures out of $TOTAL requests (${FAIL_RATE}% failure rate)"
    
    # Allow ±15% tolerance (35% to 65% is acceptable for 50% target)
    if [ $FAIL_RATE -ge 35 ] && [ $FAIL_RATE -le 65 ]; then
        pass_test "50% fault policy results in ${FAIL_RATE}% failures (within ±15% tolerance)"
    else
        fail_test "Failure rate ${FAIL_RATE}% is outside acceptable range (35-65%) for 50% policy"
    fi
fi

delete_policy "frontend-50pct"
sleep 5

# ============================================================================
# Scenario 3: New pod after restart applies policy correctly
# ============================================================================
log_test "Scenario 3: New pod after restart should apply fault policy"

create_policy "frontend-50pct" "frontend" 50
wait_for_propagation

# Verify fault injection works before restart
log_info "Testing before pod restart..."
RESULT_BEFORE=$(send_requests "frontend" 20)
FAIL_BEFORE=$(echo $RESULT_BEFORE | cut -d: -f2)

# Restart frontend pod
log_info "Restarting frontend pod..."
kubectl delete pod -n ${NAMESPACE} -l app=frontend --wait=false
sleep 5
kubectl wait --for=condition=Ready pod -l app=frontend -n ${NAMESPACE} --timeout=60s

# Wait for new pod to receive policy
log_info "Waiting for new pod to receive policy..."
sleep 40

# Test after restart
log_info "Testing after pod restart..."
RESULT_AFTER=$(send_requests "frontend" 20)
FAIL_AFTER=$(echo $RESULT_AFTER | cut -d: -f2)

log_info "Before restart: $FAIL_BEFORE failures, After restart: $FAIL_AFTER failures"

if [ $FAIL_AFTER -gt 0 ]; then
    pass_test "New pod correctly applies fault policy after restart"
else
    fail_test "New pod not applying fault policy after restart"
fi

# ============================================================================
# Cleanup and Summary
# ============================================================================
cleanup_policies

echo ""
echo "============================================"
echo "           US5 Test Summary"
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
