/// 指标收集模块
/// 
/// 实现 Wasm 中的故障注入指标收集，包括：
/// - 匹配计数 (rules matched)
/// - 阻断计数 (faults injected)
/// - 延迟统计 (delay durations)
/// - 错误率统计 (error types)

use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;

/// 故障注入指标
#[derive(Debug, Clone)]
pub struct FaultInjectionMetrics {
    /// 规则匹配总数
    pub rules_matched_total: Arc<AtomicU64>,
    /// 故障注入总数
    pub faults_injected_total: Arc<AtomicU64>,
    /// 阻断故障总数
    pub aborts_total: Arc<AtomicU64>,
    /// 延迟故障总数
    pub delays_total: Arc<AtomicU64>,
    /// 延迟总耗时（毫秒）
    pub delay_duration_total_ms: Arc<AtomicU64>,
    /// 延迟故障个数（用于计算平均延迟）
    pub delay_count: Arc<AtomicU64>,
    /// 请求总数
    pub requests_total: Arc<AtomicU64>,
    /// 故障注入失败次数
    pub injection_errors_total: Arc<AtomicU64>,
    /// 时间控制阻止注入的次数（延迟期内）
    pub time_control_wait_count: Arc<AtomicU64>,
    /// 规则过期导致不注入的次数
    pub rule_expired_count: Arc<AtomicU64>,
}

impl FaultInjectionMetrics {
    /// 创建新的指标收集器
    pub fn new() -> Self {
        Self {
            rules_matched_total: Arc::new(AtomicU64::new(0)),
            faults_injected_total: Arc::new(AtomicU64::new(0)),
            aborts_total: Arc::new(AtomicU64::new(0)),
            delays_total: Arc::new(AtomicU64::new(0)),
            delay_duration_total_ms: Arc::new(AtomicU64::new(0)),
            delay_count: Arc::new(AtomicU64::new(0)),
            requests_total: Arc::new(AtomicU64::new(0)),
            injection_errors_total: Arc::new(AtomicU64::new(0)),
            time_control_wait_count: Arc::new(AtomicU64::new(0)),
            rule_expired_count: Arc::new(AtomicU64::new(0)),
        }
    }

    /// 记录规则匹配
    pub fn record_rule_matched(&self) {
        self.rules_matched_total.fetch_add(1, Ordering::Relaxed);
    }

    /// 记录故障注入
    pub fn record_fault_injected(&self) {
        self.faults_injected_total.fetch_add(1, Ordering::Relaxed);
    }

    /// 记录阻断故障
    pub fn record_abort_fault(&self) {
        self.aborts_total.fetch_add(1, Ordering::Relaxed);
    }

    /// 记录延迟故障
    pub fn record_delay_fault(&self, delay_ms: u64) {
        self.delays_total.fetch_add(1, Ordering::Relaxed);
        self.delay_duration_total_ms.fetch_add(delay_ms, Ordering::Relaxed);
        self.delay_count.fetch_add(1, Ordering::Relaxed);
    }

    /// 记录请求
    pub fn record_request(&self) {
        self.requests_total.fetch_add(1, Ordering::Relaxed);
    }

    /// 记录注入错误
    pub fn record_injection_error(&self) {
        self.injection_errors_total.fetch_add(1, Ordering::Relaxed);
    }

    /// 记录时间控制等待
    pub fn record_time_control_wait(&self) {
        self.time_control_wait_count.fetch_add(1, Ordering::Relaxed);
    }

    /// 记录规则过期
    pub fn record_rule_expired(&self) {
        self.rule_expired_count.fetch_add(1, Ordering::Relaxed);
    }

    /// 获取规则匹配计数
    pub fn get_rules_matched(&self) -> u64 {
        self.rules_matched_total.load(Ordering::Relaxed)
    }

    /// 获取故障注入计数
    pub fn get_faults_injected(&self) -> u64 {
        self.faults_injected_total.load(Ordering::Relaxed)
    }

    /// 获取阻断故障计数
    pub fn get_aborts(&self) -> u64 {
        self.aborts_total.load(Ordering::Relaxed)
    }

    /// 获取延迟故障计数
    pub fn get_delays(&self) -> u64 {
        self.delays_total.load(Ordering::Relaxed)
    }

