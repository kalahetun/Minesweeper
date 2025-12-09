use std::collections::HashMap;
/// Wasm Plugin 请求隔离集成测试
///
/// 验证:
/// 1. 并发请求处理 - 多个请求可以同时处理
/// 2. 无状态污染 - 一个请求的故障不影响其他请求
/// 3. 规则应用一致性 - 同一规则对不同请求的应用是一致的
use std::sync::{Arc, Barrier, Mutex};

/// 模拟请求上下文
#[derive(Clone)]
pub struct RequestContext {
    request_id: u64,
    path: String,
    method: String,
    headers: Arc<Mutex<HashMap<String, String>>>,
    state: Arc<Mutex<Option<String>>>,
}

impl RequestContext {
    fn new(request_id: u64, path: &str, method: &str) -> Self {
        RequestContext {
            request_id,
            path: path.to_string(),
            method: method.to_string(),
            headers: Arc::new(Mutex::new(HashMap::new())),
            state: Arc::new(Mutex::new(None)),
        }
    }

    fn add_header(&self, name: &str, value: &str) {
        self.headers
            .lock()
            .unwrap()
            .insert(name.to_string(), value.to_string());
    }

    fn get_header(&self, name: &str) -> Option<String> {
        self.headers.lock().unwrap().get(name).cloned()
    }

    fn set_state(&self, state: String) {
        *self.state.lock().unwrap() = Some(state);
    }

    fn get_state(&self) -> Option<String> {
        self.state.lock().unwrap().clone()
    }

    fn clear_state(&self) {
        *self.state.lock().unwrap() = None;
    }
}

/// 模拟规则应用器
pub struct RuleApplier {
    rule_name: String,
    fault_type: String,
    condition: Box<dyn Fn(&RequestContext) -> bool + Send + Sync>,
}

impl RuleApplier {
    fn new(
        name: &str,
        fault_type: &str,
        condition: Box<dyn Fn(&RequestContext) -> bool + Send + Sync>,
    ) -> Self {
        RuleApplier {
            rule_name: name.to_string(),
            fault_type: fault_type.to_string(),
            condition,
        }
    }

    fn apply(&self, ctx: &RequestContext) -> bool {
        (self.condition)(ctx)
    }

    fn apply_fault(&self, ctx: &RequestContext) {
        if self.apply(ctx) {
            ctx.set_state(format!("{}:applied", self.rule_name));
        }
    }
}

#[cfg(test)]
mod stateful_tests {
    use super::*;

    /// 测试请求状态隔离
    ///
    /// 验证两个并发请求的状态完全独立
    #[test]
    fn test_request_isolation() {
        let req1 = RequestContext::new(1, "/api/users", "GET");
        let req2 = RequestContext::new(2, "/api/products", "POST");

        // 为两个请求设置不同的状态
        req1.set_state("fault:503".to_string());
        req2.set_state("fault:delay".to_string());

        // 验证状态完全隔离
        assert_eq!(
            req1.get_state(),
            Some("fault:503".to_string()),
            "请求1的状态应该独立"
        );
        assert_eq!(
            req2.get_state(),
            Some("fault:delay".to_string()),
            "请求2的状态应该独立"
        );
    }

    /// 测试并发请求处理
    ///
    /// 多个请求在同一时刻处理，验证没有竞态条件
    #[test]
    fn test_concurrent_request_handling() {
        let barrier = Arc::new(Barrier::new(3));
        let mut handles = vec![];

        for i in 0..3 {
            let barrier = Arc::clone(&barrier);
            let handle = std::thread::spawn(move || {
                let ctx = RequestContext::new(i as u64, &format!("/api/path/{}", i), "GET");

                // 同步开始
                barrier.wait();

                // 设置请求特定的状态
                ctx.set_state(format!("request-{}-processed", i));

                // 验证状态正确性
                assert_eq!(
                    ctx.get_state(),
                    Some(format!("request-{}-processed", i)),
                    "请求 {} 状态应该独立",
                    i
                );

                // 验证路径
                assert_eq!(
                    ctx.path,
                    format!("/api/path/{}", i),
                    "请求 {} 路径应该正确",
                    i
                );
            });
            handles.push(handle);
        }

        for handle in handles {
            handle.join().unwrap();
        }
    }

    /// 测试规则一致性应用
    ///
    /// 同一规则对不同请求的应用结果应该一致
    #[test]
    fn test_rule_consistency() {
        let rule = RuleApplier::new(
            "abort-503",
            "abort",
            Box::new(|ctx| ctx.path.starts_with("/api/users")),
        );

        let req1 = RequestContext::new(1, "/api/users/1", "GET");
        let req2 = RequestContext::new(2, "/api/users/2", "GET");
        let req3 = RequestContext::new(3, "/api/products", "GET");

        // 应用规则到三个请求
        rule.apply_fault(&req1);
        rule.apply_fault(&req2);
        rule.apply_fault(&req3);

        // 验证前两个请求匹配规则
        assert_eq!(
            req1.get_state(),
            Some("abort-503:applied".to_string()),
            "匹配路径的请求1应该应用规则"
        );
        assert_eq!(
            req2.get_state(),
            Some("abort-503:applied".to_string()),
            "匹配路径的请求2应该应用规则"
        );

        // 验证第三个请求不匹配规则
        assert!(req3.get_state().is_none(), "不匹配的请求3不应该应用规则");
    }

