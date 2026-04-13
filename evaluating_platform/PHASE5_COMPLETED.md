# Phase 5 完成记录：前端集成 + 实时推送 + 管理员界面

## 完成时间
2026-03-14

---

## 总体目标

打通前后端完整数据流、实现 SSE 实时日志推送、新建系统管理员界面（含资产审核和用户管理）。

---

## 子阶段 5.0 — 后端扩展

### 5.0.1 SSE 实时日志广播

**新建 `internal/hub/broadcast.go`**
- `LogEvent` 结构体：assessment_id、iteration、tool_name、output、severity、tokens_used、duration_ms、timestamp
- `LogHub`：thread-safe pub/sub，`sync.RWMutex` + `map[string][]chan LogEvent`
- `Subscribe(assessmentID)` → 返回 channel + unsubscribe 函数
- `Publish(event)` → 非阻塞 fan-out（slow consumer 直接丢弃）

**修改 `internal/agent/executor.go`**
- 新增 `hub *hub.LogHub` 字段
- `WithHub(h)` 链式方法
- `SetIteration(n int)` 方法
- `publish()` 辅助方法：每次工具调用完成后广播 LogEvent

**修改 `internal/agent/engine.go`**
- 新增 `hub *hub.LogHub` 字段
- `SetHub(h)` 方法
- `Run()` 中为每个 executor 注入 hub 并设置 iteration

**修改 `internal/api/handler/assessment.go`**
- 新增 `logHub *hub.LogHub` 和 `jwtSecret string` 字段
- 新增 `StreamLogs()` SSE 处理方法：
  - 支持 `?token=` query param（EventSource 不支持自定义 header）
  - 支持 `Authorization: Bearer` header（备用）
  - JWT 验证 + 任务归属检查（admin 可看所有）
  - 立即发送 `: connected` 注释触发响应头刷新
  - 25s 心跳 ticker 防止连接超时
  - 响应头：`Content-Type: text/event-stream`、`Cache-Control: no-cache`、`X-Accel-Buffering: no`

### 5.0.2 管理员 API

**新建 `internal/api/handler/admin.go`**
- `ListUsers` — GET /admin/users（支持 limit/offset/role/keyword 过滤）
- `UpdateUserRole` — PUT /admin/users/:id/role
- `SetUserActive` — PUT /admin/users/:id/active
- `ListPendingAssets` — GET /admin/assets/pending
- `ApproveAsset` — PUT /admin/assets/:id/approve（testing → published，visibility → public）
- `RejectAsset` — PUT /admin/assets/:id/reject（testing → draft，含驳回原因）
- `GetStats` — GET /admin/stats（聚合：total_users、users_by_role、total_assessments、pending_assets、total_revenue、weekly_assessments[7]）

**修改 `internal/repository/user.go`**
- 所有 SELECT 查询补充 `is_active` 字段扫描
- 新增 `ListAll(limit, offset, role, keyword)` — 支持关键词搜索
- 新增 `UpdateRole(id, role)`
- 新增 `SetActive(id, active)`
- 新增 `CountByRole()` → `map[string]int`

**修改 `internal/repository/asset.go`**
- 新增 `ListByStatus(status, limit, offset)` — 无所有权限制
- 新增 `AdminApprove(id)` — testing → published，visibility = public
- 新增 `AdminReject(id)` — testing → draft
- 新增 `CountPendingAssets()`

**修改 `internal/repository/billing.go`**
- 新增 `SumTotalRevenue()` — 统计平台总收入

**修改 `internal/repository/assessment.go`**
- 新增 `CountStats()` — 返回总数 + 近 7 天每日评估数

**修改 `internal/model/user.go`**
- 新增 `IsActive bool` 字段（DB 列已存在，model 缺失导致 scan 失败）

**修改 `cmd/server/main.go`**
- 创建 `logHub := hub.NewLogHub()`，注入 `agentEngine.SetHub(logHub)`
- `NewAssessmentHandler` 传入 `logHub` 和 `cfg.Auth.JWTSecret`
- 新增 `adminHandler := handler.NewAdminHandler(...)`
- 新增 SSE 路由：`GET /api/v1/assessments/:id/stream`
- 新增 admin 路由组（`RequireRole("admin")` 中间件）：
  ```
  GET  /api/v1/admin/users
  PUT  /api/v1/admin/users/:id/role
  PUT  /api/v1/admin/users/:id/active
  GET  /api/v1/admin/assets/pending
  PUT  /api/v1/admin/assets/:id/approve
  PUT  /api/v1/admin/assets/:id/reject
  GET  /api/v1/admin/stats
  ```
- Redis 连接改为非 fatal（当前未使用，不影响启动）

---

## 子阶段 5.1 — 前端基础

### Service 层（全部新建）

| 文件 | 说明 |
|------|------|
| `src/services/api.ts` | Axios 基础实例，baseURL 从 `VITE_API_BASE` 读取，请求拦截器注入 JWT，响应拦截器处理 401 |
| `src/services/auth.ts` | login/logout/register/getStoredUser/getToken |
| `src/services/assessment.ts` | create/list/get/cancel/streamLogs（EventSource SSE 客户端） |
| `src/services/report.ts` | get/getByAssessmentID/download |
| `src/services/billing.ts` | getBalance/getRecords/getTransactions/recharge/getEarnings/getPrices |
| `src/services/asset.ts` | listPublic/listMine/get/create/update/publish/submitForReview/deprecate |
| `src/services/admin.ts` | listUsers/updateUserRole/setUserActive/listPendingAssets/approveAsset/rejectAsset/getStats |

### AuthContext & ProtectedRoute