    /// 获取平均延迟（毫秒）
    pub fn get_average_delay_ms(&self) -> f64 {
        let total = self.delay_duration_total_ms.load(Ordering::Relaxed);
        let count = self.delay_count.load(Ordering::Relaxed);
        
        if count == 0 {
            0.0
        } else {
            total as f64 / count as f64
        }
    }

    /// 获取请求总数
    pub fn get_requests_total(&self) -> u64 {
        self.requests_total.load(Ordering::Relaxed)
    }

    /// 获取注入错误计数
    pub fn get_injection_errors(&self) -> u64 {
        self.injection_errors_total.load(Ordering::Relaxed)
    }

    /// 获取时间控制等待计数
    pub fn get_time_control_wait_count(&self) -> u64 {
        self.time_control_wait_count.load(Ordering::Relaxed)
    }

    /// 获取规则过期计数
    pub fn get_rule_expired_count(&self) -> u64 {
        self.rule_expired_count.load(Ordering::Relaxed)
    }

    /// 获取故障注入率 (百分比)
    pub fn get_injection_rate(&self) -> f64 {
        let total = self.requests_total.load(Ordering::Relaxed);
        let injected = self.faults_injected_total.load(Ordering::Relaxed);
        
        if total == 0 {
            0.0
        } else {
            (injected as f64 / total as f64) * 100.0
        }
    }

    /// 获取错误率 (百分比)
    pub fn get_error_rate(&self) -> f64 {
        let total = self.requests_total.load(Ordering::Relaxed);
        let errors = self.injection_errors_total.load(Ordering::Relaxed);
        
        if total == 0 {
            0.0
        } else {
            (errors as f64 / total as f64) * 100.0
        }
    }

    /// 重置所有指标
    pub fn reset(&self) {
        self.rules_matched_total.store(0, Ordering::Relaxed);
        self.faults_injected_total.store(0, Ordering::Relaxed);
        self.aborts_total.store(0, Ordering::Relaxed);
        self.delays_total.store(0, Ordering::Relaxed);
        self.delay_duration_total_ms.store(0, Ordering::Relaxed);
        self.delay_count.store(0, Ordering::Relaxed);
        self.requests_total.store(0, Ordering::Relaxed);
        self.injection_errors_total.store(0, Ordering::Relaxed);
        self.time_control_wait_count.store(0, Ordering::Relaxed);
        self.rule_expired_count.store(0, Ordering::Relaxed);
    }

    /// 获取快照（用于报告）
    pub fn snapshot(&self) -> MetricsSnapshot {
        MetricsSnapshot {
            rules_matched: self.get_rules_matched(),
            faults_injected: self.get_faults_injected(),
            aborts: self.get_aborts(),
            delays: self.get_delays(),
            average_delay_ms: self.get_average_delay_ms(),
            requests_total: self.get_requests_total(),
            injection_errors: self.get_injection_errors(),
            time_control_wait_count: self.get_time_control_wait_count(),
            rule_expired_count: self.get_rule_expired_count(),
            injection_rate_percent: self.get_injection_rate(),
            error_rate_percent: self.get_error_rate(),
        }
    }
}

impl Default for FaultInjectionMetrics {
    fn default() -> Self {
        Self::new()
    }
}

/// 指标快照（不可变快照，用于报告）
#[derive(Debug, Clone)]
pub struct MetricsSnapshot {
    pub rules_matched: u64,
    pub faults_injected: u64,
    pub aborts: u64,
    pub delays: u64,
    pub average_delay_ms: f64,
    pub requests_total: u64,
    pub injection_errors: u64,
    pub time_control_wait_count: u64,
    pub rule_expired_count: u64,
    pub injection_rate_percent: f64,
    pub error_rate_percent: f64,
}

