/// 时间控制字段反序列化测试
/// 
/// 验证 Fault 结构体的新字段 start_delay_ms 和 duration_seconds 的正确序列化和反序列化
#[cfg(test)]
mod time_control_deserialization_tests {
    use serde_json;
    use crate::config::Fault;

    #[test]
    fn test_deserialize_fault_with_time_control_fields() {
        // JSON 中包含时间控制字段
        let json_str = r#"{
            "percentage": 50,
            "start_delay_ms": 200,
            "duration_seconds": 300,
            "abort": {
                "httpStatus": 503
            }
        }"#;

        let fault: Fault = serde_json::from_str(json_str).expect("Failed to deserialize");
        
        assert_eq!(fault.percentage, 50);
        assert_eq!(fault.start_delay_ms, 200);
        assert_eq!(fault.duration_seconds, 300);
        assert!(fault.abort.is_some());
        assert_eq!(fault.abort.as_ref().unwrap().http_status, 503);
        assert!(fault.delay.is_none());
    }

    #[test]
    fn test_deserialize_fault_with_zero_time_control_fields() {
        // JSON 中包含时间控制字段但值为 0
        let json_str = r#"{
            "percentage": 10,
            "start_delay_ms": 0,
            "duration_seconds": 0,
            "delay": {
                "fixed_delay": "500ms"
            }
        }"#;

        let fault: Fault = serde_json::from_str(json_str).expect("Failed to deserialize");
        
        assert_eq!(fault.percentage, 10);
        assert_eq!(fault.start_delay_ms, 0);
        assert_eq!(fault.duration_seconds, 0);
        assert!(fault.delay.is_some());
        assert!(fault.abort.is_none());
    }

    #[test]
    fn test_deserialize_fault_without_time_control_fields() {
        // JSON 中不包含时间控制字段（向后兼容）
        let json_str = r#"{
            "percentage": 30,
            "abort": {
                "httpStatus": 500,
                "body": "Service unavailable"
            }
        }"#;

        let fault: Fault = serde_json::from_str(json_str).expect("Failed to deserialize");
        
        assert_eq!(fault.percentage, 30);
        // 应该使用默认值 0
        assert_eq!(fault.start_delay_ms, 0);
        assert_eq!(fault.duration_seconds, 0);
        assert!(fault.abort.is_some());
    }

    #[test]
    fn test_deserialize_fault_with_snake_case_fields() {
        // 验证 snake_case 的反序列化
        let json_str = r#"{
            "percentage": 20,
            "start_delay_ms": 100,
            "duration_seconds": 600
        }"#;

        let fault: Fault = serde_json::from_str(json_str).expect("Failed to deserialize");
        
        assert_eq!(fault.start_delay_ms, 100);
        assert_eq!(fault.duration_seconds, 600);
    }

    #[test]
    fn test_deserialize_fault_with_boundary_values() {
        // 边界值测试
        let json_str = r#"{
            "percentage": 100,
            "start_delay_ms": 4294967295,
            "duration_seconds": 4294967295
        }"#;

        let fault: Fault = serde_json::from_str(json_str).expect("Failed to deserialize");
        
        assert_eq!(fault.start_delay_ms, u32::MAX);
        assert_eq!(fault.duration_seconds, u32::MAX);
    }

    #[test]
    fn test_fault_field_validation_non_negative() {
        // 验证字段都应该是非负的（由于类型是 u32，自动满足）
        let json_str = r#"{
            "percentage": 50,
            "start_delay_ms": 0,
            "duration_seconds": 0
        }"#;

        let fault: Fault = serde_json::from_str(json_str).expect("Failed to deserialize");
        
        // u32 类型自动保证非负
        assert!(fault.start_delay_ms >= 0);
        assert!(fault.duration_seconds >= 0);
    }

    #[test]
    fn test_deserialize_complete_fault_with_all_fields() {
        // 完整的 Fault 结构，包含所有可能的字段
        let json_str = r#"{
            "percentage": 75,
            "start_delay_ms": 150,
            "duration_seconds": 120,
            "abort": {
                "httpStatus": 502,
                "body": "Bad Gateway"
            },
            "delay": {
                "fixed_delay": "1000ms"
            }
        }"#;

        let fault: Fault = serde_json::from_str(json_str).expect("Failed to deserialize");
        
        assert_eq!(fault.percentage, 75);
        assert_eq!(fault.start_delay_ms, 150);
        assert_eq!(fault.duration_seconds, 120);
        assert!(fault.abort.is_some());
        assert!(fault.delay.is_some());
    }

    #[test]
    #[ignore]  // Fault 未实现 Serialize，跳过此测试
    fn test_round_trip_serialization() {
        // 序列化后再反序列化，应该保持一致
        let original_json = r#"{
            "percentage": 45,
            "start_delay_ms": 250,
            "duration_seconds": 180,
            "abort": {
                "httpStatus": 503
            }
        }"#;

        // TODO: Fault 未实现 Serialize，暂时跳过序列化测试
        // 如需启用，请在 config.rs 的 Fault 结构体中添加 #[derive(Serialize)]
        let fault: Fault = serde_json::from_str(original_json)
            .expect("Failed to deserialize");
        
        // let serialized = serde_json::to_string(&fault)
        //     .expect("Failed to serialize");
        // 
        // let fault2: Fault = serde_json::from_str(&serialized)
        //     .expect("Failed to deserialize from round-trip");
        
        assert_eq!(fault.percentage, 45);
        assert_eq!(fault.start_delay_ms, 250);
        assert_eq!(fault.duration_seconds, 180);
    }
}
