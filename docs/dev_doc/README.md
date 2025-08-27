# HFI 开发者文档

本目录包含 HFI (HTTP Fault Injection) 系统的完整开发者文档。

## 📋 文档索引

### 核心架构文档
- **[ARCHITECTURE.md](./ARCHITECTURE.md)** - 系统宏观架构设计
  - 高级概述和系统上下文
  - 核心设计原则详解
  - 技术选型与理由分析
  - 完整数据流解析

### 详细设计文档
- **[design_doc/Design.md](./design_doc/Design.md)** - 总体设计文档
- **[design_doc/Design_1_Control_Plane.md](./design_doc/Design_1_Control_Plane.md)** - 控制平面详细设计
- **[design_doc/Design_2_Wasm_plugin.md](./design_doc/Design_2_Wasm_plugin.md)** - WASM 插件详细设计
- **[design_doc/Design_3_CLI.md](./design_doc/Design_3_CLI.md)** - CLI 工具详细设计

### 开发规划
- **[design_doc/Development_plan.md](./design_doc/Development_plan.md)** - 完整的开发计划和任务分解

## 🎯 阅读指南

### 新开发者入门路径
1. **开始**: [ARCHITECTURE.md](./ARCHITECTURE.md) - 了解系统整体架构
2. **深入**: [design_doc/Design.md](./design_doc/Design.md) - 学习详细设计方案
3. **实现**: 根据具体模块阅读对应的详细设计文档
4. **开发**: [design_doc/Development_plan.md](./design_doc/Development_plan.md) - 查看开发任务和进度

### 架构师审查路径
1. [ARCHITECTURE.md](./ARCHITECTURE.md) - 核心架构决策
2. [design_doc/Design.md](./design_doc/Design.md) - 技术选型和模块划分
3. 各模块详细设计文档 - 接口定义和实现方案

### 产品经理了解路径
1. [ARCHITECTURE.md](./ARCHITECTURE.md) 的"高级概述"部分
2. [design_doc/Development_plan.md](./design_doc/Development_plan.md) 的任务规划部分

## 🔧 开发环境

### 前置条件
- Go 1.24+
- Rust 1.89+
- Docker
- Kubernetes 集群 (kind/minikube/k3s)

### 快速开始
```bash
# 克隆项目
git clone <repository-url>
cd hfi

# 查看快速开始指南
cat QUICKSTART.md

# 构建所有组件
make build-all
```

## 📝 文档维护

### 更新原则
- **架构变更**: 必须更新 ARCHITECTURE.md
- **接口变更**: 必须更新对应的详细设计文档
- **新功能**: 必须更新 Development_plan.md 的任务状态

### 文档格式
- 使用 Markdown 格式
- 架构图使用 Mermaid 语法
- 代码示例使用相应语言的语法高亮

### 审查流程
所有文档变更都需要通过 Pull Request 进行审查，确保：
- 技术准确性
- 描述清晰性
- 格式一致性

---

## 📞 联系方式

如有任何技术问题或文档改进建议，请：
- 提交 GitHub Issue
- 发起 GitHub Discussion
- 联系项目维护者

**维护团队**: HFI 开发组
**更新日期**: 2025年8月27日
