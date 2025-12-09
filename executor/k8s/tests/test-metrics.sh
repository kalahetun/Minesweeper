#!/bin/bash
# E2E Test for Feature 008: Wasm Metrics Exposure
#
# This script validates that the three HFI metrics are properly exposed
# through Envoy stats endpoint with wasmcustom prefix.
#
# Prerequisites:
#   - kubectl configured for cluster access
#   - Wasm plugin deployed with wasmcustom.* metric names
#   - At least one pod with Istio sidecar in demo namespace
#
# Usage:
#   ./test-metrics.sh [--namespace demo] [--app frontend]
#
# Exit Codes:
#   0 - All metrics found and correctly named
#   1 - One or more metrics missing or incorrect
#   2 - Environment not ready / prerequisites missing

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="${1:-demo}"
APP_LABEL="${2:-frontend}"
REQUIRED_METRICS=(
    "wasmcustom_hfi_faults_aborts_total"
    "wasmcustom_hfi_faults_delays_total"
    "wasmcustom_hfi_faults_delay_duration_milliseconds"
)

echo "=========================================="
echo "Feature 008: Wasm Metrics Exposure Test"
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

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_fail() {
    echo -e "${RED}[✗]${NC} $1"
}

# Check prerequisites
log_info "Checking prerequisites..."

if ! command -v kubectl &> /dev/null; then
    log_error "kubectl not found. Please install kubectl."
    exit 2
fi

if ! kubectl get namespace $NAMESPACE &> /dev/null; then
    log_error "Namespace $NAMESPACE not found"
    exit 2
fi

log_success "Prerequisites verified"
echo ""

# ==================================================
# TEST 1: Find Target Pod
# ==================================================
echo "=========================================="
echo "TEST 1: Find Target Pod"
echo "=========================================="

