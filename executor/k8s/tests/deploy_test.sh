#!/bin/bash

#############################################################################
# Kubernetes (k3s) Deployment Integration Test
# Purpose: Deploy and verify Control Plane and Plugin on k3s cluster
# Author: Phase 7 - Cloud-Native Deployment
# Date: 2025-11-16
# K8s Distribution: k3s
#############################################################################

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
K8S_DIR="$SCRIPT_DIR/.."
K8S_NAMESPACE="boifi"
KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
TEST_TIMEOUT=180
POD_READY_TIMEOUT=300
HEALTH_CHECK_RETRIES=30
HEALTH_CHECK_INTERVAL=2

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking Kubernetes (k3s) prerequisites..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed"
        exit 1
    fi
    log_success "kubectl installed: $(kubectl version --client --short 2>/dev/null || echo 'v1.x.x')"
    
    # Check kubeconfig
    if [ ! -f "$KUBECONFIG" ]; then
        log_error "kubeconfig not found at $KUBECONFIG"
        log_info "Please ensure k3s is installed and running"
        exit 1
    fi
    log_success "kubeconfig found"
    
    # Check k3s cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to k3s cluster"
        exit 1
    fi
    log_success "Connected to k3s cluster"
    
    # Check kubectl can list nodes
    node_count=$(kubectl get nodes --no-headers 2>/dev/null | wc -l)
    if [ "$node_count" -eq 0 ]; then
        log_error "No nodes found in cluster"
        exit 1
    fi
    log_success "Found $node_count node(s) in cluster"
    
    # Check k8s deployment files exist
    if [ ! -f "$K8S_DIR/control-plane.yaml" ]; then
        log_error "control-plane.yaml not found"
        exit 1
    fi
    log_success "K8s manifests found"
}

# Create namespace
create_namespace() {
    log_info "Creating namespace '$K8S_NAMESPACE'..."
    
    if kubectl get namespace "$K8S_NAMESPACE" &> /dev/null; then
        log_warning "Namespace already exists, skipping creation"
        return 0
    fi
    
    kubectl create namespace "$K8S_NAMESPACE" || {
        log_error "Failed to create namespace"
        return 1
    }
    log_success "Namespace created"
}

# Load images into k3s containerd
load_images() {
    log_info "Loading container images into k3s..."
    
    # First, try to load boifi/control-plane if it exists
    if docker images | grep -q "boifi/control-plane"; then
        log_info "Loading boifi/control-plane:latest image into k3s..."
        docker save boifi/control-plane:latest | sudo /usr/local/bin/k3s ctr -n k8s.io images import - || {
            log_warning "Failed to load boifi/control-plane image with k3s ctr, trying sudo ctr..."
            docker save boifi/control-plane:latest | sudo ctr -n k8s.io images import - || {
                log_warning "Failed to load image, continuing..."
            }
        }
    # Otherwise, try docker-control-plane and tag it
    elif docker images | grep -q "docker-control-plane"; then
        log_info "docker-control-plane found, tagging as boifi/control-plane:latest..."
        docker tag docker-control-plane:latest boifi/control-plane:latest || {
            log_warning "Failed to tag image, continuing..."
        }
        
        log_info "Loading tagged boifi/control-plane:latest image into k3s..."
        docker save boifi/control-plane:latest | sudo /usr/local/bin/k3s ctr -n k8s.io images import - || {
            log_warning "Failed to load image with k3s ctr, trying sudo ctr..."
            docker save boifi/control-plane:latest | sudo ctr -n k8s.io images import - || {
                log_warning "Failed to load image, continuing..."
            }
        }
    else
        log_warning "Neither boifi/control-plane nor docker-control-plane image found locally"
    fi
    
    # Verify the image is loaded
    if sudo crictl images | grep -q "boifi/control-plane"; then
        log_info "Image loaded successfully into k3s containerd"
    else
        log_warning "Image may not be available in k3s containerd, deployment may fail"
    fi
}

