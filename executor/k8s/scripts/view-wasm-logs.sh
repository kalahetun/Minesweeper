#!/bin/bash
# View Wasm plugin logs from Envoy (grep for wasm/hfi related entries)
# Usage: ./view-wasm-logs.sh <pod-name-pattern> [namespace] [lines]

set -e

POD_PATTERN="${1:-frontend}"
NAMESPACE="${2:-demo}"
LINES="${3:-100}"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${GREEN}=== Wasm Plugin Logs for '$POD_PATTERN' in namespace '$NAMESPACE' ===${NC}"

# Find matching pods
PODS=$(kubectl get pods -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null | tr ' ' '\n' | grep -i "$POD_PATTERN" || true)

if [ -z "$PODS" ]; then
    echo -e "${RED}No pods found matching pattern '$POD_PATTERN' in namespace '$NAMESPACE'${NC}"
    exit 1
fi

for POD in $PODS; do
    echo -e "${YELLOW}--- Pod: $POD ---${NC}"
    
    # Get container ID using crictl
    CONTAINER_ID=$(sudo crictl ps -q --label "io.kubernetes.pod.name=$POD" --label "io.kubernetes.container.name=istio-proxy" 2>/dev/null | head -1)
    
    if [ -z "$CONTAINER_ID" ]; then
        echo -e "${RED}Could not find istio-proxy container for pod $POD${NC}"
        continue
    fi
    
    echo -e "${GREEN}Container ID: $CONTAINER_ID${NC}"
    echo ""
    
    # Get logs and filter for wasm/hfi related entries
    echo -e "${CYAN}[Wasm/HFI related logs]${NC}"
    sudo crictl logs --tail="$LINES" "$CONTAINER_ID" 2>&1 | grep -iE "wasm|hfi|plugin|fault|configured|policies|rule|matched|abort|delay" || echo "(No wasm-related logs found)"
    
    echo ""
    echo -e "${CYAN}[Recent access logs with fault injection]${NC}"
    sudo crictl logs --tail="$LINES" "$CONTAINER_ID" 2>&1 | grep -E "HTTP/[12]\.[01]\" 50[0-9]" | head -10 || echo "(No 5xx responses found)"
    
    echo ""
done

echo -e "${GREEN}=== Done ===${NC}"
