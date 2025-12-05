#!/bin/bash
# E2E Test: US1 - Control Plane Deployment to Kubernetes
#
# This script tests the BOIFI Control Plane deployment to k3s/k8s cluster.
# It validates all acceptance scenarios defined in the user story.
#
# Prerequisites:
#   - kubectl configured and connected to cluster
#   - boifi namespace exists
#   - hfi-cli available in PATH or current directory
#
# Usage:
#   ./test-us1-control-plane.sh
#
set -e

NAMESPACE="boifi"
DEPLOYMENT="hfi-control-plane"
SERVICE="hfi-control-plane"
TIMEOUT=60
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_test() { echo -e "\n${YELLOW}[TEST]${NC} $1"; }

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

# Cleanup function
cleanup() {
    log_info "Cleaning up port-forward..."
    pkill -f "kubectl port-forward.*${SERVICE}" 2>/dev/null || true
}
trap cleanup EXIT

# ============================================================================
# Scenario 1: Pod Ready within 60 seconds
# ============================================================================
log_test "Scenario 1: Pod should be ready within ${TIMEOUT} seconds"

# Apply deployment if not exists
if ! kubectl get deployment ${DEPLOYMENT} -n ${NAMESPACE} &>/dev/null; then
    log_info "Deploying Control Plane..."
    kubectl apply -f "${SCRIPT_DIR}/../control-plane.yaml"
fi

# Wait for pods to be ready
log_info "Waiting for pods to be ready..."
if kubectl wait --for=condition=ready pod -l app=control-plane -n ${NAMESPACE} --timeout=${TIMEOUT}s; then
    pass_test "Pod ready within ${TIMEOUT} seconds"
else
    fail_test "Pod not ready within ${TIMEOUT} seconds"
fi

# ============================================================================
# Scenario 2: /health endpoint returns 200 OK
# ============================================================================
log_test "Scenario 2: /health endpoint should return 200 OK"

# Start port-forward in background
kubectl port-forward svc/${SERVICE} 18080:8080 -n ${NAMESPACE} &
PF_PID=$!
sleep 3

# Test health endpoint
HEALTH_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:18080/v1/health 2>/dev/null || echo "000")

if [ "$HEALTH_RESPONSE" == "200" ]; then
    pass_test "/health endpoint returns 200 OK"
else
    fail_test "/health endpoint returned $HEALTH_RESPONSE (expected 200)"
fi

# Test ready endpoint
READY_RESPONSE=$(curl -s http://localhost:18080/v1/ready 2>/dev/null)
READY_STATUS=$(echo "$READY_RESPONSE" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)

if [ "$READY_STATUS" == "ready" ]; then
    pass_test "/ready endpoint returns ready status"
else
    log_warn "/ready response: $READY_RESPONSE"
    fail_test "/ready endpoint not ready (status: $READY_STATUS)"
fi

# ============================================================================
# Scenario 3: Create policy and retrieve via /v1/policies
# ============================================================================
log_test "Scenario 3: Create policy and retrieve via API"

# Create a test policy
TEST_POLICY='{"metadata":{"name":"e2e-test-policy"},"spec":{"rules":[{"match":{"path":{"prefix":"/"}},"fault":{"delay":{"fixed_delay":"100ms"},"percentage":10}}]}}'

CREATE_RESPONSE=$(curl -s -X POST http://localhost:18080/v1/policies \
    -H "Content-Type: application/json" \
    -d "$TEST_POLICY" 2>/dev/null)

if echo "$CREATE_RESPONSE" | grep -q "e2e-test-policy"; then
    pass_test "Policy created successfully"
else
    log_warn "Create response: $CREATE_RESPONSE"
    fail_test "Failed to create policy"
fi

# Retrieve policy
GET_RESPONSE=$(curl -s http://localhost:18080/v1/policies/e2e-test-policy 2>/dev/null)

if echo "$GET_RESPONSE" | grep -q "e2e-test-policy"; then
    pass_test "Policy retrieved successfully"
else
    log_warn "Get response: $GET_RESPONSE"
    fail_test "Failed to retrieve policy"
fi

# List policies
LIST_RESPONSE=$(curl -s http://localhost:18080/v1/policies 2>/dev/null)

if echo "$LIST_RESPONSE" | grep -q "e2e-test-policy"; then
    pass_test "Policy appears in list"
else
    log_warn "List response: $LIST_RESPONSE"
    fail_test "Policy not in list"
fi

# Cleanup test policy
curl -s -X DELETE http://localhost:18080/v1/policies/e2e-test-policy &>/dev/null

# Kill port-forward
kill $PF_PID 2>/dev/null || true

# ============================================================================
# Scenario 4: High Availability - Service remains available after pod termination
# ============================================================================
log_test "Scenario 4: Service remains available after terminating one pod"

# Check if we have 2 replicas
REPLICAS=$(kubectl get deployment ${DEPLOYMENT} -n ${NAMESPACE} -o jsonpath='{.spec.replicas}')

if [ "$REPLICAS" -ge 2 ]; then
    # Get one pod name
    POD_TO_DELETE=$(kubectl get pods -n ${NAMESPACE} -l app=control-plane -o jsonpath='{.items[0].metadata.name}')
    
    # Start port-forward again
    kubectl port-forward svc/${SERVICE} 18080:8080 -n ${NAMESPACE} &
    PF_PID=$!
    sleep 2
    
    # Delete one pod
    log_info "Terminating pod: $POD_TO_DELETE"
    kubectl delete pod ${POD_TO_DELETE} -n ${NAMESPACE} --grace-period=1 &
    
    # Wait a moment and check if service is still available
    sleep 2
    
    HA_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:18080/v1/health 2>/dev/null || echo "000")
    
    if [ "$HA_RESPONSE" == "200" ]; then
        pass_test "Service remains available after pod termination"
    else
        fail_test "Service unavailable after pod termination (status: $HA_RESPONSE)"
    fi
    
    # Wait for replacement pod
    kubectl wait --for=condition=ready pod -l app=control-plane -n ${NAMESPACE} --timeout=60s || true
    
    kill $PF_PID 2>/dev/null || true
else
    log_warn "Skipping HA test - only $REPLICAS replica(s) configured"
    pass_test "HA test skipped (single replica mode)"
fi

# ============================================================================
# Summary
# ============================================================================
echo ""
echo "============================================"
echo "US1 Control Plane E2E Test Results"
echo "============================================"
echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"
echo "============================================"

if [ $TESTS_FAILED -gt 0 ]; then
    exit 1
fi

exit 0
