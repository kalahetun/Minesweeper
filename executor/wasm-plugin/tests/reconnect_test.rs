// WASM Plugin Reconnect Logic Tests
// Integration tests for reconnection logic

#[cfg(test)]
mod reconnect_tests {
    use std::time::{Duration, Instant};

    /// Error type classification
    #[derive(Debug, Clone, Copy, PartialEq, Eq)]
    pub enum ErrorType {
        Temporary,
        Permanent,
        Unknown,
    }

    impl ErrorType {
        pub fn from_status_code(status: u32) -> Self {
            match status {
                500..=599 => ErrorType::Temporary,
                400..=499 => ErrorType::Permanent,
                _ => ErrorType::Unknown,
            }
        }
    }

    /// Simple reconnection manager for testing
    pub struct ReconnectManager {
        pub attempts: u32,
        pub initial_delay: Duration,
        pub max_delay: Duration,
        pub max_attempts: u32,
        pub current_delay: Duration,
        pub is_reconnecting: bool,
    }

    impl ReconnectManager {
        pub fn new() -> Self {
            Self::with_config(Duration::from_secs(1), Duration::from_secs(30), 5)
        }

        pub fn with_config(
            initial_delay: Duration,
            max_delay: Duration,
            max_attempts: u32,
        ) -> Self {
            Self {
                attempts: 0,
                initial_delay,
                max_delay,
                max_attempts,
                current_delay: initial_delay,
                is_reconnecting: false,
            }
        }

        pub fn on_failure(&mut self) -> Option<Duration> {
            self.is_reconnecting = true;

            if self.attempts >= self.max_attempts {
                return None;
            }

            self.attempts += 1;
            let delay = self.current_delay;

            // Calculate next delay (exponential backoff with cap)
            let next_delay = self.current_delay.as_millis() as u64 * 2;
            let next_delay_ms = std::cmp::min(next_delay, self.max_delay.as_millis() as u64);
            self.current_delay = Duration::from_millis(next_delay_ms);

            Some(delay)
        }

        pub fn on_success(&mut self) {
            self.attempts = 0;
            self.current_delay = self.initial_delay;
            self.is_reconnecting = false;
        }

        pub fn is_reconnecting(&self) -> bool {
            self.is_reconnecting
        }

        pub fn get_attempts(&self) -> u32 {
            self.attempts
        }
    }

    #[test]
    fn test_reconnect_state_machine() {
        let mut manager = ReconnectManager::new();

        // Initial state
        assert_eq!(manager.attempts, 0);
        assert!(!manager.is_reconnecting());

        // First failure
        let _ = manager.on_failure();
        assert_eq!(manager.attempts, 1);
        assert!(manager.is_reconnecting());

        // Success resets state
        manager.on_success();
        assert_eq!(manager.attempts, 0);
        assert!(!manager.is_reconnecting());
    }

    #[test]
    fn test_exponential_backoff_progression() {
        let mut manager =
            ReconnectManager::with_config(Duration::from_millis(100), Duration::from_secs(10), 5);

        let delays: Vec<_> = (0..5).filter_map(|_| manager.on_failure()).collect();

        assert_eq!(delays[0], Duration::from_millis(100));
        assert_eq!(delays[1], Duration::from_millis(200));
        assert_eq!(delays[2], Duration::from_millis(400));
        assert_eq!(delays[3], Duration::from_millis(800));
        assert_eq!(delays[4], Duration::from_millis(1600));
    }

    #[test]
    fn test_max_delay_capped() {
        let mut manager = ReconnectManager::with_config(
            Duration::from_millis(100),
            Duration::from_millis(300), // Cap at 300ms
            10,
        );

        let mut last_delay = Duration::from_millis(0);
        for _ in 0..10 {
            if let Some(delay) = manager.on_failure() {
                assert!(delay <= manager.max_delay);
                last_delay = delay;
            }
        }

        assert_eq!(last_delay, Duration::from_millis(300));
    }

    #[test]
    fn test_max_attempts_enforced() {
        let mut manager =
            ReconnectManager::with_config(Duration::from_millis(50), Duration::from_secs(5), 3);

        // Should succeed 3 times
        assert!(manager.on_failure().is_some());
        assert!(manager.on_failure().is_some());
        assert!(manager.on_failure().is_some());

        // 4th should fail
        assert!(manager.on_failure().is_none());
    }

    #[test]
    fn test_reset_after_success() {
        let mut manager =
            ReconnectManager::with_config(Duration::from_millis(100), Duration::from_secs(10), 5);

        // Accumulate failures
        manager.on_failure();
        manager.on_failure();
        manager.on_failure();
        assert_eq!(manager.attempts, 3);
        assert!(manager.is_reconnecting());

        // Success resets everything
        manager.on_success();
        assert_eq!(manager.attempts, 0);
        assert!(!manager.is_reconnecting());
        assert_eq!(manager.current_delay, Duration::from_millis(100));
    }

