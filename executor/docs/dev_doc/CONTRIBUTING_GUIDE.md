# 贡献指南

欢迎为 HFI (HTTP Fault Injection) 项目贡献代码！本文档将指导您如何参与项目开发，包括标准的贡献流程以及如何扩展系统功能。

## 📋 目录

- [贡献流程](#贡献流程)
- [开发环境设置](#开发环境设置)
- [代码规范](#代码规范)
- [扩展系统功能](#扩展系统功能)
- [测试指南](#测试指南)
- [文档贡献](#文档贡献)
- [社区准则](#社区准则)

## 🔄 贡献流程

### GitHub Fork & Pull Request 流程

我们采用标准的 GitHub 协作模式：

#### 1. Fork 和 Clone

```bash
# 1. 在 GitHub 上 Fork 项目
# 2. Clone 您的 Fork
git clone https://github.com/YOUR_USERNAME/wasm_fault_injection.git
cd wasm_fault_injection

# 3. 添加上游仓库
git remote add upstream https://github.com/ORIGINAL_OWNER/wasm_fault_injection.git
```

#### 2. 创建功能分支

```bash
# 从 main 分支创建新分支
git checkout main
git pull upstream main
git checkout -b feature/your-feature-name

# 或者修复 bug
git checkout -b fix/bug-description
```

#### 3. 开发和提交

```bash
# 进行您的修改
# ...

# 提交更改
git add .
git commit -m "feat: add response corruption fault type"
```

#### 4. 同步上游更改

```bash
# 定期同步上游更改
git fetch upstream
git rebase upstream/main
```

#### 5. 推送和创建 Pull Request

```bash
# 推送到您的 Fork
git push origin feature/your-feature-name

# 在 GitHub 上创建 Pull Request
```

### Commit Message 格式

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 格式：

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

#### Type 类型

| Type | 说明 | 示例 |
|------|------|------|
| `feat` | 新功能 | `feat: add response corruption fault` |
| `fix` | Bug 修复 | `fix: resolve memory leak in wasm plugin` |
| `docs` | 文档更新 | `docs: update API reference` |
| `style` | 代码格式（不影响功能） | `style: format go code with gofmt` |
| `refactor` | 重构（不是新功能或bug修复） | `refactor: extract common fault logic` |
| `test` | 添加或修改测试 | `test: add unit tests for delay fault` |
| `chore` | 构建过程或辅助工具变动 | `chore: update dependencies` |
| `perf` | 性能优化 | `perf: optimize wasm memory usage` |
| `ci` | CI/CD 配置 | `ci: add github actions workflow` |

#### Scope 范围（可选）

| Scope | 说明 |
|-------|------|
| `control-plane` | Control Plane 相关修改 |
| `wasm` | WASM 插件相关修改 |
| `cli` | CLI 工具相关修改 |
| `api` | API 定义相关修改 |
| `docs` | 文档相关修改 |
| `build` | 构建系统相关修改 |

#### 示例

```bash
# 好的 commit message
feat(wasm): add response corruption fault type
fix(control-plane): resolve SSE connection leak
docs(api): update fault injection policy spec
test(cli): add integration tests for policy commands

# 包含详细描述的 commit
feat(wasm): add response corruption fault type

- Implement ResponseCorruptionAction in config.rs
- Add corruption logic in executor.rs
- Support text replacement and JSON field modification
- Add comprehensive unit tests

Closes #123
```

## 🛠️ 开发环境设置

### 前置要求

```bash
# 安装必要工具
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh  # Rust
go install golang.org/dl/go1.21.0@latest                        # Go 1.21+
sudo apt-get install docker.io docker-compose                   # Docker

# 安装 wasm32 目标
rustup target add wasm32-unknown-unknown

# 安装开发工具
cargo install wasm-pack
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### 本地开发环境

```bash
# 1. 克隆并进入项目目录
git clone https://github.com/YOUR_USERNAME/wasm_fault_injection.git
cd wasm_fault_injection

# 2. 启动开发环境
make dev-setup

# 3. 构建项目
make build

# 4. 运行测试
make test

# 5. 启动服务
make dev-start
```

### IDE 配置

推荐使用 VS Code 配置：

```json
// .vscode/settings.json
{
  "go.toolsManagement.checkForUpdates": "local",
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--fast"],
  "rust-analyzer.cargo.target": "wasm32-unknown-unknown",
  "rust-analyzer.checkOnSave.command": "clippy"
}
```

## 📝 代码规范

### Go 代码规范

```bash
# 格式化代码
go fmt ./...

# 运行 linter
golangci-lint run

# 运行测试
go test ./...
```

#### Go 编码标准

- 遵循 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 使用 `gofmt` 格式化代码
- 所有导出的函数和类型必须有注释
- 错误处理：优先返回错误而不是 panic

```go
// 好的例子
func CreatePolicy(ctx context.Context, policy *FaultInjectionPolicy) (*Policy, error) {
    if policy == nil {
        return nil, fmt.Errorf("policy cannot be nil")
    }
    
    // 业务逻辑
    return result, nil
}

// 避免
func CreatePolicy(policy *FaultInjectionPolicy) *Policy {
    if policy == nil {
        panic("policy cannot be nil")  // 避免 panic
    }
    return result
}
```

### Rust 代码规范

```bash
# 格式化代码
cargo fmt

# 运行 clippy
cargo clippy -- -D warnings

# 运行测试
cargo test
```

#### Rust 编码标准

- 遵循 [Rust API Guidelines](https://rust-lang.github.io/api-guidelines/)
- 使用 `cargo fmt` 格式化代码
- 所有 `pub` 项必须有文档注释
- 优先使用 `Result<T, E>` 处理错误

```rust
// 好的例子
/// 执行故障注入逻辑
pub fn execute_fault(&self, fault: &FaultConfig) -> Result<FaultResult, FaultError> {
    match fault.fault_type {
        FaultType::Delay(ref delay) => self.execute_delay(delay),
        FaultType::Abort(ref abort) => self.execute_abort(abort),
        _ => Err(FaultError::UnsupportedFaultType),
    }
}

// 避免
pub fn execute_fault(&self, fault: &FaultConfig) -> FaultResult {
    // 缺少错误处理
}
```

## 🚀 扩展系统功能

### Case Study 1: 添加新的故障类型 - "Response Corruption"

让我们通过一个具体例子来说明如何为系统添加新的故障类型：**Response Corruption**（响应体修改）。

#### 步骤 1: Control Plane - 添加数据结构

**文件**: `control-plane/api/v1alpha1/types.go`

```go
// 在 Fault 结构体中添加新字段
type Fault struct {
    Delay              *DelayAction              `json:"delay,omitempty"`
    Abort              *AbortAction              `json:"abort,omitempty"`
    RateLimit          *RateLimitAction          `json:"rateLimit,omitempty"`
    ResponseModification *ResponseModificationAction `json:"responseModification,omitempty"`
    // 新增: Response Corruption 故障类型
    ResponseCorruption *ResponseCorruptionAction `json:"responseCorruption,omitempty"`
}

// 定义新的故障配置结构
type ResponseCorruptionAction struct {
    // 影响的请求百分比
    Percentage float32 `json:"percentage"`
    
    // 修改类型
    Type CorruptionType `json:"type"`
    
    // 文本替换配置
    TextReplacement *TextReplacementConfig `json:"textReplacement,omitempty"`
    
    // JSON 字段修改配置
    JSONFieldModification *JSONFieldModificationConfig `json:"jsonFieldModification,omitempty"`
    
    // 二进制数据损坏配置
    BinaryCorruption *BinaryCorruptionConfig `json:"binaryCorruption,omitempty"`
}

type CorruptionType string

const (
    CorruptionTypeTextReplacement     CorruptionType = "textReplacement"
    CorruptionTypeJSONFieldModification CorruptionType = "jsonFieldModification"
    CorruptionTypeBinaryCorruption    CorruptionType = "binaryCorruption"
)

type TextReplacementConfig struct {
    Pattern     string `json:"pattern"`     // 正则表达式模式
    Replacement string `json:"replacement"` // 替换内容
    MaxReplacements int32 `json:"maxReplacements,omitempty"` // 最大替换次数
}

type JSONFieldModificationConfig struct {
    FieldPath   string      `json:"fieldPath"`   // JSON 字段路径，如 "user.name"
    NewValue    interface{} `json:"newValue"`    // 新值
    Operation   string      `json:"operation"`   // "replace", "delete", "corrupt"
}

type BinaryCorruptionConfig struct {
    CorruptionRate float32 `json:"corruptionRate"` // 字节损坏率 (0.0-1.0)
    RandomSeed     int64   `json:"randomSeed,omitempty"` // 随机种子
}
```

#### 步骤 2: WASM Plugin - 同步数据结构

**文件**: `wasm/src/config.rs`

```rust
// 在 Fault 结构体中添加对应字段
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Fault {
    pub delay: Option<DelayAction>,
    pub abort: Option<AbortAction>,
    pub rate_limit: Option<RateLimitAction>,
    pub response_modification: Option<ResponseModificationAction>,
    // 新增: Response Corruption 配置
    pub response_corruption: Option<ResponseCorruptionAction>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResponseCorruptionAction {
    pub percentage: f32,
    pub r#type: CorruptionType,
    pub text_replacement: Option<TextReplacementConfig>,
    pub json_field_modification: Option<JsonFieldModificationConfig>,
    pub binary_corruption: Option<BinaryCorruptionConfig>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum CorruptionType {
    TextReplacement,
    JsonFieldModification,
    BinaryCorruption,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TextReplacementConfig {
    pub pattern: String,
    pub replacement: String,
    pub max_replacements: Option<i32>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JsonFieldModificationConfig {
    pub field_path: String,
    pub new_value: serde_json::Value,
    pub operation: String,  // "replace", "delete", "corrupt"
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BinaryCorruptionConfig {
    pub corruption_rate: f32,
    pub random_seed: Option<i64>,
}
```

#### 步骤 3: WASM Plugin - 实现执行逻辑

**文件**: `wasm/src/executor.rs`

```rust
impl FaultExecutor {
    pub fn execute_fault(&self, fault: &Fault) -> Result<FaultResult, FaultError> {
        // 在现有的匹配分支中添加新的故障类型
        if let Some(delay) = &fault.delay {
            if self.should_apply_fault(delay.percentage) {
                return self.execute_delay(delay);
            }
        }

        if let Some(abort) = &fault.abort {
            if self.should_apply_fault(abort.percentage) {
                return self.execute_abort(abort);
            }
        }

        if let Some(rate_limit) = &fault.rate_limit {
            if self.should_apply_fault(rate_limit.percentage) {
                return self.execute_rate_limit(rate_limit);
            }
        }

        // 新增: Response Corruption 执行逻辑
        if let Some(corruption) = &fault.response_corruption {
            if self.should_apply_fault(corruption.percentage) {
                return self.execute_response_corruption(corruption);
            }
        }

        Ok(FaultResult::NoFault)
    }

    fn execute_response_corruption(&self, corruption: &ResponseCorruptionAction) -> Result<FaultResult, FaultError> {
        // Response corruption 需要在响应阶段处理，这里只是标记
        Ok(FaultResult::ResponseCorruption(corruption.clone()))
    }
}

// 在 FaultResult 枚举中添加新的结果类型
#[derive(Debug, Clone)]
pub enum FaultResult {
    NoFault,
    Delay(Duration),
    Abort { status: u32, body: String, headers: Vec<(String, String)> },
    RateLimit { rejected: bool, status: u32, body: String },
    // 新增: Response Corruption 结果
    ResponseCorruption(ResponseCorruptionAction),
}
```

**文件**: `wasm/src/lib.rs` - 在 HTTP Context 中处理响应

```rust
impl Context for HttpContext {
    // 在响应体处理中实现 corruption 逻辑
    fn on_http_response_body(&mut self, body_size: usize, end_of_stream: bool) -> Action {
        // 检查是否需要执行 response corruption
        if let Some(corruption_config) = &self.response_corruption_config {
            match self.apply_response_corruption(body_size, corruption_config) {
                Ok(_) => log::info!("Response corruption applied successfully"),
                Err(e) => log::error!("Failed to apply response corruption: {:?}", e),
            }
        }

        Action::Continue
    }
}

impl HttpContext {
    fn apply_response_corruption(
        &mut self, 
        body_size: usize, 
        config: &ResponseCorruptionAction
    ) -> Result<(), Box<dyn std::error::Error>> {
        // 获取响应体数据
        let body = self.get_http_response_body(0, body_size)
            .ok_or("Failed to get response body")?;

        let corrupted_body = match config.r#type {
            CorruptionType::TextReplacement => {
                self.apply_text_replacement(&body, config.text_replacement.as_ref())?
            },
            CorruptionType::JsonFieldModification => {
                self.apply_json_modification(&body, config.json_field_modification.as_ref())?
            },
            CorruptionType::BinaryCorruption => {
                self.apply_binary_corruption(&body, config.binary_corruption.as_ref())?
            },
        };

        // 设置修改后的响应体
        self.set_http_response_body(0, body_size, &corrupted_body);
        
        Ok(())
    }

    fn apply_text_replacement(
        &self, 
        body: &[u8], 
        config: Option<&TextReplacementConfig>
    ) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
        let config = config.ok_or("Text replacement config is required")?;
        let body_str = String::from_utf8_lossy(body);
        
        // 使用正则表达式进行替换（简化版本，实际需要更复杂的正则处理）
        let modified = body_str.replace(&config.pattern, &config.replacement);
        
        Ok(modified.into_bytes())
    }

    fn apply_json_modification(
        &self, 
        body: &[u8], 
        config: Option<&JsonFieldModificationConfig>
    ) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
        let config = config.ok_or("JSON modification config is required")?;
        
        // 解析 JSON
        let mut json: serde_json::Value = serde_json::from_slice(body)?;
        
        // 根据字段路径修改值
        self.modify_json_field(&mut json, &config.field_path, &config.new_value, &config.operation)?;
        
        // 序列化回 JSON
        let modified = serde_json::to_vec(&json)?;
        Ok(modified)
    }

    fn apply_binary_corruption(
        &self, 
        body: &[u8], 
        config: Option<&BinaryCorruptionConfig>
    ) -> Result<Vec<u8>, Box<dyn std::error::Error>> {
        let config = config.ok_or("Binary corruption config is required")?;
        let mut corrupted = body.to_vec();
        
        // 使用指定的损坏率随机修改字节
        let corruption_count = (body.len() as f32 * config.corruption_rate) as usize;
        
        for _ in 0..corruption_count {
            let index = self.generate_random_index(body.len())?;
            corrupted[index] = corrupted[index].wrapping_add(1); // 简单的字节修改
        }
        
        Ok(corrupted)
    }
}
```

#### 步骤 4: 添加配置验证

**文件**: `control-plane/internal/handlers/policy.go`

```go
func validateFaultInjectionPolicy(policy *v1alpha1.FaultInjectionPolicy) error {
    // 现有验证逻辑...
    
    // 添加 Response Corruption 验证
    if fault := policy.Spec.Fault; fault != nil {
        if corruption := fault.ResponseCorruption; corruption != nil {
            if err := validateResponseCorruption(corruption); err != nil {
                return fmt.Errorf("invalid response corruption config: %w", err)
            }
        }
    }
    
    return nil
}

func validateResponseCorruption(corruption *v1alpha1.ResponseCorruptionAction) error {
    if corruption.Percentage < 0 || corruption.Percentage > 100 {
        return fmt.Errorf("percentage must be between 0 and 100, got %f", corruption.Percentage)
    }
    
    switch corruption.Type {
    case v1alpha1.CorruptionTypeTextReplacement:
        if corruption.TextReplacement == nil {
            return fmt.Errorf("text replacement config is required for text replacement type")
        }
        if corruption.TextReplacement.Pattern == "" {
            return fmt.Errorf("pattern is required for text replacement")
        }
    case v1alpha1.CorruptionTypeJSONFieldModification:
        if corruption.JSONFieldModification == nil {
            return fmt.Errorf("JSON field modification config is required")
        }
        if corruption.JSONFieldModification.FieldPath == "" {
            return fmt.Errorf("field path is required for JSON modification")
        }
    case v1alpha1.CorruptionTypeBinaryCorruption:
        if corruption.BinaryCorruption == nil {
            return fmt.Errorf("binary corruption config is required")
        }
        rate := corruption.BinaryCorruption.CorruptionRate
        if rate < 0 || rate > 1 {
            return fmt.Errorf("corruption rate must be between 0 and 1, got %f", rate)
        }
    default:
        return fmt.Errorf("unsupported corruption type: %s", corruption.Type)
    }
    
    return nil
}
```

#### 步骤 5: 添加测试

**文件**: `wasm/src/executor_test.rs`

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_response_corruption_text_replacement() {
        let corruption = ResponseCorruptionAction {
            percentage: 100.0,
            r#type: CorruptionType::TextReplacement,
            text_replacement: Some(TextReplacementConfig {
                pattern: "success".to_string(),
                replacement: "failure".to_string(),
                max_replacements: None,
            }),
            json_field_modification: None,
            binary_corruption: None,
        };

        let executor = FaultExecutor::new();
        let result = executor.execute_response_corruption(&corruption).unwrap();
        
        match result {
            FaultResult::ResponseCorruption(config) => {
                assert_eq!(config.percentage, 100.0);
                assert!(matches!(config.r#type, CorruptionType::TextReplacement));
            },
            _ => panic!("Expected ResponseCorruption result"),
        }
    }

    #[test]
    fn test_response_corruption_json_modification() {
        let corruption = ResponseCorruptionAction {
            percentage: 50.0,
            r#type: CorruptionType::JsonFieldModification,
            text_replacement: None,
            json_field_modification: Some(JsonFieldModificationConfig {
                field_path: "status".to_string(),
                new_value: serde_json::Value::String("error".to_string()),
                operation: "replace".to_string(),
            }),
            binary_corruption: None,
        };

        let executor = FaultExecutor::new();
        let result = executor.execute_response_corruption(&corruption).unwrap();
        
        match result {
            FaultResult::ResponseCorruption(_) => {
                // 测试通过
            },
            _ => panic!("Expected ResponseCorruption result"),
        }
    }
}
```

#### 步骤 6: 更新文档和示例

**文件**: `examples/response-corruption-example.yaml`

```yaml
apiVersion: hfi.io/v1alpha1
kind: FaultInjectionPolicy
metadata:
  name: response-corruption-demo
  namespace: default
spec:
  priority: 100
  enabled: true
  match:
    path:
      prefix: "/api/v1/users"
  fault:
    responseCorruption:
      percentage: 30.0
      type: textReplacement
      textReplacement:
        pattern: '"status":\s*"success"'
        replacement: '"status": "error"'
        maxReplacements: 1
```

### Case Study 2: 添加新的匹配条件 - "Query Parameter Matcher"

让我们扩展请求匹配功能，添加对查询参数的精确匹配支持。

#### 步骤 1: Control Plane - 扩展匹配结构

**文件**: `control-plane/api/v1alpha1/types.go`

```go
// 扩展 Match 结构体
type Match struct {
    Method      *MethodMatch      `json:"method,omitempty"`
    Path        *PathMatch        `json:"path,omitempty"`
    Headers     []HeaderMatch     `json:"headers,omitempty"`
    // 新增: Query Parameter 匹配
    QueryParams []QueryParamMatch `json:"queryParams,omitempty"`
    Body        *BodyMatch        `json:"body,omitempty"`
    SourceIP    []string          `json:"sourceIP,omitempty"`
}

// 定义新的查询参数匹配结构
type QueryParamMatch struct {
    // 参数名称
    Name string `json:"name"`
    
    // 匹配值（与现有的 StringMatch 保持一致）
    Value *StringMatch `json:"value,omitempty"`
    
    // 是否检查参数存在性（忽略值）
    Present *bool `json:"present,omitempty"`
    
    // 反向匹配
    Invert bool `json:"invert,omitempty"`
}

// 复用现有的 StringMatch 结构
type StringMatch struct {
    Exact  string `json:"exact,omitempty"`
    Prefix string `json:"prefix,omitempty"`
    Suffix string `json:"suffix,omitempty"`
    Regex  string `json:"regex,omitempty"`
}
```

#### 步骤 2: WASM Plugin - 同步匹配结构

**文件**: `wasm/src/config.rs`

```rust
// 扩展 Match 结构体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Match {
    pub method: Option<MethodMatch>,
    pub path: Option<PathMatch>,
    pub headers: Option<Vec<HeaderMatch>>,
    // 新增: Query Parameter 匹配
    pub query_params: Option<Vec<QueryParamMatch>>,
    pub body: Option<BodyMatch>,
    pub source_ip: Option<Vec<String>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QueryParamMatch {
    pub name: String,
    pub value: Option<StringMatch>,
    pub present: Option<bool>,
    pub invert: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StringMatch {
    pub exact: Option<String>,
    pub prefix: Option<String>,
    pub suffix: Option<String>,
    pub regex: Option<String>,
}
```

#### 步骤 3: WASM Plugin - 实现匹配逻辑

**文件**: `wasm/src/matcher.rs`

```rust
impl RequestMatcher {
    pub fn matches(&self, request_info: &RequestInfo) -> bool {
        // 现有匹配逻辑...
        
        // 添加 Query Parameter 匹配
        if !self.matches_query_params(request_info) {
            return false;
        }
        
        true
    }
    
    fn matches_query_params(&self, request_info: &RequestInfo) -> bool {
        let query_params = match &self.match_config.query_params {
            Some(params) => params,
            None => return true, // 没有查询参数条件，默认匹配
        };
        
        for param_match in query_params {
            if !self.matches_single_query_param(request_info, param_match) {
                return false;
            }
        }
        
        true
    }
    
    fn matches_single_query_param(
        &self, 
        request_info: &RequestInfo, 
        param_match: &QueryParamMatch
    ) -> bool {
        let query_string = match request_info.query_string.as_ref() {
            Some(qs) => qs,
            None => {
                // 没有查询字符串，检查是否要求参数不存在
                return param_match.invert || param_match.present != Some(true);
            }
        };
        
        let param_value = self.extract_query_param(query_string, &param_match.name);
        let param_exists = param_value.is_some();
        
        // 检查存在性匹配
        if let Some(present) = param_match.present {
            let result = param_exists == present;
            return if param_match.invert { !result } else { result };
        }
        
        // 检查值匹配
        if let Some(value_match) = &param_match.value {
            let result = match param_value {
                Some(value) => self.matches_string_value(&value, value_match),
                None => false, // 参数不存在，无法进行值匹配
            };
            return if param_match.invert { !result } else { result };
        }
        
        // 如果既没有 present 也没有 value，默认只检查参数存在
        let result = param_exists;
        if param_match.invert { !result } else { result }
    }
    
    fn extract_query_param(&self, query_string: &str, param_name: &str) -> Option<String> {
        // 解析查询字符串，提取指定参数的值
        for param in query_string.split('&') {
            if let Some((key, value)) = param.split_once('=') {
                if key == param_name {
                    // URL 解码
                    return Some(self.url_decode(value));
                }
            } else if param == param_name {
                // 参数存在但没有值（如 ?debug）
                return Some(String::new());
            }
        }
        None
    }
    
    fn matches_string_value(&self, value: &str, string_match: &StringMatch) -> bool {
        if let Some(exact) = &string_match.exact {
            return value == exact;
        }
        
        if let Some(prefix) = &string_match.prefix {
            return value.starts_with(prefix);
        }
        
        if let Some(suffix) = &string_match.suffix {
            return value.ends_with(suffix);
        }
        
        if let Some(regex_pattern) = &string_match.regex {
            // 简化版本的正则匹配（实际实现需要使用正则库）
            return value.contains(regex_pattern);
        }
        
        false
    }
    
    fn url_decode(&self, encoded: &str) -> String {
        // 简化版本的 URL 解码
        encoded.replace("%20", " ")
               .replace("%3D", "=")
               .replace("%26", "&")
        // 实际实现应该使用完整的 URL 解码
    }
}

// 扩展 RequestInfo 结构体包含查询字符串
#[derive(Debug, Clone)]
pub struct RequestInfo {
    pub method: String,
    pub path: String,
    pub query_string: Option<String>,
    pub headers: std::collections::HashMap<String, String>,
    pub body: Option<Vec<u8>>,
    pub source_ip: Option<String>,
}
```

#### 步骤 4: 更新请求信息收集

**文件**: `wasm/src/lib.rs`

```rust
impl HttpContext {
    fn collect_request_info(&self) -> RequestInfo {
        let method = self.get_http_request_header(":method")
            .unwrap_or_else(|| "GET".to_string());
            
        let path = self.get_http_request_header(":path")
            .unwrap_or_else(|| "/".to_string());
        
        // 分离路径和查询字符串
        let (path_only, query_string) = if let Some(query_start) = path.find('?') {
            let path_part = path[..query_start].to_string();
            let query_part = if query_start + 1 < path.len() {
                Some(path[query_start + 1..].to_string())
            } else {
                Some(String::new())
            };
            (path_part, query_part)
        } else {
            (path, None)
        };
        
        // 收集其他请求信息...
        let headers = self.collect_headers();
        let source_ip = self.get_source_ip();
        
        RequestInfo {
            method,
            path: path_only,
            query_string,
            headers,
            body: None, // 在需要时收集
            source_ip,
        }
    }
}
```

#### 步骤 5: 添加验证逻辑

**文件**: `control-plane/internal/handlers/policy.go`

```go
func validateMatch(match *v1alpha1.Match) error {
    // 现有验证逻辑...
    
    // 添加 Query Parameter 验证
    for i, queryParam := range match.QueryParams {
        if err := validateQueryParamMatch(&queryParam); err != nil {
            return fmt.Errorf("invalid query parameter match at index %d: %w", i, err)
        }
    }
    
    return nil
}

func validateQueryParamMatch(queryParam *v1alpha1.QueryParamMatch) error {
    if queryParam.Name == "" {
        return fmt.Errorf("query parameter name cannot be empty")
    }
    
    // 检查 present 和 value 不能同时设置
    hasPresent := queryParam.Present != nil
    hasValue := queryParam.Value != nil
    
    if hasPresent && hasValue {
        return fmt.Errorf("cannot specify both 'present' and 'value' for query parameter '%s'", queryParam.Name)
    }
    
    // 验证 value 匹配
    if hasValue {
        if err := validateStringMatch(queryParam.Value); err != nil {
            return fmt.Errorf("invalid value match for query parameter '%s': %w", queryParam.Name, err)
        }
    }
    
    return nil
}

func validateStringMatch(stringMatch *v1alpha1.StringMatch) error {
    matchCount := 0
    
    if stringMatch.Exact != "" {
        matchCount++
    }
    if stringMatch.Prefix != "" {
        matchCount++
    }
    if stringMatch.Suffix != "" {
        matchCount++
    }
    if stringMatch.Regex != "" {
        matchCount++
    }
    
    if matchCount == 0 {
        return fmt.Errorf("at least one match type must be specified")
    }
    
    if matchCount > 1 {
        return fmt.Errorf("only one match type can be specified")
    }
    
    // 验证正则表达式语法
    if stringMatch.Regex != "" {
        if _, err := regexp.Compile(stringMatch.Regex); err != nil {
            return fmt.Errorf("invalid regex pattern '%s': %w", stringMatch.Regex, err)
        }
    }
    
    return nil
}
```

#### 步骤 6: 添加测试用例

**文件**: `wasm/src/matcher_test.rs`

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_query_param_exact_match() {
        let match_config = Match {
            query_params: Some(vec![QueryParamMatch {
                name: "version".to_string(),
                value: Some(StringMatch {
                    exact: Some("v1".to_string()),
                    prefix: None,
                    suffix: None,
                    regex: None,
                }),
                present: None,
                invert: false,
            }]),
            ..Default::default()
        };
        
        let request_info = RequestInfo {
            method: "GET".to_string(),
            path: "/api/users".to_string(),
            query_string: Some("version=v1&limit=10".to_string()),
            headers: std::collections::HashMap::new(),
            body: None,
            source_ip: None,
        };
        
        let matcher = RequestMatcher::new(match_config);
        assert!(matcher.matches_query_params(&request_info));
    }

    #[test]
    fn test_query_param_present_check() {
        let match_config = Match {
            query_params: Some(vec![QueryParamMatch {
                name: "debug".to_string(),
                value: None,
                present: Some(true),
                invert: false,
            }]),
            ..Default::default()
        };
        
        let request_info = RequestInfo {
            method: "GET".to_string(),
            path: "/api/test".to_string(),
            query_string: Some("debug&version=v1".to_string()),
            headers: std::collections::HashMap::new(),
            body: None,
            source_ip: None,
        };
        
        let matcher = RequestMatcher::new(match_config);
        assert!(matcher.matches_query_params(&request_info));
    }

    #[test]
    fn test_query_param_invert_match() {
        let match_config = Match {
            query_params: Some(vec![QueryParamMatch {
                name: "production".to_string(),
                value: None,
                present: Some(true),
                invert: true, // 要求 production 参数不存在
            }]),
            ..Default::default()
        };
        
        let request_info = RequestInfo {
            method: "GET".to_string(),
            path: "/api/test".to_string(),
            query_string: Some("debug=true".to_string()),
            headers: std::collections::HashMap::new(),
            body: None,
            source_ip: None,
        };
        
        let matcher = RequestMatcher::new(match_config);
        assert!(matcher.matches_query_params(&request_info)); // production 不存在，匹配成功
    }
}
```

#### 步骤 7: 更新配置示例

**文件**: `examples/query-param-matching-example.yaml`

```yaml
apiVersion: hfi.io/v1alpha1
kind: FaultInjectionPolicy
metadata:
  name: query-param-demo
  namespace: default
spec:
  priority: 100
  enabled: true
  match:
    path:
      prefix: "/api/"
    queryParams:
      # 匹配 version=beta 的请求
      - name: "version"
        value:
          exact: "beta"
      # 要求存在 debug 参数（不关心值）
      - name: "debug"
        present: true
      # 要求不存在 production 参数
      - name: "production"
        present: true
        invert: true
      # 匹配 user_id 以 test_ 开头的请求
      - name: "user_id"
        value:
          prefix: "test_"
  fault:
    delay:
      percentage: 50.0
      fixedDelay: "1s"
```

## 🧪 测试指南

### 单元测试

#### Go 测试

```bash
# 运行所有 Go 测试
go test ./...

# 运行特定包的测试
go test ./control-plane/internal/handlers

# 运行测试并生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

#### Rust 测试

```bash
# 运行所有 Rust 测试
cargo test

# 运行特定模块的测试
cargo test matcher

# 运行测试并显示输出
cargo test -- --nocapture
```

### 集成测试

```bash
# 启动测试环境
make test-env

# 运行端到端测试
make test-e2e

# 清理测试环境
make test-clean
```

### 性能测试

```bash
# 运行性能基准测试
make benchmark

# 压力测试
make load-test
```

## 📚 文档贡献

### 文档类型

1. **API 文档**: 更新 `API_REFERENCE.md`
2. **用户指南**: 更新 `README.md` 和相关教程
3. **开发文档**: 更新架构设计和开发指南
4. **示例代码**: 添加到 `examples/` 目录

### 文档规范

- 使用 Markdown 格式
- 包含代码示例
- 添加适当的图表和流程图
- 保持链接的有效性
- 遵循一致的格式和风格

## 🤝 社区准则

### 行为准则

- 保持友善和专业
- 尊重不同的观点和经验水平
- 建设性地给出反馈
- 专注于对项目有益的讨论

### 提问指南

在提出问题时，请提供：

1. **明确的问题描述**
2. **重现步骤**
3. **预期行为 vs 实际行为**
4. **环境信息**（操作系统、版本等）
5. **相关日志或错误信息**

### Issue 和 PR 模板

我们提供了标准化的模板：

- **Bug Report**: 报告系统缺陷
- **Feature Request**: 请求新功能
- **Documentation**: 文档改进
- **Performance**: 性能优化

## 🚀 发布流程

### 版本规范

我们使用 [Semantic Versioning](https://semver.org/)：

- **MAJOR**: 不兼容的 API 变更
- **MINOR**: 向后兼容的功能添加
- **PATCH**: 向后兼容的问题修复

### 发布步骤

1. **创建发布分支**
   ```bash
   git checkout -b release/v1.2.0
   ```

2. **更新版本号**
   ```bash
   # 更新相关文件中的版本号
   vim VERSION
   vim control-plane/cmd/root.go
   vim wasm/Cargo.toml
   ```

3. **更新 CHANGELOG**
   ```bash
   vim CHANGELOG.md
   ```

4. **创建 Release PR**
5. **合并到 main 分支**
6. **创建 Git Tag**
   ```bash
   git tag -a v1.2.0 -m "Release v1.2.0"
   git push origin v1.2.0
   ```

7. **构建和发布制品**

## 🔍 故障排除

### 常见开发问题

#### WASM 编译问题

```bash
# 清理并重新构建
cargo clean
make wasm-build

# 检查依赖
rustup target list --installed
```

#### Go 模块问题

```bash
# 清理模块缓存
go clean -modcache

# 更新依赖
go mod tidy
```

#### Docker 问题

```bash
# 清理 Docker 环境
docker system prune -f

# 重新构建镜像
make docker-rebuild
```

### 调试技巧

#### WASM 插件调试

```rust
// 在 WASM 代码中添加日志
log::info!("Debug info: {:?}", some_variable);
log::error!("Error occurred: {}", error_message);
```

```bash
# 查看 Envoy 日志
docker logs envoy-proxy 2>&1 | grep -i fault
```

#### Control Plane 调试

```go
// 在 Go 代码中添加日志
log.Printf("Debug info: %+v", someStruct)
log.Error("Error occurred", "error", err)
```

```bash
# 查看 Control Plane 日志
docker logs control-plane 2>&1 | grep -i error
```

---

**感谢您对 HFI 项目的贡献！** 🙏

如果您有任何问题，请随时通过 Issue 或讨论区联系我们。我们期待与您一起建设更好的故障注入平台！

**相关链接**:
- [项目 README](../../README.md)
- [API 参考文档](API_REFERENCE.md)
- [架构设计文档](../design_doc/ARCHITECTURE.md)
- [快速开始指南](../QUICKSTART.md)
