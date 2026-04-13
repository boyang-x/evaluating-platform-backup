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
- 平台从专家门户资源里选样本/模板/已组合攻击。
- 必要时调用外部 MCP 能力进行样本改写。
- 然后对被测 LLM 执行测试并生成报告。

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

## 3. 目录速览

### 3.1 平台主工程 `evaluating_platform/`

- `cmd/server/main.go`
  后端启动入口。负责初始化数据库、Redis、MinIO、内部 MCP Server、内部 MCP Client、外部 MCP 管理器、HTTP API。
- `internal/chat/`
  企业聊天会话、计划生成、确认执行、进度卡片更新。
- `internal/agent/`
  评测编排引擎与执行器。负责把计划落成一系列 MCP 工具调用。
- `internal/mcptools/`
  平台内部 MCP 工具集合。包括资源推荐、样本/模板预览、载荷增强、执行、报告，以及对外部 MCP 的桥接工具。
- `internal/externalmcp/`
  外部 MCP 服务管理层。负责连接、同步、代理第三方 MCP 工具到平台内部。
- `internal/repository/`
  数据库访问层。
- `internal/report/`
  报告生成逻辑。
- `internal/sample/`
  样本加载、预览、CSV 解析。
- `internal/api/`
  HTTP handler 与中间件。
- `frontend/`
  React + Vite 前端。
- `docs/external-mcp/`
  外部 MCP 接入规范与模板。
- `docs/qagent-controlled-orchestration.md`
  QAgent 受控编排接入方案草案，说明如何在不丢失流程控制权的前提下，引入 QAgent 与 skill。
- `docs/qagent-skill-workflow.md`
  描述未来接入 skill 且由 QAgent 进行受控编排时，推荐采用的端到端流程。
- `docker-compose.yml`
  本地联调入口，负责拉起 PostgreSQL、Redis、MinIO、backend、frontend、CCBOS MCP。

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
- CCBOS MCP：`http://localhost:18191`
- PostgreSQL：`5432`
- Redis：`6379`
- MinIO S3：`9000`
- MinIO Console：`9001`

常用启动方式：

在 `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform` 下执行：

```powershell
docker compose up -d --build backend ccbos_mcp
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

### 5.3 关于外部 MCP

- 平台已支持把第三方 MCP Server 同步成内部代理工具。
- 编排层默认不应暴露原始代理工具名，必要时走平台封装后的业务工具。
- 大结果集能力优先使用“句柄式返回”，避免把完整敏感内容直接暴露给编排模型。

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
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\internal\mcptools\external_rewrite_tools.go`
  样本送入 CCBOS 改写并导回会话。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\internal\externalmcp\manager.go`
  外部 MCP 连接/同步/调用。
- `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform\internal\externalmcp\sse_client.go`
  自定义 SSE 客户端，解决长调用超时问题。
- `C:\Users\wangboyang\Desktop\evaluating_platform\CC-BOS\mcp-server\rewrite.go`
- `C:\Users\wangboyang\Desktop\evaluating_platform\CC-BOS\mcp-server\optimization.go`
- `C:\Users\wangboyang\Desktop\evaluating_platform\CC-BOS\mcp-server\response_parsing.go`
  CCBOS 返回结果解析与容错。

## 7. 最近已确认的事实

- 企业侧“文言文越狱测试”已经能直接生成计划卡片。
- `resource_mode_preference` 已支持 `sample_rewrite`。
- 平台内部 MCP Client 已切到自定义超时感知 SSE 客户端，避免默认 60 秒响应超时截断长流程。
- CCBOS MCP 的返回解析已做容错增强，可兼容多种 JSON 形状。
- 当前如果专家门户只有合规检测样本，那么文言文链路会基于这些现有样本执行；这属于现有数据约束，不是链路阻塞。

## 8. 常用测试命令

在 `C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform` 下：

```powershell
go test ./internal/externalmcp ./internal/chat ./internal/agent ./internal/mcptools
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

## 9. 新线程建议阅读顺序

如果以后开新线程，建议先读：

1. 本文件 `PROJECT_GUIDE.md`
2. `evaluating_platform/docs/architecture/agent-architecture.svg`
3. `evaluating_platform/docs/qagent-controlled-orchestration.md`
4. `evaluating_platform/docs/qagent-skill-workflow.md`
5. `evaluating_platform/cmd/server/main.go`
6. `evaluating_platform/internal/chat/session.go`
7. `evaluating_platform/internal/agent/engine.go`
8. `evaluating_platform/internal/mcptools/external_rewrite_tools.go`
9. `evaluating_platform/internal/externalmcp/`
10. `CC-BOS/mcp-server/`

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
