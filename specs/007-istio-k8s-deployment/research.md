# Research: Istio/K8s Multi-Pod Deployment

**Feature**: 007-istio-k8s-deployment  
**Date**: 2025-12-05

## Research Topics

### 1. Istio WasmPlugin CRD Configuration

**Decision**: Use Istio `WasmPlugin` CRD with `url` field pointing to OCI image or HTTP URL

**Rationale**: 
- Istio 1.24+ supports WasmPlugin CRD natively
- Can use OCI registry (ghcr.io, docker.io) or HTTP URL for plugin distribution
- `pluginConfig` field allows passing control plane address to plugin

**Alternatives Considered**:
- EnvoyFilter with inline Wasm config - More complex, less portable
- ConfigMap mounting - Doesn't work with sidecar injection model

**Implementation**:
```yaml
apiVersion: extensions.istio.io/v1alpha1
kind: WasmPlugin
metadata:
  name: boifi-fault-injection
  namespace: demo  # Target namespace
spec:
  selector:
    matchLabels:
      # Empty = all pods in namespace
  url: oci://ghcr.io/boifi/wasm-plugin:latest  # Or http://...
  phase: AUTHN  # Run early in filter chain
  failStrategy: FAIL_OPEN  # Don't block traffic on plugin errors
  pluginConfig:
    control_plane_address: "hfi-control-plane.boifi.svc.cluster.local:8080"
```

### 2. Envoy Node Metadata for Service Identity

**Decision**: Extract `WORKLOAD_NAME` and `NAMESPACE` from Envoy bootstrap node metadata

**Rationale**:
- Istio injects this metadata into every sidecar
- Available via `proxy_wasm::hostcalls::get_property(vec!["node", "metadata", "WORKLOAD_NAME"])`
- No additional configuration needed

**Verification** (from live cluster):
```json
{
  "id": "sidecar~10.42.0.246~frontend-5bbd7f4cf9-kw8fb.demo~demo.svc.cluster.local",
  "cluster": "frontend.demo",
  "metadata": {
    "WORKLOAD_NAME": "frontend",
    "NAMESPACE": "demo",
    "LABELS": {
      "app": "frontend",
      "service.istio.io/canonical-name": "frontend"
    }
  }
}
```

**Implementation in Rust**:
```rust
fn get_service_identity() -> (String, String) {
    let workload = get_property(vec!["node", "metadata", "WORKLOAD_NAME"])
        .and_then(|v| String::from_utf8(v).ok())
        .unwrap_or_else(|| "*".to_string());
    
    let namespace = get_property(vec!["node", "metadata", "NAMESPACE"])
        .and_then(|v| String::from_utf8(v).ok())
        .unwrap_or_else(|| "*".to_string());
    
    (workload, namespace)
}
```

### 3. Service Selector Matching Logic

**Decision**: Add `selector` field to Policy with `service` and `namespace` matching

**Rationale**:
- Simple glob/exact matching is sufficient for MVP
- Wildcard `*` matches all services/namespaces
- Matches Kubernetes label selector patterns users already know

**Matching Rules**:
1. `selector.service: "*"` → matches all services
2. `selector.service: "frontend"` → exact match only
3. `selector.namespace: "demo"` → matches only pods in demo namespace
4. Empty selector → treated as `*` (matches all)

**Policy Schema Update**:
```yaml
metadata:
  name: "frontend-fault-policy"
spec:
  selector:                    # NEW FIELD
    service: "frontend"        # Match specific service
    namespace: "demo"          # Match specific namespace
  rules:
    - match:
        method:
          exact: "GET"
      fault:
        percentage: 50
        abort:
          httpStatus: 503
```

### 4. Control Plane Service Discovery

**Decision**: Use Kubernetes ClusterIP service with stable DNS name

**Rationale**:
- Control plane already deployed as `hfi-control-plane.boifi.svc.cluster.local`
- DNS resolution works across namespaces by default
- No special NetworkPolicy needed (tested)

**Configuration**:
- Control plane service: `hfi-control-plane.boifi.svc.cluster.local:8080`
- Wasm plugin connects via HTTP to `/v1/policies`
- SSE connection for real-time updates

### 5. Plugin Distribution Strategy

**Decision**: Build and push OCI image to GitHub Container Registry (ghcr.io)

**Rationale**:
- OCI is the Istio-preferred distribution method
- Works with image pull secrets if private
- Can use `imagePullPolicy: Always` for development

**Build Process**:
```bash
# Build wasm binary
cargo build --target wasm32-unknown-unknown --release

# Create OCI image (using crane or docker)
crane push target/wasm32-unknown-unknown/release/hfi_wasm_plugin.wasm \
  ghcr.io/boifi/wasm-plugin:latest
```

**Alternative for local development**:
- Use HTTP server to serve wasm file: `url: http://wasm-server.boifi:8080/plugin.wasm`

### 6. Multi-Pod Percentage Distribution

**Decision**: Each pod independently applies percentage - aggregate rate equals configured rate

**Rationale**:
- With random distribution of requests across pods, independent 30% on each pod = 30% overall
- No coordination needed between pods
- Simpler implementation, scales naturally

**Mathematical Proof**:
- 3 pods, each with 30% fault rate
- Request randomly routed to one pod
- P(fault) = P(pod1)*0.3 + P(pod2)*0.3 + P(pod3)*0.3 = (1/3)*0.3 + (1/3)*0.3 + (1/3)*0.3 = 0.3

**Note**: This assumes load balancer distributes evenly. Sticky sessions could skew results.

## Resolved Clarifications

| Topic | Resolution |
|-------|------------|
| How to identify service in Wasm? | Use Envoy node metadata WORKLOAD_NAME |
| How to deploy plugin to Istio? | Use WasmPlugin CRD |
| How to distribute plugin binary? | OCI image to ghcr.io |
| How to handle multi-pod scenarios? | Independent percentage per pod |
| What if control plane unreachable? | Fail-open (passthrough mode) |

## Next Steps

1. Create `data-model.md` with ServiceSelector entity
2. Update `contracts/openapi.yaml` with selector fields
3. Create `quickstart.md` for Istio deployment
4. Generate tasks for implementation
