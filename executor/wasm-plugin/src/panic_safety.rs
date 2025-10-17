use log::error;
use std::panic;

/// 设置全局 panic hook 以确保 panic 安全性
pub fn setup_panic_hook() {
    panic::set_hook(Box::new(|panic_info| {
        let payload = if let Some(s) = panic_info.payload().downcast_ref::<&str>() {
            s.to_string()
        } else if let Some(s) = panic_info.payload().downcast_ref::<String>() {
            s.clone()
        } else {
            "Unknown panic payload".to_string()
        };

        let location = if let Some(location) = panic_info.location() {
            format!("{}:{}:{}", location.file(), location.line(), location.column())
        } else {
            "Unknown location".to_string()
        };

        error!(
            "PANIC occurred in WASM plugin - Payload: '{}', Location: {}",
            payload, location
        );

        // 也使用 proxy_wasm 的日志功能，确保在 Envoy 日志中也能看到
        proxy_wasm::hostcalls::log(
            proxy_wasm::types::LogLevel::Critical,
            &format!(
                "[WASM PANIC] Payload: '{}', Location: {}",
                payload, location
            ),
        ).unwrap_or(());
    }));

    // 记录 panic hook 已设置
    log::info!("Global panic hook has been set up for WASM plugin safety");
}

/// 包装可能 panic 的操作，提供额外的错误上下文
pub fn safe_execute<F, R>(operation_name: &str, f: F) -> Option<R>
where
    F: FnOnce() -> R + panic::UnwindSafe,
{
    match panic::catch_unwind(f) {
        Ok(result) => Some(result),
        Err(panic_payload) => {
            let payload_str = if let Some(s) = panic_payload.downcast_ref::<&str>() {
                s.to_string()
            } else if let Some(s) = panic_payload.downcast_ref::<String>() {
                s.clone()
            } else {
                "Unknown panic payload".to_string()
            };

            error!(
                "Caught panic in operation '{}': {}",
                operation_name, payload_str
            );

            proxy_wasm::hostcalls::log(
                proxy_wasm::types::LogLevel::Error,
                &format!(
                    "[WASM SAFE_EXECUTE] Panic caught in '{}': {}",
                    operation_name, payload_str
                ),
            ).unwrap_or(());

            None
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_safe_execute_success() {
        let result = safe_execute("test_operation", || 42);
        assert_eq!(result, Some(42));
    }

    #[test]
    fn test_safe_execute_panic() {
        let result = safe_execute("test_panic", || {
            panic!("Test panic");
            #[allow(unreachable_code)]
            42
        });
        assert_eq!(result, None);
    }

    #[test]
    fn test_safe_execute_string_panic() {
        let result = safe_execute("test_string_panic", || {
            // 测试字符串 panic
            panic!("String panic message");
            #[allow(unreachable_code)]
            42
        });
        assert_eq!(result, None);
    }
}
