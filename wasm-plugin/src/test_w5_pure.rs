//! W-5 Pure unit tests - Testing data structures and algorithms without proxy-wasm dependencies

#[cfg(test)]
mod tests {
    use std::sync::{Arc, RwLock};
    use std::thread;
    use std::time::Duration;

    // Import only the data types we need
    use crate::config::{CompiledRuleSet, CompiledRule, MatchCondition, PathMatcher, StringMatcher, Fault, AbortAction, DelayAction};

    /// Test basic JSON parsing and rule compilation
    #[test]
    fn test_config_parsing() {
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
                            "httpStatus": 500,
                            "body": "Test fault"
                        }
                    }
                }
            ]
        }"#;

        let result = CompiledRuleSet::from_slice(config_json.as_bytes());
        assert!(result.is_ok(), "Failed to parse configuration JSON");

        let ruleset = result.unwrap();
        assert_eq!(ruleset.rules.len(), 1);
        assert_eq!(ruleset.rules[0].name, "test-rule");
        assert_eq!(ruleset.rules[0].fault.percentage, 100);

        // Verify path matcher
        assert!(ruleset.rules[0].match_condition.path.is_some());
        if let Some(ref path_matcher) = ruleset.rules[0].match_condition.path {
            assert_eq!(path_matcher.exact, Some("/test".to_string()));
        }

        // Verify fault configuration
        assert!(ruleset.rules[0].fault.abort.is_some());
        if let Some(ref abort) = ruleset.rules[0].fault.abort {
            assert_eq!(abort.http_status, 500);
            assert_eq!(abort.body, Some("Test fault".to_string()));
        }
    }

    /// Test Arc<RwLock<Option<CompiledRuleSet>>> pattern for thread-safe rule storage
    #[test]
    fn test_thread_safe_rule_storage() {
        let rules: Arc<RwLock<Option<CompiledRuleSet>>> = Arc::new(RwLock::new(None));

        // Test initial empty state
        {
            let guard = rules.read().expect("Failed to acquire read lock");
            assert!(guard.is_none(), "Rules should initially be None");
        }

        // Test rule update with write lock
        let config_json = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "thread-safe-test",
                    "match": {
                        "path": {"prefix": "/api"}
                    },
                    "fault": {
                        "percentage": 50,
                        "delay": {"fixedDelay": "100ms"}
                    }
                }
            ]
        }"#;

        // Update rules using write lock
        if let Ok(ruleset) = CompiledRuleSet::from_slice(config_json.as_bytes()) {
            let mut guard = rules.write().expect("Failed to acquire write lock");
            *guard = Some(ruleset);
        }

        // Verify update with read lock
        {
            let guard = rules.read().expect("Failed to acquire read lock");
            assert!(guard.is_some(), "Rules should be present after update");
            
            if let Some(ref ruleset) = *guard {
                assert_eq!(ruleset.rules.len(), 1);
                assert_eq!(ruleset.rules[0].name, "thread-safe-test");
                assert_eq!(ruleset.rules[0].fault.percentage, 50);
                
                // Verify path matcher
                if let Some(ref path_matcher) = ruleset.rules[0].match_condition.path {
                    assert_eq!(path_matcher.prefix, Some("/api".to_string()));
                }
                
                // Verify delay fault
                assert!(ruleset.rules[0].fault.delay.is_some());
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
                    "name": "concurrent-access-test",
                    "match": {
                        "path": {"exact": "/concurrent"}
                    },
                    "fault": {
                        "percentage": 25,
                        "abort": {"httpStatus": 503}
                    }
                }
            ]
        }"#;

        // Initialize rules
        if let Ok(ruleset) = CompiledRuleSet::from_slice(config_json.as_bytes()) {
            let mut guard = rules.write().expect("Failed to acquire write lock");
            *guard = Some(ruleset);
        }

        let rules_clone = rules.clone();

        // Spawn reader thread
        let read_handle = thread::spawn(move || {
            for _ in 0..10 {
                let guard = rules_clone.read().expect("Failed to acquire read lock");
                if let Some(ref ruleset) = *guard {
                    assert_eq!(ruleset.rules[0].name, "concurrent-access-test");
                    assert_eq!(ruleset.rules[0].fault.percentage, 25);
                }
                thread::sleep(Duration::from_millis(1));
            }
        });

        // Main thread also reads
        for _ in 0..5 {
            let guard = rules.read().expect("Failed to acquire read lock");
            if let Some(ref ruleset) = *guard {
                assert_eq!(ruleset.rules.len(), 1);
                assert_eq!(ruleset.rules[0].name, "concurrent-access-test");
            }
            thread::sleep(Duration::from_millis(2));
        }

        read_handle.join().expect("Reader thread should complete successfully");
    }

    /// Test different fault types (delay vs abort)
    #[test]
    fn test_fault_types() {
        // Test delay fault
        let delay_config = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "delay-rule",
                    "match": {"path": {"exact": "/slow"}},
                    "fault": {
                        "percentage": 100,
                        "delay": {"fixedDelay": "500ms"}
                    }
                }
            ]
        }"#;

        let delay_ruleset = CompiledRuleSet::from_slice(delay_config.as_bytes()).unwrap();
        assert!(delay_ruleset.rules[0].fault.delay.is_some());
        assert!(delay_ruleset.rules[0].fault.abort.is_none());

        // Test abort fault
        let abort_config = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "abort-rule",
                    "match": {"path": {"exact": "/error"}},
                    "fault": {
                        "percentage": 100,
                        "abort": {
                            "httpStatus": 404,
                            "body": "Not Found"
                        }
                    }
                }
            ]
        }"#;

        let abort_ruleset = CompiledRuleSet::from_slice(abort_config.as_bytes()).unwrap();
        assert!(abort_ruleset.rules[0].fault.abort.is_some());
        assert!(abort_ruleset.rules[0].fault.delay.is_none());

        if let Some(ref abort) = abort_ruleset.rules[0].fault.abort {
            assert_eq!(abort.http_status, 404);
            assert_eq!(abort.body, Some("Not Found".to_string()));
        }
    }

    /// Test the W-5 integration pattern for rule management
    #[test]
    fn test_w5_integration_pattern() {
        // This test simulates the W-5 integration pattern:
        // 1. Configuration update (on_http_call_response)
        // 2. Rule storage (Arc<RwLock<Option<CompiledRuleSet>>>)
        // 3. Request processing (on_http_request_headers)

        let rules: Arc<RwLock<Option<CompiledRuleSet>>> = Arc::new(RwLock::new(None));

        let config_json = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "w5-integration-test",
                    "match": {
                        "path": {"exact": "/integration"},
                        "headers": [
                            {"name": "test-header", "exact": "test-value"}
                        ]
                    },
                    "fault": {
                        "percentage": 100,
                        "abort": {
                            "httpStatus": 503,
                            "body": "Service Unavailable"
                        }
                    }
                }
            ]
        }"#;

        // Step 1: Configuration update (simulates on_http_call_response)
        if let Ok(ruleset) = CompiledRuleSet::from_slice(config_json.as_bytes()) {
            let mut guard = rules.write().expect("Failed to acquire write lock");
            *guard = Some(ruleset);
        }

        // Step 2: Request processing (simulates on_http_request_headers)
        let guard = rules.read().expect("Failed to acquire read lock");
        if let Some(ref ruleset) = *guard {
            let rule = &ruleset.rules[0];
            assert_eq!(rule.name, "w5-integration-test");
            
            // Verify path matcher (used by matcher::find_first_match)
            if let Some(ref path_matcher) = rule.match_condition.path {
                assert_eq!(path_matcher.exact, Some("/integration".to_string()));
            }
            
            // Verify headers matcher
            assert!(rule.match_condition.headers.is_some());
            if let Some(ref headers) = rule.match_condition.headers {
                assert_eq!(headers.len(), 1);
                assert_eq!(headers[0].name, "test-header");
                assert_eq!(headers[0].exact, Some("test-value".to_string()));
            }

            // Verify fault configuration (used by executor::execute_fault)
            assert_eq!(rule.fault.percentage, 100);
            assert!(rule.fault.abort.is_some());
            if let Some(ref abort) = rule.fault.abort {
                assert_eq!(abort.http_status, 503);
                assert_eq!(abort.body, Some("Service Unavailable".to_string()));
            }
        } else {
            panic!("Rules should be present after configuration update");
        }
    }

    /// Test complex rule configurations with multiple matchers
    #[test]
    fn test_complex_rule_configuration() {
        let config_json = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "complex-rule",
                    "match": {
                        "path": {"prefix": "/api/v1"},
                        "method": {"exact": "POST"},
                        "headers": [
                            {"name": "content-type", "exact": "application/json"},
                            {"name": "authorization", "prefix": "Bearer "}
                        ]
                    },
                    "fault": {
                        "percentage": 75,
                        "delay": {"fixedDelay": "250ms"}
                    }
                }
            ]
        }"#;

        let ruleset = CompiledRuleSet::from_slice(config_json.as_bytes()).unwrap();
        let rule = &ruleset.rules[0];

        // Verify all matchers are properly configured
        assert_eq!(rule.name, "complex-rule");
        
        // Path matcher
        if let Some(ref path_matcher) = rule.match_condition.path {
            assert_eq!(path_matcher.prefix, Some("/api/v1".to_string()));
        }

        // Method matcher  
        if let Some(ref method_matcher) = rule.match_condition.method {
            assert_eq!(method_matcher.exact, Some("POST".to_string()));
        }

        // Headers matchers
        if let Some(ref headers) = rule.match_condition.headers {
            assert_eq!(headers.len(), 2);
            
            assert_eq!(headers[0].name, "content-type");
            assert_eq!(headers[0].exact, Some("application/json".to_string()));
            
            assert_eq!(headers[1].name, "authorization");
            assert_eq!(headers[1].prefix, Some("Bearer ".to_string()));
        }

        // Fault configuration
        assert_eq!(rule.fault.percentage, 75);
        assert!(rule.fault.delay.is_some());
        assert!(rule.fault.abort.is_none());
    }

    /// Test error handling in configuration parsing
    #[test]
    fn test_configuration_error_handling() {
        // Test invalid JSON
        let invalid_json = r#"{"version": "1.0", "rules": [invalid}"#;
        let result = CompiledRuleSet::from_slice(invalid_json.as_bytes());
        assert!(result.is_err(), "Should fail on invalid JSON");

        // Test missing required fields
        let incomplete_config = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "incomplete-rule",
                    "match": {}
                }
            ]
        }"#;
        let result = CompiledRuleSet::from_slice(incomplete_config.as_bytes());
        assert!(result.is_err(), "Should fail on missing fault configuration");
    }
}
