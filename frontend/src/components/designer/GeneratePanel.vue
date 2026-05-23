<template>
  <div class="flex flex-col h-full overflow-hidden bg-surface-1 relative">

    <!-- Loading Overlay -->
    <div v-if="loading" class="absolute inset-0 z-50 bg-surface-1/80 backdrop-blur-sm flex flex-col items-center justify-center space-y-4">
      <div class="relative">
        <div class="w-12 h-12 border-4 border-violet-500/20 border-t-violet-500 rounded-full animate-spin"></div>
        <div class="absolute inset-0 flex items-center justify-center">
          <svg class="text-violet-400" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
          </svg>
        </div>
      </div>
      <div class="flex flex-col items-center">
        <span class="text-sm font-semibold text-text">{{ loadingLabel }}</span>
        <span class="text-[10px] text-text-muted animate-pulse">{{ loadingSubLabel }}</span>
      </div>
      <button
        @click="loading = false"
        class="mt-4 px-3 py-1 text-[10px] text-text-muted hover:text-text border border-slate-700 hover:border-slate-500 rounded transition-colors"
      >
        Dismiss Overlay
      </button>
    </div>

    <!-- Tab Bar: Generate | Trace -->
    <div class="flex border-b border-border shrink-0">
      <button
        @click="activeTab = 'generate'"
        :class="['flex-1 py-2 text-[10px] font-bold uppercase tracking-wider transition-colors', activeTab === 'generate' ? 'text-violet-400 border-b-2 border-violet-400' : 'text-text-muted hover:text-text-muted']"
      >Generate</button>
      <button
        @click="activeTab = 'trace'"
        :class="['flex-1 py-2 text-[10px] font-bold uppercase tracking-wider transition-colors flex items-center justify-center gap-1.5', activeTab === 'trace' ? 'text-teal-400 border-b-2 border-teal-400' : 'text-text-muted hover:text-text-muted']"
      >
        Trace
        <span v-if="interactions.length" class="flex h-1.5 w-1.5 rounded-full bg-teal-500 animate-pulse" />
      </button>
    </div>

    <!-- ── GENERATE TAB ──────────────────────────────────────────────── -->
    <div v-if="activeTab === 'generate'" class="flex-1 flex flex-col min-h-0">
      <div class="flex-1 overflow-y-auto p-4 space-y-5 custom-scrollbar">

        <!-- Model Provider -->
        <div class="p-3 rounded-lg bg-surface-0/50 border border-border/50 space-y-1.5">
          <div class="flex items-center justify-between">
            <label class="text-[10px] text-text-muted uppercase font-bold tracking-tighter">Model Provider</label>
            <span v-if="activeProviderLabel" class="text-[9px] text-violet-400 font-medium tracking-wide bg-violet-400/10 px-1.5 py-0.5 rounded">{{ activeProviderLabel }}</span>
          </div>
          <select
            :value="provider"
            @change="emit('update:provider', $event.target.value)"
            class="input w-full text-[11px] h-8 bg-surface-1 border-border/40 focus:border-violet-500/50 transition-all font-sans"
          >
            <option value="">(Platform Default)</option>
            <option v-for="p in enabledProviders" :key="p.id" :value="p.id">{{ p.label }}</option>
          </select>
        </div>

        <!-- Process Description -->
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <label class="text-[10px] text-text-muted uppercase font-bold tracking-tighter">Process Description</label>
            <button @click="emit('requestDescriptionModal')" class="text-text-muted hover:text-violet-400 transition-colors" title="Expand editor">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/>
              </svg>
            </button>
          </div>
          <textarea
            :value="description"
            @input="onDescriptionInput"
            class="input text-sm h-28 resize-none font-sans leading-relaxed"
            placeholder="Describe the end-to-end business process in detail..."
          />
        </div>

        <!-- Pipeline Steps -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="text-[10px] text-text-muted uppercase font-bold tracking-tighter">Generation Pipeline</label>
            <button
              v-if="props.pipelineStep > 0"
              @click="resetPipeline"
              class="text-[9px] text-text-muted hover:text-red-400 uppercase font-bold underline decoration-dotted underline-offset-2"
            >Reset</button>
          </div>

          <!-- Step indicators -->
          <div class="flex items-center gap-1">
            <div v-for="(step, i) in pipelineSteps" :key="i" class="flex-1 flex flex-col items-center gap-1">
              <button
                @click="runStep(i)"
                :disabled="loading || (i > 0 && !description.trim())"
                :class="[
                  'w-6 h-6 rounded-full flex items-center justify-center text-[9px] font-bold transition-all',
                  props.pipelineStep > i   ? 'bg-emerald-500 text-white cursor-pointer hover:scale-110 active:scale-95' :
                  props.pipelineStep === i ? 'bg-violet-500 text-white ring-2 ring-violet-400/30 cursor-pointer hover:scale-110 active:scale-95' :
                                             'bg-surface-0 text-text-muted border border-border cursor-pointer hover:border-violet-500/50 hover:text-text'
                ]"
                :title="`Run Step ${i + 1}: ${step.label}`"
              >
                <svg v-if="pipelineStep > i" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><polyline points="20 6 9 17 4 12"/></svg>
                <span v-else>{{ i + 1 }}</span>
              </button>
              <span :class="[
                'text-[8px] text-center leading-tight transition-colors',
                props.pipelineStep === i ? 'text-violet-400 font-bold' : 'text-text-muted'
              ]">{{ step.label }}</span>
            </div>
          </div>

          <!-- Current step description -->
          <div v-if="props.pipelineStep < pipelineSteps.length" class="p-2.5 rounded bg-surface-0/60 border border-border/50">
            <p class="text-[10px] text-text-muted leading-relaxed">
              <span class="text-violet-400 font-semibold">Step {{ props.pipelineStep + 1 }}:</span>
              {{ pipelineSteps[props.pipelineStep].description }}
            </p>
          </div>
          <div v-else class="p-2.5 rounded bg-emerald-500/10 border border-emerald-500/20">
            <p class="text-[10px] text-emerald-400 font-semibold">Pipeline complete — workflow is runnable.</p>
          </div>
        </div>

        <!-- Pipeline error -->
        <div v-if="pipelineError" class="p-2.5 rounded bg-red-500/10 border border-red-500/20 text-red-400 text-[10px] font-mono">
          {{ pipelineError }}
        </div>

        <!-- Soft validation warning (steps 1–3) -->
        <div v-if="props.pipelineWarning && props.pipelineStep < 4" class="p-2.5 rounded bg-amber-500/10 border border-amber-500/20 text-amber-400 text-[10px]">
          <span class="font-semibold">Note:</span> {{ props.pipelineWarning }}
        </div>

        <!-- Pipeline action button -->
        <div>
          <button
            v-if="props.pipelineStep < pipelineSteps.length"
            @click="runStep(props.pipelineStep)"
            :disabled="loading || !description.trim()"
            class="w-full btn py-2 gap-1.5 text-xs font-semibold rounded-lg bg-violet-600 hover:bg-violet-500 text-white shadow-lg shadow-violet-900/20 transition-all disabled:opacity-40"
          >
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <polyline points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
            </svg>
            {{ props.pipelineStep === 0 ? 'Start: Decompose Process' : `Run Step ${props.pipelineStep + 1}: ${pipelineSteps[props.pipelineStep].label}` }}
          </button>
          <button
            v-else
            @click="resetPipeline"
            class="w-full btn py-2 text-xs font-semibold rounded-lg bg-slate-700 hover:bg-slate-600 text-white transition-all"
          >
            Start Over
          </button>
        </div>

        <!-- Divider -->
        <div class="flex items-center gap-2 pt-1">
          <div class="flex-1 h-px bg-border/50"/>
          <span class="text-[9px] text-text-muted uppercase tracking-widest font-bold">Designer Chat</span>
          <div class="flex-1 h-px bg-border/50"/>
        </div>

        <!-- Chat-like Refinement -->
        <div class="flex-1 flex flex-col min-h-[300px] bg-surface-0/30 rounded-xl border border-border/50 overflow-hidden">
          <!-- History Stream -->
          <div 
            ref="refineScroll"
            class="flex-1 overflow-y-auto p-3 space-y-3 custom-scrollbar"
          >
            <div v-for="(item, i) in history" :key="i" 
              :class="['flex flex-col gap-1', item.role === 'user' ? 'items-end' : 'items-start']"
            >
              <span class="text-[8px] font-bold uppercase tracking-tighter text-text-muted px-1">
                {{ item.role === 'user' ? 'You' : 'Designer' }}
              </span>
              <div 
                :class="[
                  'max-w-[90%] px-3 py-2 rounded-xl text-sm leading-relaxed',
                  item.role === 'user' 
                    ? 'bg-indigo-600 text-white rounded-tr-none' 
                    : 'bg-surface-2 text-text border border-border/50 rounded-tl-none'
                ]"
              >
                {{ item.content }}
              </div>
            </div>
            
            <div v-if="!history.length" class="h-full flex flex-col items-center justify-center text-center opacity-20 py-12">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="mb-2">
                <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 1 1-7.6-13.5 8.38 8.38 0 0 1 3.8.9L21 3z"/>
              </svg>
              <p class="text-[10px]">Ask for changes, additions, or refactorings.</p>
            </div>
          </div>

          <!-- Input Area -->
          <div class="p-3 bg-surface-1/50 border-t border-border/50">
            <div class="relative">
              <textarea
                :value="prompt"
                @input="emit('update:prompt', $event.target.value)"
                @keydown.enter.meta.prevent="applyRefinement"
                class="w-full bg-surface-0 border border-border/40 rounded-lg px-3 py-2 text-sm h-16 resize-none focus:outline-none focus:ring-1 focus:ring-indigo-500/50 focus:border-indigo-500/50 transition-all pr-10"
                placeholder="e.g. Add error handling..."
              />
              <button 
                @click="applyRefinement"
                :disabled="loading || !prompt.trim()"
                class="absolute right-2 bottom-2 p-1.5 rounded-md bg-indigo-600 text-white hover:bg-indigo-500 transition-all disabled:opacity-30"
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                  <line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/>
                </svg>
              </button>
            </div>
            <div class="flex items-center justify-between mt-2">
              <span class="text-[8px] text-text-muted">⌘ + Enter to apply</span>
              <button 
                v-if="history.length"
                @click="clearHistory"
                class="text-[8px] text-text-muted hover:text-red-400 font-bold uppercase tracking-wider"
              >Clear Chat</button>
            </div>
          </div>
        </div>

        <!-- Refine error -->
        <div v-if="refineError" class="p-2.5 rounded bg-red-500/10 border border-red-500/20 text-red-400 text-[10px] font-mono">
          {{ refineError }}
        </div>

      </div>
    </div>

    <!-- ── TRACE TAB ─────────────────────────────────────────────────── -->
    <div v-else-if="activeTab === 'trace'" class="flex-1 min-h-0">
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

