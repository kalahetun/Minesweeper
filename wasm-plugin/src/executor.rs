use crate::config::{Fault, AbortAction, DelayAction};
use proxy_wasm::traits::HttpContext;
use proxy_wasm::types::Action;
use log::{info, warn};

/// Execute fault injection logic
pub fn execute_fault(
    fault: &Fault,
    http_context: &dyn HttpContext,
    context_id: u32,
) -> Action {
    // Probability check
    let random_value = generate_random_percentage();
    
    info!("Fault execution - random: {}, threshold: {}", random_value, fault.percentage);
    
    if random_value >= fault.percentage {
        info!("Random value {} >= threshold {}, continuing normally", random_value, fault.percentage);
        return Action::Continue;
    }
    
    info!("Triggering fault injection for context {}", context_id);
    
    // Execute fault based on type
    if let Some(ref abort) = fault.abort {
        info!("Executing abort fault with status {}", abort.http_status);
        execute_abort(abort, http_context)
    } else if let Some(ref delay) = fault.delay {
        info!("Executing delay fault: {}", delay.fixed_delay);
        execute_delay(delay, context_id)
    } else {
        warn!("No fault action specified, continuing");
        Action::Continue
    }
}

/// Execute abort fault
fn execute_abort(abort: &AbortAction, http_context: &dyn HttpContext) -> Action {
    let body = abort.body.as_deref().unwrap_or("Fault injection: Service unavailable");
    let headers = vec![
        ("content-type", "text/plain"),
        ("x-fault-injected", "abort"),
    ];
    
    info!("Sending abort response: status={}, body_len={}", abort.http_status, body.len());
    
    // send_http_response returns (), not a Result
    http_context.send_http_response(abort.http_status, headers, Some(body.as_bytes()));
    Action::Pause
}

/// Execute delay fault (simplified for W-4)
fn execute_delay(delay: &DelayAction, context_id: u32) -> Action {
    if let Some(duration_ms) = delay.parsed_duration_ms {
        info!("Applying delay of {}ms for context {}", duration_ms, context_id);
        
        // For demonstration, use a simple blocking approach
        // In production, this would use proper async delay mechanisms with timers
        let start_time = proxy_wasm::hostcalls::get_current_time().unwrap_or(std::time::UNIX_EPOCH);
        
        // Simple busy-wait implementation for demonstration
        // Note: This is not ideal for production but shows the delay is working
        loop {
            if let Ok(current_time) = proxy_wasm::hostcalls::get_current_time() {
                let elapsed = current_time.duration_since(start_time)
                    .unwrap_or(std::time::Duration::from_secs(0))
                    .as_millis() as u64;
                
                if elapsed >= duration_ms {
                    break;
                }
            }
        }
        
        info!("Delay of {}ms applied successfully for context {}", duration_ms, context_id);
    } else {
        warn!("Delay duration not parsed correctly: {}", delay.fixed_delay);
    }
    Action::Continue
}

/// Generate random percentage (0-100)
pub fn generate_random_percentage() -> u32 {
    // Simple PRNG implementation since proxy-wasm doesn't provide rand
    static mut SEED: u64 = 1;
    unsafe {
        // Update seed using a linear congruential generator
        SEED = SEED.wrapping_mul(1103515245).wrapping_add(12345);
        
        // Add some entropy if available
        if let Ok(time) = proxy_wasm::hostcalls::get_current_time() {
            let nanos = time.duration_since(std::time::UNIX_EPOCH)
                .unwrap_or(std::time::Duration::from_secs(0))
                .as_nanos() as u64;
            SEED = SEED.wrapping_add(nanos);
        }
        
        // Return value between 0-100
        (SEED % 101) as u32
    }
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
        };
        
        let mock_context = MockHttpContext::new();
        let result = execute_fault(&fault, &mock_context, 1);
        
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
        };
        
        let mock_context = MockHttpContext::new();
        let result = execute_fault(&fault, &mock_context, 1);
        
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
        };
        
        let mock_context = MockHttpContext::new();
        let result = execute_fault(&fault, &mock_context, 1);
        
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
        };
        
        let mock_context = MockHttpContext::new();
        let result = execute_fault(&fault, &mock_context, 1);
        
        assert_eq!(result, Action::Continue);
    }
}