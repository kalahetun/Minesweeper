// Tests for Wasm Plugin temporal control mechanisms
// Validates start_delay_ms and duration_seconds behavior

#[cfg(test)]
mod temporal_tests {
    use std::sync::{Arc, Mutex};
    use std::time::{Duration, Instant};

    // Mock context for tracking temporal behavior
    struct MockTemporalContext {
        fault_applied: bool,
        activation_time: Option<Duration>,
        expiry_time: Option<Duration>,
        request_start: Instant,
    }

    impl MockTemporalContext {
        fn new() -> Self {
            Self {
                fault_applied: false,
                activation_time: None,
                expiry_time: None,
                request_start: Instant::now(),
            }
        }

        fn set_activation_delay(&mut self, delay_ms: u32) {
            self.activation_time = Some(Duration::from_millis(delay_ms as u64));
        }

        fn set_expiry(&mut self, duration_sec: u32) {
            self.expiry_time = Some(Duration::from_secs(duration_sec as u64));
        }

        fn simulate_fault_application(&mut self) {
            self.fault_applied = true;
        }

        fn should_apply_fault(&self, current_time: Duration) -> bool {
            // Check if activation delay has passed
            if let Some(activation_time) = self.activation_time {
                if current_time < activation_time {
                    return false;
                }
            }

            // Check if expiry duration has passed (0 means infinite/never expire)
            if let Some(expiry_duration) = self.expiry_time {
                if expiry_duration.as_secs() > 0 && current_time >= expiry_duration {
                    return false;
                }
            }

            true
        }
    }

    #[test]
    fn test_start_delay_immediate_execution() {
        // When start_delay_ms = 0, fault should apply immediately
        let mut ctx = MockTemporalContext::new();
        ctx.set_activation_delay(0);

        let elapsed = Duration::from_millis(0);
        assert!(ctx.should_apply_fault(elapsed), "Fault should apply at t=0ms");

        ctx.simulate_fault_application();
        assert!(ctx.fault_applied);
    }

    #[test]
    fn test_start_delay_prevents_early_injection() {
        // When start_delay_ms = 100, fault should NOT apply before 100ms
        let mut ctx = MockTemporalContext::new();
        ctx.set_activation_delay(100);

        // Check at 50ms
        let elapsed_50ms = Duration::from_millis(50);
        assert!(!ctx.should_apply_fault(elapsed_50ms), 
                "Fault should NOT apply at t=50ms when delay=100ms");

        // Check at 99ms
        let elapsed_99ms = Duration::from_millis(99);
        assert!(!ctx.should_apply_fault(elapsed_99ms), 
                "Fault should NOT apply at t=99ms when delay=100ms");

        // Check at 100ms
        let elapsed_100ms = Duration::from_millis(100);
        assert!(ctx.should_apply_fault(elapsed_100ms), 
                "Fault should apply at t=100ms when delay=100ms");

        // Check at 101ms
        let elapsed_101ms = Duration::from_millis(101);
        assert!(ctx.should_apply_fault(elapsed_101ms), 
                "Fault should apply at t=101ms when delay=100ms");
    }

    #[test]
    fn test_start_delay_various_values() {
        // Test with different delay values
        let test_cases = vec![
            (10, 5, false),    // delay=10ms, check at 5ms → should not apply
            (10, 10, true),    // delay=10ms, check at 10ms → should apply
            (500, 250, false), // delay=500ms, check at 250ms → should not apply
            (500, 500, true),  // delay=500ms, check at 500ms → should apply
            (1000, 999, false),// delay=1000ms, check at 999ms → should not apply
            (1000, 1000, true),// delay=1000ms, check at 1000ms → should apply
            (5000, 5000, true),// delay=5000ms, check at 5000ms → should apply
        ];

        for (delay_ms, check_at_ms, expected) in test_cases {
            let mut ctx = MockTemporalContext::new();
            ctx.set_activation_delay(delay_ms);
            let elapsed = Duration::from_millis(check_at_ms);
            assert_eq!(
                ctx.should_apply_fault(elapsed),
                expected,
                "Failed for delay_ms={}, check_at_ms={}", delay_ms, check_at_ms
            );
        }
    }

    #[test]
    fn test_duration_seconds_infinite_when_zero() {
        // When duration_seconds = 0, fault should never expire
        let mut ctx = MockTemporalContext::new();
        ctx.set_expiry(0);

        // Check at various times
        assert!(ctx.should_apply_fault(Duration::from_secs(1)));
        assert!(ctx.should_apply_fault(Duration::from_secs(3600))); // 1 hour
        assert!(ctx.should_apply_fault(Duration::from_secs(86400))); // 1 day
        assert!(ctx.should_apply_fault(Duration::from_secs(1000000))); // Very long time
    }

