# AutoDAN Stealth Skill

This MaClaw executable Skill adapts AutoDAN's stealthy semantic jailbreak prompt design into platform-compatible red-team payload generation.

Sources used for adaptation:
- AutoDAN repository: https://github.com/SheltonLiu-N/AutoDAN
- Paper: "AutoDAN: Generating Stealthy Jailbreak Prompts on Aligned Large Language Models" (ICLR 2024)
- PyRIT red-team attack workflow context: https://microsoft.github.io/PyRIT/code/framework/

Platform contract:
- Requires MaClawSrv to inject the current tenant LLM config as `MACLAW_LLM_BASE_URL`, `MACLAW_LLM_API_KEY`, `MACLAW_LLM_MODEL`, and optionally `MACLAW_LLM_WIRE_API`.
- Accepts expert samples, sample questions, composed attacks, or direct enterprise requests through structured Skill args written to the input JSON path.
- Writes `output.json` with a top-level `payload_dataset`.
- Prints the same JSON to stdout using ASCII escapes for Windows-safe execution.
- Does not download local models, run gradient/genetic optimization, call the target model, judge success, or persist credentials/local paths.

Batch performance:
- Confirmed MaClaw runs should pass `test_count`, `batch_size`, and `batch_concurrency`.
- The runtime defaults to 5 payloads per tenant-LLM request with up to 5 concurrent generation batches.
- A 20-round AutoDAN run normally needs no more than 4 tenant-LLM generation requests before the platform batch tool calls the tested model.

The original AutoDAN implementation is heavier and model-local. This Skill keeps the semantic stealth rewrite idea while preserving the platform boundary: MaClaw runs the Skill, platform registers payload handles, and `execute_redteam_evaluation_batch` performs target calls and judging.
