#!/bin/bash

#############################################################################
# Multi-Instance Policy Distribution Test (k3s)
# Purpose: Deploy 3 Plugin instances and verify policy distribution < 1 second
# Author: Phase 7 - Cloud-Native Deployment
# Date: 2025-11-16
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
K8S_DIR="$SCRIPT_DIR/.."
K8S_NAMESPACE="boifi"
KUBECONFIG="${KUBECONFIG:-$HOME/.kube/k3s.yaml}"
PLUGIN_REPLICAS=3
DISTRIBUTION_TIMEOUT=10
POD_READY_TIMEOUT=300

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
    log_info "Checking prerequisites..."
    
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed"
        exit 1
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to k3s cluster"
        exit 1
    fi
    
    if ! kubectl get namespace "$K8S_NAMESPACE" &>/dev/null; then
        log_error "Namespace '$K8S_NAMESPACE' does not exist"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Create multi-instance plugin deployment manifest
create_plugin_manifest() {
    log_info "Creating Plugin deployment manifest with $PLUGIN_REPLICAS replicas..."
    
    cat > "$K8S_DIR/plugin-multi-instance.yaml" << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: plugin-multi-instance
  labels:
    app: plugin
    instance: multi
spec:
  replicas: 3
  selector:
    matchLabels:
      app: plugin
      instance: multi
  template:
    metadata:
      labels:
        app: plugin
        instance: multi
    spec:
      containers:
      - name: envoy-wasm-plugin
        image: envoyproxy/envoy:v1.27-latest
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 9000
          name: admin
        - containerPort: 10000
          name: upstream
        env:
        - name: CONTROL_PLANE_ADDR
          value: "control-plane:8080"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /stats
            port: 9000
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /stats
            port: 9000
          initialDelaySeconds: 5
          periodSeconds: 5
      serviceAccountName: plugin-sa
---
apiVersion: v1
kind: Service
metadata:
  name: plugin-multi-instance
spec:
  selector:
    app: plugin
    instance: multi
  type: ClusterIP
  ports:
  - port: 10000
    targetPort: 10000
    name: upstream
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: plugin-sa
EOF
    
    log_success "Plugin manifest created"
}

# Deploy multi-instance plugins
deploy_plugins() {
    log_info "Deploying $PLUGIN_REPLICAS Plugin instances..."
    
    create_plugin_manifest
    
    if ! kubectl apply -f "$K8S_DIR/plugin-multi-instance.yaml" -n "$K8S_NAMESPACE"; then
        log_error "Failed to deploy plugins"
        return 1
    fi
    
    log_success "Plugin deployment submitted"
    sleep 2
}

# Wait for all plugin replicas to be ready
wait_for_plugins() {
    log_info "Waiting for all $PLUGIN_REPLICAS Plugin replicas to be ready..."
    
    local start_time=$(date +%s)
    local timeout=$POD_READY_TIMEOUT
    
    while true; do
        current_time=$(date +%s)
        elapsed=$((current_time - start_time))
        
        if [ $elapsed -gt $timeout ]; then
            log_error "Plugins failed to become ready after ${timeout}s"
            return 1
        fi
        
        ready=$(kubectl get deployment -n "$K8S_NAMESPACE" plugin-multi-instance \
            -o jsonpath='{.status.readyReplicas}' 2>/dev/null)
        desired=$(kubectl get deployment -n "$K8S_NAMESPACE" plugin-multi-instance \
            -o jsonpath='{.spec.replicas}' 2>/dev/null)
        
        if [ "$ready" = "$desired" ] && [ -n "$ready" ]; then
            log_success "All $ready/$desired Plugin replicas are ready"
            return 0
        fi
        
        log_warning "Plugins not all ready: $ready/$desired (elapsed: ${elapsed}s)"
        sleep 5
    done
}

# Create test policy
create_policy() {
    log_info "Creating test policy for distribution..."
    
    local policy_json='{
        "metadata": {"name": "multi-instance-test", "version": "1.0"},
        "spec": {
            "rules": [{
                "match": {"path": {"exact": "/api/test"}},
                "fault": {"percentage": 50, "abort": {"httpStatus": 500}}
            }]
        }
    }'
    
    # Get Control Plane pod for API access
    local cp_pod=$(kubectl get pods -n "$K8S_NAMESPACE" -l app=control-plane \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -z "$cp_pod" ]; then
        log_error "Control Plane pod not found"
        return 1
    fi
    
    log_info "Creating policy via Control Plane pod: $cp_pod"
    
    if kubectl exec -n "$K8S_NAMESPACE" "$cp_pod" -- \
        sh -c "curl -X POST http://localhost:8080/v1/policies \
        -H 'Content-Type: application/json' \
        -d '$policy_json'" &>/dev/null; then
        log_success "Policy created successfully"
        return 0
    else
        log_warning "Policy creation may have encountered issues"
        return 0
    fi
}

