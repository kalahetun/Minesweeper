/// 时间控制模块
///
/// 实现 Wasm 中的故障注入时间控制逻辑，包括：
/// - 请求延迟启动 (start_delay_ms)
/// - 规则有效期检查 (duration_seconds)
///
/// 使用毫秒级精度进行时间戳计算
use std::time::UNIX_EPOCH;

/// 时间控制决策结果
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum TimeControlDecision {
    /// 应该注入故障
    Inject,
    /// 请求还在延迟期内，暂不注入
    WaitForDelay,
    /// 规则已过期，不注入
    Expired,
}

/// 规则的时间相关信息
#[derive(Debug, Clone)]
pub struct RuleTiming {
    /// 请求延迟启动时间（毫秒）
    pub start_delay_ms: u32,
    /// 规则有效期（秒），0 表示无限期
    pub duration_seconds: u32,
    /// 规则创建时间戳（毫秒）
    pub creation_time_ms: u64,
}

/// 请求的时间信息
#[derive(Debug, Clone)]
pub struct RequestTiming {
    /// 请求到达时间戳（毫秒）
    pub arrival_time_ms: u64,
    /// 从请求到达至现在的耗时（毫秒）
    pub elapsed_since_arrival_ms: u64,
}

/// 获取当前时间戳（毫秒精度）
///
/// # Returns
///
/// 返回自 Unix epoch 起的毫秒数
pub fn get_current_time_ms() -> u64 {
    // 使用 proxy-wasm 的 hostcall 获取时间，而不是 std::time::SystemTime
    // 因为 std::time::SystemTime::now() 在 wasm32 平台上会 panic
    proxy_wasm::hostcalls::get_current_time()
        .ok()
        .and_then(|t| t.duration_since(UNIX_EPOCH).ok())
        .map(|d| d.as_millis() as u64)
        .unwrap_or(0)
}

/// 计算请求经过的时间（毫秒）
///
/// # Arguments
///
/// * `arrival_time_ms` - 请求到达的时间戳
///
/// # Returns
///
/// 从请求到达至现在的毫秒数
pub fn get_elapsed_time_ms(arrival_time_ms: u64) -> u64 {
    get_current_time_ms().saturating_sub(arrival_time_ms)
}

/// 判断是否应该注入故障
///
/// 检查规则的时间约束，确定是否应该执行故障注入。
///
/// # Logic
///
/// 1. 首先检查规则有效期 (duration_seconds)
///    - 如果 duration_seconds > 0，检查规则是否已过期
///    - 如果规则已过期，返回 Expired
///
/// 2. 然后检查请求延迟启动 (start_delay_ms)
///    - 如果 start_delay_ms > 0，检查请求是否经过足够时间
///    - 如果请求还在延迟期内，返回 WaitForDelay
///
/// 3. 如果所有条件都满足，返回 Inject
///
/// # Arguments
///
/// * `rule_timing` - 规则的时间信息
/// * `request_timing` - 请求的时间信息
///
/// # Returns
///
/// 时间控制决策
pub fn should_inject_fault(
    rule_timing: &RuleTiming,
    request_timing: &RequestTiming,
) -> TimeControlDecision {
    // 第一步：检查规则有效期
    if rule_timing.duration_seconds > 0 {
        let age_ms = get_current_time_ms().saturating_sub(rule_timing.creation_time_ms);
        let validity_window_ms = (rule_timing.duration_seconds as u64).saturating_mul(1000);

        if age_ms > validity_window_ms {
            return TimeControlDecision::Expired;
        }
    }

    // 第二步：检查请求延迟启动
    if rule_timing.start_delay_ms > 0 {
        let delay_ms = rule_timing.start_delay_ms as u64;
        if request_timing.elapsed_since_arrival_ms < delay_ms {
            return TimeControlDecision::WaitForDelay;
        }
    }

    // 第三步：所有条件都满足，可以注入
    TimeControlDecision::Inject
}