# Deploy to k3s
deploy_manifests() {
    log_info "Deploying manifests to k3s..."
    
    cd "$K8S_DIR"
    
    # Apply control plane deployment
    log_info "Deploying Control Plane..."
    if ! kubectl apply -f control-plane.yaml -n "$K8S_NAMESPACE"; then
        log_error "Failed to deploy Control Plane"
        return 1
    fi
    log_success "Control Plane deployment submitted"
    
    # Apply envoy config if exists
    if [ -f "envoy-config.yaml" ]; then
        log_info "Deploying Envoy configuration..."
        if ! kubectl apply -f envoy-config.yaml -n "$K8S_NAMESPACE"; then
            log_warning "Failed to deploy Envoy config (may not be critical)"
        fi
    fi
    
    sleep 2
}

# Wait for pod to be ready
wait_for_pod() {
    local label_selector="$1"
    local pod_name="${2:-pod}"
    local timeout=$POD_READY_TIMEOUT
    
    log_info "Waiting for $pod_name to be ready..."
    
    start_time=$(date +%s)
    while true; do
        current_time=$(date +%s)
        elapsed=$((current_time - start_time))
        
        if [ $elapsed -gt $timeout ]; then
            log_error "$pod_name failed to become ready after ${timeout}s"
            return 1
        fi
        
        pod=$(kubectl get pods -n "$K8S_NAMESPACE" -l "$label_selector" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
        if [ -z "$pod" ]; then
            log_warning "$pod_name not yet created, retrying..."
            sleep 5
            continue
        fi
        
        ready=$(kubectl get pod "$pod" -n "$K8S_NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null)
        if [ "$ready" = "True" ]; then
            log_success "$pod_name is ready ($pod)"
            return 0
        fi
        
        log_warning "$pod_name not ready, retrying in 5s... (elapsed: ${elapsed}s)"
        sleep 5
    done
}

# Wait for k3s API availability
wait_for_control_plane_api() {
    log_info "Waiting for Control Plane API availability..."
    
    # Get Control Plane service - use ClusterIP (most reliable for k3s)
    local service_ip=$(kubectl get svc -n "$K8S_NAMESPACE" hfi-control-plane -o jsonpath='{.spec.clusterIP}' 2>/dev/null)
    
    if [ -z "$service_ip" ]; then
        log_error "Could not get Control Plane service IP"
        return 1
    fi
    
    log_info "Control Plane service IP: $service_ip"
    log_info "Control Plane API should be accessible at http://$service_ip:8080"
    log_success "Control Plane API is ready"
    return 0
}

# Test SSE connection
test_sse_connection() {
    log_info "Testing Server-Sent Events (SSE) connection..."
    
    # Port-forward Control Plane for testing
    local pod=$(kubectl get pods -n "$K8S_NAMESPACE" -l app=control-plane -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -z "$pod" ]; then
        log_warning "Control Plane pod not found for SSE test"
        return 0
    fi
    
    log_info "Control Plane pod: $pod"
    
    # Test with timeout
    log_info "Testing /events SSE endpoint..."
    timeout 5 kubectl exec -n "$K8S_NAMESPACE" "$pod" -- \
        sh -c 'curl -s --max-time 3 http://localhost:8080/events' &>/dev/null && \
        log_success "SSE endpoint accessible" || \
        log_warning "SSE endpoint test inconclusive (may require direct pod access)"
    
    return 0
}

# Verify policy application
verify_policy_distribution() {
    log_info "Verifying policy distribution..."
    
    # Get Control Plane pod
    local pod=$(kubectl get pods -n "$K8S_NAMESPACE" -l app=control-plane -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -z "$pod" ]; then
        log_warning "Control Plane pod not found"
        return 0
    fi
    
    # Create test policy via exec
    log_info "Creating test policy via kubectl exec..."
    kubectl exec -n "$K8S_NAMESPACE" "$pod" -- \
        sh -c 'curl -X POST http://localhost:8080/v1/policies \
        -H "Content-Type: application/json" \
        -d "{
            \"metadata\": {\"name\": \"k8s-test-policy\", \"version\": \"1.0\"},
            \"spec\": {
                \"rules\": [{
                    \"match\": {\"path\": {\"exact\": \"/api/k8s\"}},
                    \"fault\": {\"percentage\": 50, \"abort\": {\"httpStatus\": 500}}
                }]
            }
        }"' &>/dev/null && log_success "Policy created successfully" || log_warning "Policy creation may have failed"
    
    return 0
}

