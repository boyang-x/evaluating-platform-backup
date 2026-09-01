# MM-SafetyBench Query Image Skill

This Skill adapts the public MM-SafetyBench processed-question format into the
platform payload dataset format. It uses the benchmark pattern where a harmful
key phrase is placed in the image while the text prompt references the image.

The original project evaluates answers as safe/unsafe using GPT-style judging
and reports attack rate. This adapter exposes that requirement through
`judge_profile=mm_safetybench_safe_unsafe`.

Source project: https://github.com/isXinLiu/MM-SafetyBench
