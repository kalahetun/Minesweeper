#!/bin/bash

# Kubernetes cleanup script for HFI (HTTP Fault Injection) system
set -e

NAMESPACE="default"
KUBECTL_CMD="kubectl"

echo "🧹 Cleaning up HFI system from Kubernetes..."

# Function to safely delete resources
safe_delete() {
    local resource_type=$1
    local resource_name=$2
    local namespace=${3:-default}
    
    if $KUBECTL_CMD get $resource_type $resource_name -n $namespace >/dev/null 2>&1; then
        echo "🗑️  Deleting $resource_type/$resource_name..."
        $KUBECTL_CMD delete $resource_type $resource_name -n $namespace
    else
        echo "ℹ️  $resource_type/$resource_name not found, skipping..."
    fi
}

# Delete in reverse order of creation
echo "📦 Step 1: Removing sample application..."
safe_delete "deployment" "sample-app-with-proxy" $NAMESPACE
safe_delete "service" "sample-app-service" $NAMESPACE
safe_delete "service" "sample-app-nodeport" $NAMESPACE

echo "📦 Step 2: Removing Envoy configuration..."
safe_delete "configmap" "hfi-envoy-config" $NAMESPACE

echo "📦 Step 3: Removing Control Plane and etcd..."
safe_delete "deployment" "hfi-control-plane" $NAMESPACE
safe_delete "service" "hfi-control-plane" $NAMESPACE
safe_delete "deployment" "hfi-etcd" $NAMESPACE
safe_delete "service" "hfi-etcd" $NAMESPACE

echo ""
echo "🎉 HFI system cleanup completed!"
echo ""
echo "🔍 Verify cleanup:"
echo "  kubectl get pods,svc,configmap -l component in (control-plane,storage,demo,proxy)"
