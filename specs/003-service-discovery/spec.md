# Feature Specification: Service Discovery Microservice

**Feature Branch**: `003-service-discovery`  
**Created**: 2025-11-24  
**Status**: Draft  
**Input**: User description: "Implement a 'service discovery' microservice in Go... automatically and periodically probe a Kubernetes microservice environment... produce a 'Service Map'..."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Automated Topology Discovery (Priority: P1)

An SRE or platform engineer needs to automatically discover all microservices, their APIs, and call relationships in a Kubernetes cluster without manual configuration. The system should build and maintain a live topology map that other components (Recommender, Request Generator) can query to understand what can be tested and where faults can be injected.

**Why this priority**: This is the foundation for the entire fault injection system. Without knowing what services and APIs exist, the Recommender cannot recommend meaningful fault injections, and the Request Generator cannot target the right endpoints. This is a blocking dependency for P1 features in Executor and Recommender.

**Independent Test**: Can be fully tested by: (1) deploying the Discovery service in a Kubernetes cluster with 3+ microservices, (2) waiting for the initial scan cycle, (3) querying the ServiceMap via API or Redis, (4) verifying all services, APIs, and call relationships are correctly identified.

**Acceptance Scenarios**:

1. **Given** a Kubernetes cluster with 3 microservices running (`order-service`, `payment-service`, `inventory-service`), **When** Discovery component starts, **Then** within 30 seconds the system detects all three services and stores their names and namespaces in ServiceMap
2. **Given** Kubernetes services with OpenAPI/Swagger endpoints, **When** Discovery component crawls API specs, **Then** it extracts all HTTP endpoints (GET /orders/{id}, POST /orders, etc.) and operation names and stores them in the `apis` list for each service
3. **Given** distributed tracing enabled in the cluster (e.g., Jaeger), **When** Discovery component queries recent traces, **Then** it identifies call relationships (order-service → payment-service → inventory-service) and stores them as edges in the ServiceMap
4. **Given** a ServiceMap with multiple services and edges, **When** downstream components query the ServiceMap, **Then** they can construct an accurate microservice topology and understand which services call which other services

---

### User Story 2 - Incremental Topology Updates (Priority: P1)

As the system runs and new microservices are deployed or API endpoints change, Discovery must automatically detect these changes and update the ServiceMap in real-time without requiring system restart or manual intervention.

**Why this priority**: Kubernetes environments are dynamic—services scale up/down, new services deploy, endpoints change. The topology map must stay fresh to remain useful. Without this, the system becomes stale within minutes in production environments.

**Independent Test**: Can be fully tested by: (1) starting Discovery with an initial set of services, (2) deploying a new service to the cluster mid-cycle, (3) waiting for the next discovery scan, (4) verifying that the new service appears in the updated ServiceMap.

**Acceptance Scenarios**:

1. **Given** Discovery has completed an initial scan and stored a ServiceMap, **When** a new microservice is deployed to Kubernetes, **Then** within 5 minutes (next scan cycle) the new service is detected and added to the ServiceMap
2. **Given** an existing service with API endpoints [GET /orders, POST /orders], **When** the service is updated to add a new endpoint [DELETE /orders/{id}], **Then** within 5 minutes the new endpoint is detected and added to the service's `apis` list
3. **Given** a service is deleted from the Kubernetes cluster, **When** the next discovery scan runs, **Then** the service is removed from ServiceMap and any edges pointing to/from it are cleaned up
4. **Given** an existing call relationship (order-service → payment-service), **When** that relationship stops occurring (no traces detected for 2+ scan cycles), **Then** the edge is retained but marked as `active: false` or removed after a grace period (configurable)

---

### User Story 3 - Multi-Source Discovery with Graceful Degradation (Priority: P1)

Discovery should intelligently combine multiple data sources (Kubernetes API, OpenAPI/Swagger specs, Jaeger traces, service mesh if available) to build a comprehensive topology. If one source is unavailable, the system must continue operating with the available sources rather than failing completely.

