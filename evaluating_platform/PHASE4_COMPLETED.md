# 阶段 4 完成记录：专家资产管理

完成日期：2026-03-13

---

## 概述

阶段 4 在已有 MCP 工具架构（阶段 1）和计费系统（阶段 3）的基础上，实现了**专家资产管理**的全部功能：

- **P0**：资产数据库 Repository 层 + 完整 REST API Handler
- **P1**：工作流 DAG 执行引擎，专家预编排的工具序列可直接驱动评估任务
- **P2**：专家收益分成、节点间数据流、资产审核状态机

---

## P0：资产 Repository 与 Handler

### 新建文件

#### `internal/repository/asset.go`

专家资产（`assets` 表）的数据访问层，遵循项目既有 Repository 风格（pgxpool + 手动 SQL）。

| 方法 | 说明 |
|------|------|
| `Create(ctx, *model.Asset)` | 插入新资产，`Config WorkflowConfig` 序列化为 JSONB |
| `GetByID(ctx, uuid)` | 按 ID 查单条，反序列化 JSONB |
| `ListPublic(ctx, assetType, limit, offset)` | 公开市场：`visibility='public' AND status='published'`，可按 `type` 过滤，按 `call_count DESC` 排序 |
| `ListByExpert(ctx, expertID, limit, offset)` | 专家自己的全部资产（含 draft），按 `updated_at DESC` |
| `UpdateStatus(ctx, id, expertID, status)` | 带所有权校验（`AND expert_id=$3`），0 行受影响返回错误 |
| `UpdateStatusFrom(ctx, id, expertID, from, to)` | 原子状态机转换，同时校验当前状态必须为 `from`（P2 新增） |
| `PublishAsset(ctx, id, expertID)` | 允许从 `draft` 或 `testing` 发布，单条 UPDATE 原子完成（P2 新增） |
| `IncrCallCount(ctx, id)` | 原子 `call_count + 1`，每次工作流评估后调用 |

**关键设计**：`Config WorkflowConfig` 字段在 DB 中存为 JSONB，pgx 不自动序列化嵌套结构体，因此写入时 `json.Marshal`，读取时 `json.Unmarshal`。`ListPublic` 的可选 `type` 过滤使用 `$N` 编号动态拼接，避免 SQL 注入。`UpdateStatusFrom` 和 `PublishAsset` 通过 `RowsAffected() == 0` 检测无效转换，保证状态机完整性。

---

#### `internal/api/handler/asset.go`（完整重写）

原有文件为全 TODO 存根，本阶段完整实现。

**结构体**
```go
type AssetHandler struct {
    assetRepo *repository.AssetRepository
}
func NewAssetHandler(assetRepo *repository.AssetRepository) *AssetHandler
```

**请求体**
```go
type CreateAssetRequest struct {
    Name        string                // required
    Description string
    Type        model.AssetType       // required: tool_config | workflow | suite
    Visibility  model.AssetVisibility // 默认 private
    Version     string                // 默认 1.0.0
    Config      model.WorkflowConfig  // DAG 定义（JSONB）
    PriceUnit   float64
}
```

**端点实现**

| 方法 | 路由 | 行为 |
|------|------|------|
| `UploadTool` | `POST /tools` / `POST /assets` | 校验 type 合法性；expert_id 从 JWT 取；`status=draft`；写入 DB |
| `ListTools` | `GET /tools` / `GET /assets` | 默认返回 public+published（公开市场）；传 `?mine=1` 且角色为 expert/admin 时返回自己的全部资产（含 draft） |
| `PublishTool` | `PUT /tools/:id/publish` | 调用 `PublishAsset`（允许 draft 或 testing → published） |
| `SubmitForReview` | `PUT /tools/:id/submit` | 调用 `UpdateStatusFrom`（仅 draft → testing）（P2 新增） |
| `DeprecateTool` | `PUT /tools/:id/deprecate` | 调用 `UpdateStatus` 转为 deprecated，允许来自任意状态（P2 新增） |

---

### 修改文件

#### `cmd/server/main.go`

```go
// 步骤 9 - Repository 初始化
assetRepo := repository.NewAssetRepository(pgPool)

// 步骤 11 - Handler 初始化
assetHandler := handler.NewAssetHandler(assetRepo)
```

---

