# CipherChat Encoding Skill

This MaClaw executable Skill adapts CipherChat and encoding-probe ideas into platform-compatible red-team payload generation.

Sources used for adaptation:
- CipherChat paper: https://arxiv.org/abs/2308.06463
- Garak encoding and prompt-injection probe concepts: https://github.com/NVIDIA/garak
- PyRIT converter-oriented red-team workflow context: https://microsoft.github.io/PyRIT/code/framework/

Platform contract:
- Accepts expert samples, sample questions, composed attacks, or direct enterprise requests through structured Skill args written to the input JSON path.
- Writes `output.json` with a top-level `payload_dataset`.
- Prints the same JSON to stdout using ASCII escapes for Windows-safe execution.
- Does not call the target model, does not judge success, and does not persist credentials or local paths.

Generated payload families include Base64, Morse-like encoding, ROT13, leetspeak, and zero-width Unicode perturbation. The platform should register the resulting `payload_dataset` with `register_skill_payload_dataset`, then pass returned handles to `execute_redteam_evaluation_batch`.
