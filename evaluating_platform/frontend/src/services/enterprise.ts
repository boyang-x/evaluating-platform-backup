import api from './api'

export const enterpriseService = {
  // 编排 LLM 配置
  getOrchLLMConfig: () => api.get('/orchestration-llm/config').then(r => r.data),
  updateOrchLLMConfig: (data: { base_url: string; api_key: string; model?: string }) =>
    api.put('/orchestration-llm/config', data).then(r => r.data),
  testOrchLLMConnection: (data: { base_url: string; api_key: string; model?: string }) =>
    api.post('/orchestration-llm/test', data).then(r => r.data),

  // 被测 LLM 配置
  getTargetLLMConfig: () => api.get('/target-llm/config').then(r => r.data),
  updateTargetLLMConfig: (data: {
    base_url: string; api_key: string; model?: string; connector_type?: string
  }) => api.put('/target-llm/config', data).then(r => r.data),
  testTargetLLMConnection: (data: {
    base_url: string; api_key: string; model?: string; connector_type?: string
  }) => api.post('/target-llm/test', data).then(r => r.data),
}
