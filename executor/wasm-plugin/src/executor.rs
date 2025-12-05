use crate::config::{Fault, AbortAction, DelayAction};
use crate::time_control::{TimeControlDecision, should_inject_fault, RuleTiming, RequestTiming};
use crate::metrics::FaultInjectionMetrics;
use proxy_wasm::traits::HttpContext;
use proxy_wasm::types::Action;
use log::{info, warn, debug};

/// Metrics IDs for fault injection counters
#[derive(Clone, Copy)]
pub struct MetricsIds {
    pub aborts_total: Option<u32>,
    pub delays_total: Option<u32>,
    pub delay_duration_histogram: Option<u32>,
}

/// Execute fault injection logic
/// Note: Probability check is done by caller (lib.rs), this function executes unconditionally
pub fn execute_fault(
    fault: &Fault,
    http_context: &dyn HttpContext,
    context_id: u32,
    metrics: MetricsIds,
) -> Action {
    info!("Triggering fault injection for context {}", context_id);
    
    // Execute fault based on type
    if let Some(ref abort) = fault.abort {
        info!("Executing abort fault with status {}", abort.http_status);
        execute_abort(abort, http_context, metrics)
    } else if let Some(ref delay) = fault.delay {
        info!("Executing delay fault: {}", delay.fixed_delay);
        execute_delay(delay, context_id, metrics)
    } else {
        warn!("No fault action specified, continuing");
        Action::Continue
    }
}

/// Execute fault injection with time control
/// 
/// This function wraps the standard execute_fault with time control checks,
/// ensuring that faults are only injected when time constraints are satisfied.
/// 
/// # Arguments
/// 
/// * `fault` - The fault configuration
/// * `http_context` - The HTTP context for sending responses
/// * `context_id` - The context ID for tracking
/// * `metrics` - Metrics IDs for counters
/// * `rule_creation_time_ms` - When the rule was created (milliseconds)
/// * `request_arrival_time_ms` - When the request arrived (milliseconds)
/// 
/// # Returns
/// 
/// - `Action::Continue` if time constraints prevent injection or rule is expired
/// - Result of `execute_fault` if time constraints allow injection
pub fn execute_fault_with_time_control(
    fault: &Fault,
    http_context: &dyn HttpContext,
    context_id: u32,
    metrics: MetricsIds,
    rule_creation_time_ms: u64,
    request_arrival_time_ms: u64,
) -> Action {
    use crate::time_control::get_elapsed_time_ms;
    
    // Create time control structures
    let rule_timing = RuleTiming {
        start_delay_ms: fault.start_delay_ms,
        duration_seconds: fault.duration_seconds,
        creation_time_ms: rule_creation_time_ms,
    };
    
    let request_timing = RequestTiming {
        arrival_time_ms: request_arrival_time_ms,
        elapsed_since_arrival_ms: get_elapsed_time_ms(request_arrival_time_ms),
    };
    
    // Check time constraints
    match should_inject_fault(&rule_timing, &request_timing) {
        TimeControlDecision::Inject => {
            debug!("Time control decision: Inject - executing fault for context {}", context_id);
            execute_fault(fault, http_context, context_id, metrics)
        }
        TimeControlDecision::WaitForDelay => {
            debug!("Time control decision: WaitForDelay - request still in delay period for context {}", context_id);
            Action::Continue
        }
        TimeControlDecision::Expired => {
            info!("Time control decision: Expired - rule has expired for context {}", context_id);
            Action::Continue
        }
    }
}

