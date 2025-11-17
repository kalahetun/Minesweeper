use log::{debug, warn, error};
use std::time::Duration;

/// Error classification for reconnection strategy (M2 improvement)
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ErrorType {
    /// Temporary/retryable errors: timeouts, network issues, 5xx
    Temporary,
    /// Permanent/non-retryable errors: 4xx client errors
    Permanent,
    /// Unknown error type
    Unknown,
}

impl ErrorType {
    /// Classify HTTP status code into error type
    pub fn from_status_code(status: u32) -> Self {
        match status {
            // 5xx server errors and timeouts are retryable
            500..=599 => ErrorType::Temporary,
            // 4xx client errors are generally not retryable
            400..=499 => ErrorType::Permanent,
            // Success and redirects
            _ => ErrorType::Unknown,
        }
    }
}

/// 重连状态管理器，实现指数退避重连机制
pub struct ReconnectManager {
    /// 当前重连尝试次数
    pub attempts: u32,
    /// 初始重连延迟
    pub initial_delay: Duration,
    /// 最大重连延迟
    pub max_delay: Duration,
    /// 最大重连尝试次数
    pub max_attempts: u32,
    /// 当前重连延迟
    pub current_delay: Duration,
    /// 是否正在重连过程中
    pub is_reconnecting: bool,
}

impl ReconnectManager {
    /// 创建新的重连管理器
    pub fn new() -> Self {
        Self {
            attempts: 0,
            initial_delay: Duration::from_secs(1),
            max_delay: Duration::from_secs(60),
            max_attempts: 10,
            current_delay: Duration::from_secs(1),
            is_reconnecting: false,
        }
    }

    /// 创建自定义配置的重连管理器
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

    /// 处理连接失败，计算下次重连延迟
    /// 
    /// 改进 (M2): 支持错误分类，对临时错误和永久错误采取不同策略
    /// - 临时错误（5xx、超时）：使用指数退避重连
    /// - 永久错误（4xx）：快速放弃或使用较长延迟
    pub fn on_failure(&mut self) -> Option<Duration> {
        self.on_failure_with_error_type(ErrorType::Temporary)
    }
    
    /// Handle connection failure with error classification (M2)
    pub fn on_failure_with_error_type(&mut self, error_type: ErrorType) -> Option<Duration> {
        self.attempts += 1;
        
        // For permanent errors, reduce max attempts or fail immediately
        let max_attempts = match error_type {
            ErrorType::Permanent => {
                // 4xx errors are unlikely to be fixed by retry
                warn!("Permanent error detected (4xx), reducing max attempts");
                std::cmp::min(self.max_attempts, 2)
            }
            ErrorType::Temporary => self.max_attempts,
            ErrorType::Unknown => self.max_attempts,
        };
        
        if self.attempts > max_attempts {
            error!(
                "Max reconnection attempts reached: {}/{} (error type: {:?})",
                self.attempts, max_attempts, error_type
            );
            return None;
        }

        // 指数退避算法: delay = min(initial_delay * 2^attempts, max_delay)
        let exponential_delay = self.initial_delay
            .checked_mul(2_u32.checked_pow(self.attempts.saturating_sub(1)).unwrap_or(1))
            .unwrap_or(self.max_delay);
        
        self.current_delay = std::cmp::min(exponential_delay, self.max_delay);
        self.is_reconnecting = true;

        warn!(
            "Connection failed ({:?}), scheduling reconnect attempt {}/{} in {:?}",
            error_type, self.attempts, max_attempts, self.current_delay
        );

        Some(self.current_delay)
    }

    /// 处理连接成功，重置重连状态
    pub fn on_success(&mut self) {
        if self.attempts > 0 {
            debug!(
                "Reconnection successful after {} attempts",
                self.attempts
            );
        }

        self.attempts = 0;
        self.current_delay = self.initial_delay;
        self.is_reconnecting = false;
    }

