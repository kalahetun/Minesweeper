# **项目开发计划：Wasm-based Injection Executor**

**项目代号**: `Project HFI` (Holistic Fault Injector)
**核心团队**: 1-2 名工程师
**技术栈**: Go, Wasm (proxy-wasm-go-sdk), Cobra, Gin, etcd/Memory, Docker, Kubernetes, Envoy

---

## **第一阶段：原型验证 (MVP-Core) - "让它跑起来"**

**目标**: 快速打通核心数据流，验证技术可行性，建立开发信心。
**预计时间**: 1.5 周

| 任务 ID | 任务模块        | 任务描述                                                                                                      | 关键技术点/产出                                                                  | 依赖任务 |
| :------ | :-------------- | :------------------------------------------------------------------------------------------------------------ | :------------------------------------------------------------------------------- | :------- |
| **C-1** | **Control Plane** | **搭建基础框架**: 使用 Gin 搭建 HTTP 服务器，定义 `/v1/policies` 和 `/v1/config/stream` 两个空路由。            | `gin.Engine`, 路由定义                                                           | -        |
| **C-2** | **Control Plane** | **实现内存存储 (`memoryStore`)**: 实现 `IPolicyStore` 接口的内存版本，支持基本的增删改查和基于 channel 的 `Watch`。 | `map`, `sync.RWMutex`, `chan`                                                    | C-1      |
| **C-3** | **Control Plane** | **实现核心逻辑 (简版)**: 串联 API Handler -> Policy Service -> `memoryStore` -> Config Distributor 的基本流程。 | 控制器、服务层、分发器的骨架代码                                                 | C-2      |
| **W-1** | **Wasm Plugin** | **搭建基础框架**: 使用 `proxy-wasm-go-sdk` 创建一个最简插件，实现 `proxy_on_...` 生命周期函数。                 | `proxy-wasm-go-sdk` 初始化                                                       | -        |
| **W-2** | **Wasm Plugin** | **实现简版 `Config Subscriber`**: 启动一个 goroutine，能连接到 Control Plane 的 SSE 端点，并打印收到的数据。     | `net/http` 客户端, SSE 流读取                                                    | C-1      |
| **I-1** | **集成与环境**  | **搭建 Docker & Envoy 环境**: 编写 Dockerfile 用于编译 Wasm 插件，并配置一个简单的 `envoy.yaml` 来加载它。      | Docker, Envoy 配置                                                               | -        |
| **T-1** | **端到端测试**  | **手动测试数据流**: 通过 `curl` 向 Control Plane `POST` 一个策略，观察 Wasm 插件的日志是否打印出收到的配置。 | 手动验证，确保核心链路畅通                                                       | C-3, W-2 |

**第一阶段可交付成果**:
*   一个可以运行的 Control Plane 原型，策略存储在内存中。
*   一个可以加载到 Envoy 中的 Wasm 插件原型，能接收并打印来自 Control Plane 的配置。
*   一套用于本地开发和测试的 Docker & Envoy 环境。

---

下面我为你准备了**第一阶段 (原型验证 - "让它跑起来")** 的全套 Prompt。这些 Prompt 基于你的详细设计文档，旨在引导 AI 生成每个任务模块的核心骨架代码。

### **使用说明**

1.  **分步进行**: 严格按照任务 ID 的顺序，一次向 AI 提供一个 Prompt。不要试图一次性把所有任务都丢给它。
2.  **提供上下文**: 在开始第一个 Prompt (C-1) 之前，可以先给 AI 一个**总体上下文**，这将极大地提升后续生成代码的连贯性和准确性。
3.  **代码审查与迭代**: AI 生成的代码是“草稿”，而不是“成品”。你需要仔细审查、理解、重构并测试它，确保其符合你的设计规范和质量要求。

---

### **总体上下文 Prompt (在开始 C-1 之前)**

> **[System]**
> 我正在开发一个名为 `hfi` (Holistic Fault Injector) 的云原生故障注入系统。该系统由一个用 Go 编写的 **Control Plane** 和一个用 Go (proxy-wasm-go-sdk) 编写的 **Wasm Plugin** 组成。
>
> 核心思想是：
> 1.  用户通过 CLI 或 API 向 **Control Plane** 提交故障注入策略。
> 2.  **Control Plane** 存储这些策略，并将它们编译成一个统一的规则集。
> 3.  **Wasm Plugin** 作为 Envoy 的插件，与 **Control Plane** 建立长连接（使用 Server-Sent Events, SSE），实时接收这个规则集。
> 4.  **Wasm Plugin** 根据收到的规则，在网络请求层执行故障注入（如 Abort, Delay）。
>
> 接下来，我将分步请求你为我生成**第一阶段（原型验证）** 的代码。请确保你的代码风格简洁、符合 Go 的最佳实践，并遵循我后续提供的具体设计要求。我们的目标是快速搭建一个能跑通核心数据流的原型。

---

### **任务 C-1: Control Plane - 搭建基础框架**

> **[User]**
> **任务 C-1**: 为 `hfi` 的 Control Plane 搭建基础 HTTP 服务器框架。
>
> **要求**:
> 1.  使用 Go 语言和 `gin` 框架。
> 2.  创建一个 `main.go` 文件作为程序入口。
> 3.  在 `main` 函数中，初始化一个 `gin` 引擎。
> 4.  定义两个路由组 `v1`：
>     *   `/v1/policies`: 为后续的策略管理预留，现在可以留空或返回一个占位符。
>     *   `/v1/config/stream`: 这是为 Wasm 插件准备的 SSE (Server-Sent Events) 端点。
> 5.  实现 `/v1/config/stream` 的处理器：
>     *   该处理器需要设置正确的 SSE 响应头 (`Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`)。
>     *   在一个循环中，每隔 5 秒向客户端发送一个简单的 "ping" 事件，格式为 `event: ping\ndata: {"time": "..."}\n\n`。
>     *   要能处理客户端断开连接的情况，并优雅地退出循环。
> 6.  服务器监听在 `0.0.0.0:8080`。

---

### **任务 C-2: Control Plane - 实现内存存储 (`memoryStore`)**

> **[User]**
> **任务 C-2**: 为 Control Plane 实现一个基于内存的 `IPolicyStore`。
>
> **要求**:
> 1.  创建一个 `storage` 包。
> 2.  在 `storage/types.go` 中定义核心数据结构 `FaultInjectionPolicy`。现在可以简化它，只需要 `metadata.name` 和一个 `spec` 字段即可。
> 3.  在 `storage/store.go` 中，定义 `IPolicyStore` 接口，应包含以下方法：
>     *   `CreateOrUpdate(policy *FaultInjectionPolicy) error`
>     *   `Get(name string) (*FaultInjectionPolicy, bool)`
>     *   `Delete(name string) error`
>     *   `List() []*FaultInjectionPolicy`
>     *   `Watch() <-chan WatchEvent` (关键方法)
> 4.  定义 `WatchEvent` 结构体，应包含事件类型（如 `PUT`, `DELETE`）和关联的 `FaultInjectionPolicy`。
> 5.  创建 `memoryStore.go`，实现 `IPolicyStore` 接口。
>     *   使用 `map[string]*FaultInjectionPolicy` 来存储数据。
>     *   使用 `sync.RWMutex` 保证并发安全。
>     *   `Watch()` 方法的实现是核心：
>         *   `memoryStore` 内部需要维护一个订阅者 channel 列表。
>         *   `Watch()` 方法会创建一个新的 channel 并将其添加到订阅者列表中，然后返回这个 channel。
>         *   当 `CreateOrUpdate` 或 `Delete` 被调用时，`memoryStore` 需要遍历所有订阅的 channel，并向它们发送一个 `WatchEvent`。
>         *   (可选，但推荐) 考虑如何处理 channel 阻塞和移除不再监听的订阅者。

---

### **任务 C-3: Control Plane - 实现核心逻辑 (简版)**

> **[User]**
> **任务 C-3**: 将 `memoryStore` 和 SSE 端点串联起来。
>
> **要求**:
> 1.  修改 `main.go`。在 `main` 函数中，初始化一个 `memoryStore` 实例。
> 2.  创建一个新的 `ConfigDistributor` 模块（可以先是一个简单的 `struct`）。
> 3.  `ConfigDistributor` 在初始化时，启动一个 goroutine，该 goroutine 通过调用 `store.Watch()` 来监听策略变更。
> 4.  当监听到变更事件时，`ConfigDistributor` 会调用 `store.List()` 获取所有当前的策略，然后将它们（例如，简单地序列化为 JSON 字符串）存储在一个内部变量中。
> 5.  修改 `/v1/config/stream` 的处理器：
>     *   它现在应该与 `ConfigDistributor` 交互。
>     *   当一个新客户端连接时，`ConfigDistributor` 首先将**当前的全量配置**发送给这个新客户端。
>     *   然后，`ConfigDistributor` 将这个客户端的连接（例如，一个 `chan string`）注册起来。
>     *   每当 `ConfigDistributor` 因监听到变更而生成了新的全量配置时，它会将这个新配置**广播**给所有已注册的客户端连接。
>     *   移除之前每 5 秒的 "ping" 事件，改为事件驱动的配置推送。
> 6.  实现一个简单的 `/v1/policies` `POST` 处理器，它能解析一个简化版的 `FaultInjectionPolicy` JSON，并调用 `store.CreateOrUpdate()`。

---

### **W-1 (Rust版): Wasm Plugin - 搭建基础框架**

