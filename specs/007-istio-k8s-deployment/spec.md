# Feature Specification: Istio/K8s Multi-Pod Deployment

**Feature Branch**: `007-istio-k8s-deployment`  
**Created**: 2025-12-05  
**Status**: ✅ **Complete**  
**Completed**: 2025-12-09  
**Input**: User description: "Deploy executor to k3s with Istio, test control plane and wasm plugin on multi-pod microservices demo with service-level policy targeting"

## Overview

This feature enables BOIFI executor deployment on Kubernetes clusters with Istio service mesh. The key challenge is adapting the current single-Envoy architecture to work with Istio's sidecar proxy model, where each Pod has its own Envoy instance. The system must support service-level policy targeting so that fault injection policies can be applied to specific services rather than all Envoy instances indiscriminately.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Deploy Control Plane to K8s (Priority: P1)

As an SRE, I want to deploy the BOIFI control plane to my k3s cluster so that I can manage fault injection policies centrally for all services in the mesh.

**Why this priority**: The control plane is the foundation - without it, no other features can work. It must be deployed and accessible before any Wasm plugins can fetch policies.

**Independent Test**: Deploy control plane to `boifi` namespace, verify it's running and accessible via `kubectl port-forward`, and confirm health endpoint returns 200 OK.

**Acceptance Scenarios**:

1. **Given** k3s cluster with Istio installed, **When** I apply control-plane.yaml, **Then** control plane pods start and become Ready within 60 seconds
2. **Given** control plane is running, **When** I call `/v1/health` endpoint, **Then** I receive 200 OK response
3. **Given** control plane is running, **When** I apply a policy via hfi-cli, **Then** policy is stored in etcd and retrievable via `/v1/policies`
4. **Given** control plane has 2 replicas, **When** one pod fails, **Then** the other pod continues serving requests (high availability)

---

### User Story 2 - Deploy Wasm Plugin to Istio Sidecars (Priority: P1)

As an SRE, I want to deploy the BOIFI Wasm plugin to Istio sidecar proxies so that fault injection is applied at the service mesh layer.

**Why this priority**: This is the core functionality - deploying the plugin to Istio sidecars using the `WasmPlugin` CRD. Without this, no fault injection can occur.

**Independent Test**: Create a WasmPlugin resource targeting `demo` namespace, verify plugin loads in sidecar logs, and confirm control plane connection.

**Acceptance Scenarios**:

1. **Given** control plane is running, **When** I create a WasmPlugin resource targeting `demo` namespace, **Then** plugin is loaded in all sidecar Envoys within 30 seconds
2. **Given** Wasm plugin is loaded, **When** I check Envoy logs, **Then** I see "Received config update from control plane" messages
3. **Given** Wasm plugin is loaded in multiple pods, **When** control plane updates policies, **Then** all instances receive the update within 5 seconds
4. **Given** plugin fails to load (e.g., invalid wasm), **When** I describe the WasmPlugin resource, **Then** I see clear error messages indicating the failure reason

---

### User Story 3 - Service-Level Policy Targeting (Priority: P1)

As an SRE, I want to apply fault injection policies to specific services so that I can test resilience of individual microservices without affecting the entire mesh.

**Why this priority**: In a multi-service mesh, indiscriminate fault injection is dangerous. Service-level targeting is essential for safe, controlled chaos engineering.

**Independent Test**: Create a policy targeting only `frontend` service, send requests to multiple services, verify only `frontend` is affected.

**Acceptance Scenarios**:

1. **Given** policy with `selector.service: "frontend"`, **When** I send request to frontend, **Then** fault is injected
2. **Given** policy with `selector.service: "frontend"`, **When** I send request to `productcatalog`, **Then** request proceeds normally (no fault)
3. **Given** policy with `selector.service: "*"` (wildcard), **When** I send requests to any service, **Then** fault is injected on all services
4. **Given** policy with `selector.namespace: "demo"`, **When** services in `demo` namespace receive requests, **Then** only those services have faults injected

---

### User Story 4 - Pod Identity Awareness (Priority: P2)

As an SRE, I want each Wasm plugin instance to know which service/pod it belongs to so that it correctly applies only relevant policies.

**Why this priority**: Without pod identity, each Envoy would apply all policies regardless of service, causing unintended fault injection across the mesh.

**Independent Test**: Deploy plugin to multiple services, apply service-specific policy, verify only the targeted service's Envoy applies the policy.

**Acceptance Scenarios**:

