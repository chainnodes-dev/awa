<template>
  <div class="flex flex-col h-full overflow-y-auto bg-surface-0">

    <header class="px-6 py-4 border-b border-border shrink-0 bg-surface-1">
      <h1 class="text-base font-semibold text-slate-200">Settings</h1>
      <p class="text-xs text-slate-500 mt-0.5">Configure LLM providers and global defaults</p>
    </header>

    <div class="flex-1 px-6 py-6 max-w-3xl w-full mx-auto space-y-6">

      <!-- Non-admin notice -->
      <div v-if="!isAdmin" class="px-4 py-3 rounded-lg bg-amber-500/10 border border-amber-500/30 text-amber-400 text-sm">
        LLM configuration requires admin privileges.
      </div>

      <template v-if="isAdmin">
        <!-- Global default -->
        <section class="card p-5">
          <h2 class="text-sm font-semibold text-slate-300 mb-3">Global Default Provider</h2>
          <div class="flex items-center gap-3">
            <select v-model="selectedDefault" class="input text-sm flex-1"
                    @change="saveDefault">
              <option value="">— none —</option>
              <option v-for="p in enabledProviders" :key="p.id" :value="p.id">
                {{ p.label }}
              </option>
            </select>
            <span class="text-xs text-slate-500">Used when a workflow state does not specify a provider</span>
          </div>
        </section>

        <!-- Per-provider cards -->
        <section v-for="prov in llmStore.PROVIDERS" :key="prov.id" class="card p-5 space-y-4">
          <div class="flex items-center gap-3">
            <div class="flex-1">
              <h2 class="text-sm font-semibold text-slate-300">{{ prov.label }}</h2>
              <p class="text-xs text-slate-500 mt-0.5">{{ providerEndpoint(prov.id) || 'Local server' }}</p>
            </div>
            <!-- Enable toggle -->
            <label class="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" v-model="forms[prov.id].enabled" class="rounded" />
              <span class="text-xs text-slate-400">Enabled</span>
            </label>
            <!-- Default badge -->
            <span v-if="isDefaultProvider(prov.id)"
                  class="text-xs px-2 py-0.5 rounded-full bg-accent/20 text-accent font-medium">default</span>
          </div>

          <div class="grid grid-cols-2 gap-3">
            <!-- API Key (cloud providers) -->
            <div v-if="prov.needsKey" class="col-span-2">
              <label class="text-xs text-slate-500 mb-1 block">API Key</label>
              <input
                v-model="forms[prov.id].api_key"
                type="password"
                :placeholder="hasExistingKey(prov.id) ? 'Configured (leave blank to keep)' : 'Enter API key'"
                class="input text-sm w-full font-mono"
              />
            </div>

            <!-- Base URL (Ollama) -->
            <div v-if="prov.needsURL" class="col-span-2">
              <label class="text-xs text-slate-500 mb-1 block">Server URL</label>
              <input
                v-model="forms[prov.id].base_url"
                type="text"
                placeholder="http://mintbox:11434"
                class="input text-sm w-full font-mono"
              />
            </div>

            <!-- Default model -->
            <div>
              <label class="text-xs text-slate-500 mb-1 block">Default Model</label>
              <input
                v-model="forms[prov.id].default_model"
                type="text"
                :placeholder="llmStore.DEFAULT_MODELS[prov.id]"
                class="input text-sm w-full"
              />
            </div>

            <!-- Max Tokens -->
            <div>
              <label class="text-xs text-slate-500 mb-1 block">Max Output Tokens</label>
              <input
                v-model.number="forms[prov.id].max_output_tokens"
                type="number"
                placeholder="4096"
                class="input text-sm w-full"
              />
            </div>
          </div>

          <!-- Status message -->
          <div v-if="statuses[prov.id]"
               :class="['text-xs px-3 py-2 rounded', statuses[prov.id].ok
                 ? 'bg-green-500/10 text-green-400'
                 : 'bg-red-500/10 text-red-400']">
            {{ statuses[prov.id].ok ? '✓ Connected' : '✗ ' + statuses[prov.id].error }}
          </div>

          <!-- Actions -->
          <div class="flex gap-2 pt-1">
            <button
              @click="save(prov.id)"
              :disabled="saving[prov.id]"
              class="btn btn-primary text-xs px-3 py-1.5"
            >
              {{ saving[prov.id] ? 'Saving…' : 'Save' }}
            </button>
            <button
              @click="test(prov.id)"
              :disabled="testing[prov.id]"
              class="btn-ghost text-xs px-3 py-1.5"
            >
              {{ testing[prov.id] ? 'Testing…' : 'Test Connection' }}
            </button>
          </div>
        </section>

        <!-- System Prompts -->
        <section class="card p-5 space-y-4">
          <div class="flex items-center gap-2 mb-2">
            <svg class="text-violet-400" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
            </svg>
            <h2 class="text-sm font-semibold text-slate-300">AI Designer System Prompts</h2>
          </div>
          <p class="text-xs text-slate-500">
            Customize the underlying instructions used by the AI to generate and refine workflows.
            Overrides are stored in-memory and will reset on server restart.
          </p>

          <div v-for="(content, id) in systemPrompts" :key="id" class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-[10px] font-mono text-slate-400 uppercase tracking-wider">{{ id.replace(/_/g, ' ') }}</span>
              <button @click="savePrompt(id)" :disabled="savingPrompt[id]" class="text-[10px] text-violet-400 hover:text-violet-300 font-bold uppercase transition-colors">
                {{ savingPrompt[id] ? 'Saving...' : 'Save Override' }}
              </button>
            </div>
            <textarea
              v-model="systemPrompts[id]"
              class="input w-full h-32 font-mono text-[11px] leading-relaxed resize-y"
              placeholder="Enter custom system prompt..."
            />
          </div>
        </section>
      </template>

    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import { useAuthStore, api } from '@/stores/auth'
