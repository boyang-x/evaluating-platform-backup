# Project Guide

## 1. 项目是什么

这是一个面向企业与专家双侧协作的 AI 安全评测平台，目标是把“评测编排”“专家资源”“外部攻击能力”与“报告生成”串成一条可执行链路。

当前工作区包含两个强关联子项目：

- `evaluating_platform/`
  平台主工程。包含 Go 后端、React 前端、MCP 工具层、聊天编排、评测执行、报告生成、外部 MCP 接入能力。
- `CC-BOS/`
  文言文越狱优化相关项目。当前平台通过其中的 `mcp-server/` 以外部 MCP 的方式接入其改写/优化能力。

一句话理解当前核心能力：

- 企业用户在平台聊天里提出评测诉求。
- 编排 LLM 先生成计划卡片。
- 平台从专家门户资源里选样本/模板/已组合攻击，或选择已发布 `generator_skill`。
- 必要时调用外部 MCP 能力进行样本改写。
- 然后对被测 LLM 执行测试并生成报告；如果走 skill，则由独立 `skill_runner` 生成标准化 `payload_dataset` 再交给执行层。

## 2. 当前最重要的业务流

### 2.1 常规评测流

1. 企业用户在聊天中描述评测需求。
2. `internal/chat/session.go` 使用编排 LLM 生成计划。
3. 用户确认后，`internal/agent/engine.go` 驱动评测流水线。
4. 流水线通过内部 MCP 工具完成资源推荐、预览、组合/改写、执行与报告生成。

### 2.2 当前重点流：文言文越狱测试

这是近期重点打通的链路，设计原则如下：

1. 用户输入类似“请对被测 LLM 进行文言文越狱测试”。
2. 编排 LLM 直接生成计划卡片，而不是先走关键词硬编码分流。
3. 计划中的 `resource_mode_preference` 应为 `sample_rewrite`。
4. `sample_rewrite` 的含义是：
   只使用专家门户中的“样本”资源，不用模板，不用已组合攻击。
5. 样本问题会送到 CCBOS MCP 做迭代改写，目标是生成文言文形式攻击问题。
6. 改写后的完整内容不应回传给编排 LLM，只在执行层内部流转。
7. 最终调用测试工具执行，并生成报告。

目前这条链路已经打通，关键实现点在：

- `evaluating_platform/internal/chat/session.go`
- `evaluating_platform/internal/agent/engine.go`
- `evaluating_platform/internal/mcptools/external_rewrite_tools.go`
- `evaluating_platform/internal/externalmcp/`
- `CC-BOS/mcp-server/`

### 2.3 当前新增链路：Skill 导入、发布与 `skill_generated`

这是 2026-04-15 新打通的专家能力链路，设计原则如下：

1. 专家用户可在专家门户的 `/expert/skills` 页面上传 `generator_skill` 或 `interactive_web_skill` ZIP。
2. 后端会解析 `skill.yaml`、写入 `skills / skill_versions / skill_runs`，并把原始 ZIP 存入 MinIO。
3. 专家可对任意版本执行自测，只有 `self_test_passed` 或已发布版本允许发布。
4. 编排层会在 `internal/mcptools/planning_tools.go` 中把已发布 skill 纳入 `recommended_skills`。
5. 当计划进入 `skill_generated` 模式时，执行层会先调用 `preview_skill`，再调用 `run_generator_skill`。
6. `run_generator_skill` 不直接把完整敏感 payload 回传给编排层，而是把生成出的 `payload_dataset` 写入 session，后续仍由平台执行层控制。
7. 专家端 Skill 读取接口默认返回脱敏 DTO：不回传 `prompt_text`、示例正文、`result_payload`、完整日志，只保留版本摘要、校验报告、数据集摘要与自测日志摘录。

## 3. 目录速览

### 3.1 平台主工程 `evaluating_platform/`

