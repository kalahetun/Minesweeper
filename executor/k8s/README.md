# Kubernetes Deployment for HFI System

This directory contains Kubernetes manifests for deploying the complete HTTP Fault Injection (HFI) system in a Kubernetes cluster.

## 📦 Components

### 1. Control Plane (`control-plane.yaml`)
- **Deployment**: `hfi-control-plane` with 2 replicas
- **Service**: ClusterIP service exposing port 8080
- **Storage**: etcd deployment and service for persistent storage
- **Features**:
  - Health checks and readiness probes
  - Resource limits and requests
  - Environment variables for configuration

### 2. Envoy Configuration (`envoy-config.yaml`)
- **ConfigMap**: `hfi-envoy-config` containing Envoy configuration
- **Features**:
  - HTTP connection manager with Wasm filter
  - Wasm plugin configuration pointing to Control Plane service
  - Admin interface on port 9901

### 3. Sample Application (`sample-app-with-proxy.yaml`)
- **Deployment**: Sample application with Envoy sidecar
- **Containers**:
  - `httpbin`: Main application container
  - `envoy-proxy`: Sidecar proxy container
- **Init Container**: Copies Wasm plugin to shared volume
- **Services**: ClusterIP and NodePort for external access

## 🚀 Quick Start

### Prerequisites
- Kubernetes cluster (v1.20+)
- `kubectl` configured to access your cluster
- Container images built and available:
  - `hfi/control-plane:latest`
  - `hfi/wasm-plugin:latest`

### Deploy the System
```bash
# Navigate to k8s directory
cd k8s/

# Deploy all components
./deploy.sh
```

### Manual Deployment
```bash
# 1. Deploy Control Plane and etcd
kubectl apply -f control-plane.yaml

# 2. Deploy Envoy configuration
kubectl apply -f envoy-config.yaml

# 3. Deploy sample application
kubectl apply -f sample-app-with-proxy.yaml
```

### Cleanup
```bash
# Remove all components
./cleanup.sh
```

## 🔍 Verification

### Check Deployment Status
```bash
# Check all pods
kubectl get pods -l component in \(control-plane,storage,demo\)

# Check services
kubectl get svc -l app in \(hfi-control-plane,hfi-etcd,sample-app\)

# Check config maps
kubectl get configmap hfi-envoy-config
```

### View Logs
```bash
# Control Plane logs
kubectl logs -l app=hfi-control-plane

# Sample application logs
kubectl logs -l app=sample-app -c httpbin

# Envoy proxy logs
kubectl logs -l app=sample-app -c envoy-proxy
```

## 🌐 Access Services

### Internal Access (within cluster)
- **Control Plane API**: `http://hfi-control-plane.default.svc.cluster.local:8080`
- **Sample App (via Envoy)**: `http://sample-app-service.default.svc.cluster.local:8000`
- **Envoy Admin**: `http://sample-app-service.default.svc.cluster.local:9901`

### External Access (NodePort)
- **Sample App**: `http://<node-ip>:30080`
- **Envoy Admin**: `http://<node-ip>:30901`

Get node IP:
```bash
kubectl get nodes -o wide
```

## 🧪 Testing the System

### 1. Apply a Fault Injection Policy
```bash
# Port forward to Control Plane (if needed)
kubectl port-forward svc/hfi-control-plane 8080:8080 &

# Use the CLI tool to apply a policy
cd ../cli
./hficli policy apply -f ../examples/delay-policy.yaml
```

### 2. Test Fault Injection
```bash
# Port forward to sample app
kubectl port-forward svc/sample-app-service 8000:8000 &

# Send test requests
curl http://localhost:8000/get
curl http://localhost:8000/status/200
```

### 3. Monitor Envoy Admin Interface
```bash
# Port forward to Envoy admin
kubectl port-forward svc/sample-app-service 9901:9901 &

# Open in browser or curl
curl http://localhost:9901/stats | grep hfi
```

## 🔧 Configuration

### Environment Variables (Control Plane)
- `STORAGE_BACKEND`: Storage backend type (`etcd` or `memory`)
- `ETCD_ENDPOINTS`: etcd server endpoints
- `LOG_LEVEL`: Logging level

### Envoy Configuration
- Modify `envoy-config.yaml` to change Envoy behavior
- Update Wasm plugin configuration in the ConfigMap
- Adjust cluster configuration for your backend services

### Resource Limits
Adjust resource requests and limits in the deployments based on your cluster capacity:
```yaml
resources:
  requests:
    memory: "64Mi"
    cpu: "50m"
  limits:
    memory: "128Mi"
    cpu: "200m"
```

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐
│   CLI Tool      │    │  Control Plane  │
│                 │────│  (Deployment)   │
│                 │    │                 │
└─────────────────┘    └─────────────────┘
                                │
                                │ HTTP API
                                │
┌─────────────────┐    ┌─────────────────┐
│   etcd          │────│  Sample App     │
│  (Storage)      │    │  + Envoy Proxy  │
│                 │    │  (Sidecar)      │
└─────────────────┘    └─────────────────┘
```

## 🐛 Troubleshooting

### Common Issues

1. **Pods not starting**
   - Check container images are available
   - Verify resource limits are appropriate
   - Check node capacity

2. **Wasm plugin not loading**
   - Verify init container copied the plugin successfully
   - Check Envoy logs for plugin errors
   - Ensure volume mounts are correct

3. **Control Plane connectivity issues**
   - Verify service names and ports in Envoy config
   - Check network policies if any
   - Ensure DNS resolution works

### Debug Commands
```bash
# Describe resources for detailed info
kubectl describe pod <pod-name>
kubectl describe deployment <deployment-name>

# Get events
kubectl get events --sort-by=.metadata.creationTimestamp

# Port forward for direct access
kubectl port-forward pod/<pod-name> <local-port>:<container-port>
```

## 📚 Additional Resources

- [Envoy Proxy Documentation](https://www.envoyproxy.io/docs)
- [Kubernetes Documentation](https://kubernetes.io/docs)
- [Wasm in Envoy](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/wasm_filter)
