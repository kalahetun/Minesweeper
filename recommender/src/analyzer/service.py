"""
Response Analyzer 的评分函数实现

本模块实现了技术规格中定义的加权严重性评分函数 f(x)。

评分公式：
    y = (w_bug * Score_Bug + w_perf * Score_Perf + w_struct * Score_Struct) 
        / (w_bug + w_perf + w_struct)

其中：
    - Score_Bug: Bug 触发的分数，范围 [0, 10]
    - Score_Perf: 性能影响的分数，范围 [0, 10]
    - Score_Struct: 系统结构变化的分数，范围 [0, 10]
    - w_bug, w_perf, w_struct: 权重，用户可配置

最终结果范围：[0, 10]，其中 10 = 最严重
"""

import math
import logging
from typing import Dict, Any, Optional, List, Tuple
from dataclasses import dataclass
from src.types import (
    RawObservation, AnalyzerConfig, ScoringBreakdown, FaultType
)

logger = logging.getLogger(__name__)


# ============================================================================
# 评分函数的详细设计
# ============================================================================

"""
评分函数的设计思路：

1. Bug 触发评分 (Score_Bug) - 范围 [0, 10]
   ============================================
   
   检查故障注入是否触发了明确的系统故障或异常。
   
   评分规则：
   - 如果 status_code >= 500：说明服务出现了严重问题
     分数 = 10.0（最严重）
   
   - 如果 status_code >= 400：说明请求失败
     分数 = 8.0（较严重）
   
   - 如果 logs 中包含 ERROR 或 FATAL 标签：
     分数 = 6.0（中等）
   
   - 如果 error_rate > 0：至少有请求失败
     分数 = 3.0（轻微）
   
   - 否则：没有明确的故障迹象
     分数 = 0.0（无影响）
   
   实现逻辑：
   ```python
   if status_code >= 500:
       return 10.0
   elif status_code >= 400:
       return 8.0
   elif has_critical_logs:
       return 6.0
   elif error_rate > 0:
       return 3.0
   else:
       return 0.0
   ```

2. 性能影响评分 (Score_Perf) - 范围 [0, 10]
   ==========================================
   
   衡量故障注入对系统延迟的影响程度。
   
   核心指标：
   - baseline_latency: 无故障下的基线延迟（如 200ms）
   - threshold_latency: 可接受的延迟上限（如 1000ms）
   - actual_latency: 故障注入下的实际延迟
   
   评分逻辑：
   - 延迟增加倍数 = actual_latency / baseline_latency
   
   - 如果 actual_latency > threshold_latency：分数 = 10.0
   - 否则：分数 = min(9.0, (actual_latency - baseline_latency) / (threshold_latency - baseline_latency) * 9.0)
   
   数学形式：
   ```
   delay_ratio = (actual_latency - baseline_latency) / baseline_latency
   score = min(10.0, delay_ratio / 5.0 * 10.0)  // 延迟增加 5 倍 → 分数 10
   ```
   
   示例：
   - baseline_latency = 200ms
   - threshold_latency = 1000ms
   - actual_latency = 3500ms
   → delay_ratio = (3500 - 200) / 200 = 16.5
   → score = min(10.0, 16.5 / 5.0 * 10.0) = 10.0

3. 结构变化评分 (Score_Struct) - 范围 [0, 10]
   =============================================
   
   检查故障注入是否改变了系统的行为模式（如重试、降级、熔断等）。
   
   通过比较基线分布式追踪 (Trace) 和实验中的 Trace 来检测。
   
   检测项：
   1. Span 数量变化：可能表示重试或提前返回
      - 如果 current_spans > baseline_spans * 1.5：+3.0 分
   
   2. 操作序列变化：可能表示控制流改变（降级）
      - 使用编辑距离 (edit_distance) 比较 Span 名称序列
      - 如果距离 > 2：+5.0 分
   
   3. 错误标签：Span 中包含错误标记
      - 任何 Span 有 error=true：+2.0 分
   
   4. 延迟异常：某个 Span 比基线慢很多
      - 如果某个 Span 比基线慢 5 倍：+2.0 分
   
   总分 = max(这些子项的分数)  // 取最严重的变化
   
   数学形式：
   ```python
   changes = []
   if span_count_change > 50%:
       changes.append(3.0)
   if edit_distance > 2:
       changes.append(5.0)
   if has_error_spans:
       changes.append(2.0)
   if span_latency_spike > 5x:
       changes.append(2.0)
   
   score = max(changes) if changes else 0.0
   score = min(10.0, score)  // 上限 10
   ```

加权求和的规范化：
   为了避免权重对最终分数的影响，采用加权平均而非简单加权求和。
   
   最终分数 = (w_bug * Score_Bug + w_perf * Score_Perf + w_struct * Score_Struct) 
              / (w_bug + w_perf + w_struct)
   
   这样，无论权重怎么设置，最终分数仍然在 [0, 10] 范围内。

Fail-Safe 原则：
   如果某个评分子函数因为数据缺失或异常而失败，应该：
   1. 记录警告日志
   2. 返回该子函数的默认值（通常是 0.0）
   3. 继续计算其他评分维度
   4. 确保优化循环不会因为数据问题而中断
"""