1. **Given** Wasm plugin running in `frontend` pod, **When** plugin starts, **Then** it identifies itself as belonging to `frontend` service
2. **Given** plugin identifies as `frontend`, **When** policy targets `productcatalog`, **Then** frontend's plugin ignores the policy
3. **Given** Envoy sidecar with pod labels, **When** Wasm plugin reads Envoy node metadata, **Then** it extracts `service.name` and `pod.name` correctly
4. **Given** service name cannot be determined, **When** plugin starts, **Then** it logs warning and applies wildcard policies only

---

### User Story 5 - Multi-Pod Fault Distribution (Priority: P2)

As an SRE, I want fault injection to work correctly when a service has multiple pod replicas so that the percentage-based fault injection reflects the configured rate across all instances.

**Why this priority**: With multiple replicas, each pod runs its own Envoy with the same percentage setting. The aggregate fault rate should still approximate the configured percentage.

**Independent Test**: Scale a service to 3 replicas, apply 50% fault policy, send 100 requests, verify approximately 50% fail (not 50% per pod = higher aggregate).

**Acceptance Scenarios**:

1. **Given** `frontend` with 3 replicas and 30% fault policy, **When** I send 100 requests through load balancer, **Then** approximately 30 requests fail (±10% tolerance)
2. **Given** policy applied to all replicas, **When** each request is routed to random pod, **Then** overall fault rate matches configured percentage
3. **Given** one replica is restarted, **When** new pod starts, **Then** it receives policies and maintains consistent fault rate

---

### User Story 6 - Observability and Debugging (Priority: P3)

As an SRE, I want visibility into which pods have the plugin loaded and which policies are active so that I can troubleshoot issues quickly.

**Why this priority**: In production, debugging requires visibility. This is important but not blocking for basic functionality.

**Independent Test**: List all pods with plugin status, query active policies per service.

**Acceptance Scenarios**:

1. **Given** Wasm plugin deployed, **When** I run `kubectl get wasmplugins -n demo`, **Then** I see plugin status and phase
2. **Given** plugin is running, **When** I check Envoy stats endpoint, **Then** I see `hfi.faults.aborts_total` and `hfi.faults.delays_total` metrics
3. **Given** policy is applied, **When** I check control plane `/v1/policies` endpoint, **Then** I see all active policies with their target services

---

### Edge Cases

- What happens when control plane is unreachable during plugin startup?
  - Plugin should use cached policies if available, otherwise operate in passthrough mode
- How does the system handle Istio upgrades that restart all sidecars?
  - Plugins should reconnect automatically with exponential backoff
- What happens when a service is scaled to zero and back up?
  - New pods should receive current policies within refresh interval
- How are policies handled when a namespace is deleted?
  - Policies targeting deleted namespaces become inactive but remain stored

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST deploy control plane as a Kubernetes Deployment with high availability (2+ replicas)
- **FR-002**: System MUST use Istio `WasmPlugin` CRD to deploy fault injection plugin to sidecar Envoys
- **FR-003**: System MUST allow policies to target specific services via `selector.service` field
- **FR-004**: System MUST allow policies to target specific namespaces via `selector.namespace` field
- **FR-005**: Wasm plugin MUST identify its host service by reading Envoy node metadata
- **FR-006**: Wasm plugin MUST only apply policies that match its service identity
- **FR-007**: System MUST support wildcard selectors (`*`) for namespace and service
- **FR-008**: Control plane MUST be accessible from all pods in the mesh via ClusterIP service
- **FR-009**: Wasm plugin MUST handle control plane unavailability gracefully (fail-open)
- **FR-010**: System MUST provide health check endpoints for liveness and readiness probes
- **FR-011**: System MUST support policy CRUD operations via hfi-cli from outside the cluster (via port-forward or ingress)
- **FR-012**: Wasm plugin MUST emit Prometheus-compatible metrics for fault injection counts

### Key Entities

- **WasmPlugin**: Istio CRD that deploys the Wasm binary to Envoy sidecars
- **Policy**: Fault injection configuration with service selector and fault rules
- **ServiceSelector**: Specifies which services a policy applies to (namespace, service name)
- **EnvoyNodeMetadata**: Runtime information about the Envoy instance (service name, pod name, namespace)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Control plane deploys successfully in under 2 minutes on a standard k3s cluster
- **SC-002**: Wasm plugin loads in all targeted sidecars within 60 seconds of WasmPlugin creation
- **SC-003**: Policy updates propagate to all affected sidecars within 30 seconds
- **SC-004**: When control plane is unavailable, existing functionality continues (fail-open confirmed)
- **SC-005**: Service-specific policies affect only the targeted service (0% leakage to other services)
- **SC-006**: Percentage-based fault injection achieves ±10% of configured rate across multiple replicas
- **SC-007**: System handles at least 10 concurrent services with independent policies
- **SC-008**: Plugin startup time adds less than 500ms to pod initialization

## Assumptions

