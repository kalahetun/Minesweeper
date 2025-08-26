#!/bin/bash

# Kubernetes deployment script for HFI (HTTP Fault Injection) system
set -e

NAMESPACE="default"
KUBECTL_CMD="kubectl"

echo "🚀 Deploying HFI system to Kubernetes..."

# Function to wait for deployment to be ready
wait_for_deployment() {
    local deployment=$1
    local namespace=${2:-default}
    echo "⏳ Waiting for deployment/$deployment to be ready..."
    $KUBECTL_CMD wait --for=condition=available --timeout=300s deployment/$deployment -n $namespace
}

# Function to wait for pod to be ready
wait_for_pod() {
    local label_selector=$1
    local namespace=${2:-default}
    echo "⏳ Waiting for pods with selector '$label_selector' to be ready..."
    $KUBECTL_CMD wait --for=condition=ready --timeout=300s pod -l $label_selector -n $namespace
}

# Step 1: Deploy Control Plane and etcd
echo "📦 Step 1: Deploying Control Plane and etcd..."
$KUBECTL_CMD apply -f control-plane.yaml

# Wait for etcd to be ready first
wait_for_deployment "hfi-etcd" $NAMESPACE
echo "✅ etcd is ready"

# Wait for control plane to be ready
wait_for_deployment "hfi-control-plane" $NAMESPACE
echo "✅ Control Plane is ready"

# Step 2: Deploy Envoy configuration
echo "📦 Step 2: Deploying Envoy configuration..."
$KUBECTL_CMD apply -f envoy-config.yaml
echo "✅ Envoy configuration deployed"

# Step 3: Deploy sample application with Envoy sidecar
echo "📦 Step 3: Deploying sample application with Envoy sidecar..."
$KUBECTL_CMD apply -f sample-app-with-proxy.yaml

# Wait for sample app to be ready
wait_for_deployment "sample-app-with-proxy" $NAMESPACE
echo "✅ Sample application with proxy is ready"

echo ""
echo "🎉 HFI system deployment completed!"
echo ""
echo "📋 Deployment Summary:"
echo "  • Control Plane: http://hfi-control-plane.default.svc.cluster.local:8080"
echo "  • etcd Storage: http://hfi-etcd.default.svc.cluster.local:2379"
echo "  • Sample App (via Envoy): http://sample-app-service.default.svc.cluster.local:8000"
echo "  • Envoy Admin: http://sample-app-service.default.svc.cluster.local:9901"
echo ""
echo "🔍 Useful commands:"
echo "  • Check pods: kubectl get pods -l component in (control-plane,storage,demo)"
echo "  • Check services: kubectl get svc -l app in (hfi-control-plane,hfi-etcd,sample-app)"
echo "  • View Control Plane logs: kubectl logs -l app=hfi-control-plane"
echo "  • View Sample App logs: kubectl logs -l app=sample-app -c httpbin"
echo "  • View Envoy logs: kubectl logs -l app=sample-app -c envoy-proxy"
echo ""
echo "🌐 External access (if using NodePort):"
echo "  • Sample App: http://<node-ip>:30080"
echo "  • Envoy Admin: http://<node-ip>:30901"
echo ""
echo "🧪 Test the system:"
echo "  • Apply a fault injection policy using the CLI tool"
echo "  • Send requests to the sample app and observe fault injection"
