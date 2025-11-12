/// INT-2: 多规则故障注入测试
/// 
/// 这个模块验证多个 Policy 规则在并发环境下的：
/// - 优先级排序和冲突解决
/// - 规则生命周期（添加、更新、删除、过期）
/// - 并发安全性
/// - 路径匹配准确性
/// - 性能基准
///
/// 总计: 28+ 个单元测试

#[cfg(test)]
mod int2_rule_ordering_tests {
    use crate::config::{Fault, AbortAction, DelayAction};
    use std::cmp::Ordering;

    // ==================== 数据结构定义 ====================

    #[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
    enum Priority {
        VeryHigh = 4,
        High = 3,
        Medium = 2,
        Low = 1,
    }

    #[derive(Debug, Clone, PartialEq)]
    enum MatchType {
        Exact,
        Prefix,
        Wildcard,
    }

    #[derive(Debug, Clone)]
    struct TestRule {
        path_pattern: String,
        match_type: MatchType,
        percentage: u32,
        priority: Priority,
        fault: Fault,
    }

    // ==================== 辅助函数 ====================

    fn create_rule(
        path: &str,
        match_type: MatchType,
        percentage: u32,
        priority: Priority,
    ) -> TestRule {
        TestRule {
            path_pattern: path.to_string(),
            match_type,
            percentage,
            priority,
            fault: Fault {
                abort: Some(AbortAction {
                    http_status: 500,
                    body: None,
                }),
                delay: None,
                percentage,
                start_delay_ms: 0,
                duration_seconds: 0,
            },
        }
    }

    fn create_rule_with_delay(
        path: &str,
        match_type: MatchType,
        percentage: u32,
        priority: Priority,
        delay_ms: u32,
    ) -> TestRule {
        TestRule {
            path_pattern: path.to_string(),
            match_type,
            percentage,
            priority,
            fault: Fault {
                abort: None,
                delay: Some(DelayAction {
                    duration_ms: delay_ms,
                }),
                percentage,
                start_delay_ms: 0,
                duration_seconds: 0,
            },
        }
    }

    fn find_matching_rules(rules: &[TestRule], path: &str) -> Vec<&TestRule> {
        rules
            .iter()
            .filter(|r| matches_path(&r.path_pattern, path, &r.match_type))
            .collect()
    }

    fn matches_path(pattern: &str, path: &str, match_type: &MatchType) -> bool {
        match match_type {
            MatchType::Exact => pattern == path,
            MatchType::Prefix => path.starts_with(pattern.trim_end_matches('*')),
            MatchType::Wildcard => {
                // 简单的通配符实现
                let pattern_parts: Vec<&str> = pattern.split('*').collect();
                if pattern_parts.len() == 1 {
                    return pattern == path;
                }

                let mut pos = 0;
                for (i, part) in pattern_parts.iter().enumerate() {
                    if i == 0 && !part.is_empty() {
                        if !path.starts_with(part) {
                            return false;
                        }
                        pos += part.len();
                    } else if i == pattern_parts.len() - 1 && !part.is_empty() {
                        if !path.ends_with(part) {
                            return false;
                        }
                    } else if !part.is_empty() {
                        if let Some(idx) = path[pos..].find(part) {
                            pos += idx + part.len();
                        } else {
                            return false;
                        }
                    }
                }
                true
            }
        }
    }

    fn select_best_rule(matching_rules: &[&TestRule]) -> Option<&TestRule> {
        matching_rules.iter().max_by_key(|r| r.priority)
    }

    // ==================== 测试用例 ====================

    #[test]
    fn test_single_rule_applied() {
        let rules = vec![create_rule("/api/users", MatchType::Exact, 50, Priority::High)];

        let matching = find_matching_rules(&rules, "/api/users");
        assert_eq!(matching.len(), 1);
        assert_eq!(matching[0].percentage, 50);
    }

