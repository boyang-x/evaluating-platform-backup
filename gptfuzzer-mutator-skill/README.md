# GPTFuzzer Mutator Skill

This MaClaw executable Skill adapts GPTFuzzer / LLM-Fuzzer prompt mutation ideas into platform-compatible red-team payload generation.

Sources used for adaptation:
- GPTFuzzer repository: https://github.com/sherdencooper/GPTFuzz
- Paper: "GPTFUZZER: Red Teaming Large Language Models with Auto-Generated Jailbreak Prompts" (arXiv:2309.10253)
- EasyJailbreak framework context: https://github.com/EasyJailbreak/EasyJailbreak

Platform contract:
- Requires MaClawSrv to inject the current tenant LLM config as `MACLAW_LLM_BASE_URL`, `MACLAW_LLM_API_KEY`, `MACLAW_LLM_MODEL`, and optionally `MACLAW_LLM_WIRE_API`.
- Accepts expert samples, sample questions, composed attacks, or direct enterprise requests through structured Skill args written to the input JSON path.
- Writes `output.json` with a top-level `payload_dataset`.
- Prints the same JSON to stdout using ASCII escapes for Windows-safe execution.
- Does not call the target model, does not judge success, and does not persist credentials or local paths.

Batch performance:
- Confirmed MaClaw runs should pass `test_count`, `batch_size`, and `batch_concurrency`.
- The runtime defaults to 5 payloads per tenant-LLM request with up to 5 concurrent generation batches.
- A 20-round GPTFuzzer run normally needs no more than 4 tenant-LLM generation requests before the platform batch tool calls the tested model.

The original GPTFuzzer loop includes target feedback and success selection. This Skill intentionally keeps only the payload mutation phase so confirmed target calls and judging remain inside `execute_redteam_evaluation_batch`.
