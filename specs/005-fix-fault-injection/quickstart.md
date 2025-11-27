# Quickstart: Fix Fault Injection

## Prerequisites

- Docker & Docker Compose
- Go 1.20+
- Rust & Cargo (for Wasm build)
- `hfi-cli` (built from `executor/cli`)

## Build & Deploy

1. **Build Wasm Plugin**:
   ```bash
   cd executor/wasm-plugin
   make build
   ```

2. **Start Environment**:
   ```bash
   cd executor/docker
   docker-compose up -d
   ```

3. **Build CLI**:
   ```bash
   cd executor/cli
   go build -o hfi-cli main.go
   ```

## Verification Steps

### 1. Verify Abort Injection (Fix)

Apply the abort policy:
```bash
./hfi-cli policy apply -f examples/abort-policy.yaml
```

Send a request:
```bash
curl -I http://localhost:18000/
```

**Expected Output**: `HTTP/1.1 503 Service Unavailable` (was 200 OK).

### 2. Verify Delay Injection

Apply the delay policy:
```bash
./hfi-cli policy apply -f examples/delay-policy.yaml
```

Send a request:
```bash
time curl -I http://localhost:18000/
```

**Expected Output**: Request takes > 2 seconds.

### 3. Verify Expiration

Apply a short-lived policy:
```bash
./hfi-cli policy apply -f examples/time-limited-fault-policy.yaml
```

Wait 6 seconds, then curl.
**Expected Output**: 200 OK (fault expired).

### 4. Verify Start Delay

Apply start delay policy:
```bash
./hfi-cli policy apply -f examples/delayed-timed-fault-policy.yaml
```

**Expected Output**: Request pauses for 200ms before returning 503.
