package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetToolCategories 返回四大工具分类及子类型定义
// GET /api/v1/tools/categories
func GetToolCategories(c *gin.Context) {
	categories := []map[string]interface{}{
		{
			"key":         "sample_tool",
			"label":       "样本工具",
			"description": "各类攻击载荷样本集，CSV 格式加密存储",
			"sub_types": []map[string]string{
				{"key": "direct_injection", "label": "直接注入样本"},
				{"key": "malicious_instruction", "label": "恶意指令样本"},
				{"key": "compliance_detection", "label": "合规检测样本"},
				{"key": "malicious_poisoning", "label": "恶意投毒样本"},
			},
		},
		{
			"key":         "template_tool",
			"label":       "模版工具",
			"description": "攻击模版，支持变量替换和组合使用",
			"sub_types": []map[string]string{
				{"key": "role_play", "label": "角色扮演模版"},
				{"key": "multilingual", "label": "多语言模版"},
				{"key": "encoding_evasion", "label": "编码加密模版"},
			},
		},
		{
			"key":         "combo_tool",
			"label":       "组合工具",
			"description": "将样本和模版打包为评测包，定义执行顺序",
			"sub_types":   []map[string]string{},
		},
		{
			"key":         "app_detector",
			"label":       "应用检测工具",
			"description": "对接目标应用，加载攻击素材执行检测，LLM 智能判定结果",
			"sub_types": []map[string]string{
				{"key": "llm_detector", "label": "LLM 检测工具"},
				{"key": "dify_detector", "label": "Dify 智能体检测工具"},
				{"key": "custom_detector", "label": "自定义 Agent 检测工具"},
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}
