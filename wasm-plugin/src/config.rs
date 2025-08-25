use serde::{Deserialize, Deserializer};
use regex::Regex;

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
    pub rules: Vec<RuleSpec>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct RuleSpec {
    #[serde(rename = "match")]
    pub match_condition: MatchCondition,
    pub fault: Fault,
}

#[derive(Debug, Clone, Deserialize)]
pub struct CompiledRuleSet {
    pub version: String,
    pub rules: Vec<CompiledRule>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct CompiledRule {
    pub name: String,
    #[serde(rename = "match")]
    pub match_condition: MatchCondition,
    pub fault: Fault,
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
    pub fn from_slice(bytes: &[u8]) -> Result<Self, serde_json::Error> {
        let mut ruleset: CompiledRuleSet = serde_json::from_slice(bytes)?;
        
        // Pre-process delay durations for each rule
        for rule in &mut ruleset.rules {
            if let Some(ref mut delay) = rule.fault.delay {
                delay.parsed_duration_ms = parse_duration(&delay.fixed_delay);
                if delay.parsed_duration_ms.is_some() {
                    log::debug!("Parsed delay '{}' to {}ms for rule '{}'", 
                               delay.fixed_delay, 
                               delay.parsed_duration_ms.unwrap(), 
                               rule.name);
                }
            }
        }
        
        Ok(ruleset)
    }

    /// Create CompiledRuleSet from Control Plane API response
    pub fn from_policies_response(bytes: &[u8]) -> Result<Self, serde_json::Error> {
        let response: PoliciesResponse = serde_json::from_slice(bytes)?;
        
        let mut rules = Vec::new();
        for policy in response.policies {
            for rule_spec in policy.spec.rules {
                let compiled_rule = CompiledRule {
                    name: policy.metadata.name.clone(),
                    match_condition: rule_spec.match_condition,
                    fault: rule_spec.fault,
                };
                rules.push(compiled_rule);
            }
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
                    log::debug!("Parsed delay '{}' to {}ms for rule '{}'", 
                               delay.fixed_delay, 
                               delay.parsed_duration_ms.unwrap(), 
                               rule.name);
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
}