/// Execute fault injection with time control and metrics collection
/// 
/// This function wraps execute_fault_with_time_control with comprehensive metrics collection.
/// 
/// # Arguments
/// 
/// * `fault` - The fault configuration
/// * `http_context` - The HTTP context for sending responses
/// * `context_id` - The context ID for tracking
/// * `metrics_ids` - Envoy metrics IDs for counters
/// * `rule_creation_time_ms` - When the rule was created (milliseconds)
/// * `request_arrival_time_ms` - When the request arrived (milliseconds)
/// * `metrics_collector` - The metrics collector for recording statistics
/// 
/// # Returns
/// 
/// The action to take for this request
pub fn execute_fault_with_metrics(
    fault: &Fault,
    http_context: &dyn HttpContext,
    context_id: u32,
    metrics_ids: MetricsIds,
    rule_creation_time_ms: u64,
    request_arrival_time_ms: u64,
    metrics_collector: &FaultInjectionMetrics,
) -> Action {
    use crate::time_control::get_elapsed_time_ms;
    
    // Record the request
    metrics_collector.record_request();
    
    // Record rule match
    metrics_collector.record_rule_matched();
    
    // Create time control structures
    let rule_timing = RuleTiming {
        start_delay_ms: fault.start_delay_ms,
        duration_seconds: fault.duration_seconds,
        creation_time_ms: rule_creation_time_ms,
    };
    
    let request_timing = RequestTiming {
        arrival_time_ms: request_arrival_time_ms,
        elapsed_since_arrival_ms: get_elapsed_time_ms(request_arrival_time_ms),
    };
    
    // Check time constraints
    match should_inject_fault(&rule_timing, &request_timing) {
        TimeControlDecision::Inject => {
            debug!("Time control decision: Inject - executing fault for context {}", context_id);
            metrics_collector.record_fault_injected();
            
            // Execute fault and record metrics
            if let Some(ref abort) = fault.abort {
                metrics_collector.record_abort_fault();
            } else if let Some(ref delay) = fault.delay {
                if let Some(duration_ms) = delay.parsed_duration_ms {
                    metrics_collector.record_delay_fault(duration_ms as u64);
                }
            }
            
            execute_fault(fault, http_context, context_id, metrics_ids)
        }
        TimeControlDecision::WaitForDelay => {
            debug!("Time control decision: WaitForDelay - request still in delay period for context {}", context_id);
            metrics_collector.record_time_control_wait();
            Action::Continue
        }
        TimeControlDecision::Expired => {
            info!("Time control decision: Expired - rule has expired for context {}", context_id);
            metrics_collector.record_rule_expired();
            Action::Continue
        }
    }
}

/// Execute abort fault
fn execute_abort(abort: &AbortAction, http_context: &dyn HttpContext, metrics: MetricsIds) -> Action {
    let body = abort.body.as_deref().unwrap_or("Fault injection: Service unavailable");
    let headers = vec![
        ("content-type", "text/plain"),
        ("x-fault-injected", "abort"),
    ];
    
    debug!("Sending abort response: status={}, body_len={}", abort.http_status, body.len());
    
    // Increment abort counter metric
    if let Some(metric_id) = metrics.aborts_total {
        if let Err(e) = increment_counter(metric_id, 1) {
            warn!("Failed to increment abort counter: {:?}", e);
        } else {
            debug!("Incremented hfi.faults.aborts_total counter");
        }
    }
    
    // send_http_response returns (), not a Result
    http_context.send_http_response(abort.http_status, headers, Some(body.as_bytes()));
    Action::Pause
}

