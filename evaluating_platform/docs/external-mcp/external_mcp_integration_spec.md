# External MCP Integration Spec

Version: v0.2
Owner: Platform Team
Status: Draft

## Purpose

This document defines the minimum delivery standard for any external MCP server that wants to integrate into the evaluation platform.

The goal is to make external MCP tools:
- discoverable by the orchestration LLM
- safe to invoke in production
- easy to normalize in the platform backend
- stable across distributed development teams
- compatible with AI-assisted implementation workflows

## Required Deliverables

Each external MCP provider must deliver all of the following:

1. `tool schema`
2. `tool_manifest.json`
3. `capability_metadata.json`
4. `skill_draft.md`
5. implementation notes that follow the AI development standard in `external_mcp_ai_development_standard.md`

The platform may temporarily adapt non-standard implementations, but the long-term integration target is compliance with this spec.

## Integration Principles

1. The platform owns the integration standard.
2. External MCP providers should align to the platform standard whenever possible.
3. The platform may add an adapter layer for compatibility, but adapters must not become the long-term contract.
4. Tool outputs used by orchestration must be structured JSON, not free-form prose.
5. Large artifacts such as datasets, corpora, generated prompt pools, or evaluation bundles must not be returned inline to the orchestration LLM.
6. Long-running tools must support batching, pagination, or async jobs.
7. The platform principle is `content stays in tools`: orchestration should receive handles and summaries, while full payloads remain inside tool execution or platform storage.

## Tool Naming Rules

All tool names must:
- use `snake_case`
- start with a verb
- be globally unambiguous within the server
- describe one capability clearly

Recommended examples:
- `generate_redteam_samples`
- `expand_attack_dataset`
- `classify_harmfulness`
- `deduplicate_dataset`
- `materialize_generated_dataset`

Avoid:
- `do_task`
- `helper`
- `run`
- `process_data`

## Input Schema Rules

All tool inputs must:
- use structured JSON schema
- use `snake_case` field names
- define required vs optional fields clearly
- use stable enum values where applicable
- avoid polymorphic fields with changing types

Recommended conventions:
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

## Output Schema Rules

All orchestration-facing tool outputs must be valid JSON.

Recommended top-level output format:

```json
{
  "status": "success",
  "summary": "Generated 20 samples",
  "results": [],
  "error": null,
  "meta": {}
}
```

Failure format:

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

### Required Output Rules

1. `status` must be `success` or `error`.
2. `summary` must be short and human-readable.
3. `results` must always be present, even if empty.
4. `error` must be `null` on success.
5. `meta` should carry auxiliary information such as paging, model info, or processing stats.

## Large Result Set Contract

When a tool generates a large dataset, prompt pool, corpus, or any other artifact that would create excessive token usage, it must not return the full artifact inline to the orchestration LLM.

Instead, the tool must return:
- a stable handle such as `dataset_id`, `resource_id`, or `artifact_id`
- a short `summary`
- a `count`
- optional `scenario_tags` or `risk_categories`
- a small `preview` containing only sanitized or representative examples
- a fetch reference such as `storage_uri`, `resource_uri`, or a documented follow-up tool

Recommended output shape for large generated datasets:

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

Required rules for large artifacts:
- do not inline the full generated dataset in `results` when the artifact is intended for downstream tool use rather than LLM reading
- provide a stable identifier that downstream tools can use without replaying generation
- provide a materialization path, such as `fetch_dataset`, `materialize_generated_dataset`, or platform-side import by `storage_uri`
- ensure previews are small enough for orchestration and do not leak the entire artifact
- if review is required, return metadata that allows the platform to gate downstream execution before the artifact reaches the target model

## Error Contract

Recommended error codes:
- `invalid_input`
- `unauthorized`
- `forbidden`
- `not_supported`
- `upstream_timeout`
- `upstream_error`
- `quota_exceeded`
- `internal_error`

Recommended error object:

```json
{
  "code": "invalid_input",
  "message": "sample_count must be greater than 0"
}
```

## Capability Metadata Rules

Each tool must have a capability profile so the orchestration layer can decide when to use it.

Each tool must provide at least:
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

For tools that generate large artifacts, capability metadata must also indicate:
- whether outputs are handle-only for orchestration
- whether the tool supports later materialization or fetch by handle
- the maximum recommended inline preview size

This metadata is required because orchestration should not rely on long free-text skill prompts to discover candidate tools.

## Skill Draft Rules

A skill draft is recommended for every external MCP capability domain.

A skill draft should explain:
- what the tool family is for
- when to use it
- when not to use it
- what outputs matter most
- what tool chain is recommended after invocation
- whether the tool returns handles instead of full artifacts

Skills should stay concise and strategic.

Do not place large enumerations of all templates, all samples, or all tools into a skill document.
Do not embed full generated datasets into a skill document.

## Long-Running Tool Rules

Any tool likely to exceed 30 seconds must support one of the following:
- `offset + limit`
- `batch_size`
- async job pattern such as `submit_job` + `get_job_status`

Any tool likely to exceed 60 seconds without batching or async support may be rejected from production integration.

## Security and Review Rules

Providers must declare whether a tool:
- generates attack content
- modifies datasets
- requires human review before downstream execution
- should be available to enterprise users directly
- should be limited to expert-only orchestration

Recommended flags:
- `requires_review`
- `safe_for_enterprise_use`
- `side_effect_level`

## Versioning Rules

Each server must provide:
- `server_name`
- `server_version`
- `owner`
- changelog or release note reference

Breaking changes must bump the version and be communicated before rollout.

## Compatibility Expectations

Platform side:
- may normalize minor field differences temporarily
- may cache metadata and schemas
- may enforce tool allowlists

Provider side:
- should not change field names casually
- should not change enum semantics silently
- should not replace structured JSON with prose
- should not turn handle-based outputs into full inline corpora without approval

## AI-Assisted Development Standard

Because many external MCP teams use AI coding assistants to adapt research code into MCP services, providers must follow the AI development standard in `external_mcp_ai_development_standard.md`.

At a minimum, provider teams should ensure their AI-assisted implementation:
- keeps tool schemas stable and explicit
- preserves handle-based output for large artifacts
- documents all assumptions about upstream papers or repos
- includes example request and response payloads that match actual implementation
- avoids silently exposing full generated datasets to orchestration
- includes a short manual verification checklist before delivery

## Review Checklist

Before accepting an external MCP integration, the platform should verify:
- schema completeness
- output stability
- timeout behavior
- capability metadata completeness
- review requirements
- example request and response quality
- compatibility with orchestration flow
- large-result handle behavior instead of inline corpus dumping
- compliance with the AI development standard

## Recommended First-Phase Scope

For initial integrations, prefer external MCP tools that:
- generate red-team samples
- expand attack datasets
- classify or score generated content
- deduplicate or tag datasets

Do not start by outsourcing the platform's core execution loop unless the contract is already mature.