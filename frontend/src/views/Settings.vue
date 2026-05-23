<template>
  <div class="flex flex-col h-full overflow-y-auto bg-surface-0">

      <header class="px-6 py-4 border-b border-border shrink-0 bg-surface-1 flex items-center gap-6">
        <div class="flex flex-col">
          <h1 class="text-base font-bold text-text tracking-tight uppercase">Settings</h1>
          <p class="text-[10px] text-text-muted font-bold uppercase tracking-widest opacity-60">Platform Identity & AI Designer</p>
        </div>

        <div class="flex-1">
          <!-- Tab Switcher -->
          <div class="flex p-1 bg-surface-2 rounded-xl border border-border w-fit">
            <button @click="activeTab = 'general'" 
                    :class="['px-6 py-1.5 text-[11px] font-bold uppercase rounded-lg transition-all', 
                             activeTab === 'general' ? 'bg-accent text-white shadow-lg shadow-indigo-500/20' : 'text-text-muted hover:text-text']"
                    style="background-color: activeTab === 'general' ? 'var(--color-accent)' : 'transparent'">
              General
            </button>
            <button @click="activeTab = 'secrets'" 
                    :class="['px-6 py-1.5 text-[11px] font-bold uppercase rounded-lg transition-all', 
                             activeTab === 'secrets' ? 'bg-accent text-white shadow-lg shadow-indigo-500/20' : 'text-text-muted hover:text-text']"
                    style="background-color: activeTab === 'secrets' ? 'var(--color-accent)' : 'transparent'">
              Secrets
            </button>
            <button @click="activeTab = 'prompts'" 
                    :class="['px-6 py-1.5 text-[11px] font-bold uppercase rounded-lg transition-all', 
                             activeTab === 'prompts' ? 'bg-accent text-white shadow-lg shadow-indigo-500/20' : 'text-text-muted hover:text-text']"
                    style="background-color: activeTab === 'prompts' ? 'var(--color-accent)' : 'transparent'">
              System Prompts
            </button>
          </div>
        </div>

        <div v-if="entStore.status" class="flex items-center gap-2">
           <span class="text-[10px] font-bold uppercase tracking-widest text-text-muted">Tier</span>
           <span :class="tierBadgeClass(entStore.status.tier)">{{ entStore.status.tier }}</span>
        </div>
      </header>

      <div v-if="activeTab === 'general'" class="flex-1 px-6 py-6 max-w-3xl w-full mx-auto space-y-6">
        <!-- Appearance (Theme Switcher) -->
        <section class="space-y-4">
           <div class="flex items-center gap-2 px-1">
             <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-indigo-400">
               <circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
             </svg>
             <h2 class="text-sm font-bold text-text uppercase tracking-wider">Platform Appearance</h2>
           </div>

           <div class="card p-5">
              <div class="flex items-center gap-4">
                 <button 
                   v-for="mode in ['light', 'dark', 'system']" 
                   :key="mode"
                   @click="setTheme(mode)"
                   :class="[
                     'flex-1 flex flex-col items-center gap-2 p-3 rounded-lg border transition-all',
                     theme === mode ? 'border-accent bg-accent/5 text-accent' : 'border-border hover:bg-text/5 text-text-muted'
                   ]"
                 >
                   <span class="text-[10px] font-bold uppercase tracking-widest">{{ mode }}</span>
                 </button>
              </div>
           </div>
        </section>

        <!-- Branding -->
        <section v-if="isAdmin" class="space-y-4">
           <div class="flex items-center gap-2 px-1">
             <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-indigo-400">
               <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
             </svg>
              <h2 class="text-sm font-bold text-text uppercase tracking-wider">Tenant Identity & Branding</h2>
              <span v-if="!entStore.hasFeature('branding')" class="badge bg-amber-500/10 text-amber-500 text-[9px] font-bold tracking-widest">Enterprise Feature</span>
            </div>

            <div class="card p-5 space-y-4" :class="{'opacity-40 grayscale pointer-events-none select-none': !entStore.hasFeature('branding')}">
               <div class="grid grid-cols-2 gap-4">
                  <div>
                     <label class="text-[10px] font-bold text-text-muted uppercase tracking-widest block mb-1">Company Name</label>
                     <input v-model="brandingForm.name" class="input text-sm w-full" placeholder="e.g. Acme Corp" />
                  </div>
                  <div>
                     <label class="text-[10px] font-bold text-text-muted uppercase tracking-widest block mb-1">Logo URL</label>
                     <input v-model="brandingForm.logo_url" class="input text-sm w-full" placeholder="https://..." autocomplete="off" />
                  </div>
               </div>
               <div class="flex justify-end">
                  <button @click="saveBranding" :disabled="brandingBusy" class="btn-primary text-xs px-4">
                     {{ brandingBusy ? 'Saving...' : 'Save Branding' }}
                  </button>
               </div>
            </div>
        </section>


        <!-- LLM Configs -->
        <section v-if="isAdmin" class="space-y-4">
           <div class="flex items-center gap-2 px-1">
             <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-indigo-400">
               <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
             </svg>
             <h2 class="text-sm font-bold text-text uppercase tracking-wider">LLM Providers</h2>
           </div>
           <div v-for="prov in llmStore.PROVIDERS" :key="prov.id" class="card p-5 space-y-4">
             <div class="flex items-center gap-3">
               <div class="flex-1">
                 <h2 class="text-sm font-semibold text-text">{{ prov.label }}</h2>
                 <p class="text-xs text-text-muted mt-0.5">{{ providerEndpoint(prov.id) || 'Local server' }}</p>
               </div>
               <label class="flex items-center gap-2 cursor-pointer">
                 <input type="checkbox" v-model="forms[prov.id].enabled" class="rounded" />
                 <span class="text-xs text-text-muted">Enabled</span>
               </label>
               <span v-if="isDefaultProvider(prov.id)" class="text-xs px-2 py-0.5 rounded-full bg-accent/20 text-accent font-medium">default</span>
             </div>

             <div class="grid grid-cols-2 gap-3">
               <div v-if="prov.needsKey" class="col-span-2">
                 <label class="text-xs text-text-muted mb-1 block">API Key</label>
                 <input v-model="forms[prov.id].api_key" type="password" :placeholder="hasExistingKey(prov.id) ? 'Configured (leave blank to keep)' : 'Enter API key'" class="input text-sm w-full font-mono" autocomplete="new-password" />
               </div>
               <div v-if="prov.needsURL" class="col-span-2">
                 <label class="text-xs text-text-muted mb-1 block">Server URL</label>
                 <input v-model="forms[prov.id].base_url" type="text" placeholder="http://localhost:11434" class="input text-sm w-full font-mono" />
               </div>
               <div>
                 <label class="text-xs text-text-muted mb-1 block">Default Model</label>
                 <input v-model="forms[prov.id].default_model" type="text" :placeholder="llmStore.DEFAULT_MODELS[prov.id]" class="input text-sm w-full" />
               </div>
               <div>
                 <label class="text-xs text-text-muted mb-1 block">Max Output Tokens</label>
                 <input v-model.number="forms[prov.id].max_output_tokens" type="number" placeholder="4096" class="input text-sm w-full" />
               </div>
             </div>

             <div v-if="statuses[prov.id]" :class="['text-xs px-3 py-2 rounded', statuses[prov.id].ok ? 'bg-green-500/10 text-green-400' : 'bg-red-500/10 text-red-400']">
               {{ statuses[prov.id].ok ? '✓ Connected' : '✗ ' + statuses[prov.id].error }}
             </div>

             <div class="flex gap-2 pt-1">
               <button @click="save(prov.id)" :disabled="saving[prov.id]" class="btn-primary text-xs px-3 py-1.5">{{ saving[prov.id] ? 'Saving…' : 'Save' }}</button>
               <button @click="test(prov.id)" :disabled="testing[prov.id]" class="btn-ghost text-xs px-3 py-1.5">{{ testing[prov.id] ? 'Testing…' : 'Test Connection' }}</button>
               <button v-if="forms[prov.id].enabled && !isDefaultProvider(prov.id)" @click="saveDefault(prov.id)" class="btn-ghost text-violet-400 text-xs px-3 py-1.5 font-bold uppercase">Set as Default</button>
             </div>
           </div>
        </section>

        <!-- Infrastructure & Telemetry -->
        <section v-if="isAdmin" class="space-y-4">
           <div class="flex items-center gap-2 px-1">
             <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-purple-400">
               <rect x="4" y="4" width="16" height="16" rx="2"/><path d="M9 9h6v6H9z"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="M2 12h2"/><path d="M20 12h2"/>
             </svg>
             <h2 class="text-sm font-bold text-text uppercase tracking-wider">Infrastructure</h2>
           </div>

           <div class="card p-5 flex items-start justify-between">
              <div class="space-y-1">
                <h3 class="text-xs font-bold text-text">Anonymized Telemetry</h3>
                <p class="text-[11px] text-text-muted max-w-sm">
                  Share platform-wide usage metrics with Chain Nodes HQ.
                </p>
              </div>
              <button @click="toggleTelemetry" :class="['w-10 h-5 rounded-full relative transition-colors duration-200 outline-none', telemetryEnabled ? 'bg-emerald-500' : 'bg-surface-3']">
                <span :class="['absolute top-1 left-1 bg-white w-3 h-3 rounded-full transition-transform', telemetryEnabled ? 'translate-x-5' : 'translate-x-0']"></span>
              </button>
           </div>
         </section>
      </div>

      <!-- Secrets Management (Enterprise Feature) -->
      <div v-if="activeTab === 'secrets'" class="flex-1 px-6 py-6 max-w-3xl w-full mx-auto space-y-6">
        <section v-if="isAdmin" class="space-y-4">
           <div class="flex items-center gap-2 px-1">
             <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-indigo-400">
               <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
             </svg>
              <h2 class="text-sm font-bold text-text uppercase tracking-wider">Environment Secrets</h2>
              <span v-if="!entStore.hasFeature('secrets')" class="badge bg-amber-500/10 text-amber-500 text-[9px] font-bold tracking-widest">Enterprise Feature</span>
            </div>

            <div class="card p-5 space-y-4" :class="{'opacity-40 grayscale pointer-events-none select-none': !entStore.hasFeature('secrets')}">
              <div v-for="(val, idx) in secretsForm" :key="idx" class="flex gap-2 items-center">
                 <input v-model="secretsForm[idx].name" class="input text-xs w-1/3" placeholder="KEY_NAME" />
                 <input v-model="secretsForm[idx].value" type="password" class="input text-xs flex-1" placeholder="Value (hidden)" />
                 <button @click="removeSecret(idx)" class="btn-ghost text-red-400 p-2">✕</button>
              </div>
              <div class="flex justify-between items-center pt-2">
                 <button @click="addSecret" class="btn-ghost text-[10px] font-bold uppercase">+ Add Secret</button>
                 <button @click="saveSecrets" :disabled="secretsBusy" class="btn-primary text-xs px-4">
                    {{ secretsBusy ? 'Saving...' : 'Save Secrets' }}
                 </button>
              </div>
            </div>

            <div v-if="!entStore.hasFeature('secrets')" class="text-center p-6 bg-surface-2 rounded-xl border border-border">
              <p class="text-xs text-text-muted">
                Secrets Management is an Enterprise feature. Please upload an enterprise license token or upgrade your plan to unlock encrypted vault storage for keys and API credentials.
              </p>
            </div>
        </section>
      </div>

      <div v-if="activeTab === 'prompts'" class="flex-1 px-6 py-6 max-w-5xl w-full mx-auto space-y-6">
        <section class="space-y-6">
           <div class="flex items-center justify-between px-1">
             <div class="flex items-center gap-2">
               <svg class="text-violet-400" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                 <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
               </svg>
               <h2 class="text-sm font-bold text-text uppercase tracking-wider">AI Designer System Prompts</h2>
             </div>
             <p class="text-[10px] text-text-muted uppercase tracking-widest font-bold">Overrides reset on server restart</p>
           </div>

          <div v-for="(content, id) in systemPrompts" :key="id" class="space-y-2 group">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <span class="text-[10px] font-mono text-violet-400/80 uppercase tracking-widest font-bold">{{ id.replace(/workflow_/g, '').replace(/_/g, ' ') }}</span>
                <button 
                  @click="openPopout(id)"
                  class="p-1 rounded hover:bg-surface-2 text-text-muted hover:text-violet-400 transition-all opacity-0 group-hover:opacity-100"
                  title="Full-screen Editor"
                >
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/>
                  </svg>
                </button>
              </div>
              <button @click="savePrompt(id)" :disabled="savingPrompt[id]" class="text-[10px] bg-violet-500/10 text-violet-400 hover:bg-violet-500/20 px-2 py-1 rounded font-bold uppercase transition-all">
                {{ savingPrompt[id] ? 'Saving...' : 'Save Override' }}
              </button>
            </div>
            
            <div class="h-80 shadow-inner rounded-lg border border-border/50 overflow-hidden">
              <CodeEditor
                v-model="systemPrompts[id]"
                language="text"
                height="100%"
                :show-badge="false"
              />
            </div>
          </div>
        </section>
      </div>

    <CodeEditorModal
      v-if="activePopoutId"
      v-model="systemPrompts[activePopoutId]"
      :title="activePopoutId.replace(/_/g, ' ').toUpperCase()"
      language="text"
      @close="activePopoutId = null"
    />

  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import { useAuthStore, api } from '@/stores/auth'
