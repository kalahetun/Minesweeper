// Matcher unit tests for path, method, and header matching with edge cases
// Tests cover prefix, exact, and regex matching patterns

#[cfg(test)]
mod matcher_tests {
    use regex::Regex;

    // Minimal matcher implementations for testing
    #[derive(Debug, Clone)]
    struct PathMatcher {
        prefix: Option<String>,
        exact: Option<String>,
        regex: Option<String>,
        compiled_regex: Option<Regex>,
    }

    #[derive(Debug, Clone)]
    struct StringMatcher {
        exact: Option<String>,
        prefix: Option<String>,
        regex: Option<String>,
        compiled_regex: Option<Regex>,
    }

    #[derive(Debug, Clone)]
    struct HeaderMatcher {
        name: String,
        exact: Option<String>,
        prefix: Option<String>,
        regex: Option<String>,
        compiled_regex: Option<Regex>,
    }

    impl PathMatcher {
        fn matches(&self, path: &str) -> bool {
            if let Some(ref exact) = self.exact {
                return path == exact;
            }
            if let Some(ref prefix) = self.prefix {
                return path.starts_with(prefix);
            }
            if let Some(ref compiled_regex) = self.compiled_regex {
                return compiled_regex.is_match(path);
            }
            if let Some(ref regex_pattern) = self.regex {
                if let Ok(re) = Regex::new(regex_pattern) {
                    return re.is_match(path);
                }
            }
            false
        }

        fn new() -> Self {
            PathMatcher {
                prefix: None,
                exact: None,
                regex: None,
                compiled_regex: None,
            }
        }

        fn with_exact(mut self, exact: String) -> Self {
            self.exact = Some(exact);
            self
        }

        fn with_prefix(mut self, prefix: String) -> Self {
            self.prefix = Some(prefix);
            self
        }

        fn with_regex(mut self, regex: String) -> Self {
            if let Ok(compiled) = Regex::new(&regex) {
                self.compiled_regex = Some(compiled);
            }
            self.regex = Some(regex);
            self
        }
    }

    impl StringMatcher {
        fn matches(&self, value: &str) -> bool {
            if let Some(ref exact) = self.exact {
                return value == exact;
            }
            if let Some(ref prefix) = self.prefix {
                return value.starts_with(prefix);
            }
            if let Some(ref compiled_regex) = self.compiled_regex {
                return compiled_regex.is_match(value);
            }
            if let Some(ref regex_pattern) = self.regex {
                if let Ok(re) = Regex::new(regex_pattern) {
                    return re.is_match(value);
                }
            }
            false
        }

        fn new() -> Self {
            StringMatcher {
                exact: None,
                prefix: None,
                regex: None,
                compiled_regex: None,
            }
        }

        fn with_exact(mut self, exact: String) -> Self {
            self.exact = Some(exact);
            self
        }

        fn with_prefix(mut self, prefix: String) -> Self {
            self.prefix = Some(prefix);
            self
        }

        fn with_regex(mut self, regex: String) -> Self {
            if let Ok(compiled) = Regex::new(&regex) {
                self.compiled_regex = Some(compiled);
            }
            self.regex = Some(regex);
            self
        }
    }

    impl HeaderMatcher {
        fn matches(&self, value: &str) -> bool {
            if let Some(ref exact) = self.exact {
                return value == exact;
            }
            if let Some(ref prefix) = self.prefix {
                return value.starts_with(prefix);
            }
            if let Some(ref compiled_regex) = self.compiled_regex {
                return compiled_regex.is_match(value);
            }
            if let Some(ref regex_pattern) = self.regex {
                if let Ok(re) = Regex::new(regex_pattern) {
                    return re.is_match(value);
                }
            }
            false
        }

        fn new(name: String) -> Self {
            HeaderMatcher {
                name,
                exact: None,
                prefix: None,
                regex: None,
                compiled_regex: None,
            }
        }

        fn with_exact(mut self, exact: String) -> Self {
            self.exact = Some(exact);
            self
        }