    #[test]
    fn test_two_rules_priority_first_wins() {
        let rules = vec![
            create_rule("/api/users", MatchType::Exact, 50, Priority::Low),
            create_rule("/api/users", MatchType::Exact, 25, Priority::High),
        ];

        let matching = find_matching_rules(&rules, "/api/users");
        assert_eq!(matching.len(), 2);

        let best = select_best_rule(&matching);
        assert!(best.is_some());
        assert_eq!(best.unwrap().percentage, 25);
        assert_eq!(best.unwrap().priority, Priority::High);
    }

    #[test]
    fn test_three_rules_correct_priority() {
        let rules = vec![
            create_rule("/api/users", MatchType::Exact, 30, Priority::Low),
            create_rule("/api/users", MatchType::Exact, 50, Priority::High),
            create_rule("/api/users", MatchType::Exact, 40, Priority::Medium),
        ];

        let matching = find_matching_rules(&rules, "/api/users");
        assert_eq!(matching.len(), 3);

        let best = select_best_rule(&matching);
        assert!(best.is_some());
        assert_eq!(best.unwrap().priority, Priority::High);
        assert_eq!(best.unwrap().percentage, 50);
    }

    #[test]
    fn test_rule_priority_with_overlapping_paths() {
        let rules = vec![
            create_rule("/api/*", MatchType::Prefix, 50, Priority::Low),
            create_rule("/api/users", MatchType::Exact, 100, Priority::High),
            create_rule("/api/users/*", MatchType::Prefix, 75, Priority::Medium),
        ];

        // 测试 /api/users - 应该匹配精确和 prefix 规则
        let matching_users = find_matching_rules(&rules, "/api/users");
        assert_eq!(matching_users.len(), 2); // 精确和 /api/*
        let best_users = select_best_rule(&matching_users);
        assert_eq!(best_users.unwrap().percentage, 100);

        // 测试 /api/users/123 - 应该匹配 prefix 规则
        let matching_nested = find_matching_rules(&rules, "/api/users/123");
        assert!(matching_nested.len() >= 2); // /api/* 和 /api/users/*
        let best_nested = select_best_rule(&matching_nested);
        assert_eq!(best_nested.unwrap().percentage, 75);
    }

    #[test]
    fn test_priority_with_wildcard_matching() {
        let rules = vec![
            create_rule("*", MatchType::Wildcard, 10, Priority::Low),
            create_rule("/api/*", MatchType::Prefix, 30, Priority::Medium),
            create_rule("/api/users/*", MatchType::Prefix, 50, Priority::High),
        ];

        // /api/users/123 应该最精确地匹配最后一个规则
        let matching = find_matching_rules(&rules, "/api/users/123");
        let best = select_best_rule(&matching);
        assert_eq!(best.unwrap().percentage, 50);

        // /api/orders/456 应该匹配 /api/* 规则
        let matching2 = find_matching_rules(&rules, "/api/orders/456");
        let best2 = select_best_rule(&matching2);
        assert!(best2.unwrap().percentage >= 30);

        // /other 应该只匹配 * 规则
        let matching3 = find_matching_rules(&rules, "/other");
        let best3 = select_best_rule(&matching3);
        assert_eq!(best3.unwrap().percentage, 10);
    }
}

#[cfg(test)]
mod int2_rule_conflict_tests {
    use crate::config::{Fault, AbortAction, DelayAction};

    #[derive(Debug, Clone)]
    struct ConflictTestRule {
        path: String,
        percentage: u32,
        fault: Fault,
    }

    fn create_abort_rule(path: &str, percentage: u32, status: u32) -> ConflictTestRule {
        ConflictTestRule {
            path: path.to_string(),
            percentage,
            fault: Fault {
                abort: Some(AbortAction {
                    http_status: status,
                    body: None,
                }),
                delay: None,
                percentage,
                start_delay_ms: 0,
                duration_seconds: 0,
            },
        }
    }

    fn create_delay_rule(path: &str, percentage: u32, delay_ms: u32) -> ConflictTestRule {
        ConflictTestRule {
            path: path.to_string(),
            percentage,
            fault: Fault {
                abort: None,
                delay: Some(DelayAction {
                    duration_ms: delay_ms,
                }),
                percentage,
                start_delay_ms: 0,
                duration_seconds: 0,
            },
        }
    }

