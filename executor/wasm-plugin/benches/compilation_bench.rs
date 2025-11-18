use criterion::{black_box, criterion_group, criterion_main, Criterion, BenchmarkId};
use regex::Regex;

/// Simulate rule compilation
pub struct RuleCompiler;

impl RuleCompiler {
    /// Compile a single rule pattern
    pub fn compile_rule(pattern: &str) -> Result<CompiledRule, String> {
        let regex = Regex::new(pattern).map_err(|e| format!("Invalid regex: {}", e))?;
        Ok(CompiledRule { regex })
    }

    /// Compile multiple rules
    pub fn compile_rules(patterns: &[&str]) -> Result<Vec<CompiledRule>, String> {
        patterns.iter().map(|p| Self::compile_rule(p)).collect()
    }
}

pub struct CompiledRule {
    regex: Regex,
}

impl CompiledRule {
    pub fn matches(&self, text: &str) -> bool {
        self.regex.is_match(text)
    }
}

fn benchmark_compile_single_rule(c: &mut Criterion) {
    let patterns = vec![
        ("simple", "^/api/.*"),
        ("complex", "^/api/v[0-9]+/(users|products|orders)/[a-zA-Z0-9]+(/.*)?$"),
        ("with_lookahead", "^/api/(?!admin).*"),
    ];

    let mut group = c.benchmark_group("compile_single_rule");
    
    for (name, pattern) in patterns {
        group.bench_with_input(
            BenchmarkId::from_parameter(name),
            &(name, pattern),
            |b, (_, pattern)| {
                b.iter(|| {
                    RuleCompiler::compile_rule(black_box(pattern))
                })
            },
        );
    }
    group.finish();
}

fn benchmark_compile_multiple_rules(c: &mut Criterion) {
    let mut group = c.benchmark_group("compile_multiple_rules");
    
    for rule_count in [10, 50, 100].iter() {
        let patterns: Vec<String> = (0..*rule_count)
            .map(|i| format!("^/api/resource{}/.*", i))
            .collect();
        
        let pattern_refs: Vec<&str> = patterns.iter().map(|s| s.as_str()).collect();

        group.bench_with_input(
            BenchmarkId::from_parameter(*rule_count),
            rule_count,
            |b, _| {
                b.iter(|| {
                    RuleCompiler::compile_rules(black_box(&pattern_refs))
                })
            },
        );
    }
    group.finish();
}

fn benchmark_rule_matching_after_compile(c: &mut Criterion) {
    // Pre-compile rules for matching benchmark
    let patterns = vec![
        "^/api/users/.*",
        "^/api/products/.*",
        "^/api/orders/.*",
        "^/api/payments/.*",
        "^/api/reports/.*",
    ];

    let compiled = RuleCompiler::compile_rules(&patterns)
        .expect("Failed to compile rules");

    c.bench_function("match_after_compile_10_rules", |b| {
        b.iter(|| {
            let matches: Vec<_> = compiled
                .iter()
                .map(|rule| rule.matches(black_box("/api/users/12345/profile")))
                .collect();
            black_box(matches)
        })
    });
}

fn benchmark_incremental_compilation(c: &mut Criterion) {
    // Simulate incremental rule addition
    c.bench_function("incremental_compile_100", |b| {
        b.iter(|| {
            let mut rules = Vec::new();
            for i in 0..100 {
                let pattern = format!("^/api/resource{}/.*", i);
                if let Ok(rule) = RuleCompiler::compile_rule(&pattern) {
                    rules.push(rule);
                }
            }
            black_box(rules.len())
        })
    });

    c.bench_function("bulk_compile_100", |b| {
        b.iter(|| {
            let patterns: Vec<String> = (0..100)
                .map(|i| format!("^/api/resource{}/.*", i))
                .collect();
            let pattern_refs: Vec<&str> = patterns.iter().map(|s| s.as_str()).collect();
            
            let rules = RuleCompiler::compile_rules(&pattern_refs);
            black_box(rules.unwrap().len())
        })
    });
}

criterion_group!(
    benches,
    benchmark_compile_single_rule,
    benchmark_compile_multiple_rules,
    benchmark_rule_matching_after_compile,
    benchmark_incremental_compilation
);
criterion_main!(benches);
