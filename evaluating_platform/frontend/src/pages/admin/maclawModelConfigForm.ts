import type { RuntimeAppConfig } from '../../services/admin'

export function isDeepSeekURL(url?: string) {
  return String(url || '').toLowerCase().includes('deepseek.com')
}

export function inferProviderName(url?: string) {
  const lower = String(url || '').toLowerCase()
  if (lower.includes('deepseek')) return 'deepseek-prod'
  if (lower.includes('anthropic')) return 'anthropic-prod'
  if (lower.includes('openai')) return 'openai-prod'
  return 'default'
}

export function normalizeWireAPI(url?: string, wireAPI?: string) {
  const value = String(wireAPI || '').trim() || 'chat_completions'
  if (isDeepSeekURL(url) && value === 'responses') {
    return 'chat_completions'
  }
  return value
}

export function normalizeModel(url?: string, model?: string) {
  const value = String(model || '').trim()
  if (isDeepSeekURL(url) && (value === '' || value.toLowerCase() === 'deepseek-v4')) {
    return 'deepseek-v4-flash'
  }
  return value
}

export function configToForm(cfg?: RuntimeAppConfig) {
  const provider = cfg?.maclaw_llm_providers?.[0] || {}
  const url = provider.url || cfg?.maclaw_llm_url || ''
  return {
    provider_name: provider.name || cfg?.maclaw_llm_current_provider || inferProviderName(url),
    url,
    key: provider.key || '',
    model: normalizeModel(url, provider.model || cfg?.maclaw_llm_model || ''),
    wire_api: normalizeWireAPI(url, provider.wire_api || 'chat_completions'),
    protocol: provider.protocol || '',
    context_length: provider.context_length || cfg?.maclaw_llm_context_length || 0,
    timeout_sec: provider.timeout_sec || cfg?.maclaw_llm_timeout_sec || 60,
    supports_vision: provider.supports_vision || false,
  }
}

export function formToConfig(values: Record<string, any>): RuntimeAppConfig {
  const url = values.url
  const providerName = String(values.provider_name || '').trim() || inferProviderName(url)
  return {
    maclaw_llm_current_provider: providerName,
    maclaw_llm_providers: [{
      name: providerName,
      url,
      key: values.key,
      model: normalizeModel(url, values.model),
      wire_api: normalizeWireAPI(url, values.wire_api),
      protocol: values.protocol,
      context_length: values.context_length,
      timeout_sec: values.timeout_sec,
      supports_vision: values.supports_vision,
    }],
  }
}
