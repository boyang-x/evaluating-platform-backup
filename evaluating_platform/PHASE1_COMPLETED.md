# AI 安全评估平台 - 阶段 1 完成

## 本次更新内容

### 1. 工具接口标准化（OpenAI + LangChain 双格式兼容）

**新增文件：**
- `internal/tools/schema.go` - 完整 JSON Schema 支持，双格式导出

**核心特性：**
- `ParamDef` 增强：支持 `enum`、`default`、`items`（array）、`sub_props`（object）
- `ToOpenAIToolDef()` - 导出为 OpenAI Function Calling 格式
- `ToLangChainToolDef()` - 导出为 LangChain 兼容格式
- `ToToolMetas()` - 导出完整元数据（含双格式 schema）

**API 端点：**
```bash
GET /api/v1/tools/schemas?format=openai      # OpenAI 格式
GET /api/v1/tools/schemas?format=langchain   # LangChain 格式
GET /api/v1/tools/schemas?format=full        # 完整元数据
GET /api/v1/tools/schemas?category=llm       # 按分类过滤
```

### 2. Repository 层（数据库访问封装）

**新增文件：**
- `internal/repository/user.go` - 用户 CRUD + 密码验证 + 余额管理
- `internal/repository/assessment.go` - 评估任务 + 日志 + 目标系统 + 取消
- `internal/repository/report.go` - 报告查询

**核心功能：**
- 用户注册/登录（bcrypt 密码哈希）
- 评估任务状态持久化（pending → running → completed/failed/canceled）
- 评估日志批量写入
- 报告生成与存储

### 3. API Handler 接入数据库

**更新文件：**
- `internal/api/handler/auth.go` - 真实用户认证（替换 mock）
- `internal/api/handler/assessment.go` - 异步评估执行 + 状态持久化 + 取消支持
- `internal/api/handler/report.go` - 报告查询

**关键改进：**
- 评估任务创建后立即返回 `assessment_id`，后台异步执行
- 使用独立 `context.Background()` 避免 HTTP 请求取消影响后台任务
- `sync.Map cancelFuncs` 存储各任务的 goroutine cancel 函数，支持取消中断
- 完整的错误处理和日志记录

### 4. 依赖注入完成

**更新文件：**
- `cmd/server/main.go` - 完整依赖注入链

**注入链：**
```
Config → DB Pool → Repository → Handler → Router
                ↘ LLM Client → Agent Engine
                ↘ Tool Registry (双格式, 7 个工具)
```

### 5. LLM 客户端封装（pkg/llm）

**文件：**
- `pkg/llm/client.go` - OpenAI-compatible HTTP 客户端，支持普通调用和 SSE 流式
- `pkg/llm/types.go` - 完整请求/响应结构体（Message、ToolCall、ChatRequest 等）

### 6. ReAct Agent 引擎

**文件：**
- `internal/agent/engine.go` - ReAct 主循环（Action → Observation → Reflection）
- `internal/agent/planner.go` - LLM 驱动的评估计划生成
- `internal/agent/executor.go` - 工具调度执行器（含耗时统计）
- `internal/agent/context.go` - Agent 上下文与对话历史管理

**循环逻辑：**
```
[系统提示 + 工具列表] → LLM 决策 → 工具调用 → 结果注入上下文 → 循环
                                 ↓ finish_reason=stop
                           generateSummary → RunResult
```

### 7. 目标连接器（OpenAI Connector）

**文件：**
- `internal/connector/interface.go` - TargetConnector 接口定义
- `internal/connector/openai.go` - OpenAI-compatible 目标连接器
- `internal/connector/custom.go` - 自定义目标连接器

### 8. 工具库（7 个工具）

**LLM 安全类（`internal/tools/llm/`）：**

| 工具名 | 文件 | 说明 |
|--------|------|------|
| `prompt_injection` | `prompt_injection.go` | 提示词注入检测，5 种 payload，启发式漏洞判定 |
| `jailbreak` | `jailbreak.go` | 越狱攻击测试，多策略尝试 |
| `compliance_check` | `compliance_check.go` | 合规布尔检查（PII/有害内容/偏见/数据安全） |
| `compliance_evaluator` | `compliance_evaluator.go` | 合规量化评分（0-100分，4维度加权，输出等级A-F） |

**Agent 安全类（`internal/tools/agent/`）：**

| 工具名 | 文件 | 说明 |
|--------|------|------|
| `goal_hijacking` | `goal_hijacking.go` | Agent 目标劫持测试 |
| `tool_poisoning` | `tool_poisoning.go` | 工具投毒攻击测试 |

**报告类（`internal/tools/`）：**

| 工具名 | 文件 | 说明 |
|--------|------|------|
| `report_generator` | `report_generator.go` | Agent 可在 ReAct 循环内主动调用，接受 findings 参数生成结构化 JSON 报告 |

> **`compliance_evaluator` vs `compliance_check` 区别：**
> - `compliance_check`：布尔判断，输出 pass/fail 列表
> - `compliance_evaluator`：量化评分，输出 0-100 综合分 + 各维度得分 + 等级（A-F）

> **`report_generator`（工具）vs `internal/report/generator.go`（服务）区别：**
> - `internal/report/generator.go`：后处理服务，Agent 结束后由 Handler 调用
> - `tools/report_generator.go`：注册在工具表中，LLM 在 ReAct 循环最后一步主动调用

### 9. 评估报告生成服务

**文件：**
- `internal/report/generator.go` - LLM 驱动的报告生成，输出结构化 JSON（findings + metrics + summary）

---

## 阶段 1 验收标准

### ✅ 已完成

1. **LLM 客户端封装**
   - [x] OpenAI-compatible Chat API
   - [x] 流式输出（SSE）
   - [x] Function Calling 支持

