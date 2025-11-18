# Recommender 模块设计文档检查报告

检查日期: 2025-11-11  
检查范围: `/recommender/docs/design_doc/` 目录下的所有设计文档  
目的: 评估设计的合理性、一致性和完整性



## 1. 整体架构设计评估

### ✅ 优点

#### 1.1 5W2H 分解清晰完整
- WHAT: 明确了三个核心组件 (Optimizer, Coordinator, Analyzer)
- WHY: 充分论证了为什么需要这个系统（智能决策、效率提升、自动化）
- WHO: 清晰定义了用户群体（CI/CD、SRE、开发团队）
- WHEN: 详细的4个Sprint阶段规划
- WHERE: 明确了开发和运行环境
- HOW: 给出了具体的实现步骤和伪代码
- HOW MUCH: 定义了完成和成功标准

✅ 评价: 这是一个非常专业的需求文档，5W2H框架应用得非常到位。

#### 1.2 技术栈选型有充分理由
选择了：
- Python 用于核心优化逻辑 - 科学计算生态
- Go 用于执行器（已完成）- 云原生、高并发
- Rust 用于 WASM 插件 - 性能和内存安全
- FastAPI - 高性能、类型提示、自动文档
- scikit-optimize - 易于使用、内置随机森林支持
- etcd - K8s 标准、watch 机制

✅ 评价: 每个选择都有充分的、合理的论述。

#### 1.3 模块划分遵循单一职责原则
- Coordinator Service: 生命周期管理和流程编排
- Optimizer Core: 智能决策算法
- Response Analyzer: 领域知识量化

✅ 评价: 模块边界清晰，职责单一，易于并行开发和测试。



### ⚠️ 潜在问题与建议

#### 问题 1.1: Design.md 的"HOW MUCH"部分定义过于简化

当前内容:

- 完成标准: 所有 Phase 1 任务完成, 单元测试覆盖率 > 90%
- 成功标准: BOIFI 发现 bug 的次数 < 随机注入, 性能 < 100ms


问题: 
- "系统的性能符合技术规格"中的"技术规格"文档未见链接或引用
- 没有明确的指标定义（如延迟基线是多少？严重性评分的分布如何？）

建议:
- 补充一个"关键指标"表格，定义明确的量化目标
- 例如:

  | 指标 | 目标 | 度量方法 |
  |--|--|--|
  | 单次优化循环延迟 | < 100ms | 采样1000次优化循环的平均 |
  | 严重性评分的 CV | < 0.1 | 对同一故障重复执行10次 |
  | 模型收敛速度 | < 30 次迭代 | 观察最佳分数不再改进 |
  | API 可用性 | >= 99.9% | 按周统计 |



#### 问题 1.2: 关键外部依赖未充分分析

当前设计假设:
- Executor API 是稳定的、可靠的
- 目标系统的观测数据（延迟、Trace）总是有效的
- 网络通信是稳定的

问题:
- 如果 Executor API 超时或返回错误怎么办？
- 如果 Trace 数据损坏或不完整怎么办？
- 如果网络抖动导致优化循环中断怎么办？

建议:
- 在 Design.md 中增加一个"依赖风险与缓解"部分
- 例如:
  ```markdown
  ## 依赖风险与缓解策略
  
  ### Executor 服务可用性风险
  - 风险: Executor API 响应超时或返回错误
  - 概率: 中等（网络可能抖动，Executor 可能重启）
  - 影响: 当前迭代失败，需要重试
  - 缓解:
    - Executor Client 实现重试逻辑（指数退避）
    - 将失败的实验记录为"未知观测"，暂时跳过
    - 在会话报告中清晰标记哪些迭代因故失败
  
  ### Trace 数据不完整
  - 风险: Response Analyzer 无法正常计算分项评分
  - 概率: 中等（生产环境的 Trace 可能有采样、丢失）
  - 影响: 严重性评分计算不准
  - 缓解: 在 Analyzer 中实现"Fail-Safe"原则（缺失字段用默认值）
  ```



## 2. Coordinator Service 设计评估

### ✅ 优点

