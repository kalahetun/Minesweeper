# 测试架构指南 - Executor 项目

**文档版本**: 1.0  
**最后更新**: 2025-11-14  
**应用于**: executor 项目（Control Plane + CLI + Wasm Plugin）

---

## 📋 目录结构规范

### 标准化三层测试结构

所有三个组件遵循统一的测试组织方式：

```
component/
├── src/ (或主代码目录)
│   ├── main.rs / main.go
│   └── ... (业务代码)
│
└── tests/
    ├── unit/                 # 单元测试
    │   ├── module1_test.go
    │   ├── module2_test.go
    │   └── feature_test.rs
    │
    ├── integration/          # 集成测试
    │   ├── api_test.go
    │   ├── workflow_test.rs
    │   └── multi_component_test.go
    │
    ├── e2e/ (可选)          # 端到端测试
    │   ├── full_flow_test.go
    │   └── scenario_test.rs
    │
    ├── benchmarks/ (可选)    # 性能基准测试
    │   ├── matcher_bench.rs
    │   └── service_bench_test.go
    │
    └── fixtures/             # 测试数据和夹具
        ├── policies.go
        ├── policies.rs
        └── sample_policies/
```

### 各层说明

#### 单元测试 (unit/)
- **目的**: 测试单个函数、方法或模块的逻辑
- **范围**: 无外部依赖，只测试一个功能单元
- **示例**:
  - `matcher_test.rs`: 测试正则表达式匹配
  - `validator_test.go`: 测试 Policy 验证规则
  - `client_test.go`: 测试 HTTP 客户端

#### 集成测试 (integration/)
- **目的**: 测试多个组件的交互
- **范围**: 可能涉及数据库、网络、文件系统
- **示例**:
  - `policy_service_test.go`: 测试 Service → Storage → API 的交互
  - `multi_rules_test.rs`: 测试多个规则的并发处理
  - `sse_distribution_test.go`: 测试 Control Plane → Plugin 的策略分发

#### 端到端测试 (e2e/)
- **目的**: 验证完整的业务流程
- **范围**: 从用户输入到最终输出的完整链路
- **示例**:
  - `complete_workflow_test.go`: CLI apply → CP API → Plugin 执行 → 验证故障
  - `policy_update_test.rs`: 策略更新 → 规则变化 → 请求处理

#### 性能基准测试 (benchmarks/)
- **目的**: 测量和跟踪性能指标
- **范围**: 关键热路径的性能
- **示例**:
  - `matcher_bench.rs`: 单规则和 10 规则的匹配性能
  - `policy_service_bench_test.go`: CRUD 操作的性能

---

## 🏷️ 命名约定

### 文件命名

| 类型 | 语言 | 约定 | 示例 |
|------|------|------|------|
| 单元测试 | Go | `{module}_test.go` | `policy_service_test.go` |
| 单元测试 | Rust | `{module}_test.rs` | `matcher_test.rs` |
| 集成测试 | Go | `{feature}_test.go` | `policy_lifecycle_test.go` |
| 集成测试 | Rust | `{feature}_test.rs` | `multi_rules_test.rs` |
| 基准测试 | Go | `{module}_bench_test.go` | `policy_service_bench_test.go` |
| 基准测试 | Rust | `{module}_bench.rs` | `matcher_bench.rs` |

### 测试函数命名

**Go**:
```go
func TestPolicyServiceCreate(t *testing.T) { }
func TestPolicyServiceUpdate(t *testing.T) { }
func BenchmarkPolicyServiceCreate(b *testing.B) { }
```

**Rust**:
```rust
#[test]
fn test_matcher_exact_path() { }

#[bench]
fn bench_matcher_10_rules(b: &mut Bencher) { }
```

### 测试用例命名

遵循 BDD 风格的描述：

```
Test{Feature}_{Scenario}_{Expected}

示例:
- TestPolicyValidation_WithMissingName_ReturnsError
- TestMatcherRule_WithRegexPath_MatchesCorrectly
- TestSSEDistribution_With10Clients_PropagatesInUnderOneSecond
```

---

## ✅ 运行测试的方式

### Control Plane

```bash
cd executor/control-plane

# 运行所有单元测试
make test

# 运行指定的测试文件
go test -v ./tests/unit/policy_service_test.go

# 运行指定的测试函数
go test -v -run TestPolicyCreate ./tests/unit/...

# 生成覆盖率报告
make test-coverage

# 运行集成测试
make test-integ

# 运行基准测试
make bench
```

### CLI

```bash
cd executor/cli

# 运行所有测试
make test

# 生成覆盖率报告
make test-coverage

# 构建 CLI
make build

# 运行 CLI
make run
```

### Wasm Plugin

```bash
cd executor/wasm-plugin

# 运行所有测试
make test

# 运行所有测试（包括集成和 E2E）
make test-all

# 生成覆盖率报告
make test-coverage

# 运行基准测试
make bench

# 构建 WASM 二进制
make build
```

