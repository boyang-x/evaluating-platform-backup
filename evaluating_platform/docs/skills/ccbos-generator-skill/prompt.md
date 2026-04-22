# CCBOS Classical Chinese Skill

This `generator_skill` rewrites expert-portal samples into classical-Chinese attack prompts.

Behavior:
- Accept `source_samples` from the platform.
- Preserve the original malicious intent while obscuring surface wording.
- Return a standardized `payload_dataset`.
- Prefer an upstream OpenAI-compatible model configured by `SKILL_LLM_*` or `LLM_*`.
- Fall back to a deterministic local rewrite template when no upstream model is configured.

Redaction rules:
- Do not return raw execution logs or target responses inline.
- Keep dataset summaries and payload summaries short.

Supported input notes:
- `source_sample_id` is optional metadata.
- `requested_count` limits how many sample rows are consumed.
- `assessment_types` and `goal` are treated as rewrite hints only.