    /// 检查是否应该尝试重连
    pub fn should_reconnect(&self) -> bool {
        self.attempts < self.max_attempts
    }

    /// 获取当前重连延迟
    pub fn get_current_delay(&self) -> Duration {
        self.current_delay
    }

    /// 检查是否正在重连中
    pub fn is_reconnecting(&self) -> bool {
        self.is_reconnecting
    }

    /// 获取当前尝试次数
    pub fn get_attempts(&self) -> u32 {
        self.attempts
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_exponential_backoff() {
        let mut manager = ReconnectManager::with_config(
            Duration::from_millis(100),
            Duration::from_secs(10),
            5,
        );

        // 第一次失败
        let delay1 = manager.on_failure().unwrap();
        assert_eq!(delay1, Duration::from_millis(100));
        assert_eq!(manager.get_attempts(), 1);

        // 第二次失败
        let delay2 = manager.on_failure().unwrap();
        assert_eq!(delay2, Duration::from_millis(200));
        assert_eq!(manager.get_attempts(), 2);

        // 第三次失败
        let delay3 = manager.on_failure().unwrap();
        assert_eq!(delay3, Duration::from_millis(400));
        assert_eq!(manager.get_attempts(), 3);

        // 成功后重置
        manager.on_success();
        assert_eq!(manager.get_attempts(), 0);
        assert!(!manager.is_reconnecting());
    }

    #[test]
    fn test_max_attempts() {
        let mut manager = ReconnectManager::with_config(
            Duration::from_millis(100),
            Duration::from_secs(10),
            2,
        );

        // 前两次失败应该返回延迟
        assert!(manager.on_failure().is_some());
        assert!(manager.on_failure().is_some());
        
        // 第三次失败应该返回 None（超过最大尝试次数）
        assert!(manager.on_failure().is_none());
    }

    #[test]
    fn test_max_delay() {
        let mut manager = ReconnectManager::with_config(
            Duration::from_millis(100),
            Duration::from_millis(300),
            10,
        );

        manager.on_failure(); // 100ms
        manager.on_failure(); // 200ms
        let delay3 = manager.on_failure().unwrap(); // should be capped at 300ms
        assert_eq!(delay3, Duration::from_millis(300));
    }

    #[test]
    fn test_reconnect_state_transitions() {
        let mut manager = ReconnectManager::new();
        
        // 初始状态不在重连
        assert!(!manager.is_reconnecting());
        
        // 失败后进入重连状态
        let _ = manager.on_failure();
        assert!(manager.is_reconnecting());
        
        // 成功后退出重连状态
        manager.on_success();
        assert!(!manager.is_reconnecting());
    }

    #[test]
    fn test_error_type_classification() {
        // 5xx 错误是临时的
        assert_eq!(ErrorType::from_status_code(500), ErrorType::Temporary);
        assert_eq!(ErrorType::from_status_code(503), ErrorType::Temporary);
        assert_eq!(ErrorType::from_status_code(504), ErrorType::Temporary);
        
        // 4xx 错误是永久的
        assert_eq!(ErrorType::from_status_code(400), ErrorType::Permanent);
        assert_eq!(ErrorType::from_status_code(401), ErrorType::Permanent);
        assert_eq!(ErrorType::from_status_code(404), ErrorType::Permanent);
        assert_eq!(ErrorType::from_status_code(429), ErrorType::Permanent);
        
        // 其他状态代码是未知的
        assert_eq!(ErrorType::from_status_code(200), ErrorType::Unknown);
        assert_eq!(ErrorType::from_status_code(301), ErrorType::Unknown);
    }

    #[test]
    fn test_multiple_failure_recovery_cycles() {
        let mut manager = ReconnectManager::with_config(
            Duration::from_millis(50),
            Duration::from_secs(5),
            3,
        );

        // 第一个失败→恢复周期
        assert!(manager.on_failure().is_some());
        assert!(manager.is_reconnecting());
        manager.on_success();
        assert!(!manager.is_reconnecting());
        assert_eq!(manager.get_attempts(), 0);

        // 第二个失败→恢复周期
        assert!(manager.on_failure().is_some());
        assert!(manager.is_reconnecting());
        manager.on_success();
        assert!(!manager.is_reconnecting());
    }

    #[test]
    fn test_delay_values_increase_exponentially() {
        let mut manager = ReconnectManager::with_config(
            Duration::from_millis(100),
            Duration::from_secs(60),
            10,
        );

        let delay1 = manager.on_failure().unwrap();
        let delay2 = manager.on_failure().unwrap();
        let delay3 = manager.on_failure().unwrap();
        let delay4 = manager.on_failure().unwrap();

        // 验证指数增长：每次应该是前一次的两倍（直到达到最大延迟）
        assert_eq!(delay1, Duration::from_millis(100));
        assert_eq!(delay2, Duration::from_millis(200));
        assert_eq!(delay3, Duration::from_millis(400));
        assert_eq!(delay4, Duration::from_millis(800));

        // 验证递增顺序
        assert!(delay1 < delay2);
        assert!(delay2 < delay3);
        assert!(delay3 < delay4);
    }

    #[test]
    fn test_attempts_counter_increments() {
        let mut manager = ReconnectManager::with_config(
            Duration::from_millis(100),
            Duration::from_secs(10),
            5,
        );

        for i in 1..=5 {
            manager.on_failure();
            assert_eq!(manager.get_attempts(), i);
        }
    }

    #[test]
    fn test_success_resets_attempts() {
        let mut manager = ReconnectManager::with_config(
            Duration::from_millis(100),
            Duration::from_secs(10),
            5,
        );

        // 累积失败
        manager.on_failure();
        manager.on_failure();
        manager.on_failure();
        assert_eq!(manager.get_attempts(), 3);

        // 成功重置
        manager.on_success();
        assert_eq!(manager.get_attempts(), 0);
    }

    #[test]
    fn test_custom_config() {
        let manager = ReconnectManager::with_config(
            Duration::from_millis(200),
            Duration::from_secs(30),
            4,
        );

        assert_eq!(manager.initial_delay, Duration::from_millis(200));
        assert_eq!(manager.max_delay, Duration::from_secs(30));
        assert_eq!(manager.max_attempts, 4);
    }

    #[test]
    fn test_long_backoff_sequence() {
        let mut manager = ReconnectManager::with_config(
            Duration::from_millis(10),
            Duration::from_secs(10),
            8,
        );

        let mut prev_delay = Duration::from_millis(0);
        for _ in 0..8 {
            if let Some(delay) = manager.on_failure() {
                // 每个延迟应该大于等于前一个（除了达到最大延迟后）
                assert!(delay >= prev_delay);
                prev_delay = delay;
            } else {
                panic!("Expected Some(delay), got None before max attempts");
            }
        }

        // 超过最大尝试次数应该返回 None
        assert!(manager.on_failure().is_none());
    }

    #[test]
    fn test_rapid_success_failure_cycles() {
        let mut manager = ReconnectManager::with_config(
            Duration::from_millis(50),
            Duration::from_secs(5),
            3,
        );

        for _ in 0..5 {
            // 失败一次
            assert!(manager.on_failure().is_some());
            assert_eq!(manager.get_attempts(), 1);
            
            // 立即成功
            manager.on_success();
            assert_eq!(manager.get_attempts(), 0);
        }
    }

    #[test]
    fn test_delay_never_exceeds_max() {
        let mut manager = ReconnectManager::with_config(
            Duration::from_millis(100),
            Duration::from_millis(500),
            20,
        );

        let max_delay = manager.max_delay;
        for _ in 0..20 {
            if let Some(delay) = manager.on_failure() {
                assert!(
                    delay <= max_delay,
                    "Delay {:?} exceeded max {:?}",
                    delay,
                    max_delay
                );
            }
        }
    }
}