> **[User]**
> **任务 W-1 (Rust版)**: 搭建一个基于 Rust 的 Wasm 插件基础项目。
>
> **要求**:
> 1.  使用 Rust 语言和 `proxy-wasm` Rust crate。
> 2.  初始化一个新的 Rust library 项目 (`cargo new --lib hfi_wasm_plugin`)。
> 3.  配置 `Cargo.toml`：
>     *   添加 `proxy-wasm` crate 作为依赖。
>     *   设置 crate 类型为 `cdylib`，这是编译动态库（包括 Wasm）所必需的。
> 4.  在 `src/lib.rs` 文件中：
>     *   使用 `use proxy_wasm::traits::{Context, HttpContext, RootContext};` 和 `use proxy_wasm::types::{Action, LogLevel};`。
>     *   定义一个 `PluginRootContext` 结构体，用于插件级别的状态。
>     *   定义一个 `PluginHttpContext` 结构体，用于请求级别的状态。
>     *   实现 `RootContext` trait for `PluginRootContext`。
>         *   在 `on_configure` 方法中，打印一条 "Plugin configured..." 日志。
>         *   实现 `create_http_context` 方法，返回一个新的 `PluginHttpContext` 实例。
>     *   实现 `HttpContext` trait for `PluginHttpContext`。
>         *   在 `on_http_request_headers` 方法中，打印一条 "Handling request..." 日志，然后返回 `Action::Continue`。
>     *   使用 `#[no_mangle]` 宏和 `proxy_wasm::main!` 来声明插件的入口点。

---

### **W-2 (Rust版): Wasm Plugin - 实现简版 `Config Subscriber`**

> **[User]**
> **任务 W-2 (Rust版)**: 在 Rust Wasm 插件中实现一个简版的配置订阅器。
>
> **要求**:
> 1.  在 `PluginRootContext` 结构体中添加一个字段 `control_plane_address: String`。
> 2.  在 `on_configure` 方法中，解析插件配置（一个简单的 JSON 字符串，如 `{"address": "http://..."}`），并将地址存入 `control_plane_address`。可以使用 `serde_json` crate 来解析。
> 3.  **核心逻辑**: 在 `proxy-wasm` 中，我们不能直接创建后台线程。正确的做法是利用 `RootContext` 的生命周期和异步 HTTP 调用。
>     *   在 `on_configure` 成功后，立即调用 `self.dispatch_http_call` 向 Control Plane 的 `/v1/config/stream` 端点发起第一次 HTTP GET 请求。
>     *   实现 `on_http_call_response` 方法 for `RootContext`。这个方法是 `dispatch_http_call` 的异步回调。
>     *   在 `on_http_call_response` 回调中：
>         *   检查响应是否成功。
>         *   获取响应体 (body)，并使用 `log` macro 打印出来 (e.g., `log::info!("Received config: {:?}", body_str);`)。
>         *   **为了模拟持续订阅**，在 `on_http_call_response` 的末尾，再次调用 `self.dispatch_http_call` 发起下一次连接，形成一个轮询或长轮询循环。为了避免过于频繁的请求，可以在重新调度前设置一个短暂的定时器（`self.set_tick_period` 和 `on_tick`），或者如果 Control Plane 支持，可以利用 HTTP 长轮询。对于原型阶段，收到响应后立即重新请求是可以接受的。
>
> **注意**: 这种基于回调的轮询模式是 `proxy-wasm` 中实现后台任务的标准方式，它不阻塞 Envoy 的 worker 线程。

---

### **I-1 & T-1 (Rust版): 集成环境与端到端测试**

> **[User]**
> **任务 I-1 & T-1 (Rust版)**: 准备 Rust Wasm 插件的集成环境并进行手动端到端测试。
>
> **请为我提供**:
> 1.  一个用于编译 Rust Wasm 插件的 `Dockerfile`。
>     *   该 Dockerfile 应该使用官方的 Rust 镜像 (`rust:latest`) 作为构建环境。
>     *   需要安装 `wasm32-unknown-unknown` target (`rustup target add wasm32-unknown-unknown`)。
>     *   使用 `cargo build --target wasm32-unknown-unknown --release` 命令进行编译。
>     *   最终产物是 `target/wasm32-unknown-unknown/release/hfi_wasm_plugin.wasm` 文件。
>     *   为了减小 Wasm 文件体积，可以考虑使用 `wasm-strip` 或 `wasm-opt` 进行优化。
>     *   最终镜像应该是一个轻量级镜像（如 `scratch` 或 `alpine`），只包含编译好的 `.wasm` 文件。
> 2.  一个简单的 `envoy.yaml` 配置文件。
>     *   (这部分与 Go 版本**完全相同**)
>     *   它应该定义一个 listener 监听在 `0.0.0.0:18000`。
>     *   定义一个 HTTP filter chain，其中包含 `envoy.filters.http.wasm`。
>     *   `wasm` filter 的配置需要指向本地的 `.wasm` 文件，并传递 Control Plane 的地址作为插件配置（例如，`{"address": "http://host.docker.internal:8080"}`）。
>     *   定义一个 cluster 和 route，将所有请求路由到一个简单的上游服务。
> 3.  一个 `docker-compose.yaml` 文件或一组 shell 命令，用于：
>     *   (与 Go 版本类似)
>     *   构建 Control Plane 的 Go 应用。
>     *   使用上述 `Dockerfile` 构建一个包含 `.wasm` 文件的镜像，或者直接在本地编译出 `.wasm` 文件。
>     *   启动 Control Plane 容器。
>     *   启动 Envoy 容器，并挂载 `.wasm` 文件和 `envoy.yaml`。
>
> **测试步骤说明**:
> 1.  (与 Go 版本**完全相同**)
> 2.  启动所有服务。
> 3.  观察 Wasm 插件（Envoy）的日志，看它是否成功配置并打印出 `on_configure` 的日志。
> 4.  观察 Wasm 插件的日志，看它是否成功连接到 Control Plane 并打印出首次收到的配置。
> 5.  使用 `curl -X POST -d '{"metadata": {"name": "test-policy"}}' http://localhost:8080/v1/policies` 发送一个新策略。
> 6.  再次观察 Wasm 插件的日志，确认它在下一次轮询响应中，是否收到了包含新策略的更新配置。

---

## **第二阶段：功能实现 (MVP-Feature) - "让它能用"**

**目标**: 实现 MVP 的核心功能，包括故障注入逻辑和基本的 CLI 交互。
**预计时间**: 2.5 周

| 任务 ID | 任务模块        | 任务描述                                                                                                                             | 关键技术点/产出                                                                | 依赖任务 |
| :------ | :-------------- | :----------------------------------------------------------------------------------------------------------------------------------- | :----------------------------------------------------------------------------- | :------- |
| **W-3** | **Wasm Plugin** | **实现 `Request Matcher`**: 编写 Path/Method/Header 的匹配逻辑。编写详尽的单元测试。                                                    | 字符串处理, 正则表达式（预编译）, **单元测试**                                  | W-2      |
| **W-4** | **Wasm Plugin** | **实现 `Fault Executor`**: 编写 Abort 和 Delay 的执行逻辑，调用 `proxy-wasm` SDK 的相应函数。编写单元测试。                              | `proxy.SendHttpResponse`, `proxy.SetTimer`, **单元测试**                         | W-3      |
| **W-5** | **Wasm Plugin** | **完善插件逻辑**: 将 Matcher 和 Executor 集成到 `proxy_on_http_request_headers` 中。实现 `atomic.Value` 的线程安全规则更新。               | `atomic.Value`, 完整的请求处理流程                                             | W-4      |
| **C-4** | **Control Plane** | **实现 `etcdStore`**: 实现 `IPolicyStore` 接口的 etcd 版本，包括 `Watch` 机制。                                                        | `etcd/clientv3`                                                                | C-3      |
| **CL-1**| **CLI**           | **搭建 CLI 框架**: 使用 Cobra 创建 `rootCmd` 和 `policy` 子命令。实现全局标志和 `APIClient` 初始化。                                    | `cobra`, `PersistentPreRun`                                                    | -        |
| **CL-2**| **CLI**           | **实现 `apply` 命令**: 实现 `hfi-cli policy apply -f <file>` 功能，能读取 YAML/JSON 并调用 Control Plane API。                            | 文件读取, `yaml.Unmarshal`, `APIClient`                                          | CL-1, C-3|
| **T-2** | **端到端测试**  | **功能测试**: 使用 `hfi-cli` 应用一个 Abort 策略，通过 `curl` 访问被 Envoy 代理的服务，验证请求是否被中止。测试 Delay 策略。            | 完整的端到端功能验证                                                           | W-5, C-4, CL-2|

**第二阶段可交付成果**:
*   一个功能完备的 Wasm 插件，能根据规则执行 Abort 和 Delay 故障。
*   一个功能增强的 Control Plane，支持 etcd 作为持久化存储。
*   一个可用的 CLI，能通过 `apply` 命令来管理故障注入策略。
*   **一个完整的、可演示的 MVP 系统**。

---

### **使用说明**

1.  **基于第一阶段**: 确保你已经完成了第一阶段的代码，并将它们作为上下文提供给 AI。例如，在开始 W-3 之前，你可以先贴上 `plugin/types.go` 和 `plugin/plugin.go` 的代码。
2.  **迭代和重构**: 第二阶段的任务会比第一阶段更复杂。AI 生成的代码可能需要你进行更多的思考和重构，特别是在涉及逻辑和数据结构的部分。

