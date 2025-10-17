//! Simple unit test for basic functions that don't require proxy-wasm runtime

#[cfg(test)]
mod standalone_tests {
    use crate::executor::{generate_random_percentage, DelayManager};

    #[test]
    fn test_random_generation() {
        // Test that random function generates values in range
        for _ in 0..10 {
            let value = generate_random_percentage();
            assert!(value <= 100, "Random value {} should be <= 100", value);
        }
    }

    #[test]
    fn test_delay_manager_basic() {
        let mut manager = DelayManager::new();
        
        // Add some delays
        let token1 = manager.add_delay(1, 100);
        let token2 = manager.add_delay(2, 200);
        
        // Test timer handling
        assert_eq!(manager.handle_timer(token1), Some(1));
        assert_eq!(manager.handle_timer(token2), Some(2));
        assert_eq!(manager.handle_timer(999), None);
    }
}