**Why this priority**: Real-world Kubernetes clusters have heterogeneous configurations. Some services expose OpenAPI specs, some don't. Some have distributed tracing enabled, some don't. The system must be robust enough to work in all these scenarios, not just the ideal case.

**Independent Test**: Can be fully tested by: (1) starting Discovery in a cluster where only Kubernetes API is available (no OpenAPI, no Jaeger), (2) verifying services and basic metadata are discovered, (3) enabling Jaeger, (4) verifying topology relationships are added without restarting, (5) disabling Jaeger, (6) verifying system continues working with last-known relationships.

**Acceptance Scenarios**:

1. **Given** Kubernetes API is available but OpenAPI specs are not exposed, **When** Discovery runs, **Then** it discovers services by name/namespace and stores them with minimal metadata, and stores empty `apis` list for each service
2. **Given** both Kubernetes API and Jaeger are available, **When** Discovery runs, **Then** it combines both sources: service names from K8s and call relationships from Jaeger traces
3. **Given** OpenAPI/Swagger endpoint is temporarily unavailable for one service, **When** Discovery crawls APIs, **Then** it logs a warning but continues processing other services and retains the previous API spec for the unavailable service
4. **Given** a service mesh (e.g., Istio) is installed in the cluster, **When** Discovery is configured to use service mesh APIs, **Then** it detects service definitions and virtual services to enhance topology accuracy (without requiring mesh - mesh data is optional enhancement)
5. **Given** multiple discovery attempts fail for a single service due to transient errors, **When** the next successful attempt occurs, **Then** the ServiceMap is updated with fresh data, overwriting stale information

---

### User Story 4 - Periodic Execution with Configurable Intervals (Priority: P1)

Discovery must run automatically on a configurable schedule (default: every 5 minutes) to keep the topology map current. Operations teams must be able to adjust the scan interval based on their environment's churn rate without restarting the service.

**Why this priority**: Autonomous operation is essential for a production system. Manual discovery runs are not operationally viable. Configurable intervals allow teams to balance freshness (frequent scans) vs. cost (expensive API calls/traces).

**Independent Test**: Can be fully tested by: (1) starting Discovery with a 1-minute scan interval, (2) watching the service log/metrics to verify scans execute at 1-minute intervals, (3) changing the configuration to 10-minute interval, (4) verifying scans adjust to the new interval without restart.

**Acceptance Scenarios**:

1. **Given** Discovery is running with `scan_interval_seconds=300` (5 minutes), **When** time elapses, **Then** the system automatically initiates a discovery scan every 300 seconds without human intervention
2. **Given** Discovery is operational, **When** an operator updates the configuration to `scan_interval_seconds=60`, **Then** within 30 seconds the new interval takes effect and subsequent scans run at 60-second intervals
3. **Given** a discovery scan is in progress, **When** the next scheduled scan time arrives, **Then** the next scan waits for the current scan to complete before starting (no concurrent scans)
4. **Given** a discovery scan takes 15 seconds to complete, **When** the scan interval is 60 seconds, **Then** the system waits 45 seconds after completion before starting the next scan (interval measured from scan end, not start)

---

### User Story 5 - ServiceMap Publication to Shared Knowledge Base (Priority: P1)

Discovery must publish the discovered ServiceMap to a shared, queryable knowledge base (Redis) so that Recommender, Request Generator, and other components can reliably access the latest topology without direct coupling to Discovery.

**Why this priority**: Decoupling is essential for system modularity. Recommender and other downstream components must not need to call Discovery directly; they query a well-defined shared state.

**Independent Test**: Can be fully tested by: (1) starting Discovery, (2) waiting for a scan to complete, (3) connecting to Redis, (4) querying the ServiceMap key, (5) verifying valid JSON structure matches schema, (6) updating topology and verifying Redis key is updated.

**Acceptance Scenarios**:

1. **Given** Discovery completes a scan, **When** results are processed, **Then** a JSON document (`service_map.json` or key `hfi:service-map:latest`) is published to Redis with the complete ServiceMap
2. **Given** multiple components need current topology, **When** they query Redis key `hfi:service-map:latest`, **Then** they receive the most recent ServiceMap snapshot in valid JSON format
3. **Given** a new ServiceMap is published, **When** downstream components subscribe to Redis (or poll periodically), **Then** they are aware of topology changes within 10 seconds
4. **Given** Redis is temporarily unavailable, **When** Discovery completes a scan, **Then** it logs the error, retains the ServiceMap in memory, and retries publishing to Redis on the next scan cycle
5. **Given** multiple ServiceMap versions are generated, **When** Redis stores them, **Then** the system maintains a versioned history (e.g., `hfi:service-map:latest`, `hfi:service-map:v1`, `hfi:service-map:v2`) with the latest version being the default query target

### Edge Cases

- What happens if a service name conflicts with another service (same name, different namespace)? (Qualified names using namespace prefix)
- How does Discovery handle services without any OpenAPI spec and no traces? (Service is discovered from K8s but stored with empty `apis` and no edges)
- What if the Redis connection is lost during publication? (Retry with exponential backoff, then log error and skip)
- How are circular dependencies in topology handled? (Graph structure supports cycles, visualization/analysis tools must handle them)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST connect to Kubernetes API Server using `client-go` and list all `VirtualService` resources in all namespaces.
- **FR-002**: System MUST extract Service Name, API Path (prefix/exact/regex), and HTTP Method from `VirtualService` HTTP routes.
- **FR-003**: System MUST attempt to fetch OpenAPI specifications from discovered services via standard paths (`/swagger.json`, `/v3/api-docs`, etc.) on standard ports.
- **FR-004**: System MUST parse valid OpenAPI JSON responses to extract detailed API endpoints and methods, merging this data with `VirtualService` data.
- **FR-005**: System MUST connect to Jaeger Query API and retrieve traces for a configurable lookback period (default: 1 hour).
- **FR-006**: System MUST parse traces to identify caller-callee relationships between services based on span `process.serviceName` and parent references.
- **FR-007**: System MUST aggregate all discovered data into a `ServiceMap` structure containing a timestamp, map of services with API details, and list of topology edges.
- **FR-008**: System MUST serialize the `ServiceMap` to JSON and publish it to a configured Redis key (e.g., `boifi:service-map`).
- **FR-009**: System MUST publish an update notification to a Redis Channel (e.g., `boifi:service-map:updates`) upon successful map update.

### Non-Functional Requirements

- **NFR-001**: System MUST execute the discovery process periodically based on a configurable interval (default: 5 minutes).
- **NFR-002**: System MUST allow configuration of external dependencies (K8s API, Jaeger, Redis) via environment variables or command-line flags.
- **NFR-003**: System MUST handle partial failures (e.g., Jaeger down, single service OpenAPI unreachable) gracefully by logging errors and continuing with available data.
- **NFR-004**: System MUST implement structured logging for key lifecycle events (start, scan complete, publish success, errors).
- **NFR-005**: System MUST NOT crash due to malformed external data (e.g., invalid JSON from OpenAPI, corrupt trace data).

### Success Criteria

- **SC-001**: Discovery cycle completes within 30 seconds for a cluster with 50 microservices.
- **SC-002**: 100% of services with valid `VirtualService` definitions are detected and included in the ServiceMap.
- **SC-003**: Topology map accurately reflects 100% of service dependencies visible in Jaeger traces from the last hour.
- **SC-004**: System recovers from Redis connection failure and successfully publishes on the next cycle after connection is restored.

### Key Entities *(include if feature involves data)*

- **ServiceMap**: The root container for the discovery result.
  - `Timestamp`: Time of generation.
  - `Services`: Map of service names to `APIDetails`.
  - `Topology`: List of `ServiceEdge` objects.
- **APIDetails**: Detailed information about a service's interface.
  - `APIs`: List of endpoint strings (e.g., "GET /api/v1/resource").
- **ServiceEdge**: Represents a directional call between services.
  - `Source`: Caller service name.
  - `Target`: Callee service name.
  - `Count`: Number of calls observed in the lookback period.