const refineScroll = ref(null)

const props = defineProps({
  yamlSource:              { type: String, default: '' },
  description:             { type: String, default: '' },
  prompt:                  { type: String, default: '' },
  history:                 { type: Array,  default: () => [] },
  interactions:            { type: Array,  default: () => [] },
  provider:                { type: String, default: '' },
  metadataAbstract:        { type: String, default: '' },
  pipelineStep:            { type: Number, default: 0 },
  pipelineWarning:         { type: String, default: '' },
})

const emit = defineEmits([
  'update:description',
  'update:prompt',
  'update:history',
  'update:interactions',
  'update:provider',
  'update:pipelineStep',
  'update:pipelineWarning',
  'apply',
  'requestDescriptionModal',
])

// ── Tabs ──────────────────────────────────────────────────────────────────────
const activeTab = ref('generate')

// ── Pipeline state ────────────────────────────────────────────────────────────
const pipelineSteps = [
  { label: 'Decompose',  endpoint: '/designer/pipeline/decompose',  description: 'Identify all states and add a description to every node. The canvas will show a skeleton workflow.' },
  { label: 'Categorise', endpoint: '/designer/pipeline/categorise', description: 'Assign the correct Chain Nodes type to every node (code, script, hitl, wait, etc.). No runnable logic yet.' },
  { label: 'Wire',       endpoint: '/designer/pipeline/wire',       description: 'Define all transitions, guards, and the blackboard schema — the shared contract for node implementations.' },
  { label: 'Implement',  endpoint: '/designer/pipeline/implement',  description: 'Generate final code for every node in parallel using specialised prompts. Produces a runnable workflow.' },
]