# ============================================================================
# Bug 评分器
# ============================================================================

class BugScorer:
    """
    Bug 触发评分器
    
    检查故障注入是否触发了系统故障。
    """
    
    @staticmethod
    def score(observation: RawObservation, config: AnalyzerConfig) -> float:
        """
        计算 Bug 触发的分数。
        
        Args:
            observation: 原始观测数据
            config: 分析器配置
        
        Returns:
            分数，范围 [0, 10]
        """
        try:
            # 规则 1: HTTP 5xx 状态码（服务器错误）
            if observation.status_code >= 500:
                logger.info(f"检测到服务器错误: HTTP {observation.status_code}")
                return 10.0
            
            # 规则 2: HTTP 4xx 状态码（客户端错误或请求被拒绝）
            if observation.status_code >= 400:
                logger.info(f"检测到请求失败: HTTP {observation.status_code}")
                return 8.0
            
            # 规则 3: 日志中包含关键错误标签
            if observation.logs:
                critical_keywords = ["ERROR", "FATAL", "CRITICAL", "PANIC", "EXCEPTION"]
                has_critical_log = any(
                    keyword in log for log in observation.logs
                    for keyword in critical_keywords
                )
                if has_critical_log:
                    logger.info("在日志中检测到关键错误")
                    return 6.0
            
            # 规则 4: 有请求失败
            if observation.error_rate > 0:
                logger.info(f"检测到请求失败率: {observation.error_rate:.1%}")
                return 3.0
            
            # 规则 5: 没有明确的故障迹象
            return 0.0
        
        except Exception as e:
            logger.warning(f"BugScorer 计算异常: {e}，返回默认值 0.0")
            return 0.0


# ============================================================================
# 性能评分器
# ============================================================================

class PerformanceScorer:
    """
    性能影响评分器
    
    衡量故障注入对系统延迟的影响。
    """
    
    @staticmethod
    def score(observation: RawObservation, config: AnalyzerConfig) -> float:
        """
        计算性能影响的分数。
        
        Args:
            observation: 原始观测数据
            config: 分析器配置（含 baseline_latency_ms 和 threshold_latency_ms）
        
        Returns:
            分数，范围 [0, 10]
        """
        try:
            baseline = config.baseline_latency_ms
            threshold = config.threshold_latency_ms
            actual = observation.latency_ms
            
            if baseline <= 0:
                logger.warning("基线延迟无效，返回默认值 0.0")
                return 0.0
            
            # 如果实际延迟已经超过阈值，评分最高
            if actual > threshold:
                logger.info(f"延迟超过阈值: {actual:.1f}ms > {threshold:.1f}ms")
                return 10.0
            
            # 否则，按比例计算分数
            # 分数 = (actual - baseline) / (threshold - baseline) * 9.0
            # 这样，延迟在 baseline 时分数为 0，在 threshold 时分数为 9
            if threshold > baseline:
                score = (actual - baseline) / (threshold - baseline) * 9.0
            else:
                score = 0.0
            
            # 上限为 10
            score = min(10.0, max(0.0, score))
            
            delay_increase = (actual - baseline) / baseline if baseline > 0 else 0
            logger.info(f"性能影响: 延迟增加 {delay_increase:.1f}x，评分 {score:.2f}")
            
            return score
        
        except Exception as e:
            logger.warning(f"PerformanceScorer 计算异常: {e}，返回默认值 0.0")
            return 0.0


# ============================================================================
# 结构变化评分器
# ============================================================================

