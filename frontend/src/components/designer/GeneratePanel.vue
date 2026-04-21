<template>
  <div class="flex flex-col h-full overflow-hidden bg-surface-1 relative">
    <!-- Loading Overlay -->
    <div v-if="analyseLoading" class="absolute inset-0 z-50 bg-surface-1/80 backdrop-blur-sm flex flex-col items-center justify-center space-y-4">
      <div class="relative">
        <div class="w-12 h-12 border-4 border-violet-500/20 border-t-violet-500 rounded-full animate-spin"></div>
        <div class="absolute inset-0 flex items-center justify-center">
          <svg class="text-violet-400" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
          </svg>
        </div>
      </div>
      <div class="flex flex-col items-center">
        <span class="text-sm font-semibold text-slate-200">AI is thinking...</span>
        <span class="text-[10px] text-slate-500 animate-pulse">Decomposing process into ASM states</span>
      </div>
      <button 
        @click="analyseLoading = false" 
        class="mt-4 px-3 py-1 text-[10px] text-slate-400 hover:text-slate-200 border border-slate-700 hover:border-slate-500 rounded transition-colors"
      >
        Dismiss Overlay (Stop Waiting)
      </button>
    </div>

    <!-- Tab Bar -->
    <div class="flex border-b border-border shrink-0">
      <button
        @click="activeSubTab = 'logic'"
        :class="['flex-1 py-2 text-[10px] font-bold uppercase tracking-wider transition-colors', activeSubTab === 'logic' ? 'text-violet-400 border-b-2 border-violet-400' : 'text-slate-600 hover:text-slate-400']"
      >
        Logic
      </button>
      <button
        @click="activeSubTab = 'trace'"
        :class="['flex-1 py-2 text-[10px] font-bold uppercase tracking-wider transition-colors flex items-center justify-center gap-1.5', activeSubTab === 'trace' ? 'text-teal-400 border-b-2 border-teal-400' : 'text-slate-600 hover:text-slate-400']"
      >
        Trace
        <span v-if="interactions.length" class="flex h-1.5 w-1.5 rounded-full bg-teal-500 animate-pulse" />
      </button>
    </div>

    <!-- Main Content -->
    <div v-if="activeSubTab === 'logic'" class="flex-1 flex flex-col min-h-0">
      <div class="flex-1 overflow-y-auto p-4 space-y-6 custom-scrollbar">
        
        <!-- Model Selection -->
        <div class="space-y-1.5 p-3 rounded-lg bg-surface-0/50 border border-border/50">
          <div class="flex items-center justify-between">
            <label class="text-[10px] text-slate-500 uppercase font-bold tracking-tighter">Model Provider</label>
            <span v-if="activeProviderLabel" class="text-[9px] text-violet-400 font-medium tracking-wide bg-violet-400/10 px-1.5 py-0.5 rounded">{{ activeProviderLabel }}</span>
          </div>
          <select 
            :value="provider"
            @change="updateVal('provider', $event.target.value)"
            class="input w-full text-[11px] h-8 bg-surface-1 border-border/40 focus:border-violet-500/50 transition-all font-sans"
          >
            <option value="">(Platform Default)</option>
            <option v-for="p in enabledProviders" :key="p.id" :value="p.id">
              {{ p.label }}
            </option>
          </select>
          <p class="text-[9px] text-slate-600">Select an alternative model if the default is slow or offline.</p>
        </div>
        <!-- Section 1: Process Description -->
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="text-[10px] text-slate-500 uppercase font-bold tracking-tighter">Initial Process Description</label>
            <span class="text-[9px] text-slate-600 italic">Fresh generation</span>
          </div>
          <textarea
            :value="description"
            @input="updateVal('description', $event.target.value)"
            class="input text-[11px] h-32 resize-none font-sans leading-relaxed"
            placeholder="Describe the end-to-end business process in detail..."
          />
          <button
            @click="generateFresh"
            :disabled="analyseLoading || !description.trim()"
            class="w-full btn py-2 gap-1.5 text-xs font-semibold rounded-lg bg-violet-600 hover:bg-violet-500 text-white shadow-lg shadow-violet-900/20 transition-all disabled:opacity-40"
          >
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
            </svg>
            Generate Workflow
          </button>
        </div>

        <div class="h-px bg-border/50" />

        <!-- Section 2: Refinement Prompt -->
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="text-[10px] text-slate-500 uppercase font-bold tracking-tighter">Refinement Prompt</label>
            <span v-if="history.length" class="text-[9px] text-emerald-500 font-bold uppercase tracking-tighter">History Active</span>
          </div>
          <textarea
            :value="prompt"
            @input="updateVal('prompt', $event.target.value)"
            class="input text-[11px] h-20 resize-none font-sans leading-relaxed border-indigo-500/20 focus:border-indigo-500/50"
            placeholder="Request specific changes or tweaks..."
            @keydown.enter.meta.prevent="applyRefinement"
          />
          <button
            @click="applyRefinement"
            :disabled="analyseLoading || !prompt.trim()"
            class="w-full btn py-2 gap-1.5 text-xs font-semibold rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white transition-all disabled:opacity-40"
          >
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
            </svg>
            Apply Changes
          </button>
        </div>

        <div v-if="analyseError" class="p-2.5 rounded bg-red-500/10 border border-red-500/20 text-red-400 text-[10px] font-mono">
          {{ analyseError }}
        </div>

        <!-- Section 3: History (Debug) -->
        <div v-if="history.length" class="space-y-3 pt-4 border-t border-border/50">
          <div class="flex items-center justify-between">
            <h3 class="text-[10px] font-bold text-slate-600 uppercase tracking-widest">Historical Context</h3>
            <button @click="resetHistory" class="text-[9px] text-slate-500 hover:text-red-400 uppercase font-bold underline decoration-dotted underline-offset-2">Clear</button>
          </div>
          <div class="space-y-4">
            <div v-for="(msg, i) in history" :key="i" class="space-y-1.5 opacity-60 hover:opacity-100 transition-opacity">
              <div class="flex items-center gap-2">
                <span :class="['text-[8px] font-bold uppercase tracking-widest px-1 py-0.5 rounded', msg.role === 'user' ? 'text-indigo-400 bg-indigo-400/10' : 'text-emerald-400 bg-emerald-400/10']">
                  {{ msg.role === 'user' ? 'User' : 'AI' }}
                </span>
                <span class="text-[8px] text-slate-700 font-mono">Turn {{ Math.floor(i/2) + 1 }}</span>
              </div>
              <div class="text-[10px] text-slate-500 leading-relaxed font-mono whitespace-pre-wrap pl-2 border-l border-border/30">
                {{ msg.content }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Trace Content -->
    <div v-else-if="activeSubTab === 'trace'" class="flex-1 min-h-0">
      <TraceViewer :interactions="interactions" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api, useAuthStore } from '@/stores/auth'