- `cmd/server/main.go`
  后端启动入口。负责初始化数据库、Redis、MinIO、内部 MCP Server、内部 MCP Client、外部 MCP 管理器、HTTP API。
- `cmd/skillrunner/main.go`
  独立 Skill 运行时入口。接收后端传入的 Skill ZIP、运行参数与超时配置，在隔离容器中执行 skill，并回传运行结果与校验报告。
- `configs/config.yaml`
  运行配置入口。当前包含 `skill.runner_base_url` 与 `skill.runner_timeout_seconds`，用于把 backend 指向 `skill_runner`。
- `internal/chat/`
  企业聊天会话、计划生成、确认执行、进度卡片更新，以及对已发布 `interactive_web_skill` 的 `launch_skill` 直达打开。
- `internal/agent/`
  评测编排引擎与执行器。负责把计划落成一系列 MCP 工具调用。
- `internal/mcptools/`
  平台内部 MCP 工具集合。包括资源推荐、样本/模板预览、载荷增强、执行、报告、Skill 预览/运行工具，以及对外部 MCP 的桥接工具。
- `internal/externalmcp/`
  外部 MCP 服务管理层。负责连接、同步、代理第三方 MCP 工具到平台内部。
- `internal/repository/`
  数据库访问层。当前已包含 `skill.go`，负责 `skills / skill_versions / skill_runs` 的读写。
- `internal/skill/`
  Skill 域服务层。负责 Skill ZIP 解析、导入、发布、自测、运行时请求与已发布 Skill 搜索。
- `internal/report/`
  报告生成逻辑。
- `internal/sample/`
  样本加载、预览、CSV 解析。
- `internal/api/`
  HTTP handler 与中间件。当前已包含专家端 Skill 管理接口与脱敏 DTO 映射。
- `frontend/`
  React + Vite 前端。当前专家门户已新增 `/expert/skills` 页面用于管理 Skill；企业门户已支持在聊天中打开 `/enterprise/skills/:id/open` 新标签页。
- `docs/external-mcp/`
  外部 MCP 接入规范与模板。
- `docs/qagent-controlled-orchestration.md`
  QAgent 受控编排接入方案草案，说明如何在不丢失流程控制权的前提下，引入 QAgent 与 skill。
- `docs/qagent-skill-workflow.md`
  描述未来接入 skill 且由 QAgent 进行受控编排时，推荐采用的端到端流程。
- `docs/skills/generator-skill-spec.zh-CN.md`
  面向外部开发者的第三方 `generator_skill` 接入规范，约束 skill 包结构、能力声明、输出数据集格式与自测提交流程。
- `docs/skills/generator-skill-template/`
  与规范配套的第三方 skill 提交样板包，包含 `skill.yaml`、输入输出示例与自测报告模板。
- `docker-compose.yml`
  本地联调入口，负责拉起 PostgreSQL、Redis、MinIO、backend、frontend、`skill_runner`、CCBOS MCP。
- `Dockerfile.skillrunner`
  `skill_runner` 镜像定义，负责构建独立 Skill 运行时容器。

### 3.2 CCBOS 相关 `CC-BOS/`

- `CC-BOS/mcp-server/`
  当前平台实际接入的 CCBOS 外部 MCP 服务。
- `CC-BOS/README.md`
  上游算法/研究项目说明。

## 4. 运行结构

主要服务与默认端口：

- 平台后端：`http://localhost:8080`
- 平台内部 MCP SSE：由后端在 `configs` 中配置的 MCP 端口启动，当前容器内走 `18080`
- 前端：`http://localhost:5173`
- Skill Runner：`http://localhost:8091`
- CCBOS MCP：`http://localhost:18191`
- PostgreSQL：`5432`
- Redis：`6379`
- MinIO S3：`9000`
- MinIO Console：`9001`

常用启动方式：

在 `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform` 下执行：

```powershell
docker compose up -d --build backend skill_runner ccbos_mcp
```

如果需要完整前后端联调：

```powershell
docker compose up -d --build
```

健康检查：

