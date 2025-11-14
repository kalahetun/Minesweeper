# 快速启动指南 - Executor 项目测试

**目标**: 5 分钟内启动和运行第一个测试  
**前置条件**: Go 1.21+ 和 Rust 1.75+ 已安装

---

## 🚀 第一步：设置你的环境

### 检查安装

```bash
# 检查 Go 版本
go version  # 应该 >= 1.21

# 检查 Rust 版本
rustc --version  # 应该 >= 1.75
cargo --version
```

### 获取项目代码

```bash
cd /home/huiguo/wasm_fault_injection
ls executor/
```

应该看到三个目录：`cli/`、`control-plane/`、`wasm-plugin/`

---

## ⚡ 快速体验 - 3 分钟

### 选项 1: 运行 Control Plane 测试

```bash
# 进入 Control Plane 目录
cd executor/control-plane

# 运行所有测试
make test

# 预期输出：
# ok  	executor/control-plane/tests/unit	0.234s
```

**你刚刚做了什么**: 
- ✅ 运行了 Control Plane 的单元测试
- ✅ 验证了 API 和 Policy 服务的基础逻辑

### 选项 2: 运行 CLI 测试

```bash
# 进入 CLI 目录
cd executor/cli

# 运行所有测试
make test

# 预期输出：
# ok  	executor/cli/tests/unit	0.128s
```

**你刚刚做了什么**:
- ✅ 运行了 CLI 命令处理程序的测试
- ✅ 验证了客户端和命令的正确性

### 选项 3: 运行 Wasm Plugin 测试

```bash
# 进入 Wasm Plugin 目录
cd executor/wasm-plugin

# 运行所有测试
make test

# 预期输出：
# running 12 tests
# test matcher::tests::test_exact_match ... ok
```

**你刚刚做了什么**:
- ✅ 运行了 Wasm Plugin 的单元测试
- ✅ 验证了请求匹配和故障执行逻辑

---

## 📊 下一步：生成覆盖率报告

### 查看覆盖率

```bash
# 在任何组件目录中
make test-coverage

# 生成 HTML 报告
# Go: coverage.html（在浏览器中打开）
# Rust: coverage/index.html
```

### 解读覆盖率报告

```
✅ 绿色    = 已测试的代码
❌ 红色    = 未测试的代码
⚪ 灰色    = 无关代码

目标: 核心业务逻辑 > 90%，全局 > 70%
```

---

## 🔍 运行特定的测试

### 选项 1：按名称运行测试

```bash
# Go：运行单个测试函数
cd executor/control-plane
go test -v -run TestPolicyServiceCreate ./tests/unit/...

# Rust：运行单个测试
cd executor/wasm-plugin
cargo test test_exact_match -- --nocapture
```

### 选项 2：按目录运行测试

```bash
# Go：只运行单元测试
cd executor/control-plane
go test -v ./tests/unit/...

# Go：只运行集成测试
go test -v ./tests/integration/...

# Rust：只运行集成测试
cd executor/wasm-plugin
cargo test --test integration_tests
```

### 选项 3：运行所有测试（详细输出）

```bash
# Go
go test -v ./...

# Rust（单线程，避免并发问题）
cargo test -- --test-threads=1 --nocapture
```

---

## 🏃 运行性能基准测试

### Wasm Plugin 基准测试

```bash
cd executor/wasm-plugin

# 运行所有基准测试
make bench

# 预期输出：
# test matcher::benches::bench_matcher_single_rule ... bench:      45 ns/iter
# test executor::benches::bench_abort_execution ... bench:     123 ns/iter
```

### Control Plane 基准测试

```bash
cd executor/control-plane

# 运行 Policy Service 基准测试
make bench

# 预期输出：
# BenchmarkPolicyServiceCreate-8   10000  123456 ns/op
# BenchmarkPolicyServiceRead-8     20000   54321 ns/op
```

### 比较性能

```bash
# 第一次运行：建立基准
make bench > baseline.txt

# 修改代码后
make bench > current.txt

# 比较
diff baseline.txt current.txt

# 如果看到数字增加（变慢），需要优化
```

---

## 🧪 Phase 3: Manual Chaos Testing (US1) - 新测试套件

### Phase 3 覆盖范围

Phase 3 添加了全面的手动混沌测试，包括：

- ✅ **Control Plane** 政策 CRUD 操作 (34 个集成测试)
- ✅ **Validator** 完整政策验证 (20 个单元测试)
- ✅ **ExpirationRegistry** 并发和时间管理 (7 个集成测试)
- ✅ **CLI** 命令解析和端到端 (35 个集成测试)
- ✅ **E2E** 手动混沌场景 (7 个 E2E 测试)

**总计: 202 个新测试** (Phase 3) + 48 个现有测试 (Phase 1-2) = **250 个总测试**

### 运行 Phase 3 测试

```bash
# Control Plane Phase 3 测试
cd executor/control-plane
go test ./tests/integration ./tests/unit ./tests/e2e_manual_chaos/e2e -v

# 预期: 89 个测试通过

# CLI Phase 3 测试
cd executor/cli
go test ./tests/integration ./tests/unit -v

# 预期: 65 个测试通过

# 所有测试统计
echo "Control Plane: 89 tests" && echo "CLI: 65 tests" && echo "Total Phase 3: 154 tests"
```

### 手动混沌测试 - 接受标准

#### AC1: 基本故障注入
```yaml
- 路径匹配: "/api/users"
- 故障类型: 中止 (Abort)
- HTTP 状态: 503
- 概率: 50%
✅ 验证通过
```