import { useLLMStore } from '@/stores/llm'
import TraceViewer from './TraceViewer.vue'

const authStore = useAuthStore()
const llmStore = useLLMStore()

const props = defineProps({
  yamlSource: { type: String, default: '' },
  description: { type: String, default: '' },
  prompt: { type: String, default: '' },
  history: { type: Array, default: () => [] },
  interactions: { type: Array, default: () => [] },
  provider: { type: String, default: '' },
  metadataAbstract: { type: String, default: '' },
})

const emit = defineEmits([
  'update:description', 
  'update:prompt', 
  'update:history', 
  'update:interactions', 
  'update:provider',
  'apply'
])

// ── Role Check ──────────────────────────────────────────────────────────────
const isAdmin = computed(() => authStore.user?.role === 'admin' || authStore.user?.role === 'super_admin')

// ── State ────────────────────────────────────────────────────────────────────
const activeSubTab = ref('logic')
const analyseLoading = ref(false)
const analyseError = ref('')

function updateVal(key, val) {
  emit(`update:${key}`, val)
}

// ── Actions ──────────────────────────────────────────────────────────────────

async function generateFresh() {
  if (!props.description.trim()) return
  
  // Requirement: Change of initial description should start a fresh process.
  emit('update:history', [])
  emit('update:interactions', [])
  
  await performAnalyse(props.description, [])
}

async function applyRefinement() {
  if (!props.prompt.trim()) return
  
  await performAnalyse(props.prompt, props.history)
}

async function performAnalyse(text, context) {
  analyseLoading.value = true
  analyseError.value = ''
  
  try {
    const { data } = await api.post('/designer/skill/analyse', {
      description: text,
      history: context,
      existing_yaml: props.yamlSource || '',
      provider: props.provider || undefined,
    })
    
    // Update history: push current turn
    const newHistory = [...context]
    newHistory.push({ role: 'user', content: text })
    newHistory.push({ role: 'assistant', content: 'Workflow updated.' })
    emit('update:history', newHistory)
    
    // Clear prompt if it was a refinement
    if (text === props.prompt) {
      emit('update:prompt', '')
    }
    
    // Interactions for trace
    emit('update:interactions', data.interactions || [])
    
    // Notify parent to update canvas
    emit('apply', { yamlStr: data.yaml, definition: data.definition })
    
  } catch (e) {
    analyseError.value = e.response?.data?.error ?? e.message
    
    // Attempt to recover partial trace even on failure (e.g. to see System Prompt)
    if (e.response?.data?.interactions) {
      emit('update:interactions', e.response.data.interactions)
    }
  } finally {
    analyseLoading.value = false
  }
}

function resetHistory() {
  emit('update:history', [])
  emit('update:interactions', [])
}

// -- LLM Providers Logic --
const enabledProviders = computed(() => {
  return llmStore.PROVIDERS.filter(p => {
    const cfg = llmStore.configs.find(c => c.provider === p.id)
    return cfg && cfg.enabled
  })
})

const activeProviderLabel = computed(() => {
  if (!props.provider) return ''
  return llmStore.PROVIDERS.find(p => p.id === props.provider)?.label || ''
})

onMounted(() => {
  if (llmStore.configs.length === 0) {
    llmStore.fetchAll()
  }
})
</script>

<style scoped>
.custom-scrollbar::-webkit-scrollbar { width: 4px; }
.custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
.custom-scrollbar::-webkit-scrollbar-thumb { background: #2a2d3a; border-radius: 10px; }
.custom-scrollbar::-webkit-scrollbar-thumb:hover { background: #3f445b; }
</style>