```powershell
Invoke-WebRequest -UseBasicParsing http://localhost:8080/api/v1/health
```

## 4.1 代码仓库状态

当前顶层工作目录 `C:\Users\wangboyang\Desktop\evaluating_platform` 已经正式初始化为 Git 仓库，并连接到私有备份仓库：

- `origin = https://github.com/boyang-x/evaluating-platform-backup.git`
- 默认分支：`main`

建议后续开发方式：

1. 日常开发继续在当前原目录中进行，而不是在 `evaluating_platform__publish_snapshot` 临时副本中开发。
2. `main` 作为稳定基线。
3. 每次较大的实验性改动先新建分支，例如 `feature/qagent-skill`。
4. 当改动不理想时，优先通过 Git 分支或提交回退，而不是手工覆盖目录。

## 5. 当前关键设计约束

### 5.1 关于编排 LLM

- 目标是不依赖硬编码关键词命中来决定是否使用 CCBOS。
- 应优先让编排 LLM 根据用户语义与工具能力做计划。
- 但计划一旦进入执行层，具体中间载荷不要再回传给编排 LLM。

### 5.2 关于 `sample_rewrite`

- 输入资源类型必须是专家门户的“样本”。
- 不是模板。
- 不是已组合攻击。
- 样本内容经 CCBOS 改写后，只把句柄或脱敏摘要回传给编排层。
- 如果当前存在与“文言文/古文/CCBOS 改写”语义匹配的已发布 `generator_skill`，企业侧应优先规划到 `skill_generated`；只有在没有匹配 Skill 且外部 CCBOS rewrite MCP 仍处于启用状态时，才应规划到 `sample_rewrite`。

### 5.3 关于外部 MCP

- 平台已支持把第三方 MCP Server 同步成内部代理工具。
- 编排层默认不应暴露原始代理工具名，必要时走平台封装后的业务工具。
- 大结果集能力优先使用“句柄式返回”，避免把完整敏感内容直接暴露给编排模型。

### 5.4 关于 Skill

- 当前 v1 同时支持第三方 `generator_skill` 与 `interactive_web_skill`：
  - `generator_skill` 继续通过独立 `skill_runner` 生成 `payload_dataset`
  - `interactive_web_skill` 仅用于 `launch_skill -> 打开页面`，不会进入评测执行链
- 只有已发布 `generator_skill` 会进入 `recommended_skills`，并参与 `skill_generated` 资源模式。
- `generator_skill` 执行必须通过独立 `skill_runner` 完成，不能让编排层直接执行 Skill 包中的代码。
- `generator_skill` 现在可以在 `skill.yaml` 顶层声明 `config_schema`，由专家端 `/expert/skills` 动态渲染配置表单。
- Skill 配置值按“skill 级”共享存储，不按版本复制；读取、发布校验与运行时 env 注入始终以“当前选中版本的 schema”为准。
- Skill 配置值会通过 `skill_config_values` 表加密落库；若后端未配置 `crypto.master_key`，平台仍可启动，但声明了 `config_schema` 的 Skill 无法保存配置，也无法通过发布前配置校验。
- 运行时 env 注入优先级为：专家已保存值 > manifest 中的非 secret `default` > 兼容模式下旧 `metadata.runtime_config.fallback_env` 对应的进程环境变量。
- 已发布 `generator_skill` 如果后续因为配置缺失而变得不完整，不会自动改写 Skill 生命周期状态，但会从企业侧推荐与运行入口中隐藏。
- 专家端 Skill 读取接口默认走脱敏返回，不暴露 `prompt_text`、示例正文、`result_payload`、完整 stdout/stderr。
- 即使是 Skill 生成的 payload，也不应把完整敏感内容回传给编排 LLM；执行层只保留脱敏摘要与统计信息。

## 6. 当前你最可能会改的文件

- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\cmd\server\main.go`
  启动装配、内部 MCP Client、服务初始化。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\internal\chat\session.go`
  聊天入口、计划生成、确认执行、进度消息。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\internal\agent\engine.go`
  评测流水线决策与执行。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\internal\agent\executor.go`
  MCP 工具调用与超时控制。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\internal\mcptools\planning_tools.go`
  资源推荐逻辑。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\internal\mcptools\skill_tools.go`
  `preview_skill` 与 `run_generator_skill` 工具定义。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\internal\mcptools\external_rewrite_tools.go`
  样本送入 CCBOS 改写并导回会话。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\internal\api\handler\skill.go`
  专家端 Skill 管理 API 入口。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\internal\skill\service.go`
  Skill 导入、发布、自测与运行时调用主逻辑。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\cmd\skillrunner\main.go`
  独立 Skill Runner 服务入口。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\internal\externalmcp\manager.go`
  外部 MCP 连接/同步/调用。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\internal\externalmcp\sse_client.go`
  自定义 SSE 客户端，解决长调用超时问题。
- `C:\Users\wangboyang\Desktop\evaluating_platform\CC-BOS\mcp-server\rewrite.go`
- `C:\Users\wangboyang\Desktop\evaluating_platform\CC-BOS\mcp-server\optimization.go`
- `C:\Users\wangboyang\Desktop\evaluating_platform\CC-BOS\mcp-server\response_parsing.go`
  CCBOS 返回结果解析与容错。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\frontend\src\pages\expert\SkillManager.tsx`
  专家端 Skill 管理页面。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\frontend\src\services\expert.ts`
  专家端 Skill API 调用封装。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\frontend\src\pages\expert\ExpertPortal.tsx`
  专家端菜单与 Skill 路由入口。

## 7. 最近已确认的事实

- 企业侧“文言文越狱测试”已经能直接生成计划卡片。
- `resource_mode_preference` 已支持 `sample_rewrite`。
- 平台内部 MCP Client 已切到自定义超时感知 SSE 客户端，避免默认 60 秒响应超时截断长流程。
- CCBOS MCP 的返回解析已做容错增强，可兼容多种 JSON 形状。
- 当前如果专家门户只有合规检测样本，那么文言文链路会基于这些现有样本执行；这属于现有数据约束，不是链路阻塞。
- 平台已具备完整的 Skill 基础设施：`013_skills.sql`、Skill 模型/仓储、`internal/skill/` 服务、`cmd/skillrunner`、`Dockerfile.skillrunner`。
- 专家端已提供 `/api/v1/skills` 相关管理 API，并在前端新增 `/expert/skills` 页面。
- 专家端 Skill 读取接口现在默认返回脱敏 DTO，不暴露 `prompt_text`、示例正文、`result_payload`、完整运行日志；仅 `self_test` 运行提供最长 2000 字符的 `log_excerpt`。
- 编排层已支持 `recommended_skills` 与 `skill_generated`，执行时会通过 `preview_skill` / `run_generator_skill` 连接已发布 Skill 与 `skill_runner`。
- 专家端新增了 Skill 动态配置接口：`GET /api/v1/skills/:id/config` 与 `PUT /api/v1/skills/:id/config`，用于按版本查看 schema、保存配置、清空已保存值。
- `generator_skill` manifest 现在正式支持顶层 `config_schema`；旧 `metadata.runtime_config` 会在导入与读取时自动归一化为同一套内部 schema。
- `RuntimeRequest` 已支持 `env` 注入；`skill_runner` 会在保留全局 `LLM_* / SKILL_LLM_*` 的基础上，再覆盖注入 per-skill env。
- 已发布但配置不完整的 `generator_skill` 不会出现在企业侧 `recommended_skills`、`preview_skill`、`run_generator_skill` 链路中。
- 2026-04-15 已验证 `go test ./internal/externalmcp ./internal/chat ./internal/agent ./internal/mcptools ./internal/api/handler` 通过。
- 2026-04-15 已验证 `go test ./...` 通过。
- 2026-04-15 已验证 `frontend` 目录下 `npm run build` 通过。

## 8. 常用测试命令

在 `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform` 下：

```powershell
go test ./internal/externalmcp ./internal/chat ./internal/agent ./internal/mcptools
```

如果修改了专家端 Skill 接口或页面，建议同时验证：

```powershell
go test ./internal/api/handler
```

全量测试：

```powershell
go test ./...
```

查看后端日志：

```powershell
docker logs ep_backend --tail 100
```

查看 CCBOS MCP 日志：

```powershell
docker logs ep_ccbos_mcp --tail 100
```

查看 Skill Runner 日志：

```powershell
docker logs ep_skill_runner --tail 100
```

前端构建验证：

```powershell
cd frontend
npm run build
```

## 9. 新线程建议阅读顺序

如果以后开新线程，建议先读：

1. 本文件 `PROJECT_GUIDE.md`
2. `evaluating_platform/docs/architecture/agent-architecture.svg`
3. `evaluating_platform/docs/qagent-controlled-orchestration.md`
4. `evaluating_platform/docs/qagent-skill-workflow.md`
5. `evaluating_platform/docs/skills/generator-skill-spec.zh-CN.md`
6. `evaluating_platform/docs/skills/generator-skill-template/skill.yaml`
7. `evaluating_platform/cmd/server/main.go`
8. `evaluating_platform/internal/chat/session.go`
9. `evaluating_platform/internal/agent/engine.go`
10. `evaluating_platform/internal/mcptools/external_rewrite_tools.go`
11. `evaluating_platform/internal/externalmcp/`
12. `CC-BOS/mcp-server/`

## 10. 文档维护规则

这份文档的目标不是写成长篇设计文，而是让新线程能在几分钟内知道：

- 项目要做什么
- 哪条链路最重要
- 关键文件在哪里
- 当前已经打通到什么程度
- 改完代码后还应同步更新哪些事实

如果后续改动影响以下内容，必须同步更新本文件：

- 核心业务流
- 关键目录职责
- 重要文件入口
- 运行方式
- 外部 MCP/CCBOS 集成方式
- 已确认的项目现状

## 11. 近期变更汇总（截至 2026-04-18）

- 外部 MCP:
  - 专家端支持显式断开外部 MCP Server: `POST /api/v1/external-mcp-servers/:id/disconnect`
  - 断开后会清理代理工具、清空 `external_mcp_tools`，并让 `tool_count` 回到 `0`
- Skill 主链路:
  - 平台已接通 `generator_skill` / `interactive_web_skill` 的导入、版本、自测、发布、停用、启用、删除
  - `skill_generated` 支持把专家样本作为 `source_samples` 送入已发布的 generator skill，`run_generator_skill` 支持可选 `sample_id`
  - `interactive_web_skill` 只走打开页面链路，不进入评估执行；企业侧入口为 `/enterprise/skills/:id/open`
  - `interactive_web_skill` v1 要求入口 HTML 自包含或使用绝对 `http(s)` / `data:` 资源，禁止相对静态资源引用
  - 示例 interactive skill 包：`evaluating_platform/docs/skills/ai-risk-viz.zip`
- Skill 配置与运行:
  - `generator_skill` 支持顶层 `config_schema`
  - Skill 配置按 skill 级共享存储，使用 `skill_config_values` 加密落库
  - 缺少 required 配置时禁止发布；已发布但配置不完整的 generator skill 会从企业侧推荐和运行入口隐藏
  - CCBOS 当前打包 skill 为 `evaluating_platform/docs/skills/ccbos-generator-skill.zip`，版本 `1.2.0`
  - CCBOS skill 配置应在导入后的专家端 Skill 配置界面填写；必填 `SKILL_LLM_API_KEY`，可选 `SKILL_LLM_BASE_URL`、`SKILL_LLM_MODEL`
  - CCBOS skill 运行时只读取 `SKILL_LLM_*`，不再 fallback 到平台全局 `LLM_*`
  - CCBOS skill 自测为离线自测，不依赖上游 LLM 连通性
  - 旧导入 skill 如果早于 `config_schema` 落库，可能出现配置面板空白；修复方式是用新后端重导，或回填 `skill_versions.metadata.config_schema`
  - 超过 6 分钟的 stale self-test 会在 skill 列表/详情/运行记录读取时自动回收为 `timeout`
- 资产与工作流清理:
  - 常规专家资产现在直接创建为 `published`，不再走旧 review flow
  - 旧 workflow 编辑链路已删除，历史 `type = 'workflow'` 资产会从列表中过滤
  - 管理员侧普通资产审核页面和相关接口已删除
- 企业评估记录:
  - 支持单条删除：`DELETE /api/v1/assessments/:id`
  - 支持当前用户“一键全部删除”：`DELETE /api/v1/assessments`
  - `pending` / `running` 记录不会被批量删除，必须先取消
- LLM Provider:
  - 讯飞 Coding Plan OpenAI 兼容地址 `https://maas-coding-api.cn-huabei-1.xf-yun.com/v2` 只能搭配 `astron-code-latest`
  - 该约束适用于编排 LLM、被测 LLM、辅助 LLM 的 OpenAI-compatible 配置
  - 对于当前讯飞兼容编排模型，聊天规划请求中不要发送多条 `system` message；企业聊天若要附加 interactive skill 候选提示，需合并进同一条 system prompt，否则 `/api/v1/chat/sessions/:id/messages` 可能收到上游 `EngineInternalError: Bad Request`
  - OpenAI connector 现在会携带上游原始错误 body，便于定位 4xx/5xx
