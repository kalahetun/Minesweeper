use log::{info, warn, debug, error};
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Action, LogLevel};
use std::time::Duration;
use std::sync::{Arc, RwLock};

mod config;
mod matcher;
mod executor;
mod reconnect;
mod panic_safety;
#[cfg(test)]
mod test_basic;
#[cfg(test)]
mod test_w5_integration;
#[cfg(test)]
mod test_w5_unit;
#[cfg(test)]
mod test_w5_pure;

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
            current_rules: Arc::new(RwLock::new(None)),
            delay_manager: DelayManager::new(),
            reconnect_manager: ReconnectManager::new(),
            config_call_id: None,
        })
    });
}

// --- Root Context ---

// Root context for the entire plugin
struct PluginRootContext {
    control_plane_address: String,
    current_rules: Arc<RwLock<Option<CompiledRuleSet>>>,
    delay_manager: DelayManager,
    reconnect_manager: ReconnectManager,
    config_call_id: Option<u32>,
}

impl FaultExecutorContext for PluginRootContext {
    fn execute_fault_for_context(&self, _fault: &Fault, _context_id: u32) -> Action {
        Action::Continue
    }
}

impl PluginRootContext {
    fn dispatch_config_request(&mut self) {
        // 如果正在重连中，不要发起新的请求
        if self.reconnect_manager.is_reconnecting() {
            debug!("Skipping config request - reconnection in progress");
            return;
        }

        info!("Dispatching HTTP call to control plane: {}", self.control_plane_address);
        
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

    fn handle_config_failure(&mut self) {
        if let Some(delay) = self.reconnect_manager.on_failure() {
            info!("Scheduling reconnect attempt in {:?}", delay);
            self.set_tick_period(delay);
        } else {
            error!("Max reconnection attempts reached, giving up");
        }
    }

    fn handle_config_success(&mut self) {
        self.reconnect_manager.on_success();
        self.config_call_id = None;
        
        // 设置定期配置拉取
        self.set_tick_period(Duration::from_secs(30));
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
                            
                            // Update rules with write lock
                            if let Ok(mut rules) = self.current_rules.write() {
                                *rules = Some(ruleset);
                                
                                // Log rule details for debugging
                                if let Some(ref rs) = *rules {
                                    for (i, rule) in rs.rules.iter().enumerate() {
                                        info!("Rule {}: {} with {} fault percentage", i, rule.name, rule.fault.percentage);
                                        if let Some(ref path) = rule.match_condition.path {
                                            info!("  - Path matcher: exact={:?}, prefix={:?}", path.exact, path.prefix);
                                        }
                                        if let Some(ref abort) = rule.fault.abort {
                                            info!("  - Abort: status={}, body={:?}", abort.http_status, abort.body);
                                        }
                                        if let Some(ref delay) = rule.fault.delay {
                                            info!("  - Delay: {} ({}ms)", delay.fixed_delay, delay.parsed_duration_ms.unwrap_or(0));
                                        }
                                    }
                                }
                            } else {
                                warn!("Failed to acquire write lock for rules update");
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
        if self.reconnect_manager.is_reconnecting() {
            info!("Reconnection tick - attempting to fetch config");
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
        }))
    }

    fn get_type(&self) -> Option<proxy_wasm::types::ContextType> {
        Some(proxy_wasm::types::ContextType::HttpContext)
    }
}

// --- HTTP Context ---

// HTTP context for each request
struct PluginHttpContext {
    context_id: u32,
    rules: Arc<RwLock<Option<CompiledRuleSet>>>,
}

impl Context for PluginHttpContext {}

impl HttpContext for PluginHttpContext {
    fn on_http_request_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        debug!("Handling request headers for context {}", self.context_id);
        
        // Get current rules with read lock
        let rules_guard = match self.rules.read() {
            Ok(guard) => guard,
            Err(_) => {
                warn!("Failed to acquire read lock for rules, allowing request to continue");
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
            
            // Execute fault injection using the executor module
            return executor::execute_fault(&matched_rule.fault, self, self.context_id);
        } else {
            debug!("No rules matched, allowing request to continue");
        }
        
        Action::Continue
    }

    fn on_http_response_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        Action::Continue
    }
}