import { useLLMStore } from '@/stores/llm'
import { useEnterpriseStore } from '@/stores/enterprise'
import { useTheme } from '@/composables/useTheme'
import CodeEditor from '@/components/designer/CodeEditor.vue'
import CodeEditorModal from '@/components/designer/CodeEditorModal.vue'
const activeTab = ref('general')

const authStore = useAuthStore()
const llmStore = useLLMStore()
const entStore = useEnterpriseStore()
const { theme, setTheme } = useTheme()

const activePopoutId = ref(null)

function openPopout(id) {
  activePopoutId.value = id
}

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
    mistral:   'api.mistral.ai',
    groq:      'api.groq.com',
    together:  'api.together.xyz',
    fireworks: 'api.fireworks.ai',
    cohere:    'api.cohere.ai',
    qwen:      'dashscope.aliyuncs.com',
    glm:       'open.bigmodel.cn',
  }
  return map[id] ?? ''
}

const updatingLicense = ref(false)
const licenseInput = ref('')
const licenseBusy = ref(false)

const brandingForm = reactive({ name: '', logo_url: '' })
const brandingBusy = ref(false)

async function saveBranding() {
  brandingBusy.value = true
  try {
    await entStore.updateBranding(brandingForm)
  } catch (e) {
    alert('Failed to update branding: ' + (e.response?.data?.error ?? e.message))
  } finally {
    brandingBusy.value = false
  }
}

