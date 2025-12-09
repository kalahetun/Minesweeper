# wasm_fault_injection Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-11-27

## Active Technologies
- Rust 1.75+ (Wasm plugin), YAML (Kubernetes manifests) (008-wasm-metrics-exposure)
- N/A (metrics are ephemeral, stored in Envoy memory, scraped by Prometheus) (008-wasm-metrics-exposure)

- Go 1.20+ (Control Plane), Rust (Wasm Plugin) + `proxy-wasm-rust-sdk` (Plugin), `gin` (Control Plane API) (005-fix-fault-injection)

## Project Structure

```text
src/
tests/
```

## Commands

cargo test [ONLY COMMANDS FOR ACTIVE TECHNOLOGIES][ONLY COMMANDS FOR ACTIVE TECHNOLOGIES] cargo clippy

## Code Style

Go 1.20+ (Control Plane), Rust (Wasm Plugin): Follow standard conventions

## Recent Changes
- 008-wasm-metrics-exposure: Added Rust 1.75+ (Wasm plugin), YAML (Kubernetes manifests)

- 005-fix-fault-injection: Added Go 1.20+ (Control Plane), Rust (Wasm Plugin) + `proxy-wasm-rust-sdk` (Plugin), `gin` (Control Plane API)

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
