package handler

import (
	"fmt"
	"strings"
)

const xfyunCodingPlanOpenAIBase = "https://maas-coding-api.cn-huabei-1.xf-yun.com/v2"
const xfyunCodingPlanModel = "astron-code-latest"

func normalizeLLMBaseURL(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func validateOpenAICompatibleProviderConfig(baseURL, model string) error {
	normalizedBaseURL := strings.ToLower(normalizeLLMBaseURL(baseURL))
	normalizedModel := strings.TrimSpace(model)

	if normalizedBaseURL == strings.ToLower(xfyunCodingPlanOpenAIBase) && normalizedModel != xfyunCodingPlanModel {
		return fmt.Errorf("讯飞 Coding Plan 的 OpenAI 协议要求 model 固定为 %s", xfyunCodingPlanModel)
	}

	return nil
}