        fn with_prefix(mut self, prefix: String) -> Self {
            self.prefix = Some(prefix);
            self
        }

        fn with_regex(mut self, regex: String) -> Self {
            if let Ok(compiled) = Regex::new(&regex) {
                self.compiled_regex = Some(compiled);
            }
            self.regex = Some(regex);
            self
        }
    }

    // ============ PATH MATCHER TESTS ============

    #[test]
    fn test_path_exact_match() {
        let matcher = PathMatcher::new().with_exact("/payment/checkout".to_string());
        assert!(matcher.matches("/payment/checkout"));
        assert!(!matcher.matches("/payment/checkout/"));
        assert!(!matcher.matches("/payment/check out"));
    }

    #[test]
    fn test_path_exact_case_sensitive() {
        let matcher = PathMatcher::new().with_exact("/Admin/Panel".to_string());
        assert!(matcher.matches("/Admin/Panel"));
        assert!(!matcher.matches("/admin/panel"));
        assert!(!matcher.matches("/admin/Panel"));
    }

    #[test]
    fn test_path_exact_empty_string() {
        let matcher = PathMatcher::new().with_exact("".to_string());
        assert!(matcher.matches(""));
        assert!(!matcher.matches("/"));
    }

    #[test]
    fn test_path_prefix_basic() {
        let matcher = PathMatcher::new().with_prefix("/api/".to_string());
        assert!(matcher.matches("/api/"));
        assert!(matcher.matches("/api/users"));
        assert!(matcher.matches("/api/users/123"));
        assert!(!matcher.matches("/api"));
        assert!(!matcher.matches("/ap"));
    }

    #[test]
    fn test_path_prefix_no_slash() {
        let matcher = PathMatcher::new().with_prefix("api".to_string());
        assert!(matcher.matches("api"));
        assert!(matcher.matches("api/users"));
        assert!(matcher.matches("apiv1/test"));
        assert!(!matcher.matches("/api"));
    }

    #[test]
    fn test_path_prefix_root_slash() {
        let matcher = PathMatcher::new().with_prefix("/".to_string());
        assert!(matcher.matches("/"));
        assert!(matcher.matches("/anything"));
        assert!(matcher.matches("/"));
    }

    #[test]
    fn test_path_regex_simple() {
        let matcher = PathMatcher::new().with_regex("^/payment/.*".to_string());
        assert!(matcher.matches("/payment/checkout"));
        assert!(matcher.matches("/payment/validate"));
        assert!(matcher.matches("/payment/"));
        assert!(!matcher.matches("/Payment/checkout"));
        assert!(!matcher.matches("payment/checkout"));
    }

    #[test]
    fn test_path_regex_complex() {
        let matcher = PathMatcher::new().with_regex("/api/v[0-9]+/.*".to_string());
        assert!(matcher.matches("/api/v1/users"));
        assert!(matcher.matches("/api/v2/products"));
        assert!(matcher.matches("/api/v123/test"));
        assert!(!matcher.matches("/api/v/users"));
        assert!(!matcher.matches("/api/va/users"));
    }

    #[test]
    fn test_path_regex_word_boundaries() {
        let matcher = PathMatcher::new().with_regex(r"^/\w+/\w+$".to_string());
        assert!(matcher.matches("/users/123"));
        assert!(matcher.matches("/api/test"));
        assert!(!matcher.matches("/api/test/extra"));
        assert!(!matcher.matches("/api/"));
    }

    #[test]
    fn test_path_regex_optional_trailing_slash() {
        let matcher = PathMatcher::new().with_regex("^/admin/?$".to_string());
        assert!(matcher.matches("/admin"));
        assert!(matcher.matches("/admin/"));
        assert!(!matcher.matches("/admin/users"));
    }

    // ============ METHOD MATCHER TESTS ============

    #[test]
    fn test_method_exact_match() {
        let matcher = StringMatcher::new().with_exact("POST".to_string());
        assert!(matcher.matches("POST"));
        assert!(!matcher.matches("GET"));
        assert!(!matcher.matches("Post"));
        assert!(!matcher.matches("post"));
    }

