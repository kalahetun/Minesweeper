use log::{info, warn, debug, error};
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Action, LogLevel};
use std::time::Duration;
use std::sync::{Arc, Mutex};

mod config;
mod matcher;
mod executor;
mod reconnect;
mod panic_safety;
mod time_control;
mod metrics;

use config::{CompiledRuleSet, Fault};
use matcher::{RequestInfo, find_first_match};
use executor::{FaultExecutorContext, DelayManager};
use reconnect::ReconnectManager;
use panic_safety::{setup_panic_hook, safe_execute};

const CONTROL_PLANE_CLUSTER: &str = "hfi_control_plane";

#[cfg(not(test))]
#[no_mangle]
pub fn _start() {
    // 设置 panic hook 以确保 panic 安全性
    setup_panic_hook();
    
    proxy_wasm::set_log_level(LogLevel::Info);
    
    proxy_wasm::set_root_context(|_| -> Box<dyn RootContext> {
        Box::new(PluginRootContext {
            control_plane_address: String::new(),
            current_rules: Arc::new(Mutex::new(None)),
            delay_manager: DelayManager::new(),
            reconnect_manager: Arc::new(Mutex::new(ReconnectManager::new())),
            config_call_id: None,
            aborts_total_metric: None,
            delays_total_metric: None,
            delay_duration_histogram: None,
        })
    });
}

//  Root Context 

// Root context for the entire plugin
struct PluginRootContext {
    control_plane_address: String,
    current_rules: Arc<Mutex<Option<CompiledRuleSet>>>,
    delay_manager: DelayManager,
    reconnect_manager: Arc<Mutex<ReconnectManager>>,
    config_call_id: Option<u32>,
    // Metrics IDs
    aborts_total_metric: Option<u32>,
    delays_total_metric: Option<u32>,
    delay_duration_histogram: Option<u32>,
}

impl FaultExecutorContext for PluginRootContext {
    fn execute_fault_for_context(&self, _fault: &Fault, _context_id: u32) -> Action {
        Action::Continue
    }
}

impl PluginRootContext {
    fn define_metrics(&mut self) {
        // Define abort counter metric
        match proxy_wasm::hostcalls::define_metric(
            proxy_wasm::types::MetricType::Counter,
            "hfi.faults.aborts_total"
        ) {
            Ok(metric_id) => {
                debug!("Defined aborts_total metric with ID: {}", metric_id);
                self.aborts_total_metric = Some(metric_id);
            }
            Err(e) => {
                warn!("Failed to define aborts_total metric: {:?}", e);
            }
        }

        // Define delay counter metric
        match proxy_wasm::hostcalls::define_metric(
            proxy_wasm::types::MetricType::Counter,
            "hfi.faults.delays_total"
        ) {
            Ok(metric_id) => {
                debug!("Defined delays_total metric with ID: {}", metric_id);
                self.delays_total_metric = Some(metric_id);
            }
            Err(e) => {
                warn!("Failed to define delays_total metric: {:?}", e);
            }
        }

        // Define delay duration histogram metric
        match proxy_wasm::hostcalls::define_metric(
            proxy_wasm::types::MetricType::Histogram,
            "hfi.faults.delay_duration_milliseconds"
        ) {
            Ok(metric_id) => {
                debug!("Defined delay_duration_milliseconds metric with ID: {}", metric_id);
                self.delay_duration_histogram = Some(metric_id);
            }
            Err(e) => {
                warn!("Failed to define delay_duration_milliseconds metric: {:?}", e);
            }
        }
    }

    fn get_aborts_total_metric(&self) -> Option<u32> {
        self.aborts_total_metric
    }

    fn get_delays_total_metric(&self) -> Option<u32> {
        self.delays_total_metric
    }

    fn get_delay_duration_histogram(&self) -> Option<u32> {
        self.delay_duration_histogram
    }

