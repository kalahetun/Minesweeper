// WASM Plugin Configuration Parsing Tests
// This test file can be compiled for native targets (not WASM)

#[cfg(test)]
mod wasm_config_tests {

    // Simplified type definitions for testing (mirroring config.rs)
    #[derive(Debug, serde::Deserialize, Clone)]
    pub struct CompiledRuleSet {
        pub version: String,
        pub rules: Vec<CompiledRule>,
    }

    #[derive(Debug, serde::Deserialize, Clone)]
    pub struct CompiledRule {
        pub name: String,
        #[serde(rename = "match")]
        pub match_condition: MatchCondition,
        pub fault: Fault,
    }

    #[derive(Debug, serde::Deserialize, Clone)]
    pub struct MatchCondition {
        pub path: Option<PathMatcher>,
        pub method: Option<StringMatcher>,
        pub headers: Option<Vec<HeaderMatcher>>,
    }

    #[derive(Debug, serde::Deserialize, Clone)]
    pub struct PathMatcher {
        pub prefix: Option<String>,
        pub exact: Option<String>,
        pub regex: Option<String>,
    }

    #[derive(Debug, serde::Deserialize, Clone)]
    pub struct StringMatcher {
        pub exact: Option<String>,
        pub prefix: Option<String>,
        pub regex: Option<String>,
    }

    #[derive(Debug, serde::Deserialize, Clone)]
    pub struct HeaderMatcher {
        pub name: String,
        pub exact: Option<String>,
        pub prefix: Option<String>,
        pub regex: Option<String>,
    }

    #[derive(Debug, serde::Deserialize, Clone)]
    pub struct Fault {
        pub abort: Option<AbortAction>,
        pub delay: Option<DelayAction>,
        pub percentage: u32,
        #[serde(rename = "start_delay_ms")]
        #[serde(default)]
        pub start_delay_ms: u32,
        #[serde(rename = "duration_seconds")]
        #[serde(default)]
        pub duration_seconds: u32,
    }

    #[derive(Debug, serde::Deserialize, Clone)]
    pub struct AbortAction {
        #[serde(rename = "httpStatus")]
        pub http_status: u32,
        pub body: Option<String>,
    }

    #[derive(Debug, serde::Deserialize, Clone)]
    pub struct DelayAction {
        #[serde(rename = "fixed_delay")]
        pub fixed_delay: String,
    }