/// 检查时间约束的具体详情（用于调试和日志）
#[derive(Debug, Clone)]
pub struct TimeConstraintDetails {
    /// 规则年龄（毫秒）
    pub rule_age_ms: u64,
    /// 规则有效期窗口（毫秒）
    pub validity_window_ms: u64,
    /// 规则是否已过期
    pub is_expired: bool,
    /// 请求经过的时间（毫秒）
    pub elapsed_ms: u64,
    /// 所需延迟（毫秒）
    pub required_delay_ms: u64,
    /// 请求是否在延迟期内
    pub is_in_delay_period: bool,
}

/// 获取时间约束的详细信息
pub fn get_time_constraint_details(
    rule_timing: &RuleTiming,
    request_timing: &RequestTiming,
) -> TimeConstraintDetails {
    let current_time = get_current_time_ms();
    let rule_age_ms = current_time.saturating_sub(rule_timing.creation_time_ms);
    let validity_window_ms = (rule_timing.duration_seconds as u64).saturating_mul(1000);
    let is_expired = rule_timing.duration_seconds > 0 && rule_age_ms > validity_window_ms;

    let required_delay_ms = rule_timing.start_delay_ms as u64;
    let is_in_delay_period = request_timing.elapsed_since_arrival_ms < required_delay_ms;

    TimeConstraintDetails {
        rule_age_ms,
        validity_window_ms,
        is_expired,
        elapsed_ms: request_timing.elapsed_since_arrival_ms,
        required_delay_ms,
        is_in_delay_period,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_inject_immediate_no_expiry() {
        // 立即注入，无过期时间
        let rule_timing = RuleTiming {
            start_delay_ms: 0,
            duration_seconds: 0,
            creation_time_ms: 0,
        };
        let request_timing = RequestTiming {
            arrival_time_ms: 0,
            elapsed_since_arrival_ms: 0,
        };

        assert_eq!(
            should_inject_fault(&rule_timing, &request_timing),
            TimeControlDecision::Inject
        );
    }

    #[test]
    fn test_wait_for_delay() {
        // 请求在延迟期内
        let rule_timing = RuleTiming {
            start_delay_ms: 200,
            duration_seconds: 0,
            creation_time_ms: 0,
        };
        let request_timing = RequestTiming {
            arrival_time_ms: 0,
            elapsed_since_arrival_ms: 150, // 150ms < 200ms
        };

        assert_eq!(
            should_inject_fault(&rule_timing, &request_timing),
            TimeControlDecision::WaitForDelay
        );
    }

    #[test]
    fn test_inject_after_delay() {
        // 延迟期过后，应该注入
        let rule_timing = RuleTiming {
            start_delay_ms: 200,
            duration_seconds: 0,
            creation_time_ms: 0,
        };
        let request_timing = RequestTiming {
            arrival_time_ms: 0,
            elapsed_since_arrival_ms: 250, // 250ms > 200ms
        };

        assert_eq!(
            should_inject_fault(&rule_timing, &request_timing),
            TimeControlDecision::Inject
        );
    }

    #[test]
    fn test_rule_expired() {
        // 规则已过期
        let current_time = get_current_time_ms();
        let rule_timing = RuleTiming {
            start_delay_ms: 0,
            duration_seconds: 1,                                 // 1 秒有效期
            creation_time_ms: current_time.saturating_sub(2000), // 创建于 2 秒前
        };
        let request_timing = RequestTiming {
            arrival_time_ms: 0,
            elapsed_since_arrival_ms: 0,
        };

        assert_eq!(
            should_inject_fault(&rule_timing, &request_timing),
            TimeControlDecision::Expired
        );
    }

    #[test]
    fn test_rule_still_valid() {
        // 规则仍然有效
        let current_time = get_current_time_ms();
        let rule_timing = RuleTiming {
            start_delay_ms: 0,
            duration_seconds: 10,                                // 10 秒有效期
            creation_time_ms: current_time.saturating_sub(5000), // 创建于 5 秒前
        };
        let request_timing = RequestTiming {
            arrival_time_ms: 0,
            elapsed_since_arrival_ms: 0,
        };

        assert_eq!(
            should_inject_fault(&rule_timing, &request_timing),
            TimeControlDecision::Inject
        );
    }

    #[test]
    fn test_delay_and_valid_period() {
        // 延迟和有效期的组合
        let current_time = get_current_time_ms();
        let rule_timing = RuleTiming {
            start_delay_ms: 200,
            duration_seconds: 5,
            creation_time_ms: current_time.saturating_sub(2000), // 创建于 2 秒前
        };
        let request_timing = RequestTiming {
            arrival_time_ms: 0,
            elapsed_since_arrival_ms: 250, // 250ms > 200ms，在延迟期后
        };

        // 规则还有效，请求已过延迟期
        assert_eq!(
            should_inject_fault(&rule_timing, &request_timing),
            TimeControlDecision::Inject
        );
    }

    #[test]
    fn test_delay_and_rule_expired() {
        // 延迟期尚未过，但规则已过期
        let current_time = get_current_time_ms();
        let rule_timing = RuleTiming {
            start_delay_ms: 500,
            duration_seconds: 1,
            creation_time_ms: current_time.saturating_sub(2000), // 创建于 2 秒前，已过期
        };
        let request_timing = RequestTiming {
            arrival_time_ms: 0,
            elapsed_since_arrival_ms: 150, // 150ms < 500ms，仍在延迟期内
        };

        // 规则已过期，优先返回 Expired
        assert_eq!(
            should_inject_fault(&rule_timing, &request_timing),
            TimeControlDecision::Expired
        );
    }

    #[test]
    fn test_boundary_delay_equals_elapsed() {
        // 边界情况：延迟时间恰好等于经过时间
        let rule_timing = RuleTiming {
            start_delay_ms: 200,
            duration_seconds: 0,
            creation_time_ms: 0,
        };
        let request_timing = RequestTiming {
            arrival_time_ms: 0,
            elapsed_since_arrival_ms: 200, // 200ms == 200ms
        };

        // 当相等时，应该注入（延迟期已过）
        assert_eq!(
            should_inject_fault(&rule_timing, &request_timing),
            TimeControlDecision::Inject
        );
    }

    #[test]
    fn test_boundary_expiry_equals_age() {
        // 边界情况：规则年龄恰好等于有效期
        let current_time = get_current_time_ms();
        let rule_timing = RuleTiming {
            start_delay_ms: 0,
            duration_seconds: 1,
            creation_time_ms: current_time.saturating_sub(1000), // 创建于 1 秒前
        };
        let request_timing = RequestTiming {
            arrival_time_ms: 0,
            elapsed_since_arrival_ms: 0,
        };

        // 当年龄不超过有效期时，应该注入
        assert_eq!(
            should_inject_fault(&rule_timing, &request_timing),
            TimeControlDecision::Inject
        );
    }

    #[test]
    fn test_max_u32_values() {
        // 测试最大值
        let rule_timing = RuleTiming {
            start_delay_ms: u32::MAX,
            duration_seconds: u32::MAX,
            creation_time_ms: 0,
        };
        let request_timing = RequestTiming {
            arrival_time_ms: 0,
            elapsed_since_arrival_ms: 1000,
        };

        // 应该处理大数值而不崩溃
        let result = should_inject_fault(&rule_timing, &request_timing);
        // 由于 start_delay_ms 远大于 elapsed_since_arrival_ms，应该在延迟期内
        assert_eq!(result, TimeControlDecision::WaitForDelay);
    }

    #[test]
    fn test_get_time_constraint_details() {
        let current_time = get_current_time_ms();
        let rule_timing = RuleTiming {
            start_delay_ms: 200,
            duration_seconds: 5,
            creation_time_ms: current_time.saturating_sub(2000),
        };
        let request_timing = RequestTiming {
            arrival_time_ms: 0,
            elapsed_since_arrival_ms: 250,
        };

        let details = get_time_constraint_details(&rule_timing, &request_timing);

        assert!(!details.is_expired);
        assert!(!details.is_in_delay_period);
        assert_eq!(details.required_delay_ms, 200);
        assert_eq!(details.elapsed_ms, 250);
    }
}