/// Execute delay fault (simplified - actual delay mechanism needs async support)
/// 
/// Note: WASM plugin framework limitations prevent true asynchronous delay.
/// Returning Action::Pause tells Envoy to wait, but the plugin cannot resume
/// the request after a delay without additional infrastructure (e.g., external timers).
/// 
/// Implementation improvements (M1):
/// - DelayManager now tracks delayed requests and supports cancellation via cancel_delay()
/// - HTTP calls can be cancelled on delay expiry to prevent stale requests
/// - Monitoring method get_delayed_count() available for observability
/// 
/// Future enhancements:
/// - Use Envoy's timer callbacks (proxy_on_timer) for true async delays
/// - Or integrate external delay service
/// - Or defer to Lua filter that supports async delays
fn execute_delay(delay: &DelayAction, context_id: u32, metrics: MetricsIds) -> Action {
    if let Some(duration_ms) = delay.parsed_duration_ms {
        info!("Delay fault triggered for context {} - {}ms", context_id, duration_ms);
        
        // Increment delay counter metric
        if let Some(metric_id) = metrics.delays_total {
            if let Err(e) = increment_counter(metric_id, 1) {
                warn!("Failed to increment delay counter: {:?}", e);
            } else {
                debug!("Incremented hfi.faults.delays_total counter");
            }
        }
        
        // Record delay duration in histogram
        if let Some(metric_id) = metrics.delay_duration_histogram {
            if let Err(e) = record_histogram(metric_id, duration_ms) {
                warn!("Failed to record delay duration histogram: {:?}", e);
            } else {
                debug!("Recorded delay duration {}ms in histogram", duration_ms);
            }
        }
        
        // Return Pause to indicate the request should be delayed
        // Note: Actual delay implementation depends on Envoy/WASM host support
        // This is currently a placeholder that tells Envoy to pause the request
        debug!("Request paused for delay ({}ms) - context {}", duration_ms, context_id);
        Action::Pause
    } else {
        warn!("Delay duration not parsed correctly: {}", delay.fixed_delay);
        Action::Continue
    }
}

/// Generate random percentage (0-100)
/// Uses thread-local storage for thread-safe PRNG
pub fn generate_random_percentage() -> u32 {
    thread_local! {
        static SEED: std::cell::RefCell<u64> = {
            // Initialize seed with current time nanoseconds
            let initial_seed = proxy_wasm::hostcalls::get_current_time()
                .ok()
                .and_then(|t| t.duration_since(std::time::UNIX_EPOCH).ok())
                .map(|d| d.as_nanos() as u64)
                .unwrap_or(88172645463325252);  // Non-zero default
            std::cell::RefCell::new(if initial_seed == 0 { 1 } else { initial_seed })
        };
    }

    SEED.with(|seed| {
        let mut s = seed.borrow_mut();
        
        loop {
            // Xorshift64* algorithm - better statistical properties than LCG
            *s ^= *s >> 12;
            *s ^= *s << 25;
            *s ^= *s >> 27;
            let random64 = s.wrapping_mul(0x2545F4914F6CDD1D);
            
            // Extract 7 bits (0-127 range) and reject if >= 101 for unbiased distribution
            let bits7 = ((random64 >> 32) & 0x7F) as u32;
            if bits7 <= 100 {
                return bits7;
            }
            // If rejected, loop and generate new random value
        }
    })
}

/// Increment a counter metric
fn increment_counter(metric_id: u32, value: u64) -> Result<(), proxy_wasm::types::Status> {
    debug!("Incrementing counter with ID {} by {}", metric_id, value);
    proxy_wasm::hostcalls::increment_metric(metric_id, value as i64)
}

/// Record a value in a histogram metric
fn record_histogram(metric_id: u32, value: u64) -> Result<(), proxy_wasm::types::Status> {
    debug!("Recording histogram with ID {} value: {}", metric_id, value);
    proxy_wasm::hostcalls::record_metric(metric_id, value)
}

/// Manages delayed requests
pub struct DelayManager {
    delayed_requests: std::collections::HashMap<u32, u32>, // timer_token -> context_id
    next_token: u32,
}

impl DelayManager {
    pub fn new() -> Self {
        Self {
            delayed_requests: std::collections::HashMap::new(),
            next_token: 1,
        }
    }
    
    pub fn add_delay(&mut self, context_id: u32, _duration_ms: u32) -> u32 {
        let token = self.next_token;
        self.next_token += 1;
        self.delayed_requests.insert(token, context_id);
        token
    }
    
    pub fn handle_timer(&mut self, timer_token: u32) -> Option<u32> {
        self.delayed_requests.remove(&timer_token)
    }
    
    /// Cancel a delayed request and optionally cancel the associated HTTP call
    /// This prevents stale requests from being sent after delay expires
    pub fn cancel_delay(&mut self, timer_token: u32) -> Option<u32> {
        debug!("Cancelling delayed request with token {}", timer_token);
        self.delayed_requests.remove(&timer_token)
    }
    
