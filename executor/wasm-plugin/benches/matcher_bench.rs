use criterion::{black_box, criterion_group, criterion_main, BenchmarkId, Criterion};
use regex::Regex;

/// Simplified matcher for benchmarking purposes
pub struct SimpleMatcher {
    path_regex: Option<Regex>,
    methods: Vec<String>,
    headers: Vec<(String, String)>,
}

impl SimpleMatcher {
    pub fn new(
        path_regex: Option<&str>,
        methods: Vec<&str>,
        headers: Vec<(&str, &str)>,
    ) -> Result<Self, regex::Error> {
        let path_regex = path_regex.map(|p| Regex::new(p)).transpose()?;

        Ok(SimpleMatcher {
            path_regex,
            methods: methods.into_iter().map(|s| s.to_string()).collect(),
            headers: headers
                .into_iter()
                .map(|(k, v)| (k.to_string(), v.to_string()))
                .collect(),
        })
    }

    pub fn matches(&self, path: &str, method: &str, req_headers: &[(String, String)]) -> bool {
        // Path matching
        if let Some(ref regex) = self.path_regex {
            if !regex.is_match(path) {
                return false;
            }
        }

        // Method matching
        if !self.methods.is_empty() && !self.methods.contains(&method.to_string()) {
            return false;
        }

        // Header matching
        for (key, value) in &self.headers {
            if !req_headers.iter().any(|(k, v)| k == key && v == value) {
                return false;
            }
        }

        true
    }
}

fn benchmark_single_rule(c: &mut Criterion) {
    let matcher = SimpleMatcher::new(
        Some("^/api/.*"),
        vec!["GET", "POST"],
        vec![("Content-Type", "application/json")],
    )
    .unwrap();

    c.bench_function("single_rule_match", |b| {
        b.iter(|| {
            matcher.matches(
                black_box("/api/users"),
                black_box("GET"),
                black_box(&vec![(
                    "Content-Type".to_string(),
                    "application/json".to_string(),
                )]),
            )
        })
    });

    c.bench_function("single_rule_no_match", |b| {
        b.iter(|| {
            matcher.matches(
                black_box("/other/users"),
                black_box("GET"),
                black_box(&vec![(
                    "Content-Type".to_string(),
                    "application/json".to_string(),
                )]),
            )
        })
    });
}

fn benchmark_multiple_rules(c: &mut Criterion) {
    let mut group = c.benchmark_group("multiple_rules");

    for rule_count in [5, 10, 20].iter() {
        let matchers: Vec<_> = (0..*rule_count)
            .map(|i| {
                SimpleMatcher::new(
                    Some(&format!("^/api/v1/resource{}/.*", i)),
                    vec!["GET", "POST"],
                    vec![("Content-Type", "application/json")],
                )
                .unwrap()
            })
            .collect();

        group.bench_with_input(
            BenchmarkId::from_parameter(*rule_count),
            rule_count,
            |b, _| {
                b.iter(|| {
                    // Try to match against multiple rules
                    for matcher in &matchers {
                        matcher.matches(
                            black_box("/api/v1/resource5/items"),
                            black_box("GET"),
                            black_box(&vec![(
                                "Content-Type".to_string(),
                                "application/json".to_string(),
                            )]),
                        );
                    }
                })
            },
        );
    }
    group.finish();
}

fn benchmark_regex_patterns(c: &mut Criterion) {
    let patterns = vec![
        ("simple_prefix", "^/api/.*"),
        ("complex_path", "^/api/v[0-9]+/(users|products|orders)/.*"),
        ("with_params", "^/api/[a-z]+/[0-9]+/.*"),
    ];

    let mut group = c.benchmark_group("regex_patterns");

    for (name, pattern) in patterns {
        let matcher = SimpleMatcher::new(Some(pattern), vec!["GET"], vec![]).unwrap();

        group.bench_with_input(
            BenchmarkId::from_parameter(name),
            &(name, pattern),
            |b, _| {
                b.iter(|| {
                    matcher.matches(
                        black_box("/api/v2/users/12345/profile"),
                        black_box("GET"),
                        black_box(&vec![]),
                    )
                })
            },
        );
    }
    group.finish();
}

fn benchmark_header_matching(c: &mut Criterion) {
    let mut group = c.benchmark_group("header_matching");

    for header_count in [1, 3, 5].iter() {
        let headers: Vec<_> = (0..*header_count)
            .map(|i| (format!("X-Custom-Header-{}", i), format!("value-{}", i)))
            .collect();

        let headers_ref: Vec<_> = headers
            .iter()
            .map(|(k, v)| (k.as_str(), v.as_str()))
            .collect();

        let matcher = SimpleMatcher::new(Some("^/api/.*"), vec!["GET"], headers_ref).unwrap();

        let request_headers: Vec<_> = headers
            .iter()
            .map(|(k, v)| (k.clone(), v.clone()))
            .collect();

        group.bench_with_input(
            BenchmarkId::from_parameter(*header_count),
            header_count,
            |b, _| {
                b.iter(|| {
                    matcher.matches(
                        black_box("/api/resource"),
                        black_box("GET"),
                        black_box(&request_headers),
                    )
                })
            },
        );
    }
    group.finish();
}

criterion_group!(
    benches,
    benchmark_single_rule,
    benchmark_multiple_rules,
    benchmark_regex_patterns,
    benchmark_header_matching
);
criterion_main!(benches);
