# Implementation Plan: Fix Fault Injection

**Branch**: `005-fix-fault-injection` | **Date**: 2025-11-27 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/005-fix-fault-injection/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

The goal is to fix the `abort-policy` returning 200 OK instead of the expected error code, and to verify and harden the entire fault injection pipeline (delay, percentage, headers, expiration, start delay). The approach involves debugging the Wasm plugin's request interception logic, ensuring correct policy serialization/deserialization in the Control Plane, and implementing missing features like `start_delay_ms` and automatic expiration.

## Technical Context

**Language/Version**: Go 1.20+ (Control Plane), Rust (Wasm Plugin)
**Primary Dependencies**: `proxy-wasm-rust-sdk` (Plugin), `gin` (Control Plane API)
**Storage**: In-memory (Control Plane)
**Testing**: `go test` (Control Plane), `cargo test` (Plugin), `curl`/`hfi-cli` (E2E)
**Target Platform**: Linux, Envoy Proxy (Wasm)
**Project Type**: Microservices Infrastructure
**Performance Goals**: <1ms overhead when no fault injected
**Constraints**: Must run within Envoy's Wasm sandbox
**Scale/Scope**: Single Envoy instance for testing, scalable to mesh

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **Separation of Concerns**: Control Plane manages policies; Wasm Plugin enforces them.
- [x] **Declarative Configuration**: Policies defined in YAML/JSON.
- [x] **Dynamic & Real-Time**: Policies pushed via SSE/Watch (implied).
- [x] **Test-Driven Development**: Plan includes verification steps.
- [x] **Performance Priority**: Wasm plugin efficiency is a key constraint.
- [x] **Fault Tolerance**: Fail-open behavior specified.
- [x] **Simplicity**: Using existing architecture, fixing bugs.
- [x] **Lifecycle Management**: `duration_seconds` and `start_delay_ms` explicitly addressed.

## Project Structure

### Documentation (this feature)

```text
specs/005-fix-fault-injection/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
executor/
├── control-plane/       # Go Control Plane
│   ├── api/             # API definitions
│   ├── distributor.go   # Policy distribution logic
│   └── main.go
├── wasm-plugin/         # Rust Wasm Plugin
│   ├── src/
│   │   ├── lib.rs       # Main plugin logic
│   │   └── config.rs    # Configuration parsing
│   └── Cargo.toml
└── cli/                 # CLI tool for testing
```

**Structure Decision**: Using existing `executor` structure.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

N/A - No violations.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
