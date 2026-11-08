import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/stores/auth'

export const useLLMStore = defineStore('llm', () => {
  const configs          = ref([])  // [{provider, api_key (masked), default_model, base_url, enabled, is_default}]
  const profiles         = ref({})  // { fast: { id, provider, model, label, description, badge }, ... }
  const designerProfiles = ref({})  // { frontier, balanced, fast } for AI process designer
  const openRouterCatalog = ref([])
  const catalogLoading    = ref(false)
  const loading  = ref(false)
  const error    = ref(null)

  const PROVIDERS = [
    { id: 'openrouter', label: 'OpenRouter',      category: 'gateway', badge: 'Recommended', needsKey: true,  needsURL: false, description: 'Universal Gateway: access 300+ models (Claude, GPT-4, DeepSeek, Llama) with 1 key' },
    { id: 'anthropic',  label: 'Anthropic',       category: 'cloud',   badge: null,          needsKey: true,  needsURL: false, description: 'Direct Claude API connection' },
    { id: 'ollama',     label: 'Ollama (Local)',  category: 'local',   badge: null,          needsKey: false, needsURL: true,  description: 'Local on-premise model execution' },
    { id: 'openai',     label: 'OpenAI',          category: 'cloud',   badge: null,          needsKey: true,  needsURL: false, description: 'Direct OpenAI GPT-4o / o1 API' },
    { id: 'deepseek',   label: 'DeepSeek',        category: 'cloud',   badge: null,          needsKey: true,  needsURL: false, description: 'Direct DeepSeek API' },
    { id: 'grok',       label: 'Grok (xAI)',      category: 'cloud',   badge: null,          needsKey: true,  needsURL: false, description: 'Direct xAI Grok API' },
    { id: 'gemini',     label: 'Gemini',          category: 'cloud',   badge: null,          needsKey: true,  needsURL: false, description: 'Google Generative Language API' },
    { id: 'groq',       label: 'Groq',            category: 'cloud',   badge: null,          needsKey: true,  needsURL: false, description: 'Ultra-fast LPU inference' },
    { id: 'mistral',    label: 'Mistral AI',      category: 'cloud',   badge: null,          needsKey: true,  needsURL: false, description: 'Direct Mistral AI API' },
    { id: 'together',   label: 'Together AI',     category: 'cloud',   badge: null,          needsKey: true,  needsURL: false, description: 'Cloud open-source inference' },
    { id: 'fireworks',  label: 'Fireworks AI',    category: 'cloud',   badge: null,          needsKey: true,  needsURL: false, description: 'Fast inference platform' },
    { id: 'cohere',     label: 'Cohere',          category: 'cloud',   badge: null,          needsKey: true,  needsURL: false, description: 'Direct Cohere Command API' },
    { id: 'qwen',       label: 'Qwen (Alibaba)',  category: 'cloud',   badge: null,          needsKey: true,  needsURL: false, description: 'Alibaba DashScope API' },
    { id: 'glm',        label: 'GLM (Zhipu)',     category: 'cloud',   badge: null,          needsKey: true,  needsURL: false, description: 'Zhipu BigModel API' },
  ]

  const DEFAULT_MODELS = {
    openrouter: '~deepseek/deepseek-flash-latest',
    anthropic:  'claude-sonnet-4-5',
    openai:     'gpt-4o',
    grok:       'grok-3',
    deepseek:   'deepseek-v4-flash',
    mistral:    'mistral-large-latest',
    groq:       'llama-3.1-70b-versatile',
    together:   'meta-llama/Llama-3-70b-chat-hf',
    fireworks:  'accounts/fireworks/models/llama-v3p1-70b-instruct',
    cohere:     'command-r-plus',
    qwen:       'qwen-plus',
    glm:        'glm-4-plus',
    gemini:     'gemini-2.0-flash',
    ollama:     'llama3.2',
  }

  async function fetchAll() {
    loading.value = true
    error.value   = null
    try {
      const { data } = await api.get('/llm-configs')
      configs.value  = data ?? []
    } catch (e) {
      error.value = e.response?.data?.error ?? e.message
    } finally {
      loading.value = false
    }
  }

  function getConfig(provider) {
    return configs.value.find(c => c.provider === provider)
  }

  async function upsert(provider, body) {
    await api.put(`/llm-configs/${provider}`, body)
    await fetchAll()
  }

  async function remove(provider) {
    await api.delete(`/llm-configs/${provider}`)
    await fetchAll()
  }

  async function setDefault(provider) {
    await api.post('/llm-configs/default', { provider })
    await fetchAll()
  }

  async function testConnection(provider) {
    const { data } = await api.post(`/llm-configs/${provider}/test`)
    return data
  }

  async function fetchProfiles() {
    try {
      const { data } = await api.get('/llm-configs/profiles')
      profiles.value = data ?? {}
      return profiles.value
    } catch (e) {
      console.error('Failed to fetch LLM profiles', e)
      return {}
    }
  }

  async function saveProfiles(updated) {
    const { data } = await api.put('/llm-configs/profiles', updated)
    if (data?.profiles) {
      profiles.value = data.profiles
    }
    return profiles.value
  }

  async function fetchDesignerProfiles() {
    try {
      const { data } = await api.get('/llm-configs/designer-profiles')
      designerProfiles.value = data ?? {}
      return designerProfiles.value
    } catch (e) {
      console.error('Failed to fetch designer profiles', e)
      return {}
    }
  }

  async function saveDesignerProfiles(updated) {
    const { data } = await api.put('/llm-configs/designer-profiles', updated)
    if (data?.profiles) {
      designerProfiles.value = data.profiles
    }
    return designerProfiles.value
  }

  async function fetchOpenRouterModels(force = false) {
    if (openRouterCatalog.value.length > 0 && !force) {
      return openRouterCatalog.value
    }
    catalogLoading.value = true
    try {
      const { data } = await api.get('/llm-configs/openrouter/models')
      openRouterCatalog.value = data ?? []
      return openRouterCatalog.value
    } catch (e) {
      console.error('Failed to fetch OpenRouter catalog', e)
      return []
    } finally {
      catalogLoading.value = false
    }
  }

  return {
    configs, profiles, designerProfiles, openRouterCatalog, catalogLoading, loading, error,
    PROVIDERS, DEFAULT_MODELS,
    fetchAll, getConfig, upsert, remove, setDefault, testConnection,
    fetchProfiles, saveProfiles, fetchDesignerProfiles, saveDesignerProfiles, fetchOpenRouterModels,
  }
})