#### 2.1 API 设计符合 REST 规范
- 使用标准的 HTTP 方法 (POST, GET)
- 清晰的路由设计 (`/v1/sessions`, `/v1/sessions/{session_id}`)
- 请求/响应体格式完整

#### 2.2 组件职责清晰
- API Server: 处理 HTTP 协议细节
- Session Manager: 管理会话生命周期
- Worker: 执行优化主循环

✅ 评价: 这是标准的分层架构，易于理解和测试。

#### 2.3 状态转换图清晰
- 请求处理的生命周期
- 会话的状态机 (PENDING → RUNNING → STOPPING → COMPLETED/FAILED)



### ⚠️ 潜在问题与建议

#### 问题 2.1: Worker 的"冷启动"逻辑未充分说明

当前设计:
```
冷启动阶段：循环 n_initial_points 次，生成随机点
```

问题:
- 随机点的生成策略是什么？（均匀分布？）
- `n_initial_points` 如何设定？有推荐值吗？
- 是否支持"预热点"（Warm start）？例如，从历史实验中导入已知的好点？

建议:
- 补充伪代码或算法描述
- 添加一个"冷启动配置"部分
  ```python
  class ColdStartConfig:
      n_initial_points: int = 10  # 推荐: min(20, search_space_size)
      strategy: str = "random"  # 可选: "lhs" (Latin Hypercube Sampling)
      warm_start_points: Optional[List[FaultPlan]] = None
  ```



#### 问题 2.2: 并发控制与会话隔离未明确

当前设计:
- "使用线程锁保证并发安全"（提到但未详细说明）

问题:
- 多个 Worker 会竞争同一个 Optimizer Core 吗？
  - 如果是，会导致状态竞争条件
  - 如果否，每个 Worker 都有独立的 Optimizer 实例吗？
- Session 之间是否完全独立，还是共享某些状态？
- 如果两个 Session 都在优化同一个故障空间，是否会相互干扰？

建议:
- 添加一个"并发模型"部分，明确说明：
  ```markdown
  ## 并发模型
  
  每个 Session 拥有独立的：
  - Optimizer 实例（维护独立的观测历史）
  - Executor Client 实例（但可能底层共享 HTTP 连接池）
  - Response Analyzer 实例（无状态，只读配置）
  
  Session Manager 使用 `threading.Lock` 保护 `sessions` 字典的读写。
  
  Thread Safety 保证：
  - 并发创建、查询、停止 Session 是安全的
  - 单个 Session 内的优化循环是单线程的（Worker.run() 在后台线程中执行）
  - 来自不同 Session 的优化循环可以真正并行执行（Python GIL 可能会有影响）
  ```



#### 问题 2.3: 会话持久化与恢复未涉及

当前设计:
- Session 状态存储在内存 (`dict`)

问题:
- 如果 Coordinator Service 崩溃或重启，所有会话都会丢失！
- 长时间运行的优化（几小时几天）没有断点恢复机制

建议 (优先级: 中等，可作为 Phase 2 特性):
- 添加一个"持久化层"设计
  ```markdown
  ## 持久化与恢复 (Future Consideration)
  
  为了支持长时间运行的优化任务，应该实现会话持久化：
  
  1. 选择存储方案:
     - 简单: 定期将会话快照保存到本地文件系统
     - 可靠: 使用 Redis 或 etcd 存储会话状态和 Optimizer 历史
  
  2. 恢复机制:
     - 服务启动时，扫描所有已持久化的会话
     - 恢复 Optimizer 的观测历史 (Xi, yi)
     - 对于正在运行的会话，重新启动 Worker
     - 对于已完成的会话，只提供查询功能
  ```



## 3. Optimizer Core 设计评估

### ✅ 优点

#### 3.1 接口抽象设计优秀
- 定义了 `IOptimizer` 接口，实现了"策略模式"
- 允许未来轻松切换优化算法实现（scikit-optimize → BoTorch）
- 接口简洁明了：只有 3 个方法 (`propose`, `record`, `get_best_result`)

✅ 评价: 这是典型的良好设计，符合SOLID原则。

