/**
 * LLM 直连请求工具模块
 * 提取自 chat.vue，提供通用的模型直连请求、响应解析等功能
 */

import { parseJsonData } from './configTest'
import { resolveLLMProvider, getProviderQuickUrl } from '../views/admin/forms/llmCatalog'

// ================= 常量 =================
export const CHAT_TIMEOUT_MS = 60000

export const PROVIDER_LABELS = {
  azure: 'Azure OpenAI',
  anthropic: 'Anthropic',
  zhipu: '智谱AI',
  aliyun: '阿里云',
  doubao: '豆包',
  siliconflow: '硅基流动',
  deepseek: 'DeepSeek',
  openai: 'OpenAI',
  ollama: 'Ollama',
  dify: 'Dify',
  coze: 'Coze'
}

// ================= 工具函数 =================

export const joinUrl = (base, path) => {
  const b = String(base || '').trim().replace(/\/+$/, '')
  const p = String(path || '').replace(/^\/+/, '')
  return `${b}/${p}`
}

export const providerLabel = (provider) => PROVIDER_LABELS[provider] || provider || '未知提供商'

export const nowTime = () =>
  new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })

// ================= 配置解析 =================

/**
 * 安全解析单条 LLM 配置，异常时返回 null
 */
export const parseLLMConfigRow = (row) => {
  try {
    if (!row || typeof row !== 'object') return null
    const configId = String(row.config_id || '').trim()
    if (!configId) return null
    const jsonData = parseJsonData(row.json_data)
    const provider = resolveLLMProvider(row.provider, jsonData.type)
    const modelName =
      String(jsonData.model_name || jsonData.model || jsonData.model_id || '').trim() ||
      String(row.name || '').trim() ||
      configId
    const baseUrl = String(jsonData.base_url || '').trim() || getProviderQuickUrl(provider)
    const apiKey = String(jsonData.api_key || '').trim()
    const botId = String(jsonData.bot_id || '').trim()
    const rawMaxTokens = Number(jsonData.max_tokens)
    const maxTokens = Number.isFinite(rawMaxTokens) ? Math.min(100000, Math.max(1, rawMaxTokens)) : 4000
    const rawTemperature = Number(jsonData.temperature)
    const temperature = Number.isFinite(rawTemperature) ? Math.min(2, Math.max(0, rawTemperature)) : 0.7
    const rawTopP = Number(jsonData.top_p)
    const topP = Number.isFinite(rawTopP) ? Math.min(1, Math.max(0, rawTopP)) : 0.9
    return {
      id: row.id,
      config_id: configId,
      name: String(row.name || ''),
      provider,
      type: String(jsonData.type || '').trim(),
      model_name: modelName,
      base_url: baseUrl,
      api_key: apiKey,
      bot_id: botId,
      max_tokens: maxTokens,
      temperature,
      top_p: topP,
      enabled: !!row.enabled,
      is_default: !!row.is_default,
      label: `${modelName}（${providerLabel(provider)}）`
    }
  } catch (err) {
    console.warn('解析 LLM 配置失败，已跳过该配置:', err, row)
    return null
  }
}

// ================= 响应解析 =================

/**
 * 解析模型回复文本，兼容多种响应结构
 */
export const parseModelReply = (data) => {
  const cozeAnswer = Array.isArray(data?.data?.messages)
    ? data.data.messages.find((m) => m?.type === 'answer')?.content || ''
    : ''
  const candidates = [
    data?.choices?.[0]?.message?.content,
    data?.choices?.[0]?.delta?.content,
    data?.answer,
    data?.content,
    data?.reply,
    data?.text,
    data?.output_text,
    cozeAnswer,
    typeof data === 'string' ? data : ''
  ]
  const found = candidates.find((item) => typeof item === 'string' && item.trim())
  return found ? found.trim() : ''
}

