# wasm_fault_injection Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-11-27

## Active Technologies
- Rust 1.75+ (Wasm plugin), YAML (Kubernetes manifests) (008-wasm-metrics-exposure)
- N/A (metrics are ephemeral, stored in Envoy memory, scraped by Prometheus) (008-wasm-metrics-exposure)
- Bash (验证脚本), YAML (策略文件), Markdown (文档) + kubectl, curl, jq (验证脚本依赖) (009-cli-examples-update)
- N/A (文件系统，无数据库) (009-cli-examples-update)
- Rust 1.75+ (Wasm Plugin), Go 1.20+ (Control Plane, CLI) + proxy-wasm-rust-sdk, serde, gin (Control Plane API) (010-fix-wasm-delay-bug)
- N/A (策略存储在内存中，通过 Control Plane 分发) (010-fix-wasm-delay-bug)

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
- 010-fix-wasm-delay-bug: Added Rust 1.75+ (Wasm Plugin), Go 1.20+ (Control Plane, CLI) + proxy-wasm-rust-sdk, serde, gin (Control Plane API)
- 009-cli-examples-update: Added Bash (验证脚本), YAML (策略文件), Markdown (文档) + kubectl, curl, jq (验证脚本依赖)
- 008-wasm-metrics-exposure: Added Rust 1.75+ (Wasm plugin), YAML (Kubernetes manifests)


<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