---

### **任务 W-3 (Rust版): Wasm Plugin - 实现 `Request Matcher`**

> **[User]**
> **任务 W-3 (Rust版)**: 为 Wasm 插件实现 `Request Matcher` 逻辑。
>
> **要求**:
> 1.  在 Rust 项目中创建一个新的模块，例如 `matcher.rs`。
> 2.  定义数据结构：
>     *   创建 `config.rs` 模块，使用 `serde` crate (with `derive(Deserialize)`) 来定义从 Control Plane 接收的配置结构体，包括 `CompiledRuleSet`, `CompiledRule`, `MatchCondition`, `PathMatcher`, `HeaderMatcher` 等。
>     *   在 `PathMatcher` 中，如果存在 `regex` 字段，需要一个非序列化的字段来存放编译后的正则表达式对象，例如 `#[serde(skip)] compiled_regex: Option<Regex>`。你需要 `regex` crate。
>     *   实现一个 `impl CompiledRuleSet` 的方法，例如 `from_slice(bytes: &[u8]) -> Result<Self, serde_json::Error>`，该方法在反序列化后，会遍历所有规则并**预编译**所有正则表达式。
> 3.  在 `matcher.rs` 中，实现核心匹配函数：
>     *   `fn find_first_match<'a>(http_context: &dyn HttpContext, rules: &'a [CompiledRule]) -> Option<&'a CompiledRule>`
>     *   该函数按顺序遍历 `rules`。
>     *   对于每个 `rule`，调用一个辅助函数 `fn is_match(http_context: &dyn HttpContext, condition: &MatchCondition) -> bool`。
>     *   `is_match` 函数需要实现对 Path (prefix, exact, regex), Method, Headers 的匹配逻辑。
>     *   匹配逻辑应遵循**短路原则**：任何一个条件不匹配，立刻返回 `false`。
>     *   为了性能，应避免在匹配逻辑中进行不必要的字符串分配（多使用 `&str`）。
> 4.  在 `src/lib.rs` 的 `PluginHttpContext` 中，调用 `find_first_match`。
> 5.  为 `matcher.rs` 中的核心匹配逻辑编写详尽的**单元测试**。使用 `#[cfg(test)]` 模块，模拟 `HttpContext` trait (如果需要) 和各种规则。

---

### **任务 W-4 (Rust版): Wasm Plugin - 实现 `Fault Executor`**

> **[User]**
> **任务 W-4 (Rust版)**: 为 Wasm 插件实现 `Fault Executor` 逻辑。
>
> **要求**:
> 1.  在 Rust 项目中创建一个新的模块，例如 `executor.rs`。
> 2.  在 `config.rs` 中，完善 `Fault` 相关的结构体定义 (`AbortAction`, `DelayAction`)，同样使用 `serde`。
>     *   对于 `DelayAction`，添加一个非序列化的 `parsed_duration: Option<Duration>` 字段，用于缓存预解析的时间。你需要 `humantime` 或类似 crate 来解析时间字符串。
>     *   在 `CompiledRuleSet::from_slice` 方法中，增加**预解析** `fixedDelay` 字符串的逻辑。
> 3.  在 `executor.rs` 中，实现核心执行函数：
>     *   `fn execute_fault(http_context: &dyn HttpContext, fault: &Fault) -> Action`
>     *   该函数首先进行**概率检查** (`Percentage`)。可以使用 `proxy_wasm::hostcalls::get_random_bytes` 或一个简单的伪随机数生成器。
>     *   如果概率命中：
>         *   如果定义了 `abort`，调用 `http_context.send_http_response()`，并返回 `Action::Pause`。
>         *   如果定义了 `delay`，调用 `http_context.set_property()` 来存储需要恢复执行的上下文信息（因为 `on_timer` 是在 `RootContext` 中触发的），然后调用 `root_context.set_timer()`。返回 `Action::Pause`。
>     *   如果概率未命中或没有定义故障动作，返回 `Action::Continue`。
> 4.  **处理 Delay**:
>     *   在 `PluginRootContext` 中实现 `on_timer` 方法。
>     *   在 `on_timer` 中，你需要一种方法来找到是哪个请求触发的这个 timer。一种常见的模式是在 `set_timer` 前，将 `context_id` (来自 `PluginHttpContext`) 存入一个 `HashMap<u32, ...>` 中。
>     *   `on_timer` 触发后，通过 token 找到对应的 `context_id`，然后调用 `proxy_wasm::hostcalls::resume_http_request(context_id)` 来恢复被暂停的请求。
> 5.  为 `executor.rs` 的概率检查和动作选择逻辑编写单元测试。

---

### **任务 W-5 (Rust版): Wasm Plugin - 完善插件逻辑**

> **[User]**
> **任务 W-5 (Rust版)**: 将 Matcher 和 Executor 完整地集成到插件的生命周期中。
>
> **要求**:
> 1.  在 `PluginRootContext` 中，使用 `Arc<RwLock<Option<CompiledRuleSet>>>` 或类似并发原语来线程安全地存储当前生效的规则集。`Arc` 允许多个 `HttpContext` 共享对规则的只读访问。
> 2.  修改 `RootContext::on_http_call_response` (来自 W-2) 的逻辑：
>     *   当收到新的配置时，调用 `CompiledRuleSet::from_slice` 进行解析和预处理（编译 regex, 解析 duration）。
>     *   如果成功，获取写锁并更新 `PluginRootContext` 中存储的规则集。
> 3.  修改 `HttpContext::on_http_request_headers` 的逻辑：
>     *   首先，从 `RootContext` 获取对规则集的读锁或快照。
>     *   如果存在规则，调用 `matcher::find_first_match`。
>     *   如果匹配到 `rule`，调用 `executor::execute_fault` 并返回其结果 (`Action::Pause` 或 `Action::Continue`)。
>     *   如果没有匹配到，或者没有规则，返回 `Action::Continue`。

---

### **任务 C-4: Control Plane - 实现 `etcdStore`**

> **[User]**
> **任务 C-4**: 为 Control Plane 实现一个基于 `etcd` 的 `IPolicyStore`。
>
> **要求**:
> 1.  在 `storage` 包中创建 `etcd_store.go`。
> 2.  定义 `EtcdStore` 结构体，它包含一个 `*clientv3.Client`。
> 3.  实现 `IPolicyStore` 接口：
>     *   `CreateOrUpdate`: 使用 etcd 的 `Txn` (事务) 来实现原子性的“创建或更新”。可以先 `Get`，如果存在则 `Put` 更新，如果不存在则 `Put` 创建。或者使用 `If` 条件。
>     *   `Get`: 使用 `client.Get`。
>     *   `Delete`: 使用 `client.Delete`。
>     *   `List`: 使用 `client.Get` 并带上 `clientv3.WithPrefix()` 选项来获取所有 key。
>     *   **`Watch` (关键)**:
>         *   创建一个 Go channel (`chan WatchEvent`)。
>         *   启动一个新的 goroutine。
>         *   在这个 goroutine 中，调用 `client.Watch` 并带上 `clientv3.WithPrefix()` 来监听所有策略 key 的变化。
>         *   在一个 `for wresp := range rch` 循环中，处理 etcd 返回的事件。
>         *   将 etcd 的 `mvccpb.Event` 转换成我们自己定义的 `WatchEvent`（`PUT` 或 `DELETE`），并发送到之前创建的 Go channel 中。
>         *   需要处理 `Watch` 的 context cancellation，以便在服务关闭时能优雅地停止这个 goroutine。

---

### **任务 CL-1: CLI - 搭建 CLI 框架**

> **[User]**
> **任务 CL-1**: 为 `hfi-cli` 搭建基础框架。
>
> **要求**:
> 1.  使用 `cobra` 库。
> 2.  创建 `cmd` 包。在 `cmd/root.go` 中，定义 `rootCmd`。
> 3.  为 `rootCmd` 添加**持久化全局标志** (Persistent Flags)：
>     *   `--control-plane-addr` (string, default: `http://localhost:8080`)
>     *   `--timeout` (duration, default: `30s`)
> 4.  实现 `rootCmd` 的 `PersistentPreRunE` 钩子：
>     *   在这个钩子中，根据 `--control-plane-addr` 和 `--timeout` 初始化一个全局可访问的 `APIClient` 实例。这个 `APIClient` 应该是一个封装了 `*http.Client` 的结构体。
> 5.  创建一个 `cmd/policy.go` 文件，定义一个 `policyCmd`，它是一个容器命令，并将其作为 `rootCmd` 的子命令添加。

---

### **任务 CL-2: CLI - 实现 `apply` 命令**

> **[User]**
> **任务 CL-2**: 为 `hfi-cli` 实现 `policy apply` 命令。
>
> **要求**:
> 1.  在 `cmd/policy_apply.go` 中创建 `applyCmd`。
> 2.  为 `applyCmd` 添加一个必需的标志：`-f, --filename` (string)。
> 3.  实现 `applyCmd` 的 `RunE` 函数：
>     *   检查 `-f` 标志是否已提供。
>     *   读取 `-f` 指定的文件内容。你需要支持 YAML 和 JSON 格式，`sigs.k8s.io/yaml` 这个库可以很好地同时处理这两种格式。
>     *   将文件内容反序列化到一个 `FaultInjectionPolicy` 结构体中。
>     *   从 `rootCmd` 获取之前初始化好的 `APIClient` 实例。
>     *   调用 `APIClient` 的 `CreateOrUpdatePolicy` 方法（你需要先实现这个方法），将反序列化后的策略发送给 Control Plane。
>     *   处理 `APIClient` 返回的错误，并打印用户友好的错误信息。
>     *   如果成功，打印一条类似 `faultinjectionpolicy.hfi.dev "..." applied` 的消息。

