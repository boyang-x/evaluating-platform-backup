import api from './api'

export const expertService = {
  getCategories: () => api.get('/tools/categories').then((r) => r.data),

  uploadSample: (data: FormData) =>
    api.post('/samples', data, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }).then((r) => r.data),

  listSamples: (params?: { sub_type?: string; limit?: number; offset?: number }) =>
    api.get('/samples', { params }).then((r) => r.data),

  deleteSample: (id: string) => api.delete(`/samples/${id}`).then((r) => r.data),

  previewSample: (id: string, limit?: number) =>
    api.get(`/samples/${id}/preview`, { params: { limit } }).then((r) => r.data),

  uploadComposedAttack: (data: FormData) =>
    api.post('/composed-attacks', data, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }).then((r) => r.data),

  listComposedAttacks: (params?: { sub_type?: string; limit?: number; offset?: number }) =>
    api.get('/composed-attacks', { params }).then((r) => r.data),

  deleteComposedAttack: (id: string) => api.delete(`/composed-attacks/${id}`).then((r) => r.data),

  previewComposedAttack: (id: string, limit?: number) =>
    api.get(`/composed-attacks/${id}/preview`, { params: { limit } }).then((r) => r.data),

  listExternalMCPServers: () =>
    api.get('/external-mcp-servers').then((r) => r.data),

  createExternalMCPServer: (data: {
    name: string
    namespace?: string
    description?: string
    base_url: string
    transport_type?: string
    auth_type?: string
    auth_key?: string
    auth_header?: string
    auth_prefix?: string
    upstream_base_url?: string
    upstream_api_key?: string
    upstream_model?: string
    upstream_timeout_seconds?: number
    timeout_seconds?: number
    enabled?: boolean
    skill_prompt?: string
  }) => api.post('/external-mcp-servers', data).then((r) => r.data),

  updateExternalMCPServer: (id: string, data: {
    name: string
    namespace?: string
    description?: string
    base_url: string
    transport_type?: string
    auth_type?: string
    auth_key?: string
    auth_header?: string
    auth_prefix?: string
    upstream_base_url?: string
    upstream_api_key?: string
    upstream_model?: string
    upstream_timeout_seconds?: number
    timeout_seconds?: number
    enabled?: boolean
    skill_prompt?: string
  }) => api.put(`/external-mcp-servers/${id}`, data).then((r) => r.data),

  deleteExternalMCPServer: (id: string) =>
    api.delete(`/external-mcp-servers/${id}`).then((r) => r.data),

  testExternalMCPServer: (id: string) =>
    api.post(`/external-mcp-servers/${id}/test`).then((r) => r.data),

  syncExternalMCPServer: (id: string) =>
    api.post(`/external-mcp-servers/${id}/sync`).then((r) => r.data),

  listExternalMCPTools: (id: string) =>
    api.get(`/external-mcp-servers/${id}/tools`).then((r) => r.data),

  createTemplate: (data: {
    sub_type: string
    name: string
    description?: string
    content: string
    variables?: { name: string; description: string; required: boolean; default?: string }[]
  }) => api.post('/templates', data).then((r) => r.data),

  listTemplates: (params?: { sub_type?: string; limit?: number; offset?: number }) =>
    api.get('/templates', { params }).then((r) => r.data),

  updateTemplate: (id: string, data: {
    sub_type: string
    name: string
    description?: string
    content: string
    variables?: { name: string; description: string; required: boolean; default?: string }[]
  }) => api.put(`/templates/${id}`, data).then((r) => r.data),

  deleteTemplate: (id: string) => api.delete(`/templates/${id}`).then((r) => r.data),

  uploadTemplateCSV: (data: FormData) =>
    api.post('/templates/upload-csv', data, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }).then((r) => r.data),

  listEvalPackages: (params?: { limit?: number; offset?: number }) =>
    api.get('/eval-packages', { params }).then((r) => r.data),

  getAuxLLMConfig: () => api.get('/auxiliary-llm/config').then((r) => r.data),

  updateAuxLLMConfig: (data: { base_url: string; api_key: string; model?: string }) =>
    api.put('/auxiliary-llm/config', data).then((r) => r.data),

  testAuxLLMConnection: (data: { base_url: string; api_key: string; model?: string }) =>
    api.post('/auxiliary-llm/test', data).then((r) => r.data),

  createDetector: (data: {
    name: string
    sub_type: string
    description?: string
    target_config: { type: string; base_url: string; api_key: string; model?: string; app_id?: string }
    sample_ids?: string[]
    template_ids?: string[]
    package_ids?: string[]
  }) => api.post('/detectors', data).then((r) => r.data),

  listDetectors: (params?: { limit?: number; offset?: number }) =>
    api.get('/detectors', { params }).then((r) => r.data),

  getDetector: (id: string) => api.get(`/detectors/${id}`).then((r) => r.data),

  updateDetector: (id: string, data: {
    name: string
    sub_type: string
    description?: string
    target_config: { type: string; base_url: string; api_key: string; model?: string; app_id?: string }
    sample_ids?: string[]
    template_ids?: string[]
    package_ids?: string[]
  }) => api.put(`/detectors/${id}`, data).then((r) => r.data),

  deleteDetector: (id: string) => api.delete(`/detectors/${id}`).then((r) => r.data),

  runDetector: (id: string) => api.post(`/detectors/${id}/run`).then((r) => r.data),
}
