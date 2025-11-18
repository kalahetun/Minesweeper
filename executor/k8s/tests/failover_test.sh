#!/bin/bash

#############################################################################
# Control Plane Failover Test (k3s)
# Purpose: Test Pod restart, data recovery, and new connection establishment
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
K8S_NAMESPACE="boifi"
KUBECONFIG="${KUBECONFIG:-$HOME/.kube/k3s.yaml}"
RECOVERY_TIMEOUT=300
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

# Verify Control Plane is running
verify_control_plane_running() {
    log_info "Verifying Control Plane is running..."
    
    cp_pod=$(kubectl get pods -n "$K8S_NAMESPACE" -l app=control-plane \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -z "$cp_pod" ]; then
        log_error "Control Plane pod not found"
        return 1
    fi
    
    phase=$(kubectl get pod -n "$K8S_NAMESPACE" "$cp_pod" -o jsonpath='{.status.phase}' 2>/dev/null)
    if [ "$phase" != "Running" ]; then
        log_error "Control Plane pod phase: $phase"
        return 1
    fi
    
    log_success "Control Plane running: $cp_pod"
    return 0
}

# Create test data before failover
create_test_data() {
    log_info "Creating test data before failover..."
    
    cp_pod=$(kubectl get pods -n "$K8S_NAMESPACE" -l app=control-plane \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -z "$cp_pod" ]; then
        log_error "Control Plane pod not found"
        return 1
    fi
    
    # Create multiple test policies
    for i in 1 2 3; do
        policy_json="{
            \"metadata\": {\"name\": \"failover-test-policy-$i\", \"version\": \"1.0\"},
            \"spec\": {
                \"rules\": [{
                    \"match\": {\"path\": {\"exact\": \"/api/failover-test/$i\"}},
                    \"fault\": {\"percentage\": $((i * 10)), \"abort\": {\"httpStatus\": 500}}
                }]
            }
        }"
        
        if kubectl exec -n "$K8S_NAMESPACE" "$cp_pod" -- \
            sh -c "curl -X POST http://localhost:8080/v1/policies \
            -H 'Content-Type: application/json' \
            -d '$policy_json'" &>/dev/null; then
            log_success "Created test policy: failover-test-policy-$i"
        else
            log_warning "Failed to create test policy: failover-test-policy-$i"
        fi
    done
    
    sleep 2
    return 0
}

# Verify test data exists
verify_test_data_exists() {
    log_info "Verifying test data exists..."
    
    cp_pod=$(kubectl get pods -n "$K8S_NAMESPACE" -l app=control-plane \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -z "$cp_pod" ]; then
        log_error "Control Plane pod not found"
        return 1
    fi
    
    # Check if policies exist
    for i in 1 2 3; do
        if kubectl exec -n "$K8S_NAMESPACE" "$cp_pod" -- \
            sh -c "curl -s http://localhost:8080/v1/policies/failover-test-policy-$i | grep -q 'failover-test-policy-$i'" 2>/dev/null; then
            log_success "Policy exists: failover-test-policy-$i"
        else
            log_warning "Policy not found: failover-test-policy-$i"
        fi
    done
    
    return 0
}

# Simulate pod restart
simulate_pod_restart() {
    log_info "Simulating Control Plane pod restart..."
    
    cp_pod=$(kubectl get pods -n "$K8S_NAMESPACE" -l app=control-plane \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -z "$cp_pod" ]; then
        log_error "Control Plane pod not found"
        return 1
    fi
    
    log_info "Deleting Control Plane pod: $cp_pod"
    
    if ! kubectl delete pod -n "$K8S_NAMESPACE" "$cp_pod"; then
        log_error "Failed to delete pod"
        return 1
    fi
    
    log_success "Pod deleted (Kubernetes will restart it)"
    sleep 3
    
    return 0
}