    fn dispatch_config_request(&mut self) {
        // 如果正在重连中，不要发起新的请求
        if let Ok(manager) = self.reconnect_manager.lock() {
            if manager.is_reconnecting() {
                debug!("Skipping config request - reconnection in progress");
                return;
            }
        } else {
            error!("Failed to acquire lock on reconnect manager");
            return;
        }
        
        debug!("Dispatching HTTP call to control plane: {}", self.control_plane_address);
        
        let result = safe_execute("dispatch_http_call", || {
            self.dispatch_http_call(
                CONTROL_PLANE_CLUSTER,
                vec![
                    (":method", "GET"),
                    (":path", "/v1/policies"),  // 改为使用策略列表 API
                    (":authority", &self.control_plane_address),
                    ("accept", "application/json"),
                ],
                None,
                vec![],
                Duration::from_secs(10), // 减少超时时间
            )
        });

        match result {
            Some(Ok(call_id)) => {
                info!("HTTP call dispatched successfully with ID: {}", call_id);
                self.config_call_id = Some(call_id);
            },
            Some(Err(e)) => {
                warn!("Failed to dispatch HTTP call: {:?}", e);
                self.handle_config_failure();
            },
            None => {
                error!("Panic occurred during HTTP call dispatch");
                self.handle_config_failure();
            }
        }
    }

    fn handle_config_success(&mut self) {
        if let Ok(mut manager) = self.reconnect_manager.lock() {
            manager.on_success();
        }
        self.config_call_id = None;
        
        // 设置定期配置拉取
        self.set_tick_period(Duration::from_secs(30));
    }
    
    fn handle_config_failure(&mut self) {
        if let Ok(mut manager) = self.reconnect_manager.lock() {
            if let Some(delay) = manager.on_failure() {
                info!("Scheduling reconnect attempt in {:?}", delay);
                self.set_tick_period(delay);
            } else {
                error!("Max reconnection attempts reached, giving up");
            }
        } else {
            error!("Failed to acquire lock on reconnect manager");
        }
    }
}

impl Context for PluginRootContext {
    fn on_http_call_response(&mut self, _token_id: u32, _num_headers: usize, body_size: usize, _num_trailers: usize) {
        // 检查响应状态
        let response_status = self.get_http_call_response_header(":status")
            .and_then(|status| status.parse::<u16>().ok())
            .unwrap_or(500);

        info!("Received HTTP response - Status: {}, Body size: {}", response_status, body_size);

        // 检查是否是成功的响应
        if response_status < 200 || response_status >= 300 {
            warn!("Received non-success status code: {}", response_status);
            self.handle_config_failure();
            return;
        }

        // 安全地处理响应体
        let result = safe_execute("process_config_response", || {
            if let Some(body) = self.get_http_call_response_body(0, body_size) {
                if let Ok(body_str) = std::str::from_utf8(&body) {
                    info!("Received config update from control plane: {}", body_str.trim());
                    
                    // Try to parse the received configuration from policies API
                    match CompiledRuleSet::from_policies_response(&body) {
                        Ok(ruleset) => {
                            info!("Successfully parsed {} rules from control plane", ruleset.rules.len());
                            
                            // Update rules with mutex lock
                            if let Ok(mut rules) = self.current_rules.lock() {
                                *rules = Some(ruleset);
                                
                                // Log rule details for debugging
                                if let Some(ref rs) = *rules {
                                    for (i, rule) in rs.rules.iter().enumerate() {
                                        debug!("Rule {}: {} with {} fault percentage", i, rule.name, rule.fault.percentage);
                                        if let Some(ref path) = rule.match_condition.path {
                                            debug!("  - Path matcher: exact={:?}, prefix={:?}", path.exact, path.prefix);
                                        }
                                        if let Some(ref abort) = rule.fault.abort {
                                            debug!("  - Abort: status={}, body={:?}", abort.http_status, abort.body);
                                        }
                                        if let Some(ref delay) = rule.fault.delay {
                                            debug!("  - Delay: {} ({}ms)", delay.fixed_delay, delay.parsed_duration_ms.unwrap_or(0));
                                        }
                                    }
                                }
                            } else {
                                warn!("Failed to acquire lock for rules update");
                            }
                            true // 解析成功
                        }
                        Err(e) => {
                            warn!("Failed to parse configuration from control plane: {}", e);
                            debug!("Raw response body: {}", body_str);
                            false // 解析失败
                        }
                    }
                } else {
                    warn!("Received non-UTF8 response body from control plane");
                    false
                }
            } else {
                warn!("Failed to get response body from control plane");
                false
            }
        });

        match result {
            Some(true) => {
                // 配置解析成功
                self.handle_config_success();
            }
            Some(false) => {
                // 配置解析失败
                warn!("Config parsing failed, treating as failure");
                self.handle_config_failure();
            }
            None => {
                // 发生 panic
                error!("Panic occurred during config response processing");
                self.handle_config_failure();
            }
        }
    }
}

