"""
搜索空间配置与转换 (Search Space Configuration & Conversion)

本模块负责：
1. 定义搜索空间的 YAML/JSON Schema
2. 将用户友好的配置转换为 scikit-optimize 能理解的格式
3. 处理条件维度（如："delay_seconds 只在 fault_type='delay' 时有效"）

设计原则：
- 配置优先于代码：所有搜索空间定义都来自 YAML/JSON 配置
- 灵活性：支持条件维度和复杂的约束
- 可理解性：配置文件应该易于人工编辑和理解
"""

import yaml
import json
from typing import List, Dict, Any, Optional
from dataclasses import dataclass
import skopt.space
from src.types import SearchSpaceDimension, SearchSpaceConfig


# ============================================================================
# 配置 Schema 定义（文档与注释）
# ============================================================================

"""
搜索空间配置文件的 Schema 示例：

```yaml
# fault_space_config.yaml

# 基本信息
name: "HTTP Fault Injection Search Space"
description: "搜索最致命的故障注入组合"

# 维度定义
dimensions:
  # 维度 1: 故障类型（分类）
  - name: "fault_type"
    type: "categorical"
    values: ["delay", "abort", "error_injection"]
    default: "delay"
    required: true
  
  # 维度 2: 目标服务（分类）
  - name: "service"
    type: "categorical"
    values: ["PaymentService", "OrderService", "InventoryService", "NotificationService"]
    default: "PaymentService"
    required: true
  
  # 维度 3: 目标 API 路径（分类）
  - name: "api"
    type: "categorical"
    values: 
      - "/api/v1/payment"
      - "/api/v1/order"
      - "/api/v1/inventory"
      - "/api/v1/notify"
    default: "/api/v1/payment"
    required: true
  
  # 维度 4: 注入百分比（整数）
  - name: "percentage"
    type: "integer"
    min: 1
    max: 100
    default: 50
    required: true
  
  # 维度 5: 延迟时间（浮点）
  # 条件：仅当 fault_type == "delay" 时有效
  - name: "delay_seconds"
    type: "real"
    min: 0.1
    max: 30.0
    default: 2.0
    required: false
    condition:
      field: "fault_type"
      value: "delay"
  
  # 维度 6: 中止时的 HTTP 状态码（整数）
  # 条件：仅当 fault_type == "abort" 时有效
  - name: "abort_http_status"
    type: "integer"
    min: 400
    max: 599
    default: 503
    required: false
    condition:
      field: "fault_type"
      value: "abort"
  
  # 维度 7: 重试次数（整数）
  # 全局维度，所有情况都有效
  - name: "retry_count"
    type: "integer"
    min: 0
    max: 5
    default: 0
    required: false

# 条件维度处理策略
# 策略 1: "expand" - 展开所有维度到搜索空间，执行器忽略不相关的参数
# 策略 2: "filter" - 动态过滤维度，只包含在当前条件下有效的维度
# 策略 3: "encode" - 编码条件逻辑到参数中（最复杂）
conditional_strategy: "expand"  # 默认使用展开策略

# 约束条件（可选）
constraints:
  # 约束 1: 不同的服务不能同时注入故障
  - type: "mutually_exclusive"
    dimensions: ["service"]
    message: "一次只能对一个服务注入故障"
  
  # 约束 2: 百分比和延迟的组合约束
  - type: "custom"
    expression: "percentage <= 50 or delay_seconds <= 2.0"
    message: "如果百分比 > 50%，则延迟必须 <= 2秒"

```

配置文件的关键特性：
1. YAML 格式，易于编辑和版本控制
2. 支持条件维度（通过 condition 字段）
3. 支持多种处理策略（expand, filter, encode）
4. 支持约束条件（constraints）
"""


# ============================================================================
# SpaceConverter 类
# ============================================================================

