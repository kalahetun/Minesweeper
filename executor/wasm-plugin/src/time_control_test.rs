/// 时间控制集成测试
/// 
/// 测试 time_control 模块与整个故障注入系统的集成

use crate::time_control::{
    TimeControlDecision, RuleTiming, RequestTiming, 
    should_inject_fault, get_current_time_ms, get_elapsed_time_ms,
    get_time_constraint_details
};

#[test]
fn test_time_control_integration_immediate_fault() {
    // 场景：立即注入故障，无时间限制
    let rule_timing = RuleTiming {
        start_delay_ms: 0,
        duration_seconds: 0,
        creation_time_ms: get_current_time_ms(),
    };
    
    let request_timing = RequestTiming {
        arrival_time_ms: get_current_time_ms(),
        elapsed_since_arrival_ms: 0,
    };
    
    let decision = should_inject_fault(&rule_timing, &request_timing);
    assert_eq!(decision, TimeControlDecision::Inject);
}

#[test]
fn test_time_control_integration_delayed_fault() {
    // 场景：延迟 500ms 后注入故障
    let rule_timing = RuleTiming {
        start_delay_ms: 500,
        duration_seconds: 0,
        creation_time_ms: get_current_time_ms(),
    };
    
    // 模拟请求已到达 300ms
    let request_timing = RequestTiming {
        arrival_time_ms: get_current_time_ms(),
        elapsed_since_arrival_ms: 300,
    };
    
    let decision = should_inject_fault(&rule_timing, &request_timing);
    assert_eq!(decision, TimeControlDecision::WaitForDelay);
    
    // 模拟请求已到达 600ms
    let request_timing_later = RequestTiming {
        arrival_time_ms: get_current_time_ms(),
        elapsed_since_arrival_ms: 600,
    };
    
    let decision_later = should_inject_fault(&rule_timing, &request_timing_later);
    assert_eq!(decision_later, TimeControlDecision::Inject);
}

#[test]
fn test_time_control_integration_expiring_fault() {
    // 场景：规则在 2 秒内有效，之后过期
    let current_time = get_current_time_ms();
    let rule_timing = RuleTiming {
        start_delay_ms: 0,
        duration_seconds: 2,
        creation_time_ms: current_time,
    };
    
    let request_timing = RequestTiming {
        arrival_time_ms: current_time,
        elapsed_since_arrival_ms: 0,
    };
    
    // 检查规则创建后立即应该可以注入
    let mut decision = should_inject_fault(&rule_timing, &request_timing);
    assert_eq!(decision, TimeControlDecision::Inject);
    
    // 模拟规则已创建 3 秒（超过 2 秒有效期）
    let old_rule_timing = RuleTiming {
        start_delay_ms: 0,
        duration_seconds: 2,
        creation_time_ms: current_time.saturating_sub(3000),
    };
    
    decision = should_inject_fault(&old_rule_timing, &request_timing);
    assert_eq!(decision, TimeControlDecision::Expired);
}

#[test]
fn test_time_control_integration_combined_constraints() {
    // 场景：组合约束 - 延迟 300ms 且规则在 5 秒内有效
    let current_time = get_current_time_ms();
    let rule_timing = RuleTiming {
        start_delay_ms: 300,
        duration_seconds: 5,
        creation_time_ms: current_time.saturating_sub(1000),  // 1秒前创建
    };
    
    let request_timing = RequestTiming {
        arrival_time_ms: current_time,
        elapsed_since_arrival_ms: 200,  // 200ms < 300ms
    };
    
    // 还在延迟期内
    let decision = should_inject_fault(&rule_timing, &request_timing);
    assert_eq!(decision, TimeControlDecision::WaitForDelay);
    
    // 经过 400ms
    let request_timing_later = RequestTiming {
        arrival_time_ms: current_time,
        elapsed_since_arrival_ms: 400,
    };
    
    // 可以注入
    let decision_later = should_inject_fault(&rule_timing, &request_timing_later);
    assert_eq!(decision_later, TimeControlDecision::Inject);
}

