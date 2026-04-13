# External MCP AI Development Standard

Version: v0.1
Owner: Platform Team
Status: Draft

## Purpose

This document is intended for external teams who use AI coding assistants to convert research projects, open-source repos, or internal prototypes into MCP servers that integrate with the evaluation platform.

It is written so that it can be given directly to an AI assistant during implementation.

## Primary Goal

Build an MCP server that:
- follows the platform integration contract
- exposes stable tool schemas
- avoids returning large artifacts inline to orchestration
- is reviewable by humans
- is reproducible across multiple development teams

## Required Inputs For AI-Assisted Development

When asking an AI assistant to build or adapt an MCP server, provide all of the following:

1. the source paper or repo reference
2. the target MCP tool list
3. the platform integration spec
4. the capability metadata template
5. the tool manifest template
6. the skill draft template
7. any constraints about deployment, auth, and timeout behavior

Do not ask the AI assistant to "just wrap this repo as MCP" without these constraints.

## Development Requirements

The implemented MCP server must:
- use structured JSON input and output
- keep field names stable and in `snake_case`
- separate orchestration summaries from full artifacts
- support batching, pagination, or async jobs for long tasks
- expose stable handles such as `dataset_id`, `resource_id`, or `artifact_id` for large outputs
- document any unsupported paper features explicitly

## Large Artifact Rule

If the adapted project generates datasets, corpora, prompts, or other large artifacts, the AI-assisted implementation must not return the full artifact inline to orchestration.

Instead, it must:
- store or reference the artifact internally
- return a handle
- return a short summary
- return a small preview only
- provide a follow-up way to fetch or materialize the artifact

This rule is mandatory.

## MCP Wrapping Strategy

When adapting a paper project into MCP tools, the AI assistant should:
- split the project into concrete tool capabilities instead of one oversized `run_all` tool
- keep one tool focused on one capability
- expose any large-generation step as a handle-producing tool
- expose optional follow-up tools for fetch, materialization, scoring, deduplication, or tagging
- avoid coupling generation, execution, and reporting into one opaque tool unless explicitly required

## Tool Design Checklist

For each tool, the AI assistant must define:
- tool name
- purpose
- input schema
- output schema
- timeout or batching behavior
- whether the tool has side effects
- whether the tool requires review before downstream use
- what downstream tools are expected next

## Output Design Checklist

For each tool output, the AI assistant must ensure:
- valid JSON only
- short `summary`
- stable top-level keys
- explicit error object
- no silent field dropping
- no hidden switching between inline output and handle output

If a tool returns handles for large outputs, the assistant must document:
- handle field name
- artifact type
- fetch or materialization path
- preview size policy

## Skill Authoring Checklist

If the MCP capability domain needs a skill draft, the AI assistant should keep it:
- short
- strategy-oriented
- focused on when and how to use the tool family
- explicit about handle-based outputs

The skill draft must not include:
- the full generated dataset
- long enumerations of all resources
- implementation-only details that belong in schemas or code comments

## Traceability Rules

The AI-assisted implementation must include a short implementation note stating:
- which paper, repo, or algorithm was adapted
- which parts were kept intact
- which parts were simplified
- which parts were omitted
- which parts were restructured for MCP compatibility

If a paper algorithm cannot be faithfully exposed through synchronous MCP tools, that limitation must be stated clearly.

## Human Review Checklist

Before delivery, the provider team must manually verify:
- the example requests work
- the example responses match real output
- schemas are consistent with implementation
- large outputs are handle-based, not inlined
- timeouts are documented
- dangerous outputs are marked `requires_review` where appropriate
- any AI-generated code that touches auth, storage, or serialization was reviewed by a human

## Recommended Prompt To Give An AI Coding Assistant

Use a prompt like this:

```text
Build an MCP server from the attached project.

Requirements:
1. Follow the platform's external MCP integration spec.
2. All tool inputs and outputs must be structured JSON.
3. Use snake_case field names.
4. Split capabilities into clear MCP tools instead of one large run_all tool.
5. If the project generates datasets or prompt pools, do not return the full artifact inline. Return a handle such as dataset_id, a short summary, count, preview, and a fetch/materialization path.
6. Add timeout-aware batching or async job support for long-running operations.
7. Fill in tool_manifest.json, capability_metadata.json, and skill_draft.md so they match the real implementation.
8. Include a short implementation note describing what was adapted from the source project and what was changed for MCP compatibility.
9. Do not invent unsupported features. State limitations explicitly.
```

## Delivery Package

A provider using AI-assisted development should deliver:
- implementation code
- `tool_manifest.json`
- `capability_metadata.json`
- `skill_draft.md`
- implementation note
- example request and response payloads
- short manual verification checklist