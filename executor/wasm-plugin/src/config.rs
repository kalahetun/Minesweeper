use regex::Regex;
use serde::{Deserialize, Deserializer};

// Re-export ServiceSelector from identity module for convenience
pub use crate::identity::ServiceSelector;

// API 响应结构 - 用于解析 Control Plane 的 /v1/policies 响应
#[derive(Debug, Clone, Deserialize)]
pub struct PoliciesResponse {
    pub policies: Vec<PolicyWrapper>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct PolicyWrapper {
    pub metadata: PolicyMetadata,
    pub spec: PolicySpec,
}

#[derive(Debug, Clone, Deserialize)]
pub struct PolicyMetadata {
    pub name: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct PolicySpec {
    /// Service selector for policy targeting (NEW for multi-pod deployment)
    /// If None or contains wildcards, applies to all services
    #[serde(default)]
    pub selector: Option<ServiceSelector>,
    pub rules: Vec<RuleSpec>,
}

impl PolicySpec {
    /// Returns the effective selector, defaulting to wildcard if not specified.
    pub fn effective_selector(&self) -> ServiceSelector {
        self.selector
            .clone()
            .unwrap_or_else(ServiceSelector::wildcard)
    }
}

#[derive(Debug, Clone, Deserialize)]
pub struct RuleSpec {
    #[serde(rename = "match")]
    pub match_condition: MatchCondition,
    pub fault: Fault,
}

#[derive(Debug, Clone, Deserialize)]
pub struct CompiledRuleSet {
    #[allow(dead_code)]
    pub version: String,
    pub rules: Vec<CompiledRule>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct CompiledRule {
    pub name: String,
    #[serde(rename = "match")]
    pub match_condition: MatchCondition,
    pub fault: Fault,
    /// 规则创建时间戳（毫秒），用于过期检查
    #[serde(skip)]
    pub creation_time_ms: u64,
}

#[derive(Debug, Clone, Deserialize)]
pub struct MatchCondition {
    pub path: Option<PathMatcher>,
    pub method: Option<StringMatcher>,
    pub headers: Option<Vec<HeaderMatcher>>,
}

#[derive(Debug, Clone)]
pub struct PathMatcher {
    pub prefix: Option<String>,
    pub exact: Option<String>,
    pub regex: Option<String>,
    pub compiled_regex: Option<Regex>,
}

#[derive(Debug, Clone)]
pub struct StringMatcher {
    pub exact: Option<String>,
    pub prefix: Option<String>,
    pub regex: Option<String>,
    pub compiled_regex: Option<Regex>,
}

#[derive(Debug, Clone)]
pub struct HeaderMatcher {
    pub name: String,
    pub exact: Option<String>,
    pub prefix: Option<String>,
    pub regex: Option<String>,
    pub compiled_regex: Option<Regex>,
}

#[derive(Debug, Clone, Deserialize)]
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

#[derive(Debug, Clone, Deserialize)]
pub struct AbortAction {
    #[serde(rename = "httpStatus")]
    pub http_status: u32,
    pub body: Option<String>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct DelayAction {
    #[serde(rename = "fixed_delay")]
    pub fixed_delay: String,
    #[serde(skip)]
    pub parsed_duration_ms: Option<u64>,
}

// Custom deserializers for matchers to handle regex compilation
impl<'de> Deserialize<'de> for PathMatcher {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        #[derive(Deserialize)]
        struct PathMatcherHelper {
            prefix: Option<String>,
            exact: Option<String>,
            regex: Option<String>,
        }

        let helper = PathMatcherHelper::deserialize(deserializer)?;
        let compiled_regex = if let Some(ref regex_str) = helper.regex {
            match Regex::new(regex_str) {
                Ok(regex) => Some(regex),
                Err(e) => {
                    log::warn!("Failed to compile regex '{}': {}", regex_str, e);
                    None
                }
            }
        } else {
            None
        };