import { useLLMStore } from '@/stores/llm'

const authStore = useAuthStore()
const llmStore  = useLLMStore()

const isAdmin = computed(() => authStore.user?.role === 'admin' || authStore.user?.role === 'super_admin')

// Build a reactive form object per provider
const forms = reactive({})
llmStore.PROVIDERS.forEach(p => {
  forms[p.id] = { api_key: '', base_url: '', default_model: '', max_output_tokens: 4096, enabled: false }
})

const saving  = reactive({})
const testing = reactive({})
const statuses = reactive({})

const selectedDefault = ref('')

const enabledProviders = computed(() =>
  llmStore.PROVIDERS.filter(p => forms[p.id]?.enabled)
)

// ...

function isDefaultProvider(id) {
  return llmStore.configs.find(c => c.provider === id)?.is_default === true
}

function hasExistingKey(id) {
  return llmStore.configs.find(c => c.provider === id)?.api_key === '***'
}

function providerEndpoint(id) {
  const map = {
    openai:    'api.openai.com',
    grok:      'api.x.ai',
    deepseek:  'api.deepseek.com',
    gemini:    'generativelanguage.googleapis.com',
    anthropic: 'api.anthropic.com',
  }
  return map[id] ?? ''
}

function syncFormsFromStore() {
  llmStore.PROVIDERS.forEach(p => {
    const cfg = llmStore.configs.find(c => c.provider === p.id)
    if (cfg) {
      forms[p.id].enabled            = cfg.enabled
      forms[p.id].default_model      = cfg.default_model
      forms[p.id].max_output_tokens  = cfg.max_output_tokens || 4096
      forms[p.id].base_url           = cfg.base_url ?? ''
      forms[p.id].api_key            = ''  // never pre-fill masked keys
    } else {
      forms[p.id].enabled       = false
      forms[p.id].default_model = ''
      forms[p.id].base_url      = ''
      forms[p.id].api_key       = ''
    }
  })
  const def = llmStore.configs.find(c => c.is_default)
  selectedDefault.value = def?.provider ?? ''
}

async function save(provider) {
  saving[provider] = true
  statuses[provider] = null
  try {
    await llmStore.upsert(provider, {
      api_key:           forms[provider].api_key,
      base_url:          forms[provider].base_url,
      default_model:     forms[provider].default_model,
      max_output_tokens: forms[provider].max_output_tokens,
      enabled:           forms[provider].enabled,
    })
    syncFormsFromStore()
  } catch (e) {
    statuses[provider] = { ok: false, error: e.response?.data?.error ?? e.message }
  } finally {
    saving[provider] = false
  }
}

async function test(provider) {
  testing[provider] = true
  statuses[provider] = null
  try {
    const result = await llmStore.testConnection(provider)
    statuses[provider] = result
  } catch (e) {
    statuses[provider] = { ok: false, error: e.response?.data?.error ?? e.message }
  } finally {
    testing[provider] = false
  }
}

async function saveDefault() {
  if (!selectedDefault.value) return
  try {
    await llmStore.setDefault(selectedDefault.value)
  } catch (e) {
    console.error('Failed to set default provider', e)
  }
}

// -- System Prompts --
const systemPrompts = reactive({
  workflow_generator_base: '',
  skill_analyser_preamble: '',
  workflow_refinement_addendum: '',
})
const savingPrompt = reactive({})

async function fetchPrompts() {
  try {
    const { data } = await api.get('/designer/prompts')
    Object.assign(systemPrompts, data)
  } catch (e) {
    console.error('Failed to fetch system prompts', e)
  }
}

async function savePrompt(id) {
  savingPrompt[id] = true
  try {
    await api.put(`/designer/prompts/${id}`, { content: systemPrompts[id] })
  } catch (e) {
    alert('Failed to save prompt: ' + (e.response?.data?.error ?? e.message))
  } finally {
    savingPrompt[id] = false
  }
}

onMounted(async () => {
  if (isAdmin.value) {
    await llmStore.fetchAll()
    syncFormsFromStore()
    await fetchPrompts()
  }
})
</script>
