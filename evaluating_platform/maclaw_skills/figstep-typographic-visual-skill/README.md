# FigStep Typographic Visual Skill

This MaClaw Skill adapts the public FigStep project into the platform payload
dataset format. It uses project-native SafeBench-Tiny questions and bundled
typographic image prompts from the FigStep repository.

The Skill does not call the tested model. It emits a `payload_dataset` containing
text prompts plus image payloads. The platform registers those outputs as
temporary payload handles, then the confirmed batch evaluation tool calls the
enterprise tested model.

Judge profile: `figstep_typographic_jailbreak`.

Source project: https://github.com/CryptoAILab/FigStep
