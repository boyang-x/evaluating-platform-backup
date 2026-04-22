# Workspace Instructions

## 1. `AGENTS.md` 的作用

`AGENTS.md` 是这个工作区给编码智能体的本地协作说明书。

它的作用不是描述产品功能，而是约束“在这个仓库里应该如何工作”，例如：

- 开始改动前先读哪些文档。
- 哪些业务约束不能被误改。
- 哪些链路是当前重点。
- 改动后至少要做哪些验证。
- 哪些文档必须和代码一起同步更新。

后续任何进入这个工作区的智能体，都应该把这份文件当成仓库级工作约定。

## 2. 开始较大改动前先读什么

在开始任何中大型改动前，先阅读：

- `C:\Users\wangboyang\Desktop\evaluating_platform\PROJECT_GUIDE.md`
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\docs\architecture\current-agent-workflow.md`

如果改动涉及未来受控编排 / QAgent / Skill 形态，再补充阅读：

- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\docs\qagent-controlled-orchestration.md`
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\docs\qagent-skill-workflow.md`

目标是先建立对以下事实的共同上下文，再开始改代码：

- 这是一个“企业聊天入口 + 受控评测编排 + 专家资源供给 + Skill / 外部 MCP 能力接入 + 报告生成”的平台。
- 当前智能体是**受控编排型智能体**，不是完全自由自治 Agent。
- 企业门户的主链路是“生成计划卡片 -> 用户确认 -> 后台执行 -> 回传进度与报告”。

## 3. 文档同步是必做项

只要本次修改影响了下面任一项，就必须同步更新 `PROJECT_GUIDE.md`：

- 核心业务流程
- 关键架构关系
- 服务启动方式
- 重要目录职责
- 关键文件入口
- 外部 MCP 接入方式
- Skill 接入方式或运行边界
- CCBOS / 文言文改写链路
- 当前已经验证通过或已知受限的事实

如果修改直接影响了“当前智能体如何工作”的理解，也要同步更新：

- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\docs\architecture\current-agent-workflow.md`

不要把文档更新当成收尾可选项，而要把它当成和代码变更同等重要的交付物。

## 4. 当前智能体的真实形态

修改代码时，不要把当前系统误当成“自由聊天机器人”。

当前真实形态是：

- 聊天层：编排 LLM 理解需求并生成结构化动作。
- 控制层：平台代码掌握状态机和流程主权。
- 执行层：`agent.Engine` 调用内部 MCP 工具完成资源选择、载荷准备、增强、执行和报告。
- 能力层：Skill、外部 MCP、专家资源都通过受控入口纳入执行体系。

因此，优先遵守下面这些原则：

- 先计划，再确认，再执行。
- LLM 负责语义理解，不负责绕开平台状态机。
- 能力必须通过受控工具面进入，不要让聊天层直接越过平台执行底座。

## 5. 企业门户主链路约束

企业门户当前的正确工作流应该是：

1. 用户描述评测需求。
2. 后端生成真正的计划卡片，而不是只回一段 prose 文本计划。
3. 用户点击确认执行。
4. 系统创建 assessment 并进入后台执行。
5. 前端展示单条当前进度卡和最终报告卡。

做企业聊天相关修改时，优先保证这些约束：

- 当请求已经足够具体时，应优先返回 `plan_confirm` 卡片，而不是普通文本计划。
- `确认执行 / 执行` 应作用于计划卡片，不应卡死在“继续生成计划”的循环里。
- `interactive_web_skill` 只负责打开页面，不进入评测执行链。
- `generator_skill` 才是进入评测执行链的 Skill 形态。

## 6. 资源模式与能力路由约束

当前评测执行的资源模式至少包括：

- `sample_template`
- `composed_attack`
- `sample_rewrite`
- `skill_generated`

修改路由逻辑时，优先遵守下面这些规则：

- `sample_rewrite` 表示“只使用专家门户样本，再送入外部改写能力重写”。
- `sample_rewrite` 不应使用模板。
- `sample_rewrite` 不应使用已组合攻击。
- 中间改写后的完整 payload 不应暴露回编排 LLM。

