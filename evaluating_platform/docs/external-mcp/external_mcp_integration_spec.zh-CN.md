# 外部 MCP 接入规范

版本：v0.2  
归属：平台团队  
状态：草案

## 目的

本文档定义了任何希望接入评测平台的外部 MCP Server 的最低交付标准。

目标是让外部 MCP 工具具备以下特性：
- 能被编排 LLM 发现和理解
- 能在生产环境中安全调用
- 能被平台后端稳定归一化处理
- 能适配分布式协作开发场景
- 能适配 AI 辅助开发场景

## 必须交付的内容

每个外部 MCP 提供方必须交付以下 5 项内容：

1. `tool schema`
2. `tool_manifest.json`
3. `capability_metadata.json`
4. `skill_draft.md`
5. 一份符合 `external_mcp_ai_development_standard.zh-CN.md` 的开发说明

平台可以在短期内适配非标准实现，但长期接入目标应当是符合本规范。

## 接入原则

1. 接入标准由平台定义。
2. 外部 MCP 提供方应尽量兼容平台标准。
3. 平台可以增加适配层做兼容，但适配层不应成为长期契约。
4. 供编排层使用的工具输出必须是结构化 JSON，而不是纯自然语言。
5. 数据集、语料池、提示词池、评测包等大结果集，不应把完整正文直接返回给编排 LLM。
6. 长耗时工具必须支持分批、分页或异步任务。
7. 平台遵循“内容不出工具”的原则：编排层看句柄和摘要，正文内容留在工具内部流转或平台存储中。

## 工具命名规范

所有工具名必须：
- 使用 `snake_case`
- 以动词开头
- 在同一服务内全局唯一
- 清晰表达单一能力

推荐示例：
- `generate_redteam_samples`
- `expand_attack_dataset`
- `classify_harmfulness`
- `deduplicate_dataset`
- `materialize_generated_dataset`

不推荐：
- `do_task`
- `helper`
- `run`
- `process_data`

## 输入 Schema 规范

所有工具输入必须：
- 使用结构化 JSON schema
- 使用 `snake_case` 字段名
- 明确必填与可选字段
- 枚举值保持稳定
- 避免同一字段在不同请求中出现类型漂移

推荐字段风格：
- `assessment_type`
- `risk_categories`
- `language`
- `sample_count`
- `offset`
- `limit`
- `batch_size`
- `dataset_id`
- `resource_id`
- `storage_uri`

## 输出 Schema 规范

所有供编排层消费的工具输出必须是合法 JSON。

推荐的顶层输出格式：

```json
{
  "status": "success",
  "summary": "Generated 20 samples",
  "results": [],
  "error": null,
  "meta": {}
}
```

失败时推荐格式：

```json
{
  "status": "error",
  "summary": "Generation failed",
  "results": [],
  "error": {
    "code": "upstream_timeout",
    "message": "provider timeout"
  },
  "meta": {}
}
```

### 输出字段要求

1. `status` 必须是 `success` 或 `error`
2. `summary` 应为简短的人类可读摘要
3. `results` 必须始终存在，即使为空数组
4. 成功时 `error` 必须为 `null`
5. `meta` 用于放置分页、模型信息、处理统计等辅助信息

## 大结果集返回规范

当工具生成了大数据集、提示词池、语料库或其他可能导致 token 爆炸的产物时，不应把完整内容直接内联返回给编排 LLM。

这类工具必须优先返回：
- 稳定句柄，例如 `dataset_id`、`resource_id`、`artifact_id`
- 简短 `summary`
- `count`
- 可选 `scenario_tags` 或 `risk_categories`
- 少量 `preview`，只包含脱敏或代表性示例
- 一个后续可拉取的引用，例如 `storage_uri`、`resource_uri`，或约定好的后续工具

推荐的大结果集返回结构：

```json
{
  "status": "success",
  "summary": "Generated 200 compliance-oriented prompts.",
  "results": [],
  "error": null,
  "meta": {
    "dataset_id": "ds_123",
    "count": 200,
    "risk_categories": ["violence", "privacy"],
    "preview": [
      "sanitized example 1",
      "sanitized example 2"
    ],
    "storage_uri": "mcp://external-server/datasets/ds_123",
    "inline_content_truncated": true
  }
}
```