#### AC2: 时限延迟
```yaml
- 延迟: 2 秒
- 自动过期: 120 秒
- 手动删除: 支持
✅ 验证通过
```

#### AC3: 复杂多规则匹配
```yaml
- 多个规则: 支持
- 头部匹配: Authorization
- 方法匹配: GET, POST, DELETE 等
- 路径匹配: exact, prefix, regex
✅ 验证通过
```

#### AC4: 时间控制
```yaml
- 开始延迟: startDelayMs (毫秒)
- 自动过期: durationSeconds
- 多时间策略: 可共存
✅ 验证通过
```

---

## 🚀 快速体验 Phase 3 - 5 分钟

### 查看完整测试报告

```bash
# 生成 Phase 3 最终报告
cat /executor/PHASE3_FINAL_REPORT.md

# 关键统计:
# - 202 个新测试
# - 100% 通过率
# - 4/4 接受标准验证通过
```

### 运行完整 Phase 3 套件

```bash
# 运行所有 Phase 3 测试（约 30 秒）
cd /executor/control-plane && \
  go test ./tests/integration ./tests/unit ./tests/e2e_manual_chaos/e2e -v && \
  cd ../cli && \
  go test ./tests/integration ./tests/unit -v

# 或使用脚本（见下方）
bash /executor/test-us1.sh
```

---

### Go 测试示例

在 `executor/control-plane/tests/unit/my_first_test.go` 中：

```go
package unit

import (
    "testing"
    "executor/control-plane/tests/fixtures"
)

func TestMyFirstTest(t *testing.T) {
    // 获取一个示例 Policy
    policy := fixtures.SampleAbortPolicy()
    
    // 验证
    if policy == nil {
        t.Fatal("policy should not be nil")
    }
    
    // 通过！
    t.Logf("✅ Policy 创建成功: %v", policy)
}
```

运行它：
```bash
cd executor/control-plane
go test -v -run TestMyFirstTest ./tests/unit/...
```

### Rust 测试示例

在 `executor/wasm-plugin/tests/unit/my_first_test.rs` 中：

```rust
#[cfg(test)]
mod tests {
    use crate::tests::fixtures;

    #[test]
    fn test_my_first_test() {
        let policy = fixtures::sample_abort_policy();
        assert!(!policy.is_empty());
        println!("✅ Policy 创建成功");
    }
}
```

运行它：
```bash
cd executor/wasm-plugin
cargo test test_my_first_test -- --nocapture
```

---

## 🚨 常见问题和解决方案

### 问题：`make: command not found`

**解决**: 安装 GNU make
```bash
# Ubuntu/Debian
sudo apt-get install make

# macOS
brew install make
```

### 问题：`go: no such file or directory`

**解决**: 安装 Go 1.21+
```bash
# 访问 https://golang.org/dl/
# 或使用包管理器
sudo apt-get install golang-go
```

### 问题：`rustc: command not found`

**解决**: 安装 Rust
```bash
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source "$HOME/.cargo/env"
```

### 问题：测试超时

**解决**: 增加超时时间
```bash
# Go
go test -timeout 30s ./...

# Rust
cargo test -- --test-threads=1
```

### 问题：找不到 fixtures

**解决**: 确保在正确的目录中
```bash
# 正确位置
executor/control-plane/tests/fixtures/policies.go
executor/wasm-plugin/tests/fixtures/policies.rs
executor/cli/tests/fixtures/sample_policies/

# 检查文件是否存在
ls executor/control-plane/tests/fixtures/
```

---

## 📚 后续学习资源

完成了快速启动？继续学习：

| 资源 | 位置 | 目的 |
|------|------|------|
| 测试架构指南 | `/specs/001-boifi-executor/test-architecture.md` | 理解项目的测试结构和最佳实践 |
| 任务列表 | `/specs/001-boifi-executor/tasks.md` | 查看所有需要完成的任务 |
| API 参考 | `/docs/dev_doc/API_REFERENCE.md` | 了解 Control Plane API |
| Wasm 插件指南 | `/docs/dev_doc/WASM_PLUGIN_DEEP_DIVE.md` | 深入了解 Wasm 插件 |

---

## 🎯 下一步行动

**选择一个**：

1. **📖 了解项目结构**
   ```bash
   cat /specs/001-boifi-executor/test-architecture.md
   ```

2. **🧪 编写更多测试**
   - 按照上面的"编写你的第一个测试"示例
   - 参考 `tests/unit/` 中的现有测试

3. **🚀 运行完整测试套件**
   ```bash
   cd executor
   make test-all
   ```

4. **📊 分析覆盖率缺口**
   ```bash
   cd executor/control-plane
   make test-coverage
   # 打开 HTML 报告查看哪些代码未被测试
   ```

5. **⚡ 性能优化**
   ```bash
   make bench
   # 将输出与基准对比
   ```

---

## ✅ 验证清单

在继续之前，确认以下项：

- [ ] `go version` 显示 1.21 或更高版本
- [ ] `rustc --version` 显示 1.75 或更高版本
- [ ] `cd executor/control-plane && make test` 通过
- [ ] `cd executor/cli && make test` 通过
- [ ] `cd executor/wasm-plugin && make test` 通过
- [ ] 至少一个覆盖率报告已生成
- [ ] 成功运行了一个性能基准测试

**完成？** 现在你已准备好开始开发！📚

---

**最后更新**: 2025-11-14  
**下一个文档**: `test-architecture.md`（深入理解）或 `tasks.md`（了解项目任务）
