package mcptools

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
	"evaluating_platform/internal/sample"
)

// registerSampleQueryTools 注册攻击样本查询工具（不依赖 assessment_id 和 ConnectorPool）
func registerSampleQueryTools(s *server.MCPServer, sampleRepo *repository.AttackSampleRepository, loader *sample.Loader) {
	// list_attack_samples - 列出攻击样本
	listTool := mcp.NewTool("list_attack_samples",
		mcp.WithDescription("列出攻击样本资源卡片，返回适用评估类型、标签和规划摘要，帮助编排 LLM 选择资源"),
		mcp.WithString("expert_id",
			mcp.Description("专家用户 ID（可选，为空时返回已发布的公开样本）"),
		),
		mcp.WithString("sub_type",
			mcp.Description("按子类型过滤（可选）"),
		),
	)
	s.AddTool(listTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		expertIDStr, _ := req.GetArguments()["expert_id"].(string)
		subType, _ := req.GetArguments()["sub_type"].(string)

		var samples []model.AttackSample
		var err error

		if expertIDStr == "" {
			// 无 expert_id：回退查询已发布的公开样本
			samples, _, err = sampleRepo.ListPublished(ctx, subType, 10000, 0)
		} else {
			// 有 expert_id：查询该专家的样本
			expertID, parseErr := uuid.Parse(expertIDStr)
			if parseErr != nil {
				return mcp.NewToolResultError("invalid expert_id"), nil
			}
			samples, _, err = sampleRepo.ListByExpert(ctx, expertID, subType, 10000, 0)
		}

		if err != nil {
			return mcp.NewToolResultError("query samples failed: " + err.Error()), nil
		}

		items := make([]resourceCard, 0, len(samples))
		for _, s := range samples {
			items = append(items, buildSampleCard(s, nil))
		}

		b, _ := json.Marshal(items)
		return mcp.NewToolResultText(string(b)), nil
	})

	// get_attack_sample - 获取样本详情
	getTool := mcp.NewTool("get_attack_sample",
		mcp.WithDescription("根据样本 ID 获取攻击样本的元数据详情；编排决策优先使用 preview_attack_sample"),
		mcp.WithString("sample_id",
			mcp.Required(),
			mcp.Description("攻击样本 ID"),
		),
	)
	s.AddTool(getTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sampleIDStr, _ := req.GetArguments()["sample_id"].(string)

		sampleID, err := uuid.Parse(sampleIDStr)
		if err != nil {
			return mcp.NewToolResultError("invalid sample_id"), nil
		}

		s, err := sampleRepo.GetByID(ctx, sampleID)
		if err != nil {
			return mcp.NewToolResultError("sample not found: " + sampleIDStr), nil
		}

		type sampleDetail struct {
			ID          uuid.UUID `json:"id"`
			Name        string    `json:"name"`
			SubType     string    `json:"sub_type"`
			Description string    `json:"description"`
			SampleCount int       `json:"sample_count"`
		}

		detail := sampleDetail{
			ID:          s.ID,
			Name:        s.Name,
			SubType:     s.SubType,
			Description: s.Description,
			SampleCount: s.SampleCount,
		}

		b, _ := json.Marshal(detail)
		return mcp.NewToolResultText(string(b)), nil
	})

	registerPreviewAttackSample(s, sampleRepo, loader)
}