#### 3.2 状态机设计清晰
- UNINITIALIZED → COLD_START → WARM
- 冷启动与贝叶斯优化阶段的区分很清楚

#### 3.3 异常处理矩阵详尽
- 定义了多个自定义异常类型
- 对每个异常情况都有明确的处理策略



### ⚠️ 潜在问题与建议

#### 问题 3.1: Space Converter 的设计过于简略

当前内容 (Design_2_Optimizer Core.md 最后):
```
## 模块 2.3: Space Converter (`optimizer/space_converter.py`)

*   职责: 将用户友好的 JSON/YAML 搜索空间配置文件，转换为 `scikit-optimize` 能理解的 `Dimension` 对象列表。
*   技术: 纯 Python 逻辑。
*   核心逻辑:
    *   一个函数 `def convert_space_config(config: dict) -> List[skopt.space.Dimension]:`
    *   它会解析配置，根据 `type` 字段（如 `categorical`, `real`, `integer`）创建相应的 `skopt.space.Categorical`, `skopt.space.Real` 等对象。
```

问题:
- 没有给出配置文件的 Schema 示例
- 没有说明如何处理嵌套、复杂的搜索空间
- 没有错误处理细节

建议:
- 补充"配置 Schema 示例"
  ```yaml
  # fault_space_config.yaml
  
  search_space:
    - name: "service"
      type: "categorical"
      values: ["PaymentService", "OrderService", "InventoryService"]
    
    - name: "fault_type"
      type: "categorical"
      values: ["delay", "abort", "error_injection"]
    
    - name: "delay_seconds"
      type: "real"
      bounds: [0.1, 10.0]  # 最小 0.1s，最大 10s
      depend_on: {fault_type: "delay"}  # 条件依赖
    
    - name: "error_code"
      type: "integer"
      bounds: [400, 599]
      depend_on: {fault_type: "abort"}
  ```

- 补充"条件空间处理"说明
  ```markdown
  ## 条件搜索空间 (Conditional Search Space)
  
  当前 scikit-optimize 不原生支持条件维度。处理方案：
  
  1. 展开式 (Conservative): 将所有维度都加入搜索空间，Executor 忽略不相关的参数
     - 优点: 简单
     - 缺点: 搜索空间变大，效率低
  
  2. 过滤式 (Recommended): 在 Optimizer.propose() 时，根据某些参数值动态修改搜索空间
     - 优点: 搜索空间保持紧凑
     - 缺点: 实现复杂，需要特殊处理
  
  3. 编码式: 将条件逻辑编码到参数编码中
     - 优点: 不需要修改搜索空间
     - 缺点: 实现最复杂，容易出错
  
  推荐方案: 先使用方案1（展开式），如果性能不满足需求，再升级为方案2。
  ```



#### 问题 3.2: 负分转换的必要性与细节

当前设计:
```
Wrapper 层需要在调用 tell() 时传入 `-score`，因为 skopt 最小化目标函数
```

问题:
- 这个设计增加了认知负担（每次看到 `-score` 都要想一下为什么）
- 在 `get_best_result()` 中需要再次取反，容易出错

建议:
- 在 Wrapper 的注释中明确说明原因和流程
  ```python
  def record(self, point: Dict[str, Any], score: float):
      """
      Record an observation.
      
      Args:
          point: The fault injection plan (dict).
          score: The severity score (higher is worse/more severe).
      
      Note:
          scikit-optimize minimizes the objective function by default.
          Since we want to maximize severity (higher score = more severe),
          we negate the score before passing to skopt.
      """
      point_list = self._dict_to_list(point)
      # Negate: skopt minimizes, but we want to maximize severity
      self.skopt_optimizer.tell(point_list, -score)
  
  def get_best_result(self) -> Tuple[Dict[str, Any], float]:
      """
      Get the best result found so far.
      
      Returns:
          (best_plan, best_score) where best_score is the highest severity found.
      
      Note:
          We take the minimum of the negated scores, then negate back to get the
          actual maximum severity score.
      """
      if len(self.skopt_optimizer.yi) == 0:
          return (None, -float('inf'))
      
      # skopt stores negated scores in yi, so min(yi) = highest severity
      min_idx = np.argmin(self.skopt_optimizer.yi)
      best_plan_list = self.skopt_optimizer.Xi[min_idx]
      best_plan_dict = self._list_to_dict(best_plan_list)
      # Un-negate to get actual severity score
      best_score = -self.skopt_optimizer.yi[min_idx]
      
      return (best_plan_dict, best_score)
  ```



