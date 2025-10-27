# Docker Configuration

This directory contains all Docker-related files for the HFI executor.

## Files

- **Dockerfile.backend** - Builds the test backend service (nginx)
- **Dockerfile.controlplane** - Builds the HFI control plane service (Go)
- **Dockerfile.wasm** - Builds the WASM plugin (Rust)
- **docker-compose.yaml** - Orchestrates all services for local development
- **envoy.yaml** - Envoy proxy configuration (loaded by docker-compose)

## Usage

### Build and Run with Docker Compose

```bash
# From the executor root directory
cd ..
docker-compose -f docker/docker-compose.yaml up
```

This will start:
- **etcd** - Configuration store
- **control-plane** - HFI control plane (Go)
- **wasm-builder** - Builds the WASM plugin
- **backend** - Test backend service
- **envoy** - Envoy proxy with WASM plugin

### Build Individual Services

```bash
# Build control plane
docker build -f docker/Dockerfile.controlplane -t hfi-control-plane .

# Build WASM plugin
docker build -f docker/Dockerfile.wasm -t hfi-wasm-plugin .

# Build backend
docker build -f docker/Dockerfile.backend -t hfi-backend .
```

## Configuration Files

### envoy.yaml
Envoy proxy configuration file that:
- Listens on port 18000 for incoming traffic
- Provides admin interface on port 19000
- Loads the WASM plugin for fault injection
- Routes traffic to the backend service
- Connects to the control plane for dynamic configuration

The configuration is automatically loaded by docker-compose and mounted into the Envoy container.

## Service Details

### Control Plane
- **Port**: 8080
- **Healthcheck**: GET /v1/health
- **Dependencies**: etcd

### WASM Plugin Builder
- **Build**: Multi-stage build with Rust
- **Output**: plugin.wasm
- **Optimization**: Uses wasm-opt for binary size reduction

### Test Backend
- **Port**: 80 (internal)
- **Purpose**: Test target for fault injection

### Envoy Proxy
- **Port**: 18000 (external traffic)
- **Port**: 19000 (admin interface)
- **Dependencies**: wasm-builder, control-plane
