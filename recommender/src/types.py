"""
数据模型定义 (Data Models)

本模块定义了 Recommender 系统中所有关键的数据结构。
所有模块都应该使用这里定义的类型，以确保跨模块的数据一致性。

设计原则:
1. 使用 dataclass 作为数据容器（自动生成 __init__, __repr__, __eq__ 等）
2. 使用 type hints 提供清晰的类型信息
3. 使用 Pydantic 模型作为 API 的请求/响应类型（自动验证）
4. 文档化每个字段的含义和约束
"""

from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional, Tuple
from enum import Enum
from datetime import datetime
import json


# ============================================================================
# 枚举类型 (Enums)
# ============================================================================

class FaultType(str, Enum):
    """故障注入的类型"""
    DELAY = "delay"                      # 延迟注入
    ABORT = "abort"                      # 中止请求（返回错误状态码）
    ERROR_INJECTION = "error_injection"  # 错误注入（模拟异常）


class SessionStatus(str, Enum):
    """优化会话的状态"""
    PENDING = "PENDING"        # 待启动
    RUNNING = "RUNNING"        # 运行中
    STOPPING = "STOPPING"      # 停止中（优雅停止）
    COMPLETED = "COMPLETED"    # 完成
    FAILED = "FAILED"          # 失败


# ============================================================================
# 故障注入相关 (Fault Injection Models)
# ============================================================================

@dataclass
class FaultPlan:
    """
    表示一个具体的故障注入方案。
    
    这是优化器提议的点，也是执行器执行的计划。
    
    Attributes:
        service: 目标服务名（如 "PaymentService", "OrderService"）
        api: 目标 API 路径（如 "/api/v1/payment", "/api/v1/order"）
        fault_type: 故障类型（延迟、中止、错误注入）
        percentage: 影响的请求百分比，范围 [0, 100]
        delay_seconds: 延迟时间（秒），仅在 fault_type == DELAY 时有意义
                       范围: [0.1, 30.0]
        abort_http_status: 中止时返回的 HTTP 状态码，仅在 fault_type == ABORT 时有意义
                           常见: 500, 502, 503, 504
        error_message: 错误消息，仅在 fault_type == ERROR_INJECTION 时有意义
    """
    service: str
    api: str
    fault_type: FaultType
    percentage: int  # 0-100
    delay_seconds: Optional[float] = None
    abort_http_status: Optional[int] = None
    error_message: Optional[str] = None
    
    def __post_init__(self):
        """验证字段的有效性"""
        if not 0 <= self.percentage <= 100:
            raise ValueError(f"percentage 必须在 [0, 100] 范围内，得到: {self.percentage}")
        
        if self.fault_type == FaultType.DELAY:
            if self.delay_seconds is None:
                raise ValueError("fault_type=DELAY 时必须指定 delay_seconds")
            if not 0.1 <= self.delay_seconds <= 30.0:
                raise ValueError(f"delay_seconds 必须在 [0.1, 30.0] 范围内，得到: {self.delay_seconds}")
        
        if self.fault_type == FaultType.ABORT:
            if self.abort_http_status is None:
                raise ValueError("fault_type=ABORT 时必须指定 abort_http_status")
            if not 400 <= self.abort_http_status <= 599:
                raise ValueError(f"abort_http_status 必须在 [400, 599] 范围内，得到: {self.abort_http_status}")
    
    def to_dict(self) -> Dict[str, Any]:
        """转换为字典，用于序列化"""
        result = {
            "service": self.service,
            "api": self.api,
            "fault_type": self.fault_type.value,
            "percentage": self.percentage,
        }
        if self.delay_seconds is not None:
            result["delay_seconds"] = self.delay_seconds
        if self.abort_http_status is not None:
            result["abort_http_status"] = self.abort_http_status
        if self.error_message is not None:
            result["error_message"] = self.error_message
        return result
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "FaultPlan":
        """从字典构建实例"""
        fault_type_str = data["fault_type"]
        fault_type = FaultType(fault_type_str)
        
        return cls(
            service=data["service"],
            api=data["api"],
            fault_type=fault_type,
            percentage=data["percentage"],
            delay_seconds=data.get("delay_seconds"),
            abort_http_status=data.get("abort_http_status"),
            error_message=data.get("error_message"),
        )