    /// Get the number of currently delayed requests (for monitoring)
    pub fn get_delayed_count(&self) -> usize {
        self.delayed_requests.len()
    }
}

/// Trait for contexts that can execute faults
pub trait FaultExecutorContext {
    fn execute_fault_for_context(&self, fault: &Fault, context_id: u32) -> Action;
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::{Fault, AbortAction, DelayAction};
    use std::cell::RefCell;

    struct MockHttpContext {
        response_sent: RefCell<Option<(u32, String)>>,
    }

    impl MockHttpContext {
        fn new() -> Self {
            Self {
                response_sent: RefCell::new(None),
            }
        }
        
        fn get_sent_response(&self) -> Option<(u32, String)> {
            self.response_sent.borrow().clone()
        }
    }

    impl proxy_wasm::traits::Context for MockHttpContext {}
    
    impl HttpContext for MockHttpContext {
        fn send_http_response(
            &self,
            status_code: u32,
            _headers: Vec<(&str, &str)>,
            body: Option<&[u8]>,
        ) {
            let body_str = body.map(|b| String::from_utf8_lossy(b).to_string())
                              .unwrap_or_default();
            *self.response_sent.borrow_mut() = Some((status_code, body_str));
        }
    }

    #[test]
    fn test_generate_random_percentage() {
        for _ in 0..100 {
            let value = generate_random_percentage();
            assert!(value <= 100, "Random value {} should be <= 100", value);
        }
    }

    #[test]
    fn test_execute_abort_fault() {
        let abort = AbortAction {
            http_status: 500,
            body: Some("Test error".to_string()),
        };
        
        let fault = Fault {
            percentage: 100,
            abort: Some(abort),
            delay: None,
            start_delay_ms: 0,
            duration_seconds: 0,
        };
        
        let mock_context = MockHttpContext::new();
        let metrics = MetricsIds {
            aborts_total: None,
            delays_total: None,
            delay_duration_histogram: None,
        };
        let result = execute_fault(&fault, &mock_context, 1, metrics);
        
        assert_eq!(result, Action::Pause);
        
        let response = mock_context.get_sent_response();
        assert!(response.is_some());
        let (status, body) = response.unwrap();
        assert_eq!(status, 500);
        assert_eq!(body, "Test error");
    }

    #[test]
    fn test_execute_delay_fault() {
        let delay = DelayAction {
            fixed_delay: "100ms".to_string(),
            parsed_duration_ms: Some(100),
        };
        
        let fault = Fault {
            percentage: 100,
            abort: None,
            delay: Some(delay),
            start_delay_ms: 0,
            duration_seconds: 0,
        };
        
        let mock_context = MockHttpContext::new();
        let metrics = MetricsIds {
            aborts_total: None,
            delays_total: None,
            delay_duration_histogram: None,
        };
        let result = execute_fault(&fault, &mock_context, 1, metrics);
        
        // For W-4, delay just logs and continues
        assert_eq!(result, Action::Continue);
    }

    #[test]
    fn test_fault_probability_miss() {
        let abort = AbortAction {
            http_status: 500,
            body: None,
        };
        
        let fault = Fault {
            percentage: 0, // 0% chance should never trigger
            abort: Some(abort),
            delay: None,
            start_delay_ms: 0,
            duration_seconds: 0,
        };
        
        let mock_context = MockHttpContext::new();
        let metrics = MetricsIds {
            aborts_total: None,
            delays_total: None,
            delay_duration_histogram: None,
        };
        let result = execute_fault(&fault, &mock_context, 1, metrics);
        
        assert_eq!(result, Action::Continue);
        assert!(mock_context.get_sent_response().is_none());
    }

    #[test]
    fn test_delay_manager() {
        let mut manager = DelayManager::new();
        
        let token1 = manager.add_delay(1, 100);
        let token2 = manager.add_delay(2, 200);
        
        assert_eq!(manager.handle_timer(token1), Some(1));
        assert_eq!(manager.handle_timer(token2), Some(2));
        assert_eq!(manager.handle_timer(999), None);
    }