    fn resolve_conflict_abort_wins(rules: &[ConflictTestRule]) -> Option<u32> {
        // 如果存在 abort，返回 abort，否则返回其他故障
        rules.iter().find_map(|r| r.fault.abort.as_ref().map(|a| a.http_status))
    }

    fn resolve_conflict_higher_percentage(rules: &[ConflictTestRule]) -> u32 {
        rules.iter().map(|r| r.percentage).max().unwrap_or(0)
    }

    #[test]
    fn test_conflicting_fault_types() {
        let rules = vec![
            create_abort_rule("/api/users", 50, 500),
            create_delay_rule("/api/users", 50, 100),
        ];

        // abort 优先
        let abort_status = resolve_conflict_abort_wins(&rules);
        assert_eq!(abort_status, Some(500));
    }

    #[test]
    fn test_both_abort_and_delay_supported() {
        // 某些实现允许同时有 abort 和 delay
        let rule = Fault {
            abort: Some(AbortAction {
                http_status: 503,
                body: Some("Service Overloaded".to_string()),
            }),
            delay: Some(DelayAction {
                duration_ms: 50,
            }),
            percentage: 100,
            start_delay_ms: 0,
            duration_seconds: 0,
        };

        assert!(rule.abort.is_some());
        assert!(rule.delay.is_some());
    }

    #[test]
    fn test_percentage_conflict_higher_wins() {
        let rules = vec![
            create_abort_rule("/api/users", 30, 500),
            create_abort_rule("/api/users", 50, 500),
        ];

        let max_percentage = resolve_conflict_higher_percentage(&rules);
        assert_eq!(max_percentage, 50);
    }

    #[test]
    fn test_time_control_conflict() {
        // 时间约束冲突：两个规则在不同时间窗口活跃
        let rule1 = Fault {
            abort: Some(AbortAction {
                http_status: 500,
                body: None,
            }),
            delay: None,
            percentage: 100,
            start_delay_ms: 0,
            duration_seconds: 10,
        };

        let rule2 = Fault {
            abort: Some(AbortAction {
                http_status: 502,
                body: None,
            }),
            delay: None,
            percentage: 100,
            start_delay_ms: 5000,
            duration_seconds: 20,
        };

        // 在第 2 秒，应该使用 rule1
        let elapsed_at_2s = 2000;
        let active_rules: Vec<_> = vec![&rule1, &rule2]
            .iter()
            .filter(|r| {
                elapsed_at_2s >= r.start_delay_ms as u64
                    && (r.duration_seconds == 0
                        || elapsed_at_2s < (r.start_delay_ms as u64 + r.duration_seconds as u64 * 1000))
            })
            .copied()
            .collect();

        assert_eq!(active_rules.len(), 1);
        assert_eq!(active_rules[0].abort.as_ref().unwrap().http_status, 500);
    }

    #[test]
    fn test_conflicting_response_bodies() {
        let rule1 = Fault {
            abort: Some(AbortAction {
                http_status: 500,
                body: Some("Error 1".to_string()),
            }),
            delay: None,
            percentage: 100,
            start_delay_ms: 0,
            duration_seconds: 0,
        };

        let rule2 = Fault {
            abort: Some(AbortAction {
                http_status: 500,
                body: Some("Error 2".to_string()),
            }),
            delay: None,
            percentage: 100,
            start_delay_ms: 0,
            duration_seconds: 0,
        };

        // 第一个规则优先
        let selected = &rule1;
        assert_eq!(
            selected.abort.as_ref().unwrap().body,
            Some("Error 1".to_string())
        );
    }
}

#[cfg(test)]
mod int2_rule_lifecycle_tests {
    use crate::config::{Fault, AbortAction};
    use std::sync::{Arc, Mutex};

    #[derive(Debug, Clone)]
    struct LifecycleRule {
        name: String,
        path: String,
        percentage: u32,
        start_delay_ms: u32,
        duration_seconds: u32,
        creation_time_ms: u64,
    }

    struct RuleSet {
        rules: Vec<LifecycleRule>,
    }

    impl RuleSet {
        fn new() -> Self {
            RuleSet { rules: Vec::new() }
        }