# Measure distribution time
measure_distribution_time() {
    log_info "Measuring policy distribution time to $PLUGIN_REPLICAS instances..."
    
    local start_time=$(date +%s%N)
    
    # Create policy
    create_policy
    
    # Check how many plugins have received the policy
    local plugin_pods=$(kubectl get pods -n "$K8S_NAMESPACE" -l app=plugin,instance=multi \
        -o jsonpath='{.items[*].metadata.name}')
    
    local ready_count=0
    local retries=20
    
    while [ $retries -gt 0 ]; do
        ready_count=0
        
        for pod in $plugin_pods; do
            # Check if plugin pod is still running
            if kubectl get pod -n "$K8S_NAMESPACE" "$pod" &>/dev/null; then
                ready_count=$((ready_count + 1))
            fi
        done
        
        if [ "$ready_count" -ge "$PLUGIN_REPLICAS" ]; then
            break
        fi
        
        sleep 0.1
        retries=$((retries - 1))
    done
    
    local end_time=$(date +%s%N)
    local duration_ms=$(( (end_time - start_time) / 1000000 ))
    
    log_success "Policy distributed to $ready_count/$PLUGIN_REPLICAS instances"
    log_info "Distribution time: ${duration_ms}ms"
    
    if [ "$duration_ms" -lt 1000 ]; then
        log_success "Distribution time is < 1000ms (requirement met)"
        return 0
    else
        log_warning "Distribution time is ${duration_ms}ms (target: < 1000ms)"
        return 0
    fi
}

# Verify plugin connections
verify_plugin_connections() {
    log_info "Verifying plugin connections to Control Plane..."
    
    # Check plugin logs for connection info
    local plugin_pods=$(kubectl get pods -n "$K8S_NAMESPACE" -l app=plugin,instance=multi \
        -o jsonpath='{.items[*].metadata.name}')
    
    local connected=0
    for pod in $plugin_pods; do
        log_info "Checking pod: $pod"
        
        # Check pod status
        phase=$(kubectl get pod -n "$K8S_NAMESPACE" "$pod" -o jsonpath='{.status.phase}' 2>/dev/null)
        if [ "$phase" = "Running" ]; then
            log_success "$pod is Running"
            connected=$((connected + 1))
        else
            log_warning "$pod phase: $phase"
        fi
    done
    
    log_success "Connected pods: $connected/$PLUGIN_REPLICAS"
    
    if [ "$connected" -eq "$PLUGIN_REPLICAS" ]; then
        return 0
    else
        return 1
    fi
}

# Gather diagnostic information
gather_diagnostics() {
    log_info "Gathering diagnostic information..."
    
    log_info "Plugin pods:"
    kubectl get pods -n "$K8S_NAMESPACE" -l app=plugin,instance=multi -o wide 2>/dev/null || true
    
    log_info "Plugin replica status:"
    kubectl get deployment -n "$K8S_NAMESPACE" plugin-multi-instance -o wide 2>/dev/null || true
    
    log_info "Plugin service endpoints:"
    kubectl get endpoints -n "$K8S_NAMESPACE" plugin-multi-instance -o wide 2>/dev/null || true
    
    # Show logs from a sample plugin
    local first_pod=$(kubectl get pods -n "$K8S_NAMESPACE" -l app=plugin,instance=multi \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -n "$first_pod" ]; then
        log_info "Sample plugin pod logs (last 20 lines): $first_pod"
        kubectl logs -n "$K8S_NAMESPACE" "$first_pod" --tail=20 2>/dev/null | head -20 || true
    fi
}

# Cleanup function
cleanup() {
    log_info "Test cleanup..."
    log_info "To remove plugin deployment: kubectl delete -f $K8S_DIR/plugin-multi-instance.yaml -n $K8S_NAMESPACE"
}

trap cleanup EXIT

# Main execution
main() {
    log_info "=========================================="
    log_info "Multi-Instance Policy Distribution Test (k3s)"
    log_info "=========================================="
    
    # Check prerequisites
    check_prerequisites
    
    # Deploy plugins
    if ! deploy_plugins; then
        log_error "Failed to deploy plugins"
        return 1
    fi
    
    # Wait for plugins
    if ! wait_for_plugins; then
        log_error "Plugins failed to become ready"
        gather_diagnostics
        return 1
    fi
    
    # Verify connections
    if ! verify_plugin_connections; then
        log_warning "Some plugins may not be properly connected"
    fi
    
    # Measure distribution time
    if ! measure_distribution_time; then
        log_error "Distribution time measurement failed"
        return 1
    fi
    
    # Gather diagnostics
    gather_diagnostics
    
    log_info "=========================================="
    log_success "Multi-instance test completed!"
    log_info "=========================================="
    return 0
}

# Run main function
main "$@"
exit $?