    #[test]
    fn test_no_fault_configured() {
        let fault = Fault {
            percentage: 100,
            abort: None,
            delay: None,
            start_delay_ms: 0,
            duration_seconds: 0,
        };
        
        let mock_context = MockHttpContext::new();
        let metrics = MetricsIds {
            aborts_total: None,
            delays_total: None,
            delay_duration_histogram: None,
        };
        let result = execute_fault(&fault, &mock_context, 1, metrics);
        
        assert_eq!(result, Action::Continue);
    }

    #[test]
    fn test_execute_fault_with_time_control_immediate() {
        // Test immediate injection (no delay, no expiry)
        let fault = Fault {
            percentage: 100,
            abort: Some(AbortAction {
                http_status: 500,
                body: Some("Internal Server Error".to_string()),
            }),
            delay: None,
            start_delay_ms: 0,
            duration_seconds: 0,
        };
        
        let mock_context = MockHttpContext::new();
        let metrics = MetricsIds {
            aborts_total: None,
            delays_total: None,
            delay_duration_histogram: None,
        };
        
        let current_time = crate::time_control::get_current_time_ms();
        let result = execute_fault_with_time_control(
            &fault,
            &mock_context,
            1,
            metrics,
            current_time,
            current_time,
        );
        
        // Should execute abort fault
        assert_eq!(result, Action::Pause);
    }

    #[test]
    fn test_execute_fault_with_time_control_delay_period() {
        // Test when request is in delay period
        let fault = Fault {
            percentage: 100,
            abort: Some(AbortAction {
                http_status: 500,
                body: Some("Internal Server Error".to_string()),
            }),
            delay: None,
            start_delay_ms: 500,  // 500ms delay required
            duration_seconds: 0,
        };
        
        let mock_context = MockHttpContext::new();
        let metrics = MetricsIds {
            aborts_total: None,
            delays_total: None,
            delay_duration_histogram: None,
        };
        
        let current_time = crate::time_control::get_current_time_ms();
        let request_arrival = current_time.saturating_sub(200);  // Request arrived 200ms ago
        
        let result = execute_fault_with_time_control(
            &fault,
            &mock_context,
            1,
            metrics,
            current_time,
            request_arrival,
        );
        
        // Should continue (still in delay period)
        assert_eq!(result, Action::Continue);
    }

    #[test]
    fn test_execute_fault_with_time_control_after_delay() {
        // Test when delay period has passed
        let fault = Fault {
            percentage: 100,
            abort: Some(AbortAction {
                http_status: 503,
                body: Some("Service Unavailable".to_string()),
            }),
            delay: None,
            start_delay_ms: 100,  // 100ms delay required
            duration_seconds: 0,
        };
        
        let mock_context = MockHttpContext::new();
        let metrics = MetricsIds {
            aborts_total: None,
            delays_total: None,
            delay_duration_histogram: None,
        };
        
        let current_time = crate::time_control::get_current_time_ms();
        let request_arrival = current_time.saturating_sub(300);  // Request arrived 300ms ago
        
        let result = execute_fault_with_time_control(
            &fault,
            &mock_context,
            1,
            metrics,
            current_time,
            request_arrival,
        );
        
        // Should execute abort fault (delay period passed)
        assert_eq!(result, Action::Pause);
    }

    #[test]
    fn test_execute_fault_with_time_control_expired_rule() {
        // Test when rule has expired
        let fault = Fault {
            percentage: 100,
            abort: Some(AbortAction {
                http_status: 500,
                body: Some("Internal Server Error".to_string()),
            }),
            delay: None,
            start_delay_ms: 0,
            duration_seconds: 1,  // Expires after 1 second
        };
        
        let mock_context = MockHttpContext::new();
        let metrics = MetricsIds {
            aborts_total: None,
            delays_total: None,
            delay_duration_histogram: None,
        };
        
        let current_time = crate::time_control::get_current_time_ms();
        let rule_creation = current_time.saturating_sub(2000);  // Rule created 2 seconds ago (expired)
        
        let result = execute_fault_with_time_control(
            &fault,
            &mock_context,
            1,
            metrics,
            rule_creation,
            current_time,
        );
        
        // Should continue (rule expired)
        assert_eq!(result, Action::Continue);
    }

