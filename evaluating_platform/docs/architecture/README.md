# Architecture Docs

- `agent-architecture.mmd`
  当前智能体整体架构图的 Mermaid 源文件，已包含 Skill、Skill Runner、外部 MCP 与内部 MCP 工具层。
- `agent-architecture.svg`
  上述整体架构图渲染后的图片文件。
- `current-agent-workflow.md`
  当前智能体工作流与框架说明文档。
- `current-agent-workflow-sequence.mmd`
  企业聊天到评测执行时序图的 Mermaid 源文件。
- `current-agent-workflow-sequence.svg`
  上述时序图渲染后的图片文件。
- `current-agent-workflow-state.mmd`
  会话状态与执行状态图的 Mermaid 源文件。
- `current-agent-workflow-state.svg`
  上述状态图渲染后的图片文件。

建议优先保留并维护 `.mmd` 源文件；当工作流或架构发生变化时，先更新源文件，再重新导出对应的 `.svg`。
