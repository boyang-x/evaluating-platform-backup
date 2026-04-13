# External MCP Templates

This folder contains starter templates for integrating third-party MCP servers into the evaluation platform.

Files:
- `external_mcp_integration_spec.md`: platform-side integration contract
- `external_mcp_integration_spec.zh-CN.md`: Chinese version of the integration contract
- `external_mcp_ai_development_standard.md`: AI-assisted development standard for teams adapting research code into MCP services
- `external_mcp_ai_development_standard.zh-CN.md`: Chinese version of the AI-assisted development standard
- `tool_manifest.template.json`: server-level integration manifest template
- `capability_metadata.template.json`: orchestration-facing capability metadata template
- `skill_draft.template.md`: concise skill draft template for capability domains
- `skill_draft.template.zh-CN.md`: Chinese skill draft template

Recommended usage:
1. send the integration spec and AI development standard to the external MCP team
2. ask them to fill in the JSON templates and skill draft
3. require them to implement large-result tools with handle-based outputs instead of inline corpora
4. review the returned schema, metadata, skill draft, and implementation notes before enabling the server in production