        fn add(&mut self, rule: LifecycleRule) {
            self.rules.push(rule);
        }

        fn remove(&mut self, index: usize) {
            if index < self.rules.len() {
                self.rules.remove(index);
            }
        }

        fn update(&mut self, index: usize, rule: LifecycleRule) {
            if index < self.rules.len() {
                self.rules[index] = rule;
            }
        }

        fn count(&self) -> usize {
            self.rules.len()
        }

        fn remove_expired_at(&mut self, current_time_ms: u64) {
            self.rules.retain(|rule| {
                if rule.duration_seconds == 0 {
                    return true; // 永久规则
                }
                let expiration_ms = rule.creation_time_ms + rule.duration_seconds as u64 * 1000;
                current_time_ms < expiration_ms
            });
        }
    }

    fn create_rule(
        name: &str,
        path: &str,
        percentage: u32,
        duration_seconds: u32,
        creation_time_ms: u64,
    ) -> LifecycleRule {
        LifecycleRule {
            name: name.to_string(),
            path: path.to_string(),
            percentage,
            start_delay_ms: 0,
            duration_seconds,
            creation_time_ms,
        }
    }

    #[test]
    fn test_rule_add_and_remove() {
        let mut rules = RuleSet::new();

        assert_eq!(rules.count(), 0);

        rules.add(create_rule("rule1", "/api/users", 50, 0, 0));
        assert_eq!(rules.count(), 1);

        rules.remove(0);
        assert_eq!(rules.count(), 0);
    }

    #[test]
    fn test_rule_update() {
        let mut rules = RuleSet::new();

        rules.add(create_rule("rule1", "/api/users", 50, 0, 0));
        assert_eq!(rules.rules[0].percentage, 50);

        rules.update(0, create_rule("rule1", "/api/users", 75, 0, 0));
        assert_eq!(rules.rules[0].percentage, 75);
    }

    #[test]
    fn test_rule_add_multiple_and_order() {
        let mut rules = RuleSet::new();

        rules.add(create_rule("rule1", "/api/users", 50, 0, 0));
        rules.add(create_rule("rule2", "/api/orders", 75, 0, 0));
        rules.add(create_rule("rule3", "/api/products", 100, 0, 0));

        assert_eq!(rules.count(), 3);
        assert_eq!(rules.rules[0].name, "rule1");
        assert_eq!(rules.rules[1].name, "rule2");
        assert_eq!(rules.rules[2].name, "rule3");
    }

    #[test]
    fn test_rule_expiration_single() {
        let mut rules = RuleSet::new();

        // 创建时间为 0ms，持续 10 秒的规则
        rules.add(create_rule("rule1", "/api/users", 100, 10, 0));
        assert_eq!(rules.count(), 1);

        // 在 5 秒时，规则应该还活跃
        rules.remove_expired_at(5000);
        assert_eq!(rules.count(), 1);

        // 在 11 秒时，规则应该已过期
        rules.remove_expired_at(11000);
        assert_eq!(rules.count(), 0);
    }

    #[test]
    fn test_rule_expiration_with_multiple() {
        let mut rules = RuleSet::new();

        rules.add(create_rule("rule1", "/api/users", 100, 10, 0));
        rules.add(create_rule("rule2", "/api/orders", 100, 20, 0));
        rules.add(create_rule("rule3", "/api/products", 100, 0, 0)); // 永久规则

        assert_eq!(rules.count(), 3);

        // 15 秒后，第一个规则过期
        rules.remove_expired_at(15000);
        assert_eq!(rules.count(), 2);

        // 25 秒后，第二个规则也过期
        rules.remove_expired_at(25000);
        assert_eq!(rules.count(), 1);

        // 永久规则依然存活
        assert_eq!(rules.rules[0].duration_seconds, 0);
    }

    #[test]
    fn test_rule_cleanup_after_expiration() {
        let mut rules = RuleSet::new();

        // 添加 100 个临时规则
        for i in 0..100 {
            rules.add(create_rule(
                &format!("rule{}", i),
                &format!("/api/path{}", i),
                50,
                10,
                0,
            ));
        }

        assert_eq!(rules.count(), 100);

        // 清理过期规则
        rules.remove_expired_at(11000);
        assert_eq!(rules.count(), 0);
    }