class StructureScorer:
    """
    结构变化评分器
    
    通过分析分布式追踪 (Trace)，检测系统行为的变化（如重试、降级等）。
    """
    
    @staticmethod
    def score(observation: RawObservation, config: AnalyzerConfig) -> float:
        """
        计算结构变化的分数。
        
        Args:
            observation: 原始观测数据
            config: 分析器配置（含 baseline_trace）
        
        Returns:
            分数，范围 [0, 10]
        """
        try:
            # 检查是否有基线 Trace
            if not config.baseline_trace:
                logger.warning("未配置基线 Trace，无法检测结构变化")
                return 0.0
            
            # 检查当前 Trace 是否有效
            if not observation.trace_data:
                logger.warning("当前观测中没有 Trace 数据")
                return 0.0
            
            scores = []
            
            # 子检测 1: Span 数量变化
            span_score = StructureScorer._check_span_count_change(
                observation.trace_data, config.baseline_trace
            )
            if span_score > 0:
                scores.append(span_score)
            
            # 子检测 2: 操作序列变化
            seq_score = StructureScorer._check_operation_sequence_change(
                observation.trace_data, config.baseline_trace
            )
            if seq_score > 0:
                scores.append(seq_score)
            
            # 子检测 3: 错误标签
            error_score = StructureScorer._check_error_spans(observation.trace_data)
            if error_score > 0:
                scores.append(error_score)
            
            # 子检测 4: 延迟异常
            latency_score = StructureScorer._check_span_latency_spike(
                observation.trace_data, config.baseline_trace
            )
            if latency_score > 0:
                scores.append(latency_score)
            
            # 取最严重的变化
            final_score = max(scores) if scores else 0.0
            final_score = min(10.0, final_score)
            
            if final_score > 0:
                logger.info(f"检测到结构变化，评分 {final_score:.2f}")
            
            return final_score
        
        except Exception as e:
            logger.warning(f"StructureScorer 计算异常: {e}，返回默认值 0.0")
            return 0.0
    
    @staticmethod
    def _check_span_count_change(current_trace: Dict, baseline_trace: Dict) -> float:
        """检查 Span 数量的变化（可能表示重试）"""
        try:
            baseline_spans = len(baseline_trace.get("spans", []))
            current_spans = len(current_trace.get("spans", []))
            
            if baseline_spans == 0:
                return 0.0
            
            # 如果当前 Span 数 > baseline 的 1.5 倍，表示有重试
            if current_spans > baseline_spans * 1.5:
                logger.info(f"检测到 Span 数量增加: {baseline_spans} → {current_spans}")
                return 3.0
            
            return 0.0
        except Exception as e:
            logger.warning(f"检查 Span 数量变化失败: {e}")
            return 0.0
    
    @staticmethod
    def _check_operation_sequence_change(current_trace: Dict, baseline_trace: Dict) -> float:
        """检查操作序列的变化（可能表示降级或控制流改变）"""
        try:
            baseline_ops = [s.get("operationName", "") for s in baseline_trace.get("spans", [])]
            current_ops = [s.get("operationName", "") for s in current_trace.get("spans", [])]
            
            if not baseline_ops or not current_ops:
                return 0.0
            
            # 计算编辑距离
            distance = StructureScorer._edit_distance(baseline_ops, current_ops)
            
            # 如果编辑距离 > 2，表示操作序列有明显变化
            if distance > 2:
                logger.info(f"检测到操作序列变化: 编辑距离 = {distance}")
                return 5.0
            
            return 0.0
        except Exception as e:
            logger.warning(f"检查操作序列变化失败: {e}")
            return 0.0
    
    @staticmethod
    def _check_error_spans(current_trace: Dict) -> float:
        """检查 Trace 中是否有错误标记的 Span"""
        try:
            spans = current_trace.get("spans", [])
            
            for span in spans:
                if span.get("error") or span.get("tags", {}).get("error") == "true":
                    logger.info("检测到错误 Span")
                    return 2.0
            
            return 0.0
        except Exception as e:
            logger.warning(f"检查错误 Span 失败: {e}")
            return 0.0
    
    @staticmethod
    def _check_span_latency_spike(current_trace: Dict, baseline_trace: Dict) -> float:
        """检查是否有 Span 延迟异常（比基线慢 5 倍以上）"""
        try:
            baseline_spans = {s.get("operationName"): s.get("duration", 0) 
                             for s in baseline_trace.get("spans", [])}
            current_spans = {s.get("operationName"): s.get("duration", 0) 
                            for s in current_trace.get("spans", [])}
            
            for op_name, current_duration in current_spans.items():
                if op_name in baseline_spans:
                    baseline_duration = baseline_spans[op_name]
                    if baseline_duration > 0:
                        ratio = current_duration / baseline_duration
                        if ratio > 5.0:
                            logger.info(f"检测到 Span 延迟异常: {op_name} 延迟 {ratio:.1f}x")
                            return 2.0
            
            return 0.0
        except Exception as e:
            logger.warning(f"检查 Span 延迟异常失败: {e}")
            return 0.0
    
    @staticmethod
    def _edit_distance(list1: List[str], list2: List[str]) -> int:
        """计算两个列表之间的编辑距离（Levenshtein Distance）"""
        m, n = len(list1), len(list2)
        dp = [[0] * (n + 1) for _ in range(m + 1)]
        
        for i in range(m + 1):
            dp[i][0] = i
        for j in range(n + 1):
            dp[0][j] = j
        
        for i in range(1, m + 1):
            for j in range(1, n + 1):
                if list1[i - 1] == list2[j - 1]:
                    dp[i][j] = dp[i - 1][j - 1]
                else:
                    dp[i][j] = 1 + min(dp[i - 1][j], dp[i][j - 1], dp[i - 1][j - 1])
        
        return dp[m][n]