    /// 测试多规则处理
    ///
    /// 多个规则对同一请求的处理 - 第一个匹配的规则应该执行
    #[test]
    fn test_multiple_rules_ordering() {
        let rule1 = RuleApplier::new(
            "rule1-abort",
            "abort",
            Box::new(|ctx| ctx.path.starts_with("/api/users")),
        );

        let rule2 = RuleApplier::new("rule2-delay", "delay", Box::new(|ctx| ctx.method == "GET"));

        let ctx = RequestContext::new(1, "/api/users/1", "GET");

        // 应用第一个规则
        rule1.apply_fault(&ctx);
        assert_eq!(
            ctx.get_state(),
            Some("rule1-abort:applied".to_string()),
            "第一个匹配的规则应该执行"
        );

        // 重置状态，应用第二个规则
        ctx.clear_state();
        rule2.apply_fault(&ctx);
        assert_eq!(
            ctx.get_state(),
            Some("rule2-delay:applied".to_string()),
            "第二个规则也能执行"
        );
    }

    /// 测试头部隔离
    ///
    /// 请求的头部应该不会相互影响
    #[test]
    fn test_header_isolation() {
        let req1 = RequestContext::new(1, "/api/v1", "GET");
        let req2 = RequestContext::new(2, "/api/v2", "GET");

        // 为两个请求添加不同的头部
        req1.add_header("X-Request-ID", "req-1");
        req2.add_header("X-Request-ID", "req-2");

        // 验证头部隔离
        assert_eq!(
            req1.get_header("X-Request-ID"),
            Some("req-1".to_string()),
            "请求1的头部应该独立"
        );
        assert_eq!(
            req2.get_header("X-Request-ID"),
            Some("req-2".to_string()),
            "请求2的头部应该独立"
        );

        // 添加不同的头部到 req1
        req1.add_header("X-Custom", "value1");
        // req2 不应该有这个头部
        assert!(
            req2.get_header("X-Custom").is_none(),
            "req2不应该有req1的头部"
        );
    }

    /// 测试状态一致性
    ///
    /// 一旦请求的状态被设置，后续读取应该返回相同的值
    #[test]
    fn test_state_consistency() {
        let ctx = RequestContext::new(1, "/api/test", "GET");

        // 设置状态多次
        ctx.set_state("state-1".to_string());
        let state1 = ctx.get_state();

        ctx.set_state("state-2".to_string());
        let state2 = ctx.get_state();

        // 验证每次都返回最新的状态
        assert_eq!(state1, Some("state-1".to_string()), "第一次应该返回state-1");
        assert_eq!(state2, Some("state-2".to_string()), "第二次应该返回state-2");
    }

    /// 测试并发规则应用
    ///
    /// 多个规则可以并发应用到不同的请求上
    #[test]
    fn test_concurrent_rule_application() {
        let rule = Arc::new(RuleApplier::new(
            "concurrent-rule",
            "abort",
            Box::new(|_| true),
        ));

        let barrier = Arc::new(Barrier::new(5));
        let mut handles = vec![];

        for i in 0..5 {
            let rule = Arc::clone(&rule);
            let barrier = Arc::clone(&barrier);

            let handle = std::thread::spawn(move || {
                let ctx = RequestContext::new(i as u64, "/api/test", "GET");
                barrier.wait();

                // 并发应用规则
                rule.apply_fault(&ctx);

                // 验证结果
                assert_eq!(
                    ctx.get_state(),
                    Some("concurrent-rule:applied".to_string()),
                    "线程 {} 的规则应该被正确应用",
                    i
                );
            });
            handles.push(handle);
        }

        for handle in handles {
            handle.join().unwrap();
        }
    }

    /// 测试规则条件评估的一致性
    ///
    /// 对于相同的请求，规则条件应该总是返回相同的结果
    #[test]
    fn test_rule_condition_consistency() {
        let ctx = RequestContext::new(1, "/api/users", "GET");

        let rule = RuleApplier::new(
            "consistent-rule",
            "abort",
            Box::new(|ctx| ctx.path.contains("users")),
        );

        // 多次评估规则条件
        let result1 = rule.apply(&ctx);
        let result2 = rule.apply(&ctx);
        let result3 = rule.apply(&ctx);

        assert!(result1, "第一次评估应该返回 true");
        assert!(result2, "第二次评估应该返回 true");
        assert!(result3, "第三次评估应该返回 true");

        // 所有评估应该相同
        assert_eq!(result1, result2, "评估结果应该一致");
        assert_eq!(result2, result3, "评估结果应该一致");
    }
}

#[cfg(test)]
mod advanced_isolation_tests {
    use super::*;

    /// 测试请求清理
    ///
    /// 验证请求的状态可以被正确清理
    #[test]
    fn test_request_cleanup() {
        let ctx = RequestContext::new(1, "/api/test", "GET");
        ctx.set_state("some-state".to_string());
        assert!(ctx.get_state().is_some(), "初始状态应该被设置");

        // 清理状态
        ctx.clear_state();
        assert!(ctx.get_state().is_none(), "清理后状态应该为空");
    }

    /// 测试无泄露验证
    ///
    /// 使用多个请求实例，验证没有全局状态泄露
    #[test]
    fn test_no_global_state_leakage() {
        let requests: Vec<_> = (0..10)
            .map(|i| RequestContext::new(i, &format!("/api/path/{}", i), "GET"))
            .collect();

        // 设置不同的状态
        for (i, req) in requests.iter().enumerate() {
            req.set_state(format!("state-{}", i));
        }

        // 验证没有泄露
        for (i, req) in requests.iter().enumerate() {
            assert_eq!(
                req.get_state(),
                Some(format!("state-{}", i)),
                "请求 {} 的状态应该完全隔离",
                i
            );
        }
    }
}