---

### **任务 T-2: 端到端功能测试**

> **[User]**
> **任务 T-2**: 进行第二阶段的端到端功能测试。
>
> **请为我提供一份详细的测试清单，以验证本阶段所有功能是否正常工作。**
>
> **测试环境**:
> *   Go Control Plane (连接到 etcd)
> *   etcd 实例
> *   Rust Wasm Plugin (加载到 Envoy)
> *   Envoy 代理
> *   hfi-cli 二进制文件
>
> **测试清单**:
>
> 1.  **[ ] 启动与连接**:
>     *   `[ ]` 启动 Control Plane, etcd, Envoy。
>     *   `[ ]` 检查 Control Plane 日志，确认已连接到 etcd。
>     *   `[ ]` 检查 Envoy 日志，确认 Wasm 插件已加载并开始轮询 Control Plane。
>
> 2.  **[ ] CLI `apply` Abort 策略**:
>     *   `[ ]` 编写一个 `abort-policy.yaml` 文件，匹配特定路径 (e.g., `/`)，设置为 100% 概率返回 503 状态码。
>     *   `[ ]` 执行 `hfi-cli policy apply -f abort-policy.yaml`，确认返回成功信息。
>     *   `[ ]` 检查 Control Plane 日志，确认策略已存入 etcd。
>     *   `[ ]` 检查 Envoy 日志，确认 Wasm 插件收到了包含新规则的配置。
>     *   `[ ]` 执行 `curl -v http://localhost:18000/`，**验证响应是否为 HTTP 503**。
>
> 3.  **[ ] CLI `apply` 更新策略 (Delay)**:
>     *   `[ ]` 编写一个 `delay-policy.yaml`，匹配同一路径，设置为 100% 概率延迟 2 秒。
>     *   `[ ]` 执行 `hfi-cli policy apply -f delay-policy.yaml`，更新现有策略。
>     *   `[ ]` 再次检查 Envoy 日志，确认配置已更新。
>     *   `[ ]` 执行 `time curl http://localhost:18000/`，**验证请求耗时是否约等于 2 秒**。
>
> 4.  **[ ] 概率与匹配测试**:
>     *   `[ ]` 更新策略，将概率设置为 50%。连续执行 `curl` 10 次，**验证大约一半的请求被注入故障**。
>     *   `[ ]` 更新策略，增加 Header 匹配条件 (e.g., `x-user: test`)。
>     *   `[ ]` 执行 `curl -H "x-user: test" ...`，**验证故障被注入**。
>     *   `[ ]` 执行 `curl ...` (不带 Header)，**验证请求被正常放行**。
>
> 5.  **[ ] 策略删除 (手动)**:
>     *   `[ ]` 手动使用 `etcdctl del` 删除策略。
>     *   `[ ]` 检查 Envoy 日志，**验证 Wasm 插件收到了不含该规则的空配置**。
>     *   `[ ]` 执行 `curl ...`，**验证所有请求都恢复正常**。

---

### **任务 W-3: Wasm Plugin - 实现 `Request Matcher`**

> **[User]**
> **任务 W-3**: 在 Wasm 插件中实现 `Request Matcher` 模块。
>
> **上下文**:
> Control Plane 推送的 `CompiledRuleSet` 结构体如下：
> ```go
> // a/pkg/types/types.go
> type CompiledRuleSet struct {
> 	Version string         `json:"version"`
> 	Rules   []CompiledRule `json:"rules"`
> }
>
> type CompiledRule struct {
> 	Name  string         `json:"name"`
> 	Match MatchCondition `json:"match"`
> 	Fault Fault          `json:"fault"`
> }
>
> type MatchCondition struct {
> 	Path    *PathMatcher    `json:"path,omitempty"`
> 	Method  *StringMatcher  `json:"method,omitempty"`
> 	Headers []HeaderMatcher `json:"headers,omitempty"`
> }
>
> // ... (其他 Matcher 和 Fault 结构体定义)
> ```
>
> **要求**:
> 1.  创建一个 `matcher` 包。
> 2.  在 `matcher/matcher.go` 中，创建一个 `FindFirstMatch` 函数。
>     *   函数签名: `func FindFirstMatch(reqInfo RequestInfo, rules []CompiledRule) (*CompiledRule, bool)`
> 3.  定义 `RequestInfo` 结构体，用于在匹配开始前一次性收集所有需要的请求信息。
>     *   `Path: string`
>     *   `Method: string`
>     *   `Headers: map[string]string` (简化为单值 map)
> 4.  `FindFirstMatch` 的逻辑是：
>     *   按顺序遍历 `rules` 列表。
>     *   对每个 `rule`，调用一个内部的 `isMatch(reqInfo, rule.Match)` 函数。
>     *   如果 `isMatch` 返回 `true`，则立即返回该 `rule` 和 `true`。
>     *   如果遍历完所有规则都未匹配，则返回 `nil` 和 `false`。
> 5.  实现 `isMatch` 函数的逻辑：
>     *   它必须同时满足 `Path`, `Method`, 和所有 `Headers` 的匹配条件 (AND 逻辑)。
>     *   任何一个条件未定义，则视为匹配成功。
>     *   **Path Matcher**: 支持 `prefix`, `exact`, `regex`。在**配置加载时**（`Config Subscriber` 模块）就应该预编译 `regex` 字符串为 `*regexp.Regexp` 对象，并存入 `PathMatcher` 结构体中，以避免在热路径上编译。
>     *   **Method Matcher**: 支持 `exact` 匹配。
>     *   **Header Matcher**: 支持 `exact`, `prefix`, `regex` 匹配。同样需要预编译 `regex`。
> 6.  为 `matcher` 包编写详尽的单元测试，覆盖所有匹配类型和边界情况（如规则为空，条件为空，部分匹配，完全匹配等）。

---

### **任务 W-4: Wasm Plugin - 实现 `Fault Executor`**

> **[User]**
> **任务 W-4**: 在 Wasm 插件中实现 `Fault Executor` 模块。
>
> **上下文**:
> `Fault` 结构体定义如下：
> ```go
> // a/pkg/types/types.go
> type Fault struct {
> 	Abort      *AbortAction `json:"abort,omitempty"`
> 	Delay      *DelayAction `json:"delay,omitempty"`
> 	Percentage int          `json:"percentage"`
> }
>
> type AbortAction struct {
> 	HttpStatus int    `json:"httpStatus"`
> 	Body       string `json:"body,omitempty"`
> }
>
> type DelayAction struct {
> 	FixedDelay string `json:"fixedDelay"` // e.g., "2s", "100ms"
> }
> ```
>
> **要求**:
> 1.  创建一个 `executor` 包。
> 2.  在 `executor/executor.go` 中，创建一个 `Execute` 函数。
>     *   函数签名: `func Execute(fault *Fault) types.Action`
> 3.  `Execute` 函数的逻辑：
>     *   **百分比检查**: 首先，生成一个 `0-99` 的随机数，如果它小于 `fault.Percentage`，则执行故障。否则，直接返回 `types.ActionContinue`。
>     *   如果需要执行故障：
>         *   **Abort 逻辑**: 如果 `fault.Abort` 不为 `nil`，调用 `proxy.SendHttpResponse()` 发送指定的状态码和响应体。然后返回 `types.ActionPause`。
>         *   **Delay 逻辑**: 如果 `fault.Delay` 不为 `nil`，调用 `proxy.SetTimer()`。延迟时间需要从 `fault.Delay.FixedDelay` 字符串解析。这个解析应该在**配置加载时**完成，并缓存为 `time.Duration`，避免在热路径上解析。然后返回 `types.ActionPause`。
>         *   如果 `fault` 中没有定义任何 action，记录一条警告日志并返回 `types.ActionContinue`。
> 4.  为 `executor` 包编写单元测试。由于它依赖 `proxy-wasm` 的全局函数，这可能需要一些 mock 或在测试中忽略对这些函数的调用。

---

### **任务 W-5: Wasm Plugin - 完善插件逻辑**

> **[User]**
> **任务 W-5**: 将 `Matcher` 和 `Executor` 集成到 Wasm 插件的主流程中，并实现线程安全的规则更新。
>
> **要求**:
> 1.  修改 `plugin/plugin.go`。
> 2.  在 `pluginContext` 中，使用 `atomic.Value` 来存储 `*CompiledRuleSet`。
> 3.  修改 `ConfigSubscriber` 的规则更新逻辑：
>     *   当收到新的 `CompiledRuleSet` JSON 时，反序列化它。
>     *   **关键**: 在反序列化后，立即对规则进行**预处理**：编译所有 `regex` 字符串，解析所有 `delay` 时间字符串，并将结果存回结构体中。
>     *   调用 `activeRules.Store(processedRuleSet)` 来原子地更新规则。
> 4.  修改 `proxy_on_http_request_headers` 的实现：
>     *   从 `atomic.Value` 中 `Load()` 当前的规则集。
>     *   一次性收集所有需要的请求信息到 `RequestInfo` 结构体中。
>     *   调用 `matcher.FindFirstMatch()`。
>     *   如果找到匹配的规则，调用 `executor.Execute()`，并返回其结果。
>     *   如果没有匹配，返回 `types.ActionContinue`。
> 5.  实现 `proxy_on_timer` 回调函数。当定时器触发时（用于 Delay 故障），这个函数被调用。在函数内部，调用 `proxy.ResumeHttpRequest()` 来恢复被暂停的请求。