        Ok(PathMatcher {
            prefix: helper.prefix,
            exact: helper.exact,
            regex: helper.regex,
            compiled_regex,
        })
    }
}

impl<'de> Deserialize<'de> for StringMatcher {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        #[derive(Deserialize)]
        struct StringMatcherHelper {
            exact: Option<String>,
            prefix: Option<String>,
            regex: Option<String>,
        }

        let helper = StringMatcherHelper::deserialize(deserializer)?;
        let compiled_regex = if let Some(ref regex_str) = helper.regex {
            match Regex::new(regex_str) {
                Ok(regex) => Some(regex),
                Err(e) => {
                    log::warn!("Failed to compile regex '{}': {}", regex_str, e);
                    None
                }
            }
        } else {
            None
        };

        Ok(StringMatcher {
            exact: helper.exact,
            prefix: helper.prefix,
            regex: helper.regex,
            compiled_regex,
        })
    }
}

impl<'de> Deserialize<'de> for HeaderMatcher {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        #[derive(Deserialize)]
        struct HeaderMatcherHelper {
            name: String,
            exact: Option<String>,
            prefix: Option<String>,
            regex: Option<String>,
        }

        let helper = HeaderMatcherHelper::deserialize(deserializer)?;
        let compiled_regex = if let Some(ref regex_str) = helper.regex {
            match Regex::new(regex_str) {
                Ok(regex) => Some(regex),
                Err(e) => {
                    log::warn!("Failed to compile regex '{}': {}", regex_str, e);
                    None
                }
            }
        } else {
            None
        };

        Ok(HeaderMatcher {
            name: helper.name,
            exact: helper.exact,
            prefix: helper.prefix,
            regex: helper.regex,
            compiled_regex,
        })
    }
}

impl CompiledRuleSet {
    /// Parse configuration from bytes and precompile all regular expressions and durations
    #[allow(dead_code)]
    pub fn from_slice(bytes: &[u8]) -> Result<Self, serde_json::Error> {
        let mut ruleset: CompiledRuleSet = serde_json::from_slice(bytes)?;

        // Pre-process delay durations for each rule
        for rule in &mut ruleset.rules {
            if let Some(ref mut delay) = rule.fault.delay {
                delay.parsed_duration_ms = parse_duration(&delay.fixed_delay);
                if delay.parsed_duration_ms.is_some() {
                    log::debug!(
                        "Parsed delay '{}' to {}ms for rule '{}'",
                        delay.fixed_delay,
                        delay.parsed_duration_ms.unwrap(),
                        rule.name
                    );
                }
            }
        }

        Ok(ruleset)
    }

