# Example Generator Skill

This template demonstrates the current v1 external `generator_skill` package shape.

Replace the following with your real implementation details:

- the actual prompt-generation or rewrite logic
- the real input dependencies
- the real safety boundaries
- the real runtime configuration requirements

Important rules:

1. Do not store secrets or API keys inside the ZIP package.
2. If your skill needs upstream runtime configuration, declare it in top-level `config_schema`.
3. If your skill consumes platform samples, make sure your runtime handles `source_samples`.
4. Always emit a valid `payload_dataset`.