对“文言文 / 古文 / CCBOS 改写”相关需求，当前应遵守：

- 若存在语义匹配、已发布且可用的 `generator_skill`，优先走 `skill_generated`。
- 只有在没有匹配 Skill 且外部 CCBOS rewrite MCP 仍启用时，才走 `sample_rewrite`。
- 如果外部 MCP 已停用，就不应继续在企业侧计划、文案或执行里把它当成可用能力。

不要把“用户提到某个能力”简单等同于“系统当前能调用该能力”；能力可用性必须结合当前发布状态、启用状态和配置完整性判断。

## 7. 数据边界与隐私约束

当前系统已经在向“控制面 / 数据面分离”演进，改动时不要破坏这条边界。

必须优先遵守：

- 完整样本、改写结果、Skill 生成出的敏感 payload，应尽量停留在 session store / 执行层内部。
- 编排 LLM 更适合接收摘要、句柄、数量、元信息，而不是完整数据集正文。
- 如果某个工具当前采用 `handle_only` 返回策略，不要轻易改成把完整数据回传到聊天层。

## 8. Skill 与运行环境约束

当前 `generator_skill` 的真实执行边界是：

- 通过 `skill_runner` 以隔离容器运行。
- 当前运行时本质上是 Python 执行环境。
- 支持 `requirements.txt` 安装 Python 依赖。
- 通过环境变量注入 Skill 配置和运行输入输出路径。

不要默认认为当前 Skill 体系已经支持：

- 任意自定义基础镜像
- 系统级依赖安装
- 浏览器 / GPU / 多进程后台服务
- 任意语言运行时切换

如果需求明显超出当前 Skill Runner 边界，要明确判断它更适合：

- 扩展 `skill_runner`
- 还是做成外部 MCP 服务

## 9. 企业前端体验约束

修改企业聊天前端时，尽量不要破坏这些已经确认下来的行为：

- 新建空会话应显示欢迎态，而不是空白页。
- 欢迎标签应来自动态能力摘要，且最多展示 6 个。
- 执行中同一个 assessment 应只展示当前那张进度卡，不要重复堆叠旧阶段卡片。
- 用户滚到上方阅读时，不应被轮询消息强制拉回底部。
- 用户侧展示的是“安全分”，语义应保持为“分数越高越安全”。
- 历史脏数据或空值不要直接以 `<nil>`、`info` 等内部值裸露给用户。

## 10. 修改时的实现偏好

在这个仓库里做改动时，优先遵守：

- 优先沿现有状态机和受控编排思路扩展，而不是临时加更多关键词硬编码分流。
- 若能通过结构化字段、状态字段、工具返回契约解决问题，就不要只靠前端文案或 prompt 兜底。
- 前后端展示文案必须和真实执行路径一致，避免“文案说 MCP，实际走 Skill”这类错位。
- 如果是能力可用性问题，要从“发布状态、启用状态、配置完整性、健康状态”四个维度一起判断。

## 11. 变更后的最少验证要求

如果修改了 Go 后端核心链路，至少考虑运行：

```powershell
go test ./internal/externalmcp ./internal/chat ./internal/agent ./internal/mcptools
```

如果改动范围更大，考虑运行：

```powershell
go test ./...
```

如果修改了企业前端聊天页、欢迎页、卡片渲染或报告展示，至少考虑运行：

```powershell
cd C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\frontend
npm run build
```

如果修改影响了运行时集成、容器路由或 Skill / MCP 调用行为：

- 优先记录真实验证结果到 `PROJECT_GUIDE.md`
- 必要时提醒需要重建 `backend` 或 `skill_runner`

## 12. 修改 `AGENTS.md` 的原则

后续如果再修改这份文件，优先保持它具备这几个特点：

- 写“当前真实情况”，不要写理想化目标。
- 写“会影响后续开发决策的约束”，不要堆无关背景介绍。
- 尽量让新进入工作区的智能体读完后，能马上知道哪些地方不能误改、哪些链路是主链路、哪些验证不能省。
