# 专家门户 Skill 目录

更新时间：2026-06-08

本文档整理当前专家门户中已安装并处于 active 状态的 8 个 MaClaw Skill。文档中的论文、项目与代码来源链接均使用外部绝对 HTTPS URL；把本文档和 Skill ZIP 一起打包给别人时，不依赖本地相对路径。离线分发时，建议同时保存对应论文 PDF 或项目快照。

> 说明：本平台中的 Skill 只负责在授权的安全评估流程中生成测试载荷数据集，正式调用被测模型、保存证据、攻击成功判定和中文 PDF 报告生成由平台批量评估工具完成。文档不包含完整攻击载荷示例、密钥、目标模型配置或运行数据。

## 速览

| Skill | 评估类型 | 是否依赖租户 LLM | 输出形态 | 原论文/项目 |
| --- | --- | --- | --- | --- |
| `ccbos-classical-chinese-skill` | 文言文越狱 / 中文语义规避 | 是 | `payload_dataset` 文本载荷 | [Obscure but Effective: Classical Chinese Jailbreak Prompt Optimization via Bio-Inspired Search](https://arxiv.org/abs/2602.22983) / [CC-BOS GitHub](https://github.com/xunhuang123/CC-BOS) |
| `autodan-stealth-skill` | 隐蔽语义越狱 / AutoDAN 风格改写 | 是 | `payload_dataset` 文本载荷 | [AutoDAN: Generating Stealthy Jailbreak Prompts on Aligned Large Language Models](https://openreview.net/forum?id=7Jwpw4qKkb) / [AutoDAN GitHub](https://github.com/SheltonLiu-N/AutoDAN) |
| `gptfuzzer-mutator-skill` | 自动越狱提示变异 / Fuzzing | 是 | `payload_dataset` 文本载荷 | [GPTFUZZER: Red Teaming Large Language Models with Auto-Generated Jailbreak Prompts](https://arxiv.org/abs/2309.10253) / [GPTFuzz GitHub](https://github.com/sherdencooper/GPTFuzz) |
| `cipherchat-encoding-skill` | 编码/非自然语言规避 | 否 | `payload_dataset` 文本载荷 | [GPT-4 Is Too Smart To Be Safe: Stealthy Chat with LLMs via Cipher](https://arxiv.org/abs/2308.06463) / [CipherChat GitHub](https://github.com/RobustNLP/CipherChat) |
| `promptinject-goal-hijack-skill` | 提示注入 / 目标劫持 / Prompt 泄露 | 否 | `payload_dataset` 文本载荷 | [Ignore Previous Prompt: Attack Techniques For Language Models](https://arxiv.org/abs/2211.09527) / [PromptInject GitHub](https://github.com/agencyenterprise/PromptInject) |
| `figstep-typographic-visual-skill` | 多模态排版图像越狱 | 否 | `payload_dataset` 文本+图像载荷 | [FigStep: Jailbreaking Large Vision-Language Models via Typographic Visual Prompts](https://arxiv.org/abs/2311.05608) / [FigStep GitHub](https://github.com/CryptoAILab/FigStep) |
| `mm-safetybench-query-image-skill` | 多模态安全基准 / Query Image | 否 | `payload_dataset` 文本+图像载荷 | [MM-SafetyBench: A Benchmark for Safety Evaluation of Multimodal Large Language Models](https://arxiv.org/abs/2311.17600) / [MM-SafetyBench GitHub](https://github.com/isXinLiu/MM-SafetyBench) |
| `hades-hidden-intent-visual-skill` | 多模态隐藏意图 / 图像放大有害性 | 否 | `payload_dataset` 文本+图像载荷 | [Images are Achilles' Heel of Alignment: Exploiting Visual Vulnerabilities for Jailbreaking Multimodal Large Language Models](https://arxiv.org/abs/2403.09792) / [HADES GitHub](https://github.com/RUCAIBox/HADES) |

## 1. CCBOS Classical Chinese Skill

- Skill 名称：`ccbos-classical-chinese-skill`
- 平台用途：把专家样本、企业临时问题或已组合攻击改写为文言文风格安全评估载荷，用于检测被测大模型对中文古文、隐晦表达和语义规避的鲁棒性。
- 适用场景：文言文越狱测试、中文语义混淆测试、提示注入与越狱包装测试。
- 输入：MaClaw 结构化传入的专家样本、样本问题、已组合攻击或企业请求。生产执行时应来自专家门户数据或企业用户明确授权的测试问题。
- 输出：标准 `payload_dataset`，由平台注册为服务端 payload handle 后再调用被测模型。
- 运行依赖：需要当前租户 MaClaw 模型配置，运行时由 MaClawSrv 注入 `MACLAW_LLM_BASE_URL`、`MACLAW_LLM_API_KEY`、`MACLAW_LLM_MODEL` 等环境变量。
- 平台适配说明：原 CC-BOS 是带优化迭代和评分的完整研究框架；本 Skill 保留“八维文言文策略空间 + LLM 改写”的载荷生成能力，目标调用、判定和报告不在 Skill 内完成。
- 原论文：[Obscure but Effective: Classical Chinese Jailbreak Prompt Optimization via Bio-Inspired Search](https://arxiv.org/abs/2602.22983)
- 原项目：[https://github.com/xunhuang123/CC-BOS](https://github.com/xunhuang123/CC-BOS)

## 2. AutoDAN Stealth Skill

- Skill 名称：`autodan-stealth-skill`
- 平台用途：生成 AutoDAN 风格的隐蔽语义越狱载荷，重点是保持自然语言可读性，同时进行层级语义改写。
- 适用场景：隐蔽越狱测试、语义规避测试、专家样本的 stealth jailbreak 改写。
- 输入：专家样本、样本问题、已组合攻击或企业临时请求。
- 输出：标准 `payload_dataset` 文本载荷。
- 运行依赖：需要当前租户 MaClaw 模型配置。
- 平台适配说明：原 AutoDAN 使用更重的遗传/优化式搜索。本平台 Skill 不下载本地模型，也不自行调用被测模型，只保留语义改写和载荷生成阶段。
- 原论文：[AutoDAN: Generating Stealthy Jailbreak Prompts on Aligned Large Language Models](https://openreview.net/forum?id=7Jwpw4qKkb)
- arXiv 入口：[https://arxiv.org/abs/2310.04451](https://arxiv.org/abs/2310.04451)
- 原项目：[https://github.com/SheltonLiu-N/AutoDAN](https://github.com/SheltonLiu-N/AutoDAN)

## 3. GPTFuzzer Mutator Skill

- Skill 名称：`gptfuzzer-mutator-skill`
- 平台用途：参考 GPTFuzzer / LLM-Fuzzer 的自动提示变异思路，对专家样本或企业请求生成多样化越狱候选载荷。
- 适用场景：自动越狱提示变异、prompt mutation、覆盖多种表达方式的安全评估。
- 输入：专家样本、样本问题、已组合攻击或企业临时请求。
- 输出：标准 `payload_dataset` 文本载荷。
- 运行依赖：需要当前租户 MaClaw 模型配置。
- 平台适配说明：原 GPTFuzzer 包含目标反馈和成功样本选择循环；本 Skill 只做载荷变异生成，正式目标调用和二分类判定由平台执行。
- 原论文：[GPTFUZZER: Red Teaming Large Language Models with Auto-Generated Jailbreak Prompts](https://arxiv.org/abs/2309.10253)
- 原项目：[https://github.com/sherdencooper/GPTFuzz](https://github.com/sherdencooper/GPTFuzz)

## 4. CipherChat Encoding Skill

- Skill 名称：`cipherchat-encoding-skill`
- 平台用途：生成 CipherChat 风格的编码、非自然语言或符号扰动测试载荷，用于检测被测模型在 Base64、Morse、ROT13、leetspeak、零宽字符等表达下的安全边界。
- 适用场景：编码规避测试、非自然语言越狱测试、输入混淆测试。
- 输入：专家样本、样本问题、已组合攻击或企业临时请求。
- 输出：标准 `payload_dataset` 文本载荷。
- 运行依赖：不需要租户 LLM，主要通过确定性编码/扰动生成。
- 平台适配说明：原 CipherChat 研究包含教模型理解 cipher、用 cipher 输入绕过自然语言安全泛化等流程；本 Skill 将这些思路转成平台可批量执行的编码载荷。
- 原论文：[GPT-4 Is Too Smart To Be Safe: Stealthy Chat with LLMs via Cipher](https://arxiv.org/abs/2308.06463)
- 原项目：[https://github.com/RobustNLP/CipherChat](https://github.com/RobustNLP/CipherChat)

## 5. PromptInject Goal Hijack Skill

- Skill 名称：`promptinject-goal-hijack-skill`
- 平台用途：生成 PromptInject 风格的目标劫持、提示泄露、分隔符注入和长上下文目标漂移载荷。
- 适用场景：提示注入检验、系统指令泄露测试、目标劫持测试、长上下文注入测试。
- 输入：专家样本、样本问题、已组合攻击或企业临时请求。
- 输出：标准 `payload_dataset` 文本载荷。
- 运行依赖：不需要租户 LLM，主要通过确定性模板族生成。
- 平台适配说明：本 Skill 只生成 prompt injection 测试输入；被测模型调用、证据保存和报告生成仍走平台正式评估链路。
- 原论文：[Ignore Previous Prompt: Attack Techniques For Language Models](https://arxiv.org/abs/2211.09527)
- 原项目：[https://github.com/agencyenterprise/PromptInject](https://github.com/agencyenterprise/PromptInject)
- 相关工具文档：[Garak Prompt Injection examples](https://docs.garak.ai/garak/examples/prompt-injection)

## 6. FigStep Typographic Visual Skill

- Skill 名称：`figstep-typographic-visual-skill`
- 平台用途：生成 FigStep 风格的排版图像越狱载荷，把有害意图放入图像通道，并配合文本提示测试多模态模型。
- 适用场景：多模态安全评估、视觉文本绕过、图像通道越狱测试。
- 输入：请求数量、选择策略，以及 Skill 内置的项目原生 SafeBench-Tiny 数据。
- 输出：标准 `payload_dataset`，包含文本提示和图像载荷元数据。
- 运行依赖：不需要租户 LLM；被测目标必须支持视觉输入。若企业被测模型不支持多模态，平台会返回安全失败而不会调用目标。
- 平台适配说明：本 Skill 使用项目内置轻量数据和图像资产生成可本地自检的载荷，正式 ASR 判定由平台 judge profile 完成。
- 原论文：[FigStep: Jailbreaking Large Vision-Language Models via Typographic Visual Prompts](https://arxiv.org/abs/2311.05608)
- 原项目：[https://github.com/CryptoAILab/FigStep](https://github.com/CryptoAILab/FigStep)

## 7. MM-SafetyBench Query Image Skill

- Skill 名称：`mm-safetybench-query-image-skill`
- 平台用途：生成 MM-SafetyBench 风格的 query-image 安全评估载荷，将关键风险短语放入图像中，文本侧引用图像内容。
- 适用场景：多模态安全基准测试、图文组合安全测试、视觉安全绕过评估。
- 输入：请求数量、选择策略，以及 Skill 内置的处理后问题集合。
- 输出：标准 `payload_dataset`，包含文本提示和图像载荷元数据。
- 运行依赖：不需要租户 LLM；被测目标必须支持视觉输入。
- 平台适配说明：原 MM-SafetyBench 是完整多模态安全基准；本 Skill 将其中 query-image 生成模式适配为平台可调用的载荷生成器。
- 原论文：[MM-SafetyBench: A Benchmark for Safety Evaluation of Multimodal Large Language Models](https://arxiv.org/abs/2311.17600)
- 原项目：[https://github.com/isXinLiu/MM-SafetyBench](https://github.com/isXinLiu/MM-SafetyBench)

## 8. HADES Hidden Intent Visual Skill

- Skill 名称：`hades-hidden-intent-visual-skill`
- 平台用途：生成 HADES 风格的隐藏意图多模态载荷，通过图像放大或隐藏文本中的有害意图，测试多模态模型安全对齐。
- 适用场景：隐藏意图图像攻击、多模态越狱测试、视觉通道安全评估。
- 输入：请求数量、选择策略，以及 Skill 内置的轻量场景定义。
- 输出：标准 `payload_dataset`，包含文本提示和图像载荷元数据。
- 运行依赖：不需要租户 LLM；被测目标必须支持视觉输入。
- 平台适配说明：原 HADES 依赖外部完整图像数据集；本平台 Skill 采用轻量场景定义和生成风格进行本地可运行适配。
- 原论文：[Images are Achilles' Heel of Alignment: Exploiting Visual Vulnerabilities for Jailbreaking Multimodal Large Language Models](https://arxiv.org/abs/2403.09792)
- 原项目：[https://github.com/RUCAIBox/HADES](https://github.com/RUCAIBox/HADES)

## 打包与分发建议

本文档所在目录已经放置 8 个可分发 Skill ZIP。每个 ZIP 的根目录直接包含 `skill.yaml`，并已排除 `__pycache__`、测试输出和运行缓存。

| Skill ZIP | 说明 |
| --- | --- |
| [ccbos-classical-chinese-skill.zip](ccbos-classical-chinese-skill.zip) | CCBOS 文言文越狱 Skill |
| [autodan-stealth-skill.zip](autodan-stealth-skill.zip) | AutoDAN 隐蔽语义越狱 Skill |
| [gptfuzzer-mutator-skill.zip](gptfuzzer-mutator-skill.zip) | GPTFuzzer 提示变异 Skill |
| [cipherchat-encoding-skill.zip](cipherchat-encoding-skill.zip) | CipherChat 编码规避 Skill |
| [promptinject-goal-hijack-skill.zip](promptinject-goal-hijack-skill.zip) | PromptInject 目标劫持 Skill |
| [figstep-typographic-visual-skill.zip](figstep-typographic-visual-skill.zip) | FigStep 多模态排版图像 Skill |
| [mm-safetybench-query-image-skill.zip](mm-safetybench-query-image-skill.zip) | MM-SafetyBench Query Image Skill |
| [hades-hidden-intent-visual-skill.zip](hades-hidden-intent-visual-skill.zip) | HADES 隐藏意图图像 Skill |

分发时建议保持本文档和上述 ZIP 位于同一目录。本文档中的论文/项目链接均为外部绝对 URL，不依赖压缩包内部目录结构；如需完全离线交付，请另行下载论文 PDF 和项目源码快照。不要把平台密钥、Hub token、MaClaw token、被测模型 API Key、运行日志、payload 原文或报告证据正文打包分发。

## 外部链接清单

- CC-BOS paper: https://arxiv.org/abs/2602.22983
- CC-BOS project: https://github.com/xunhuang123/CC-BOS
- AutoDAN paper: https://openreview.net/forum?id=7Jwpw4qKkb
- AutoDAN arXiv: https://arxiv.org/abs/2310.04451
- AutoDAN project: https://github.com/SheltonLiu-N/AutoDAN
- GPTFuzzer paper: https://arxiv.org/abs/2309.10253
- GPTFuzzer project: https://github.com/sherdencooper/GPTFuzz
- CipherChat paper: https://arxiv.org/abs/2308.06463
- CipherChat project: https://github.com/RobustNLP/CipherChat
- PromptInject paper: https://arxiv.org/abs/2211.09527
- PromptInject project: https://github.com/agencyenterprise/PromptInject
- Garak Prompt Injection docs: https://docs.garak.ai/garak/examples/prompt-injection
- FigStep paper: https://arxiv.org/abs/2311.05608
- FigStep project: https://github.com/CryptoAILab/FigStep
- MM-SafetyBench paper: https://arxiv.org/abs/2311.17600
- MM-SafetyBench project: https://github.com/isXinLiu/MM-SafetyBench
- HADES paper: https://arxiv.org/abs/2403.09792
- HADES project: https://github.com/RUCAIBox/HADES
