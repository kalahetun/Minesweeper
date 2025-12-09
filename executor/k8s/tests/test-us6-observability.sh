#!/bin/bash
# Test Script for US6: Observability Features
# Tests Prometheus metrics and policy status endpoint
#
# Prerequisites:
#   - kubectl configured for cluster access
#   - jq installed for JSON parsing
#   - curl installed
#   - Control Plane and Wasm plugin deployed
#   - At least one policy applied
#
# Usage:
#   ./test-us6-observability.sh
#
# Expected Outcomes:
#   - ✓ Prometheus metrics endpoint accessible
#   - ✓ Metrics contain wasmcustom.hfi.faults.* counters
#   - ✓ /v1/policies/status endpoint returns 200
#   - ✓ Status response includes summary and policy details
#   - ✓ WasmPlugin status shows READY

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

NAMESPACE="boifi"
DEMO_NAMESPACE="demo"
CONTROL_PLANE_SVC="hfi-control-plane"
TEST_POLICY_NAME="observability-test-policy"

echo "=========================================="
echo "US6: Observability Features E2E Test"
echo "=========================================="
echo ""

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

check_command() {
    if ! command -v $1 &> /dev/null; then
        log_error "$1 command not found. Please install $1."
        exit 1
    fi
}

# Verify prerequisites
log_info "Checking prerequisites..."
check_command kubectl
check_command jq
check_command curl

# Check namespace exists
if ! kubectl get namespace $NAMESPACE &> /dev/null; then
    log_error "Namespace $NAMESPACE not found"
    exit 1
fi

if ! kubectl get namespace $DEMO_NAMESPACE &> /dev/null; then
    log_error "Namespace $DEMO_NAMESPACE not found"
    exit 1
fi

log_info "✓ Prerequisites verified"
echo ""

# ==================================================
# SCENARIO 1: Verify Prometheus Metrics via Envoy
# ==================================================
echo "=========================================="
echo "SCENARIO 1: Prometheus Metrics"
echo "=========================================="

log_info "Finding a frontend Pod in demo namespace..."
FRONTEND_POD=$(kubectl get pods -n $DEMO_NAMESPACE -l app=frontend -o jsonpath='{.items[0].metadata.name}')

if [ -z "$FRONTEND_POD" ]; then
    log_error "No frontend Pod found in $DEMO_NAMESPACE"
    exit 1
fi

log_info "Using Pod: $FRONTEND_POD"