### 运行所有组件的测试

```bash
cd executor

# 在根目录运行所有测试
make test-all
```

---

## 📊 覆盖率要求

根据项目宪法要求：

| 部分 | 目标覆盖率 |
|------|----------|
| 核心业务逻辑 | > 90% |
| 关键路径 | > 90% |
| 一般模块 | > 70% |
| 全局平均 | > 70% |

### 生成覆盖率报告

```bash
# Go 项目
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Rust 项目
cargo tarpaulin --out Html --output-dir coverage
```

---

## 🔄 测试执行流程

### 开发循环

```
1. 编写失败的测试 (TDD)
2. 编写最小化代码使测试通过
3. 运行完整测试套件验证
4. 重构代码保持测试通过
5. 提交 PR 前，确保所有测试通过
```

### CI/CD 集成

测试应在以下阶段执行：

1. **快速反馈** (每次提交):
   - 单元测试 (< 1 分钟)
   - Lint 检查

2. **完整验证** (PR 合并前):
   - 单元测试 + 集成测试 (< 5 分钟)
   - 覆盖率检查 (>70%)
   - 性能基准对比

3. **完整套件** (定期/发布前):
   - 所有测试
   - E2E 测试
   - 性能基准
   - 文档构建

---

## 🧪 测试数据和夹具

### 预定义的 Policy 对象

使用 `tests/fixtures/` 中的预定义对象：

**Go (Control Plane)**:
```go
import "executor/control-plane/tests/fixtures"

policy := fixtures.SampleAbortPolicy("test-abort")
policy := fixtures.SampleDelayPolicy("test-delay")
policy := fixtures.SampleTimedPolicy("test-timed", 60)
```

**Rust (Wasm Plugin)**:
```rust
mod fixtures { include!("../../tests/fixtures/policies.rs"); }

let policy = fixtures::sample_abort_policy("test-abort");
let policy = fixtures::multi_rule_policy("test-rules");
```

**YAML (CLI)**:
```bash
hfi-cli policy apply -f cli/tests/fixtures/sample_policies/abort-policy.yaml
```

### 创建自定义测试数据

如果需要额外的测试数据，创建新的夹具函数：

```go
// tests/fixtures/policies.go
func CustomPolicy(scenario string) map[string]interface{} {
    // 返回针对特定场景的 Policy
}
```

---

## 🐛 调试失败的测试

### Go 测试调试

```bash
# 详细输出
go test -v -run TestName ./...

# 输出 panic 堆栈跟踪
go test -v -run TestName ./... 2>&1 | head -100

# 跳过测试（调试时临时使用）
// 在测试函数开头添加
t.Skip("调试中")
```

### Rust 测试调试

```bash
# 详细输出
cargo test -- --nocapture

# 单线程运行（避免并发问题）
cargo test -- --test-threads=1 --nocapture

# 输出 backtrace
RUST_BACKTRACE=1 cargo test
```

---

## 📈 性能基准指南

### 建立基准

```bash
# 首次运行建立基准
make bench > baseline.txt

# 定期运行检查回归
make bench > current.txt
diff baseline.txt current.txt
```

### 性能目标（来自规范）

| 组件 | 操作 | 目标 | 验证方法 |
|------|------|------|---------|
| Matcher | 单规则匹配 | < 0.5ms | `matcher_bench.rs` |
| Executor | Abort/Delay | < 0.3ms | `executor_bench.rs` |
| Policy Service | CRUD | < 50ms | `policy_service_bench_test.go` |
| Plugin | 1000 req/sec | p99 < 1ms | `load_test.rs` |

---

## 🚀 快速启动

### 添加新的单元测试

1. 在 `tests/unit/` 中创建文件：`feature_test.go`
2. 导入测试框架：
   ```go
   import "testing"
   func TestFeature(t *testing.T) { }
   ```
3. 运行测试：`make test`

### 添加新的集成测试

1. 在 `tests/integration/` 中创建文件：`workflow_test.go`
2. 设置测试环境（如启动临时 DB）
3. 运行测试：`make test-integ`

### 添加性能基准

1. 在 `tests/benchmarks/` 中创建文件：`feature_bench.rs` 或 `feature_bench_test.go`
2. 实现基准函数
3. 运行：`make bench`

---

## 📝 最佳实践

1. **隔离性**: 每个测试独立运行，不依赖其他测试
2. **确定性**: 测试应总是产生相同结果
3. **清晰性**: 测试名称应清楚地说明测试的内容
4. **完整性**: 不仅测试成功路径，也测试失败路径和边界情况
5. **性能**: 测试应快速运行（单个测试 < 1s）
6. **文档**: 复杂的测试应有注释说明意图

---

**最后更新**: 2025-11-14  
**负责人**: 开发团队  
**下一步**: 参考 `quickstart.md` 快速启动测试编写
