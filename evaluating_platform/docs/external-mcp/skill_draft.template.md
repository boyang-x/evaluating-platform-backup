# Skill Draft Template

Name: redteam-sample-generation
Owner: external-mcp-provider
Status: Draft

## Purpose

Explain in 1-2 sentences what this capability domain is for.

Example:
This capability domain generates or expands red-team prompts for security evaluation. It is intended to improve coverage before running downstream template combination and target-model execution.

## When To Use

- when existing approved samples are insufficient
- when a new risk category needs broader coverage
- when the orchestration layer needs candidate prompts before execution

## When Not To Use

- do not use when enough high-quality approved samples already exist
- do not use when outputs would be sent directly to the target model without review
- do not use as a substitute for execution or reporting tools

## Tools Covered

- `generate_redteam_samples`
- `expand_attack_dataset`

## Input Guidance

Document only the fields the orchestration layer must understand.

Example:
- `assessment_type`: preferred values such as `compliance_check`, `jailbreak`
- `risk_categories`: use explicit risk labels instead of vague text
- `sample_count`: start small, then expand if coverage is still insufficient
- `language`: specify `zh`, `en`, or `multilingual` clearly

## Output Guidance

Describe what matters in the returned JSON.

Example:
- `summary`: short explanation of what was produced
- `meta.dataset_id`: stable handle for the generated dataset
- `meta.count`: total generated item count
- `meta.preview`: small sanitized preview only
- `meta.storage_uri`: fetch or import path for downstream tools
- `error`: structured failure reason

Important:
- do not describe the tool as if it returns the full generated dataset inline
- make it explicit when the capability returns handles instead of full artifacts

## Recommended Workflow

1. generate or expand samples
2. review, tag, or deduplicate the output
3. materialize or import by handle if needed
4. combine with templates inside the platform
5. optionally apply enhancement
6. execute against the target LLM

## Failure Modes

- upstream model timeout
- invalid risk category values
- excessive requested batch size
- unsupported language or assessment type
- materialization path missing for handle-based outputs

## Safety Notes

- generated samples should enter platform review before execution
- do not bypass platform logging, review, or reporting
- do not present raw generated content to enterprise users unless explicitly approved
- do not dump full corpora or prompt pools into orchestration context

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