---

### **任务 C-4: Control Plane - 实现 `etcdStore`**

> **[User]**
> **任务 C-4**: 为 Control Plane 实现一个基于 etcd 的 `IPolicyStore`。
>
> **要求**:
> 1.  在 `storage` 包中，创建 `etcd.go` 文件。
> 2.  创建一个 `etcdStore` 结构体，它需要一个 `*clientv3.Client`。
> 3.  实现 `IPolicyStore` 接口的所有方法：
>     *   `CreateOrUpdate`: 使用 etcd 的 `Put` 操作。策略对象需要序列化为 JSON 字符串。key 可以是 `hfi/policies/<policy-name>`。
>     *   `Get`: 使用 etcd 的 `Get` 操作。
>     *   `Delete`: 使用 etcd 的 `Delete` 操作。
>     *   `List`: 使用 etcd 的 `Get` 操作，并带上 `clientv3.WithPrefix()` 选项。
>     *   **`Watch` (关键)**:
>         *   在一个新的 goroutine 中，调用 `client.Watch()`，并带上 `clientv3.WithPrefix()` 来监听所有策略的变更。
>         *   当收到 etcd 的 watch event 时，将其转换为我们自己定义的 `WatchEvent`（`PUT` 或 `DELETE`）。
>         *   将转换后的 `WatchEvent` 发送到 `Watch()` 方法返回的 channel 中。
> 4.  修改 `main.go`，增加一个命令行 flag 来选择使用 `memory` 还是 `etcd` 作为存储后端。

---

### **任务 CL-1: CLI - 搭建 CLI 框架**

> **[User]**
> **任务 CL-1**: 使用 Cobra 搭建 `hfi-cli` 的基础框架。
>
> **要求**:
> 1.  创建一个 `cmd` 包。
> 2.  在 `cmd/root.go` 中，创建 `rootCmd`。
> 3.  为 `rootCmd` 添加持久化的全局标志 (Persistent Flags)：
>     *   `--control-plane-addr`: Control Plane 的地址，默认值为 `http://localhost:8080`。
>     *   `--timeout`: API 请求的超时时间，默认值为 `30s`。
> 4.  创建一个 `client` 包，并在其中定义 `APIClient` 的骨架（`struct` 和 `NewClient` 函数）。
> 5.  在 `rootCmd` 的 `PersistentPreRunE` 钩子中，根据全局标志的值来初始化 `APIClient`，并将其存储在一个包级变量或 `rootCmd` 的注解中，以便子命令可以访问。
> 6.  在 `cmd/policy.go` 中，创建一个 `policyCmd`，它是一个容器命令，并将其作为子命令添加到 `rootCmd`。

---

### **任务 CL-2: CLI - 实现 `apply` 命令**

> **[User]**
> **任务 CL-2**: 在 `hfi-cli` 中实现 `policy apply` 命令。
>
> **要求**:
> 1.  在 `cmd/apply.go` 中，创建 `applyCmd`，并将其作为 `policyCmd` 的子命令。
> 2.  为 `applyCmd` 添加一个必需的标志：`-f, --filename string`。
> 3.  实现 `applyCmd` 的 `RunE` 函数：
>     *   检查 `filename` 标志是否被提供。
>     *   从 `rootCmd` 获取共享的 `APIClient` 实例。
>     *   读取 `-f` 指定的文件内容。
>     *   使用 `yaml.Unmarshal`（`gopkg.in/yaml.v2` 或 `v3`）将文件内容解析为 `FaultInjectionPolicy` 结构体。这个库能同时处理 YAML 和 JSON。
>     *   调用 `APIClient.CreateOrUpdatePolicy()` 方法，将解析出的策略对象发送到 Control Plane。
>     *   如果 API 调用成功，打印一条成功的消息，如 `faultinjectionpolicy.hfi.dev "my-policy" applied`。
>     *   如果任何步骤失败，返回一个错误，让 Cobra 的主循环来处理打印。
> 4.  在 `client` 包中，实现 `CreateOrUpdatePolicy` 方法。它需要将策略对象序列化为 JSON，并向 Control Plane 的 `/v1/policies` 端点发送一个 `POST` 请求。

---

### **任务 T-2: 端到端功能测试**

> **[User]**
> **任务 T-2**: 准备第二阶段的端到端功能测试计划。
>
> **请为我提供一个详细的测试步骤清单，用于验证第二阶段开发的所有功能。**
>
> **测试环境**:
> *   Control Plane (使用 etcd 存储) 正在运行。
> *   Envoy (加载了 Wasm 插件) 正在运行。
> *   一个简单的上游服务（如 `httpbin`）可供 Envoy 路由。
> *   `hfi-cli` 二进制文件可用。
>
> **需要验证的场景**:
> 1.  **无故障场景**: 确认在没有任何策略的情况下，请求能正常通过 Envoy。
> 2.  **Abort 故障注入**:
>     *   如何编写一个 `policy.yaml` 文件来中止所有对 `/get` 路径的请求？
>     *   如何使用 `hfi-cli policy apply` 应用它？
>     *   如何使用 `curl` 来验证对 `/get` 的请求确实返回了指定的错误码？
>     *   如何验证对其他路径（如 `/post`）的请求不受影响？
> 3.  **Delay 故障注入**:
>     *   如何编写一个 `policy.yaml` 来对所有 `POST` 请求注入 2 秒的延迟？
>     *   如何应用它？
>     *   如何使用 `curl -w "time_total: %{time_total}\n"` 来验证响应时间确实增加了约 2 秒？
> 4.  **Header 匹配**:
>     *   如何编写一个策略，只对带有特定请求头（如 `user: test`）的请求注入故障？
>     *   如何验证带此头的请求被注入了故障，而不带此头的请求则正常通过？
> 5.  **策略更新与删除**:
>     *   如何修改 `policy.yaml` (例如，从 Abort 改为 Delay) 并重新 `apply`，然后验证新规则是否生效？
>     *   如何使用 `hfi-cli policy delete` (虽然还没实现，但可以手动操作 etcd) 来删除策略，并验证所有请求恢复正常？

---

## **第三阶段：健壮性与易用性增强 - "让它好用"**

**目标**: 提升系统的稳定性、可观测性和用户体验。
**预计时间**: 2 周

**优先级 P0: 核心功能与用户体验闭环 (必须完成)**

| 任务 ID   | 任务模块        | 任务描述                                                                    | 关键技术点/产出                                                                  | 依赖任务   |
| :-------- | :-------------- | :-------------------------------------------------------------------------- | :------------------------------------------------------------------------------- | :--------- |
| **CL-3**  | **CLI**         | **实现 `get` 和 `delete` 命令**: 完善 CLI 的 CRUD 功能，`get` 支持多种输出格式。 | `cobra` 子命令, 表格格式化库, `APIClient` 扩展                                   | CL-2       |
| **C-5.1** | **Control Plane** | **实现基础异常处理 (NotFound)**: 在 DAL 和 Service 层实现 `ErrNotFound` 的翻译与传递。 | `errors.Is`, `etcd` 响应处理                                                     | C-4        |
| **C-5.2** | **Control Plane** | **实现基础错误中间件**: 在 API Handler 中将 `ErrNotFound` 映射为 HTTP 404。 | `gin` 中间件, HTTP 错误响应                                                      | C-5.1      |
| **D-1**   | **部署**        | **编写 Kubernetes 清单**: 为所有组件编写 K8s Deployment, Service, ConfigMap。  | Kubernetes YAML, `initContainer` 模式, `VolumeMounts`                          | T-2        |
| **O-1.1** | **Control Plane** | **引入结构化日志**: 使用 `zap` 等库并创建日志中间件，替换所有 `log.Printf`。      | 日志库 (`zap`), `gin` 中间件                                                     | C-4        |

**优先级 P1: 健壮性与可观测性深度增强 (重要但可稍后)**

| 任务 ID   | 任务模块        | 任务描述                                                                      | 关键技术点/产出                                                                  | 依赖任务 |
| :-------- | :-------------- | :---------------------------------------------------------------------------- | :------------------------------------------------------------------------------- | :------- |
| **W-6**   | **Wasm Plugin** | **增强健壮性**: 实现指数退避重连和全局 panic hook。                             | `backoff` 算法, `on_tick`, `std::panic::set_hook`                                | W-5      |
| **O-1.2** | **Wasm Plugin** | **添加核心 Metrics**: 在插件中记录 `aborts_total` 和 `delays_total` 的 counter。  | `proxy_wasm::traits::RootContext::increment_counter`                             | W-6      |
| **C-5.3** | **Control Plane** | **完善进阶异常处理**: 增加对 `ErrAlreadyExists`, `ErrInvalidInput` 等错误的处理。 | `etcd` 事务, 业务逻辑验证, HTTP 409/400 响应                                     | C-5.2    |
| **DOC-1** | **文档**        | **编写快速开始文档**: 撰写一份完整的 `QUICKSTART.md`，指导用户完成部署和首次使用。 | Markdown, 用户教程, `kubectl` 命令示例                                           | D-1, CL-3|