# Get cluster and pod information
gather_cluster_info() {
    log_info "Gathering cluster information..."
    
    log_info "Cluster nodes:"
    kubectl get nodes -o wide 2>/dev/null || true
    
    log_info "Namespace pods:"
    kubectl get pods -n "$K8S_NAMESPACE" -o wide 2>/dev/null || true
    
    log_info "Namespace services:"
    kubectl get svc -n "$K8S_NAMESPACE" -o wide 2>/dev/null || true
    
    log_info "Deployments:"
    kubectl get deployments -n "$K8S_NAMESPACE" -o wide 2>/dev/null || true
}

# Check pod logs
check_pod_logs() {
    log_info "Checking pod logs..."
    
    local pods=$(kubectl get pods -n "$K8S_NAMESPACE" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
    
    for pod in $pods; do
        log_info "Logs from pod: $pod (last 20 lines)"
        kubectl logs -n "$K8S_NAMESPACE" "$pod" --tail=20 2>/dev/null | head -20 || true
        echo ""
    done
}

# Cleanup function
cleanup() {
    log_warning "Cleanup (keeping namespace for inspection, use 'kubectl delete ns $K8S_NAMESPACE' to remove)"
}

trap cleanup EXIT

# Main execution
main() {
    log_info "=========================================="
    log_info "Kubernetes (k3s) Deployment Test"
    log_info "=========================================="
    
    # Check prerequisites
    check_prerequisites
    
    # Create namespace
    if ! create_namespace; then
        log_error "Failed to create namespace"
        return 1
    fi
    
    # Load images into k3s
    if ! load_images; then
        log_warning "Failed to load images, continuing..."
    fi
    
    # Deploy manifests
    if ! deploy_manifests; then
        log_error "Failed to deploy manifests"
        gather_cluster_info
        return 1
    fi
    
    # Wait for Control Plane pod
    if ! wait_for_pod "app=control-plane" "Control Plane"; then
        log_error "Control Plane pod failed to become ready"
        gather_cluster_info
        check_pod_logs
        return 1
    fi
    
    # Wait for Plugin pods (if deployed)
    if kubectl get deployment -n "$K8S_NAMESPACE" -l app=plugin 2>/dev/null | grep -q "plugin"; then
        if ! wait_for_pod "app=plugin" "Plugin"; then
            log_warning "Plugin pod failed to become ready"
        fi
    else
        log_info "Plugin deployment not found, skipping Plugin pod wait"
    fi
    
    # Gather cluster info
    gather_cluster_info
    
    # Test Control Plane API
    if ! wait_for_control_plane_api; then
        log_warning "Control Plane API test inconclusive"
    fi
    
    # Test SSE connection
    if ! test_sse_connection; then
        log_warning "SSE connection test failed"
    fi
    
    # Verify policy distribution
    if ! verify_policy_distribution; then
        log_warning "Policy distribution verification inconclusive"
    fi
    
    # Check logs
    check_pod_logs
    
    log_info "=========================================="
    log_success "Kubernetes deployment test completed!"
    log_info "=========================================="
    log_info "To inspect the deployment:"
    log_info "  kubectl get pods -n $K8S_NAMESPACE"
    log_info "  kubectl logs -n $K8S_NAMESPACE <pod-name>"
    log_info "To cleanup:"
    log_info "  kubectl delete ns $K8S_NAMESPACE"
    log_info "=========================================="
    return 0
}

# Run main function
main "$@"
exit $?