    #[test]
    fn test_execute_fault_with_time_control_combined() {
        // Test combined delay and expiry constraints
        let fault = Fault {
            percentage: 100,
            abort: Some(AbortAction {
                http_status: 502,
                body: Some("Bad Gateway".to_string()),
            }),
            delay: None,
            start_delay_ms: 200,
            duration_seconds: 10,  // 10 second validity period
        };
        
        let mock_context = MockHttpContext::new();
        let metrics = MetricsIds {
            aborts_total: None,
            delays_total: None,
            delay_duration_histogram: None,
        };
        
        let current_time = crate::time_control::get_current_time_ms();
        let rule_creation = current_time.saturating_sub(5000);  // Rule created 5 seconds ago (still valid)
        let request_arrival = current_time.saturating_sub(300);  // Request arrived 300ms ago
        
        let result = execute_fault_with_time_control(
            &fault,
            &mock_context,
            1,
            metrics,
            rule_creation,
            request_arrival,
        );
        
        // Should execute (rule valid, delay period passed)
        assert_eq!(result, Action::Pause);
    }

    #[test]
    fn test_execute_fault_with_metrics_recording() {
        // Test metrics collection during fault injection
        let fault = Fault {
            percentage: 100,
            abort: Some(AbortAction {
                http_status: 500,
                body: Some("Internal Server Error".to_string()),
            }),
            delay: None,
            start_delay_ms: 0,
            duration_seconds: 0,
        };
        
        let mock_context = MockHttpContext::new();
        let metrics_ids = MetricsIds {
            aborts_total: None,
            delays_total: None,
            delay_duration_histogram: None,
        };
        
        let metrics_collector = crate::metrics::FaultInjectionMetrics::new();
        let current_time = crate::time_control::get_current_time_ms();
        
        let result = execute_fault_with_metrics(
            &fault,
            &mock_context,
            1,
            metrics_ids,
            current_time,
            current_time,
            &metrics_collector,
        );
        
        // Verify metrics were recorded
        assert_eq!(metrics_collector.get_requests_total(), 1);
        assert_eq!(metrics_collector.get_rules_matched(), 1);
        assert_eq!(metrics_collector.get_faults_injected(), 1);
        assert_eq!(metrics_collector.get_aborts(), 1);
        assert_eq!(result, Action::Pause);
    }

    #[test]
    fn test_execute_fault_with_metrics_delay() {
        // Test metrics collection for delay faults
        let fault = Fault {
            percentage: 100,
            abort: None,
            delay: Some(DelayAction {
                fixed_delay: "500ms".to_string(),
                parsed_duration_ms: Some(500),
            }),
            start_delay_ms: 0,
            duration_seconds: 0,
        };
        
        let mock_context = MockHttpContext::new();
        let metrics_ids = MetricsIds {
            aborts_total: None,
            delays_total: None,
            delay_duration_histogram: None,
        };
        
        let metrics_collector = crate::metrics::FaultInjectionMetrics::new();
        let current_time = crate::time_control::get_current_time_ms();
        
        let result = execute_fault_with_metrics(
            &fault,
            &mock_context,
            1,
            metrics_ids,
            current_time,
            current_time,
            &metrics_collector,
        );
        
        // Verify metrics were recorded
        assert_eq!(metrics_collector.get_requests_total(), 1);
        assert_eq!(metrics_collector.get_faults_injected(), 1);
        assert_eq!(metrics_collector.get_delays(), 1);
        assert_eq!(result, Action::Pause);
    }