// pipelineStep and pipelineWarning are lifted to the parent so they survive tab switches.
const pipelineError = ref('')

// ── Shared loading ────────────────────────────────────────────────────────────
const loading         = ref(false)
const loadingLabel    = ref('AI is thinking...')
const loadingSubLabel = ref('')
const refineError     = ref('')

// ── Pipeline actions ──────────────────────────────────────────────────────────
function resetPipeline() {
  emit('update:pipelineStep', 0)
  emit('update:pipelineWarning', '')
  pipelineError.value = ''
}

function onDescriptionInput(e) {
  emit('update:description', e.target.value)
  if (props.pipelineStep > 0) resetPipeline()
}

async function runStep(index) {
  const step = pipelineSteps[index]
  if (!step) return

  // Prerequisite check: index 0 needs description, others need description + existing YAML
  if (index === 0 && !props.description.trim()) {
    pipelineError.value = "Process description is required."
    return
  }
  if (index > 0 && (!props.description.trim() || !props.yamlSource.trim())) {
    pipelineError.value = index > 0 && !props.yamlSource.trim() 
      ? "Existing YAML is required for this step. Run earlier steps first or import YAML."
      : "Process description is required."
    return
  }

  loading.value = true
  loadingLabel.value = `Step ${index + 1}: ${step.label}…`
  loadingSubLabel.value = step.description
  pipelineError.value = ''
  emit('update:pipelineWarning', '')

  try {
    const { data } = await api.post(step.endpoint, {
      process_description: props.description,
      current_yaml:        props.yamlSource || '',
      provider:            props.provider || undefined,
    })

    if (data.validation_error) emit('update:pipelineWarning', data.validation_error)

    emit('update:interactions', data.interactions || [])
    emit('apply', { yamlStr: data.yaml, definition: data.definition })
    
    // Advance pipeline step if we just finished the current one
    if (index === props.pipelineStep) {
      emit('update:pipelineStep', index + 1)
    }

  } catch (e) {
    pipelineError.value = e.response?.data?.error ?? e.message
    if (e.response?.data?.interactions) emit('update:interactions', e.response.data.interactions)
  } finally {
    loading.value = false
  }
}