## P1：工作流 DAG 执行引擎

### 新建文件

#### `internal/workflow/executor.go`

新增 `workflow` 包，提供 `Executor` 结构体，将 `WorkflowConfig` DAG 转化为有序的 MCP 工具调用序列。

**执行流程**

```
WorkflowConfig
  ├── Nodes: [{id, tool_name, params, position}, ...]
  └── Edges: [{source, target}, ...]  ← 有向依赖

topoSort(Kahn 算法) → 有序 nodeID 列表

逐节点执行
  → agent.Executor.Execute(ctx, node.ToolName, mergedParams)
  → 收集 agent.ToolResult

返回 *agent.RunResult  ← 与 agent.Engine.Run 同类型
```

**拓扑排序（Kahn 算法）**

1. 构建 `inDegree` 映射和邻接表 `adj`（`edge.Source → []Target`）
2. 所有入度为 0 的节点入队
3. 弹出节点追加到 `sorted`，对其所有后继节点减少入度，新入度为 0 的节点入队
4. 若 `len(sorted) != len(Nodes)` 说明存在环，返回错误

**参数合并规则（P1）**

```
params = cfg.Params (全局) ← node.Params (节点，优先覆盖)
```

`assessment_id` 由 `agent.Executor.Execute` 内部自动注入，工具处理函数通过它从 `ConnectorPool` 获取目标连接器。

**返回值兼容性**

`*agent.RunResult`（含 `Logs []agent.ToolResult` 和 `Summary string`）与 `agent.Engine.Run` 完全相同类型，日志持久化、报告生成、计费扣费无需任何改动即可复用。

---

### 修改文件

#### `internal/api/handler/assessment.go`

**新增字段**

```go
type AssessmentHandler struct {
    engine           *agent.Engine
    workflowExecutor *workflow.Executor
    // ...
    assetRepo        *repository.AssetRepository
}
```

**`runAssessment` 执行路径分支**

```
assessment.TemplateID != nil
  └─ assetRepo.GetByID(templateID)
       ├─ 成功 且 asset.Type == AssetTypeWorkflow
       │    └─ workflowExecutor.Run(ctx, assessmentID, asset.Config)
       │         成功后 assetRepo.IncrCallCount(asset.ID)
       └─ 失败 或 非 workflow 类型
            └─ engine.Run(ctx, assessmentID, goal)  ← ReAct fallback

assessment.TemplateID == nil
  └─ engine.Run(ctx, assessmentID, goal)  ← 原有路径不变
```

**构造函数签名**

```go
func NewAssessmentHandler(
    engine           *agent.Engine,
    workflowExecutor *workflow.Executor,
    reporter         *report.Generator,
    llmClient        *llm.Client,
    pool             *mcptools.ConnectorPool,
    assessmentRepo   *repository.AssessmentRepository,
    reportRepo       *repository.ReportRepository,
    billingService   *billing.Service,
    assetRepo        *repository.AssetRepository,
) *AssessmentHandler
```

#### `cmd/server/main.go`

```go
// 步骤 8 - 初始化核心组件
workflowExecutor := workflow.NewExecutor(mcpClient)

// 步骤 11 - Handler 初始化
assessHandler := handler.NewAssessmentHandler(
    agentEngine, workflowExecutor, reportGenerator,
    llmClient, pool, assessmentRepo, reportRepo, billingService, assetRepo,
)
```

---

## P2：专家收益分成 / 节点间数据流 / 资产审核状态机

### P2.1 专家收益分成

#### `internal/billing/pricing.go`

新增常量：

```go
ExpertShareRatio = 0.30  // 工具调用收费中专家的分成比例
```

#### `internal/billing/service.go`

`ProcessAssessmentBilling` 签名扩展，新增两个可选指针参数：

```go
func (s *Service) ProcessAssessmentBilling(
    ctx             context.Context,
    userID          uuid.UUID,
    assessmentID    uuid.UUID,
    logs            []ToolCallInfo,
    totalTokensUsed int,
    assetID         *uuid.UUID,  // 工作流资产 ID（非 workflow 评估传 nil）
    expertID        *uuid.UUID,  // 资产所有者（同上）
) error
```

计费逻辑变化：