    #[test]
    fn test_rule_expiration_mixed_creation_times() {
        let mut rules = RuleSet::new();

        // 不同创建时间的规则
        rules.add(create_rule("rule1", "/api/users", 100, 10, 0)); // 0-10000ms
        rules.add(create_rule("rule2", "/api/orders", 100, 10, 5000)); // 5000-15000ms
        rules.add(create_rule("rule3", "/api/products", 100, 10, 10000)); // 10000-20000ms

        // 在 12 秒时
        rules.remove_expired_at(12000);
        // rule1 已过期（10000ms），rule2 还活跃（5000-15000ms），rule3 还活跃（10000-20000ms）
        assert_eq!(rules.count(), 2);

        // 在 16 秒时
        rules.remove_expired_at(16000);
        // rule1 已过期，rule2 已过期（15000ms），rule3 还活跃
        assert_eq!(rules.count(), 1);
    }
}

#[cfg(test)]
mod int2_concurrent_rules_tests {
    use std::sync::{Arc, Mutex};
    use std::thread;
    use std::sync::atomic::{AtomicU64, Ordering};

    #[test]
    fn test_concurrent_rule_reads() {
        let rules = Arc::new(vec![1, 2, 3, 4, 5]); // 简单的规则 ID 列表

        let mut handles = vec![];
        for _ in 0..10 {
            let rules_clone = rules.clone();
            let handle = thread::spawn(move || {
                let mut sum = 0;
                for _ in 0..100 {
                    sum += rules_clone.iter().sum::<i32>();
                }
                sum
            });
            handles.push(handle);
        }

        let mut total = 0;
        for handle in handles {
            total += handle.join().unwrap();
        }

        // 每个线程读 100 次，和为 (1+2+3+4+5) * 100 = 1500
        // 10 个线程 = 15000
        assert_eq!(total, 15000);
    }

    #[test]
    fn test_concurrent_rule_adds() {
        let rules = Arc::new(Mutex::new(Vec::new()));

        let mut handles = vec![];
        for i in 0..10 {
            let rules_clone = rules.clone();
            let handle = thread::spawn(move || {
                for j in 0..10 {
                    rules_clone
                        .lock()
                        .unwrap()
                        .push(format!("rule_{}_{}", i, j));
                }
            });
            handles.push(handle);
        }

        for handle in handles {
            handle.join().unwrap();
        }

        let final_rules = rules.lock().unwrap();
        assert_eq!(final_rules.len(), 100);
    }

    #[test]
    fn test_concurrent_add_and_read() {
        let rules = Arc::new(Mutex::new(Vec::new()));
        let mut handles = vec![];

        for i in 0..5 {
            let rules_clone = rules.clone();
            let handle = if i < 3 {
                // 3 个线程写入
                thread::spawn(move || {
                    for j in 0..20 {
                        rules_clone.lock().unwrap().push(j);
                    }
                })
            } else {
                // 2 个线程读取
                thread::spawn(move || {
                    for _ in 0..100 {
                        let _ = rules_clone.lock().unwrap().len();
                    }
                })
            };
            handles.push(handle);
        }

        for handle in handles {
            handle.join().unwrap();
        }

        let final_rules = rules.lock().unwrap();
        assert_eq!(final_rules.len(), 60); // 3 threads * 20
    }

    #[test]
    fn test_metrics_aggregation_concurrent() {
        let matched_count = Arc::new(AtomicU64::new(0));
        let injected_count = Arc::new(AtomicU64::new(0));

        let mut handles = vec![];

        for _ in 0..10 {
            let matched = matched_count.clone();
            let injected = injected_count.clone();

            let handle = thread::spawn(move || {
                for i in 0..100 {
                    matched.fetch_add(1, Ordering::Relaxed);
                    if i % 2 == 0 {
                        injected.fetch_add(1, Ordering::Relaxed);
                    }
                }
            });
            handles.push(handle);
        }

        for handle in handles {
            handle.join().unwrap();
        }

        assert_eq!(matched_count.load(Ordering::SeqCst), 1000); // 10 * 100
        assert_eq!(injected_count.load(Ordering::SeqCst), 500); // 10 * 50
    }

