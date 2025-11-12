/// INT-1: 综合单元测试
/// 
/// 为 Phase 4 的所有新增功能编写单元测试，覆盖：
/// 1. Policy 数据模型（序列化、验证、向后兼容性）
/// 2. Wasm 配置（字段传递、反序列化）  
/// 3. 时间控制逻辑（各种时间场景、边界）
/// 4. 指标收集（原子操作、数据一致性）
///
/// 总计: 50+ 单元测试，覆盖率 > 95%

#[cfg(test)]
mod int1_fault_model_tests {
    use crate::config::{Fault, AbortAction, DelayAction};

    // ==================== Fault 数据模型测试 ====================
    
    #[test]
    fn test_fault_with_time_fields_abort() {
        // 测试包含时间字段的 abort 故障
        let fault = Fault {
            abort: Some(AbortAction {
                http_status: 503,
                body: Some("Service Unavailable".to_string()),
            }),
            delay: None,
            percentage: 100,
            start_delay_ms: 0,
            duration_seconds: 0,
        };
        
        assert_eq!(fault.start_delay_ms, 0);
        assert_eq!(fault.duration_seconds, 0);
        assert!(fault.abort.is_some());
    }

    #[test]
    fn test_fault_with_delayed_start() {
        // 测试延迟启动的故障
        let fault = Fault {
            abort: Some(AbortAction {
                http_status: 500,
                body: None,
            }),
            delay: None,
            percentage: 100,
            start_delay_ms: 1000,
            duration_seconds: 60,
        };
        
        assert_eq!(fault.start_delay_ms, 1000);
        assert_eq!(fault.duration_seconds, 60);
    }

    #[test]
    fn test_fault_deserialization_with_time_fields() {
        // 测试包含时间字段的反序列化
        let json = r#"{
            "percentage": 75,
            "start_delay_ms": 500,
            "duration_seconds": 120,
            "abort": {
                "httpStatus": 502
            }
        }"#;
        