class SpaceConverter:
    """
    将用户配置的搜索空间转换为 scikit-optimize 能理解的格式。
    
    工作流程：
    1. 加载 YAML/JSON 配置文件
    2. 验证配置的合法性
    3. 根据条件策略处理条件维度
    4. 转换为 skopt.space.Dimension 对象列表
    5. 生成维度名称与索引的映射（用于字典 ↔ 列表的转换）
    """
    
    def __init__(self, config_file: str, conditional_strategy: str = "expand"):
        """
        初始化 SpaceConverter。
        
        Args:
            config_file: 配置文件路径（YAML 或 JSON）
            conditional_strategy: 条件维度的处理策略
                                 - "expand": 展开所有维度
                                 - "filter": 动态过滤
                                 - "encode": 编码（最复杂）
        """
        self.config_file = config_file
        self.conditional_strategy = conditional_strategy
        self.config = self._load_config()
        
        # 验证配置
        self._validate_config()
        
        # 转换为 skopt 格式
        self.space_config = self._convert_to_skopt_space()
        self.space_dimensions = self.space_config.dimensions
        
        # 生成维度名称与索引的映射
        self.dimension_names = [dim.name for dim in self.space_dimensions]
        self.name_to_index = {name: idx for idx, name in enumerate(self.dimension_names)}
        self.index_to_name = {idx: name for idx, name in enumerate(self.dimension_names)}
    
    def _load_config(self) -> Dict[str, Any]:
        """
        从 YAML 或 JSON 文件加载配置。
        
        Returns:
            配置字典
        
        Raises:
            FileNotFoundError: 文件不存在
            ValueError: 文件格式不支持
        """
        if self.config_file.endswith('.yaml') or self.config_file.endswith('.yml'):
            with open(self.config_file, 'r', encoding='utf-8') as f:
                return yaml.safe_load(f)
        elif self.config_file.endswith('.json'):
            with open(self.config_file, 'r', encoding='utf-8') as f:
                return json.load(f)
        else:
            raise ValueError(f"不支持的配置文件格式: {self.config_file}")
    
    def _validate_config(self) -> None:
        """
        验证配置的合法性。
        
        Raises:
            ValueError: 配置有效性检查失败
        """
        if 'dimensions' not in self.config:
            raise ValueError("配置文件缺少 'dimensions' 字段")
        
        if not isinstance(self.config['dimensions'], list):
            raise ValueError("'dimensions' 必须是一个列表")
        
        if len(self.config['dimensions']) == 0:
            raise ValueError("'dimensions' 不能为空")
        
        # 检查维度名称的唯一性
        dim_names = [dim['name'] for dim in self.config['dimensions']]
        if len(dim_names) != len(set(dim_names)):
            raise ValueError("维度名称必须唯一")
        
        # 验证每个维度
        for i, dim in enumerate(self.config['dimensions']):
            self._validate_dimension(dim, i)
    
    def _validate_dimension(self, dim: Dict[str, Any], index: int) -> None:
        """
        验证单个维度的配置。
        
        Args:
            dim: 维度配置字典
            index: 维度在列表中的索引
        
        Raises:
            ValueError: 维度配置无效
        """
        required_fields = ['name', 'type']
        for field in required_fields:
            if field not in dim:
                raise ValueError(f"维度 #{index} 缺少必需字段: {field}")
        
        dim_name = dim['name']
        dim_type = dim['type']
        
        if dim_type not in ['categorical', 'real', 'integer']:
            raise ValueError(f"维度 '{dim_name}' 的 type 必须是 categorical/real/integer")
        
        if dim_type in ['real', 'integer']:
            if 'min' not in dim or 'max' not in dim:
                raise ValueError(f"维度 '{dim_name}' 缺少 min 或 max")
            if dim['min'] >= dim['max']:
                raise ValueError(f"维度 '{dim_name}' 的 min 必须 < max")
        
        if dim_type == 'categorical':
            if 'values' not in dim:
                raise ValueError(f"维度 '{dim_name}' 缺少 'values'")
            if len(dim['values']) < 2:
                raise ValueError(f"维度 '{dim_name}' 至少需要 2 个可选值")
    
    def _convert_to_skopt_space(self) -> SearchSpaceConfig:
        """
        将配置转换为 scikit-optimize 格式。
        
        Returns:
            SearchSpaceConfig 对象
        """
        skopt_dimensions = []
        
        if self.conditional_strategy == "expand":
            # 策略 1: 展开所有维度（包括条件维度）
            for raw_dim in self.config['dimensions']:
                skopt_dim = self._convert_single_dimension(raw_dim)
                skopt_dimensions.append(skopt_dim)
        
        elif self.conditional_strategy == "filter":
            # 策略 2: 仅包含无条件维度
            # 注: 这种策略需要在运行时动态修改搜索空间，实现复杂
            for raw_dim in self.config['dimensions']:
                if 'condition' not in raw_dim:
                    skopt_dim = self._convert_single_dimension(raw_dim)
                    skopt_dimensions.append(skopt_dim)
        
        elif self.conditional_strategy == "encode":
            # 策略 3: 编码条件逻辑到参数中（未实现）
            raise NotImplementedError("encode 策略尚未实现")
        
        return SearchSpaceConfig(dimensions=skopt_dimensions)
    
    def _convert_single_dimension(self, raw_dim: Dict[str, Any]) -> SearchSpaceDimension:
        """
        将单个原始维度配置转换为 SearchSpaceDimension。
        
        Args:
            raw_dim: 原始维度配置字典
        
        Returns:
            SearchSpaceDimension 对象
        """
        name = raw_dim['name']
        dim_type = raw_dim['type']
        
        if dim_type == 'categorical':
            bounds = raw_dim['values']
        elif dim_type == 'real':
            bounds = [raw_dim['min'], raw_dim['max']]
        elif dim_type == 'integer':
            bounds = [raw_dim['min'], raw_dim['max']]
        
        return SearchSpaceDimension(
            name=name,
            dimension_type=dim_type,
            bounds=bounds,
            default=raw_dim.get('default'),
            depend_on=raw_dim.get('condition'),
        )
    
    def convert_to_skopt_dimensions(self) -> List[skopt.space.Dimension]:
        """
        将 SearchSpaceDimension 转换为 scikit-optimize 的 Dimension 对象。
        
        Returns:
            skopt 能理解的 Dimension 对象列表
        
        Raises:
            ValueError: 维度配置无效
        """
        skopt_dims = []
        
        for dim in self.space_dimensions:
            try:
                if dim.dimension_type == 'categorical':
                    skopt_dim = skopt.space.Categorical(
                        categories=dim.bounds,
                        name=dim.name,
                    )
                elif dim.dimension_type == 'real':
                    skopt_dim = skopt.space.Real(
                        low=dim.bounds[0],
                        high=dim.bounds[1],
                        name=dim.name,
                    )
                elif dim.dimension_type == 'integer':
                    skopt_dim = skopt.space.Integer(
                        low=dim.bounds[0],
                        high=dim.bounds[1],
                        name=dim.name,
                    )
                else:
                    raise ValueError(f"未知的维度类型: {dim.dimension_type}")
                
                skopt_dims.append(skopt_dim)
            
            except Exception as e:
                raise ValueError(f"转换维度 '{dim.name}' 失败: {e}")
        
        return skopt_dims
    
    def dict_to_list(self, point_dict: Dict[str, Any]) -> List[Any]:
        """
        将字典格式的点转换为列表格式（用于 skopt）。
        
        Args:
            point_dict: 字典格式的点，如 {"service": "PaymentService", "percentage": 50, ...}
        
        Returns:
            列表格式的点，如 ["PaymentService", 50, ...]
        
        Raises:
            KeyError: 缺少必需的维度
            ValueError: 维度值的类型不匹配
        """
        point_list = [None] * len(self.dimension_names)
        
        for name, value in point_dict.items():
            if name not in self.name_to_index:
                # 允许字典中有额外的字段（忽略）
                continue
            
            idx = self.name_to_index[name]
            dim = self.space_dimensions[idx]
            
            # 类型检查和转换
            try:
                if dim.dimension_type == 'categorical':
                    if value not in dim.bounds:
                        raise ValueError(
                            f"维度 '{name}' 的值 '{value}' 不在允许的列表中: {dim.bounds}"
                        )
                elif dim.dimension_type in ['real', 'integer']:
                    value_float = float(value)
                    if not dim.bounds[0] <= value_float <= dim.bounds[1]:
                        raise ValueError(
                            f"维度 '{name}' 的值 {value} 不在范围 [{dim.bounds[0]}, {dim.bounds[1]}]"
                        )
                    if dim.dimension_type == 'integer':
                        value = int(value_float)
                    else:
                        value = value_float
            except ValueError as e:
                raise ValueError(f"维度 '{name}' 的值验证失败: {e}")
            
            point_list[idx] = value
        
        # 检查所有必需维度都已提供
        for idx, (name, value) in enumerate(zip(self.dimension_names, point_list)):
            if value is None:
                dim = self.space_dimensions[idx]
                if dim.default is not None:
                    point_list[idx] = dim.default
                else:
                    raise ValueError(f"缺少必需的维度: {name}")
        
        return point_list
    
    def list_to_dict(self, point_list: List[Any]) -> Dict[str, Any]:
        """
        将列表格式的点转换为字典格式。
        
        Args:
            point_list: 列表格式的点，如 ["PaymentService", 50, ...]
        
        Returns:
            字典格式的点，如 {"service": "PaymentService", "percentage": 50, ...}
        
        Raises:
            ValueError: 列表长度不匹配
        """
        if len(point_list) != len(self.dimension_names):
            raise ValueError(
                f"列表长度 {len(point_list)} 与维度数 {len(self.dimension_names)} 不匹配"
            )
        
        return {
            name: value
            for name, value in zip(self.dimension_names, point_list)
        }
    
    def get_space_info(self) -> Dict[str, Any]:
        """
        获取搜索空间的信息（用于调试和日志）。
        
        Returns:
            包含搜索空间信息的字典
        """
        info = {
            "config_file": self.config_file,
            "conditional_strategy": self.conditional_strategy,
            "num_dimensions": len(self.space_dimensions),
            "dimensions": []
        }
        
        for dim in self.space_dimensions:
            dim_info = {
                "name": dim.name,
                "type": dim.dimension_type,
            }
            if dim.dimension_type in ['real', 'integer']:
                dim_info["range"] = [dim.bounds[0], dim.bounds[1]]
            else:
                dim_info["values"] = dim.bounds
            info["dimensions"].append(dim_info)
        
        return info