#[test]
fn test_time_control_integration_priority_expiry_over_delay() {
    // 场景：验证过期检查优先于延迟检查
    let current_time = get_current_time_ms();
    let rule_timing = RuleTiming {
        start_delay_ms: 5000,  // 需要 5 秒延迟
        duration_seconds: 1,   // 但只有 1 秒有效期
        creation_time_ms: current_time.saturating_sub(2000),  // 2秒前创建（已过期）
    };
    
    let request_timing = RequestTiming {
        arrival_time_ms: current_time,
        elapsed_since_arrival_ms: 100,  // 请求只经过 100ms
    };
    
    // 即使请求还在延迟期内，规则已过期，应该返回 Expired
    let decision = should_inject_fault(&rule_timing, &request_timing);
    assert_eq!(decision, TimeControlDecision::Expired);
}

#[test]
fn test_time_control_constraint_details() {
    // 场景：验证详细信息的准确性
    let current_time = get_current_time_ms();
    let rule_timing = RuleTiming {
        start_delay_ms: 200,
        duration_seconds: 10,
        creation_time_ms: current_time.saturating_sub(5000),
    };
    
    let request_timing = RequestTiming {
        arrival_time_ms: current_time,
        elapsed_since_arrival_ms: 300,
    };
    
    let details = get_time_constraint_details(&rule_timing, &request_timing);
    
    // 验证计算值
    assert!(!details.is_expired);
    assert!(!details.is_in_delay_period);
    assert_eq!(details.required_delay_ms, 200);
    assert_eq!(details.elapsed_ms, 300);
    assert_eq!(details.validity_window_ms, 10000);
    // rule_age_ms 应该约为 5000ms（考虑到可能的时间偏差）
    assert!(details.rule_age_ms >= 4900 && details.rule_age_ms <= 5100);
}

#[test]
fn test_time_control_persistent_rule() {
    // 场景：持久化规则 (duration_seconds = 0)
    let current_time = get_current_time_ms();
    
    // 模拟规则很早之前创建（持久化规则）
    let rule_timing = RuleTiming {
        start_delay_ms: 0,
        duration_seconds: 0,  // 无过期时间
        creation_time_ms: current_time.saturating_sub(86400000),  // 1天前创建
    };
    
    let request_timing = RequestTiming {
        arrival_time_ms: current_time,
        elapsed_since_arrival_ms: 0,
    };
    
    // 即使规则创建很久前，也应该可以注入（因为持久化）
    let decision = should_inject_fault(&rule_timing, &request_timing);
    assert_eq!(decision, TimeControlDecision::Inject);
}

#[test]
fn test_time_control_zero_delay_edge_case() {
    // 场景：边界情况 - 延迟为 0 的规则
    let rule_timing = RuleTiming {
        start_delay_ms: 0,
        duration_seconds: 5,
        creation_time_ms: get_current_time_ms(),
    };
    
    let request_timing = RequestTiming {
        arrival_time_ms: get_current_time_ms(),
        elapsed_since_arrival_ms: 0,
    };
    
    // 应该立即注入
    let decision = should_inject_fault(&rule_timing, &request_timing);
    assert_eq!(decision, TimeControlDecision::Inject);
}

#[test]
fn test_time_control_large_delay_value() {
    // 场景：很大的延迟值 (接近 u32::MAX)
    let rule_timing = RuleTiming {
        start_delay_ms: 1_000_000,  // 1000 秒 = ~16 分钟
        duration_seconds: 0,
        creation_time_ms: 0,
    };
    
    let request_timing = RequestTiming {
        arrival_time_ms: 0,
        elapsed_since_arrival_ms: 500_000,  // 只经过 500 秒
    };
    
    // 仍在延迟期内
    let decision = should_inject_fault(&rule_timing, &request_timing);
    assert_eq!(decision, TimeControlDecision::WaitForDelay);
}

