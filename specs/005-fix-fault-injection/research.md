# Research: Fix Fault Injection

**Feature**: Fix Fault Injection
**Status**: Complete

## 1. Wasm Plugin Abort Logic
**Decision**: Use `send_http_response` for abort faults.
**Rationale**: The `proxy-wasm` SDK provides `send_http_response` to immediately return a response and stop processing. The current issue likely stems from incorrect usage of this API or failure to return `Action::Pause` after sending the response.
**Alternatives Considered**: Modifying headers and body manually (too complex, error-prone).

## 2. Policy Serialization
**Decision**: Use JSON for Wasm configuration.
**Rationale**: Rust `serde_json` and Go `encoding/json` are robust. The Control Plane must serialize the policy list into a JSON string that the Wasm plugin deserializes in `on_configure`.
**Alternatives Considered**: Protobuf (overkill for this stage), custom binary format (hard to debug).

## 3. Time Management in Wasm
**Decision**: Use `proxy_wasm::types::SystemTime` for expiration checks and `proxy_wasm::hostcalls::set_tick_period_milliseconds` for cleanup.
**Rationale**: Wasm runs in a sandbox. We need to check the current time against the policy's `start_time + duration`. For `start_delay_ms`, we might need to use `set_http_request_header` to mark start time or use `dispatch_http_call` with a delay (though `dispatch_http_call` is for external calls). A better approach for `start_delay_ms` inside the request path is to use `std::thread::sleep` (BLOCKING - BAD) or `proxy_wasm::hostcalls::resume_http_request` combined with a timer.
**Correction**: `std::thread::sleep` blocks the Envoy worker thread. We MUST NOT use it. The correct way to implement delay is to return `Action::Pause` and set a timer using `set_tick_period_milliseconds` or similar callback mechanism to resume the request.

## 4. Start Delay Implementation
**Decision**: Use `set_effective_context` or similar to pause and resume.
**Rationale**: To implement `start_delay_ms` without blocking:
1. Return `Action::Pause` from `on_http_request_headers`.
2. Schedule a callback (if supported by SDK) or use a tick loop to check if delay has passed.
3. Call `resume_http_request`.
**Refinement**: The Rust SDK supports `set_tick_period_milliseconds`. We can set a tick, and in `on_tick`, check if it's time to resume. However, `on_tick` is global for the root context or stream context. Stream context `on_tick` is ideal.

## 5. Service Identity
**Decision**: Read `node.metadata` from Envoy.
**Rationale**: Envoy exposes node metadata to Wasm. We can configure the `app` or `service` name in the Envoy bootstrap config and read it in the plugin to filter policies.
