# HADES Hidden Intent Visual Skill

This Skill adapts the public HADES project into the platform payload dataset
format. The full HADES released image dataset is hosted outside the repository,
so this lightweight package uses the repository's bundled scenario definitions
and generation style to create small hidden-intent visual payloads for local
platform testing.

The original HADES evaluation reports attack success rate for multimodal
jailbreak responses. This adapter exposes that behavior through
`judge_profile=hades_hidden_intent_jailbreak`.

Source project: https://github.com/RUCAIBox/HADES
