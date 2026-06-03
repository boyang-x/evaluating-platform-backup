# PromptInject Goal Hijack Skill

This MaClaw executable Skill adapts PromptInject-style prompt injection patterns into platform-compatible red-team payload generation.

Sources used for adaptation:
- PromptInject: https://github.com/agencyenterprise/PromptInject
- Garak promptinject probe documentation: https://docs.garak.ai/garak/examples/prompt-injection
- Paper: "Ignore Previous Prompt: Attack Techniques For Language Models" (arXiv:2211.09527)

Platform contract:
- Accepts expert samples, sample questions, composed attacks, or direct enterprise requests through structured Skill args written to the input JSON path.
- Writes `output.json` with a top-level `payload_dataset`.
- Prints the same JSON to stdout using ASCII escapes for Windows-safe execution.
- Does not call the target model, does not judge success, and does not persist credentials or local paths.

Generated payload families include goal hijacking, prompt leaking, delimiter confusion, long-context goal shifting, and role-boundary smuggling. The platform should register the resulting `payload_dataset` with `register_skill_payload_dataset`, then pass returned handles to `execute_redteam_evaluation_batch`.