impl MetricsSnapshot {
    /// 输出人类可读的报告
    pub fn report(&self) -> String {
        format!(
            "=== Fault Injection Metrics Report ===\n\
             Rules Matched:              {}\n\
             Faults Injected:            {}\n\
             ├─ Aborts:                  {}\n\
             ├─ Delays:                  {}\n\
             └─ Average Delay:           {:.2} ms\n\
             Total Requests:             {}\n\
             Injection Rate:             {:.2}%\n\
             Injection Errors:           {}\n\
             Error Rate:                 {:.2}%\n\
             Time Control Wait Count:    {}\n\
             Rule Expired Count:         {}",
            self.rules_matched,
            self.faults_injected,
            self.aborts,
            self.delays,
            self.average_delay_ms,
            self.requests_total,
            self.injection_rate_percent,
            self.injection_errors,
            self.error_rate_percent,
            self.time_control_wait_count,
            self.rule_expired_count,
        )
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_metrics_creation() {
        let metrics = FaultInjectionMetrics::new();
        
        assert_eq!(metrics.get_rules_matched(), 0);
        assert_eq!(metrics.get_faults_injected(), 0);
        assert_eq!(metrics.get_requests_total(), 0);
    }

    #[test]
    fn test_record_rule_matched() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_rule_matched();
        metrics.record_rule_matched();
        metrics.record_rule_matched();
        
        assert_eq!(metrics.get_rules_matched(), 3);
    }

    #[test]
    fn test_record_fault_injected() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_fault_injected();
        metrics.record_fault_injected();
        
