use log::{info, warn};
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Action, LogLevel};
use std::time::Duration;

const CONTROL_PLANE_CLUSTER: &str = "hfi_control_plane";

#[no_mangle]
pub fn _start() {
    proxy_wasm::set_log_level(LogLevel::Info);
    proxy_wasm::set_root_context(|_| -> Box<dyn RootContext> {
        Box::new(PluginRootContext {
            control_plane_address: String::new(),
        })
    });
}

// --- Root Context ---

// Root context for the entire plugin
struct PluginRootContext {
    control_plane_address: String,
}

impl PluginRootContext {
    fn dispatch_config_request(&self) {
        info!("Dispatching HTTP call to control plane: {}", self.control_plane_address);
        match self.dispatch_http_call(
            CONTROL_PLANE_CLUSTER,
            vec![
                (":method", "GET"),
                (":path", "/v1/config/stream"),
                (":authority", &self.control_plane_address),
            ],
            None,
            vec![],
            Duration::from_secs(30), // 减少超时时间
        ) {
            Ok(call_id) => info!("HTTP call dispatched successfully with ID: {}", call_id),
            Err(e) => {
                warn!("Failed to dispatch HTTP call: {:?}", e);
                info!("Will retry in next cycle");
            }
        }
    }
}

impl Context for PluginRootContext {
    fn on_http_call_response(&mut self, _token_id: u32, _num_headers: usize, body_size: usize, _num_trailers: usize) {
        if let Some(body) = self.get_http_call_response_body(0, body_size) {
            if let Ok(body_str) = std::str::from_utf8(&body) {
                info!("Received config update from control plane: {}", body_str.trim());
            } else {
                warn!("Received non-UTF8 response body from control plane");
            }
        } else {
            warn!("Failed to get response body from control plane");
        }

        // Re-dispatch the request to continue polling for updates.
        self.dispatch_config_request();
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
        info!("Timer tick - making config request");
        self.dispatch_config_request();
        // 设置更长的间隔用于后续轮询
        self.set_tick_period(Duration::from_secs(30));
    }

    fn create_http_context(&self, _context_id: u32) -> Option<Box<dyn HttpContext>> {
        Some(Box::new(PluginHttpContext))
    }

    fn get_type(&self) -> Option<proxy_wasm::types::ContextType> {
        Some(proxy_wasm::types::ContextType::HttpContext)
    }
}

// --- HTTP Context ---

// HTTP context for each request
struct PluginHttpContext;

impl Context for PluginHttpContext {}

impl HttpContext for PluginHttpContext {
    fn on_http_request_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        info!("Handling request...");
        Action::Continue
    }

    fn on_http_response_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        Action::Continue
    }
}