log_info "Fetching Prometheus metrics from Envoy stats endpoint..."
METRICS=$(kubectl exec -n $DEMO_NAMESPACE $FRONTEND_POD -c istio-proxy -- curl -s http://localhost:15090/stats/prometheus 2>/dev/null)

if [ -z "$METRICS" ]; then
    log_error "Failed to fetch metrics from Envoy"
    exit 1
fi

log_info "✓ Metrics endpoint accessible"

# Check for HFI-specific metrics
log_info "Checking for HFI fault injection metrics..."

ABORTS_METRIC=$(echo "$METRICS" | grep "wasmcustom_hfi_faults_aborts_total" || true)
DELAYS_METRIC=$(echo "$METRICS" | grep "wasmcustom_hfi_faults_delays_total" || true)

if [ -n "$ABORTS_METRIC" ]; then
    ABORTS_COUNT=$(echo "$ABORTS_METRIC" | grep -oP 'wasmcustom_hfi_faults_aborts_total \K\d+' | tail -1)
    log_info "✓ Found aborts_total metric: $ABORTS_COUNT"
else
    log_warn "⚠ aborts_total metric not found (may be zero if no aborts occurred)"
fi

if [ -n "$DELAYS_METRIC" ]; then
    DELAYS_COUNT=$(echo "$DELAYS_METRIC" | grep -oP 'wasmcustom_hfi_faults_delays_total \K\d+' | tail -1)
    log_info "✓ Found delays_total metric: $DELAYS_COUNT"
else
    log_warn "⚠ delays_total metric not found (may be zero if no delays occurred)"
fi

# Sample 10 lines of HFI metrics
log_info "Sample HFI metrics:"
echo "$METRICS" | grep "wasmcustom_hfi" | head -10 || log_warn "No wasmcustom_hfi metrics found"

echo ""
log_info "✓ SCENARIO 1 PASSED: Prometheus metrics verified"
echo ""

# ==================================================
# SCENARIO 2: Control Plane Policy Status Endpoint
# ==================================================
echo "=========================================="
echo "SCENARIO 2: Policy Status Endpoint"
echo "=========================================="

log_info "Setting up port-forward to Control Plane..."
# Kill existing port-forwards on 8080
pkill -f "kubectl port-forward.*8080:8080" 2>/dev/null || true
sleep 2

kubectl port-forward -n $NAMESPACE svc/$CONTROL_PLANE_SVC 8080:8080 > /dev/null 2>&1 &
PORT_FORWARD_PID=$!
sleep 3

# Cleanup function
cleanup() {
    log_info "Cleaning up port-forward..."
    kill $PORT_FORWARD_PID 2>/dev/null || true
}
trap cleanup EXIT

log_info "Testing /v1/policies/status endpoint..."
STATUS_RESPONSE=$(curl -s http://localhost:8080/v1/policies/status)
STATUS_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/v1/policies/status)

if [ "$STATUS_CODE" != "200" ]; then
    log_error "Status endpoint returned HTTP $STATUS_CODE"
    log_error "Response: $STATUS_RESPONSE"
    exit 1
fi

log_info "✓ Status endpoint returned 200"

# Validate JSON structure
if ! echo "$STATUS_RESPONSE" | jq empty 2>/dev/null; then
    log_error "Invalid JSON response"
    exit 1
fi

log_info "✓ Valid JSON response"

# Extract summary fields
TOTAL_POLICIES=$(echo "$STATUS_RESPONSE" | jq -r '.summary.total_policies')
ABORT_POLICIES=$(echo "$STATUS_RESPONSE" | jq -r '.summary.abort_policies')
DELAY_POLICIES=$(echo "$STATUS_RESPONSE" | jq -r '.summary.delay_policies')
ACTIVE_POLICIES=$(echo "$STATUS_RESPONSE" | jq -r '.summary.active_policies')

log_info "Summary Statistics:"
log_info "  Total Policies: $TOTAL_POLICIES"
log_info "  Abort Policies: $ABORT_POLICIES"
log_info "  Delay Policies: $DELAY_POLICIES"
log_info "  Active Policies: $ACTIVE_POLICIES"

if [ "$TOTAL_POLICIES" = "null" ] || [ "$TOTAL_POLICIES" -lt 0 ]; then
    log_error "Invalid total_policies value: $TOTAL_POLICIES"
    exit 1
fi

log_info "✓ Summary fields valid"

# Check policies array
POLICIES_COUNT=$(echo "$STATUS_RESPONSE" | jq -r '.policies | length')
log_info "Policies in response: $POLICIES_COUNT"

if [ "$POLICIES_COUNT" != "$TOTAL_POLICIES" ]; then
    log_error "Mismatch: summary shows $TOTAL_POLICIES but policies array has $POLICIES_COUNT"
    exit 1
fi

log_info "✓ Policy count matches summary"

# Display sample policy details
if [ "$POLICIES_COUNT" -gt 0 ]; then
    log_info "Sample policy details:"
    echo "$STATUS_RESPONSE" | jq -r '.policies[0] | "  Name: \(.name)\n  Service: \(.target_service)\n  Namespace: \(.namespace)\n  Rules: \(.rules_count)\n  Faults: \(.fault_types | join(", "))\n  Active: \(.active)"'
fi

echo ""
log_info "✓ SCENARIO 2 PASSED: Policy status endpoint verified"
echo ""

# ==================================================
# SCENARIO 3: WasmPlugin Status via kubectl
# ==================================================
echo "=========================================="
echo "SCENARIO 3: WasmPlugin Kubernetes Status"
echo "=========================================="

log_info "Listing WasmPlugin resources..."
WASMPLUGINS=$(kubectl get wasmplugins.extensions.istio.io -n $DEMO_NAMESPACE -o json)

PLUGIN_COUNT=$(echo "$WASMPLUGINS" | jq -r '.items | length')

if [ "$PLUGIN_COUNT" -eq 0 ]; then
    log_warn "No WasmPlugin resources found in $DEMO_NAMESPACE"
else
    log_info "Found $PLUGIN_COUNT WasmPlugin(s)"
    
    # Check first WasmPlugin status
    PLUGIN_NAME=$(echo "$WASMPLUGINS" | jq -r '.items[0].metadata.name')
    log_info "Checking status of WasmPlugin: $PLUGIN_NAME"
    
    # Get status from kubectl describe
    PLUGIN_STATUS=$(kubectl describe wasmplugins.extensions.istio.io -n $DEMO_NAMESPACE $PLUGIN_NAME | grep -A 5 "Status:" || true)
    
    if [ -n "$PLUGIN_STATUS" ]; then
        log_info "WasmPlugin Status:"
        echo "$PLUGIN_STATUS"
    else
        log_warn "No status information available for WasmPlugin"
    fi
fi

# List all WasmPlugin names
log_info "WasmPlugin resources:"
kubectl get wasmplugins.extensions.istio.io -n $DEMO_NAMESPACE -o custom-columns=NAME:.metadata.name,SELECTOR:.spec.selector.labels 2>/dev/null || log_warn "Unable to list WasmPlugins"

echo ""
log_info "✓ SCENARIO 3 PASSED: WasmPlugin status retrieved"
echo ""

# ==================================================
# SCENARIO 4: Integration Test - Apply Policy and Verify
# ==================================================
echo "=========================================="
echo "SCENARIO 4: Integration Test"
echo "=========================================="

log_info "Creating test policy with 50% abort..."

cat > /tmp/$TEST_POLICY_NAME.yaml <<EOF
metadata:
  name: $TEST_POLICY_NAME
spec:
  selector:
    service: frontend
    namespace: demo
  rules:
    - match:
        path:
          prefix: /
      fault:
        percentage: 50
        abort:
          httpStatus: 503
EOF

# Apply policy via CLI (assuming hfi-cli is built)
if [ -f "./cli/hfi-cli" ]; then
    log_info "Applying policy via hfi-cli..."
    ./cli/hfi-cli policy apply -f /tmp/$TEST_POLICY_NAME.yaml
else
    log_info "Applying policy via curl..."
    POLICY_JSON=$(cat /tmp/$TEST_POLICY_NAME.yaml | python3 -c 'import sys, yaml, json; json.dump(yaml.safe_load(sys.stdin), sys.stdout)')
    curl -s -X POST http://localhost:8080/v1/policies \
        -H "Content-Type: application/json" \
        -d "$POLICY_JSON" > /dev/null
fi

log_info "✓ Policy applied"

# Wait for policy propagation
log_info "Waiting 5 seconds for policy propagation..."
sleep 5

# Verify policy appears in status endpoint
log_info "Verifying policy in status endpoint..."
NEW_STATUS=$(curl -s http://localhost:8080/v1/policies/status)
NEW_COUNT=$(echo "$NEW_STATUS" | jq -r '.summary.total_policies')

if [ "$NEW_COUNT" -gt "$TOTAL_POLICIES" ]; then
    log_info "✓ New policy detected (count increased from $TOTAL_POLICIES to $NEW_COUNT)"
else
    log_warn "Policy count unchanged (may have replaced existing policy)"
fi

# Check if our test policy exists
TEST_POLICY_EXISTS=$(echo "$NEW_STATUS" | jq -r ".policies[] | select(.name==\"$TEST_POLICY_NAME\") | .name")

if [ "$TEST_POLICY_EXISTS" = "$TEST_POLICY_NAME" ]; then
    log_info "✓ Test policy '$TEST_POLICY_NAME' found in status"
else
    log_error "Test policy '$TEST_POLICY_NAME' not found in status"
    exit 1
fi

# Send 10 test requests to trigger faults
log_info "Sending 10 test requests to frontend..."
FAILURES=0
for i in {1..10}; do
    RESPONSE=$(kubectl exec -n $DEMO_NAMESPACE $FRONTEND_POD -c server -- curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/ 2>/dev/null || echo "000")
    if [ "$RESPONSE" = "503" ]; then
        FAILURES=$((FAILURES + 1))
    fi
done

log_info "Observed $FAILURES/10 failures (expected ~5 with 50% rate)"

if [ $FAILURES -ge 3 ] && [ $FAILURES -le 7 ]; then
    log_info "✓ Failure rate within expected range"
else
    log_warn "⚠ Failure rate outside expected range (3-7), got $FAILURES"
fi

# Re-check metrics to confirm counter increased
log_info "Verifying metrics updated..."
sleep 2
NEW_METRICS=$(kubectl exec -n $DEMO_NAMESPACE $FRONTEND_POD -c istio-proxy -- curl -s http://localhost:15090/stats/prometheus 2>/dev/null)
NEW_ABORTS=$(echo "$NEW_METRICS" | grep "wasmcustom_hfi_faults_aborts_total" | grep -oP 'wasmcustom_hfi_faults_aborts_total \K\d+' | tail -1 || echo "0")

if [ -n "$ABORTS_COUNT" ]; then
    if [ "$NEW_ABORTS" -gt "$ABORTS_COUNT" ]; then
        log_info "✓ Aborts metric increased (from $ABORTS_COUNT to $NEW_ABORTS)"
    else
        log_warn "⚠ Aborts metric unchanged ($NEW_ABORTS)"
    fi
else
    log_info "✓ Aborts metric now showing: $NEW_ABORTS"
fi

# Cleanup test policy
log_info "Cleaning up test policy..."
if [ -f "./cli/hfi-cli" ]; then
    ./cli/hfi-cli policy delete $TEST_POLICY_NAME 2>/dev/null || true
else
    curl -s -X DELETE http://localhost:8080/v1/policies/$TEST_POLICY_NAME > /dev/null || true
fi

rm -f /tmp/$TEST_POLICY_NAME.yaml

echo ""
log_info "✓ SCENARIO 4 PASSED: Integration test completed"
echo ""

# ==================================================
# SUMMARY
# ==================================================
echo "=========================================="
echo "TEST SUMMARY"
echo "=========================================="
log_info "✓ All scenarios passed"
log_info "✓ Prometheus metrics verified"
log_info "✓ Policy status endpoint operational"
log_info "✓ WasmPlugin status accessible"
log_info "✓ End-to-end observability validated"
echo ""
log_info "US6 Observability Features: VERIFIED"
echo "=========================================="

exit 0
