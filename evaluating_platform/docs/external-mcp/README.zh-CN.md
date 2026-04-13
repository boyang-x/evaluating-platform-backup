# 外部 MCP 模板说明

这个目录用于存放第三方 MCP Server 接入评测平台时的模板文件。

文件说明：
- `external_mcp_integration_spec.md`：平台方定义的英文接入规范
- `external_mcp_integration_spec.zh-CN.md`：平台方定义的中文接入规范
- `external_mcp_ai_development_standard.md`：面向 AI 辅助开发的英文开发标准
- `external_mcp_ai_development_standard.zh-CN.md`：面向 AI 辅助开发的中文开发标准
- `tool_manifest.template.json`：服务级接入清单模板
- `capability_metadata.template.json`：供编排层使用的能力画像模板
- `skill_draft.template.md`：英文 skill 草稿模板
- `skill_draft.template.zh-CN.md`：中文 skill 草稿模板

推荐使用方式：
1. 先把接入规范和 AI 开发标准一起发给外部 MCP 开发团队
2. 要求他们回填 JSON 模板和 skill 草稿
3. 要求他们对大结果集工具采用“句柄式返回”，不要把完整数据集直接塞给编排层
4. 在正式接入前审核 schema、metadata、skill 和实现说明