// ── Refine actions ────────────────────────────────────────────────────────────
async function applyRefinement() {
  if (!props.prompt.trim()) return

  loading.value = true
  loadingLabel.value = 'Applying refinement…'
  loadingSubLabel.value = props.prompt.slice(0, 80)
  refineError.value = ''

  try {
    const { data } = await api.post('/designer/process/analyse', {
      description:   props.prompt,
      history:       props.history,
      existing_yaml: props.yamlSource || '',
      provider:      props.provider || undefined,
    })

    emit('update:history', [
      ...props.history,
      { role: 'user',      content: props.prompt },
      { role: 'assistant', content: data.explanation || 'Workflow updated.' },
    ])
    emit('update:prompt', '')
    emit('update:interactions', data.interactions || [])
    emit('apply', { yamlStr: data.yaml, definition: data.definition })

    // Auto-scroll chat
    setTimeout(() => {
      if (refineScroll.value) {
        refineScroll.value.scrollTop = refineScroll.value.scrollHeight
      }
    }, 50)

  } catch (e) {
    refineError.value = e.response?.data?.error ?? e.message
    if (e.response?.data?.interactions) emit('update:interactions', e.response.data.interactions)
  } finally {
    loading.value = false
  }
}

function clearHistory() {
  emit('update:history', [])
  emit('update:interactions', [])
}

// ── LLM providers ─────────────────────────────────────────────────────────────
const enabledProviders = computed(() =>
  llmStore.PROVIDERS.filter(p => {
    const cfg = llmStore.configs.find(c => c.provider === p.id)
    return cfg && cfg.enabled
  })
)

const activeProviderLabel = computed(() =>
  props.provider ? llmStore.PROVIDERS.find(p => p.id === props.provider)?.label || '' : ''
)

onMounted(() => {
  if (llmStore.configs.length === 0) llmStore.fetchAll()
})
</script>

<style scoped>
.custom-scrollbar::-webkit-scrollbar { width: 4px; }
.custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
.custom-scrollbar::-webkit-scrollbar-thumb { background: #2a2d3a; border-radius: 10px; }
.custom-scrollbar::-webkit-scrollbar-thumb:hover { background: #3f445b; }
</style>