    #[test]
    fn test_method_prefix_unusual() {
        let matcher = StringMatcher::new().with_prefix("P".to_string());
        assert!(matcher.matches("POST"));
        assert!(matcher.matches("PUT"));
        assert!(matcher.matches("PATCH"));
        assert!(!matcher.matches("GET"));
    }

    #[test]
    fn test_method_regex_all_methods() {
        let matcher = StringMatcher::new().with_regex("^(GET|POST|PUT|DELETE|PATCH)$".to_string());
        assert!(matcher.matches("GET"));
        assert!(matcher.matches("POST"));
        assert!(matcher.matches("PUT"));
        assert!(matcher.matches("DELETE"));
        assert!(matcher.matches("PATCH"));
        assert!(!matcher.matches("GETT"));
        assert!(!matcher.matches("get"));
    }

    #[test]
    fn test_method_empty_string() {
        let matcher = StringMatcher::new().with_exact("".to_string());
        assert!(matcher.matches(""));
        assert!(!matcher.matches("GET"));
    }

    // ============ HEADER MATCHER TESTS ============

    #[test]
    fn test_header_exact_value() {
        let matcher = HeaderMatcher::new("X-Chaos-Test".to_string())
            .with_exact("true".to_string());
        assert!(matcher.matches("true"));
        assert!(!matcher.matches("True"));
        assert!(!matcher.matches("false"));
    }

    #[test]
    fn test_header_prefix_match() {
        let matcher = HeaderMatcher::new("User-Agent".to_string())
            .with_prefix("Mozilla".to_string());
        assert!(matcher.matches("Mozilla/5.0"));
        assert!(matcher.matches("Mozilla"));
        assert!(!matcher.matches("Safari"));
        assert!(!matcher.matches("mozilla/5.0"));
    }

    #[test]
    fn test_header_content_type_regex() {
        let matcher = HeaderMatcher::new("Content-Type".to_string())
            .with_regex("^application/(json|xml).*".to_string());
        assert!(matcher.matches("application/json"));
        assert!(matcher.matches("application/json; charset=utf-8"));
        assert!(matcher.matches("application/xml"));
        assert!(!matcher.matches("text/html"));
        assert!(!matcher.matches("application/form-data"));
    }

    #[test]
    fn test_header_authorization_bearer_token() {
        let matcher = HeaderMatcher::new("Authorization".to_string())
            .with_regex("^Bearer .+$".to_string());
        assert!(matcher.matches("Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"));
        assert!(matcher.matches("Bearer token123"));
        assert!(!matcher.matches("Bearer"));
        assert!(!matcher.matches("Basic dXNlcjpwYXNz"));
    }

    #[test]
    fn test_header_accept_language() {
        let matcher = HeaderMatcher::new("Accept-Language".to_string())
            .with_regex("(en|zh)(-CN)?".to_string());
        assert!(matcher.matches("en"));
        assert!(matcher.matches("en-US"));
        assert!(matcher.matches("zh"));
        assert!(matcher.matches("zh-CN"));
        assert!(matcher.matches("Accept-Language: zh-CN")); // Contains the pattern
        assert!(!matcher.matches("fr"));
    }

    #[test]
    fn test_header_name_is_stored() {
        let matcher = HeaderMatcher::new("X-Custom-Header".to_string());
        assert_eq!(matcher.name, "X-Custom-Header");
    }

    // ============ COMBINED MATCHING TESTS ============

    #[test]
    fn test_multiple_path_conditions_only_one_matches() {
        let m1 = PathMatcher::new().with_exact("/exact".to_string());
        let m2 = PathMatcher::new().with_prefix("/exact".to_string());
        let m3 = PathMatcher::new().with_regex("^/exact$".to_string());

        assert!(m1.matches("/exact"));
        assert!(m2.matches("/exact"));
        assert!(m3.matches("/exact"));

        assert!(!m1.matches("/exact/extra"));
        assert!(m2.matches("/exact/extra"));
        assert!(!m3.matches("/exact/extra"));
    }