#### 问题 3.3: 模型训练和重训练的计算成本未评估

当前设计:
- 每次 `record()` 后，scikit-optimize 会重新训练随机森林模型

问题:
- 随机森林训练的计算成本随着观测点数增加而增加
- 如果有 1000 次实验迭代，是否会导致明显的性能退化？
- 是否需要定期清理旧观测点，只保留最近的 N 个？

建议:
- 补充"性能特性"分析
  ```markdown
  ## 性能特性
  
  ### 模型训练时间复杂度
  
  scikit-optimize 使用随机森林作为代理模型：
  
  | n_observations | 构建时间 (ms) | 单次提议时间 (ms) | 总计 (ms) |
  |-|--|--|-|
  | 10             | ~5           | ~1              | ~6       |
  | 100            | ~20          | ~2              | ~22      |
  | 1000           | ~100         | ~5              | ~105     |
  
  在 100 次迭代的预算下，模型训练总成本约为 1-2 秒（可接受）。
  在 10000 次迭代的预算下，可能需要考虑优化或换用更轻量的代理模型。
  
  ### 缓解方案 (如需要)
  
  1. Subsampling: 只用最近的 1000 个观测点训练模型，丢弃更古老的观测
  2. Incremental Model: 换用支持增量学习的模型（如 SGDRegressor）
  3. Batch Optimization: 每 N 个 `record()` 调用后再进行一次 Wrapper 的模型更新
  ```



## 4. Response Analyzer 设计评估

### ✅ 优点

#### 4.1 策略模式应用恰当
- `IScorer` 接口定义了每个评分维度的契约
- 支持轻松添加新的评分维度（如"资源消耗分"）

#### 4.2 Fail-Safe 原则设计好
- 任何一个 Scorer 的失败都不会导致整个 `calculate_severity` 失败
- 缺失字段自动回退到默认值，确保优化循环能继续

#### 4.3 可配置性强
- 所有权重、阈值都在 `AnalyzerConfig` 中
- 领域专家可以轻松调整评分策略



### ⚠️ 潜在问题与建议

#### 问题 4.1: 评分函数的具体公式未给出

当前设计:
```
total_score = w_bug * Score_Bug + w_perf * Score_Perf + w_struct * Score_Struct
```

问题:
- 各个分项的范围是什么？[0, 1]？[0, 10]？无界？
- 权重的推荐值是什么？
- 不同维度之间是否需要归一化？

建议:
- 补充"评分公式详解"
  ```markdown
  ## 评分公式详解
  
  ### 输入数据规范化
  
  | 分项 | 原始范围 | 映射到 [0, 10] | 说明 |
  |--|--|--|--|
  | Score_Bug | [0, 1] | `score_bug * 10` | 0 = 未触发 bug, 1 = 明确的致命 bug |
  | Score_Perf | [0, ∞) | `min(latency_increase / 1000ms, 10)` | 相对于基线的延迟增加 |
  | Score_Struct | [0, 1] | `score_struct * 10` | 0 = 无变化, 1 = 完全改变行为 |
  
  ### 加权求和
  
  ```
  severity_score = (w_bug * Score_Bug + w_perf * Score_Perf + w_struct * Score_Struct) 
                    / (w_bug + w_perf + w_struct)
  ```
  
  结果范围: [0, 10]，其中 10 = 最严重。
  
  ### 推荐权重配置
  
  ```python
  AnalyzerConfig(
      weights={
          "bug": 10.0,      # Bug 最重要
          "perf": 2.0,      # 性能次之
          "struct": 5.0     # 结构变化也很重要
      },
      baseline_latency_ms=200,
      threshold_latency_ms=1000
  )
  ```
  ```



#### 问题 4.2: 分布式追踪 (Trace) 的分析方法过于抽象

