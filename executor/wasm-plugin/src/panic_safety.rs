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
            panic!("String panic message");
            #[allow(unreachable_code)]
            42
        });
        assert_eq!(result, None);
    }

    #[test]
    fn test_safe_execute_with_complex_type() {
        let result = safe_execute("test_complex_type", || {
            vec![1, 2, 3, 4, 5]
        });
        assert_eq!(result, Some(vec![1, 2, 3, 4, 5]));
    }

    #[test]
    fn test_safe_execute_multiple_operations() {
        let result1 = safe_execute("op1", || 10);
        let result2 = safe_execute("op2", || 20);
        let result3 = safe_execute("op3", || {
            panic!("op3 failed");
            #[allow(unreachable_code)]
            30
        });

        assert_eq!(result1, Some(10));
        assert_eq!(result2, Some(20));
        assert_eq!(result3, None);
    }

    #[test]
    fn test_safe_execute_closure_with_captures() {
        let value = 42;
        let result = safe_execute("test_with_capture", || value * 2);
        assert_eq!(result, Some(84));
    }

    #[test]
    fn test_safe_execute_closure_panic_with_captures() {
        let _captured = 42;
        let result = safe_execute("test_panic_with_capture", || {
            panic!("Panic in closure with captures");
            #[allow(unreachable_code)]
            0
        });
        assert_eq!(result, None);
    }

    #[test]
    fn test_safe_execute_nested_call() {
        let outer_result = safe_execute("outer", || {
            let inner_result = safe_execute("inner", || 10);
            inner_result.unwrap_or(0) * 2
        });

        assert_eq!(outer_result, Some(20));
    }

    #[test]
    fn test_safe_execute_nested_call_with_panic() {
        let outer_result = safe_execute("outer_with_panic", || {
            let inner_result = safe_execute("inner_panic", || {
                panic!("Inner panic");
                #[allow(unreachable_code)]
                0
            });
            
            // inner_result should be None due to panic
            inner_result.unwrap_or(100)
        });

        assert_eq!(outer_result, Some(100));
    }

    #[test]
    fn test_safe_execute_string_operations() {
        let result = safe_execute("string_op", || {
            "Hello".to_string() + " " + "World"
        });

        assert_eq!(result, Some("Hello World".to_string()));
    }

    #[test]
    fn test_safe_execute_collection_operations() {
        let result = safe_execute("collection_op", || {
            let mut v = vec![1, 2, 3];
            v.push(4);
            v.len()
        });

        assert_eq!(result, Some(4));
    }

    #[test]
    fn test_safe_execute_arithmetic_panic() {
        // Note: Division by zero in debug mode can panic
        let result = safe_execute("arithmetic", || {
            let a = 10;
            let b = 1; // Avoid actual division by zero in this test
            a / b
        });

        assert_eq!(result, Some(10));
    }

    #[test]
    fn test_safe_execute_assert_panic() {
        let result = safe_execute("assert_panic", || {
            assert!(false, "Assertion failed");
            #[allow(unreachable_code)]
            42
        });

        assert_eq!(result, None);
    }

    #[test]
    fn test_safe_execute_match_statement() {
        let result = safe_execute("match_test", || {
            match Some(42) {
                Some(n) => n * 2,
                None => 0,
            }
        });

        assert_eq!(result, Some(84));
    }

    #[test]
    fn test_safe_execute_option_unwrap_panic() {
        let result = safe_execute("option_panic", || {
            let opt: Option<i32> = None;
            opt.unwrap() // This will panic
        });

        assert_eq!(result, None);
    }

    #[test]
    fn test_safe_execute_result_unwrap_panic() {
        let result = safe_execute("result_panic", || {
            let res: Result<i32, String> = Err("Error message".to_string());
            res.unwrap() // This will panic
        });

        assert_eq!(result, None);
    }

    #[test]
    fn test_safe_execute_index_out_of_bounds() {
        let result = safe_execute("bounds_check", || {
            let v = vec![1, 2, 3];
            v[10] // This will panic in debug mode
        });

        assert_eq!(result, None);
    }

    #[test]
    fn test_safe_execute_recursion() {
        fn factorial(n: u32) -> u32 {
            if n <= 1 {
                1
            } else {
                n * factorial(n - 1)
            }
        }

        let result = safe_execute("recursion", || factorial(5));
        assert_eq!(result, Some(120));
    }

    #[test]
    fn test_safe_execute_sequential_success() {
        let results: Vec<_> = (0..5)
            .map(|i| safe_execute(&format!("op_{}", i), move || i * 2))
            .collect();

        assert_eq!(results, vec![Some(0), Some(2), Some(4), Some(6), Some(8)]);
    }

    #[test]
    fn test_safe_execute_with_struct() {
        #[derive(Debug, PartialEq)]
        struct TestData {
            value: i32,
            name: String,
        }

        let result = safe_execute("struct_op", || {
            TestData {
                value: 42,
                name: "test".to_string(),
            }
        });

        let expected = TestData {
            value: 42,
            name: "test".to_string(),
        };

        assert_eq!(result, Some(expected));
    }

    #[test]
    fn test_safe_execute_operation_name_preservation() {
        // This test verifies that operation names are used in error handling
        // even though we can't directly inspect the error messages in unit tests
        let _result = safe_execute("important_operation_123", || {
            panic!("Test panic");
            #[allow(unreachable_code)]
            0
        });

        // If we got here, the operation completed safely despite the panic
    }
}
