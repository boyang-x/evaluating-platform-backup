package billing

// ToolPrices 各工具定价（元/次调用）
var ToolPrices = map[string]float64{
	"prompt_injection":     0.50,
	"jailbreak":            0.30,
	"compliance_check":     0.40,
	"compliance_evaluator": 1.00,
	"goal_hijacking":       0.60,
	"tool_poisoning":       0.60,
	"report_generator":     2.00,
}

// ToolCategories 工具分类
var ToolCategories = map[string]string{
	"prompt_injection":     "llm_security",
	"jailbreak":            "llm_security",
	"compliance_check":     "compliance",
	"compliance_evaluator": "compliance",
	"goal_hijacking":       "agent_security",
	"tool_poisoning":       "agent_security",
	"report_generator":     "reporting",
}

const (
	// DefaultToolPrice 未登记工具的兜底价格（元/次）
	DefaultToolPrice = 0.20
	// MinAssessmentBalance 发起评估所需的最低余额（元）
	MinAssessmentBalance = 5.00
	// TokenCostPer1K 每 1000 tokens 费用（元）
	TokenCostPer1K = 0.01
	// ExpertShareRatio 专家工具被调用时的收益分成比例
	ExpertShareRatio = 0.30
)

// ToolPrice 返回指定工具的单次调用价格
func ToolPrice(toolName string) float64 {
	if p, ok := ToolPrices[toolName]; ok {
		return p
	}
	return DefaultToolPrice
}

// ToolCategory 返回工具分类
func ToolCategory(toolName string) string {
	if c, ok := ToolCategories[toolName]; ok {
		return c
	}
	return "unknown"
}

// CalculateToolCost 计算单次工具调用费用（工具价 + token 费）
func CalculateToolCost(toolName string, tokensUsed int) float64 {
	toolCost := ToolPrice(toolName)
	tokenCost := float64(tokensUsed) / 1000.0 * TokenCostPer1K
	return toolCost + tokenCost
}

// CalculateTokenCost 仅计算 token 费用（不含工具价）
func CalculateTokenCost(tokensUsed int) float64 {
	return float64(tokensUsed) / 1000.0 * TokenCostPer1K
}