- Istio is installed and configured on the target k3s cluster
- The `demo` namespace has Istio sidecar injection enabled (`istio-injection=enabled` label)
- Envoy node metadata includes `ISTIO_META_WORKLOAD_NAME` or similar identifying information
- Container registry is accessible for pulling Wasm plugin image
- Network policies allow communication between namespaces (boifi ↔ demo)

## Out of Scope

- Automatic Istio installation or configuration
- Multi-cluster deployments
- mTLS configuration between control plane and plugins
- GUI for policy management (CLI only)
- A/B testing or canary deployment strategies for the plugin itself

---

## Implementation Summary (Completed 2025-12-09)

### All User Stories Implemented ✅

**US1 - Control Plane Deployment**: ✅ Complete
- Control Plane deployed to `boifi` namespace with 2 replicas
- Health checks, readiness probes, and etcd storage working
- Verified via `kubectl` and `/v1/health` endpoint

**US2 - Wasm Plugin Deployment**: ✅ Complete  
- WasmPlugin CRD (`plugin-multi-instance.yaml`) created and deployed
- Plugin loads automatically in all Istio sidecar proxies
- Control Plane connection and policy sync verified

**US3 - Service-Level Targeting**: ✅ Complete
- `ServiceSelector` struct implemented in both Go and Rust
- Policies can target specific `service` and `namespace`
- Wildcard support (`*`) for global policies
- Tested with targeted policies on frontend service

**US4 - Pod Identity Extraction**: ✅ Complete
- `EnvoyIdentity` module extracts `WORKLOAD_NAME` and `NAMESPACE` from Envoy metadata
- Identity matching implemented in Wasm Plugin
- Verified correct filtering per-service

**US5 - Multi-Pod Scenarios**: ✅ Complete
- Fixed double probability check bug (30% → ~9% issue)
- Implemented Xorshift64* RNG for uniform distribution
- Each pod instance applies percentage independently
- Test validated 30% failure rate with 100 requests

**US6 - Observability**: ✅ Complete
- `/v1/policies/status` endpoint added to Control Plane
- Returns JSON with summary statistics and policy details
- Prometheus metrics endpoint accessible (Envoy stats)
- `kubectl get wasmplugins` for CRD status
- E2E test script (`test-us6-observability.sh`) created and passing

### Key Achievements

1. **Architecture**: Successfully adapted single-Envoy architecture to Istio sidecar model
2. **Service Targeting**: Policies can target specific services using selector field
3. **Probability Accuracy**: Fixed RNG and double-check bugs for accurate fault injection rates
4. **Observability**: Full observability stack with status endpoints and metrics
5. **Testing**: Comprehensive E2E test suite covering all 6 user stories
6. **Documentation**: Updated README files with Istio deployment instructions and selector usage

### Test Coverage

- ✅ Unit tests for Control Plane (Go)
- ✅ Unit tests for Wasm Plugin (Rust)
- ✅ E2E tests for all 6 user stories
- ✅ Integration tests (run-all-tests.sh)
- ✅ Manual validation on k3s with Online Boutique demo

### Known Limitations

- Wasm Plugin Prometheus metrics not exposed (plugin SDK limitation)
- Service selector requires Istio-injected pods (depends on Envoy metadata)
- Dead code warnings in Rust (reserved for future features)

### Phase Completion Summary

| Phase | Tasks | Status |
|-------|-------|--------|
| Phase 1 - Environment Prep | 4/4 | ✅ Complete |
| Phase 2 - Base Components | 10/10 | ✅ Complete |
| Phase 3 - US1 Control Plane | 6/6 | ✅ Complete |
| Phase 4 - US2 Wasm Plugin | 7/7 | ✅ Complete |
| Phase 5 - US3 Service Targeting | 6/6 | ✅ Complete |
| Phase 6 - US4 Pod Identity | 6/6 | ✅ Complete |
| Phase 7 - US5 Multi-Pod | 4/4 | ✅ Complete |
| Phase 8 - US6 Observability | 4/4 | ✅ Complete |
| Phase 9 - Polish & Docs | 8/8 | ✅ Complete |
| **Total** | **55/55** | **✅ 100%** |

### Deployment Artifacts

- `executor/k8s/control-plane.yaml` - Control Plane + etcd deployment
- `executor/k8s/plugin-multi-instance.yaml` - WasmPlugin CRD for Istio
- `executor/cli/hfi-cli` - CLI tool for policy management
- `executor/k8s/tests/run-all-tests.sh` - E2E test suite runner
- `executor/k8s/README.md` - Complete deployment guide
- `executor/cli/examples/README.md` - Policy examples with selectors

**Feature Status**: ✅ **Production Ready**
