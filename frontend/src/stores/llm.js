import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/stores/auth'

export const useLLMStore = defineStore('llm', () => {
  const configs  = ref([])  // [{provider, api_key (masked), default_model, base_url, enabled, is_default}]
  const loading  = ref(false)
  const error    = ref(null)

  const PROVIDERS = [
    { id: 'anthropic', label: 'Anthropic',    needsKey: true,  needsURL: false },
    { id: 'openai',    label: 'OpenAI',        needsKey: true,  needsURL: false },
    { id: 'grok',      label: 'Grok (xAI)',    needsKey: true,  needsURL: false },
    { id: 'deepseek',  label: 'Deepseek',      needsKey: true,  needsURL: false },
    { id: 'gemini',    label: 'Gemini',         needsKey: true,  needsURL: false },
    { id: 'mistral',   label: 'Mistral AI',    needsKey: true,  needsURL: false },
    { id: 'groq',      label: 'Groq',          needsKey: true,  needsURL: false },
    { id: 'together',  label: 'Together AI',   needsKey: true,  needsURL: false },
    { id: 'fireworks', label: 'Fireworks AI',  needsKey: true,  needsURL: false },
    { id: 'cohere',    label: 'Cohere',        needsKey: true,  needsURL: false },
    { id: 'qwen',      label: 'Qwen (Alibaba)',needsKey: true,  needsURL: false },
    { id: 'glm',       label: 'GLM (Zhipu)',   needsKey: true,  needsURL: false },
    { id: 'ollama',    label: 'Ollama (local)', needsKey: false, needsURL: true  },
  ]

  const DEFAULT_MODELS = {
    anthropic: 'claude-sonnet-4-5',
    openai:    'gpt-4o',
    grok:      'grok-3',
    deepseek:  'deepseek-chat',
    mistral:   'mistral-large-latest',
    groq:      'llama-3.1-70b-versatile',
    together:  'meta-llama/Llama-3-70b-chat-hf',
    fireworks: 'accounts/fireworks/models/llama-v3p1-70b-instruct',
    cohere:    'command-r-plus',
    qwen:      'qwen-plus',
    glm:       'glm-4-plus',
    gemini:    'gemini-2.0-flash',
    ollama:    'llama3.2',
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

  return {
    configs, loading, error,
    PROVIDERS, DEFAULT_MODELS,
    fetchAll, getConfig, upsert, remove, setDefault, testConnection,
  }
})