impl RootContext for PluginRootContext {
    fn on_configure(&mut self, _plugin_configuration_size: usize) -> bool {
        info!("Plugin configured...");
        
        // Define metrics
        self.define_metrics();
        
        if let Some(config_bytes) = self.get_plugin_configuration() {
            if let Ok(config_str) = std::str::from_utf8(&config_bytes) {
                let config_str = config_str.trim();
                if config_str.is_empty() {
                    info!("Plugin configuration is empty, using default control plane address");
                    self.control_plane_address = "control-plane:8080".to_string();
                } else {
                    info!("Control plane address from config: {}", config_str);
                    self.control_plane_address = config_str.to_string();
                }
            } else {
                warn!("Configuration is not valid UTF-8: {:?}", config_bytes);
                info!("Using default control plane address");
                self.control_plane_address = "control-plane:8080".to_string();
            }
        } else {
            info!("Plugin configuration not found, using default control plane address: control-plane:8080");
            self.control_plane_address = "control-plane:8080".to_string();
        }
        
        // 设置定时器延迟调用，避免在配置期间直接调用
        self.set_tick_period(Duration::from_secs(1));
        true
    }

    fn on_tick(&mut self) {
        debug!("Tick event received");
        
        // 如果正在重连过程中，发起新的配置请求
        let is_reconnecting = self.reconnect_manager
            .lock()
            .map(|manager| manager.is_reconnecting())
            .unwrap_or(false);
            
        if is_reconnecting {
            // 降低日志级别为 debug，避免日志噪音
            debug!("Reconnection tick - attempting to fetch config");
            self.dispatch_config_request();
        } else {
            // 正常的定期配置拉取
            debug!("Regular config polling tick");
            self.dispatch_config_request();
        }
    }
    
    fn create_http_context(&self, context_id: u32) -> Option<Box<dyn HttpContext>> {
        Some(Box::new(PluginHttpContext {
            context_id,
            rules: self.current_rules.clone(),
            metrics: executor::MetricsIds {
                aborts_total: self.aborts_total_metric,
                delays_total: self.delays_total_metric,
                delay_duration_histogram: self.delay_duration_histogram,
            },
            pending_action: None,
        }))
    }

    fn get_type(&self) -> Option<proxy_wasm::types::ContextType> {
        Some(proxy_wasm::types::ContextType::HttpContext)
    }
}

//  HTTP Context 

/// HTTP 上下文的等待状态
#[derive(Debug, Clone)]
enum PendingAction {
    /// 等待 start_delay_ms 完成后执行故障注入
    StartDelay {
        /// 要执行的故障配置
        fault: Fault,
        /// 规则名称（用于日志）
        rule_name: String,
    },
    /// 等待 delay fault 完成后继续请求
    DelayFault,
}

// HTTP context for each request
struct PluginHttpContext {
    context_id: u32,
    rules: Arc<Mutex<Option<CompiledRuleSet>>>,
    metrics: executor::MetricsIds,
    /// 当前等待的动作（用于 on_http_call_response 回调）
    pending_action: Option<PendingAction>,
}