当前设计:
```
StructureScorer: 分析 trace_data，将其与已知的基线 Trace 进行比对，以检测重试、降级等模式
```

问题:
- 如何获取"已知的基线 Trace"？
- 如何进行 Trace 比对？（逐字符比较？语义比较？）
- 重试、降级的定义是什么？

建议:
- 补充"Trace 分析方法"
  ```markdown
  ## StructureScorer - Trace 分析详解
  
  ### 基线 Trace 的获取
  
  1. 离线方式 (当前推荐):
     - 在不注入故障的情况下，执行一次目标请求，记录其完整的分布式追踪
     - 将该 Trace 存储为 baseline_trace.json，与 AnalyzerConfig 一起配置
     - 启动优化时，加载 baseline_trace.json
  
  2. 在线方式 (未来优化):
     - 在优化的冷启动阶段，自动执行一次无故障请求来获取基线 Trace
  
  ### Trace 比对算法
  
  输入: `current_trace` (当前请求的 Trace), `baseline_trace` (基线 Trace)
  
  ```python
  def compare_traces(current_trace, baseline_trace) -> StructureChangeDetected:
      changes = []
      
      # 1. 检查 Span 数量变化（可能表示重试或提前返回）
      if len(current_trace.spans) > len(baseline_trace.spans) * 1.5:
          changes.append(("high_retry_rate", 0.5))
      
      # 2. 检查 Span 名称序列变化（可能表示降级或异常处理）
      current_seq = [s.operation_name for s in current_trace.spans]
      baseline_seq = [s.operation_name for s in baseline_trace.spans]
      if edit_distance(current_seq, baseline_seq) > 2:
          changes.append(("control_flow_change", 0.7))
      
      # 3. 检查错误标签（5xx, 4xx 错误）
      if any(s.is_error for s in current_trace.spans):
          changes.append(("error_detected", 0.8))
      
      # 4. 检查延迟异常（某个 Span 比基线慢很多）
      for curr_span, base_span in zip_longest(current_trace.spans, baseline_trace.spans):
          if base_span and curr_span.duration > base_span.duration * 5:
              changes.append(("span_latency_spike", 0.4))
      
      # 综合评分
      if not changes:
          return 0.0
      else:
          scores = [score for _, score in changes]
          return max(scores)  # 取最严重的变化
  ```
  
  ### 推荐的 Trace 格式
  
  假设采用 Jaeger 或 OpenTelemetry 格式：
  
  ```json
  {
    "traceID": "abc123",
    "spans": [
      {
        "spanID": "1",
        "operationName": "http.request",
        "startTime": 1000000,
        "duration": 100000,  // microseconds
        "tags": {"http.status_code": 200, "http.method": "GET"},
        "logs": [
          {"timestamp": 1000100, "message": "Started"},
          {"timestamp": 1000200, "message": "Got response"}
        ]
      },
      // ... 更多 spans
    ]
  }
  ```
  ```



#### 问题 4.3: 不同故障类型的评分差异未涵盖

当前设计:
- 评分函数是通用的，不区分故障类型（延迟？中止？错误注入？）

问题:
- 延迟故障和中止故障对系统的影响方式完全不同
- 相同的评分不应该被用来比较这两种不同的故障

建议 (优先级: 低，可作为 Phase 2 增强):
- 考虑在 AnalyzerConfig 中添加故障类型特定的配置
  ```python
  class AnalyzerConfig:
      global_weights: Dict[str, float] = {...}
      
      # 故障类型特定的权重和阈值
      fault_type_configs: Dict[str, Dict] = {
          "delay": {
              "weights": {"bug": 5, "perf": 10, "struct": 2},  # 性能权重更高
              "baseline_latency_ms": 200
          },
          "abort": {
              "weights": {"bug": 10, "perf": 5, "struct": 3},  # Bug 权重更高
          },
          "error_injection": {
              "weights": {"bug": 10, "perf": 1, "struct": 5},
          }
      }
  ```



## 5. 跨模块设计问题

### 问题 5.1: Executor Client 的设计与错误恢复

涉及文档: Design.md 提到 Executor Client 的 `apply_and_observe()` 方法