# ============================================================================
# 三种条件维度处理策略的对比说明
# ============================================================================

"""
条件维度处理策略对比：

## 策略 1: Expand（展开式）
优点：
  - 实现最简单
  - 搜索空间定义清晰明了
  - 易于理解和调试

缺点：
  - 搜索空间变大（包含不相关的维度）
  - 效率可能较低
  - 优化器需要学会"忽略"不相关维度

使用场景：
  - 条件维度较少
  - 搜索空间不太大（< 1000）

实现方式：
  所有维度都加入搜索空间，执行器 Executor 忽略不相关的参数。


## 策略 2: Filter（过滤式）
优点：
  - 搜索空间更紧凑
  - 优化效率更高
  - 更符合实际需求

缺点：
  - 实现较复杂（需要动态修改搜索空间）
  - 需要跟踪条件变化

使用场景：
  - 条件维度较多
  - 搜索空间较大

实现方式：
  根据当前点的某些维度值，动态过滤出有效的维度。
  例如，如果 fault_type="delay"，则只搜索 delay_seconds；
  如果 fault_type="abort"，则只搜索 abort_http_status。


## 策略 3: Encode（编码式）
优点：
  - 搜索空间紧凑
  - 不需要动态修改搜索空间

缺点：
  - 实现最复杂
  - 容易出错
  - 难以理解和维护

使用场景：
  - 有大量复杂的条件依赖关系
  - 需要极高的搜索效率

实现方式：
  将条件逻辑编码到参数中，例如：
  - 单个参数编码多个概念
  - 使用编码的参数推导出实际的故障计划


推荐选择：
  初期（Phase 1）使用 Expand 策略，简单可靠。
  如果性能不满足需求，升级到 Filter 策略。
  通常不需要 Encode 策略（太复杂）。
"""


# ============================================================================
# 测试
# ============================================================================

def test_space_converter():
    """测试 SpaceConverter 的基本功能"""
    # 这个测试需要一个示例配置文件
    # 为了演示，这里仅展示调用方式
    
    # converter = SpaceConverter("config/fault_space_config.yaml", conditional_strategy="expand")
    # print(converter.get_space_info())
    
    # point_dict = {"service": "PaymentService", "percentage": 50, "delay_seconds": 2.0}
    # point_list = converter.dict_to_list(point_dict)
    # print(f"Dict -> List: {point_dict} -> {point_list}")
    
    # point_dict_back = converter.list_to_dict(point_list)
    # print(f"List -> Dict: {point_list} -> {point_dict_back}")
    
    print("SpaceConverter 测试方法已定义，需要配置文件才能运行")


if __name__ == "__main__":
    test_space_converter()