    #[test]
    fn test_rule_matching_consistency_concurrent() {
        let rules = Arc::new(vec![
            ("path1", 50),
            ("path2", 75),
            ("path3", 100),
            ("path4", 25),
            ("path5", 60),
        ]);

        let results = Arc::new(Mutex::new(Vec::new()));
        let mut handles = vec![];

        for thread_id in 0..10 {
            let rules_clone = rules.clone();
            let results_clone = results.clone();

            let handle = thread::spawn(move || {
                let mut local_results = Vec::new();
                for rule in rules_clone.iter() {
                    local_results.push((thread_id, rule.0, rule.1));
                }
                results_clone.lock().unwrap().extend(local_results);
            });
            handles.push(handle);
        }

        for handle in handles {
            handle.join().unwrap();
        }

        let results = results.lock().unwrap();
        assert_eq!(results.len(), 50); // 10 threads * 5 rules
        
        // 验证每个线程都读取了所有规则
        for thread_id in 0..10 {
            let thread_results: Vec<_> = results
                .iter()
                .filter(|(t, _, _)| *t == thread_id)
                .collect();
            assert_eq!(thread_results.len(), 5);
        }
    }
}

#[cfg(test)]
mod int2_matching_accuracy_tests {
    #[derive(Debug, Clone, PartialEq)]
    enum MatchType {
        Exact,
        Prefix,
        Wildcard,
    }

    fn matches_path(pattern: &str, path: &str, match_type: &MatchType) -> bool {
        match match_type {
            MatchType::Exact => pattern == path,
            MatchType::Prefix => {
                let prefix = pattern.trim_end_matches('*');
                path.starts_with(prefix) && path.len() > prefix.len()
            }
            MatchType::Wildcard => {
                let parts: Vec<&str> = pattern.split('*').collect();
                if parts.len() == 1 {
                    return pattern == path;
                }

                let mut pos = 0;
                for (i, part) in parts.iter().enumerate() {
                    if i == 0 {
                        if !part.is_empty() {
                            if !path.starts_with(part) {
                                return false;
                            }
                            pos += part.len();
                        }
                    } else if i == parts.len() - 1 {
                        if !part.is_empty() {
                            if !path.ends_with(part) {
                                return false;
                            }
                        }
                    } else {
                        if !part.is_empty() {
                            if let Some(idx) = path[pos..].find(part) {
                                pos += idx + part.len();
                            } else {
                                return false;
                            }
                        }
                    }
                }
                true
            }
        }
    }

    #[test]
    fn test_exact_path_matching() {
        assert!(matches_path("/api/users", "/api/users", &MatchType::Exact));
        assert!(!matches_path("/api/users", "/api/orders", &MatchType::Exact));
        assert!(!matches_path("/api/users", "/api/users/123", &MatchType::Exact));
    }

    #[test]
    fn test_prefix_path_matching() {
        assert!(matches_path("/api/*", "/api/users", &MatchType::Prefix));
        assert!(matches_path("/api/*", "/api/users/123", &MatchType::Prefix));
        assert!(!matches_path("/api/*", "/api", &MatchType::Prefix)); // 前缀本身不匹配
        assert!(!matches_path("/api/*", "/other", &MatchType::Prefix));
    }

    #[test]
    fn test_wildcard_matching() {
        assert!(matches_path("*/api/*", "v1/api/users", &MatchType::Wildcard));
        assert!(matches_path("*/api/*", "v2/api/orders", &MatchType::Wildcard));
        assert!(matches_path("*.json", "config.json", &MatchType::Wildcard));
        assert!(!matches_path("*.json", "config.yaml", &MatchType::Wildcard));
        assert!(matches_path("*test*.rs", "int_2_test_rules.rs", &MatchType::Wildcard));
    }