impl Context for PluginHttpContext {
    fn on_http_call_response(
        &mut self,
        _token_id: u32,
        _num_headers: usize,
        _body_size: usize,
        _num_trailers: usize,
    ) {
        match self.pending_action.take() {
            Some(PendingAction::StartDelay { fault, rule_name }) => {
                // start_delay_ms 等待完成，现在执行实际的故障注入
                info!("Start delay completed for context {}, now injecting fault from rule '{}'", 
                      self.context_id, rule_name);
                
                // 检查是否有 delay fault 需要执行
                if let Some(delay) = &fault.delay {
                    if let Some(duration_ms) = delay.parsed_duration_ms {
                        info!("Executing delay fault after start_delay: {}ms for context {}", 
                              duration_ms, self.context_id);
                        
                        // Increment delay counter metric
                        if let Some(metric_id) = self.metrics.delays_total {
                            if let Err(e) = proxy_wasm::hostcalls::increment_metric(metric_id, 1) {
                                warn!("Failed to increment delay counter: {:?}", e);
                            } else {
                                debug!("Incremented hfi.faults.delays_total counter (after start_delay)");
                            }
                        }
                        
                        // Record delay duration in histogram
                        if let Some(metric_id) = self.metrics.delay_duration_histogram {
                            if let Err(e) = proxy_wasm::hostcalls::record_metric(metric_id, duration_ms) {
                                warn!("Failed to record delay duration histogram: {:?}", e);
                            }
                        }
                        
                        let timeout = Duration::from_millis(duration_ms);
                        self.pending_action = Some(PendingAction::DelayFault);
                        
                        match self.dispatch_http_call(
                            "hfi_delay_cluster",
                            vec![
                                (":method", "GET"),
                                (":path", "/delay"),
                                (":authority", "delay.local"),
                            ],
                            None,
                            vec![],
                            timeout,
                        ) {
                            Ok(_) => {
                                info!("Delay fault triggered after start_delay for context {} - {}ms", 
                                      self.context_id, duration_ms);
                                // 保持暂停状态，等待 delay 完成
                                return;
                            }
                            Err(e) => {
                                warn!("Failed to dispatch delay call: {:?}, resuming request", e);
                                self.resume_http_request();
                                return;
                            }
                        }
                    }
                }
                
                // 执行 abort fault
                if let Some(abort) = &fault.abort {
                    info!("Executing abort fault after start_delay: {} for context {}", 
                          abort.http_status, self.context_id);
                    
                    // Increment abort counter metric
                    if let Some(metric_id) = self.metrics.aborts_total {
                        if let Err(e) = proxy_wasm::hostcalls::increment_metric(metric_id, 1) {
                            warn!("Failed to increment abort counter: {:?}", e);
                        } else {
                            debug!("Incremented hfi.faults.aborts_total counter (after start_delay)");
                        }
                    }
                    
                    let body = abort.body.clone().unwrap_or_else(|| 
                        "Fault injection: Service unavailable".to_string());
                    self.send_http_response(
                        abort.http_status,
                        vec![("content-type", "text/plain"), ("x-fault-injected", "abort")],
                        Some(body.as_bytes()),
                    );
                    return;
                }
                
                // 没有具体的故障配置，继续请求
                self.resume_http_request();
            }
            Some(PendingAction::DelayFault) => {
                // Delay fault 完成，继续请求
                info!("Delay fault completed for context {}, resuming request", self.context_id);
                self.resume_http_request();
            }
            None => {
                // 没有等待的动作，这应该不会发生
                warn!("Unexpected http_call_response for context {} with no pending action", self.context_id);
                self.resume_http_request();
            }
        }
    }
}

impl HttpContext for PluginHttpContext {
    fn on_http_request_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        debug!("Handling request headers for context {}", self.context_id);
        
        // Get current rules with lock
        let rules_guard = match self.rules.lock() {
            Ok(guard) => guard,
            Err(e) => {
                warn!("Failed to acquire lock for rules: {:?}, allowing request to continue", e);
                return Action::Continue;
            }
        };
        
        // Check if rules are available
        let rules = match rules_guard.as_ref() {
            Some(ruleset) => &ruleset.rules,
            None => {
                debug!("No rules configured, allowing request to continue");
                return Action::Continue;
            }
        };
        
        // Extract request information
        let request_info = RequestInfo::from_http_context(self);
        info!("Processing request: {} {} (context: {})", 
              request_info.method, request_info.path, self.context_id);
        