问题:
- 方法如何等待执行结果？轮询？长连接？
- 超时时间是多少？
- 如果 Executor 返回错误，是重试还是继续？

建议:
- 补充一个"Executor Client 集成指南"
  ```markdown
  ## Executor Client 实现细节
  
  ### 如何获取执行结果
  
  执行器的 `/v1/policies` API 采用异步模式：
  - POST 请求立即返回 `202 Accepted` + `execution_id`
  - 需要通过 polling 获取最终结果
  
  ```python
  def apply_and_observe(plan: FaultPlan) -> RawObservation:
      # 1. 提交故障注入
      response = POST "http://executor/v1/policies" + plan_json
      execution_id = response.execution_id
      
      # 2. Polling 获取结果
      max_retries = 60  # 最多等待 60 秒
      for i in range(max_retries):
          result = GET f"http://executor/v1/executions/{execution_id}"
          if result.status == "COMPLETED":
              return result.observation
          elif result.status == "FAILED":
              raise ExecutionFailedError(result.error_message)
          time.sleep(1)
      
      # 3. 超时
      raise ExecutionTimeoutError(f"Execution {execution_id} timed out after 60s")
  ```
  
  ### 错误恢复策略
  
  | 错误类型 | 可恢复？ | 策略 |
  |--|--|--|
  | 网络超时 | 是 | 指数退避重试，最多 3 次 |
  | 执行超时 | 否 | 记录为"未知观测"，跳过此迭代 |
  | 无效的故障组合 | 否 | 应该在 Space Converter 中提前验证 |
  | Executor 内部错误 | 否 | 中断会话，返回 500 错误 |
  ```



### 问题 5.2: 数据结构定义的跨模块一致性

问题:
- Design 文档中多次出现 `FaultPlan`, `RawObservation`, `AnalyzerConfig` 等类型
- 但没有给出 Python 的精确定义（dataclass 还是 dict？）
- 没有统一的地方定义这些类型

建议:
- 在 Design.md 中添加一个"数据模型定义"部分
  ```markdown
  ## 数据模型定义 (Data Models)
  
  所有关键数据结构的 Python 定义应该统一放在 `recommender/src/types.py` 中。
  
  ### FaultPlan
  ```python
  from dataclasses import dataclass
  from typing import Any, Dict
  
  @dataclass
  class FaultPlan:
      """表示一个具体的故障注入方案"""
      service: str                       # 目标服务名
      fault_type: str                    # "delay" | "abort" | "error_injection"
      percentage: int                    # 0-100, 影响的请求百分比
      delay_seconds: Optional[float] = None
      abort_http_status: Optional[int] = None
      # ... 更多字段
      
      def to_executor_policy(self) -> Dict:
          """转换为 Executor API 能理解的格式"""
          # ...
  ```
  
  ### RawObservation
  ```python
  @dataclass
  class RawObservation:
      """执行器返回的原始观测数据"""
      status_code: int
      latency_ms: float
      error_rate: float                  # 0-1
      trace_data: Dict                   # 分布式追踪数据
      logs: List[str]                    # 应用日志片段
      timestamp: datetime
  ```
  
  ### AnalyzerConfig
  ```python
  @dataclass
  class AnalyzerConfig:
      """Response Analyzer 的配置"""
      weights: Dict[str, float]          # bug, perf, struct 的权重
      baseline_latency_ms: float = 200.0
      threshold_latency_ms: float = 1000.0
      baseline_trace: Optional[Dict] = None
  ```
  ```



## 6. 可维护性与演进性评估

### ✅ 优点

- 接口隔离: 每个模块都有清晰的接口定义
- 策略模式应用: 易于扩展和替换实现
- 配置驱动: 避免硬编码，支持灵活配置

### ⚠️ 建议

#### 6.1 缺少详细的"实现清单"

问题: Design 文档说了"做什么"和"为什么"，但对于"实际怎么写代码"的指导不够具体。