        assert_eq!(metrics.get_faults_injected(), 2);
    }

    #[test]
    fn test_record_abort_fault() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_abort_fault();
        metrics.record_abort_fault();
        metrics.record_abort_fault();
        
        assert_eq!(metrics.get_aborts(), 3);
    }

    #[test]
    fn test_record_delay_fault() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_delay_fault(100);
        metrics.record_delay_fault(200);
        metrics.record_delay_fault(300);
        
        assert_eq!(metrics.get_delays(), 3);
        assert_eq!(metrics.delay_duration_total_ms.load(Ordering::Relaxed), 600);
        assert!((metrics.get_average_delay_ms() - 200.0).abs() < 0.01);
    }

    #[test]
    fn test_get_average_delay_ms_no_delays() {
        let metrics = FaultInjectionMetrics::new();
        
        assert_eq!(metrics.get_average_delay_ms(), 0.0);
    }

    #[test]
    fn test_get_injection_rate() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_request();
        metrics.record_request();
        metrics.record_request();
        metrics.record_request();
        
        metrics.record_fault_injected();
        metrics.record_fault_injected();
        
        let rate = metrics.get_injection_rate();
        assert!((rate - 50.0).abs() < 0.01); // 2/4 = 50%
    }

    #[test]
    fn test_get_error_rate() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_request();
        metrics.record_request();
        metrics.record_request();
        metrics.record_request();
        metrics.record_request();
        
        metrics.record_injection_error();
        
        let rate = metrics.get_error_rate();
        assert!((rate - 20.0).abs() < 0.01); // 1/5 = 20%
    }

    #[test]
    fn test_record_request() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_request();
        metrics.record_request();
        metrics.record_request();
        
        assert_eq!(metrics.get_requests_total(), 3);
    }

    #[test]
    fn test_record_injection_error() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_injection_error();
        
        assert_eq!(metrics.get_injection_errors(), 1);
    }

    #[test]
    fn test_record_time_control_wait() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_time_control_wait();
        metrics.record_time_control_wait();
        
        assert_eq!(metrics.get_time_control_wait_count(), 2);
    }

    #[test]
    fn test_record_rule_expired() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_rule_expired();
        
        assert_eq!(metrics.get_rule_expired_count(), 1);
    }

    #[test]
    fn test_reset_metrics() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_rule_matched();
        metrics.record_fault_injected();
        metrics.record_abort_fault();
        metrics.record_request();
        
        assert!(metrics.get_rules_matched() > 0);
        
        metrics.reset();
        
        assert_eq!(metrics.get_rules_matched(), 0);
        assert_eq!(metrics.get_faults_injected(), 0);
        assert_eq!(metrics.get_aborts(), 0);
        assert_eq!(metrics.get_requests_total(), 0);
    }

    #[test]
    fn test_metrics_snapshot() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_request();
        metrics.record_request();
        metrics.record_request();
        metrics.record_fault_injected();
        metrics.record_abort_fault();
        metrics.record_delay_fault(100);
        
        let snapshot = metrics.snapshot();
        
        assert_eq!(snapshot.requests_total, 3);
        assert_eq!(snapshot.faults_injected, 1);
        assert_eq!(snapshot.aborts, 1);
        assert_eq!(snapshot.delays, 1);
    }

    #[test]
    fn test_concurrent_record() {
        let metrics = FaultInjectionMetrics::new();
        let metrics_clone1 = FaultInjectionMetrics {
            rules_matched_total: Arc::clone(&metrics.rules_matched_total),
            faults_injected_total: Arc::clone(&metrics.faults_injected_total),
            aborts_total: Arc::clone(&metrics.aborts_total),
            delays_total: Arc::clone(&metrics.delays_total),
            delay_duration_total_ms: Arc::clone(&metrics.delay_duration_total_ms),
            delay_count: Arc::clone(&metrics.delay_count),
            requests_total: Arc::clone(&metrics.requests_total),
            injection_errors_total: Arc::clone(&metrics.injection_errors_total),
            time_control_wait_count: Arc::clone(&metrics.time_control_wait_count),
            rule_expired_count: Arc::clone(&metrics.rule_expired_count),
        };
        let metrics_clone2 = FaultInjectionMetrics {
            rules_matched_total: Arc::clone(&metrics.rules_matched_total),
            faults_injected_total: Arc::clone(&metrics.faults_injected_total),
            aborts_total: Arc::clone(&metrics.aborts_total),
            delays_total: Arc::clone(&metrics.delays_total),
            delay_duration_total_ms: Arc::clone(&metrics.delay_duration_total_ms),
            delay_count: Arc::clone(&metrics.delay_count),
            requests_total: Arc::clone(&metrics.requests_total),
            injection_errors_total: Arc::clone(&metrics.injection_errors_total),
            time_control_wait_count: Arc::clone(&metrics.time_control_wait_count),
            rule_expired_count: Arc::clone(&metrics.rule_expired_count),
        };
        
        // 模拟并发操作
        metrics.record_request();
        metrics_clone1.record_request();
        metrics_clone2.record_request();
        
        metrics.record_fault_injected();
        metrics_clone1.record_abort_fault();
        metrics_clone2.record_delay_fault(50);
        
        // 验证结果（因为使用了原子操作，结果应该是准确的）
        assert_eq!(metrics.get_requests_total(), 3);
        assert_eq!(metrics.get_faults_injected(), 1);
        assert_eq!(metrics.get_aborts(), 1);
        assert_eq!(metrics.get_delays(), 1);
    }

    #[test]
    fn test_metrics_snapshot_report() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_request();
        metrics.record_request();
        metrics.record_request();
        metrics.record_request();
        metrics.record_fault_injected();
        metrics.record_fault_injected();
        metrics.record_abort_fault();
        metrics.record_delay_fault(100);
        metrics.record_delay_fault(200);
        
        let snapshot = metrics.snapshot();
        let report = snapshot.report();
        
        // 验证报告包含关键信息
        assert!(report.contains("Fault Injection Metrics Report"));
        assert!(report.contains("Requests"));
        assert!(report.contains("Faults Injected"));
        assert!(report.contains("Injection Rate"));
    }

    #[test]
    fn test_zero_injection_rate() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_request();
        metrics.record_request();
        
        // 没有故障注入
        let rate = metrics.get_injection_rate();
        assert_eq!(rate, 0.0);
    }

    #[test]
    fn test_full_injection_rate() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_request();
        metrics.record_request();
        metrics.record_request();
        
        metrics.record_fault_injected();
        metrics.record_fault_injected();
        metrics.record_fault_injected();
        
        let rate = metrics.get_injection_rate();
        assert!((rate - 100.0).abs() < 0.01); // 3/3 = 100%
    }

    #[test]
    fn test_large_delay_values() {
        let metrics = FaultInjectionMetrics::new();
        
        metrics.record_delay_fault(1000);
        metrics.record_delay_fault(2000);
        metrics.record_delay_fault(3000);
        
        assert_eq!(metrics.get_delays(), 3);
        assert!((metrics.get_average_delay_ms() - 2000.0).abs() < 0.01);
    }
}
