# Data Model: Istio/K8s Multi-Pod Deployment

**Feature**: 007-istio-k8s-deployment  
**Date**: 2025-12-05

## Entity Relationship Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     FaultInjectionPolicy                         │
├─────────────────────────────────────────────────────────────────┤
│ metadata:                                                        │
│   name: string (required, unique)                                │
│   version: string (optional)                                     │
│                                                                  │
│ spec:                                                            │
│   selector: ServiceSelector (NEW)                                │
│   rules: []FaultRule                                             │
└────────────────────┬────────────────────────────────────────────┘
                     │
                     │ 1:1
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                     ServiceSelector (NEW)                        │
├─────────────────────────────────────────────────────────────────┤
│ service: string                                                  │
│   - "*" = match all services (default)                          │
│   - "frontend" = exact match                                     │
│                                                                  │
│ namespace: string                                                │
│   - "*" = match all namespaces (default)                        │
│   - "demo" = exact match                                         │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                     EnvoyIdentity (Runtime)                      │
├─────────────────────────────────────────────────────────────────┤
│ workload_name: string (from WORKLOAD_NAME)                       │
│ namespace: string (from NAMESPACE)                               │
│ pod_name: string (from NAME, optional)                           │
│ cluster: string (from node.cluster, optional)                    │
└─────────────────────────────────────────────────────────────────┘
```

## Entity Definitions

### ServiceSelector (NEW)

Specifies which services a policy applies to. Added to `FaultInjectionPolicy.spec`.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `service` | string | No | `"*"` | Service name to match. `"*"` matches all services. |
| `namespace` | string | No | `"*"` | Namespace to match. `"*"` matches all namespaces. |

**Validation Rules**:
- Empty string treated as `"*"` (wildcard)
- Case-sensitive matching
- No regex support in v1 (KISS principle)

**Matching Logic**:
```
matches(policy, envoy_identity) =
    (policy.selector.service == "*" OR policy.selector.service == envoy_identity.workload_name)
    AND
    (policy.selector.namespace == "*" OR policy.selector.namespace == envoy_identity.namespace)
```

### EnvoyIdentity (Runtime Entity)

Represents the identity of an Envoy sidecar instance. Extracted at plugin startup.

| Field | Type | Source | Description |
|-------|------|--------|-------------|
| `workload_name` | string | `node.metadata.WORKLOAD_NAME` | Kubernetes workload name (e.g., "frontend") |
| `namespace` | string | `node.metadata.NAMESPACE` | Kubernetes namespace (e.g., "demo") |
| `pod_name` | string | `node.metadata.NAME` | Pod name (for logging) |
| `cluster` | string | `node.cluster` | Istio cluster ID (e.g., "frontend.demo") |

**Extraction in Rust**:
```rust
struct EnvoyIdentity {
    workload_name: String,
    namespace: String,
    pod_name: Option<String>,
}

impl EnvoyIdentity {
    fn from_envoy_metadata() -> Self {
        Self {
            workload_name: get_property(vec!["node", "metadata", "WORKLOAD_NAME"])
                .and_then(|v| String::from_utf8(v).ok())
                .unwrap_or_else(|| "*".to_string()),
            namespace: get_property(vec!["node", "metadata", "NAMESPACE"])
                .and_then(|v| String::from_utf8(v).ok())
                .unwrap_or_else(|| "*".to_string()),
            pod_name: get_property(vec!["node", "metadata", "NAME"])
                .and_then(|v| String::from_utf8(v).ok()),
        }
    }
    
    fn matches_selector(&self, selector: &ServiceSelector) -> bool {
        let service_matches = selector.service == "*" || selector.service == self.workload_name;
        let namespace_matches = selector.namespace == "*" || selector.namespace == self.namespace;
        service_matches && namespace_matches
    }
}
```

### Updated FaultInjectionPolicy

Existing entity with new `selector` field.

```yaml
# Before (applies to all Envoys)
metadata:
  name: "test-policy"
spec:
  rules:
    - match: ...
      fault: ...

# After (applies to specific service)
metadata:
  name: "frontend-policy"
spec:
  selector:                    # NEW
    service: "frontend"
    namespace: "demo"
  rules:
    - match: ...
      fault: ...
```

## State Transitions

### Plugin Lifecycle

```
┌─────────────┐     on_configure()      ┌─────────────────┐
│   Created   │ ───────────────────────▶│ Identity Loaded │
└─────────────┘                         └────────┬────────┘
                                                 │
                                                 │ fetch policies
                                                 ▼
┌─────────────┐     on_http_call_response   ┌─────────────────┐
│   Running   │ ◀───────────────────────────│ Policies Loaded │
└─────────────┘                             └─────────────────┘
       │
       │ on_http_request_headers
       ▼
┌─────────────────────────────────────────────────────────────┐
│ For each incoming request:                                   │
│   1. Get matching policies for this service identity         │
│   2. Apply fault injection rules                             │
│   3. Continue or abort request                               │
└─────────────────────────────────────────────────────────────┘
```

### Policy Matching Flow

```
Request arrives at Envoy sidecar
        │
        ▼
┌───────────────────────────┐
│ Get EnvoyIdentity         │
│ (workload=frontend,       │
│  namespace=demo)          │
└───────────┬───────────────┘
            │
            ▼
┌───────────────────────────┐
│ Filter policies by        │
│ selector match            │
│ (only policies targeting  │
│  frontend or *)           │
└───────────┬───────────────┘
            │
            ▼
┌───────────────────────────┐
│ Apply first matching rule │
│ (existing logic)          │
└───────────────────────────┘
```

## Backward Compatibility

| Scenario | Behavior |
|----------|----------|
| Policy without `selector` field | Treated as `selector: {service: "*", namespace: "*"}` |
| Plugin receiving old-format policy | Works as before (applies to all) |
| Old plugin receiving new-format policy | Ignores `selector` field, applies to all |

## JSON Schema Changes

```json
{
  "type": "object",
  "properties": {
    "metadata": { ... },
    "spec": {
      "type": "object",
      "properties": {
        "selector": {
          "type": "object",
          "properties": {
            "service": {
              "type": "string",
              "default": "*",
              "description": "Service name to match, or * for all"
            },
            "namespace": {
              "type": "string", 
              "default": "*",
              "description": "Namespace to match, or * for all"
            }
          }
        },
        "rules": { ... }
      }
    }
  }
}
```