**新建 `src/context/AuthContext.tsx`**
- `AuthProvider`：从 localStorage 恢复 user，提供 login/logout
- `useAuth()` hook

**新建 `src/components/ProtectedRoute.tsx`**
- 未登录 → 重定向 `/login`
- 角色不匹配 → 重定向到对应角色默认路由

### App.tsx 重构
- `<AuthProvider>` 包裹全部路由
- 所有门户路由用 `<ProtectedRoute allowedRoles={[...]}>` 保护
- 新增 `/admin/*` 路由组

---

## 子阶段 5.2 — 企业门户

| 文件 | 改动 |
|------|------|
| `Login.tsx` | 去掉 role 选择器，调用真实 API，按 role 路由 |
| `EnterprisePortal.tsx` | 真实余额/用户名，补充菜单（billing/market/settings） |
| `Dashboard.tsx` | 调用 `assessment.list()` 计算统计，展示最近 5 条 |
| `AssessmentList.tsx` | 真实数据 + 5s 自动轮询 + SSE Drawer（width=600，实时日志） |
| `NewAssessment.tsx` | 调用 `assessment.create()`，支持 template_id，处理 402 |
| `ReportView.tsx` | 调用 `report.get(id)`，动态渲染 findings |
| `BillingPage.tsx` | 新建：余额/充值/账单/交易 Tab 布局 |
| `MarketPlace.tsx` | 新建：公开资产卡片 Grid，"发起评估"跳转 |
| `Settings.tsx` | 新建：账户信息展示 |

---

## 子阶段 5.3 — 专家门户

| 文件 | 改动 |
|------|------|
| `ExpertPortal.tsx` | 真实收益/用户名，补充 `/expert/market` 菜单 |
| `ToolManager.tsx` | 调用 `assetService` 真实 CRUD（create/publish/submitForReview/deprecate） |
| `WorkflowEditor.tsx` | 完整重写：React Flow 拖拽编辑器，左侧工具库，右侧属性 Drawer，保存转换为 WorkflowConfig |
| `AssetMarket.tsx` | 新建：只读浏览公开资产，详情 Modal 展示节点列表 |

**依赖安装：** `npm install reactflow`

---

## 子阶段 5.4 — 管理员门户（全部新建）

| 文件 | 说明 |
|------|------|
| `AdminPortal.tsx` | 紫色主题（#8b5cf6），15s 轮询待审数量 Badge，底部统计卡 |
| `AdminDashboard.tsx` | 4 个统计卡（用户/评估/待审/收入），角色分布，最近评估表格 |
| `UserManagement.tsx` | 分页表格，内联 Select 改角色，Switch 启用/禁用 |
| `AssetReview.tsx` | 待审资产表格，展开行显示节点，通过/驳回操作，驳回 Modal |

---

## Bug 修复

| 问题 | 修复 |
|------|------|
| `model/user.go` 缺少 `IsActive` 字段 | 添加 `IsActive bool` 字段，修复所有 scan |
| 迁移文件 admin 密码 hash 错误 | 更新为正确的 bcrypt hash（admin123456） |
| SSE 响应头未立即刷新 | 添加 `: connected` 初始注释触发 flush |
| `WorkflowEditor.tsx` 对象字面量重复 `label` 属性 | 将字符串 label 重命名为 `labelStr` |
| `AssetReview.tsx` 未使用的 `Collapse` import | 移除 import 和 suppress 变量 |
| `Dashboard.tsx` 未使用的 `riskConfig` | 移除定义和 suppress 变量 |
| Redis 连接失败导致服务无法启动 | 改为 Warn 级别日志，非 fatal |

---

## 验证结果

### 后端编译
```
go build ./...  → 无错误
```

### 前端构建
```
npm run build   → 无错误（bundle 1.4MB，含 reactflow + antd）
```

### API 测试（全部通过）

**SSE 端点**
- ✅ `?token=` query param 认证
- ✅ `Authorization: Bearer` header 认证
- ✅ 无效 token → `{"error":"invalid token"}`
- ✅ 缺少 token → `{"error":"missing token"}`
- ✅ 连接保持，25s 心跳
- ✅ 立即返回 `: connected` 注释

**管理员 API**
- ✅ `GET /admin/stats` → 用户数/评估数/待审数/收入/周数据
- ✅ `GET /admin/users` → 分页用户列表含 is_active
- ✅ `GET /admin/assets/pending` → 待审资产列表
- ✅ `PUT /admin/assets/:id/approve` → testing → published

**企业 API**
- ✅ `GET /billing/balance` → 余额
- ✅ `GET /billing/prices` → 工具定价
- ✅ `POST /billing/recharge` → 充值成功
- ✅ `GET /billing/transactions` → 交易记录
- ✅ `GET /assessments` → 评估列表
- ✅ `GET /tools` → 公开资产市场

**专家 API**
- ✅ `POST /assets` → 创建资产（draft）
- ✅ `GET /assets?mine=1` → 我的资产列表
- ✅ `PUT /tools/:id/submit` → draft → testing
- ✅ `GET /billing/earnings` → 收益记录

**完整资产生命周期**
- ✅ 专家创建（draft）→ 提交审核（testing）→ 管理员审核通过（published）→ 市场可见

---

## 测试账户

| 邮箱 | 角色 | 密码 |
|------|------|------|
| admin@platform.local | admin | admin123456 |
expert@demo.com	12345678	专家
enterprise@demo.com	12345678	企业

---

## 待完成（Phase 6）

- PDF 报告导出
- Prometheus 监控指标
- Redis 实际使用（会话缓存/限流）
- 真实 LLM API 接入后的端到端评估测试
