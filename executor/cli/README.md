# HFI CLI (Hardware Fault Injection CLI)

A command-line interface for managing fault injection policies in the HFI system.

## Project Structure

```
cli/
├── main.go              # Main entry point
├── go.mod              # Go module definition
├── cmd/                # Command implementations
│   ├── root.go         # Root command with global flags
│   └── policy.go       # Policy container command
├── client/             # API client library
│   └── client.go       # HTTP client for Control Plane API
└── hfi-cli             # Compiled binary
```

## Features Implemented (Task CL-1)

✅ **Framework Setup**
- Uses `cobra` library for CLI framework
- Modular command structure with `cmd` package
- Clean separation of concerns

✅ **Root Command (`cmd/root.go`)**
- Defines `rootCmd` as the main CLI entry point
- Comprehensive help text and examples
- Persistent global flags available to all subcommands:
  - `--control-plane-addr` (string, default: `http://localhost:8080`)
  - `--timeout` (duration, default: `30s`)

✅ **Global Flag Handling**
- `PersistentPreRunE` hook initializes shared resources
- Validates control plane address format
- Creates global `APIClient` instance with proper timeout configuration
- Performs optional health check with warning on failure

✅ **API Client (`client/client.go`)**
- Interface-based design (`IAPIClient`) for testability
- Concrete implementation (`APIClient`) encapsulates `*http.Client`
- URL validation and scheme checking
- Custom error types (`APIError`) for structured error handling
- Health check using Control Plane's `/v1/policies` endpoint
- Context support for cancellation and timeouts

✅ **Policy Container Command (`cmd/policy.go`)**
- Container command for policy-related operations
- Comprehensive help text with examples
- Shows help when called without subcommands
- Ready for subcommands to be added

## Usage

### Basic Commands

```bash
# Show help
./hfi-cli --help

# Show policy commands
./hfi-cli policy --help

# Use custom control plane address
./hfi-cli --control-plane-addr http://my-control-plane:8080 policy

# Use custom timeout
./hfi-cli --timeout 60s policy
```

### Error Handling

The CLI provides clear error messages for various scenarios:

- **Invalid URL format**: `Error: failed to initialize API client: invalid URL scheme`
- **Connection refused**: `Warning: Control plane health check failed: connection refused`
- **Network timeouts**: Handled gracefully with context cancellation

## Build and Run

```bash
# Install dependencies
go mod tidy

# Build the CLI
go build -o hfi-cli .

# Run the CLI
./hfi-cli --help
```

## Architecture Highlights

1. **Cobra Framework**: Industry-standard CLI framework used by tools like `kubectl`
2. **Interface-based Design**: `IAPIClient` interface allows for easy mocking and testing
3. **Global State Management**: Shared API client initialized once and reused by all subcommands
4. **Error Handling**: Structured error types with clear user-facing messages
5. **Context Support**: All API calls support context for cancellation and timeouts
6. **Health Checks**: Optional connectivity validation with graceful error handling

## Next Steps

Ready for implementation of:
- `apply` subcommand (Task CL-2)
- `get` subcommand (Task CL-3) 
- `delete` subcommand (Task CL-4)
- Advanced output formatting (table, YAML, JSON)
- Policy file validation and parsing
