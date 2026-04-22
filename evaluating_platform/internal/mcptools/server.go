package mcptools

import (
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"evaluating_platform/internal/repository"
	"evaluating_platform/internal/sample"
	skillpkg "evaluating_platform/internal/skill"
	"evaluating_platform/pkg/storage"
)

// NewMCPServer creates and registers all MCP tools used by the platform.
func NewMCPServer(
	loader *sample.Loader,
	composedLoader *sample.ComposedAttackLoader,
	tplRepo *repository.TemplateRepository,
	sampleRepo *repository.AttackSampleRepository,
	composedRepo *repository.ComposedAttackRepository,
	auxLLMRepo *repository.AuxiliaryLLMRepository,
	targetLLMRepo *repository.TargetLLMRepository,
	minioClient *storage.MinIOClient,
	reportRepo *repository.ReportRepository,
	sessionStore *SessionStore,
	skillService *skillpkg.Service,
) *server.MCPServer {
	s := server.NewMCPServer("ai-security-tools", "1.0.0")
	if sessionStore == nil {
		sessionStore = NewSessionStore(30 * time.Minute)
	}

	if sampleRepo != nil {
		registerSampleQueryTools(s, sampleRepo, loader)
	}
	if composedRepo != nil {
		registerComposedAttackQueryTools(s, composedRepo, composedLoader)
	}
	if tplRepo != nil {
		registerListTemplates(s, tplRepo)
		registerGetTemplate(s, tplRepo)
	}
	if sampleRepo != nil && tplRepo != nil {
		registerRecommendResources(s, sampleRepo, composedRepo, tplRepo, skillService)
	}

	if tplRepo != nil && loader != nil {
		registerCombineTemplateSample(s, tplRepo, loader, sessionStore)
	}
	if composedRepo != nil && composedLoader != nil {
		registerLoadComposedAttack(s, composedLoader, sessionStore)
	}
	if skillService != nil {
		registerPreviewSkill(s, skillService)
		registerRunGeneratorSkill(s, skillService, loader, sessionStore)
	}
	if auxLLMRepo != nil {
		registerEnhancePayloads(s, auxLLMRepo, sessionStore)
	}
	if targetLLMRepo != nil {
		registerExecutePayloads(s, targetLLMRepo, sessionStore)
	}
	if auxLLMRepo != nil {
		registerGenerateReport(s, auxLLMRepo, sessionStore, minioClient, reportRepo)
	}

	return s
}

// StartSSEServer starts the MCP SSE server.
func StartSSEServer(mcpServer *server.MCPServer, port int) error {
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	sseServer := server.NewSSEServer(mcpServer, server.WithBaseURL(baseURL))
	return sseServer.Start(fmt.Sprintf(":%d", port))
}