export const extractErrorText = (data) => {
  const candidates = [
    data?.error?.message,
    data?.message,
    data?.error_message,
    data?.detail,
    typeof data === 'string' ? data : ''
  ]
  const found = candidates.find((item) => typeof item === 'string' && item.trim())
  return found ? found.trim() : ''
}

// ================= 网络请求 =================

const doFetchJson = async (url, options) => {
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), CHAT_TIMEOUT_MS)
  try {
    const response = await fetch(url, { ...options, signal: controller.signal })
    let data = null
    const text = await response.text()
    if (text) {
      try {
        data = JSON.parse(text)
      } catch {
        data = text
      }
    }
    if (!response.ok) {
      const detail = extractErrorText(data)
      throw new Error(`HTTP ${response.status}${detail ? `：${detail}` : ''}`)
    }
    return data
  } finally {
    clearTimeout(timer)
  }
}

const buildHeaders = (config) => {
  const headers = { 'Content-Type': 'application/json' }
  if (config.api_key) {
    headers.Authorization = `Bearer ${config.api_key}`
  }
  return headers
}

// ================= Provider 发送函数 =================

/**
 * OpenAI 兼容类 provider（openai、ollama、deepseek、zhipu、aliyun、doubao、siliconflow、anthropic、azure 等）
 */
export const sendOpenAICompatible = async (config, msgs) => {
  const url = joinUrl(config.base_url, 'chat/completions')
  const body = {
    model: config.model_name,
    messages: msgs,
    max_tokens: config.max_tokens,
    temperature: config.temperature,
    top_p: config.top_p
  }
  const data = await doFetchJson(url, {
    method: 'POST',
    headers: buildHeaders(config),
    body: JSON.stringify(body)
  })
  return parseModelReply(data)
}

/**
 * Dify：阻塞式 chat-messages 接口
 */
export const sendDify = async (config, msgs) => {
  const url = joinUrl(config.base_url, 'chat-messages')
  const lastUser = [...msgs].reverse().find((m) => m.role === 'user')
  const body = {
    inputs: {},
    query: lastUser?.content || '',
    response_mode: 'blocking',
    conversation_id: '',
    user: 'manager-chat-ui'
  }
  const data = await doFetchJson(url, {
    method: 'POST',
    headers: buildHeaders(config),
    body: JSON.stringify(body)
  })
  return parseModelReply(data)
}

/**
 * Coze：需要 bot_id，非流式 /v3/chat
 */
export const sendCoze = async (config, msgs) => {
  if (!config.bot_id) {
    throw new Error('当前 Coze 配置缺少 bot_id，请先在 LLM 配置中填写 Bot ID')
  }
  const url = joinUrl(config.base_url, 'v3/chat')
  const body = {
    bot_id: config.bot_id,
    user_id: 'manager-chat-ui',
    stream: false,
    auto_save_history: false,
    additional_messages: msgs.map((m) => ({
      role: m.role === 'assistant' ? 'assistant' : 'user',
      content: m.content,
      content_type: 'text'
    }))
  }
  const data = await doFetchJson(url, {
    method: 'POST',
    headers: buildHeaders(config),
    body: JSON.stringify(body)
  })
  return parseModelReply(data)
}

/**
 * 统一入口：根据 provider 自动选择发送方式
 */
export const requestModelDirect = async (config, msgs) => {
  if (!config.base_url) {
    throw new Error('当前配置缺少 base_url，请先在 LLM 配置中填写 Base URL')
  }
  if (config.provider === 'dify') return sendDify(config, msgs)
  if (config.provider === 'coze') return sendCoze(config, msgs)
  return sendOpenAICompatible(config, msgs)
}

/**
 * 格式化请求错误信息
 */
export const formatRequestError = (error) => {
  if (error?.name === 'AbortError') return '请求超时，请稍后重试'
  if (error instanceof TypeError) {
    return `网络错误：${error.message || 'Failed to fetch'}（可能是网络不通或跨域限制）`
  }
  return error?.message || '未知错误'
}
