use crate::config::{CompiledRule, MatchCondition, PathMatcher, StringMatcher, HeaderMatcher};
use proxy_wasm::traits::HttpContext;
use std::collections::HashMap;

/// Extracted request information for efficient matching
#[derive(Debug)]
pub struct RequestInfo {
    pub path: String,
    pub method: String,
    pub headers: HashMap<String, String>,
}

impl RequestInfo {
    /// Extract request information from HttpContext
    pub fn from_http_context(http_context: &dyn HttpContext) -> Self {
        let path = http_context
            .get_http_request_header(":path")
            .unwrap_or_else(|| "/".to_string());
        
        let method = http_context
            .get_http_request_header(":method")
            .unwrap_or_else(|| "GET".to_string());
        
        // Extract all headers for matching
        let mut headers = HashMap::new();
        
        // Note: In a real implementation, we would need to iterate through all headers
        // Since proxy-wasm doesn't provide a direct way to get all headers,
        // we'll extract common headers that might be used for matching
        let common_headers = [
            "host", "user-agent", "accept", "accept-language", "accept-encoding",
            "authorization", "content-type", "content-length", "x-forwarded-for",
            "x-real-ip", "x-user-id", "x-tenant-id", "x-service", "x-version"
        ];
        
        for header_name in &common_headers {
            if let Some(value) = http_context.get_http_request_header(header_name) {
                headers.insert(header_name.to_string(), value);
            }
        }
        
        RequestInfo {
            path,
            method,
            headers,
        }
    }
}

/// Find the first matching rule for the given request
pub fn find_first_match<'a>(
    request_info: &RequestInfo,
    rules: &'a [CompiledRule],
) -> Option<&'a CompiledRule> {
    for rule in rules {
        if is_match(request_info, &rule.match_condition) {
            log::debug!("Request matched rule: {}", rule.name);
            return Some(rule);
        }
    }
    None
}

/// Check if request matches the given condition
fn is_match(request_info: &RequestInfo, condition: &MatchCondition) -> bool {
    // Path matching (if specified)
    if let Some(ref path_matcher) = condition.path {
        if !match_path(&request_info.path, path_matcher) {
            return false;
        }
    }
    
    // Method matching (if specified)
    if let Some(ref method_matcher) = condition.method {
        if !match_string(&request_info.method, method_matcher) {
            return false;
        }
    }
    
    // Header matching (all headers must match if specified)
    if let Some(ref header_matchers) = condition.headers {
        for header_matcher in header_matchers {
            if !match_header(&request_info.headers, header_matcher) {
                return false;
            }
        }
    }
    
    true
}

/// Match request path against path matcher
fn match_path(path: &str, matcher: &PathMatcher) -> bool {
    // Check exact match
    if let Some(ref exact) = matcher.exact {
        return path == exact;
    }
    
    // Check prefix match
    if let Some(ref prefix) = matcher.prefix {
        return path.starts_with(prefix);
    }
    
    // Check regex match
    if let Some(ref regex) = matcher.compiled_regex {
        return regex.is_match(path);
    }
    
    // If no conditions specified, match everything
    true
}

/// Match string against string matcher
fn match_string(value: &str, matcher: &StringMatcher) -> bool {
    // Check exact match
    if let Some(ref exact) = matcher.exact {
        return value == exact;
    }
    
    // Check prefix match
    if let Some(ref prefix) = matcher.prefix {
        return value.starts_with(prefix);
    }
    
    // Check regex match
    if let Some(ref regex) = matcher.compiled_regex {
        return regex.is_match(value);
    }
    
    // If no conditions specified, match everything
    true
}