@dataclass
class RawObservation:
    """
    执行器返回的原始观测数据。
    
    这是执行完一个故障注入计划后，从目标系统收集到的所有原始数据。
    Response Analyzer 会将这些原始数据转化为单一的"严重性评分"。
    
    Attributes:
        status_code: 最终响应的 HTTP 状态码（如 200, 503）
        latency_ms: 请求的端到端延迟（毫秒）
        error_rate: 在这个故障注入期间，目标服务返回错误的请求比例 [0, 1]
        success_count: 成功的请求数
        error_count: 失败的请求数
        trace_data: 分布式追踪数据（Jaeger/OpenTelemetry 格式）
                    包含 spans、timing、错误信息等
        logs: 应用日志片段（与该实验相关的关键日志）
        timestamp: 实验执行的时间戳
        execution_id: 执行器返回的执行 ID，用于追踪
    """
    status_code: int
    latency_ms: float
    error_rate: float  # 0-1
    success_count: int = 0
    error_count: int = 0
    trace_data: Dict[str, Any] = field(default_factory=dict)
    logs: List[str] = field(default_factory=list)
    timestamp: Optional[datetime] = None
    execution_id: Optional[str] = None
    
    def __post_init__(self):
        """验证字段的有效性"""
        if not 0.0 <= self.error_rate <= 1.0:
            raise ValueError(f"error_rate 必须在 [0, 1] 范围内，得到: {self.error_rate}")
        
        if self.timestamp is None:
            self.timestamp = datetime.now()


# ============================================================================
# 评分与分析相关 (Scoring & Analysis Models)
# ============================================================================

@dataclass
class ScoringBreakdown:
    """
    严重性评分的分解（用于调试和分析）。
    
    Attributes:
        bug_score: Bug 触发的分数 [0, 10]
        perf_score: 性能影响的分数 [0, 10]
        struct_score: 结构变化的分数 [0, 10]
        total_score: 加权后的总分 [0, 10]
        weights: 使用的权重
    """
    bug_score: float
    perf_score: float
    struct_score: float
    total_score: float
    weights: Dict[str, float]


@dataclass
class AnalyzerConfig:
    """
    Response Analyzer 的配置。
    
    这个类封装了所有与严重性评分相关的可配置参数。
    允许用户灵活调整评分策略。
    
    Attributes:
        weights: 各个评分维度的权重
                 - bug: Bug 触发权重（默认 10.0）
                 - perf: 性能影响权重（默认 2.0）
                 - struct: 结构变化权重（默认 5.0）
        baseline_latency_ms: 基线延迟（毫秒），用于计算性能影响
                             默认: 200ms
        threshold_latency_ms: 延迟阈值，超过此值视为"严重"性能问题
                             默认: 1000ms
        baseline_trace: 基线分布式追踪数据，用于与当前 Trace 比对
                        格式: Jaeger/OpenTelemetry JSON
        fault_type_configs: 故障类型特定的配置覆盖（可选）
                           允许不同故障类型使用不同的权重和阈值
    """
    weights: Dict[str, float] = field(
        default_factory=lambda: {
            "bug": 10.0,
            "perf": 2.0,
            "struct": 5.0,
        }
    )
    baseline_latency_ms: float = 200.0
    threshold_latency_ms: float = 1000.0
    baseline_trace: Optional[Dict[str, Any]] = None
    fault_type_configs: Dict[FaultType, Dict[str, Any]] = field(default_factory=dict)
    
    def get_weight(self, dimension: str, fault_type: Optional[FaultType] = None) -> float:
        """
        获取指定维度的权重。
        
        如果指定了 fault_type 且在 fault_type_configs 中有特定配置，
        则使用故障类型特定的权重；否则使用全局权重。
        """
        if fault_type and fault_type in self.fault_type_configs:
            config = self.fault_type_configs[fault_type]
            if "weights" in config and dimension in config["weights"]:
                return config["weights"][dimension]
        
        return self.weights.get(dimension, 1.0)


# ============================================================================
# 优化器相关 (Optimizer Models)
# ============================================================================

@dataclass
class BestResult:
    """
    优化器找到的最佳结果。
    
    Attributes:
        best_plan: 最佳故障注入计划
        best_score: 最高严重性评分
        iteration: 找到最佳结果时的迭代次数
        timestamp: 找到最佳结果的时间
    """
    best_plan: Optional[FaultPlan]
    best_score: float
    iteration: int = 0
    timestamp: Optional[datetime] = None


# ============================================================================
# 会话相关 (Session Models)
# ============================================================================

@dataclass
class OptimizationProgress:
    """
    优化进度信息。
    
    Attributes:
        completed_trials: 已完成的实验次数
        total_trials: 总实验预算
        best_result: 迄今为止的最佳结果
        current_iteration: 当前迭代号
    """
    completed_trials: int
    total_trials: int
    best_result: BestResult
    current_iteration: int = 0
    
    @property
    def progress_percentage(self) -> float:
        """返回完成百分比 [0, 100]"""
        if self.total_trials == 0:
            return 0.0
        return (self.completed_trials / self.total_trials) * 100


@dataclass
class SessionState:
    """
    优化会话的状态快照。
    
    Attributes:
        session_id: 会话唯一标识
        status: 会话状态
        progress: 优化进度
        config: 搜索空间和分析器配置
        created_at: 创建时间
        updated_at: 最后更新时间
        error_message: 如果失败，记录错误信息
    """
    session_id: str
    status: SessionStatus
    progress: OptimizationProgress
    config: Dict[str, Any]  # 包含 search_space_config 和 analysis_config
    created_at: datetime
    updated_at: datetime
    error_message: Optional[str] = None


