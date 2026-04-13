# Skill 草稿模板

名称：`redteam-sample-generation`  
归属：`external-mcp-provider`  
状态：Draft

## Purpose

用 1-2 句话说明这个能力域的用途。

示例：  
这个能力域用于生成或扩写红队测试提示词，以支持安全评测前的样本准备。它主要用于在模板拼接和目标模型执行之前补充覆盖面。

## When To Use

- 当现有已审核样本不足时
- 当新的风险类别需要补充覆盖时
- 当编排层需要先获得候选攻击样本再进入执行链路时

## When Not To Use

- 当平台中已经有足够高质量样本时不要使用
- 当输出结果会在未审核的情况下直接发往被测模型时不要使用
- 不要把它当作执行工具或报告工具的替代品

## Tools Covered

- `generate_redteam_samples`
- `expand_attack_dataset`

## Input Guidance

只描述编排层真正需要理解的关键输入字段。

示例：
- `assessment_type`：推荐使用 `compliance_check`、`jailbreak` 等固定值
- `risk_categories`：请传明确的风险标签，不要只给模糊自然语言
- `sample_count`：建议先小批量生成，再根据覆盖情况追加
- `language`：明确指定 `zh`、`en` 或 `multilingual`

## Output Guidance

说明返回 JSON 里哪些字段最重要。

示例：
- `summary`：本次生成结果摘要
- `meta.dataset_id`：生成数据集的稳定句柄
- `meta.count`：生成条目总数
- `meta.preview`：少量脱敏预览
- `meta.storage_uri`：供后续工具拉取或导入的路径
- `error`：结构化错误原因

特别说明：
- 不要把这个能力描述成“直接返回完整生成数据集”
- 如果能力是句柄式返回，必须在 skill 中明确写出来

## Recommended Workflow

1. 生成或扩写样本
2. 对输出进行审核、打标签或去重
3. 必要时按句柄物化或导入结果
4. 在平台内部与模板拼接
5. 如有必要再做增强
6. 再对被测 LLM 执行测试

## Failure Modes

- 上游模型超时
- 风险类别值不合法
- 请求批量过大
- 不支持指定语言或评测类型
- 句柄返回后缺少后续物化路径

## Safety Notes

- 生成的样本应先进入平台审核流程再使用
- 不要绕过平台日志、审核与报告机制
- 未经明确批准，不要直接把原始生成内容暴露给企业用户
- 不要把整份语料或提示词池灌进编排上下文

## Example Request

```json
{
  "assessment_type": "compliance_check",
  "risk_categories": ["violence", "privacy"],
  "language": "zh",
  "sample_count": 20
}
```

## Example Output

```json
{
  "status": "success",
  "summary": "Generated 20 compliance-oriented red-team samples.",
  "results": [],
  "error": null,
  "meta": {
    "dataset_id": "ds_001",
    "count": 20,
    "preview": [
      "sanitized example 1",
      "sanitized example 2"
    ],
    "storage_uri": "mcp://external-redteam-mcp/datasets/ds_001",
    "inline_content_truncated": true
  }
}
```