/// Match header against header matcher
fn match_header(headers: &HashMap<String, String>, matcher: &HeaderMatcher) -> bool {
    // Get header value (case-insensitive lookup)
    let header_value = headers
        .iter()
        .find(|(k, _)| k.to_lowercase() == matcher.name.to_lowercase())
        .map(|(_, v)| v.as_str());
    
    let Some(value) = header_value else {
        // Header not present - this is a non-match
        return false;
    };
    
    // Check exact match
    if let Some(ref exact) = matcher.exact {
        return value == exact;
    }
    
    // Check prefix match
    if let Some(ref prefix) = matcher.prefix {
        return value.starts_with(prefix);
    }
    
    // Check regex match
    if let Some(ref regex) = matcher.compiled_regex {
        return regex.is_match(value);
    }
    
    // If no conditions specified but header exists, match
    true
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::{Fault, AbortAction};
    use std::collections::HashMap;

    // Simple test helper that doesn't depend on proxy-wasm
    fn create_request_info(path: &str, method: &str, headers: Vec<(&str, &str)>) -> RequestInfo {
        let mut header_map = HashMap::new();
        for (k, v) in headers {
            header_map.insert(k.to_string(), v.to_string());
        }
        
        RequestInfo {
            path: path.to_string(),
            method: method.to_string(),
            headers: header_map,
        }
    }

    fn create_test_rule(name: &str, condition: MatchCondition) -> CompiledRule {
        CompiledRule {
            name: name.to_string(),
            match_condition: condition,
            fault: Fault {
                abort: Some(AbortAction {
                    http_status: 503,
                    body: Some("Test error".to_string()),
                }),
                delay: None,
                percentage: 100,
            },
        }
    }

    #[test]
    fn test_exact_path_match() {
        let request_info = create_request_info("/api/users", "GET", vec![]);
        
        let condition = MatchCondition {
            path: Some(PathMatcher {
                exact: Some("/api/users".to_string()),
                prefix: None,
                regex: None,
                compiled_regex: None,
            }),
            method: None,
            headers: None,
        };
        
        let rule = create_test_rule("exact-path", condition);
        let rules = vec![rule];
        
        assert!(find_first_match(&request_info, &rules).is_some());
        
        // Test non-matching path
        let request_info = create_request_info("/api/posts", "GET", vec![]);
        assert!(find_first_match(&request_info, &rules).is_none());
    }

    #[test]
    fn test_prefix_path_match() {
        let request_info = create_request_info("/api/users/123", "GET", vec![]);
        
        let condition = MatchCondition {
            path: Some(PathMatcher {
                exact: None,
                prefix: Some("/api/users".to_string()),
                regex: None,
                compiled_regex: None,
            }),
            method: None,
            headers: None,
        };
        
        let rule = create_test_rule("prefix-path", condition);
        let rules = vec![rule];
        
        assert!(find_first_match(&request_info, &rules).is_some());
        
        // Test non-matching path
        let request_info = create_request_info("/api/posts", "GET", vec![]);
        assert!(find_first_match(&request_info, &rules).is_none());
    }

    #[test]
    fn test_regex_path_match() {
        use regex::Regex;
        
        let request_info = create_request_info("/api/users/123", "GET", vec![]);
        
        let condition = MatchCondition {
            path: Some(PathMatcher {
                exact: None,
                prefix: None,
                regex: Some(r"^/api/users/\d+$".to_string()),
                compiled_regex: Some(Regex::new(r"^/api/users/\d+$").unwrap()),
            }),
            method: None,
            headers: None,
        };
        
        let rule = create_test_rule("regex-path", condition);
        let rules = vec![rule];
        
        assert!(find_first_match(&request_info, &rules).is_some());
        
        // Test non-matching path
        let request_info = create_request_info("/api/users/abc", "GET", vec![]);
        assert!(find_first_match(&request_info, &rules).is_none());
    }

    #[test]
    fn test_method_match() {
        let request_info = create_request_info("/api/users", "POST", vec![]);
        
        let condition = MatchCondition {
            path: None,
            method: Some(StringMatcher {
                exact: Some("POST".to_string()),
                prefix: None,
                regex: None,
                compiled_regex: None,
            }),
            headers: None,
        };
        
        let rule = create_test_rule("method-match", condition);
        let rules = vec![rule];
        
        assert!(find_first_match(&request_info, &rules).is_some());
        
        // Test non-matching method
        let request_info = create_request_info("/api/users", "GET", vec![]);
        assert!(find_first_match(&request_info, &rules).is_none());
    }

    #[test]
    fn test_header_match() {
        let request_info = create_request_info("/api/users", "GET", vec![("x-user-id", "12345")]);
        
        let condition = MatchCondition {
            path: None,
            method: None,
            headers: Some(vec![HeaderMatcher {
                name: "x-user-id".to_string(),
                exact: Some("12345".to_string()),
                prefix: None,
                regex: None,
                compiled_regex: None,
            }]),
        };
        
        let rule = create_test_rule("header-match", condition);
        let rules = vec![rule];
        
        assert!(find_first_match(&request_info, &rules).is_some());
        
        // Test non-matching header value
        let request_info = create_request_info("/api/users", "GET", vec![("x-user-id", "67890")]);
        assert!(find_first_match(&request_info, &rules).is_none());
    }

    #[test]
    fn test_combined_match() {
        let request_info = create_request_info(
            "/api/users/123", 
            "GET", 
            vec![("x-user-id", "12345")]
        );
        
        let condition = MatchCondition {
            path: Some(PathMatcher {
                exact: None,
                prefix: Some("/api/users".to_string()),
                regex: None,
                compiled_regex: None,
            }),
            method: Some(StringMatcher {
                exact: Some("GET".to_string()),
                prefix: None,
                regex: None,
                compiled_regex: None,
            }),
            headers: Some(vec![HeaderMatcher {
                name: "x-user-id".to_string(),
                exact: Some("12345".to_string()),
                prefix: None,
                regex: None,
                compiled_regex: None,
            }]),
        };
        
        let rule = create_test_rule("combined-match", condition);
        let rules = vec![rule];
        
        assert!(find_first_match(&request_info, &rules).is_some());
        
        // Test failure when one condition doesn't match
        let request_info = create_request_info(
            "/api/posts",  // Different path
            "GET", 
            vec![("x-user-id", "12345")]
        );
        assert!(find_first_match(&request_info, &rules).is_none());
    }

    #[test]
    fn test_multiple_rules_first_match() {
        let request_info = create_request_info("/api/users", "GET", vec![]);
        
        let rule1 = create_test_rule("rule1", MatchCondition {
            path: Some(PathMatcher {
                exact: Some("/api/posts".to_string()),
                prefix: None,
                regex: None,
                compiled_regex: None,
            }),
            method: None,
            headers: None,
        });
        
        let rule2 = create_test_rule("rule2", MatchCondition {
            path: Some(PathMatcher {
                exact: Some("/api/users".to_string()),
                prefix: None,
                regex: None,
                compiled_regex: None,
            }),
            method: None,
            headers: None,
        });
        
        let rule3 = create_test_rule("rule3", MatchCondition {
            path: Some(PathMatcher {
                prefix: Some("/api".to_string()),
                exact: None,
                regex: None,
                compiled_regex: None,
            }),
            method: None,
            headers: None,
        });
        
        let rules = vec![rule1, rule2, rule3];
        
        let matched = find_first_match(&request_info, &rules).unwrap();
        assert_eq!(matched.name, "rule2"); // Should match the first applicable rule
    }

    #[test]
    fn test_no_conditions_matches_all() {
        let request_info = create_request_info("/any/path", "GET", vec![]);
        
        let condition = MatchCondition {
            path: None,
            method: None,
            headers: None,
        };
        
        let rule = create_test_rule("match-all", condition);
        let rules = vec![rule];
        
        assert!(find_first_match(&request_info, &rules).is_some());
    }
}