2. **ReAct 引擎核心循环**
   - [x] Action → Observation → Reflection 三阶段循环
   - [x] 最大迭代次数限制（防死循环）
   - [x] 自动生成最终评估总结

3. **工具注册中心**
   - [x] 线程安全 Registry（`sync.RWMutex`）
   - [x] 支持 OpenAI Function Calling 格式导出
   - [x] 支持 LangChain Tool 格式导出
   - [x] 双格式 API 端点

4. **3 个原计划基础工具**
   - [x] `prompt_injection` - 提示词注入测试
   - [x] `compliance_evaluator` - 合规量化评分（0-100）
   - [x] `report_generator` - Agent 可调用的报告生成工具

5. **目标连接器**
   - [x] OpenAI-compatible Connector
   - [x] Custom Connector

6. **数据库持久化**
   - [x] 用户注册/登录（bcrypt）
   - [x] 评估任务 CRUD
   - [x] 评估日志存储
   - [x] 报告生成与查询

7. **异步任务执行**
   - [x] 评估任务后台执行
   - [x] 状态实时更新（pending → running → completed）
   - [x] 错误处理与日志

8. **API 完整性**
   - [x] `POST /api/v1/auth/register`
   - [x] `POST /api/v1/auth/login`
   - [x] `POST /api/v1/assessments`（创建并异步执行）
   - [x] `GET  /api/v1/assessments`（分页列表）
   - [x] `GET  /api/v1/assessments/:id`（详情 + 报告摘要）
   - [x] `POST /api/v1/assessments/:id/cancel`（取消任务）
   - [x] `GET  /api/v1/reports/:id`
   - [x] `GET  /api/v1/tools/schemas`（双格式）

---

## 取消任务说明

取消接口会执行两步操作：

1. **中断 goroutine**：通过 `cancelFuncs sync.Map` 找到对应任务的 `context.CancelFunc` 并调用，立即中断 Agent 正在进行的 LLM 调用或工具执行
2. **更新数据库**：将状态更新为 `canceled`，仅允许 `pending` 或 `running` 状态的任务被取消，且只能取消自己的任务

```bash
curl -X POST http://localhost:8080/api/v1/assessments/ASSESSMENT_ID/cancel \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 本地测试步骤

### 1. 启动基础设施

```bash
docker-compose up -d postgres redis minio minio-init
docker-compose ps
```

### 2. 配置环境变量

```bash
cp .env.example .env
# 填入：LLM_API_KEY=sk-xxxx  JWT_SECRET=your-secret
```

### 3. 启动后端

```bash
go mod tidy
go run ./cmd/server
```

### 4. 完整 API 测试

```bash
# 1. 注册
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","name":"测试用户","role":"enterprise","org_name":"测试公司"}'

# 2. 登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# 3. 创建评估任务
curl -X POST http://localhost:8080/api/v1/assessments \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"name":"测试评估","goal":"测试目标 LLM 是否容易受到提示词注入攻击","target_type":"openai","target_url":"https://api.openai.com/v1","target_key":"sk-xxxx","target_model":"gpt-4o"}'

# 4. 查询评估状态
curl http://localhost:8080/api/v1/assessments/ASSESSMENT_ID \
  -H "Authorization: Bearer YOUR_TOKEN"

# 5. 取消评估任务
curl -X POST http://localhost:8080/api/v1/assessments/ASSESSMENT_ID/cancel \
  -H "Authorization: Bearer YOUR_TOKEN"

# 6. 获取工具 schema（OpenAI 格式，共 7 个工具）
curl http://localhost:8080/api/v1/tools/schemas?format=openai

# 7. 获取工具 schema（LangChain 格式）
curl http://localhost:8080/api/v1/tools/schemas?format=langchain
```

---

## 数据库表结构

```sql
users                  -- 用户表
target_systems         -- 目标系统表
assets                 -- 专家资产表
assessments            -- 评估任务表（status: pending|running|completed|failed|canceled）
assessment_logs        -- 评估日志表
reports                -- 报告表
billing_records        -- 计费记录表
balance_transactions   -- 余额变动表
audit_logs             -- 审计日志表
```

---

## 下一步（阶段 2）

1. **工具库扩展**
   - 补充剩余 5 个工具（`memory_injection`, `multi_turn_attack`, `rag_poisoning`, `agent_loop_detection`, `sensitive_output_check`）
   - 工具参数配置化（支持专家自定义）

2. **计费系统**
   - 工具调用计量中间件
   - Token 用量统计（对接 Higress）
   - 余额扣费与账单生成

3. **专家资产管理**
   - 工作流 DAG 定义与执行
   - 资产发布与版本管理
   - 专家收益分成

4. **前端集成**
   - 企业门户完整流程
   - 专家门户工作流编排（React Flow）
   - 实时日志推送（WebSocket）

---

## 已知问题

- [ ] PDF/HTML 报告导出未实现（阶段 6）
- [ ] 专家资产调用未实现（阶段 4）
- [ ] 流式 WebSocket 推送未实现（阶段 5）
- [ ] Higress 集成未测试（需要实际部署）

---

## 技术栈

- **后端**: Go 1.21 + Gin + pgx/v5 + go-redis/v9
- **数据库**: PostgreSQL 16 + Redis 7
- **LLM**: OpenAI-compatible API
- **前端**: React 18 + Vite + Ant Design
- **网关**: Higress AI Gateway
- **部署**: Docker + Docker Compose

---

## 贡献者

- 阶段 0: 基础设施准备（数据库 Schema + Docker + Config）
- 阶段 1: 核心引擎 MVP（工具标准化 + Repository + Handler + Agent 引擎 + 3 个原计划工具补齐）