    #[test]
    fn test_all_header_types_in_one_set() {
        let h1 = HeaderMatcher::new("X-Exact".to_string())
            .with_exact("value".to_string());
        let h2 = HeaderMatcher::new("X-Prefix".to_string())
            .with_prefix("pre".to_string());
        let h3 = HeaderMatcher::new("X-Regex".to_string())
            .with_regex("^[0-9]+$".to_string());

        assert!(h1.matches("value"));
        assert!(h2.matches("prefix-value"));
        assert!(h3.matches("12345"));
    }

    // ============ EDGE CASES ============

    #[test]
    fn test_path_special_characters() {
        let matcher = PathMatcher::new()
            .with_exact("/api/users?id=123&name=test".to_string());
        assert!(matcher.matches("/api/users?id=123&name=test"));
        assert!(!matcher.matches("/api/users?id=123"));
    }

    #[test]
    fn test_path_unicode_characters() {
        let matcher = PathMatcher::new().with_exact("/users/李明".to_string());
        assert!(matcher.matches("/users/李明"));
        assert!(!matcher.matches("/users/李明/"));
    }

    #[test]
    fn test_path_encoded_characters() {
        let matcher = PathMatcher::new()
            .with_exact("/search?q=hello%20world".to_string());
        assert!(matcher.matches("/search?q=hello%20world"));
        assert!(!matcher.matches("/search?q=hello world"));
    }

    #[test]
    fn test_path_dot_in_regex() {
        let matcher = PathMatcher::new().with_regex(r"^/file\..+$".to_string());
        assert!(matcher.matches("/file.txt"));
        assert!(matcher.matches("/file.json"));
        assert!(!matcher.matches("/file-txt"));
    }

    #[test]
    fn test_very_long_path() {
        let long_path = "/".to_string() + &"segment/".repeat(100);
        let matcher = PathMatcher::new().with_prefix("/".to_string());
        assert!(matcher.matches(&long_path));

        let exact_matcher = PathMatcher::new().with_exact(long_path.clone());
        assert!(exact_matcher.matches(&long_path));
    }

    #[test]
    fn test_regex_special_characters_escaped() {
        let matcher = PathMatcher::new()
            .with_regex(r"^/api/v1\.0.*$".to_string());
        assert!(matcher.matches("/api/v1.0"));
        assert!(matcher.matches("/api/v1.0/users"));
        assert!(!matcher.matches("/api/v1a0"));
    }

    #[test]
    fn test_empty_regex_pattern() {
        let matcher = PathMatcher::new().with_regex("".to_string());
        // Empty regex should match everything
        assert!(matcher.matches("anything"));
        assert!(matcher.matches(""));
    }

    #[test]
    fn test_header_case_insensitive_name_but_sensitive_value() {
        let matcher = HeaderMatcher::new("X-Custom".to_string())
            .with_exact("Value".to_string());
        // Name is case-insensitive in HTTP, but our matcher stores it as-is
        assert_eq!(matcher.name, "X-Custom");
        // Value matching should be case-sensitive
        assert!(matcher.matches("Value"));
        assert!(!matcher.matches("value"));
    }

    #[test]
    fn test_path_vs_query_string() {
        let matcher = PathMatcher::new().with_exact("/api/search".to_string());
        assert!(matcher.matches("/api/search"));
        assert!(!matcher.matches("/api/search?q=test"));

        let matcher_with_query = PathMatcher::new()
            .with_exact("/api/search?q=test".to_string());
        assert!(matcher_with_query.matches("/api/search?q=test"));
    }

    #[test]
    fn test_no_matcher_configuration_returns_false() {
        let matcher = PathMatcher::new();
        assert!(!matcher.matches("/any/path"));
        assert!(!matcher.matches(""));

        let string_matcher = StringMatcher::new();
        assert!(!string_matcher.matches("anything"));
    }
}