#[test]
fn test_time_control_large_duration_value() {
    // 场景：很大的有效期值 (接近 u32::MAX)
    let current_time = get_current_time_ms();
    let rule_timing = RuleTiming {
        start_delay_ms: 0,
        duration_seconds: 1_000_000,  // ~1000000 秒 = ~11.5 天
        creation_time_ms: current_time.saturating_sub(1000),
    };
    
    let request_timing = RequestTiming {
        arrival_time_ms: current_time,
        elapsed_since_arrival_ms: 0,
    };
    
    // 应该仍然有效
    let decision = should_inject_fault(&rule_timing, &request_timing);
    assert_eq!(decision, TimeControlDecision::Inject);
}

#[test]
fn test_time_control_saturation_arithmetic() {
    // 场景：确保时间计算中的饱和运算能正确处理
    let rule_timing = RuleTiming {
        start_delay_ms: 100,
        duration_seconds: 10,
        creation_time_ms: 100,  // 很小的时间戳
    };
    
    let request_timing = RequestTiming {
        arrival_time_ms: 50,  // 比规则创建时间还早（模拟时间不同步）
        elapsed_since_arrival_ms: 200,
    };
    
    // 应该能正确处理，不应该崩溃或溢出
    let decision = should_inject_fault(&rule_timing, &request_timing);
    // 可以注入或等待延迟（具体取决于算术实现）
    assert!(matches!(decision, 
        TimeControlDecision::Inject | TimeControlDecision::WaitForDelay | TimeControlDecision::Expired
    ));
}

#[test]
fn test_time_control_rapid_decisions() {
    // 场景：连续快速做出多个决策
    let current_time = get_current_time_ms();
    let rule_timing = RuleTiming {
        start_delay_ms: 1000,
        duration_seconds: 30,
        creation_time_ms: current_time,
    };
    
    // 模拟请求在不同时间的多个检查
    let request_arrival = current_time;
    
    for i in 0..10 {
        let elapsed = i * 150;  // 每次增加 150ms
        let request_timing = RequestTiming {
            arrival_time_ms: request_arrival,
            elapsed_since_arrival_ms: elapsed,
        };
        
        let decision = should_inject_fault(&rule_timing, &request_timing);
        
        if elapsed < 1000 {
            assert_eq!(decision, TimeControlDecision::WaitForDelay);
        } else {
            assert_eq!(decision, TimeControlDecision::Inject);
        }
    }
}

#[test]
fn test_time_control_multiple_rules_different_timings() {
    // 场景：多个规则以不同的时间约束工作
    let current_time = get_current_time_ms();
    let request_timing = RequestTiming {
        arrival_time_ms: current_time,
        elapsed_since_arrival_ms: 500,
    };
    
    // 规则 1：立即注入
    let rule1 = RuleTiming {
        start_delay_ms: 0,
        duration_seconds: 0,
        creation_time_ms: current_time,
    };
    assert_eq!(should_inject_fault(&rule1, &request_timing), TimeControlDecision::Inject);
    
    // 规则 2：延迟 1000ms
    let rule2 = RuleTiming {
        start_delay_ms: 1000,
        duration_seconds: 0,
        creation_time_ms: current_time,
    };
    assert_eq!(should_inject_fault(&rule2, &request_timing), TimeControlDecision::WaitForDelay);
    
    // 规则 3：2 秒有效期
    let rule3 = RuleTiming {
        start_delay_ms: 0,
        duration_seconds: 2,
        creation_time_ms: current_time.saturating_sub(3000),
    };
    assert_eq!(should_inject_fault(&rule3, &request_timing), TimeControlDecision::Expired);
}

#[test]
fn test_time_control_elapsed_time_calculation() {
    // 场景：验证经过时间的计算
    let start_time = get_current_time_ms();
    let elapsed1 = get_elapsed_time_ms(start_time);
    
    // 应该返回接近 0（或很小的值）
    assert!(elapsed1 < 100);  // 容许最多 100ms 的时间偏差
    
    // 模拟 100ms 前的时间
    let past_time = start_time.saturating_sub(100);
    let elapsed2 = get_elapsed_time_ms(past_time);
    
    // 应该返回大约 100ms
    assert!(elapsed2 >= 90 && elapsed2 <= 150);
}
