use std::time::Duration;

// 导入我们的模块以进行测试
mod reconnect;
mod panic_safety;

use reconnect::ReconnectManager;
use panic_safety::{setup_panic_hook, safe_execute};

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_reconnect_manager_basic() {
        let mut manager = ReconnectManager::new();
        
        // 初始状态应该允许连接
        assert!(manager.can_attempt());
        
        // 记录失败应该增加延迟
        manager.record_failure();
        let first_delay = manager.get_next_delay();
        
        // 第二次失败应该有更长的延迟
        manager.record_failure();
        let second_delay = manager.get_next_delay();
        
        assert!(second_delay >= first_delay);
        println!("First delay: {:?}, Second delay: {:?}", first_delay, second_delay);
    }

    #[test]
    fn test_reconnect_manager_max_attempts() {
        let mut manager = ReconnectManager::new();
        
        // 记录多次失败，直到达到最大尝试次数
        for i in 0..=ReconnectManager::MAX_ATTEMPTS {
            if i < ReconnectManager::MAX_ATTEMPTS {
                assert!(manager.can_attempt(), "Should allow attempt {}", i);
                manager.record_failure();
            } else {
                assert!(!manager.can_attempt(), "Should not allow attempt after max attempts");
            }
        }
    }

    #[test]
    fn test_reconnect_manager_success_resets() {
        let mut manager = ReconnectManager::new();
        
        // 记录一些失败
        manager.record_failure();
        manager.record_failure();
        
        let delay_before_success = manager.get_next_delay();
        
        // 记录成功应该重置状态
        manager.record_success();
        
        let delay_after_success = manager.get_next_delay();
        
        // 成功后延迟应该重置为初始值
        assert!(delay_after_success < delay_before_success);
        println!("Delay before success: {:?}, after success: {:?}", 
                delay_before_success, delay_after_success);
    }

    #[test]
    fn test_panic_safety_safe_execute() {
        // 测试正常执行
        let result = safe_execute(|| {
            "success".to_string()
        });
        assert_eq!(result, Some("success".to_string()));

        // 测试 panic 处理
        let result = safe_execute(|| {
            panic!("Test panic");
            #[allow(unreachable_code)]
            "never reached".to_string()
        });
        assert_eq!(result, None);
    }

    #[test]
    fn test_exponential_backoff_bounds() {
        let mut manager = ReconnectManager::new();
        
        // 测试指数退避是否在合理范围内
        for i in 0..10 {
            manager.record_failure();
            let delay = manager.get_next_delay();
            
            // 延迟应该在合理范围内
            assert!(delay >= Duration::from_millis(100), "Delay too small at attempt {}: {:?}", i, delay);
            assert!(delay <= Duration::from_secs(300), "Delay too large at attempt {}: {:?}", i, delay);
            
            println!("Attempt {}: delay = {:?}", i + 1, delay);
        }
    }
}

fn main() {
    println!("WASM Plugin Robustness Test");
    
    // 设置 panic hook
    setup_panic_hook();
    
    // 测试重连管理器
    let mut reconnect_manager = ReconnectManager::new();
    println!("Initial reconnect manager state: can_attempt = {}", reconnect_manager.can_attempt());
    
    // 模拟一些失败
    for i in 1..=3 {
        reconnect_manager.record_failure();
        let delay = reconnect_manager.get_next_delay();
        println!("Failure {}: next delay = {:?}", i, delay);
    }
    
    // 模拟成功重连
    reconnect_manager.record_success();
    println!("After success: next delay = {:?}", reconnect_manager.get_next_delay());
    
    // 测试 panic 安全执行
    println!("\nTesting panic safety:");
    
    let result1 = safe_execute(|| {
        println!("Normal execution");
        42
    });
    println!("Normal execution result: {:?}", result1);
    
    let result2 = safe_execute(|| {
        panic!("Intentional panic for testing");
        #[allow(unreachable_code)]
        0
    });
    println!("Panic execution result: {:?}", result2);
    
    println!("Robustness test completed successfully!");
}