    #[test]
    fn test_error_classification() {
        // Server errors = Temporary (retryable)
        assert_eq!(ErrorType::from_status_code(500), ErrorType::Temporary);
        assert_eq!(ErrorType::from_status_code(502), ErrorType::Temporary);
        assert_eq!(ErrorType::from_status_code(503), ErrorType::Temporary);
        assert_eq!(ErrorType::from_status_code(599), ErrorType::Temporary);

        // Client errors = Permanent (not retryable)
        assert_eq!(ErrorType::from_status_code(400), ErrorType::Permanent);
        assert_eq!(ErrorType::from_status_code(401), ErrorType::Permanent);
        assert_eq!(ErrorType::from_status_code(404), ErrorType::Permanent);
        assert_eq!(ErrorType::from_status_code(429), ErrorType::Permanent);
        assert_eq!(ErrorType::from_status_code(499), ErrorType::Permanent);

        // Others = Unknown
        assert_eq!(ErrorType::from_status_code(100), ErrorType::Unknown);
        assert_eq!(ErrorType::from_status_code(200), ErrorType::Unknown);
        assert_eq!(ErrorType::from_status_code(301), ErrorType::Unknown);
    }

    #[test]
    fn test_multiple_failure_recovery_cycles() {
        let mut manager =
            ReconnectManager::with_config(Duration::from_millis(50), Duration::from_secs(5), 3);

        for cycle in 0..3 {
            // Fail twice
            let _ = manager.on_failure();
            let _ = manager.on_failure();
            assert_eq!(manager.attempts, 2);

            // Recover
            manager.on_success();
            assert_eq!(manager.attempts, 0);
        }
    }

    #[test]
    fn test_delay_monotonically_increases() {
        let mut manager =
            ReconnectManager::with_config(Duration::from_millis(50), Duration::from_secs(10), 10);

        let mut prev_delay = Duration::from_millis(0);
        for _ in 0..8 {
            if let Some(delay) = manager.on_failure() {
                assert!(
                    delay >= prev_delay,
                    "Delays should be monotonically increasing"
                );
                prev_delay = delay;
            }
        }
    }

    #[test]
    fn test_custom_initial_delay() {
        let manager =
            ReconnectManager::with_config(Duration::from_millis(200), Duration::from_secs(30), 5);

        assert_eq!(manager.initial_delay, Duration::from_millis(200));
        assert_eq!(manager.current_delay, Duration::from_millis(200));
    }

    #[test]
    fn test_attempts_counter() {
        let mut manager = ReconnectManager::new();

        for expected in 1..=5 {
            manager.on_failure();
            assert_eq!(manager.get_attempts(), expected);
        }
    }

    #[test]
    fn test_reconnecting_flag_lifecycle() {
        let mut manager = ReconnectManager::new();

        // Initial: not reconnecting
        assert!(!manager.is_reconnecting());

        // After 1st failure: reconnecting
        manager.on_failure();
        assert!(manager.is_reconnecting());

        // Still reconnecting after 2nd failure
        manager.on_failure();
        assert!(manager.is_reconnecting());

        // After success: not reconnecting
        manager.on_success();
        assert!(!manager.is_reconnecting());
    }

    #[test]
    fn test_graceful_degradation_under_sustained_failures() {
        let mut manager =
            ReconnectManager::with_config(Duration::from_millis(100), Duration::from_secs(5), 5);

        // Simulate sustained failures
        for attempt in 1..=5 {
            let result = manager.on_failure();
            assert!(
                result.is_some(),
                "Should return delay for attempt {}",
                attempt
            );
            assert_eq!(manager.attempts, attempt);
        }

        // After max attempts, should stop retrying
        let result = manager.on_failure();
        assert!(result.is_none(), "Should stop retrying after max attempts");
        assert_eq!(manager.attempts, 5); // Attempts don't increment beyond max
    }

    #[test]
    fn test_recovery_from_failed_state() {
        let mut manager =
            ReconnectManager::with_config(Duration::from_millis(50), Duration::from_secs(5), 3);

        // Fail max attempts
        manager.on_failure();
        manager.on_failure();
        manager.on_failure();
        assert!(manager.on_failure().is_none()); // Max reached

        // Try to recover
        manager.on_success();
        assert_eq!(manager.attempts, 0);

        // Should be able to retry again
        assert!(manager.on_failure().is_some());
    }

    #[test]
    fn test_configuration_persistence() {
        let initial_delay = Duration::from_millis(250);
        let max_delay = Duration::from_secs(20);
        let max_attempts = 7;

        let mut manager = ReconnectManager::with_config(initial_delay, max_delay, max_attempts);

        // Configuration should persist
        assert_eq!(manager.initial_delay, initial_delay);
        assert_eq!(manager.max_delay, max_delay);
        assert_eq!(manager.max_attempts, max_attempts);

        // After operations, configuration should remain unchanged
        manager.on_failure();
        manager.on_failure();
        assert_eq!(manager.max_attempts, max_attempts);

        manager.on_success();
        assert_eq!(manager.initial_delay, initial_delay);
    }
}