这类工具必须满足：
- 如果结果主要用于后续工具链消费，而不是给 LLM 阅读，则不要在 `results` 中放完整正文
- 必须返回稳定句柄，供后续工具直接使用，而不是重复生成
- 必须提供“物化”路径，例如 `fetch_dataset`、`materialize_generated_dataset` 或平台侧按 `storage_uri` 导入
- `preview` 必须足够小，不能变相把整份数据集泄露给编排层
- 如果结果需要审核，必须返回足够的 metadata，让平台在进入被测模型前完成 gating

## 错误契约规范

推荐的错误码：
- `invalid_input`
- `unauthorized`
- `forbidden`
- `not_supported`
- `upstream_timeout`
- `upstream_error`
- `quota_exceeded`
- `internal_error`

推荐错误对象格式：

```json
{
  "code": "invalid_input",
  "message": "sample_count must be greater than 0"
}
```

## 能力画像 Metadata 规范

每个工具都必须带有能力画像，以便编排层判断何时应该调用它。

每个工具至少需要提供以下字段：
- `tool_name`
- `planner_summary`
- `assessment_types`
- `scenario_tags`
- `risk_categories`
- `input_resource_types`
- `output_resource_types`
- `cost_level`
- `latency_level`
- `requires_review`
- `recommended_next_steps`

对于会生成大结果集的工具，还必须额外声明：
- 输出是否对编排层采用“句柄优先”模式
- 是否支持按句柄再次拉取或物化
- 建议的最大内联预览数量

这部分 metadata 是必须的，因为编排层不应该依赖长篇 skill 文本去发现候选工具。

## Skill 草稿规范

建议为每个外部 MCP 的能力域提供一份 skill 草稿。

skill 草稿应说明：
- 这类工具整体是做什么的
- 什么时候应该用
- 什么时候不要用
- 输出里哪些字段最重要
- 工具调用后推荐接什么链路
- 该工具是否返回句柄而不是完整产物

Skill 应保持简洁、偏策略。

不要把所有模板、所有样本或所有工具枚举都塞进 skill 文本里。
不要把完整生成数据集直接塞进 skill 文本里。

## 长耗时工具规范

任何可能超过 30 秒的工具，必须支持以下至少一种能力：
- `offset + limit`
- `batch_size`
- 异步任务模式，例如 `submit_job` + `get_job_status`

任何可能超过 60 秒且又不支持分批或异步的工具，平台可以拒绝在生产环境接入。

## 安全与审核规范

提供方必须声明工具是否：
- 会生成攻击性内容
- 会修改数据集
- 需要人工审核后才能进入下游执行
- 可以直接开放给企业侧使用
- 应限制为专家侧专用能力

推荐标志位：
- `requires_review`
- `safe_for_enterprise_use`
- `side_effect_level`

## 版本规范

每个服务必须提供：
- `server_name`
- `server_version`
- `owner`
- changelog 或发布说明引用

任何 breaking change 都必须升级版本并提前通知平台方。

## 兼容性约定

平台侧：
- 可以短期兼容轻微字段差异
- 可以缓存工具 schema 和 capability metadata
- 可以配置工具白名单

提供方：
- 不应随意改字段名
- 不应默默改变枚举语义
- 不应把结构化 JSON 结果改成自然语言结果
- 未经平台确认，不应把句柄式输出改成完整大语料内联输出

## AI 辅助开发标准

由于很多外部 MCP 团队会使用 AI 编码助手，把论文项目或开源项目改造成 MCP 服务，因此所有提供方都应遵循 `external_mcp_ai_development_standard.zh-CN.md`。

至少应确保 AI 辅助开发产物满足：
- tool schema 明确且稳定
- 大结果集默认采用句柄式返回
- 明确记录论文、仓库和实现假设
- request/response 示例与实际实现一致
- 不会悄悄把完整生成数据集暴露给编排层
- 交付前附带简短人工核对清单

## 接入审核清单

平台在接入外部 MCP 前应确认：
- schema 完整性
- 输出结构稳定性
- 超时与长任务策略
- capability metadata 是否完整
- 是否明确需要审核
- request/response 示例是否清晰
- 是否与平台编排链路兼容
- 大结果集是否按句柄式返回，而不是整包正文回传
- 是否符合 AI 辅助开发标准

## 第一阶段推荐接入范围

初期更推荐接入这类外部 MCP 工具：
- 红队样本生成
- 攻击数据集扩写
- 内容分类或风险评分
- 样本去重与打标签

在契约尚未成熟前，不建议一开始就把平台核心执行链完全外包给外部 MCP。