const secretsForm = ref([])
const secretsBusy = ref(false)

function addSecret() {
  secretsForm.value.push({ name: '', value: '' })
}
function removeSecret(idx) {
  secretsForm.value.splice(idx, 1)
}

async function saveSecrets() {
  secretsBusy.value = true
  try {
    const payload = {}
    secretsForm.value.forEach(s => {
      if (s.name) payload[s.name] = s.value
    })
    await entStore.updateSecrets(payload)
    alert('Secrets saved successfully.')
  } catch (e) {
    alert('Failed to save secrets: ' + (e.response?.data?.error ?? e.message))
  } finally {
    secretsBusy.value = false
  }
}

async function saveLicense() {
  if (!licenseInput.value) return
  licenseBusy.value = true
  try {
    await entStore.setLicense(licenseInput.value)
    updatingLicense.value = false
    licenseInput.value = ''
  } catch (e) {
    alert('License update failed: ' + (e.response?.data?.error ?? e.message))
  } finally {
    licenseBusy.value = false
  }
}

function usagePercent(curr, max) {
  if (!max) return 0
  return Math.min(100, Math.round((curr / max) * 100))
}

function tierBadgeClass(tier) {
  const base = 'badge'
  if (tier === 'enterprise') return `${base} bg-purple-500/20 text-purple-400`
  if (tier === 'pro') return `${base} bg-indigo-500/20 text-indigo-400`
  return `${base} bg-slate-500/20 text-text-muted`
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

async function saveDefault(provider) {
  const p = provider || selectedDefault.value
  if (!p) return
  try {
    await llmStore.setDefault(p)
  } catch (e) {
    console.error('Failed to set default provider', e)
  }
}

// -- System Prompts --
const systemPrompts = reactive({
  workflow_generator_base: '',
  skill_analyser_preamble: '',
  workflow_refinement_addendum: '',
  workflow_decompose: '',
  workflow_categorise: '',
  workflow_wire: '',
  workflow_implement_finish: '',
  workflow_debugger: '',
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
  if (!isAdmin.value) return

  // Load core settings
  try {
    await Promise.allSettled([
      llmStore.fetchAll(),
      fetchPrompts()
    ])
    syncFormsFromStore()
  } catch (e) { console.error('Failed to load core settings', e) }

  // Load Enterprise/SuperAdmin settings (may fail if not licensed)
  try {
    await Promise.allSettled([
      entStore.fetchBranding().then(() => Object.assign(brandingForm, entStore.branding)),
      entStore.fetchSecrets().then(() => {
        secretsForm.value = Object.entries(entStore.secrets).map(([k, v]) => ({ name: k, value: '' }))
      }),
      entStore.fetchAuditLogs(),
      entStore.fetchStatus()
    ])
  } catch (e) { console.error('Failed to load enterprise settings', e) }

  if (authStore.hasRole('super_admin')) {
    try {
      await fetchTelemetry()
    } catch (e) { console.error('Failed to fetch telemetry', e) }
  }
})

const telemetryEnabled = ref(false)

async function fetchTelemetry() {
  try {
    const { data } = await api.get('/admin/telemetry')
    telemetryEnabled.value = data.enabled
  } catch (e) { console.error('Failed to fetch telemetry', e) }
}

async function toggleTelemetry() {
  const next = !telemetryEnabled.value
  try {
    await api.post('/admin/telemetry', { enabled: next })
    telemetryEnabled.value = next
  } catch (e) { alert('Failed to toggle telemetry') }
}

function formatDateTime(ts) {
  if (!ts) return '-'
  return new Date(ts).toLocaleString(undefined, {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
  })
}
</script>