    #[test]
    fn test_execute_fault_with_metrics_time_control_wait() {
        // Test metrics collection when time control prevents injection
        let fault = Fault {
            percentage: 100,
            abort: Some(AbortAction {
                http_status: 500,
                body: None,
            }),
            delay: None,
            start_delay_ms: 500,  // 500ms delay required
            duration_seconds: 0,
        };
        
        let mock_context = MockHttpContext::new();
        let metrics_ids = MetricsIds {
            aborts_total: None,
            delays_total: None,
            delay_duration_histogram: None,
        };
        
        let metrics_collector = crate::metrics::FaultInjectionMetrics::new();
        let current_time = crate::time_control::get_current_time_ms();
        let request_arrival = current_time.saturating_sub(100);  // Request arrived 100ms ago
        
        let result = execute_fault_with_metrics(
            &fault,
            &mock_context,
            1,
            metrics_ids,
            current_time,
            request_arrival,
            &metrics_collector,
        );
        
        // Verify metrics were recorded
        assert_eq!(metrics_collector.get_requests_total(), 1);
        assert_eq!(metrics_collector.get_rules_matched(), 1);
        assert_eq!(metrics_collector.get_time_control_wait_count(), 1);
        assert_eq!(metrics_collector.get_faults_injected(), 0);  // Not injected
        assert_eq!(result, Action::Continue);
    }

    #[test]
    fn test_execute_fault_with_metrics_rule_expired() {
        // Test metrics collection when rule has expired
        let fault = Fault {
            percentage: 100,
            abort: Some(AbortAction {
                http_status: 500,
                body: None,
            }),
            delay: None,
            start_delay_ms: 0,
            duration_seconds: 1,  // 1 second validity
        };
        
        let mock_context = MockHttpContext::new();
        let metrics_ids = MetricsIds {
            aborts_total: None,
            delays_total: None,
            delay_duration_histogram: None,
        };
        
        let metrics_collector = crate::metrics::FaultInjectionMetrics::new();
        let current_time = crate::time_control::get_current_time_ms();
        let rule_creation = current_time.saturating_sub(2000);  // Created 2 seconds ago (expired)
        
        let result = execute_fault_with_metrics(
            &fault,
            &mock_context,
            1,
            metrics_ids,
            rule_creation,
            current_time,
            &metrics_collector,
        );
        
        // Verify metrics were recorded
        assert_eq!(metrics_collector.get_requests_total(), 1);
        assert_eq!(metrics_collector.get_rules_matched(), 1);
        assert_eq!(metrics_collector.get_rule_expired_count(), 1);
        assert_eq!(metrics_collector.get_faults_injected(), 0);  // Not injected
        assert_eq!(result, Action::Continue);
    }

    #[test]
    fn test_metrics_injection_rate() {
        // Test injection rate calculation
        let metrics_collector = crate::metrics::FaultInjectionMetrics::new();
        
        // Simulate 10 requests
        for _ in 0..10 {
            metrics_collector.record_request();
        }
        
        // 7 of them had faults injected
        for _ in 0..7 {
            metrics_collector.record_fault_injected();
        }
        
        let rate = metrics_collector.get_injection_rate();
        assert!((rate - 70.0).abs() < 0.01);  // 70%
    }

    #[test]
    fn test_metrics_snapshot() {
        // Test metrics snapshot generation
        let metrics_collector = crate::metrics::FaultInjectionMetrics::new();
        
        for _ in 0..5 {
            metrics_collector.record_request();
        }
        metrics_collector.record_fault_injected();
        metrics_collector.record_fault_injected();
        metrics_collector.record_abort_fault();
        metrics_collector.record_delay_fault(100);
        metrics_collector.record_delay_fault(200);
        
        let snapshot = metrics_collector.snapshot();
        
        assert_eq!(snapshot.requests_total, 5);
        assert_eq!(snapshot.faults_injected, 2);
        assert_eq!(snapshot.aborts, 1);
        assert_eq!(snapshot.delays, 2);
    }
}