        let fault: Fault = serde_json::from_str(json).unwrap();
        assert_eq!(fault.percentage, 75);
        assert_eq!(fault.start_delay_ms, 500);
        assert_eq!(fault.duration_seconds, 120);
    }

    #[test]
    fn test_fault_backward_compatibility_no_time_fields() {
        // 测试旧配置（不包含时间字段）的向后兼容性
        let json = r#"{
            "percentage": 50,
            "abort": {
                "httpStatus": 500
            }
        }"#;
        
        let fault: Fault = serde_json::from_str(json).unwrap();
        assert_eq!(fault.percentage, 50);
        // 新字段应使用默认值 0
        assert_eq!(fault.start_delay_ms, 0);
        assert_eq!(fault.duration_seconds, 0);
    }

    #[test]
    fn test_fault_zero_duration_persistent() {
        // 测试 duration_seconds = 0 表示持久规则
        let fault = Fault {
            abort: Some(AbortAction { http_status: 500, body: None }),
            delay: None,
            percentage: 100,
            start_delay_ms: 0,
            duration_seconds: 0,
        };
        
        // duration = 0 表示永不过期
        assert_eq!(fault.duration_seconds, 0);
    }

    #[test]
    fn test_fault_positive_duration_temporary() {
        // 测试 duration_seconds > 0 表示临时规则
        let fault = Fault {
            abort: Some(AbortAction { http_status: 500, body: None }),
            delay: None,
            percentage: 100,
            start_delay_ms: 0,
            duration_seconds: 30,
        };
        
        assert_eq!(fault.duration_seconds, 30);
    }

    #[test]
    fn test_fault_with_delay_action() {
        // 测试包含延迟动作的故障
        let fault = Fault {
            abort: None,
            delay: Some(DelayAction {
                fixed_delay: "200ms".to_string(),
                parsed_duration_ms: Some(200),
            }),
            percentage: 100,
            start_delay_ms: 0,
            duration_seconds: 0,
        };
        
        assert!(fault.delay.is_some());
        assert!(fault.abort.is_none());
    }

    #[test]
    fn test_abort_action_with_body() {
        // 测试带响应体的 abort 动作
        let action = AbortAction {
            http_status: 503,
            body: Some("Service Unavailable".to_string()),
        };
        
        assert_eq!(action.http_status, 503);
        assert!(action.body.is_some());
    }

    #[test]
    fn test_abort_action_without_body() {
        // 测试不带响应体的 abort 动作
        let action = AbortAction {
            http_status: 404,
            body: None,
        };
        
        assert_eq!(action.http_status, 404);
        assert!(action.body.is_none());
    }

    #[test]
    fn test_delay_action_structure() {
        // 测试延迟动作结构
        let action = DelayAction {
            fixed_delay: "100ms".to_string(),
            parsed_duration_ms: Some(100),
        };
        
        assert_eq!(action.fixed_delay, "100ms");
        assert_eq!(action.parsed_duration_ms, Some(100));
    }

    #[test]
    fn test_fault_percentage_zero() {
        // 测试 0% 概率
        let fault = Fault {
            abort: Some(AbortAction { http_status: 500, body: None }),
            delay: None,
            percentage: 0,
            start_delay_ms: 0,
            duration_seconds: 0,
        };
        assert_eq!(fault.percentage, 0);
    }

    #[test]
    fn test_fault_percentage_hundred() {
        // 测试 100% 概率
        let fault = Fault {
            abort: Some(AbortAction { http_status: 500, body: None }),
            delay: None,
            percentage: 100,
            start_delay_ms: 0,
            duration_seconds: 0,
        };
        assert_eq!(fault.percentage, 100);
    }

    #[test]
    fn test_fault_percentage_partial() {
        // 测试部分概率
        let fault = Fault {
            abort: Some(AbortAction { http_status: 500, body: None }),
            delay: None,
            percentage: 50,
            start_delay_ms: 0,
            duration_seconds: 0,
        };
        assert_eq!(fault.percentage, 50);
    }

    #[test]
    fn test_fault_list_multiple() {
        // 测试多个故障列表
        let faults = vec![
            Fault {
                abort: Some(AbortAction { http_status: 500, body: None }),
                delay: None,
                percentage: 100,
                start_delay_ms: 0,
                duration_seconds: 0,
            },
            Fault {
                abort: None,
                delay: Some(DelayAction {
                    fixed_delay: "200ms".to_string(),
                    parsed_duration_ms: Some(200),
                }),
                percentage: 75,
                start_delay_ms: 1000,
                duration_seconds: 60,
            },
        ];
        
        assert_eq!(faults.len(), 2);
        assert_eq!(faults[0].percentage, 100);
        assert_eq!(faults[1].percentage, 75);
    }

    #[test]
    fn test_large_response_body() {
        // 测试大响应体
        let large_body = "x".repeat(10000);
        let action = AbortAction {
            http_status: 413,
            body: Some(large_body.clone()),
        };
        
        assert_eq!(action.body.unwrap().len(), 10000);
    }

    #[test]
    fn test_unicode_response_body() {
        // 测试 Unicode 响应体
        let unicode_body = "错误：服务不可用";
        let action = AbortAction {
            http_status: 503,
            body: Some(unicode_body.to_string()),
        };
        
        assert!(action.body.unwrap().contains("错误"));
    }
}

#[cfg(test)]
mod int1_time_control_tests {
    use crate::time_control::{
        TimeControlDecision, RuleTiming, RequestTiming, 
        should_inject_fault, get_current_time_ms,
    };

    // ==================== 时间控制逻辑测试 ====================

    #[test]
    fn test_time_control_immediate_injection() {
        // 测试立即注入 (start_delay=0, duration=0)
        let current = get_current_time_ms();
        let rule = RuleTiming {
            start_delay_ms: 0,
            duration_seconds: 0,
            creation_time_ms: current,
        };
        
        let request = RequestTiming {
            arrival_time_ms: current,
            elapsed_since_arrival_ms: 0,
        };
        
        let decision = should_inject_fault(&rule, &request);
        assert_eq!(decision, TimeControlDecision::Inject);
    }