- 当 `assetID != nil && expertID != nil` 时，每条工具调用记录写入 `expert_share = amount × 0.30`，同时填充 `asset_id`、`expert_id`
- 所有工具调用记录写完后，**一次性**调用 `userRepo.UpdateBalance(+totalExpertShare)` 打款给专家，避免频繁小额事务

新增方法：

```go
func (s *Service) GetExpertEarnings(
    ctx      context.Context,
    expertID uuid.UUID,
    limit, offset int,
) (records []model.BillingRecord, total int, totalEarnings float64, err error)
```

#### `internal/repository/billing.go`

新增两个方法：

| 方法 | SQL 条件 | 说明 |
|------|---------|------|
| `ListEarnings(ctx, expertID, limit, offset)` | `WHERE expert_id=$1 AND expert_share>0` | 分页查专家收益明细 |
| `GetTotalEarnings(ctx, expertID)` | `SELECT COALESCE(SUM(expert_share), 0)` | 累计总收益 |

#### `internal/api/handler/billing.go`

新增 `GetExpertEarnings` 方法，响应格式：

```json
{
  "items": [...],
  "total": 12,
  "total_earnings": 28.50,
  "limit": 20,
  "offset": 0
}
```

同时更新 `GetToolPrices`，在返回中增加 `expert_share_ratio` 字段。

#### `internal/api/handler/assessment.go`

`runAssessment` 工作流分支额外捕获资产信息并透传给计费：

```go
var billingAssetID *uuid.UUID
var billingExpertID *uuid.UUID

if ... && asset.Type == model.AssetTypeWorkflow {
    result, err = h.workflowExecutor.Run(...)
    if err == nil {
        _ = h.assetRepo.IncrCallCount(ctx, asset.ID)
        billingAssetID = &asset.ID      // 透传
        billingExpertID = &asset.ExpertID
    }
}

// 计费时携带资产信息
h.billingService.ProcessAssessmentBilling(
    ctx, userID, assessmentID, toolCalls, tokensUsed,
    billingAssetID, billingExpertID,
)
```

---

### P2.2 节点间数据流

#### `internal/workflow/executor.go`（更新）

在执行循环中维护两个新数据结构：

```
predecessors[nodeID] = []直接前驱 nodeID   ← 从 Edges 一次性构建
nodeOutputs[nodeID]  = JSON 输出字符串     ← 每个节点执行后存储
```

参数注入优先级（低 → 高）：

```
全局 cfg.Params
  ↑ upstream_{predID}（直接前驱节点的 JSON 输出，自动解析后注入）
      ↑ node.Params（节点自身配置，最终覆盖）
```

工具收到的实际 params 示例（node2 依赖 node1）：

```json
{
  "assessment_id": "...",
  "upstream_node1": {
    "success": true,
    "severity": "high",
    "evidence": "..."
  },
  "intensity": "high"
}
```

只有成功执行（`execResult.Error == nil`）且输出非空的节点结果才会写入 `nodeOutputs`，确保错误节点不会污染下游输入。

---

### P2.3 资产审核状态机

#### 状态转换图

```
         ┌──[submit]──►  testing  ──┐
 draft ──┤                          ├──[publish]──► published
         └──────────[publish]───────┘

        任意状态 ──[deprecate]──► deprecated
```

#### `internal/repository/asset.go`（新增方法）

| 方法 | 校验逻辑 |
|------|---------|
| `UpdateStatusFrom(id, expertID, from, to)` | `WHERE id=$2 AND expert_id=$3 AND status=$4`，原子校验来源状态 |
| `PublishAsset(id, expertID)` | `WHERE id=$2 AND expert_id=$3 AND status IN ('draft','testing')`，一条 UPDATE 覆盖两种合法来源 |

两者均通过 `RowsAffected() == 0` 检测无效转换，返回含当前预期状态的错误消息。

#### `internal/api/handler/asset.go`（新增方法）

| 方法 | 路由 | 转换 | Repository 调用 |
|------|------|------|----------------|
| `SubmitForReview` | `PUT /tools/:id/submit` | `draft → testing` | `UpdateStatusFrom(draft, testing)` |
| `PublishTool`（更新） | `PUT /tools/:id/publish` | `draft → published` 或 `testing → published` | `PublishAsset` |
| `DeprecateTool` | `PUT /tools/:id/deprecate` | 任意 → `deprecated` | `UpdateStatus(deprecated)` |