    #[test]
    fn test_parse_valid_ruleset() {
        let json_str = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "test-rule",
                    "match": {
                        "path": {
                            "exact": "/api/test"
                        }
                    },
                    "fault": {
                        "percentage": 50,
                        "abort": {
                            "httpStatus": 500
                        }
                    }
                }
            ]
        }"#;

        let ruleset: CompiledRuleSet = serde_json::from_str(json_str).unwrap();
        assert_eq!(ruleset.version, "1.0");
        assert_eq!(ruleset.rules.len(), 1);
        assert_eq!(ruleset.rules[0].name, "test-rule");
        assert_eq!(ruleset.rules[0].fault.percentage, 50);
    }

    #[test]
    fn test_parse_ruleset_with_delay() {
        let json_str = r#"{
            "version": "2.0",
            "rules": [
                {
                    "name": "delay-rule",
                    "match": {
                        "path": {
                            "prefix": "/api"
                        }
                    },
                    "fault": {
                        "percentage": 100,
                        "delay": {
                            "fixed_delay": "2s"
                        }
                    }
                }
            ]
        }"#;

        let ruleset: CompiledRuleSet = serde_json::from_str(json_str).unwrap();
        let rule = &ruleset.rules[0];
        assert!(rule.fault.delay.is_some());
        assert_eq!(rule.fault.delay.as_ref().unwrap().fixed_delay, "2s");
    }

    #[test]
    fn test_parse_multiple_rules() {
        let json_str = r#"{
            "version": "3.0",
            "rules": [
                {
                    "name": "rule-1",
                    "match": {"path": {"exact": "/users"}},
                    "fault": {"percentage": 25, "abort": {"httpStatus": 500}}
                },
                {
                    "name": "rule-2",
                    "match": {"path": {"exact": "/orders"}},
                    "fault": {"percentage": 50, "delay": {"fixed_delay": "1s"}}
                },
                {
                    "name": "rule-3",
                    "match": {"method": {"exact": "POST"}},
                    "fault": {"percentage": 75, "abort": {"httpStatus": 503}}
                }
            ]
        }"#;

        let ruleset: CompiledRuleSet = serde_json::from_str(json_str).unwrap();
        assert_eq!(ruleset.rules.len(), 3);
        assert_eq!(ruleset.rules[0].name, "rule-1");
        assert_eq!(ruleset.rules[1].name, "rule-2");
        assert_eq!(ruleset.rules[2].name, "rule-3");
    }

    #[test]
    fn test_parse_header_matchers() {
        let json_str = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "header-rule",
                    "match": {
                        "headers": [
                            {"name": "Authorization", "regex": "Bearer .*"},
                            {"name": "Content-Type", "prefix": "application/"}
                        ]
                    },
                    "fault": {"percentage": 100, "abort": {"httpStatus": 401}}
                }
            ]
        }"#;

        let ruleset: CompiledRuleSet = serde_json::from_str(json_str).unwrap();
        let headers = &ruleset.rules[0].match_condition.headers;
        assert!(headers.is_some());
        let headers = headers.as_ref().unwrap();
        assert_eq!(headers.len(), 2);
        assert_eq!(headers[0].name, "Authorization");
        assert_eq!(headers[1].name, "Content-Type");
    }

    #[test]
    fn test_parse_percentage_boundary() {
        for percentage in [0, 1, 50, 99, 100].iter() {
            let json_str = format!(
                r#"{{
                "version": "1.0",
                "rules": [{{
                    "name": "pct-rule",
                    "match": {{"path": {{"exact": "/test"}}}},
                    "fault": {{"percentage": {}, "abort": {{"httpStatus": 500}}}}
                }}]
            }}"#,
                percentage
            );

            let ruleset: CompiledRuleSet = serde_json::from_str(&json_str).unwrap();
            assert_eq!(ruleset.rules[0].fault.percentage, *percentage);
        }
    }

    #[test]
    fn test_parse_timing_controls() {
        let json_str = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "timed-rule",
                    "match": {"path": {"exact": "/api/test"}},
                    "fault": {
                        "percentage": 100,
                        "start_delay_ms": 1000,
                        "duration_seconds": 300,
                        "abort": {"httpStatus": 500}
                    }
                }
            ]
        }"#;

        let ruleset: CompiledRuleSet = serde_json::from_str(json_str).unwrap();
        let fault = &ruleset.rules[0].fault;
        assert_eq!(fault.start_delay_ms, 1000);
        assert_eq!(fault.duration_seconds, 300);
    }

    #[test]
    fn test_parse_empty_ruleset() {
        let json_str = r#"{
            "version": "1.0",
            "rules": []
        }"#;

        let ruleset: CompiledRuleSet = serde_json::from_str(json_str).unwrap();
        assert_eq!(ruleset.rules.len(), 0);
    }

    #[test]
    fn test_parse_invalid_json() {
        let json_str = r#"{ invalid json }"#;
        let result = serde_json::from_str::<CompiledRuleSet>(json_str);
        assert!(result.is_err());
    }

    #[test]
    fn test_parse_path_exact_matcher() {
        let json_str = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "exact-path-rule",
                    "match": {"path": {"exact": "/api/v1/users"}},
                    "fault": {"percentage": 50, "abort": {"httpStatus": 500}}
                }
            ]
        }"#;

        let ruleset: CompiledRuleSet = serde_json::from_str(json_str).unwrap();
        let path = &ruleset.rules[0].match_condition.path;
        assert!(path.is_some());
        assert_eq!(
            path.as_ref().unwrap().exact.as_ref().unwrap(),
            "/api/v1/users"
        );
    }

    #[test]
    fn test_parse_path_prefix_matcher() {
        let json_str = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "prefix-path-rule",
                    "match": {"path": {"prefix": "/api/v1"}},
                    "fault": {"percentage": 75, "abort": {"httpStatus": 503}}
                }
            ]
        }"#;

        let ruleset: CompiledRuleSet = serde_json::from_str(json_str).unwrap();
        let path = &ruleset.rules[0].match_condition.path;
        assert!(path.is_some());
        assert_eq!(path.as_ref().unwrap().prefix.as_ref().unwrap(), "/api/v1");
    }

    #[test]
    fn test_parse_regex_matcher() {
        let json_str = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "regex-rule",
                    "match": {"path": {"regex": "^/api/v[0-9]+/.*"}},
                    "fault": {"percentage": 100, "abort": {"httpStatus": 500}}
                }
            ]
        }"#;

        let ruleset: CompiledRuleSet = serde_json::from_str(json_str).unwrap();
        let path = &ruleset.rules[0].match_condition.path;
        assert!(path.is_some());
        assert_eq!(
            path.as_ref().unwrap().regex.as_ref().unwrap(),
            "^/api/v[0-9]+/.*"
        );
    }

    #[test]
    fn test_parse_abort_with_body() {
        let json_str = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "abort-with-body-rule",
                    "match": {"path": {"exact": "/api/test"}},
                    "fault": {
                        "percentage": 100,
                        "abort": {
                            "httpStatus": 503,
                            "body": "Service Temporarily Unavailable"
                        }
                    }
                }
            ]
        }"#;

        let ruleset: CompiledRuleSet = serde_json::from_str(json_str).unwrap();
        let abort = &ruleset.rules[0].fault.abort;
        assert!(abort.is_some());
        assert_eq!(abort.as_ref().unwrap().http_status, 503);
        assert_eq!(
            abort.as_ref().unwrap().body.as_ref().unwrap(),
            "Service Temporarily Unavailable"
        );
    }

    #[test]
    fn test_parse_method_matcher() {
        let json_str = r#"{
            "version": "1.0",
            "rules": [
                {
                    "name": "method-rule",
                    "match": {"method": {"exact": "POST"}},
                    "fault": {"percentage": 100, "abort": {"httpStatus": 400}}
                }
            ]
        }"#;

        let ruleset: CompiledRuleSet = serde_json::from_str(json_str).unwrap();
        let method = &ruleset.rules[0].match_condition.method;
        assert!(method.is_some());
        assert_eq!(method.as_ref().unwrap().exact.as_ref().unwrap(), "POST");
    }

    #[test]
    fn test_parse_complex_policy() {
        let json_str = r#"{
            "version": "3.0",
            "rules": [
                {
                    "name": "complex-rule",
                    "match": {
                        "path": {"regex": "^/api/.*"},
                        "method": {"exact": "GET"},
                        "headers": [
                            {"name": "User-Agent", "prefix": "curl/"}
                        ]
                    },
                    "fault": {
                        "percentage": 50,
                        "start_delay_ms": 500,
                        "duration_seconds": 120,
                        "abort": {
                            "httpStatus": 503,
                            "body": "Service Temporarily Unavailable"
                        }
                    }
                }
            ]
        }"#;

        let ruleset: CompiledRuleSet = serde_json::from_str(json_str).unwrap();
        let rule = &ruleset.rules[0];

        assert_eq!(rule.name, "complex-rule");
        assert!(rule.match_condition.path.is_some());
        assert!(rule.match_condition.method.is_some());
        assert!(rule.match_condition.headers.is_some());

        let fault = &rule.fault;
        assert_eq!(fault.percentage, 50);
        assert_eq!(fault.start_delay_ms, 500);
        assert_eq!(fault.duration_seconds, 120);
        assert!(fault.abort.is_some());
        assert_eq!(fault.abort.as_ref().unwrap().http_status, 503);
    }

    #[test]
    fn test_parse_large_ruleset() {
        let mut rules = String::new();
        for i in 0..50 {
            if i > 0 {
                rules.push(',');
            }
            rules.push_str(&format!(
                r#"{{
                "name": "rule-{}",
                "match": {{"path": {{"exact": "/api/endpoint{}"}}}},
                "fault": {{"percentage": {}, "abort": {{"httpStatus": 500}}}}
            }}"#,
                i,
                i,
                (i % 100) as u32
            ));
        }

        let json_str = format!(
            r#"{{
            "version": "1.0",
            "rules": [{}]
        }}"#,
            rules
        );

        let ruleset: CompiledRuleSet = serde_json::from_str(&json_str).unwrap();
        assert_eq!(ruleset.rules.len(), 50);

        // Verify a few rules
        assert_eq!(ruleset.rules[0].name, "rule-0");
        assert_eq!(ruleset.rules[25].name, "rule-25");
        assert_eq!(ruleset.rules[49].name, "rule-49");
    }
}