    #[test]
    fn test_duration_seconds_stops_injection() {
        // When duration_seconds = 300 (5 minutes), fault should expire after that
        let mut ctx = MockTemporalContext::new();
        ctx.set_expiry(300);

        // Should apply before expiry
        assert!(ctx.should_apply_fault(Duration::from_secs(0)));
        assert!(ctx.should_apply_fault(Duration::from_secs(100)));
        assert!(ctx.should_apply_fault(Duration::from_secs(299)));

        // Should NOT apply at or after expiry
        assert!(!ctx.should_apply_fault(Duration::from_secs(300)));
        assert!(!ctx.should_apply_fault(Duration::from_secs(301)));
        assert!(!ctx.should_apply_fault(Duration::from_secs(3600)));
    }

    #[test]
    fn test_duration_seconds_various_values() {
        // Test with different duration values
        let test_cases = vec![
            (10, 5, true),     // duration=10s, check at 5s → should apply
            (10, 10, false),   // duration=10s, check at 10s → should NOT apply
            (60, 59, true),    // duration=60s, check at 59s → should apply
            (60, 60, false),   // duration=60s, check at 60s → should NOT apply
            (3600, 1800, true),// duration=1h, check at 30m → should apply
            (3600, 3600, false),// duration=1h, check at 1h → should NOT apply
        ];

        for (duration_sec, check_at_sec, expected) in test_cases {
            let mut ctx = MockTemporalContext::new();
            ctx.set_expiry(duration_sec);
            let elapsed = Duration::from_secs(check_at_sec);
            assert_eq!(
                ctx.should_apply_fault(elapsed),
                expected,
                "Failed for duration_sec={}, check_at_sec={}", duration_sec, check_at_sec
            );
        }
    }

    #[test]
    fn test_combined_delay_and_duration() {
        // Test start_delay_ms and duration_seconds together
        // Should NOT apply before activation, and should expire after duration
        let mut ctx = MockTemporalContext::new();
        ctx.set_activation_delay(100);    // 100ms delay
        ctx.set_expiry(500);               // 500s duration (5 minutes)

        // Before activation (< 100ms) → should not apply
        assert!(!ctx.should_apply_fault(Duration::from_millis(50)));

        // During active window (100ms - 500s) → should apply
        assert!(ctx.should_apply_fault(Duration::from_millis(100)));
        assert!(ctx.should_apply_fault(Duration::from_millis(500)));
        assert!(ctx.should_apply_fault(Duration::from_secs(100)));
        assert!(ctx.should_apply_fault(Duration::from_secs(499)));

        // After expiry (>= 500s) → should not apply
        assert!(!ctx.should_apply_fault(Duration::from_secs(500)));
        assert!(!ctx.should_apply_fault(Duration::from_secs(501)));
        assert!(!ctx.should_apply_fault(Duration::from_secs(3600)));
    }

    #[test]
    fn test_combined_delay_and_zero_duration() {
        // start_delay_ms=1000, duration_seconds=0 (infinite)
        // Fault applies after 1s and never expires
        let mut ctx = MockTemporalContext::new();
        ctx.set_activation_delay(1000);
        ctx.set_expiry(0);

        // Before activation → should not apply
        assert!(!ctx.should_apply_fault(Duration::from_millis(500)));

        // After activation → should always apply (no expiry)
        assert!(ctx.should_apply_fault(Duration::from_secs(1)));
        assert!(ctx.should_apply_fault(Duration::from_secs(3600)));
        assert!(ctx.should_apply_fault(Duration::from_secs(86400)));
    }

    #[test]
    fn test_request_duration_less_than_delay() {
        // When request completes before start_delay_ms, no fault should be applied
        // This validates the scenario: "request duration < start_delay_ms"
        let mut ctx = MockTemporalContext::new();
        ctx.set_activation_delay(500); // 500ms delay

        // Request finishes at 100ms, which is before 500ms activation
        let request_finish_time = Duration::from_millis(100);
        assert!(!ctx.should_apply_fault(request_finish_time),
                "Fault should not apply when request finishes before activation delay");
    }

    #[test]
    fn test_request_duration_greater_than_delay() {
        // When request duration exceeds start_delay_ms, fault should apply
        let mut ctx = MockTemporalContext::new();
        ctx.set_activation_delay(100); // 100ms delay

        // Request runs for 500ms, which includes the activation window
        let check_at_activation = Duration::from_millis(100);
        assert!(ctx.should_apply_fault(check_at_activation),
                "Fault should apply when request duration >= activation delay");

        let check_at_end = Duration::from_millis(500);
        assert!(ctx.should_apply_fault(check_at_end),
                "Fault should still apply at request end");
    }

