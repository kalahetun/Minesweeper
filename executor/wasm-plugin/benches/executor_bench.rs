use criterion::{black_box, criterion_group, criterion_main, BenchmarkId, Criterion};
use std::time::Duration;

/// Fault execution types for benchmarking
#[derive(Clone, Debug)]
pub enum FaultType {
    Abort { status_code: u32 },
    Delay { milliseconds: u64 },
}

/// Simplified executor for benchmarking purposes
pub struct SimpleExecutor {
    fault: FaultType,
}

impl SimpleExecutor {
    pub fn new(fault: FaultType) -> Self {
        SimpleExecutor { fault }
    }

    /// Simulate fault execution
    pub fn execute(&self) -> ExecutionResult {
        match &self.fault {
            FaultType::Abort { status_code } => {
                // Simulate abort fault execution
                ExecutionResult {
                    executed: true,
                    status_code: Some(*status_code),
                    delay_millis: 0,
                }
            }
            FaultType::Delay { milliseconds } => {
                // Simulate delay fault execution
                ExecutionResult {
                    executed: true,
                    status_code: None,
                    delay_millis: *milliseconds,
                }
            }
        }
    }

    /// Simulate atomic execution (no partial state)
    pub fn execute_atomic(&self) -> ExecutionResult {
        let start = std::time::Instant::now();
        let result = self.execute();
        let _ = start.elapsed(); // Measure but ignore for this benchmark
        result
    }
}

#[derive(Clone, Debug)]
pub struct ExecutionResult {
    pub executed: bool,
    pub status_code: Option<u32>,
    pub delay_millis: u64,
}

fn benchmark_abort_execution(c: &mut Criterion) {
    let executor = SimpleExecutor::new(FaultType::Abort { status_code: 500 });

    c.bench_function("abort_execution", |b| b.iter(|| executor.execute()));

    c.bench_function("abort_execution_atomic", |b| {
        b.iter(|| executor.execute_atomic())
    });
}

fn benchmark_delay_execution(c: &mut Criterion) {
    let mut group = c.benchmark_group("delay_execution");

    for delay_ms in [10, 50, 100, 500].iter() {
        let executor = SimpleExecutor::new(FaultType::Delay {
            milliseconds: *delay_ms,
        });

        group.bench_with_input(BenchmarkId::from_parameter(delay_ms), delay_ms, |b, _| {
            b.iter(|| executor.execute())
        });
    }
    group.finish();
}

fn benchmark_execution_overhead(c: &mut Criterion) {
    let abort_executor = SimpleExecutor::new(FaultType::Abort { status_code: 500 });
    let delay_executor = SimpleExecutor::new(FaultType::Delay { milliseconds: 1 });

    let mut group = c.benchmark_group("execution_overhead");
    group.measurement_time(Duration::from_secs(10));

    group.bench_function("abort_vs_baseline", |b| {
        b.iter(|| {
            // Simulate 100 requests, one with abort fault
            let results: Vec<_> = (0..100)
                .map(|i| {
                    if i == 50 {
                        abort_executor.execute()
                    } else {
                        ExecutionResult {
                            executed: false,
                            status_code: None,
                            delay_millis: 0,
                        }
                    }
                })
                .collect();
            black_box(results.len())
        })
    });

    group.bench_function("delay_vs_baseline", |b| {
        b.iter(|| {
            // Simulate 100 requests, one with delay fault
            let results: Vec<_> = (0..100)
                .map(|i| {
                    if i == 50 {
                        delay_executor.execute()
                    } else {
                        ExecutionResult {
                            executed: false,
                            status_code: None,
                            delay_millis: 0,
                        }
                    }
                })
                .collect();
            black_box(results.len())
        })
    });

    group.finish();
}

fn benchmark_concurrent_execution(c: &mut Criterion) {
    let executor = SimpleExecutor::new(FaultType::Abort { status_code: 500 });

    c.bench_function("concurrent_execution_10", |b| {
        b.iter(|| {
            // Simulate 10 concurrent executions
            let results: Vec<_> = (0..10).map(|_| executor.execute()).collect();
            black_box(results.len())
        })
    });

    c.bench_function("concurrent_execution_100", |b| {
        b.iter(|| {
            // Simulate 100 concurrent executions
            let results: Vec<_> = (0..100).map(|_| executor.execute()).collect();
            black_box(results.len())
        })
    });
}

criterion_group!(
    benches,
    benchmark_abort_execution,
    benchmark_delay_execution,
    benchmark_execution_overhead,
    benchmark_concurrent_execution
);
criterion_main!(benches);
