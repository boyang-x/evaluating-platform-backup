# CCBOS Classical Chinese Skill

This `generator_skill` rewrites expert-portal samples or direct user-supplied questions into classical-Chinese prompts.

Behavior:
- Accept `source_samples` from the platform.
- Accept direct structured question fields such as `user_questions`, `source_questions`, `question`, or `user_question`.
- Accept a free-form `rewrite_request` or `user_input` string, then let the upstream LLM extract the exact question text that should be rewritten into a parameterized question list.
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
- When both `source_samples` and direct-question fields are present, `source_samples` take priority.
- `rewrite_request` and `user_input` are best used when the caller wants the model to parse a natural-language instruction and pass the extracted question into the rewrite pipeline.
