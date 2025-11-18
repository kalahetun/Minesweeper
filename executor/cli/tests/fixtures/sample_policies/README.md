# CLI 测试用例 - 示例 Policy YAML 文件集合

这个目录包含用于 CLI 集成测试的预定义 YAML 策略文件。

## 文件说明

### 有效的策略文件

- `abort-policy.yaml` - 测试中止请求的策略
- `delay-policy.yaml` - 测试延迟请求的策略
- `header-policy.yaml` - 测试基于请求头匹配的策略
- `time-limited-policy.yaml` - 测试带自动过期的策略
- `multi-rule-policy.yaml` - 测试多规则策略

### 无效的策略文件（用于错误处理测试）

- `invalid-missing-name.yaml` - 缺少 metadata.name
- `invalid-missing-rules.yaml` - 缺少 spec.rules
- `invalid-bad-regex.yaml` - 包含无效的正则表达式

## 使用示例

```bash
# 应用策略
hfi-cli policy apply -f abort-policy.yaml

# 查看策略列表
hfi-cli policy list

# 获取特定策略
hfi-cli policy get abort-policy

# 删除策略
hfi-cli policy delete abort-policy
```
