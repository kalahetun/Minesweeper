# Quickstart: BOIFI Executor on Kubernetes with Istio

**Feature**: 007-istio-k8s-deployment  
**Time to complete**: ~15 minutes

## Prerequisites

- Kubernetes cluster (k3s, k8s, or minikube)
- Istio 1.20+ installed
- kubectl configured
- Docker registry access (for wasm plugin image)

```bash
# Verify Istio installation
istioctl version

# Verify cluster
kubectl cluster-info
```

## Step 1: Deploy Control Plane

```bash
# Create namespace
kubectl create namespace boifi

# Deploy control plane with etcd
kubectl apply -f executor/k8s/control-plane.yaml

# Wait for ready
kubectl wait --for=condition=ready pod -l app=control-plane -n boifi --timeout=120s

# Verify
kubectl get pods -n boifi
# NAME                             READY   STATUS    RESTARTS   AGE
# control-plane-xxxxxxxxx-xxxxx    1/1     Running   0          1m
# etcd-xxxxxxxxx-xxxxx             1/1     Running   0          1m
```

## Step 2: Build and Push Wasm Plugin

```bash
cd executor/wasm-plugin

# Build wasm module
make build

# Tag and push to registry (adjust registry URL)
docker buildx build --platform wasi/wasm \
  -t your-registry.io/boifi-wasm-plugin:v1 \
  --push .

# Or for local testing with k3s
docker save your-registry.io/boifi-wasm-plugin:v1 | \
  k3s ctr images import -
```

## Step 3: Deploy WasmPlugin to Istio

Create `wasmplugin.yaml`:

```yaml
apiVersion: extensions.istio.io/v1alpha1
kind: WasmPlugin
metadata:
  name: boifi-fault-injection
  namespace: demo  # Target namespace
spec:
  selector:
    matchLabels:
      # Apply to all pods in namespace, or be more specific
      # app: frontend
  url: oci://your-registry.io/boifi-wasm-plugin:v1
  phase: AUTHN  # Run early in filter chain
  failStrategy: FAIL_OPEN  # Don't break traffic on plugin failure
  pluginConfig:
    control_plane_address: "http://control-plane.boifi.svc.cluster.local:8080"
    poll_interval_ms: 5000
```

Apply:
```bash
kubectl apply -f wasmplugin.yaml

# Verify
kubectl get wasmplugins -n demo
```

## Step 4: Create a Test Policy

Create a policy targeting a specific service:

```yaml
# frontend-delay.yaml
metadata:
  name: frontend-delay
spec:
  selector:
    service: frontend
    namespace: demo
  rules:
    - match:
        path_prefix: "/"
      fault:
        delay_ms: 500
        percentage: 50
```

Apply via CLI:
```bash
# Port-forward to control plane
kubectl port-forward -n boifi svc/control-plane 8080:8080 &

# Apply policy
./hfi-cli policy apply -f frontend-delay.yaml

# List policies
./hfi-cli policy list
```

## Step 5: Verify Fault Injection

```bash
# Get frontend pod
FRONTEND_POD=$(kubectl get pod -n demo -l app=frontend -o jsonpath='{.items[0].metadata.name}')

# Send test requests and observe delays
for i in {1..10}; do
  time kubectl exec -n demo $FRONTEND_POD -c istio-proxy -- \
    curl -s -w "\n" localhost:15001/test
done

# Check Envoy stats for fault injection
kubectl exec -n demo $FRONTEND_POD -c istio-proxy -- \
  curl -s localhost:15000/stats | grep -i fault
```

## Step 6: Service-Specific Policy Example

Create policies that target different services:

```yaml
# checkout-abort.yaml - Only affects checkoutservice
metadata:
  name: checkout-abort
spec:
  selector:
    service: checkoutservice
    namespace: demo
  rules:
    - match:
        path_prefix: "/hipstershop.CheckoutService/"
      fault:
        abort_code: 503
        percentage: 25
---
# payment-delay.yaml - Only affects paymentservice  
metadata:
  name: payment-delay
spec:
  selector:
    service: paymentservice
    namespace: demo
  rules:
    - fault:
        delay_ms: 2000
        percentage: 10
```

```bash
./hfi-cli policy apply -f checkout-abort.yaml
./hfi-cli policy apply -f payment-delay.yaml

# Verify policies
./hfi-cli policy list
# NAME              SERVICE           NAMESPACE   STATUS
# checkout-abort    checkoutservice   demo        active
# payment-delay     paymentservice    demo        active
```

## Troubleshooting

### Plugin not loading

```bash
# Check WasmPlugin status
kubectl describe wasmplugin boifi-fault-injection -n demo

# Check Envoy logs
kubectl logs -n demo $POD -c istio-proxy | grep -i wasm
```

### Policy not applied

```bash
# Check control plane logs
kubectl logs -n boifi -l app=control-plane

# Verify plugin can reach control plane
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s http://control-plane.boifi.svc.cluster.local:8080/health
```

### Check Envoy identity metadata

```bash
# Get node metadata (verify WORKLOAD_NAME/NAMESPACE)
kubectl exec -n demo $POD -c istio-proxy -- \
  curl -s localhost:15000/server_info | jq '.node.metadata'
```

## Clean Up

```bash
# Remove policies
./hfi-cli policy delete frontend-delay
./hfi-cli policy delete checkout-abort
./hfi-cli policy delete payment-delay

# Remove WasmPlugin
kubectl delete wasmplugin boifi-fault-injection -n demo

# Remove control plane
kubectl delete -f executor/k8s/control-plane.yaml
kubectl delete namespace boifi
```

## Next Steps

- See [spec.md](spec.md) for detailed user stories
- See [data-model.md](data-model.md) for policy schema
- See [research.md](research.md) for technical decisions