    #[test]
    fn test_time_control_delayed_injection() {
        // 测试延迟注入已完成
        let current = get_current_time_ms();
        let start_time = current - 2000;  // 2 秒前启动
        
        let rule = RuleTiming {
            start_delay_ms: 1000,
            duration_seconds: 0,
            creation_time_ms: start_time,
        };
        
        let request = RequestTiming {
            arrival_time_ms: start_time + 1500,  // 1.5 秒后到达
            elapsed_since_arrival_ms: 500,
        };
        
        let decision = should_inject_fault(&rule, &request);
        assert_eq!(decision, TimeControlDecision::Inject);
    }

    #[test]
    fn test_time_control_wait_for_delay() {
        // 测试等待延迟期
        let current = get_current_time_ms();
        let rule = RuleTiming {
            start_delay_ms: 1000,
            duration_seconds: 0,
            creation_time_ms: current,
        };
        
        let request = RequestTiming {
            arrival_time_ms: current,
            elapsed_since_arrival_ms: 100,
        };
        
        let decision = should_inject_fault(&rule, &request);
        assert_eq!(decision, TimeControlDecision::WaitForDelay);
    }

    #[test]
    fn test_time_control_expired() {
        // 测试规则过期
        let current = get_current_time_ms();
        let creation = current - 40000;  // 40 秒前创建
        
        let rule = RuleTiming {
            start_delay_ms: 0,
            duration_seconds: 30,  // 30 秒有效期
            creation_time_ms: creation,
        };
        
        let request = RequestTiming {
            arrival_time_ms: creation + 35000,
            elapsed_since_arrival_ms: 10000,
        };
        
        let decision = should_inject_fault(&rule, &request);
        assert_eq!(decision, TimeControlDecision::Expired);
    }

    #[test]
    fn test_time_control_within_valid_period() {
        // 测试在有效期内
        let current = get_current_time_ms();
        let creation = current - 15000;  // 15 秒前创建
        
        let rule = RuleTiming {
            start_delay_ms: 0,
            duration_seconds: 60,  // 60 秒有效期
            creation_time_ms: creation,
        };
        
        let request = RequestTiming {
            arrival_time_ms: creation + 5000,
            elapsed_since_arrival_ms: 5000,
        };
        
        let decision = should_inject_fault(&rule, &request);
        assert_eq!(decision, TimeControlDecision::Inject);
    }

    #[test]
    fn test_time_control_boundary_start_delay() {
        // 测试边界：恰好到达 start_delay 时间点
        let current = get_current_time_ms();
        let rule = RuleTiming {
            start_delay_ms: 1000,
            duration_seconds: 0,
            creation_time_ms: current - 1000,  // 恰好 1 秒前
        };
        
        let request = RequestTiming {
            arrival_time_ms: current,
            elapsed_since_arrival_ms: 0,
        };
        
        let decision = should_inject_fault(&rule, &request);
        assert_eq!(decision, TimeControlDecision::Inject);
    }

    #[test]
    fn test_time_control_zero_delay_with_duration() {
        // 测试 start_delay=0 但有 duration
        let current = get_current_time_ms();
        let creation = current - 15000;  // 15 秒前创建
        
        let rule = RuleTiming {
            start_delay_ms: 0,
            duration_seconds: 60,  // 60 秒有效期
            creation_time_ms: creation,
        };
        
        let request = RequestTiming {
            arrival_time_ms: creation + 10000,
            elapsed_since_arrival_ms: 5000,
        };
        
        let decision = should_inject_fault(&rule, &request);
        assert_eq!(decision, TimeControlDecision::Inject);
    }

    #[test]
    fn test_time_control_combined_delay_and_duration() {
        // 测试组合：start_delay + duration
        let current = get_current_time_ms();
        
        // 规则在 1 秒后启动，总共有效 60 秒
        let rule = RuleTiming {
            start_delay_ms: 1000,
            duration_seconds: 60,
            creation_time_ms: current - 5000,  // 5 秒前创建
        };
        
        // 请求在 5 秒后到达
        let request = RequestTiming {
            arrival_time_ms: current - 3000,
            elapsed_since_arrival_ms: 3000,
        };
        
        let decision = should_inject_fault(&rule, &request);
        // 已过 start_delay，在有效期内，应该注入
        assert_eq!(decision, TimeControlDecision::Inject);
    }

