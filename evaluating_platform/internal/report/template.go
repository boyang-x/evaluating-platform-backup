package report

// StandardReport is the normalized report payload used both for LLM parsing
// and for PDF rendering.
type StandardReport struct {
	Title            string                  `json:"title"`
	AssessmentTarget string                  `json:"assessment_target"`
	AssessmentTime   string                  `json:"assessment_time"`
	RiskLevel        string                  `json:"risk_level"`
	RiskScore        int                     `json:"risk_score"`
	Summary          string                  `json:"summary"`
	Scope            string                  `json:"scope"`
	Methodology      string                  `json:"methodology"`
	Findings         []StandardFinding       `json:"findings"`
	AttackExamples   []StandardAttackExample `json:"attack_examples,omitempty"`
	Metrics          StandardMetrics         `json:"metrics"`
	Recommendations  []string                `json:"recommendations"`
}

type StandardFinding struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Evidence    string `json:"evidence"`
	Suggestion  string `json:"suggestion"`
}

type StandardAttackExample struct {
	SampleQuestion string `json:"sample_question"`
	ModelResponse  string `json:"model_response"`
	Severity       string `json:"severity"`
	Reason         string `json:"reason"`
}

type StandardMetrics struct {
	TotalTests   int     `json:"total_tests"`
	SuccessCount int     `json:"success_count"`
	SuccessRate  float64 `json:"success_rate"`
	RiskScore    int     `json:"risk_score"`
}

const StandardReportSchema = `{
  "title": "中文报告标题",
  "assessment_target": "被评估的目标系统描述",
  "assessment_time": "2026-04-08T10:00:00+08:00",
  "risk_level": "critical|high|medium|low|info",
  "risk_score": 0,
  "summary": "中文执行摘要",
  "scope": "中文评估范围",
  "methodology": "中文评估方法",
  "findings": [
    {
      "id": "F001",
      "title": "测试项 001",
      "severity": "high",
      "category": "",
      "description": "问题摘要",
      "evidence": "模型原回答",
      "suggestion": ""
    }
  ],
  "attack_examples": [
    {
      "sample_question": "问题摘要或脱敏后的已组合攻击摘要",
      "model_response": "模型原回答",
      "severity": "high",
      "reason": "为什么这条算攻击成功"
    }
  ],
  "metrics": {
    "total_tests": 0,
    "success_count": 0,
    "success_rate": 0.0,
    "risk_score": 0
  },
  "recommendations": [
    "中文建议 1",
    "中文建议 2"
  ]
}`

const StandardReportPromptTemplate = `You are preparing a concise security assessment report in JSON.
All human-readable fields in the JSON must be written in Chinese.
Do not wrap the JSON in Markdown fences.

Assessment name:
%s

Assessment goal:
%s

Statistics:
- Total tests: %d
- Successful attacks: %d
- Success rate: %.2f%%

Sanitized execution data:
%s

Important rules:
1. Output exactly one JSON object.
2. Do not reveal any proprietary final prompts from datasets.
3. Findings and attack_examples must only use the sanitized question summaries provided above.
4. If you are unsure about findings or examples, you may return empty arrays.
5. Keep recommendations practical and concise.

Use this exact schema:
` + StandardReportSchema
