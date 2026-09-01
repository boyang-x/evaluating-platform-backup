# CCBOS Classical Chinese Skill

This MaClaw Skill adapts the public CC-BOS approach for the platform's large-model safety-evaluation workflow.

CC-BOS is an LLM-driven classical-Chinese jailbreak prompt optimization method. This Skill therefore requires the current MaClaw tenant model configuration at runtime. It does not store or ask the browser for a separate API key.

## Model Config

Configure the model in the administrator portal:

- 管理员门户 -> 模型配置
- Platform default config applies to all tenants.
- Per-account overrides can be set from the administrator account/tenant view.

MaClawSrv injects the selected tenant configuration as runtime environment variables:

- `MACLAW_LLM_BASE_URL`
- `MACLAW_LLM_API_KEY`
- `MACLAW_LLM_MODEL`
- `MACLAW_LLM_WIRE_API`

## Input Data

The Skill prefers questions supplied by MaClaw from platform expert data:

- `samples`
- `expert_samples`
- `sample_questions`
- `questions`
- `composed_attacks`
- `payloads`

Each item may contain `id`, `question`, `prompt`, `content`, `text`, `payload`, `goal`, and `category`.

The bundled `examples/input.json` is for self-check only. Production runs should pass selected expert samples or explicit enterprise questions through MaClaw `manage_skill(action="run", args={...})`.

## Batch Performance

For confirmed enterprise evaluations, MaClaw passes `test_count`, `batch_size`, and `batch_concurrency` in the Skill args. The default runtime batching is 5 payloads per tenant-LLM request with up to 5 concurrent generation batches, so a 20-round CCBOS run normally needs no more than 4 tenant-LLM generation requests before the platform batch tool calls the tested model.

## Output

The Skill prints and writes a JSON object containing:

- `payload_dataset`
- `metadata.generation_backend=tenant_llm_ccbos`
- safe source sample IDs and strategy summaries

It must not print or persist API keys, target credentials, raw runtime config, local paths, or internal logs.