log_info "Finding pod with label app=$APP_LABEL in namespace $NAMESPACE..."
POD=$(kubectl get pods -n $NAMESPACE -l app=$APP_LABEL -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

if [ -z "$POD" ]; then
    log_fail "No pod found with label app=$APP_LABEL in namespace $NAMESPACE"
    log_error "Available pods:"
    kubectl get pods -n $NAMESPACE
    exit 2
fi

log_success "Found pod: $POD"
echo ""

# ==================================================
# TEST 2: Verify Istio Sidecar
# ==================================================
echo "=========================================="
echo "TEST 2: Verify Istio Sidecar"
echo "=========================================="

log_info "Checking for istio-proxy container..."
if ! kubectl get pod -n $NAMESPACE $POD -o jsonpath='{.spec.containers[*].name}' | grep -q "istio-proxy"; then
    log_fail "Pod $POD does not have istio-proxy sidecar"
    log_error "Containers in pod:"
    kubectl get pod -n $NAMESPACE $POD -o jsonpath='{.spec.containers[*].name}'
    echo ""
    exit 2
fi

log_success "istio-proxy sidecar found"
echo ""

# ==================================================
# TEST 3: Query Envoy Stats Endpoint
# ==================================================
echo "=========================================="
echo "TEST 3: Query Envoy Stats Endpoint"
echo "=========================================="

log_info "Querying Envoy stats endpoint (/stats/prometheus)..."
METRICS_OUTPUT=$(kubectl exec -n $NAMESPACE $POD -c istio-proxy -- \
    curl -s http://localhost:15090/stats/prometheus 2>/dev/null)

if [ -z "$METRICS_OUTPUT" ]; then
    log_fail "Failed to retrieve metrics from Envoy stats endpoint"
    exit 1
fi

METRICS_COUNT=$(echo "$METRICS_OUTPUT" | wc -l)
log_success "Retrieved $METRICS_COUNT lines of metrics"
echo ""

# ==================================================
# TEST 4: Verify Required Metrics Exist
# ==================================================
echo "=========================================="
echo "TEST 4: Verify Required Metrics"
echo "=========================================="

FAILED_METRICS=()
PASSED_METRICS=()

for metric in "${REQUIRED_METRICS[@]}"; do
    log_info "Checking for metric: $metric"
    
    if echo "$METRICS_OUTPUT" | grep -q "^$metric"; then
        # Get the metric value
        METRIC_LINE=$(echo "$METRICS_OUTPUT" | grep "^$metric" | head -1)
        log_success "Found: $METRIC_LINE"
        PASSED_METRICS+=("$metric")
    else
        log_fail "Missing: $metric"
        FAILED_METRICS+=("$metric")
    fi
    echo ""
done

# ==================================================
# TEST 5: Verify Histogram Buckets
# ==================================================
echo "=========================================="
echo "TEST 5: Verify Histogram Buckets"
echo "=========================================="

log_info "Checking histogram buckets for delay_duration_milliseconds..."

EXPECTED_BUCKETS=(
    "0.5" "1" "5" "10" "25" "50" "100" "250" "500" "1000" 
    "2500" "5000" "10000" "30000" "60000" "300000" "600000" 
    "1800000" "3600000" "+Inf"
)

HISTOGRAM_METRIC="wasmcustom_hfi_faults_delay_duration_milliseconds"
BUCKET_COUNT=0

for bucket in "${EXPECTED_BUCKETS[@]}"; do
    # Escape special characters for grep
    ESCAPED_BUCKET=$(echo "$bucket" | sed 's/+/\\+/g')
    if echo "$METRICS_OUTPUT" | grep -q "${HISTOGRAM_METRIC}_bucket{le=\"${ESCAPED_BUCKET}\"}"; then
        ((BUCKET_COUNT++)) || true
    fi
done

if [ $BUCKET_COUNT -eq ${#EXPECTED_BUCKETS[@]} ]; then
    log_success "All $BUCKET_COUNT histogram buckets found"
else
    log_warn "Found $BUCKET_COUNT/${#EXPECTED_BUCKETS[@]} histogram buckets"
fi

# Check histogram sum and count
if echo "$METRICS_OUTPUT" | grep -q "${HISTOGRAM_METRIC}_sum"; then
    SUM_VALUE=$(echo "$METRICS_OUTPUT" | grep "${HISTOGRAM_METRIC}_sum" | awk '{print $2}')
    log_success "Found histogram sum: $SUM_VALUE"
else
    log_fail "Missing histogram sum"
    FAILED_METRICS+=("${HISTOGRAM_METRIC}_sum")
fi

if echo "$METRICS_OUTPUT" | grep -q "${HISTOGRAM_METRIC}_count"; then
    COUNT_VALUE=$(echo "$METRICS_OUTPUT" | grep "${HISTOGRAM_METRIC}_count" | awk '{print $2}')
    log_success "Found histogram count: $COUNT_VALUE"
else
    log_fail "Missing histogram count"
    FAILED_METRICS+=("${HISTOGRAM_METRIC}_count")
fi

echo ""

# ==================================================
# TEST 6: Verify No Old Metric Names
# ==================================================
echo "=========================================="
echo "TEST 6: Verify No Old Metric Names"
echo "=========================================="

log_info "Checking for old metric names (hfi.faults.*)..."

OLD_METRICS_FOUND=false
OLD_METRIC_PATTERNS=(
    "^hfi_faults_aborts_total"
    "^hfi_faults_delays_total"
    "^hfi_faults_delay_duration_milliseconds"
)

for old_metric in "${OLD_METRIC_PATTERNS[@]}"; do
    if echo "$METRICS_OUTPUT" | grep -q "$old_metric"; then
        log_fail "Found old metric name pattern: $old_metric (should use wasmcustom prefix)"
        OLD_METRICS_FOUND=true
    fi
done

if [ "$OLD_METRICS_FOUND" = false ]; then
    log_success "No old metric names found (all using wasmcustom prefix)"
else
    log_error "Old metric names detected - plugin needs update"
    FAILED_METRICS+=("old_metric_names")
fi

echo ""

# ==================================================
# TEST 7: Optional - Check EnvoyFilter
# ==================================================
echo "=========================================="
echo "TEST 7: Optional EnvoyFilter Check"
echo "=========================================="

log_info "Checking if EnvoyFilter is deployed..."
if kubectl get envoyfilter hfi-wasm-metrics -n $NAMESPACE &> /dev/null; then
    log_success "EnvoyFilter 'hfi-wasm-metrics' found in namespace $NAMESPACE"
    
    # Check if it has the right configuration
    if kubectl get envoyfilter hfi-wasm-metrics -n $NAMESPACE -o yaml | grep -q "wasmcustom"; then
        log_success "EnvoyFilter contains wasmcustom pattern"
    else
        log_warn "EnvoyFilter exists but may not have wasmcustom pattern"
    fi
else
    log_warn "EnvoyFilter not found (optional - metrics should work without it)"
fi

echo ""

# ==================================================
# SUMMARY
# ==================================================
echo "=========================================="
echo "TEST SUMMARY"
echo "=========================================="
echo ""

echo -e "Target Pod: ${BLUE}$POD${NC} (namespace: $NAMESPACE)"
echo -e "Total Metrics Lines: ${BLUE}$METRICS_COUNT${NC}"
echo ""

echo "Required Metrics Status:"
echo "  ✓ Passed: ${#PASSED_METRICS[@]}"
echo "  ✗ Failed: ${#FAILED_METRICS[@]}"
echo ""

if [ ${#FAILED_METRICS[@]} -eq 0 ]; then
    echo -e "${GREEN}=========================================="
    echo -e "  ✓ ALL TESTS PASSED"
    echo -e "==========================================${NC}"
    echo ""
    echo "All three HFI metrics are properly exposed with wasmcustom prefix:"
    for metric in "${PASSED_METRICS[@]}"; do
        echo "  ✓ $metric"
    done
    echo ""
    exit 0
else
    echo -e "${RED}=========================================="
    echo -e "  ✗ TESTS FAILED"
    echo -e "==========================================${NC}"
    echo ""
    echo "Missing or incorrect metrics:"
    for metric in "${FAILED_METRICS[@]}"; do
        echo "  ✗ $metric"
    done
    echo ""
    echo "Troubleshooting steps:"
    echo "  1. Check Wasm plugin logs:"
    echo "     kubectl logs -n $NAMESPACE $POD -c istio-proxy | grep -i wasm"
    echo ""
    echo "  2. Verify plugin binary has updated metric names:"
    echo "     grep 'wasmcustom' executor/wasm-plugin/src/lib.rs"
    echo ""
    echo "  3. See METRICS_SOLUTION.md for detailed troubleshooting"
    echo ""
    exit 1
fi
