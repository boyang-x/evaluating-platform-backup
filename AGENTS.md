# Workspace Instructions

## 1. 先读项目介绍

在这个工作区开始任何较大改动前，先阅读：

- `C:\Users\wangboyang\Desktop\evaluating_platform\PROJECT_GUIDE.md`

目标是先建立对整个平台、CCBOS 集成方式、当前重点链路与关键文件的共同上下文，再开始改代码。

## 2. 文档同步是必做项

只要本次修改影响了下面任一项，就必须同步更新 `PROJECT_GUIDE.md`：

- 核心业务流程
- 关键架构关系
- 服务启动方式
- 重要目录职责
- 关键文件入口
- 外部 MCP 接入方式
- CCBOS/文言文改写链路
- 当前已经验证通过或已知受限的事实

不要把文档更新当成“可选收尾项”，而要当成和代码变更同等重要的交付物。

## 3. 当前重点约束

- 企业侧“文言文越狱测试”应由编排 LLM 基于语义生成计划，而不是依赖关键词硬编码路由。
- `sample_rewrite` 表示只使用专家门户“样本”资源，再送入 CCBOS MCP 改写。
- `sample_rewrite` 不应使用模板，也不应使用已组合攻击。
- 中间改写后的完整载荷不要暴露回编排 LLM。

## 4. 变更后最少验证要求

如果你修改了 Go 后端核心链路，至少考虑运行：

```powershell
go test ./internal/externalmcp ./internal/chat ./internal/agent ./internal/mcptools
```

如果改动范围更大，考虑运行：

```powershell
go test ./...
```

如果变更影响运行时集成，优先补一句当前真实验证结果到 `PROJECT_GUIDE.md`。