1.  **先完成 P0 任务**：交付一个功能完整、可在 K8s 中部署、具备基本调试能力的版本。
2.  **再进行 P1 任务**：在这个稳定的基础上，逐步加固数据平面的稳定性和系统的可观测性，并完善 API 的专业性和文档。

**第三阶段可交付成果**:
*   一个健壮性显著提升的系统，具备自动恢复和错误处理能力。
*   一个功能完整的 CLI 工具。
*   基础的可观测性能力（Metrics & Logging）。
*   一套可以在 Kubernetes 集群中部署的 YAML 文件。
*   **一份高质量的用户文档，可以对外分享**。


### **优先级 P0: 核心功能与用户体验闭环**


#### **任务 CL-3: CLI - 实现 `get` 和 `delete` 命令**

> **[User]**
> **任务 CL-3**: 完善 `hfi-cli` 的 `get` 和 `delete` 命令。
>
> **要求**:
> 1.  **实现 `get` 命令**:
>     *   在 `cmd/policy_get.go` 中创建 `getCmd`。
>     *   `getCmd` 可以接受 0 个或 1 个参数（策略名称）。
>     *   添加一个 `-o, --output` 标志，支持 `table` (默认), `yaml`, `json` 三种格式。
>     *   在 `RunE` 逻辑中：
>         *   如果没有参数，调用 `APIClient.ListPolicies()`。
>         *   如果有一个参数，调用 `APIClient.GetPolicyByName()`。
>         *   根据 `--output` 标志的值，将返回的结果进行格式化输出。
>         *   对于 `table` 格式，使用一个库（如 `github.com/olekukonko/tablewriter`）来生成对齐的、对人类友好的表格。表格应包含 `NAME`, `PRIORITY`, `MATCHES`, `FAULTS` 等关键摘要信息。
> 2.  **实现 `delete` 命令**:
>     *   在 `cmd/policy_delete.go` 中创建 `deleteCmd`。
>     *   `deleteCmd` 需要 1 个参数（策略名称）。
>     *   在 `RunE` 逻辑中，调用 `APIClient.DeletePolicy()`。
>     *   处理 `APIClient` 可能返回的错误（例如，`NotFound` 错误）。
>     *   如果成功，打印一条类似 `faultinjectionpolicy.hfi.dev "..." deleted` 的消息。
> 3.  **完善 `APIClient`**:
>     *   在 `client.go` 中，实现 `GetPolicyByName`, `ListPolicies`, `DeletePolicy` 这三个方法，完成与 Control Plane API 的交互。

---

#### **任务 C-5.1 & C-5.2: Control Plane - 实现基础异常处理**

> **[User]**
> **任务 C-5 (基础部分)**: 为 Control Plane 实现最核心的异常处理，以支持 CLI 的 `get` 和 `delete` 功能。
>
> **要求**:
> 1.  **定义领域错误**: 在 `storage` 包中，定义一个可导出的错误变量 `var ErrNotFound = errors.New("not found")`。
> 2.  **DAL 层错误翻译**:
>     *   在 `etcd_store.go` 中，修改 `GetByName` 和 `Delete` 方法。
>     *   当 etcd 客户端返回的结果表明 key 不存在时（例如，`Get` 响应的 `Count == 0`），这些方法必须返回 `storage.ErrNotFound`。
> 3.  **Service 层错误传递**:
>     *   确保 `PolicyService` 的 `GetPolicyByName` 和 `DeletePolicy` 方法在从 DAL 层收到 `storage.ErrNotFound` 时，能将其原样向上传递。
> 4.  **API Handler 层错误处理**:
>     *   创建一个 `gin` 中间件 `ErrorHandlerMiddleware`。
>     *   在 `PolicyController` 的 `Get` 和 `Delete` 方法中，当从 `PolicyService` 收到错误时，使用 `c.Error(err)` 将错误附加到上下文中。
>     *   在 `ErrorHandlerMiddleware` 中，使用 `errors.Is(err, storage.ErrNotFound)` 进行判断。
>     *   如果错误是 `ErrNotFound`，则返回 **HTTP 404 Not Found** 状态码和一个包含错误信息的 JSON 响应体。
>     *   对于所有其他类型的错误，暂时统一返回 **HTTP 500 Internal Server Error**。

---

> **[User]**
> **任务 D-1**: 编写一套用于在 Kubernetes 中部署整个系统的 YAML 清单。
>
> **请为我提供**:
> 1.  **`control-plane.yaml`**:
>     *   一个 `Deployment` 用于部署 Control Plane。应包含 `replicas`, `selector`, `template` 等字段。容器镜像指向你构建的 Control Plane 镜像。
>     *   一个 `Service` (`ClusterIP` 类型)，将流量暴露给集群内部的其他服务（主要是 Envoy）。
> 2.  **`envoy-config.yaml`**:
>     *   一个 `ConfigMap`，其中包含 `envoy.yaml` 的内容。
>     *   `envoy.yaml` 的配置需要修改：
>         *   Wasm 插件的 `vm_config.code` 部分应从本地文件路径改为指向一个 volume mount 路径 (e.g., `/etc/envoy/hfi_plugin.wasm`)。
>         *   插件配置中的 `address` 应指向 Control Plane 的 Kubernetes Service 名称 (e.g., `http://hfi-control-plane.default.svc.cluster.local:8080`)。
> 3.  **`sample-app-with-proxy.yaml`** (示例):
>     *   一个用于演示的 `Deployment`，例如部署一个简单的 `httpbin` 或 `nginx`。
>     *   关键：这个 `Deployment` 的 `PodSpec` 应该包含**两个容器**：
>         *   一个是你的应用容器 (`httpbin`)。
>         *   另一个是 `envoyproxy/envoy` **sidecar 容器**。
>     *   需要设置 `Volume` 和 `VolumeMounts`，将 `envoy-config.yaml` (`ConfigMap`) 挂载到 Envoy 容器的 `/etc/envoy` 路径。
>     *   还需要一种方式将 Wasm 插件二进制文件提供给 Envoy 容器。一个简单的方法是使用 `initContainer` 从一个包含 Wasm 文件的镜像中把它复制到一个 `emptyDir` volume 中，然后主 Envoy 容器再挂载这个 `emptyDir`。

---

> **[User]**
> **任务 O-1 (Control Plane 部分)**: 为 Control Plane 引入结构化日志。
>
> **要求**:
> 1.  引入一个结构化的日志库，例如 `uber-go/zap`。
> 2.  创建一个 `gin` 日志中间件 `LoggingMiddleware`。
> 3.  这个中间件应该使用 `zap` 记录每个请求的详细信息，包括：请求方法、路径、状态码、延迟、客户端 IP 等。
> 4.  日志应该是 JSON 格式，便于机器解析和后续的日志聚合系统（如 ELK, Loki）处理。
> 5.  替换掉项目中所有 `log.Printf` 的调用，改用 `zap` logger，并尽可能添加上下文信息（例如，在处理策略时，带上 `policy_name` 字段）。

---

### **优先级 P1: 健壮性与可观测性深度增强**

---

#### **任务 W-6: Wasm Plugin - 增强健壮性**

> **[User]**
> **任务 W-6 (Rust版)**: 增强 Wasm 插件的健壮性。
>
> **要求**:
> 1.  **实现指数退避重连**:
>     *   在 `Config Subscriber` 逻辑中（`PluginRootContext`），引入一个管理重连状态的机制。可以使用 `backoff` crate 或手动实现。
>     *   你需要一个 `reconnect_delay: Duration` 和一个 `reconnect_attempts: u32` 字段。
>     *   当 `on_http_call_response` 失败（例如，网络错误、服务器返回非 200）时，不要立即重新调度 HTTP call。
>     *   而是计算下一次重试的延迟时间（例如 `delay = min(initial_delay * 2^attempts, max_delay)`），然后调用 `self.set_tick_period(delay)`。
>     *   实现 `on_tick` 方法 for `RootContext`。当 `on_tick` 被调用时，才真正发起下一次 `dispatch_http_call`。
>     *   当一次 `on_http_call_response` 成功时，需要重置重连状态（`reconnect_attempts = 0`）。
> 2.  **Panic Safety**:
>     *   在 `proxy_wasm::main!` 宏注册的 `_start` 函数中，使用 `std::panic::set_hook` 设置一个全局的 panic hook。
>     *   在这个 hook 中，使用 `log::error!` 或 `proxy_wasm::hostcalls::log` 记录下 panic 的详细信息（payload 和 location）。
>     *   这可以确保任何未捕获的 panic 都会被记录下来，便于调试，而不会静默地导致 Wasm VM 崩溃。

---

#### **任务 C-5.3: Control Plane - 完善进阶异常处理**

> **[User]**
> **任务 C-5 (进阶部分)**: 扩展 Control Plane 的异常处理，使其能响应更多类型的业务错误。
>
> **要求**:
> 1.  **定义更多领域错误**: 在 `storage` 和 `service` 包中，定义并导出 `ErrAlreadyExists`, `ErrInvalidInput` 等错误。
> 2.  **扩展 DAL 层**: 在 `etcdStore` 的 `CreateOrUpdate` 方法中，使用 etcd 事务来区分“创建”和“更新”，并在尝试创建已存在的资源时返回 `storage.ErrAlreadyExists`。
> 3.  **扩展 Service 层**: 在 `PolicyService` 的 `CreateOrUpdate` 方法中，增加业务逻辑验证。如果验证失败，返回 `service.ErrInvalidInput` 并附带详细信息。
> 4.  **扩展 API Handler**: 在 `ErrorHandlerMiddleware` 中，增加对 `ErrAlreadyExists` 和 `ErrInvalidInput` 的处理逻辑，分别映射到 **HTTP 409 Conflict** 和 **HTTP 400 Bad Request**。