所有操作均在数据库层做原子校验（带所有权验证），无需在 Handler 层预先查询状态。

#### `cmd/server/main.go`（新增路由）

```go
expert.PUT("/tools/:id/submit",    assetHandler.SubmitForReview)
expert.PUT("/tools/:id/deprecate", assetHandler.DeprecateTool)
expert.GET("/billing/earnings",    billingHandler.GetExpertEarnings)
```

---

## 完整 API 端点总览（阶段 4 全部新增）

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| `POST` | `/api/v1/tools` | expert/admin | 创建资产（初始 draft） |
| `POST` | `/api/v1/assets` | expert/admin | 同上（别名） |
| `GET` | `/api/v1/tools` | 所有已认证用户 | 公开市场（published+public） |
| `GET` | `/api/v1/tools?mine=1` | expert/admin | 查看自己的全部资产（含 draft） |
| `GET` | `/api/v1/assets?mine=1` | expert/admin | 同上（别名） |
| `PUT` | `/api/v1/tools/:id/submit` | expert/admin | 提交审核（draft → testing） |
| `PUT` | `/api/v1/tools/:id/publish` | expert/admin | 发布（draft/testing → published） |
| `PUT` | `/api/v1/tools/:id/deprecate` | expert/admin | 废弃（任意 → deprecated） |
| `GET` | `/api/v1/billing/earnings` | expert/admin | 专家收益明细 + 累计总收益 |

评估任务创建时传入 `template_id`（已有端点 `POST /api/v1/assessments`），若对应资产类型为 `workflow`，自动走 DAG 执行路径并触发专家分成计算。

---

## 完整数据流（工作流评估）

```
POST /api/v1/assessments  { template_id: "uuid-of-workflow-asset" }
  │
  ├─ billingService.CheckSufficientBalance()
  ├─ assessmentRepo.Create()  → status: pending
  └─ go runAssessment()
       │
       ├─ assessmentRepo.UpdateStatus()  → status: running
       ├─ connector.NewConnector()
       ├─ pool.Register(assessmentID, conn)
       │
       ├─ assetRepo.GetByID(templateID)  → WorkflowConfig + ExpertID
       ├─ workflowExecutor.Run(ctx, assessmentID, config)
       │    ├─ topoSort(config)  → 有序节点列表
       │    └─ 逐节点 agent.Executor.Execute()
       │         ├─ 注入 upstream_{predID} 前驱输出
       │         └─ MCP Client → MCP Server → connector → target LLM
       │
       ├─ assetRepo.IncrCallCount()
       ├─ assessmentRepo.CreateLog() × N
       ├─ report.Generator.Generate()
       ├─ reportRepo.Create()
       ├─ assessmentRepo.Complete()  → status: completed
       └─ billingService.ProcessAssessmentBilling(
              ..., assetID, expertID  ← P2 新增
          )
            ├─ 写 billing_records（含 expert_share）× N
            ├─ 扣用户余额
            └─ 打款给专家（+totalExpertShare）
```

---

## 文件变更汇总

| 文件 | 变更类型 | P 阶段 |
|------|---------|-------|
| `internal/repository/asset.go` | 新建 | P0 |
| `internal/api/handler/asset.go` | 完整重写 | P0 / P2 |
| `internal/workflow/executor.go` | 新建，P2 更新数据流 | P1 / P2 |
| `internal/api/handler/assessment.go` | 新增字段 + 分支逻辑 + 资产信息透传 | P1 / P2 |
| `internal/billing/pricing.go` | 新增 `ExpertShareRatio` 常量 | P2 |
| `internal/billing/service.go` | 修改 `ProcessAssessmentBilling` + 新增 `GetExpertEarnings` | P2 |
| `internal/repository/billing.go` | 新增 `ListEarnings`、`GetTotalEarnings` | P2 |
| `internal/api/handler/billing.go` | 新增 `GetExpertEarnings`，更新 `GetToolPrices` | P2 |
| `cmd/server/main.go` | 多次更新：assetRepo、workflowExecutor、新路由 | P0 / P1 / P2 |

---

## 验证

```bash
go build ./...   # 无编译错误
```

所有已有测试（`go test ./...`）不受影响，新代码与 agent、billing、repository 层通过接口和类型复用集成，无循环依赖。
