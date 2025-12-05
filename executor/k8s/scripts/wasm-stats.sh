#!/bin/bash
# Quick stats check for Wasm plugin metrics
# Usage: ./wasm-stats.sh <pod-name-pattern> [namespace]

set -e

POD_PATTERN="${1:-frontend}"
NAMESPACE="${2:-demo}"

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${GREEN}=== Wasm Plugin Stats for '$POD_PATTERN' in namespace '$NAMESPACE' ===${NC}"

# Find matching pods
POD=$(kubectl get pods -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null | tr ' ' '\n' | grep -i "$POD_PATTERN" | head -1)

if [ -z "$POD" ]; then
    echo -e "${RED}No pods found matching pattern '$POD_PATTERN' in namespace '$NAMESPACE'${NC}"
    exit 1
fi

echo -e "${YELLOW}Pod: $POD${NC}"
echo ""

# Get stats via Envoy admin API
echo -e "${CYAN}[Wasm Runtime Stats]${NC}"
kubectl exec -n "$NAMESPACE" "$POD" -c istio-proxy -- curl -s localhost:15000/stats 2>&1 | grep -E "^wasm\." || echo "(No wasm stats found)"

echo ""
echo -e "${CYAN}[HFI Custom Metrics]${NC}"
kubectl exec -n "$NAMESPACE" "$POD" -c istio-proxy -- curl -s localhost:15000/stats 2>&1 | grep -E "wasmcustom.*hfi" || echo "(No HFI metrics found)"

echo ""
echo -e "${CYAN}[Control Plane Cluster Stats]${NC}"
kubectl exec -n "$NAMESPACE" "$POD" -c istio-proxy -- curl -s localhost:15000/clusters 2>&1 | grep "hfi-control-plane" | grep -E "cx_active|rq_total|health" || echo "(No control plane cluster stats found)"

echo ""
echo -e "${GREEN}=== Done ===${NC}"