建议:
- 在每个 Sprint 的设计文档（如 Design_1_CoordinatorService.md）的最后，加上一个"实现清单"
  ```markdown
  ## 实现清单 (Implementation Checklist)
  
  ### 文件结构
  ```
  recommender/src/
  ├── main.py
  ├── api/
  │   ├── __init__.py
  │   ├── server.py              # FastAPI 应用
  │   ├── routers/
  │   │   ├── __init__.py
  │   │   └── sessions.py        # /v1/sessions 路由
  │   ├── models/
  │   │   ├── __init__.py
  │   │   └── api_models.py      # Pydantic 模型
  │   └── dependencies.py        # 依赖注入
  ├── coordinator/
  │   ├── __init__.py
  │   ├── session_manager.py
  │   └── worker.py
  ├── optimizer/
  │   ├── __init__.py
  │   ├── interface.py           # IOptimizer 接口
  │   ├── skopt_wrapper.py
  │   └── space_converter.py
  ├── analyzer/
  │   ├── __init__.py
  │   ├── service.py             # AnalyzerService
  │   ├── scorers.py             # 各个 Scorer 实现
  │   └── types.py
  ├── clients/
  │   ├── __init__.py
  │   └── executor_client.py
  ├── types.py                   # 共享数据模型
  ├── config.py                  # 全局配置
  └── logger.py                  # 日志设置
  ```
  
  ### 关键 TODO 项
  - [ ] API Server: 实现 FastAPI 应用、路由、异常处理
  - [ ] Session Manager: 实现会话生命周期管理
  - [ ] Worker: 实现主优化循环
  - [ ] Optimizer Interface: 定义抽象接口
  - [ ] Scikit-Optimizer Wrapper: 集成 scikit-optimize
  - [ ] Space Converter: 实现空间转换逻辑
  - [ ] Analyzer Service: 实现评分函数和各个 Scorer
  - [ ] Executor Client: 实现与 Executor 的通信
  - [ ] 单元测试: 每个模块的 unit tests
  - [ ] 集成测试: 端到端的优化循环测试
  - [ ] Docker: 编写 Dockerfile
  - [ ] 文档: 补充 API 文档、使用指南
  ```



#### 6.2 缺少测试策略

问题: 设计文档中提到"单元测试覆盖率 > 90%"，但没有给出测试策略。

建议:
- 添加一个"测试策略"部分
  ```markdown
  ## 测试策略 (Testing Strategy)
  
  ### 单元测试 (Unit Tests)
  
  | 模块 | 测试类型 | 关键用例 | 预期覆盖率 |
  |--|--|--|-|
  | Optimizer Interface & Wrapper | 状态转换、负分转换 | 冷启动、提议、记录 | >= 95% |
  | Space Converter | 配置解析、空间定义 | 各类型维度、条件依赖 | >= 90% |
  | Analyzer Service | 评分计算、Scorer 回退 | 正常、缺失数据、异常 | >= 95% |
  | Session Manager | 并发操作、状态管理 | 创建、查询、停止 | >= 90% |
  | Executor Client | 重试逻辑、错误处理 | 成功、超时、失败 | >= 85% |
  
  ### 集成测试 (Integration Tests)
  
  1. Optimizer + Space Converter: 完整的优化循环（10-20 次迭代）
  2. Analyzer + Executor Client: 原始观测数据 → 评分
  3. Coordinator + Worker: 完整的优化会话（50-100 次迭代，模拟 Executor）
  
  ### 系统测试 (System Tests)
  
  1. 基准测试 (Baseline): 随机注入 N 次，vs 优化 N 次，对比找到的最严重故障
  2. 稳定性测试: 长时间运行（几小时），检查内存泄漏、性能退化
  3. 故障恢复测试: 中断优化、重启、继续，验证状态恢复
  
  ### 性能测试 (Benchmarks)
  
  ```python
  # tests/benchmarks/test_performance.py
  
  def benchmark_single_optimization_loop():
      """一个完整的优化循环应该 < 1 秒"""
      # Propose, Execute, Analyze, Record
      
  def benchmark_optimizer_training():
      """模型训练时间随观测点数的增长"""
      # 测试 10, 100, 1000 个观测点的情况
  ```
  ```



## 7. 总体评估与优先级建议