    /// Create CompiledRuleSet from Control Plane API response
    /// Filters policies based on the provided service identity.
    ///
    /// Fail-open behavior:
    /// - If identity is None or invalid, only wildcard policies are applied
    /// - If filtering fails for any reason, policies are still loaded
    pub fn from_policies_response(
        bytes: &[u8],
        identity: Option<&crate::identity::EnvoyIdentity>,
    ) -> Result<Self, serde_json::Error> {
        let response: PoliciesResponse = serde_json::from_slice(bytes)?;

        // 获取当前时间戳用于记录规则创建时间
        let current_time_ms = crate::time_control::get_current_time_ms();

        let mut rules = Vec::new();
        let mut filtered_count = 0;
        let mut total_count = 0;
        let mut wildcard_count = 0;

        // Determine if we're in fail-open mode (no valid identity)
        let fail_open_mode = match identity {
            Some(id) => !id.is_valid(),
            None => true,
        };

        if fail_open_mode {
            log::warn!(
                "Policy filtering in fail-open mode: only wildcard policies will be applied"
            );
        }

        for policy in response.policies {
            total_count += 1;

            // Check if this policy applies to the current service
            let selector = policy.spec.effective_selector();
            let is_wildcard = selector.service == "*" && selector.namespace == "*";

            if is_wildcard {
                wildcard_count += 1;
            }

            // In fail-open mode, only apply wildcard policies
            let matches = if fail_open_mode {
                is_wildcard
            } else {
                match identity {
                    Some(id) => id.matches_selector(&selector),
                    None => true,
                }
            };

            if !matches {
                log::debug!(
                    "Policy '{}' filtered out: selector {}.{} doesn't match identity{}",
                    policy.metadata.name,
                    selector.service,
                    selector.namespace,
                    if fail_open_mode {
                        " (fail-open mode)"
                    } else {
                        ""
                    }
                );
                filtered_count += 1;
                continue;
            }

            log::debug!(
                "Policy '{}' matches: selector {}.{} applies to this service",
                policy.metadata.name,
                selector.service,
                selector.namespace
            );

            for rule_spec in policy.spec.rules {
                let compiled_rule = CompiledRule {
                    name: policy.metadata.name.clone(),
                    match_condition: rule_spec.match_condition,
                    fault: rule_spec.fault,
                    creation_time_ms: current_time_ms,
                };
                rules.push(compiled_rule);
            }
        }

        if fail_open_mode {
            log::info!(
                "Policy filtering (fail-open): {} total, {} wildcard, {} applicable (non-wildcard policies ignored)",
                total_count,
                wildcard_count,
                total_count - filtered_count
            );
        } else {
            log::info!(
                "Policy filtering: {} total, {} filtered out, {} applicable",
                total_count,
                filtered_count,
                total_count - filtered_count
            );
        }

        let mut ruleset = CompiledRuleSet {
            version: "1.0".to_string(),
            rules,
        };

        // Pre-process delay durations for each rule
        for rule in &mut ruleset.rules {
            if let Some(ref mut delay) = rule.fault.delay {
                delay.parsed_duration_ms = parse_duration(&delay.fixed_delay);
                if delay.parsed_duration_ms.is_some() {
                    log::debug!(
                        "Parsed delay '{}' to {}ms for rule '{}'",
                        delay.fixed_delay,
                        delay.parsed_duration_ms.unwrap(),
                        rule.name
                    );
                }
            }
        }

        Ok(ruleset)
    }
}

/// Parse duration string (e.g., "2s", "100ms") to milliseconds
fn parse_duration(duration_str: &str) -> Option<u64> {
    let duration_str = duration_str.trim().to_lowercase();

    if duration_str.ends_with("ms") {
        if let Ok(ms) = duration_str[..duration_str.len() - 2].parse::<u64>() {
            return Some(ms);
        }
    } else if duration_str.ends_with('s') {
        if let Ok(s) = duration_str[..duration_str.len() - 1].parse::<u64>() {
            return Some(s * 1000);
        }
    } else if duration_str.ends_with('m') {
        if let Ok(m) = duration_str[..duration_str.len() - 1].parse::<u64>() {
            return Some(m * 60 * 1000);
        }
    }

    // Try parsing as plain number (assume milliseconds)
    if let Ok(ms) = duration_str.parse::<u64>() {
        return Some(ms);
    }

    log::warn!("Failed to parse duration: {}", duration_str);
    None
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_duration() {
        assert_eq!(parse_duration("100ms"), Some(100));
        assert_eq!(parse_duration("2s"), Some(2000));
        assert_eq!(parse_duration("1m"), Some(60000));
        assert_eq!(parse_duration("500"), Some(500));
        assert_eq!(parse_duration("invalid"), None);
    }

    #[test]
    fn test_compiled_ruleset_from_json() {
        let json = r#"
        {
            "version": "1.0",
            "rules": [
                {
                    "name": "test-rule",
                    "match": {
                        "path": {
                            "regex": "^/api/.*"
                        },
                        "method": {
                            "exact": "GET"
                        }
                    },
                    "fault": {
                        "abort": {
                            "httpStatus": 503,
                            "body": "Service Unavailable"
                        },
                        "percentage": 100
                    }
                }
            ]
        }
        "#;

        let ruleset = CompiledRuleSet::from_slice(json.as_bytes()).unwrap();
        assert_eq!(ruleset.version, "1.0");
        assert_eq!(ruleset.rules.len(), 1);

        let rule = &ruleset.rules[0];
        assert_eq!(rule.name, "test-rule");

        // Verify regex was compiled
        let path_matcher = rule.match_condition.path.as_ref().unwrap();
        assert!(path_matcher.compiled_regex.is_some());

        // Verify method matcher
        let method_matcher = rule.match_condition.method.as_ref().unwrap();
        assert_eq!(method_matcher.exact.as_ref().unwrap(), "GET");
    }

    #[test]
    fn test_parse_delay_action() {
        let json = r#"
        {
            "version": "1.0",
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
        }
        "#;

        let ruleset = CompiledRuleSet::from_slice(json.as_bytes()).unwrap();
        let rule = &ruleset.rules[0];

        assert_eq!(rule.fault.delay.as_ref().unwrap().fixed_delay, "2s");
        assert_eq!(
            rule.fault.delay.as_ref().unwrap().parsed_duration_ms,
            Some(2000)
        );
    }

    #[test]
    fn test_parse_path_prefix_matcher() {
        let json = r#"
        {
            "version": "1.0",
            "rules": [
                {
                    "name": "prefix-rule",
                    "match": {
                        "path": {
                            "prefix": "/api/v1"
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
        }
        "#;

        let ruleset = CompiledRuleSet::from_slice(json.as_bytes()).unwrap();
        let rule = &ruleset.rules[0];
        let path_matcher = rule.match_condition.path.as_ref().unwrap();

        assert_eq!(path_matcher.prefix.as_ref().unwrap(), "/api/v1");
        assert!(path_matcher.exact.is_none());
    }

    #[test]
    fn test_parse_method_matcher() {
        let json = r#"
        {
            "version": "1.0",
            "rules": [
                {
                    "name": "method-rule",
                    "match": {
                        "method": {
                            "exact": "POST"
                        }
                    },
                    "fault": {
                        "percentage": 75,
                        "abort": {
                            "httpStatus": 503
                        }
                    }
                }
            ]
        }
        "#;

        let ruleset = CompiledRuleSet::from_slice(json.as_bytes()).unwrap();
        let rule = &ruleset.rules[0];

        assert_eq!(
            rule.match_condition
                .method
                .as_ref()
                .unwrap()
                .exact
                .as_ref()
                .unwrap(),
            "POST"
        );
    }

    #[test]
    fn test_parse_header_matchers() {
        let json = r#"
        {
            "version": "1.0",
            "rules": [
                {
                    "name": "header-rule",
                    "match": {
                        "headers": [
                            {
                                "name": "Authorization",
                                "regex": "Bearer .*"
                            },
                            {
                                "name": "Content-Type",
                                "prefix": "application/"
                            }
                        ]
                    },
                    "fault": {
                        "percentage": 100,
                        "abort": {
                            "httpStatus": 401
                        }
                    }
                }
            ]
        }
        "#;

        let ruleset = CompiledRuleSet::from_slice(json.as_bytes()).unwrap();
        let rule = &ruleset.rules[0];

        let headers = rule.match_condition.headers.as_ref().unwrap();
        assert_eq!(headers.len(), 2);
        assert_eq!(headers[0].name, "Authorization");
        assert_eq!(headers[1].name, "Content-Type");
    }

    #[test]
    fn test_parse_multiple_rules() {
        let json = r#"
        {
            "version": "2.0",
            "rules": [
                {
                    "name": "rule-1",
                    "match": {
                        "path": {
                            "exact": "/api/users"
                        }
                    },
                    "fault": {
                        "percentage": 50,
                        "abort": {
                            "httpStatus": 500
                        }
                    }
                },
                {
                    "name": "rule-2",
                    "match": {
                        "path": {
                            "exact": "/api/orders"
                        }
                    },
                    "fault": {
                        "percentage": 25,
                        "delay": {
                            "fixed_delay": "1s"
                        }
                    }
                }
            ]
        }
        "#;

        let ruleset = CompiledRuleSet::from_slice(json.as_bytes()).unwrap();
        assert_eq!(ruleset.rules.len(), 2);
        assert_eq!(ruleset.version, "2.0");

        assert_eq!(ruleset.rules[0].name, "rule-1");
        assert_eq!(ruleset.rules[1].name, "rule-2");
    }

    #[test]
    fn test_parse_percentage_boundary() {
        let json = r#"
        {
            "version": "1.0",
            "rules": [
                {
                    "name": "pct-0",
                    "match": {"path": {"exact": "/a"}},
                    "fault": {"percentage": 0, "abort": {"httpStatus": 500}}
                },
                {
                    "name": "pct-50",
                    "match": {"path": {"exact": "/b"}},
                    "fault": {"percentage": 50, "abort": {"httpStatus": 500}}
                },
                {
                    "name": "pct-100",
                    "match": {"path": {"exact": "/c"}},
                    "fault": {"percentage": 100, "abort": {"httpStatus": 500}}
                }
            ]
        }
        "#;

        let ruleset = CompiledRuleSet::from_slice(json.as_bytes()).unwrap();
        assert_eq!(ruleset.rules[0].fault.percentage, 0);
        assert_eq!(ruleset.rules[1].fault.percentage, 50);
        assert_eq!(ruleset.rules[2].fault.percentage, 100);
    }

    #[test]
    fn test_parse_timing_controls() {
        let json = r#"
        {
            "version": "1.0",
            "rules": [
                {
                    "name": "timed-rule",
                    "match": {
                        "path": {
                            "exact": "/api/test"
                        }
                    },
                    "fault": {
                        "percentage": 100,
                        "start_delay_ms": 1000,
                        "duration_seconds": 300,
                        "abort": {
                            "httpStatus": 500
                        }
                    }
                }
            ]
        }
        "#;

        let ruleset = CompiledRuleSet::from_slice(json.as_bytes()).unwrap();
        let rule = &ruleset.rules[0];

        assert_eq!(rule.fault.start_delay_ms, 1000);
        assert_eq!(rule.fault.duration_seconds, 300);
    }

    #[test]
    fn test_parse_empty_ruleset() {
        let json = r#"
        {
            "version": "1.0",
            "rules": []
        }
        "#;

        let ruleset = CompiledRuleSet::from_slice(json.as_bytes()).unwrap();
        assert_eq!(ruleset.rules.len(), 0);
        assert_eq!(ruleset.version, "1.0");
    }

    #[test]
    fn test_parse_invalid_json() {
        let json = r#"{ invalid json "#;

        let result = CompiledRuleSet::from_slice(json.as_bytes());
        assert!(result.is_err());
    }

    #[test]
    fn test_parse_regex_compilation() {
        let json = r#"
        {
            "version": "1.0",
            "rules": [
                {
                    "name": "regex-rule",
                    "match": {
                        "path": {
                            "regex": "^/api/v[0-9]+/.*"
                        }
                    },
                    "fault": {
                        "percentage": 100,
                        "abort": {
                            "httpStatus": 500
                        }
                    }
                }
            ]
        }
        "#;

        let ruleset = CompiledRuleSet::from_slice(json.as_bytes()).unwrap();
        let rule = &ruleset.rules[0];
        let path_matcher = rule.match_condition.path.as_ref().unwrap();

        assert!(path_matcher.compiled_regex.is_some());
        assert_eq!(path_matcher.regex.as_ref().unwrap(), "^/api/v[0-9]+/.*");
    }

    #[test]
    fn test_parse_complex_policy() {
        let json = r#"
        {
            "version": "3.0",
            "rules": [
                {
                    "name": "complex-rule",
                    "match": {
                        "path": {
                            "regex": "^/api/.*"
                        },
                        "method": {
                            "exact": "GET"
                        },
                        "headers": [
                            {
                                "name": "User-Agent",
                                "prefix": "curl/"
                            }
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
        }
        "#;

        let ruleset = CompiledRuleSet::from_slice(json.as_bytes()).unwrap();
        let rule = &ruleset.rules[0];

        // Verify all components parsed
        assert_eq!(rule.name, "complex-rule");
        assert!(rule.match_condition.path.is_some());
        assert!(rule.match_condition.method.is_some());
        assert!(rule.match_condition.headers.is_some());
        assert_eq!(rule.fault.percentage, 50);
        assert!(rule.fault.abort.is_some());
        assert_eq!(rule.fault.start_delay_ms, 500);
        assert_eq!(rule.fault.duration_seconds, 120);
    }
}