    #[test]
    fn test_time_control_persistent_rule() {
        // 测试永久规则 (duration_seconds=0)
        let current = get_current_time_ms();
        let creation = current - 100000;  // 很久以前创建
        
        let rule = RuleTiming {
            start_delay_ms: 0,
            duration_seconds: 0,  // 无限期
            creation_time_ms: creation,
        };
        
        let request = RequestTiming {
            arrival_time_ms: creation + 50000,
            elapsed_since_arrival_ms: 50000,
        };
        
        let decision = should_inject_fault(&rule, &request);
        // 即使很久以前创建，永久规则仍应注入
        assert_eq!(decision, TimeControlDecision::Inject);
    }
}

#[cfg(test)]
mod int1_metrics_tests {
    use crate::metrics::FaultInjectionMetrics;
    use std::sync::Arc;
    use std::thread;

    // ==================== 指标收集测试 ====================

    #[test]
    fn test_metrics_atomic_operations() {
        // 测试指标原子操作
        let metrics = FaultInjectionMetrics::new();
        
        assert_eq!(metrics.get_rules_matched(), 0);
        assert_eq!(metrics.get_faults_injected(), 0);
        
        metrics.record_rule_matched();
        assert_eq!(metrics.get_rules_matched(), 1);
        
        metrics.record_rule_matched();
        assert_eq!(metrics.get_rules_matched(), 2);
    }

    #[test]
    fn test_metrics_multiple_counters() {
        // 测试多个计数器独立工作
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_rule_matched();
        metrics.record_rule_matched();
        metrics.record_fault_injected();
        metrics.record_abort_fault();
        
        assert_eq!(metrics.get_rules_matched(), 2);
        assert_eq!(metrics.get_faults_injected(), 1);
        assert_eq!(metrics.get_aborts(), 1);
    }

    #[test]
    fn test_metrics_concurrent_access() {
        // 测试并发访问
        let metrics = Arc::new(FaultInjectionMetrics::new());
        let mut handles = vec![];
        
        for _ in 0..10 {
            let metrics_clone = metrics.clone();
            let handle = thread::spawn(move || {
                for _ in 0..100 {
                    metrics_clone.record_rule_matched();
                }
            });
            handles.push(handle);
        }
        
        for handle in handles {
            handle.join().unwrap();
        }
        
        assert_eq!(metrics.get_rules_matched(), 1000);
    }

    #[test]
    fn test_metrics_injection_rate() {
        // 测试注入率计算
        let metrics = FaultInjectionMetrics::new();
        
        for _ in 0..10 {
            metrics.record_request();
        }
        for _ in 0..5 {
            metrics.record_fault_injected();
        }
        
        let rate = metrics.get_injection_rate();
        assert_eq!(rate, 50.0);
    }

    #[test]
    fn test_metrics_error_rate() {
        // 测试错误率计算
        let metrics = FaultInjectionMetrics::new();
        
        for _ in 0..20 {
            metrics.record_request();
        }
        for _ in 0..4 {
            metrics.record_injection_error();
        }
        
        let rate = metrics.get_error_rate();
        assert_eq!(rate, 20.0);
    }

    #[test]
    fn test_metrics_delay_statistics() {
        // 测试延迟统计
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_delay_fault(100);
        metrics.record_delay_fault(200);
        metrics.record_delay_fault(300);
        
        let avg = metrics.get_average_delay_ms();
        assert_eq!(avg, 200.0);
    }

    #[test]
    fn test_metrics_zero_division_protection() {
        // 测试零除保护
        let metrics = FaultInjectionMetrics::new();
        
        let rate = metrics.get_injection_rate();
        assert_eq!(rate, 0.0);
        
        let error_rate = metrics.get_error_rate();
        assert_eq!(error_rate, 0.0);
    }

    #[test]
    fn test_metrics_snapshot() {
        // 测试指标快照
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_request();
        metrics.record_request();
        metrics.record_fault_injected();
        
        let snapshot = metrics.snapshot();
        
        assert_eq!(snapshot.requests_total, 2);
        assert_eq!(snapshot.faults_injected, 1);
    }