---

#### **任务 O-1.2: Wasm Plugin - 添加核心 Metrics**

> **[User]**
> **任务 O-1 (Wasm Plugin 部分)**: 在 Wasm 插件中添加核心 Metrics。
>
> **要求**:
> 1.  在 `executor.rs` 的 `execute_fault` 函数中，当确定要注入故障时，调用 `proxy_wasm` 的 metrics API。
> 2.  定义两个 counter 指标：`hfi.faults.aborts_total` 和 `hfi.faults.delays_total`。
> 3.  当执行 Abort 动作时，调用 `root_context.increment_counter("hfi.faults.aborts_total", 1)`。
> 4.  当执行 Delay 动作时，调用 `root_context.increment_counter("hfi.faults.delays_total", 1)`。
> 5.  (可选) 可以定义一个 histogram 指标 `hfi.faults.delay_duration_milliseconds`，并使用 `root_context.record_histogram` 来记录延迟的毫秒数。

---

#### **任务 DOC-1: 文档 - 编写快速开始文档** ✅

> **[User]**
> **任务 DOC-1**: 撰写一份快速开始 (Quick Start) 指南。
>
> **要求**:
> 1.  使用 Markdown 格式，创建一个 `QUICKSTART.md` 文件。
> 2.  **文档结构**:
>     *   **简介**: 简要介绍 `hfi` 是什么。
>     *   **先决条件**: 列出运行本指南所需的环境，例如 `kubectl`, `docker`, `kind` 或 `minikube`。
>     *   **步骤 1: 部署控制平面**: 提供 `kubectl apply -f ...` 命令来部署 `control-plane.yaml`。
>     *   **步骤 2: 部署示例应用与 Envoy Sidecar**: 提供命令来部署 `sample-app-with-proxy.yaml`。
>     *   **步骤 3: 安装 hfi-cli**: 提供下载和安装 `hfi-cli` 的说明。
>     *   **步骤 4: 注入你的第一个故障**:
>         *   提供一个简单的 `policy.yaml` 示例。
>         *   展示如何使用 `hfi-cli policy apply -f policy.yaml` 来应用它。
>     *   **步骤 5: 验证结果**:
>         *   展示如何通过 `kubectl port-forward` 来访问示例应用。
>         *   展示如何使用 `curl` 来触发请求并观察故障注入的效果。
>     *   **清理**: 提供 `kubectl delete` 命令来清理所有创建的资源。
> 3.  **清晰简洁**: 确保每个步骤都有明确的命令和预期的输出，让用户可以轻松地复制粘贴并成功运行。

---

## **第四阶段：进阶功能与生态集成 - "让它强大"**

**目标**: 实现设计中的进阶功能，并为与上层智能系统集成做好准备。
**预计时间**: 持续迭代

| 任务 ID | 任务模块         | 任务描述                                                                                              | 关键技术点/产出                                          | 依赖任务 |
| :------ | :--------------- | :---------------------------------------------------------------------------------------------------- | :------------------------------------------------------- | :------- |
| **W-7** | **Wasm Plugin**  | **实现基于 Body 的匹配**: 扩展 Matcher，增加对请求体的匹配能力（可能需要缓冲请求体）。                       | `proxy_on_http_request_body`                             | W-5      |
| **F-1** | **功能扩展**     | **增加新故障类型**: 扩展 `Fault` 定义和 Executor，支持如“响应篡改”等新故障类型。                           | `proxy_on_http_response_headers`                         | W-5      |
| **C-6** | **Control Plane**| **增强高可用**: 研究 Control Plane 的多副本部署方案，以及使用 leader election 保证 Distributor 的单一实例工作。 | `client-go/tools/leaderelection`                         | D-1      |
| **D-2** | **部署**         | **提供 Helm Chart**: 将 Kubernetes 清单打包成一个 Helm Chart，实现一键部署和配置。                         | Helm                                                     | D-1      |
| **I-2** | **集成**         | **与 `Recommender` 集成**: 与上层智能推荐系统进行 API 对接和端到端联调测试。                               | API 对接, 协同调试                                       | T-2      |
| **DOC-2**| **文档**         | **完善开发者文档**: 详细描述每个模块的设计、API 规范和扩展方式。                                          | API 文档, 开发者指南                                     | DOC-1    |

**第四阶段可交付成果**:
*   一个功能更强大、更灵活的故障注入系统。
*   一个生产级的部署方案 (Helm Chart)。
*   与生态系统成功集成的验证。
*   **一个完整的、可开源的、具备社区吸引力的项目**。



### **任务 DOC-2: 开发者文档 - 细分与拆解**

开发者文档的目标读者是**未来的你、你的同事，或者任何想要理解项目源码、进行二次开发、修复 Bug 或贡献新功能的开发者**。

| 任务 ID     | 文档/章节名称           | 核心内容                                                               | 目标读者                               |
| :---------- | :---------------------- | :--------------------------------------------------------------------- | :------------------------------------- |
| **DOC-2.1** | `ARCHITECTURE.md`       | 宏观架构、设计原则、核心数据流。                                       | 所有开发者，快速建立整体认知。         |
| **DOC-2.2** | `DEVELOPMENT.md`        | 本地开发环境搭建、编译、运行、测试指南。                               | 任何想在本地修改和运行代码的开发者。   |
| **DOC-2.3** | `CONTROL_PLANE_DEEP_DIVE.md` | 控制平面内部模块详解 (API/Service/DAL/Distributor)。                   | 专注于后端开发的开发者。               |
| **DOC-2.4** | `WASM_PLUGIN_DEEP_DIVE.md`   | Wasm 插件内部模块详解 (Context/Subscriber/Matcher/Executor)。          | 专注于数据平面和 Rust/Wasm 开发的开发者。 |
| **DOC-2.5** | `API_REFERENCE.md`      | `FaultInjectionPolicy` 资源的详细字段说明和 Control Plane 的 REST API 规范。 | 所有开发者，特别是需要与 API 交互的用户。 |
| **DOC-2.6** | `CONTRIBUTING_GUIDE.md` | 如何贡献代码，特别是如何添加新的故障类型或匹配条件。                   | 潜在的社区贡献者。                     |

---

### **任务 DOC-2.1: 宏观架构文档**

> **[User]**
> **任务 DOC-2.1**: 撰写宏观架构文档 (`ARCHITECTURE.md`)。
>
> **要求**:
> 1.  **高级概述**:
>     *   用一两段话阐述项目的目标和要解决的问题。
>     *   包含一张高级别的**系统上下文图**（可以复用或优化 `Design.md` 中的 PlantUML 图），展示用户、CLI、Control Plane、Data Plane (Envoy+Wasm) 之间的关系。
> 2.  **核心设计原则**:
>     *   详细阐述项目遵循的关键设计原则，并解释“为什么”这么选：
>         *   **控制平面与数据平面分离**: 解释其带来的好处（性能、可靠性、独立扩展等）。
>         *   **决策与执行分离**: 解释 `Recommender` (未来) 与 `Executor` (本项目) 的关系。
>         *   **声明式 API**: 解释为什么采用类似 Kubernetes CRD 的声明式配置。
> 3.  **技术选型与理由**:
>     *   简要列出核心技术栈（Go for Control Plane, Rust for Wasm Plugin, etcd, SSE 等）。
>     *   为每个技术选项提供 1-2 句的**选型理由**，特别是要解释**为什么选择 Rust 而不是 Go 来开发 Wasm 插件**（提及性能、内存安全、TinyGo 的问题等）。
> 4.  **核心数据流**:
>     *   包含一张**数据流图**（可以复用 `Design.md` 中的），并配以文字逐步解释从 `hfi-cli policy apply` 到 Wasm 插件注入故障的完整流程。

---

### **任务 DOC-2.2: 开发者环境搭建指南**

> **[User]**
> **任务 DOC-2.2**: 撰写本地开发环境搭建指南 (`DEVELOPMENT.md`)。
>
> **要求**:
> 1.  **先决条件**:
>     *   清晰列出开发所需的所有工具链及其推荐版本，例如：
>         *   Go (e.g., 1.21+)
>         *   Rust (e.g., 1.70+) & `wasm32-unknown-unknown` target
>         *   Docker & Docker Compose
>         *   `etcd` (可以通过 Docker 运行)
>         *   `kubectl` & `kind`/`minikube`
> 2.  **代码获取与结构**:
>     *   提供 `git clone` 命令。
>     *   简要介绍项目顶层目录结构（例如，`cmd/`放 CLI，`pkg/`放共享库，`internal/`放内部实现，`wasm/`放 Wasm 插件代码）。
> 3.  **构建指南**:
>     *   提供独立的编译命令：
>         *   如何编译 Control Plane 二进制文件 (`go build ...`)。
>         *   如何编译 Wasm 插件 (`cargo build --target wasm32-unknown-unknown ...`)。
>         *   如何编译 CLI (`go build ...`)。
> 4.  **本地运行 (非 K8s)**:
>     *   提供一个 `docker-compose.yaml` 或一组脚本，用于在本地快速启动 Control Plane, etcd, 和一个加载了 Wasm 插件的 Envoy 实例。这对于快速调试非常重要。
> 5.  **测试指南**:
>     *   如何运行 Go 的单元测试 (`go test ./...`)。
>     *   如何运行 Rust 的单元测试 (`cargo test`)。
>     *   (可选) 简要说明如何运行端到端测试。