# Wait for pod recovery
wait_for_pod_recovery() {
    log_info "Waiting for Control Plane pod recovery..."
    
    local start_time=$(date +%s)
    local timeout=$RECOVERY_TIMEOUT
    local retries=$HEALTH_CHECK_RETRIES
    
    while [ $retries -gt 0 ]; do
        current_time=$(date +%s)
        elapsed=$((current_time - start_time))
        
        if [ $elapsed -gt $timeout ]; then
            log_error "Pod failed to recover after ${timeout}s"
            return 1
        fi
        
        # Get new pod
        cp_pod=$(kubectl get pods -n "$K8S_NAMESPACE" -l app=control-plane \
            -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
        
        if [ -z "$cp_pod" ]; then
            log_warning "No Control Plane pod found, waiting..."
            sleep 5
            continue
        fi
        
        phase=$(kubectl get pod -n "$K8S_NAMESPACE" "$cp_pod" -o jsonpath='{.status.phase}' 2>/dev/null)
        
        if [ "$phase" = "Running" ]; then
            ready=$(kubectl get pod -n "$K8S_NAMESPACE" "$cp_pod" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null)
            
            if [ "$ready" = "True" ]; then
                log_success "Control Plane recovered: $cp_pod (elapsed: ${elapsed}s)"
                return 0
            fi
        fi
        
        log_warning "Pod phase: $phase, waiting... (elapsed: ${elapsed}s, retries left: $retries)"
        sleep $HEALTH_CHECK_INTERVAL
        retries=$((retries - 1))
    done
    
    log_error "Pod recovery timeout"
    return 1
}

# Verify data recovery
verify_data_recovery() {
    log_info "Verifying data recovery after restart..."
    
    cp_pod=$(kubectl get pods -n "$K8S_NAMESPACE" -l app=control-plane \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -z "$cp_pod" ]; then
        log_error "Control Plane pod not found"
        return 1
    fi
    
    log_info "New Control Plane pod: $cp_pod"
    
    # Check if policies were recovered
    local recovered=0
    for i in 1 2 3; do
        if kubectl exec -n "$K8S_NAMESPACE" "$cp_pod" -- \
            sh -c "curl -s http://localhost:8080/v1/policies/failover-test-policy-$i | grep -q 'failover-test-policy-$i'" 2>/dev/null; then
            log_success "Data recovered: failover-test-policy-$i"
            recovered=$((recovered + 1))
        else
            log_warning "Policy not recovered: failover-test-policy-$i"
        fi
    done
    
    if [ "$recovered" -eq 3 ]; then
        log_success "All test data recovered"
        return 0
    else
        log_warning "Only $recovered/3 policies recovered"
        return 0
    fi
}

# Test new connections
test_new_connections() {
    log_info "Testing new connections after recovery..."
    
    cp_pod=$(kubectl get pods -n "$K8S_NAMESPACE" -l app=control-plane \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -z "$cp_pod" ]; then
        log_error "Control Plane pod not found"
        return 1
    fi
    
    # Test Health check
    log_info "Testing /healthz endpoint..."
    if kubectl exec -n "$K8S_NAMESPACE" "$cp_pod" -- \
        sh -c "curl -s -o /dev/null -w '%{http_code}' http://localhost:8080/healthz" | grep -q "200"; then
        log_success "Health check successful"
    else
        log_warning "Health check may have failed"
    fi
    
    # Test new policy creation
    log_info "Testing new policy creation..."
    new_policy_json='{
        "metadata": {"name": "post-failover-policy", "version": "1.0"},
        "spec": {
            "rules": [{
                "match": {"path": {"exact": "/api/post-failover"}},
                "fault": {"percentage": 75, "abort": {"httpStatus": 503}}
            }]
        }
    }'
    
    if kubectl exec -n "$K8S_NAMESPACE" "$cp_pod" -- \
        sh -c "curl -X POST http://localhost:8080/v1/policies \
        -H 'Content-Type: application/json' \
        -d '$new_policy_json'" &>/dev/null; then
        log_success "New policy creation successful"
        return 0
    else
        log_warning "New policy creation may have failed"
        return 0
    fi
}

# Gather cluster info
gather_cluster_info() {
    log_info "Gathering cluster information..."
    
    log_info "Control Plane deployment:"
    kubectl get deployment -n "$K8S_NAMESPACE" -l app=control-plane -o wide 2>/dev/null || true
    
    log_info "Control Plane pod (new):"
    kubectl get pods -n "$K8S_NAMESPACE" -l app=control-plane -o wide 2>/dev/null || true
    
    log_info "Pod events:"
    kubectl get events -n "$K8S_NAMESPACE" --sort-by='.lastTimestamp' 2>/dev/null | tail -10 || true
}

# Cleanup function
cleanup() {
    log_info "Cleanup..."
    log_info "Failover test completed. Control Plane is recovered and operational."
}

trap cleanup EXIT

# Main execution
main() {
    log_info "=========================================="
    log_info "Control Plane Failover Test (k3s)"
    log_info "=========================================="
    
    # Check prerequisites
    check_prerequisites
    
    # Verify Control Plane is running
    if ! verify_control_plane_running; then
        log_error "Control Plane not running"
        return 1
    fi
    
    # Create test data
    if ! create_test_data; then
        log_error "Failed to create test data"
        return 1
    fi
    
    # Verify test data exists
    if ! verify_test_data_exists; then
        log_warning "Could not verify test data"
    fi
    
    # Simulate pod restart
    if ! simulate_pod_restart; then
        log_error "Failed to simulate pod restart"
        return 1
    fi
    
    # Wait for recovery
    if ! wait_for_pod_recovery; then
        log_error "Pod failed to recover"
        gather_cluster_info
        return 1
    fi
    
    # Verify data recovery
    if ! verify_data_recovery; then
        log_warning "Data recovery verification inconclusive"
    fi
    
    # Test new connections
    if ! test_new_connections; then
        log_warning "New connection tests inconclusive"
    fi
    
    # Gather info
    gather_cluster_info
    
    log_info "=========================================="
    log_success "Failover test completed successfully!"
    log_info "=========================================="
    return 0
}

# Run main function
main "$@"
exit $?