        // Find matching rule
        if let Some(matched_rule) = find_first_match(&request_info, rules) {
            info!("Request matched rule '{}' with {}% fault probability", 
                  matched_rule.name, matched_rule.fault.percentage);
            
            // Phase 6: Check policy expiration (duration_seconds)
            if matched_rule.fault.duration_seconds > 0 {
                let current_time_ms = time_control::get_current_time_ms();
                let rule_age_ms = current_time_ms.saturating_sub(matched_rule.creation_time_ms);
                let validity_window_ms = (matched_rule.fault.duration_seconds as u64) * 1000;
                
                if rule_age_ms > validity_window_ms {
                    info!("Rule '{}' has expired (age: {}ms > validity: {}ms), skipping fault injection",
                          matched_rule.name, rule_age_ms, validity_window_ms);
                    return Action::Continue;
                } else {
                    debug!("Rule '{}' is still valid (age: {}ms <= validity: {}ms)",
                           matched_rule.name, rule_age_ms, validity_window_ms);
                }
            }
            
            // Check percentage before proceeding
            let random_value = executor::generate_random_percentage();
            if random_value >= matched_rule.fault.percentage as u32 {
                debug!("Fault not triggered due to percentage (random: {}, threshold: {})", 
                       random_value, matched_rule.fault.percentage);
                return Action::Continue;
            }
            
            // Phase 7: Check start_delay_ms (per-request fault injection delay)
            // start_delay_ms is a REQUEST-LEVEL delay - each request waits this duration
            // before the fault is injected. This simulates "late-stage" failures.
            if matched_rule.fault.start_delay_ms > 0 {
                let start_delay_ms = matched_rule.fault.start_delay_ms as u64;
                info!("Applying start_delay_ms: {}ms for rule '{}' context {}", 
                      start_delay_ms, matched_rule.name, self.context_id);
                
                // Set up pending action to execute fault after delay
                self.pending_action = Some(PendingAction::StartDelay {
                    fault: matched_rule.fault.clone(),
                    rule_name: matched_rule.name.clone(),
                });
                
                let timeout = Duration::from_millis(start_delay_ms);
                match self.dispatch_http_call(
                    "hfi_delay_cluster",
                    vec![
                        (":method", "GET"),
                        (":path", "/start-delay"),
                        (":authority", "delay.local"),
                    ],
                    None,
                    vec![],
                    timeout,
                ) {
                    Ok(_) => {
                        info!("Start delay initiated for context {} - {}ms", self.context_id, start_delay_ms);
                        return Action::Pause;
                    }
                    Err(e) => {
                        warn!("Failed to dispatch start_delay call: {:?}, executing fault immediately", e);
                        self.pending_action = None;
                        // Fall through to execute fault immediately
                    }
                }
            }
            
            // Execute fault immediately (no start_delay or start_delay dispatch failed)
            
            // Check if this is a delay fault - we need special handling
            if let Some(delay) = &matched_rule.fault.delay {
                if let Some(duration_ms) = delay.parsed_duration_ms {
                    info!("Executing delay fault: {}ms for context {}", duration_ms, self.context_id);
                    
                    // Set pending action for delay fault
                    self.pending_action = Some(PendingAction::DelayFault);
                    
                    // Use dispatch_http_call with timeout to implement delay
                    let timeout = Duration::from_millis(duration_ms);
                    
                    match self.dispatch_http_call(
                        "hfi_delay_cluster",
                        vec![
                            (":method", "GET"),
                            (":path", "/delay"),
                            (":authority", "delay.local"),
                        ],
                        None,
                        vec![],
                        timeout,
                    ) {
                        Ok(_) => {
                            info!("Delay fault triggered for context {} - {}ms", self.context_id, duration_ms);
                            return Action::Pause;
                        }
                        Err(e) => {
                            warn!("Failed to dispatch delay call: {:?}, continuing without delay", e);
                            self.pending_action = None;
                            return Action::Continue;
                        }
                    }
                }
            }
            
            // Execute other fault types (abort) using the executor module
            return executor::execute_fault(&matched_rule.fault, self, self.context_id, self.metrics);
        } else {
            debug!("No rules matched, allowing request to continue");
        }
        
        Action::Continue
    }

    fn on_http_response_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        Action::Continue
    }
}