### 📊 设计质量评分

| 维度 | 评分 | 注释 |
|--|--|--|
| 架构清晰度 | 9/10 | 模块划分清晰，但并发模型需要澄清 |
| 接口设计 | 9/10 | 接口隔离好，支持扩展性 |
| 完整性 | 7/10 | 缺少数据模型定义、配置 Schema、性能分析 |
| 可维护性 | 8/10 | 模块化好，但缺少实现细节和测试策略 |
| 风险分析 | 6/10 | 缺少外部依赖风险、持久化等考虑 |
| 整体评分 | 7.8/10 | 好的高层设计，需要补充细节 |



### 🎯 建议的补充工作（优先级）

#### P0 (必须做)
1. ✅ 补充"数据模型定义"（跨模块一致性）
2. ✅ 澄清"并发模型"和"会话隔离"
3. ✅ 补充"配置 Schema 示例"
4. ✅ 详细说明"Space Converter"的实现

#### P1 (强烈建议)
5. ✅ 补充"Executor Client 集成指南"
6. ✅ 补充"Trace 分析方法"（StructureScorer 具体实现）
7. ✅ 补充"性能特性分析"（模型训练成本）
8. ✅ 添加"测试策略"（单元、集成、系统、性能测试）
9. ✅ 补充"实现清单"（文件结构、TODO 项）

#### P2 (可选，Phase 2)
10. 补充"依赖风险与缓解"部分
11. 补充"会话持久化与恢复"设计
12. 补充"故障类型特定的评分"配置
13. 详细的"冷启动配置"设计



## 8. 文件清单与建议

### 当前设计文档结构

```
recommender/docs/design_doc/
├── Design.md                        # 整体 5W2H 分解
├── Design_1_CoordinatorService.md  # Coordinator Service 详细设计
├── Design_2_Optimizer Core.md      # Optimizer Core 详细设计
└── Design_3_Response Analyzer.md   # Response Analyzer 详细设计
```

### 建议的补充文档

```
recommender/docs/design_doc/
├── Design.md
├── Design_1_CoordinatorService.md
├── Design_2_Optimizer Core.md
├── Design_3_Response Analyzer.md
├── Design_4_Data_Models.md         # 【新增】共享数据模型定义
├── Design_5_Executor_Integration.md # 【新增】与 Executor 的集成
├── Design_6_Testing_Strategy.md     # 【新增】测试策略
└── Design_7_Deployment.md           # 【新增】部署、配置、监控
```



## 9. 结论

### ✅ 总体评价

这份设计文档展现了专业的架构思维：

1. 顶层清晰: 5W2H 框架应用得非常好，需求、目标、时间规划都很清楚
2. 模块化设计: 按照单一职责原则进行了合理的模块划分
3. 接口隔离: 设计了良好的接口抽象，支持未来的扩展和替换
4. 质量关注: 有详细的异常处理矩阵、状态转换图

### ⚠️ 需要改进之处

1. 细节不足: 
   - 数据模型定义不清
   - 配置 Schema 示例缺失
   - 具体算法细节（如 Trace 分析）太抽象
   - 性能特性未分析

2. 风险未充分覆盖:
   - 外部依赖风险（Executor 可能不稳定）
   - 长时间运行的持久化（会话断点恢复）
   - 并发模型有歧义

3. 实现指导不够:
   - 缺少文件结构清单
   - 缺少测试策略
   - 缺少"实现 TODO"列表

### 🎯 建议的后续步骤

1. 立即完成 (Sprint 开始前):
   - 补充所有 P0 级别的详细设计
   - 确定数据模型和配置 Schema
   - 澄清并发模型

2. Sprint 进行中:
   - 按照补充的详细设计逐个实现
   - 定期同步设计文档和代码实现
   - 建立代码与文档的"双向追踪"（Design → Code → Tests）

3. 开发完成后:
   - 补充 P2 级别的优化设计
   - 编写运维和部署文档
   - 整理"最佳实践"和"常见坑"指南



报告生成时间: 2025-11-11  
审查人: AI Code Assistant  
下一步: 根据本报告的建议，更新设计文档并启动实现。
