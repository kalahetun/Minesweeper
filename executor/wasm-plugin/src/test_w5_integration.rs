//! W-5 Integration tests

#[cfg(test)]
mod tests {
    use std::sync::{Arc, RwLock};
    use std::thread;
    use crate::CompiledRuleSet;
    
    #[test]
    fn test_rules_update_integration() {
        // Test that rules can be properly updated in thread-safe manner
        let rules = Arc::new(RwLock::new(None::<CompiledRuleSet>));
        
        // Simulate configuration update
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
        
        // Parse and update rules
        if let Ok(ruleset) = CompiledRuleSet::from_slice(config_json.as_bytes()) {
            let mut guard = rules.write().expect("Failed to acquire write lock");
            *guard = Some(ruleset);
        }
        
        // Verify rules were updated
        let guard = rules.read().expect("Failed to acquire read lock");
        assert!(guard.is_some());
        if let Some(ref ruleset) = *guard {
            assert_eq!(ruleset.rules.len(), 1);
            assert_eq!(ruleset.rules[0].name, "test-rule");
            assert_eq!(ruleset.rules[0].fault.percentage, 100);
        }
    }
    
    #[test]
    fn test_concurrent_rules_access() {
        use std::thread;
        use std::time::Duration;
        
        let rules = Arc::new(RwLock::new(None::<CompiledRuleSet>));
        let rules_clone = rules.clone();
        
        // Simulate concurrent read access
        let read_handle = thread::spawn(move || {
            for _ in 0..10 {
                if let Ok(_guard) = rules_clone.read() {
                    // Simulate read operation
                    thread::sleep(Duration::from_millis(1));
                }
            }
        });
        
        // Simulate configuration updates
        let write_handle = thread::spawn(move || {
            for i in 0..5 {
                if let Ok(mut guard) = rules.write() {
                    // Simulate rule update
                    *guard = None; // Reset for test
                    thread::sleep(Duration::from_millis(2));
                }
            }
        });
        
        // Wait for both threads to complete
        read_handle.join().expect("Read thread should complete");
        write_handle.join().expect("Write thread should complete");
    }
}
