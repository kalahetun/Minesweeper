//! W-5 Unit tests for data structures and logic

#[cfg(test)]
mod tests {
    use std::sync::{Arc, RwLock};
    use std::thread;
    use std::time::Duration;
    use crate::CompiledRuleSet;

    /// Test basic rule compilation and storage
    #[test]
    fn test_rule_compilation() {
        let config_json = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "test-rule",
                    "match": {
                        "path": {"exact": "/test"}
                    },
                    "fault": {
                        "percentage": 100,
                        "abort": {
                            "http_status": 500,
                            "body": "Test fault"
                        }
                    }
                }
            ]
        }"#;

        let result = CompiledRuleSet::from_slice(config_json.as_bytes());
        assert!(result.is_ok());

        let ruleset = result.unwrap();
        assert_eq!(ruleset.rules.len(), 1);
        assert_eq!(ruleset.rules[0].name, "test-rule");
        assert_eq!(ruleset.rules[0].fault.percentage, 100);

        // Test path matching
        if let Some(ref path_matcher) = ruleset.rules[0].match_condition.path {
            assert_eq!(path_matcher.exact, Some("/test".to_string()));
        } else {
            panic!("Path matcher should be present");
        }
    }

    /// Test thread-safe rule storage with Arc<RwLock<>>
    #[test]
    fn test_thread_safe_rule_storage() {
        let rules: Arc<RwLock<Option<CompiledRuleSet>>> = Arc::new(RwLock::new(None));

        // Test initial state
        {
            let guard = rules.read().unwrap();
            assert!(guard.is_none());
        }

        // Test rule update
        let config_json = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "test-rule",
                    "match": {
                        "path": {"exact": "/api"}
                    },
                    "fault": {
                        "percentage": 50,
                        "delay": {"fixed_delay": "100ms"}
                    }
                }
            ]
        }"#;

        if let Ok(ruleset) = CompiledRuleSet::from_slice(config_json.as_bytes()) {
            let mut guard = rules.write().unwrap();
            *guard = Some(ruleset);
        }

        // Test read access
        {
            let guard = rules.read().unwrap();
            assert!(guard.is_some());
            if let Some(ref ruleset) = *guard {
                assert_eq!(ruleset.rules.len(), 1);
                assert_eq!(ruleset.rules[0].name, "test-rule");
            }
        }
    }

    /// Test concurrent access to rule storage
    #[test]
    fn test_concurrent_rule_access() {
        let rules: Arc<RwLock<Option<CompiledRuleSet>>> = Arc::new(RwLock::new(None));

        // Setup initial rules
        let config_json = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "concurrent-test",
                    "match": {
                        "path": {"prefix": "/api"}
                    },
                    "fault": {
                        "percentage": 25,
                        "abort": {"http_status": 503}
                    }
                }
            ]
        }"#;

        if let Ok(ruleset) = CompiledRuleSet::from_slice(config_json.as_bytes()) {
            let mut guard = rules.write().unwrap();
            *guard = Some(ruleset);
        }

        let rules_clone = rules.clone();

        // Simulate concurrent read access
        let read_handle = thread::spawn(move || {
            for _ in 0..10 {
                let guard = rules_clone.read().unwrap();
                if let Some(ref ruleset) = *guard {
                    assert_eq!(ruleset.rules[0].name, "concurrent-test");
                }
                thread::sleep(Duration::from_millis(1));
            }
        });

        // Simulate main thread access
        for _ in 0..5 {
            let guard = rules.read().unwrap();
            if let Some(ref ruleset) = *guard {
                assert_eq!(ruleset.rules.len(), 1);
            }
            thread::sleep(Duration::from_millis(2));
        }

        read_handle.join().expect("Thread should complete successfully");
    }

    /// Test basic matcher functionality (without HTTP context)
    #[test]
    fn test_matcher_logic() {
        let config_json = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "exact-match",
                    "match": {
                        "path": {"exact": "/exact"},
                        "method": "GET"
                    },
                    "fault": {
                        "percentage": 100,
                        "abort": {"http_status": 500}
                    }
                },
                {
                    "name": "prefix-match",
                    "match": {
                        "path": {"prefix": "/api/"},
                        "method": "POST"
                    },
                    "fault": {
                        "percentage": 50,
                        "delay": {"fixed_delay": "200ms"}
                    }
                }
            ]
        }"#;

        let ruleset = CompiledRuleSet::from_slice(config_json.as_bytes()).unwrap();
        assert_eq!(ruleset.rules.len(), 2);

        // Verify rule configurations
        assert_eq!(ruleset.rules[0].name, "exact-match");
        if let Some(ref method_matcher) = ruleset.rules[0].match_condition.method {
            assert_eq!(method_matcher.exact, Some("GET".to_string()));
        }

        assert_eq!(ruleset.rules[1].name, "prefix-match");
        if let Some(ref method_matcher) = ruleset.rules[1].match_condition.method {
            assert_eq!(method_matcher.exact, Some("POST".to_string()));
        }
    }

    /// Test executor fault configuration
    #[test]
    fn test_executor_fault_types() {
        let delay_config = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "delay-rule",
                    "match": {"path": {"exact": "/slow"}},
                    "fault": {
                        "percentage": 100,
                        "delay": {"fixed_delay": "500ms"}
                    }
                }
            ]
        }"#;

        let abort_config = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "abort-rule",
                    "match": {"path": {"exact": "/error"}},
                    "fault": {
                        "percentage": 100,
                        "abort": {
                            "http_status": 404,
                            "body": "Not Found"
                        }
                    }
                }
            ]
        }"#;

        // Test delay fault
        let delay_ruleset = CompiledRuleSet::from_slice(delay_config.as_bytes()).unwrap();
        assert!(delay_ruleset.rules[0].fault.delay.is_some());
        assert!(delay_ruleset.rules[0].fault.abort.is_none());

        // Test abort fault
        let abort_ruleset = CompiledRuleSet::from_slice(abort_config.as_bytes()).unwrap();
        assert!(abort_ruleset.rules[0].fault.abort.is_some());
        assert!(abort_ruleset.rules[0].fault.delay.is_none());

        if let Some(ref abort) = abort_ruleset.rules[0].fault.abort {
            assert_eq!(abort.http_status, 404);
            assert_eq!(abort.body, Some("Not Found".to_string()));
        }
    }

    /// Test the integration pattern: Rule storage -> Matcher -> Executor
    #[test]
    fn test_integration_pattern() {
        // Setup rule storage (as would be done in PluginRootContext)
        let rules: Arc<RwLock<Option<CompiledRuleSet>>> = Arc::new(RwLock::new(None));

        let config_json = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "integration-test",
                    "match": {
                        "path": {"exact": "/integration"},
                        "headers": [
                            {"name": "test-header", "exact": "test-value"}
                        ]
                    },
                    "fault": {
                        "percentage": 100,
                        "abort": {
                            "http_status": 503,
                            "body": "Service Unavailable"
                        }
                    }
                }
            ]
        }"#;

        // Step 1: Configuration update (as would happen in on_http_call_response)
        if let Ok(ruleset) = CompiledRuleSet::from_slice(config_json.as_bytes()) {
            let mut guard = rules.write().unwrap();
            *guard = Some(ruleset);
        }

        // Step 2: Request processing (as would happen in on_http_request_headers)
        let guard = rules.read().unwrap();
        if let Some(ref ruleset) = *guard {
            // Simulate finding a matching rule
            let rule = &ruleset.rules[0];
            assert_eq!(rule.name, "integration-test");
            
            // Verify matcher configuration
            if let Some(ref path_matcher) = rule.match_condition.path {
                assert_eq!(path_matcher.exact, Some("/integration".to_string()));
            }
            
            // Verify headers configuration
            assert!(rule.match_condition.headers.is_some());
            if let Some(ref headers) = rule.match_condition.headers {
                assert_eq!(headers.len(), 1);
                assert_eq!(headers[0].name, "test-header");
                assert_eq!(headers[0].exact, Some("test-value".to_string()));
            }

            // Verify fault configuration
            assert_eq!(rule.fault.percentage, 100);
            assert!(rule.fault.abort.is_some());
            if let Some(ref abort) = rule.fault.abort {
                assert_eq!(abort.http_status, 503);
                assert_eq!(abort.body, Some("Service Unavailable".to_string()));
            }
        } else {
            panic!("Rules should be present after configuration");
        }
    }
}