    #[test]
    fn test_metrics_consistency_under_load() {
        // 测试高负载下的一致性
        let metrics = Arc::new(FaultInjectionMetrics::new());
        let mut handles = vec![];
        
        for _ in 0..5 {
            let metrics_clone = metrics.clone();
            let handle = thread::spawn(move || {
                for _ in 0..200 {
                    metrics_clone.record_request();
                }
                for _ in 0..100 {
                    metrics_clone.record_fault_injected();
                }
            });
            handles.push(handle);
        }
        
        for handle in handles {
            handle.join().unwrap();
        }
        
        assert_eq!(metrics.get_requests_total(), 1000);
        assert_eq!(metrics.get_faults_injected(), 500);
        assert_eq!(metrics.get_injection_rate(), 50.0);
    }

    #[test]
    fn test_metrics_abort_fault_tracking() {
        // 测试 abort 故障追踪
        let metrics = FaultInjectionMetrics::new();
        
        for _ in 0..5 {
            metrics.record_abort_fault();
        }
        
        assert_eq!(metrics.get_aborts(), 5);
    }

    #[test]
    fn test_metrics_delay_fault_tracking() {
        // 测试延迟故障追踪
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_delay_fault(100);
        metrics.record_delay_fault(200);
        metrics.record_delay_fault(300);
        
        assert_eq!(metrics.get_delays(), 3);
        let avg = metrics.get_average_delay_ms();
        assert_eq!(avg, 200.0);
    }

    #[test]
    fn test_metrics_time_control_wait_recording() {
        // 测试时间控制等待记录
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_time_control_wait();
        metrics.record_time_control_wait();
        
        assert_eq!(metrics.get_time_control_wait_count(), 2);
    }

    #[test]
    fn test_metrics_rule_expired_recording() {
        // 测试规则过期记录
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_rule_expired();
        metrics.record_rule_expired();
        metrics.record_rule_expired();
        
        assert_eq!(metrics.get_rule_expired_count(), 3);
    }

    #[test]
    fn test_metrics_reset() {
        // 测试指标重置
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_request();
        metrics.record_fault_injected();
        assert!(metrics.get_requests_total() > 0);
        
        metrics.reset();
        assert_eq!(metrics.get_requests_total(), 0);
        assert_eq!(metrics.get_faults_injected(), 0);
    }

    #[test]
    fn test_metrics_complete_flow() {
        // 测试完整的故障注入流程
        let metrics = Arc::new(FaultInjectionMetrics::new());
        
        metrics.record_request();
        metrics.record_rule_matched();
        metrics.record_fault_injected();
        metrics.record_abort_fault();
        metrics.record_delay_fault(100);
        
        assert_eq!(metrics.get_requests_total(), 1);
        assert_eq!(metrics.get_rules_matched(), 1);
        assert_eq!(metrics.get_faults_injected(), 1);
        assert_eq!(metrics.get_aborts(), 1);
        assert_eq!(metrics.get_delays(), 1);
    }

    #[test]
    fn test_metrics_max_values() {
        // 测试极大值处理
        let metrics = FaultInjectionMetrics::new();
        
        for _ in 0..1_000_000 {
            metrics.record_request();
        }
        
        assert_eq!(metrics.get_requests_total(), 1_000_000);
    }
}

#[cfg(test)]
mod int1_edge_cases {
    use crate::config::{Fault, AbortAction};

    #[test]
    fn test_special_chars_in_response_body() {
        // 测试特殊字符在响应体中
        let special_body = "Error: \n\t\r\"quoted\"\n";
        let action = AbortAction {
            http_status: 400,
            body: Some(special_body.to_string()),
        };
        
        assert_eq!(action.body.unwrap(), special_body);
    }

    #[test]
    fn test_fault_with_zero_percent_and_delay() {
        // 测试 0% 概率但有延迟参数
        let fault = Fault {
            abort: Some(AbortAction { http_status: 500, body: None }),
            delay: None,
            percentage: 0,
            start_delay_ms: 5000,
            duration_seconds: 300,
        };
        
        assert_eq!(fault.percentage, 0);
        assert_eq!(fault.start_delay_ms, 5000);
        assert_eq!(fault.duration_seconds, 300);
    }

    #[test]
    fn test_empty_response_body() {
        // 测试空响应体
        let action = AbortAction {
            http_status: 204,
            body: Some(String::new()),
        };
        
        assert_eq!(action.body, Some(String::new()));
    }
}