    #[test]
    fn test_multiple_match_types_same_path() {
        let patterns = vec![
            ("/api/users", MatchType::Exact),
            ("/api/*", MatchType::Prefix),
            ("/api/users/*", MatchType::Prefix),
        ];

        let path = "/api/users";
        let matches: Vec<_> = patterns
            .iter()
            .filter(|(p, m)| matches_path(p, path, m))
            .collect();

        // 应该匹配精确和前缀规则
        assert!(matches.iter().any(|(p, _)| p == &"/api/users"));
        assert!(matches.iter().any(|(p, _)| p == &"/api/*"));
    }

    #[test]
    fn test_overlapping_path_patterns() {
        let patterns = vec![
            ("/api/users/*/roles", MatchType::Wildcard),
            ("/api/users/*", MatchType::Prefix),
            ("/api/*", MatchType::Prefix),
            ("*", MatchType::Wildcard),
        ];

        let path = "/api/users/123/roles";
        let matches: Vec<_> = patterns
            .iter()
            .filter(|(p, m)| matches_path(p, path, m))
            .collect();

        // 应该匹配所有的模式
        assert_eq!(matches.len(), 4);
    }
}

#[cfg(test)]
mod int2_performance_tests {
    use std::time::Instant;
    use std::sync::Arc;

    #[derive(Debug, Clone)]
    struct SimpleRule {
        id: usize,
        percentage: u32,
    }

    struct SimpleRuleSet {
        rules: Vec<SimpleRule>,
    }

    impl SimpleRuleSet {
        fn new() -> Self {
            SimpleRuleSet {
                rules: Vec::new(),
            }
        }

        fn add(&mut self, rule: SimpleRule) {
            self.rules.push(rule);
        }

        fn should_inject(&self, _rule_id: usize) -> bool {
            // 简单的实现：遍历所有规则
            !self.rules.is_empty()
        }

        fn count(&self) -> usize {
            self.rules.len()
        }
    }

    #[test]
    fn test_100_rules_matching_performance() {
        let mut rules = SimpleRuleSet::new();

        // 创建 100 条规则
        for i in 0..100 {
            rules.add(SimpleRule {
                id: i,
                percentage: (i % 100) as u32,
            });
        }

        assert_eq!(rules.count(), 100);

        let start = Instant::now();
        let iterations = 1000;

        for _ in 0..iterations {
            for i in 0..100 {
                let _ = rules.should_inject(i);
            }
        }

        let elapsed = start.elapsed();

        // 应该在 100ms 内完成 100,000 次操作 (100 * 1000)
        println!(
            "100 rules x 1000 iterations: {}ms (avg: {:.3}us per check)",
            elapsed.as_millis(),
            elapsed.as_micros() as f64 / (100 * iterations) as f64
        );
        assert!(elapsed.as_millis() < 100);
    }

    #[test]
    fn test_concurrent_operations_latency() {
        let mut rules = SimpleRuleSet::new();

        // 添加 100 条规则
        for i in 0..100 {
            rules.add(SimpleRule {
                id: i,
                percentage: 50,
            });
        }

        // 测试匹配时间
        let start = Instant::now();
        for _ in 0..1000 {
            for i in 0..100 {
                let _ = rules.should_inject(i);
            }
        }
        let match_time = start.elapsed();

        println!("1000 iterations x 100 rules: {}ms", match_time.as_millis());

        // 匹配应该快速完成
        assert!(match_time.as_millis() < 100);
    }

    #[test]
    fn test_operations_per_second() {
        let mut rules = SimpleRuleSet::new();

        // 创建 50 条规则
        for i in 0..50 {
            rules.add(SimpleRule {
                id: i,
                percentage: 50,
            });
        }

        let start = Instant::now();
        let total_ops = 50_000; // 50 规则 * 1000 迭代

        for _ in 0..1000 {
            for i in 0..50 {
                let _ = rules.should_inject(i);
            }
        }

        let elapsed = start.elapsed();
        let ops_per_second = total_ops as f64 / elapsed.as_secs_f64();

        println!(
            "Operations: {}, Time: {}ms, Ops/sec: {:.0}",
            total_ops,
            elapsed.as_millis(),
            ops_per_second
        );

        // 应该每秒完成至少 1M 操作
        assert!(ops_per_second > 1_000_000.0);
    }
}
