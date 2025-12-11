# API Contracts: 009-cli-examples-update

**Status**: N/A - 本功能不引入新 API

## 说明

本功能专注于更新 CLI 示例文件和创建验证脚本，不涉及 API 变更。

现有 API 保持不变：
- `POST /v1/policies` - 创建/更新策略
- `GET /v1/policies` - 获取策略列表
- `GET /v1/policies/:name` - 获取单个策略
- `DELETE /v1/policies/:name` - 删除策略

## 参考

- Control Plane API 定义: `executor/control-plane/api/policy_controller.go`
- 策略数据模型: `executor/control-plane/storage/types.go`
- CLI 类型定义: `executor/cli/types/policy.go`