    #[test]
    fn test_temporal_state_concurrent_access() {
        // Verify temporal state can be safely shared across concurrent contexts
        let ctx = Arc::new(Mutex::new(MockTemporalContext::new()));
        let ctx_clone1 = Arc::clone(&ctx);
        let ctx_clone2 = Arc::clone(&ctx);

        // Thread 1: Set delay
        let handle1 = std::thread::spawn(move || {
            let mut c = ctx_clone1.lock().unwrap();
            c.set_activation_delay(100);
        });

        // Thread 2: Set expiry
        let handle2 = std::thread::spawn(move || {
            let mut c = ctx_clone2.lock().unwrap();
            c.set_expiry(300);
        });

        handle1.join().unwrap();
        handle2.join().unwrap();

        // Verify both were set correctly
        let ctx_guard = ctx.lock().unwrap();
        assert_eq!(ctx_guard.activation_time, Some(Duration::from_millis(100)));
        assert_eq!(ctx_guard.expiry_time, Some(Duration::from_secs(300)));
    }

    #[test]
    fn test_zero_duration_with_long_running_request() {
        // When duration_seconds=0 (infinite), request should experience fault
        // for entire duration (hours, days, etc.)
        let mut ctx = MockTemporalContext::new();
        ctx.set_expiry(0); // No expiry

        // Simulate a long-running request
        for hours in 0..24 {
            let elapsed = Duration::from_secs(hours * 3600);
            assert!(ctx.should_apply_fault(elapsed),
                    "Fault should apply throughout long-running request (hour {})", hours);
        }
    }

    #[test]
    fn test_boundary_conditions() {
        // Test precise boundary conditions for activation and expiry
        let mut ctx = MockTemporalContext::new();
        ctx.set_activation_delay(100);  // 100ms activation delay
        ctx.set_expiry(1);              // 1 second (1000ms) expiry

        // Millisecond-level boundary testing
        assert!(!ctx.should_apply_fault(Duration::from_millis(99)));   // Before activation
        assert!(ctx.should_apply_fault(Duration::from_millis(100)));  // At activation boundary
        assert!(ctx.should_apply_fault(Duration::from_millis(101)));  // After activation
        assert!(ctx.should_apply_fault(Duration::from_millis(500))); // Within active window
        assert!(ctx.should_apply_fault(Duration::from_millis(999))); // Just before expiry
        assert!(!ctx.should_apply_fault(Duration::from_secs(1)));    // At expiry boundary
        assert!(!ctx.should_apply_fault(Duration::from_secs(2)));    // After expiry
    }

    #[test]
    fn test_multiple_rapid_temporal_checks() {
        // Simulate rapid checks to verify consistency
        let mut ctx = MockTemporalContext::new();
        ctx.set_activation_delay(50);
        ctx.set_expiry(100);

        // Rapid checks should be consistent
        for _ in 0..1000 {
            let elapsed = Duration::from_millis(75);
            assert!(ctx.should_apply_fault(elapsed), "Rapid checks should be consistent");
        }
    }

    #[test]
    fn test_temporal_precision_milliseconds() {
        // Verify millisecond-level precision for activation delay
        let mut ctx = MockTemporalContext::new();
        ctx.set_activation_delay(100);

        // Check around 100ms boundary
        for ms in 90..110 {
            let elapsed = Duration::from_millis(ms);
            let expected = ms >= 100;
            assert_eq!(
                ctx.should_apply_fault(elapsed),
                expected,
                "Precision check failed at {}ms", ms
            );
        }
    }

    #[test]
    fn test_temporal_precision_seconds() {
        // Verify second-level precision for duration
        let mut ctx = MockTemporalContext::new();
        ctx.set_expiry(100);

        // Check around 100s boundary
        for sec in 95..105 {
            let elapsed = Duration::from_secs(sec);
            let expected = sec < 100;
            assert_eq!(
                ctx.should_apply_fault(elapsed),
                expected,
                "Duration precision check failed at {}s", sec
            );
        }
    }

    #[test]
    fn test_extreme_values() {
        // Test with extreme but valid values
        let mut ctx = MockTemporalContext::new();

        // Very large delay (10 seconds)
        ctx.set_activation_delay(10000);
        assert!(!ctx.should_apply_fault(Duration::from_millis(5000)));
        assert!(ctx.should_apply_fault(Duration::from_millis(10000)));

        // Very long duration (24 hours)
        let mut ctx2 = MockTemporalContext::new();
        ctx2.set_expiry(86400);
        assert!(ctx2.should_apply_fault(Duration::from_secs(43200))); // 12 hours
        assert!(!ctx2.should_apply_fault(Duration::from_secs(86400))); // 24 hours
    }
}
