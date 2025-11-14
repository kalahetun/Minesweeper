// tests/fixtures/policies.rs - Wasm Plugin 测试夹具

/// 预定义的测试 Policy 对象集合

/// 返回一个中止请求的示例策略 JSON
pub fn sample_abort_policy(name: &str) -> String {
    format!(
        r#"{{
            "metadata": {{"name": "{}"}},
            "spec": {{
                "rules": [{{
                    "match": {{
                        "method": "GET",
                        "path": {{"exact": "/payment/checkout"}}
                    }},
                    "fault": {{
                        "abort": {{"percentage": 50, "statusCode": 503}}
                    }}
                }}],
                "start_delay_ms": 0,
                "duration_seconds": 0
            }}
        }}"#,
        name
    )
}

/// 返回一个延迟请求的示例策略 JSON
pub fn sample_delay_policy(name: &str) -> String {
    format!(
        r#"{{
            "metadata": {{"name": "{}"}},
            "spec": {{
                "rules": [{{
                    "match": {{
                        "method": "POST",
                        "path": {{"prefix": "/api/"}}
                    }},
                    "fault": {{
                        "delay": {{"percentage": 100, "delayMs": 200}}
                    }}
                }}],
                "start_delay_ms": 0,
                "duration_seconds": 0
            }}
        }}"#,
        name
    )
}

/// 返回一个带时间控制的示例策略 JSON
pub fn sample_timed_policy(name: &str, duration_seconds: u32) -> String {
    format!(
        r#"{{
            "metadata": {{"name": "{}"}},
            "spec": {{
                "rules": [{{
                    "match": {{
                        "method": "GET",
                        "path": {{"regex": "^/test/.*"}}
                    }},
                    "fault": {{
                        "abort": {{"percentage": 100, "statusCode": 500}}
                    }}
                }}],
                "start_delay_ms": 100,
                "duration_seconds": {}
            }}
        }}"#,
        name, duration_seconds
    )
}

/// 返回一个基于请求头匹配的示例策略 JSON
pub fn sample_header_match_policy(name: &str) -> String {
    format!(
        r#"{{
            "metadata": {{"name": "{}"}},
            "spec": {{
                "rules": [{{
                    "match": {{
                        "headers": [{{"name": "x-chaos-enabled", "exact": "true"}}]
                    }},
                    "fault": {{
                        "delay": {{"percentage": 75, "delayMs": 500}}
                    }}
                }}],
                "start_delay_ms": 0,
                "duration_seconds": 0
            }}
        }}"#,
        name
    )
}

/// 返回一个无效的策略 JSON 用于测试验证
pub fn invalid_policy(reason: &str) -> Option<String> {
    match reason {
        "missing_name" => Some(
            r#"{
                "metadata": {},
                "spec": {"rules": []}
            }"#
            .to_string(),
        ),
        "missing_rules" => Some(
            r#"{
                "metadata": {"name": "test-policy"},
                "spec": {}
            }"#
            .to_string(),
        ),
        "invalid_json" => None,
        _ => Some(r#"{"invalid": "structure"}"#.to_string()),
    }
}

/// 多规则策略，用于测试规则匹配优先级
pub fn multi_rule_policy(name: &str) -> String {
    format!(
        r#"{{
            "metadata": {{"name": "{}"}},
            "spec": {{
                "rules": [
                    {{
                        "match": {{"method": "GET", "path": {{"exact": "/payment"}}}},
                        "fault": {{"abort": {{"percentage": 100, "statusCode": 500}}}}
                    }},
                    {{
                        "match": {{"method": "POST", "path": {{"prefix": "/api/"}}}},
                        "fault": {{"delay": {{"percentage": 100, "delayMs": 200}}}}
                    }},
                    {{
                        "match": {{"headers": [{{"name": "x-debug", "exact": "true"}}]}},
                        "fault": {{"abort": {{"percentage": 50, "statusCode": 403}}}}
                    }}
                ],
                "start_delay_ms": 0,
                "duration_seconds": 0
            }}
        }}"#,
        name
    )
}
