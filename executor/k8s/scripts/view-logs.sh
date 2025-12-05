#!/bin/bash
# Helper script to view istio-proxy logs in WSL2/k3s environment
# Usage: ./view-logs.sh <pod-name-pattern> [namespace] [lines]
#
# This script uses crictl directly to bypass kubectl logs timeout issues
# that can occur in WSL2 environments.

set -e

POD_PATTERN="${1:-frontend}"
NAMESPACE="${2:-demo}"
LINES="${3:-50}"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Searching for istio-proxy containers matching '$POD_PATTERN' in namespace '$NAMESPACE' ===${NC}"

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
    
    # Get logs using crictl
    sudo crictl logs --tail="$LINES" "$CONTAINER_ID" 2>&1
    
    echo ""
done

echo -e "${GREEN}=== Log viewing complete ===${NC}"