- 运维 / 排障:
  - 如果前端新增 API 报 `404`，但源码里已经有对应后端路由，优先怀疑旧 backend 镜像
  - 当前 Docker 会把 Go 源码打进 backend 镜像；改 handler / route 后需要 `docker compose up -d --build backend`
  - 已出现过的同类问题：`GET /api/v1/skills/:id/config`、`DELETE /api/v1/assessments`
  - 重建后可用 `docker logs ep_backend --tail 120` 检查 Gin 路由列表是否真的加载了新接口

 - 企业门户欢迎标签：
   - 新增企业侧聚合接口：`GET /api/v1/enterprise/welcome-capabilities`
   - 返回内容基于“已发布可用的 Skill、已启用且健康的外部 MCP、以及当前样本/攻击链/模板/多语种与编码变形增强能力概况”动态生成
   - 后端统一输出企业友好的标签文案和点击提示语，前端最多展示 6 个标签，并在页面重新获得焦点时自动刷新

### 11.1 最近验证

- `2026-04-20`: 已定位企业门户聊天“发送后没反应”为后端 `/api/v1/chat/sessions/:id/messages` 返回 `500`，根因是讯飞兼容编排模型不接受多条 `system` message；将 interactive skill 候选提示合并回单条 system prompt 后，本地 API 级验证恢复正常
- `2026-04-20`: 企业门户欢迎页标签已改为动态能力摘要，新的 `GET /api/v1/enterprise/welcome-capabilities` 会聚合已发布可用的 Skill、健康的外部 MCP，以及当前样本/攻击链/模板/多语种与编码变形增强能力；前端欢迎页会在初次进入和窗口重新聚焦时刷新，且最多展示 6 个标签
- `2026-04-20`: 企业侧“文言文/古文/CCBOS 改写”规划已改为先检查能力可用性：若存在语义匹配的已发布 `generator_skill`，优先走 `skill_generated`；只有在没有匹配 Skill 且外部 CCBOS rewrite MCP 仍启用时，才走 `sample_rewrite`；若旧计划误落到 `sample_rewrite` 但运行时发现 `ccbos` 已停用，执行层会自动回退到匹配 Skill
- `2026-04-20`: 已定位企业侧 `skill_generated` 执行偶发 `timeout waiting for SSE response after 14.999...s` 的根因：Agent 全局 `tool_timeout_seconds=15` 会截断长耗时 `run_generator_skill`；执行器现已为 `run_generator_skill` 提供 5 分钟专用超时窗口，不再被 15 秒默认值提前中断
- `2026-04-20`: 企业聊天计划卡片会按最终归一化后的 `resource_mode_preference` 校正文案；当实际模式为 `skill_generated` 时，即使编排 LLM 的原始 `message / goal` 仍提到 `CCBOS MCP`，对用户展示的计划说明也会回落为 Skill 方案描述，避免“展示写 MCP、执行却走 Skill”的错位
- `2026-04-20`: 企业聊天新增“纯文本计划兜底收敛”逻辑：如果编排 LLM 没按 `confirm_plan` JSON 返回、而是输出了“评估计划概要 / 资源模式 / 是否确认执行”这类 prose 计划，后端会自动转成真正的 `plan_confirm` 卡片；如果用户在这种历史会话里继续输入“执行/确认执行”，系统也会优先从最近一条计划性文本中恢复卡片，而不是再次回复“先生成计划”
- `2026-04-20`: 企业聊天页不再在每次轮询拿到新消息后强制滚动到底部；只有当用户本来就在底部附近时，消息列表才会自动跟随，从而避免执行评估时翻到上方阅读又被拽回底部
- `2026-04-20`: 用户侧评分展示已改为“安全分（越高越安全）”；内部仍保留 `risk_score` 作为风险分与风险等级计算依据，但企业聊天报告卡片与 PDF 导出会展示由 `risk_score` 反推的 `security score`
- `2026-04-20`: 企业聊天页的计划卡片会在重新拉取会话时，用 `session.plan_info` 回填最新确认过的 `test_count`，避免“卡片显示 20 条、实际执行 10 条”的前端显示错位
- `2026-04-20`: 企业聊天执行中的进度展示已改为“每个 assessment 仅展示当前那一张进度卡”；当报告卡片出现后，会自动隐藏旧进度卡，从而避免重复的“正在启动任务”与历史阶段卡片一直保持加载态
- `2026-04-20`: 企业聊天计划卡片里的 `评估目标` 若上游返回空值，后端不再把 Go 的 `nil` 格式化成字面量 `"<nil>"`；新计划会自动回落到归一化后的目标文案，前端对历史脏数据也会做展示兜底
- `2026-04-20`: 企业聊天报告卡片现在会把内部 `risk_level` 枚举完整映射成中文标签；`info` 不再直接显示英文，而会展示为“未发现明显风险”，`critical` 也会展示为“严重风险”
- `2026-04-20`: 企业聊天前端已做一轮结构性清理：`ChatPage.tsx` 现在主要负责页面状态与事件分发，卡片组件拆到 `frontend/src/pages/enterprise/ChatCards.tsx`，纯展示清洗逻辑拆到 `frontend/src/pages/enterprise/chatDisplay.ts`
- `2026-04-20`: 已验证 `go test ./internal/chat ./internal/api/handler` passed
- `2026-04-20`: 已验证 `frontend` 目录 `npm run build` passed
- `2026-04-18`: `go test ./...` passed
- `2026-04-18`: `frontend` 目录 `npm run build` passed
- `2026-04-15`: `docs/skills/ccbos-generator-skill/runtime/selfcheck.py` passed
- `2026-04-22`: 新增当前智能体工作流与架构梳理文档：`evaluating_platform/docs/architecture/current-agent-workflow.md`，覆盖企业聊天规划、计划确认、评测编排、Skill / MCP 分支、数据边界与当前框架特征，便于后续做产品化演进讨论
- `2026-04-22`: `docs/architecture/` 目录已补充当前智能体三张 SVG 架构图 / 时序图 / 状态图与对应 `.mmd` 源文件，`current-agent-workflow.md` 现改为直接引用渲染图片，便于阅读与后续维护