---

### **任务 DOC-2.3: Control Plane 深度解析**

> **[User]**
> **任务 DOC-2.3**: 撰写 Control Plane 深度解析文档 (`CONTROL_PLANE_DEEP_DIVE.md`)。
>
> **要求**:
> 1.  **模块架构**:
>     *   包含一张 Control Plane 的**模块架构图**（复用 `Design_1_Control_Plane.md` 中的图）。
>     *   详细描述每个核心模块的职责：
>         *   **API Handler**: 如何使用 `gin`，中间件链（日志、错误处理）的工作方式。
>         *   **Policy Service**: 作为业务逻辑核心，如何与 DAL 和 Distributor 解耦。
>         *   **Storage Abstraction Layer (DAL)**: `IPolicyStore` 接口的重要性，以及 `etcdStore` 的实现要点（特别是 `Watch` 机制）。
>         *   **Config Distributor**: SSE 推送模型，`ClientManager` 的并发管理，以及配置编译的逻辑。
> 2.  **代码导览**:
>     *   指出关键代码的位置，例如 "API 路由定义在 `.../api/routes.go`"，"etcd 的 Watch 逻辑在 `.../storage/etcd_store.go` 的 `Watch()` 方法中"。
>     *   这有助于新开发者快速定位代码。

---

### **任务 DOC-2.4: Wasm 插件深度解析**

> **[User]**
> **任务 DOC-2.4**: 撰写 Wasm 插件深度解析文档 (`WASM_PLUGIN_DEEP_DIVE.md`)。
>
> **要求**:
> 1.  **核心概念与生命周期**:
>     *   解释 `proxy-wasm` 的核心概念：`RootContext` vs `HttpContext` 的区别和生命周期。
>     *   包含一张**模块架构图**（复用 `Design_2_Wasm_plugin.md` 中的图）。
> 2.  **模块详解**:
>     *   **Config Subscriber**: 解释如何利用 `on_http_call_response` 和 `on_tick` 实现异步、带退避重试的配置拉取循环。
>     *   **状态管理**: 详细说明 `Arc<RwLock<...>>` 如何用于在 `RootContext` 和多个 `HttpContext` 之间安全地共享状态（规则集）。
>     *   **Request Matcher**: 解释其无状态、高性能的设计，以及正则表达式预编译等优化。
>     *   **Fault Executor**: 解释如何将声明式的 `Fault` 对象翻译成对 Host ABI 的调用，特别是 `Delay` 故障如何通过 `set_timer` 和 `on_timer` 跨上下文实现。
> 3.  **性能与安全考量**:
>     *   强调插件中的性能优化点（零分配、短路评估等）。
>     *   解释 `panic hook` 的作用以及为何插件必须是 panic-safe 的。

---

### **任务 DOC-2.5: API 参考**

> **[User]**
> **任务 DOC-2.5**: 撰写 API 参考文档 (`API_REFERENCE.md`)。
>
> **要求**:
> 1.  **`FaultInjectionPolicy` 资源规范**:
>     *   提供一个完整的 `FaultInjectionPolicy` YAML 示例。
>     *   使用表格或列表，**详细描述每一个字段**的含义、类型、是否必需以及可选值。例如：
>         *   `metadata.name`: (string, required) 策略的唯一名称。
>         *   `spec.priority`: (integer, optional, default: 0) 规则优先级，数字越大优先级越高。
>         *   `spec.match.path.prefix`: (string) 匹配以此为前缀的 URL 路径。
>         *   ... 依此类推，覆盖所有匹配和故障字段。
> 2.  **Control Plane REST API**:
>     *   使用表格格式，列出所有 API 端点。
>     *   对每个端点，指明：**Method**, **Path**, **描述**, **请求体 (Body)**, **成功响应 (Success Response)**, **错误响应 (Error Responses)**。
>     *   例如：
>         *   **Create or Update Policy**
>         *   `POST /v1/policies`
>         *   描述：...
>         *   请求体：`FaultInjectionPolicy` JSON 对象。
>         *   成功响应：`201 Accepted`
>         *   错误响应：`400 Bad Request`, `409 Conflict`, `500 Internal Server Error`

---

### **任务 DOC-2.6: 贡献指南** ✅

> **[User]**
> **任务 DOC-2.6**: 撰写贡献指南 (`CONTRIBUTING_GUIDE.md`)。
>
> **要求**:
> 1.  **贡献流程**:
>     *   简述标准的 GitHub Fork & Pull Request 流程。
>     *   说明对 Commit Message 格式的要求（例如，Conventional Commits）。
> 2.  **如何添加新的故障类型 (Case Study)**:
>     *   以一个假设的新故障类型为例，例如 **"Response Corruption"**（修改响应体）。
>     *   **步骤 1 (Control Plane)**: 指出需要在 `.../api/v1alpha1/types.go` 的 `Fault` 结构体中添加 `ResponseCorruption *ResponseCorruptionAction`。
>     *   **步骤 2 (Wasm Plugin)**: 指出需要在 `wasm/src/config.rs` 中同步定义。
>     *   **步骤 3 (Wasm Plugin)**: 指出需要在 `wasm/src/executor.rs` 的 `execute_fault` 函数中增加一个分支，并可能需要在 `on_http_response_body` 生命周期钩子中实现具体逻辑。
> 3.  **如何添加新的匹配条件 (Case Study)**:
>     *   以一个假设的新匹配条件为例，例如 **"Query Parameter Matcher"**。
>     *   同样分步骤指出需要在 Control Plane 和 Wasm Plugin 的哪些文件的哪些结构体和函数中进行修改。

---

## **风险管理**

1.  **技术风险**: Wasm for Envoy 开发有学习曲线。
    *   **缓解措施**: **第一阶段**专门用于技术原型验证，尽早暴露问题。保持 Wasm 插件逻辑简单，复杂性上移到 Control Plane。
2.  **进度风险**: 任务估算可能不准。
    *   **缓解措施**: 采用小步快跑的方式，每个阶段都有明确的可交付成果。如果某个任务延期，可以及时调整后续计划，优先保证 MVP 的核心功能。
3.  **环境风险**: Kubernetes 和 Envoy 的环境配置复杂。
    *   **缓解措施**: **第一阶段**就搭建可复用的本地开发环境 (`docker-compose` 或 `kind`)，并将其代码化，所有开发者使用同一套环境。

## **可行性评估结论**

### **总体评价：高度可行 (8.5/10)**

**✅ 主要优势：**
- 技术架构设计合理，基于成熟的云原生技术栈
- 开发策略科学，MVP优先，小步迭代
- 风险识别充分，有具体的缓解措施
- 任务分解清晰，依赖关系明确

**⚠️ 建议改进：**
- **时间估算调整**: 第一阶段建议延长至2-2.5周，为技术学习预留充足时间
- **技术预研强化**: 增加Wasm和Envoy的前置学习任务
- **备选方案准备**: 考虑Lua插件作为技术验证的备选方案
- **测试策略补充**: 增加性能测试和稳定性测试的具体计划
- **文档驱动开发**: 在编码前先完善API文档和架构设计

**🚀 实施建议：**
1. **第0阶段（新增）**: 技术预研和环境准备（1周）
2. **调整时间分配**: 各阶段适当延长0.5-1周
3. **增加检查点**: 每个任务完成后进行技术审查
4. **准备Plan B**: 为关键技术点准备降级方案

---

## **详细可行性分析**

### **技术可行性 (9/10)**
- **Wasm + Envoy**: 技术成熟，有Istio等成功案例
- **Go生态**: proxy-wasm-go-sdk已稳定，社区活跃
- **服务网格**: 基于Istio的部署方案已被广泛验证
- **风险**: Wasm调试相对复杂，需要充分的技术预研

### **资源可行性 (8/10)**
- **人力**: 1-2名工程师的配置合理
- **时间**: 总体6-7周的时间规划可行，但建议各阶段适当延长
- **技能要求**: 需要Go、云原生、Envoy经验，学习曲线可控

### **架构可行性 (9/10)**
- **分层设计**: 控制平面与数据平面分离设计优秀
- **技术选型**: 基于成熟技术栈，避免了过度创新的风险
- **扩展性**: 模块化设计便于后续功能扩展

### **实施可行性 (8/10)**
- **阶段划分**: MVP优先的策略降低了实施风险
- **依赖管理**: 任务依赖关系清晰，便于并行开发
- **交付物**: 每阶段都有明确的可验证交付物

**总结**: 这是一个设计优秀、规划合理的开发计划。通过适当的时间调整和风险缓解，项目成功率很高。建议按照调整后的计划执行。

---

## **建议的调整时间线**

| 阶段 | 原计划时间 | 建议时间 | 主要调整内容 |
|:-----|:-----------|:---------|:-------------|
| **第0阶段（新增）** | - | **1周** | 技术预研、环境搭建、学习Wasm基础 |
| **第一阶段** | 1.5周 | **2.5周** | 增加Wasm调试和环境配置时间 |
| **第二阶段** | 2.5周 | **3周** | 增加集成测试和性能调优时间 |
| **第三阶段** | 2周 | **2.5周** | 增加文档编写和部署验证时间 |
| **第四阶段** | 持续迭代 | **持续迭代** | 保持原计划 |
| **总计（前三阶段）** | 6周 | **8周** | 增加33%缓冲时间 |

---