# ============================================================================
# 搜索空间相关 (Search Space Models)
# ============================================================================

@dataclass
class SearchSpaceDimension:
    """
    搜索空间的一个维度定义。
    
    Attributes:
        name: 维度名称（如 "delay_seconds", "service", "percentage"）
        dimension_type: 维度类型 ("real", "integer", "categorical")
        bounds: 对于 real/integer，指定 [min, max]；对于 categorical，指定 [value1, value2, ...]
        default: 默认值
        depend_on: 条件依赖（可选），如 {"fault_type": "delay"}
                  表示只在 fault_type="delay" 时有效
    """
    name: str
    dimension_type: str  # "real", "integer", "categorical"
    bounds: List[Any]  # [min, max] for real/integer, [val1, val2, ...] for categorical
    default: Any = None
    depend_on: Optional[Dict[str, str]] = None


@dataclass
class SearchSpaceConfig:
    """
    搜索空间的完整定义。
    
    Attributes:
        dimensions: 所有维度的列表
        constraints: 额外的约束条件（可选）
    """
    dimensions: List[SearchSpaceDimension]
    constraints: Optional[List[Dict[str, Any]]] = None


# ============================================================================
# 执行器通信相关 (Executor Communication Models)
# ============================================================================

@dataclass
class ExecutorConfig:
    """
    执行器（Executor）的连接配置。
    
    Attributes:
        endpoint: 执行器 API 的基础 URL（如 "http://hfi-control-plane:8080"）
        timeout_seconds: 单个执行的超时时间（秒）
        max_retries: 失败重试次数
        retry_backoff_factor: 重试的指数退避因子
    """
    endpoint: str
    timeout_seconds: float = 60.0
    max_retries: int = 3
    retry_backoff_factor: float = 2.0


# ============================================================================
# 冷启动相关 (Cold Start Models)
# ============================================================================

@dataclass
class ColdStartConfig:
    """
    冷启动（初始化阶段）的配置。
    
    Attributes:
        n_initial_points: 冷启动阶段生成的随机点数量
        strategy: 采样策略 ("random", "lhs" 拉丁超立方体采样)
        warm_start_points: 预热点（从历史或经验中导入的已知好点）
    """
    n_initial_points: int = 10
    strategy: str = "random"  # "random", "lhs"
    warm_start_points: List[FaultPlan] = field(default_factory=list)


# ============================================================================
# 测试和示例数据
# ============================================================================

def create_sample_fault_plan() -> FaultPlan:
    """创建一个示例故障注入计划"""
    return FaultPlan(
        service="PaymentService",
        api="/api/v1/payment",
        fault_type=FaultType.DELAY,
        percentage=50,
        delay_seconds=2.5,
    )


def create_sample_raw_observation() -> RawObservation:
    """创建一个示例原始观测数据"""
    return RawObservation(
        status_code=503,
        latency_ms=3500.0,
        error_rate=0.45,
        success_count=55,
        error_count=45,
        trace_data={
            "traceID": "abc123",
            "spans": [
                {"spanID": "1", "operationName": "http.request", "duration": 3500000},
                {"spanID": "2", "operationName": "db.query", "duration": 2000000, "error": True},
            ]
        },
        logs=[
            "2025-11-11T10:00:00Z ERROR Payment service timeout",
            "2025-11-11T10:00:02Z INFO Retrying payment request",
        ],
        execution_id="exec-abc123",
    )


if __name__ == "__main__":
    # 测试数据模型的正确性
    print("=== 测试 FaultPlan ===")
    plan = create_sample_fault_plan()
    print(f"Plan: {plan}")
    print(f"Plan as dict: {plan.to_dict()}")
    print(f"Plan from dict: {FaultPlan.from_dict(plan.to_dict())}")
    
    print("\n=== 测试 RawObservation ===")
    obs = create_sample_raw_observation()
    print(f"Observation: {obs}")
    print(f"Error rate: {obs.error_rate}")
    
    print("\n=== 测试 AnalyzerConfig ===")
    config = AnalyzerConfig()
    print(f"Config: {config}")
    print(f"Bug weight: {config.get_weight('bug')}")
    print(f"Perf weight for DELAY: {config.get_weight('perf', FaultType.DELAY)}")
    
    print("\n=== 测试 OptimizationProgress ===")
    best = BestResult(best_plan=plan, best_score=7.5, iteration=25)
    progress = OptimizationProgress(
        completed_trials=25,
        total_trials=100,
        best_result=best,
        current_iteration=25,
    )
    print(f"Progress: {progress}")
    print(f"Progress percentage: {progress.progress_percentage:.1f}%")
    
    print("\n✓ 所有数据模型测试通过!")