# ============================================================================
# 主分析器服务
# ============================================================================

class AnalyzerService:
    """
    Response Analyzer 的主服务。
    
    负责整合三个评分器，计算最终的严重性评分。
    """
    
    def __init__(self, config: AnalyzerConfig):
        """
        初始化分析器服务。
        
        Args:
            config: 分析器配置
        """
        self.config = config
        self.bug_scorer = BugScorer()
        self.perf_scorer = PerformanceScorer()
        self.struct_scorer = StructureScorer()
    
    def calculate_severity(
        self, observation: RawObservation
    ) -> Tuple[float, Optional[ScoringBreakdown]]:
        """
        计算严重性评分。
        
        Args:
            observation: 原始观测数据
        
        Returns:
            (总分数, 分解信息) 或 (总分数, None) 如果缺少分解信息
        
        Raises:
            ValueError: 输入数据无效
        """
        if observation is None:
            raise ValueError("observation 不能为 None")
        
        # 计算三个分项评分
        score_bug = self.bug_scorer.score(observation, self.config)
        score_perf = self.perf_scorer.score(observation, self.config)
        score_struct = self.struct_scorer.score(observation, self.config)
        
        # 获取权重（考虑故障类型特定的配置）
        # 注: 这里假设 observation 中有 fault_type 信息
        fault_type = None  # 从其他地方获取
        w_bug = self.config.get_weight("bug", fault_type)
        w_perf = self.config.get_weight("perf", fault_type)
        w_struct = self.config.get_weight("struct", fault_type)
        
        # 加权平均（规范化）
        total_weight = w_bug + w_perf + w_struct
        if total_weight == 0:
            logger.warning("权重之和为 0，返回 0 分")
            return 0.0, None
        
        total_score = (
            w_bug * score_bug +
            w_perf * score_perf +
            w_struct * score_struct
        ) / total_weight
        
        # 确保分数在 [0, 10] 范围内
        total_score = min(10.0, max(0.0, total_score))
        
        # 构建分解信息（用于调试）
        breakdown = ScoringBreakdown(
            bug_score=score_bug,
            perf_score=score_perf,
            struct_score=score_struct,
            total_score=total_score,
            weights={"bug": w_bug, "perf": w_perf, "struct": w_struct},
        )
        
        logger.info(
            f"严重性评分: Bug={score_bug:.2f}, Perf={score_perf:.2f}, "
            f"Struct={score_struct:.2f} → 总分={total_score:.2f}"
        )
        
        return total_score, breakdown


# ============================================================================
# 测试与示例
# ============================================================================

def test_scoring_functions():
    """测试所有评分函数"""
    from src.types import create_sample_raw_observation, AnalyzerConfig
    
    print("=== 测试评分函数 ===\n")
    
    # 创建测试数据
    obs = create_sample_raw_observation()
    config = AnalyzerConfig(
        baseline_latency_ms=200.0,
        threshold_latency_ms=1000.0,
    )
    
    # 测试各个评分器
    print(f"1. Bug 评分器")
    bug_score = BugScorer.score(obs, config)
    print(f"   得分: {bug_score:.2f}\n")
    
    print(f"2. 性能评分器")
    perf_score = PerformanceScorer.score(obs, config)
    print(f"   得分: {perf_score:.2f}\n")
    
    print(f"3. 结构评分器")
    struct_score = StructureScorer.score(obs, config)
    print(f"   得分: {struct_score:.2f}\n")
    
    # 测试主服务
    print(f"4. 主分析器服务")
    service = AnalyzerService(config)
    total_score, breakdown = service.calculate_severity(obs)
    print(f"   总分: {total_score:.2f}")
    if breakdown:
        print(f"   分解: Bug={breakdown.bug_score:.2f}, Perf={breakdown.perf_score:.2f}, "
              f"Struct={breakdown.struct_score:.2f}")
    
    print("\n✓ 测试完成")


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    test_scoring_functions()
