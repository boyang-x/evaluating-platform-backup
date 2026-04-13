package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"

	"evaluating_platform/internal/agent"
	"evaluating_platform/internal/api/handler"
	"evaluating_platform/internal/api/middleware"
	"evaluating_platform/internal/billing"
	chatpkg "evaluating_platform/internal/chat"
	"evaluating_platform/internal/detector"
	"evaluating_platform/internal/externalmcp"
	"evaluating_platform/internal/hub"
	"evaluating_platform/internal/mcptools"
	"evaluating_platform/internal/report"
	"evaluating_platform/internal/repository"
	"evaluating_platform/internal/sample"
	"evaluating_platform/internal/workflow"
	"evaluating_platform/pkg/config"
	"evaluating_platform/pkg/db"
	"evaluating_platform/pkg/llm"
	"evaluating_platform/pkg/logger"
	"evaluating_platform/pkg/storage"
)

func main() {
	// ── 1. 加载配置 ────────────────────────────────────────────
	configPath := "configs/config.yaml"
	if _, err := os.Stat("configs/config.local.yaml"); err == nil {
		configPath = "configs/config.local.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("[FATAL] load config: %v", err)
	}

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
		logger.SetLevel(logger.INFO)
	}
	logger.Info("config loaded", map[string]interface{}{
		"mode":     cfg.Server.Mode,
		"port":     cfg.Server.Port,
		"model":    cfg.LLM.Model,
		"mcp_port": cfg.MCP.Port,
	})

	// ── 2. 连接数据库 ──────────────────────────────────────────
	ctx := context.Background()

	pgPool, err := db.NewPostgresPool(ctx, cfg.Database.DSN, cfg.Database.MaxConns, cfg.Database.MinConns)
	if err != nil {
		log.Fatalf("[FATAL] connect postgres: %v", err)
	}
	defer pgPool.Close()
	logger.Info("postgres connected")

	if cfg.Database.MigrateOnStart {
		if err := db.MigrateUp(ctx, pgPool, "migrations/001_init.sql"); err != nil {
			log.Fatalf("[FATAL] run migration: %v", err)
		}
		if err := db.MigrateUp(ctx, pgPool, "migrations/002_chat.sql"); err != nil {
			log.Fatalf("[FATAL] run chat migration: %v", err)
		}
		if err := db.MigrateUp(ctx, pgPool, "migrations/003_tool_categories.sql"); err != nil {
			log.Fatalf("[FATAL] run tool categories migration: %v", err)
		}
		if err := db.MigrateUp(ctx, pgPool, "migrations/004_app_detector.sql"); err != nil {
			log.Fatalf("[FATAL] run app detector migration: %v", err)
		}
		if err := db.MigrateUp(ctx, pgPool, "migrations/005_fix_attack_samples_columns.sql"); err != nil {
			log.Fatalf("[FATAL] run attack_samples column fix migration: %v", err)
		}
		if err := db.MigrateUp(ctx, pgPool, "migrations/006_auxiliary_llm_configs.sql"); err != nil {
			log.Fatalf("[FATAL] run auxiliary_llm_configs migration: %v", err)
		}
		if err := db.MigrateUp(ctx, pgPool, "migrations/007_llm_configs.sql"); err != nil {
			log.Fatalf("[FATAL] run llm_configs migration: %v", err)
		}
		if err := db.MigrateUp(ctx, pgPool, "migrations/008_remove_collecting_api_state.sql"); err != nil {
			log.Fatalf("[FATAL] run remove_collecting_api_state migration: %v", err)
		}
		if err := db.MigrateUp(ctx, pgPool, "migrations/009_template_upload_batches.sql"); err != nil {
			log.Fatalf("[FATAL] run template upload batches migration: %v", err)
		}
		if err := db.MigrateUp(ctx, pgPool, "migrations/010_composed_attacks.sql"); err != nil {
			log.Fatalf("[FATAL] run composed attacks migration: %v", err)
		}
		if err := db.MigrateUp(ctx, pgPool, "migrations/011_external_mcp_servers.sql"); err != nil {
			log.Fatalf("[FATAL] run external mcp migration: %v", err)
		}
		if err := db.MigrateUp(ctx, pgPool, "migrations/012_external_mcp_upstream_config.sql"); err != nil {
			log.Fatalf("[FATAL] run external mcp upstream config migration: %v", err)
		}
		logger.Info("database migration completed")
	}

	// ── 3. 连接 Redis（可选，当前未使用）─────────────────────────
	rdb, err := db.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		logger.Warn("redis unavailable, continuing without it", map[string]interface{}{"err": err.Error()})
	} else {
		defer rdb.Close()
		logger.Info("redis connected")
	}

	// ── 4. 初始化 LLM 客户端 ──────────────────────────────────
	if cfg.LLM.APIKey == "" {
		logger.Warn("LLM_API_KEY not set, LLM features will not work")
	}
	llmClient := llm.NewClient(cfg.LLM.BaseURL, cfg.LLM.APIKey, cfg.LLM.Model)

	// ── 4.1 初始化 MinIO 存储 ─────────────────────────────────
	var minioClient *storage.MinIOClient
	if cfg.Storage.Endpoint != "" {
		mc, err := storage.NewMinIOClient(cfg.Storage)
		if err != nil {
			logger.Warn("MinIO unavailable, sample/package features disabled", map[string]interface{}{"err": err.Error()})
		} else {
			minioClient = mc
			logger.Info("MinIO connected", map[string]interface{}{"endpoint": cfg.Storage.Endpoint})
		}
	}

	// ── 5. 创建 ConnectorPool ─────────────────────────────────
	pool := mcptools.NewConnectorPool()

	// ── 5.1 初始化 Repository（MCP Server 需要）────────────────
	sampleRepo := repository.NewAttackSampleRepository(pgPool)
	composedAttackRepo := repository.NewComposedAttackRepository(pgPool)
	tplRepo := repository.NewTemplateRepository(pgPool)
	assessmentRepo := repository.NewAssessmentRepository(pgPool)
	auxLLMRepo := repository.NewAuxiliaryLLMRepository(pgPool)
	orchLLMRepo := repository.NewOrchestrationLLMRepository(pgPool)
	targetLLMRepo := repository.NewTargetLLMRepository(pgPool)
	reportRepo := repository.NewReportRepository(pgPool)
	externalMCPServerRepo := repository.NewExternalMCPServerRepository(pgPool)
	externalMCPToolRepo := repository.NewExternalMCPToolRepository(pgPool)

	// ── 5.2 初始化样本管理组件 ─────────────────────────────────
	var sampleManager *sample.Manager
	var sampleLoader *sample.Loader
	var composedAttackManager *sample.ComposedAttackManager
	var composedAttackLoader *sample.ComposedAttackLoader
	if minioClient != nil {
		sampleManager = sample.NewManager(sampleRepo, minioClient)
		sampleLoader = sample.NewLoader(sampleRepo, minioClient)
		composedAttackManager = sample.NewComposedAttackManager(composedAttackRepo, minioClient)
		composedAttackLoader = sample.NewComposedAttackLoader(composedAttackRepo, minioClient)
		logger.Info("sample manager/loader initialized")
	}

	// ── 6. 创建并启动 MCP Server（goroutine 中运行）──────────
	sessionStore := mcptools.NewSessionStore(30 * time.Minute)
	mcpServer := mcptools.NewMCPServer(sampleLoader, composedAttackLoader, tplRepo, sampleRepo, composedAttackRepo, auxLLMRepo, targetLLMRepo, minioClient, reportRepo, sessionStore)
	externalMCPManager := externalmcp.NewManager(externalMCPServerRepo, externalMCPToolRepo, mcpServer)
	mcptools.RegisterCCBOSRewriteTools(mcpServer, sampleLoader, sessionStore, assessmentRepo, targetLLMRepo, externalMCPServerRepo, externalMCPManager)
	go func() {
		if err := mcptools.StartSSEServer(mcpServer, cfg.MCP.Port); err != nil {
			log.Fatalf("[FATAL] MCP SSE server: %v", err)
		}
	}()
	logger.Info("MCP server starting", map[string]interface{}{"port": cfg.MCP.Port})

	// 等待 MCP Server 就绪
	time.Sleep(150 * time.Millisecond)

	// ── 7. 创建 MCP Client 并连接 ─────────────────────────────
	mcpServerURL := fmt.Sprintf("http://localhost:%d/sse", cfg.MCP.Port)
	internalMCPTimeout := 10 * time.Minute
	if configured := time.Duration(cfg.Agent.ToolTimeoutSeconds) * time.Second; configured > internalMCPTimeout {
		internalMCPTimeout = configured
	}
	mcpHTTPClient := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ResponseHeaderTimeout: 15 * time.Second,
			Proxy:                 http.ProxyFromEnvironment,
		},
	}
	mcpClient, err := externalmcp.NewTimeoutAwareSSEClient(mcpServerURL, internalMCPTimeout, nil, mcpHTTPClient)
	if err != nil {
		log.Fatalf("[FATAL] create MCP client: %v", err)
	}

	clientCtx, clientCancel := context.WithCancel(context.Background())
	defer clientCancel()

	if err := mcpClient.Start(clientCtx); err != nil {
		log.Fatalf("[FATAL] start MCP client: %v", err)
	}

	// MCP 协议握手初始化
	_, err = mcpClient.Initialize(clientCtx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "ai-security-evaluator",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		log.Fatalf("[FATAL] MCP initialize: %v", err)
	}
	logger.Info("MCP client connected")

	if err := externalMCPManager.Bootstrap(ctx); err != nil {
		logger.Warn("external MCP bootstrap completed with warnings", map[string]interface{}{"err": err.Error()})
	} else {
		logger.Info("external MCP bootstrap completed")
	}

	// ── 8. 初始化核心组件 ──────────────────────────────────────
	toolTimeout := time.Duration(cfg.Agent.ToolTimeoutSeconds) * time.Second
	agentEngine := agent.NewEngine(llmClient, mcpClient, cfg.Agent.MaxIterations, toolTimeout)
	workflowExecutor := workflow.NewExecutor(mcpClient)
	reportGenerator := report.NewGenerator(llmClient, minioClient)

	// ── 8.1 初始化 LogHub（SSE 实时日志广播）────────────────────
	logHub := hub.NewLogHub()
	agentEngine.SetHub(logHub)

	// ── 9. 初始化 Repository ───────────────────────────────────
	userRepo := repository.NewUserRepository(pgPool)
	billingRepo := repository.NewBillingRepository(pgPool)
	assetRepo := repository.NewAssetRepository(pgPool)
	chatRepo := repository.NewChatRepository(pgPool)
	// sampleRepo, tplRepo, auxLLMRepo, reportRepo 已在 5.1 初始化
	detectorRepo := repository.NewAppDetectorRepository(pgPool)

	// ── 10. 初始化 Billing Service ──────────────────────────────
	billingService := billing.NewService(userRepo, billingRepo)

	// ── 11. 初始化 Handler ─────────────────────────────────────
	authHandler := handler.NewAuthHandler(cfg.Auth.JWTSecret, userRepo)
	assessHandler := handler.NewAssessmentHandler(
		agentEngine, workflowExecutor, reportGenerator, llmClient, pool,
		assessmentRepo, reportRepo, billingService, assetRepo,
		logHub, cfg.Auth.JWTSecret,
	)
	reportHandler := handler.NewReportHandler(reportRepo, minioClient)
	assetHandler := handler.NewAssetHandler(assetRepo)
	billingHandler := handler.NewBillingHandler(billingService)
	adminHandler := handler.NewAdminHandler(userRepo, assetRepo, assessmentRepo, billingRepo)
	chatManager := chatpkg.NewManager(chatRepo, assessmentRepo, orchLLMRepo, targetLLMRepo, agentEngine, reportGenerator, reportRepo, pool, logHub, billingService)
	chatHandler := handler.NewChatHandler(chatManager, chatRepo)

	// 攻击样本 & 模版 Handler
	var sampleHandler *handler.AttackSampleHandler
	var composedAttackHandler *handler.ComposedAttackHandler
	var detectorHandler *handler.AppDetectorHandler
	if sampleManager != nil {
		sampleHandler = handler.NewAttackSampleHandler(sampleRepo, sampleManager, sampleLoader, minioClient)
		composedAttackHandler = handler.NewComposedAttackHandler(composedAttackRepo, composedAttackManager, composedAttackLoader, minioClient)
	}
	// 模版 Handler 不依赖 MinIO，始终初始化
	tplHandler := handler.NewTemplateHandler(tplRepo)
	// 辅助 LLM 配置 Handler
	auxLLMHandler := handler.NewAuxiliaryLLMHandler(auxLLMRepo)
	// 编排 LLM 和被测 LLM 配置 Handler
	orchLLMHandler := handler.NewOrchestrationLLMHandler(orchLLMRepo)
	targetLLMHandler := handler.NewTargetLLMHandler(targetLLMRepo)
	externalMCPHandler := handler.NewExternalMCPHandler(externalMCPServerRepo, externalMCPToolRepo, externalMCPManager)

	// 检测引擎（即使 sampleManager 为 nil 也可创建，只是样本/评测包功能不可用）
	detEngine := detector.NewEngine(llmClient, sampleLoader, tplRepo)
	detectorHandler = handler.NewAppDetectorHandler(detectorRepo, detEngine)

	// ── 12. 配置路由 ────────────────────────────────────────────
	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 公开路由
	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/register", authHandler.Register)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "ok",
				"time":   fmt.Sprintf("%v", time.Now()),
			})
		})

		// 工具定价表（公开）
		api.GET("/billing/prices", billingHandler.GetToolPrices)

		// 工具 schema 公开查询（动态从 MCP Client 获取，支持 openai | langchain 格式）
		api.GET("/tools/schemas", func(c *gin.Context) {
			format := c.DefaultQuery("format", "openai") // openai | langchain | full
			toolsResult, err := mcpClient.ListTools(c.Request.Context(), mcp.ListToolsRequest{})
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			filteredTools := make([]mcp.Tool, 0, len(toolsResult.Tools))
			for _, t := range toolsResult.Tools {
				if !externalmcp.IsLLMVisibleTool(t.Name, t.Description) {
					continue
				}
				filteredTools = append(filteredTools, t)
			}

			switch format {
			case "langchain":
				tools := make([]map[string]interface{}, 0, len(filteredTools))
				for _, t := range filteredTools {
					tools = append(tools, map[string]interface{}{
						"name":        t.Name,
						"description": t.Description,
						"parameters":  t.InputSchema,
					})
				}
				c.JSON(200, gin.H{"tools": tools})
			default: // openai
				tools := make([]map[string]interface{}, 0, len(filteredTools))
				for _, t := range filteredTools {
					tools = append(tools, map[string]interface{}{
						"type": "function",
						"function": map[string]interface{}{
							"name":        t.Name,
							"description": t.Description,
							"parameters":  t.InputSchema,
						},
					})
				}
				c.JSON(200, gin.H{"tools": tools})
			}
		})

		// SSE 日志流（单独鉴权，支持 ?token= query param）
		api.GET("/assessments/:id/stream", assessHandler.StreamLogs)
	}

	// 需要认证的路由
	auth := api.Group("/")
	auth.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
	auth.Use(middleware.RateLimit(100, time.Minute))
	{
		// 当前用户信息（刷新 localStorage 缓存用）
		auth.GET("/auth/me", authHandler.Me)
		// 评估任务
		auth.POST("/assessments", assessHandler.Create)
		auth.GET("/assessments", assessHandler.List)
		auth.GET("/assessments/:id", assessHandler.Get)
		auth.POST("/assessments/:id/cancel", assessHandler.Cancel)

		// 报告
		auth.GET("/reports/:id", reportHandler.Get)
		auth.GET("/reports/:id/download", reportHandler.Download)

		// 工具列表（认证后可见完整信息）
		auth.GET("/tools", assetHandler.ListTools)
		auth.GET("/assets", assetHandler.ListTools)

		// 计费 & 余额
		auth.GET("/billing/balance", billingHandler.GetBalance)
		auth.GET("/billing/records", billingHandler.GetBillingRecords)
		auth.GET("/billing/transactions", billingHandler.GetTransactions)
		auth.POST("/billing/recharge", billingHandler.Recharge)

		// 企业用户专属
		enterprise := auth.Group("/")
		enterprise.Use(middleware.RequireRole("enterprise", "admin"))
		{
			enterprise.GET("/orchestration-llm/config", orchLLMHandler.GetConfig)
			enterprise.PUT("/orchestration-llm/config", orchLLMHandler.UpdateConfig)
			enterprise.POST("/orchestration-llm/test", orchLLMHandler.TestConnection)

			enterprise.GET("/target-llm/config", targetLLMHandler.GetConfig)
			enterprise.PUT("/target-llm/config", targetLLMHandler.UpdateConfig)
			enterprise.POST("/target-llm/test", targetLLMHandler.TestConnection)
		}

		// 专家/管理员专属
		expert := auth.Group("/")
		expert.Use(middleware.RequireRole("expert", "admin"))
		{
			expert.POST("/tools", assetHandler.UploadTool)
			expert.PUT("/tools/:id/publish", assetHandler.PublishTool)
			expert.PUT("/tools/:id/submit", assetHandler.SubmitForReview)
			expert.PUT("/tools/:id/deprecate", assetHandler.DeprecateTool)
			expert.POST("/assets", assetHandler.UploadTool)

			// 收益明细（专家查看自己的分成记录）
			expert.GET("/billing/earnings", billingHandler.GetExpertEarnings)

			// 工具分类定义
			expert.GET("/tools/categories", handler.GetToolCategories)

			// 攻击样本管理
			if sampleHandler != nil {
				expert.POST("/samples", sampleHandler.Upload)
				expert.GET("/samples", sampleHandler.List)
				expert.DELETE("/samples/:id", sampleHandler.Delete)
				expert.GET("/samples/:id/preview", sampleHandler.Preview)

				expert.POST("/composed-attacks", composedAttackHandler.Upload)
				expert.GET("/composed-attacks", composedAttackHandler.List)
				expert.DELETE("/composed-attacks/:id", composedAttackHandler.Delete)
				expert.GET("/composed-attacks/:id/preview", composedAttackHandler.Preview)
			}

			// 模版管理
			expert.POST("/templates", tplHandler.Create)
			expert.GET("/templates", tplHandler.List)
			expert.PUT("/templates/:id", tplHandler.Update)
			expert.DELETE("/templates/:id", tplHandler.Delete)
			expert.POST("/templates/upload-csv", tplHandler.UploadCSV)

			// 应用检测工具管理
			if detectorHandler != nil {
				expert.POST("/detectors", detectorHandler.Create)
				expert.GET("/detectors", detectorHandler.List)
				expert.GET("/detectors/:id", detectorHandler.Get)
				expert.PUT("/detectors/:id", detectorHandler.Update)
				expert.DELETE("/detectors/:id", detectorHandler.Delete)
				expert.POST("/detectors/:id/run", detectorHandler.Run)
			}

			// 辅助 LLM 配置
			expert.GET("/auxiliary-llm/config", auxLLMHandler.GetConfig)
			expert.PUT("/auxiliary-llm/config", auxLLMHandler.UpdateConfig)
			expert.POST("/auxiliary-llm/test", auxLLMHandler.TestConnection)

			expert.GET("/external-mcp-servers", externalMCPHandler.ListServers)
			expert.POST("/external-mcp-servers", externalMCPHandler.CreateServer)
			expert.PUT("/external-mcp-servers/:id", externalMCPHandler.UpdateServer)
			expert.DELETE("/external-mcp-servers/:id", externalMCPHandler.DeleteServer)
			expert.POST("/external-mcp-servers/:id/test", externalMCPHandler.TestServer)
			expert.POST("/external-mcp-servers/:id/sync", externalMCPHandler.SyncServer)
			expert.GET("/external-mcp-servers/:id/tools", externalMCPHandler.ListTools)
		}

		// 管理员专属路由
		adminGroup := auth.Group("/admin")
		adminGroup.Use(middleware.RequireRole("admin"))
		{
			adminGroup.GET("/users", adminHandler.ListUsers)
			adminGroup.PUT("/users/:id/role", adminHandler.UpdateUserRole)
			adminGroup.PUT("/users/:id/active", adminHandler.SetUserActive)
			adminGroup.GET("/assets/pending", adminHandler.ListPendingAssets)
			adminGroup.PUT("/assets/:id/approve", adminHandler.ApproveAsset)
			adminGroup.PUT("/assets/:id/reject", adminHandler.RejectAsset)
			adminGroup.GET("/stats", adminHandler.GetStats)
		}

		// Chat 会话路由（企业用户 AI 智能服务）
		chat := auth.Group("/chat")
		{
			chat.POST("/sessions", chatHandler.CreateSession)
			chat.GET("/sessions", chatHandler.ListSessions)
			chat.GET("/sessions/:id", chatHandler.GetSession)
			chat.POST("/sessions/:id/messages", chatHandler.SendMessage)
			chat.POST("/sessions/:id/confirm", chatHandler.ConfirmPlan)
			chat.DELETE("/sessions/:id", chatHandler.DeleteSession)
		}
	}

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	logger.Info("server starting", map[string]interface{}{"addr": addr})
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
