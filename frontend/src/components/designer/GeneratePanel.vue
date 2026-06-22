<template>
  <div class="flex flex-col h-full overflow-hidden bg-surface-1 relative">

    <!-- Progress Indicator Row -->
    <div class="px-4 py-2 border-b border-border bg-surface-1 flex items-center justify-between text-[11px] text-text-muted select-none shrink-0">
      <div class="flex items-center gap-1.5">
        <span class="font-bold uppercase tracking-wider text-violet-400">Step {{ props.pipelineStep < 4 ? props.pipelineStep + 1 : 4 }}/4:</span>
        <span class="font-semibold text-text truncate max-w-[120px]">{{ props.pipelineStep < 4 ? pipelineSteps[props.pipelineStep].label : 'Complete' }}</span>
      </div>
      <div class="flex gap-1 shrink-0">
        <div v-for="i in 4" :key="i"
          :class="[
            'w-1.5 h-1.5 rounded-full transition-all duration-300',
            props.pipelineStep >= i ? 'bg-emerald-500 shadow-[0_0_6px_rgba(16,185,129,0.5)]' : 'bg-border/60'
          ]"
          :title="`Step ${i}: ${pipelineSteps[i-1].label}`"
        />
      </div>
    </div>

    <!-- Main Chat Console / History Stream -->
    <div class="flex-1 flex flex-col min-h-0">
      <div 
        ref="refineScroll"
        class="flex-1 overflow-y-auto p-4 space-y-4 custom-scrollbar bg-surface-0/20"
      >
        <!-- Welcome empty state -->
        <div v-if="!chatMessages.length" class="h-full flex flex-col items-center justify-center text-center py-8 px-4 space-y-3 select-none">
          <div class="w-10 h-10 rounded-xl bg-violet-500/10 border border-violet-500/20 flex items-center justify-center text-violet-400">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
              <path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 1 1-7.6-13.5 8.38 8.38 0 0 1 3.8.9L21 3z"/>
            </svg>
          </div>
          <p class="text-xs font-semibold text-text">AI Generation & Design Chat</p>
          <p class="text-[10px] text-text-muted leading-relaxed">
            Type your process description in the input box below to trigger Auto Pilot, or write modifications to refine your visual layout.
          </p>
        </div>

        <!-- Chat history rendering -->
        <div v-else v-for="(item, i) in chatMessages" :key="i" class="space-y-1">
          <!-- Monospace System Message Pill -->
          <div v-if="item.isSystem" class="flex justify-center w-full my-2">
            <div class="px-2.5 py-0.5 rounded-full border border-border bg-surface-2 text-[9px] font-mono text-text-muted flex items-center gap-1 shadow-sm">
              <span :class="['inline-block w-1.5 h-1.5 rounded-full', item.content.startsWith('❌') ? 'bg-red-500' : 'bg-emerald-400']"></span>
              {{ item.content }}
            </div>
          </div>

          <!-- Normal Chat Bubble -->
          <div v-else :class="['flex flex-col gap-1', item.role === 'user' ? 'items-end' : 'items-start']">
            <span class="text-[8px] font-bold uppercase tracking-tighter text-text-muted px-1">
              {{ item.isInitialDescription ? 'Process Description' : (item.role === 'user' ? 'You' : 'Designer') }}
            </span>
            <div 
              :class="[
                'max-w-[90%] px-3 py-2 rounded-xl text-xs leading-relaxed border',
                item.isInitialDescription
                  ? 'bg-violet-950/20 text-text border-violet-500/30 rounded-tr-none'
                  : item.role === 'user' 
                    ? 'bg-indigo-600/10 text-text border-indigo-500/25 rounded-tr-none' 
                    : 'bg-surface-2 text-text border-border/50 rounded-tl-none'
              ]"
            >
              <div class="whitespace-pre-wrap">{{ item.content }}</div>
            </div>
          </div>
        </div>

        <!-- Inline Loader at the bottom of the chat history -->
        <div v-if="loading" class="flex flex-col gap-1 items-start mt-2">
          <span class="text-[8px] font-bold uppercase tracking-tighter text-text-muted px-1">
            Designer
          </span>
          <div class="max-w-[90%] px-3 py-2 rounded-xl text-xs leading-relaxed border bg-surface-2 text-text border-border/50 rounded-tl-none flex items-center gap-2.5">
            <div class="w-3.5 h-3.5 border-2 border-violet-500/20 border-t-violet-500 rounded-full animate-spin shrink-0"></div>
            <div class="flex flex-col min-w-0">
              <span class="text-[11px] font-semibold text-text truncate">{{ loadingLabel }}</span>
              <span class="text-[9px] text-text-muted truncate mt-0.5">{{ loadingSubLabel }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Pipeline warning warning -->
      <div v-if="props.pipelineWarning && props.pipelineStep < 4" class="px-4 py-1.5 bg-amber-500/10 border-t border-amber-500/20 text-amber-500 text-[10px] select-none shrink-0">
        ⚠ Note: {{ props.pipelineWarning }}
      </div>

      <!-- Pipeline or Refine error -->
      <div v-if="pipelineError || refineError" class="px-4 py-2 bg-red-500/10 border-t border-red-500/20 text-red-400 text-[10px] font-mono shrink-0">
        {{ pipelineError || refineError }}
      </div>

      <!-- Input Resizer Divider -->
      <div 
        class="h-1 shrink-0 bg-border hover:bg-violet-500/80 active:bg-violet-500 cursor-row-resize transition-colors select-none z-10"
        @mousedown="startInputResize"
      />

      <!-- Input Console Area -->
      <div 
        :style="{ height: inputAreaHeight + 'px' }"
        class="p-3 bg-surface-1/50 border-t border-border/50 shrink-0 flex flex-col gap-2 min-h-[150px]"
      >
        <!-- Quick-action Step Pills -->
        <div class="flex items-center gap-1.5 overflow-x-auto no-scrollbar shrink-0">
          <button 
            v-for="(step, idx) in pipelineSteps" 
            :key="idx"
            @click="runStep(idx)"
            :disabled="loading || (idx > 0 && !props.yamlSource)"
            :class="[
              'p-0.5 px-2 text-[9px] rounded-md border transition-all outline-none disabled:opacity-30 disabled:cursor-not-allowed shrink-0 font-semibold',
              idx < props.pipelineStep 
                ? 'bg-green-600/10 border-green-500/30 text-green-400 hover:bg-green-600/20 hover:border-green-500/50' 
                : idx === props.pipelineStep 
                  ? 'bg-amber-600/15 border-amber-500/50 text-amber-500 hover:bg-amber-600/25 hover:border-amber-500/70 font-bold shadow-[0_0_8px_rgba(245,158,11,0.15)]' 
                  : 'bg-surface-2 border-border/60 hover:bg-violet-600/20 hover:border-violet-500/40 text-text-muted hover:text-violet-300'
            ]"
          >
            <span v-if="idx < props.pipelineStep" class="mr-0.5 font-bold">✓</span>
            {{ idx + 1 }}: {{ step.label }}
          </button>
        </div>
        
        <!-- Textarea input -->
        <div class="flex-1 min-h-0">
          <textarea
            :value="prompt"
            @input="emit('update:prompt', $event.target.value)"
            @keydown.enter.exact.prevent="handleSubmit"
            class="w-full h-full bg-surface-0 border border-border/40 rounded-xl px-3 py-2 text-xs resize-none focus:outline-none focus:ring-1 focus:ring-indigo-500/50 focus:border-indigo-500/50 transition-all leading-relaxed font-sans"
            placeholder="Type process to run Auto Pilot, or ask for changes to refine..."
            :disabled="loading"
          />
        </div>

        <!-- Toolbar row: File Upload | Model Select | Auto Pilot | Reset | Send Button -->
        <div class="flex items-center justify-between text-[11px] select-none gap-2 shrink-0 mt-1">
          <div class="flex items-center gap-2 min-w-0 flex-1">
            <!-- File Upload -->
            <button 
              @click="triggerFileUpload" 
              :disabled="loading"
              class="p-1 rounded bg-transparent hover:bg-surface-2 text-text-muted hover:text-text transition-all border-0 outline-none cursor-pointer disabled:opacity-50 flex items-center justify-center shrink-0"
              title="Upload text file"
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
              </svg>
            </button>
            <input ref="fileInput" type="file" accept=".txt,.md" class="hidden" @change="handleFileUpload"/>

            <!-- Model Selection -->
            <select
              :value="provider"
              @change="emit('update:provider', $event.target.value)"
              :disabled="loading"
              class="bg-surface-0 border border-border text-[10px] text-text rounded px-1.5 py-0.5 outline-none cursor-pointer focus:border-violet-500 transition-colors font-sans truncate w-full disabled:opacity-50"
            >
              <option value="">(Default)</option>
              <option v-for="p in enabledProviders" :key="p.id" :value="p.id">{{ p.label }}</option>
            </select>
          </div>

          <!-- Center: Auto Pilot Checkbox -->
          <label class="flex items-center gap-1.5 text-[10px] text-text-muted hover:text-text cursor-pointer select-none font-semibold shrink-0">
            <input 
              type="checkbox" 
              v-model="autoAdvance"
              :disabled="loading"
              class="accent-indigo-500 h-3 w-3 rounded border-border bg-surface-0 focus:ring-violet-500/50 disabled:opacity-50"
            />
            Auto Pilot
          </label>

          <!-- Right: Reset and Send Buttons -->
          <div class="flex items-center gap-2 shrink-0">
            <button 
              v-if="chatMessages.length"
              @click="clearHistory"
              :disabled="loading"
              class="text-[9px] text-text-muted hover:text-red-400 font-bold uppercase tracking-wider underline decoration-dotted underline-offset-1 border-0 bg-transparent cursor-pointer disabled:opacity-50"
            >
              Reset
            </button>
            
            <button 
              @click="handleSubmit"
              :disabled="loading || !prompt.trim()"
              class="h-7 w-7 rounded-lg bg-indigo-600 text-white hover:bg-indigo-500 transition-all disabled:opacity-30 shadow-md shadow-indigo-900/10 border-0 outline-none flex items-center justify-center cursor-pointer shrink-0"
              :class="{ 'cursor-not-allowed': loading }"
              title="Send description / refinement request"
            >
              <svg v-if="loading" class="animate-spin text-white" width="12" height="12" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
              </svg>
              <svg v-else width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/>
              </svg>
            </button>
          </div>
        </div>

      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { api, useAuthStore } from '@/stores/auth'
import { useLLMStore } from '@/stores/llm'

const authStore = useAuthStore()
const llmStore = useLLMStore()

const refineScroll = ref(null)
const autoAdvance = ref(true)
const fileInput = ref(null)

// ── Input Area Resizing ──
const inputAreaHeight = ref(280)
let isDraggingInput = false
let startInputY = 0
let startInputHeight = 0

function startInputResize(e) {
  isDraggingInput = true
  startInputY = e.clientY
  startInputHeight = inputAreaHeight.value
  document.addEventListener('mousemove', doInputResize)
  document.addEventListener('mouseup', stopInputResize)
  document.body.style.cursor = 'row-resize'
  document.body.style.userSelect = 'none'
}

function doInputResize(e) {
  if (!isDraggingInput) return
  const deltaY = startInputY - e.clientY
  let newHeight = startInputHeight + deltaY
  
  // Clamp height between 150px and 550px
  if (newHeight < 150) newHeight = 150
  if (newHeight > 550) newHeight = 550
  
  inputAreaHeight.value = newHeight
}

function stopInputResize() {
  isDraggingInput = false
  document.removeEventListener('mousemove', doInputResize)
  document.removeEventListener('mouseup', stopInputResize)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}

function triggerFileUpload() {
  fileInput.value?.click()
}

function handleFileUpload(event) {
  const file = event.target.files?.[0]
  if (!file) return

  const reader = new FileReader()
  reader.onload = (e) => {
    const content = e.target?.result
    if (typeof content === 'string') {
      emit('update:prompt', content)
    }
  }
  reader.readAsText(file)
  
  event.target.value = ''
}

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

// ── Pipeline gears ──
const stepToGear = ['decompose', 'categorise', 'wire', 'implement']

const pipelineSteps = [
  { label: 'Decompose',  endpoint: '/designer/pipeline/decompose',  description: 'Identify all states and add a description to every node. The canvas will show a skeleton workflow.' },
  { label: 'Categorise', endpoint: '/designer/pipeline/categorise', description: 'Assign the correct Chain Nodes type to every node (code, script, hitl, wait, etc.). No runnable logic yet.' },
  { label: 'Wire',       endpoint: '/designer/pipeline/wire',       description: 'Define all transitions, guards, and the blackboard schema — the shared contract for node implementations.' },
  { label: 'Implement',  endpoint: '/designer/pipeline/implement',  description: 'Generate final code for every node in parallel using specialised prompts. Produces a runnable workflow.' },
]

const pipelineError = ref('')
const refineError     = ref('')

// ── Loading ──
const loading         = ref(false)
const loadingLabel    = ref('AI is thinking...')
const loadingSubLabel = ref('')

// ── Merged Chat Messages computed property ──
const chatMessages = computed(() => {
  const msgs = []
  if (props.description && props.description.trim()) {
    msgs.push({
      role: 'user',
      content: props.description,
      isInitialDescription: true
    })
  }
  props.history.forEach(item => {
    // Avoid duplicating the initial description
    if (item.role === 'user' && item.content === props.description) {
      return
    }
    msgs.push(item)
  })
  return msgs
})

// Watch chat messages and loading state to auto-scroll
watch([() => chatMessages.value, loading], async () => {
  await nextTick()
  if (refineScroll.value) {
    refineScroll.value.scrollTop = refineScroll.value.scrollHeight
  }
}, { deep: true, immediate: true })

// ── Unified submission (Auto Pilot on description, Refinement on subsequent) ──
async function handleSubmit() {
  const text = props.prompt.trim()
  if (!text) return

  if (!props.description || !props.description.trim()) {
    emit('update:description', text)
    emit('update:prompt', '')
    nextTick(async () => {
      await runAutoPilot()
    })
  } else {
    await applyRefinement()
  }
}

// ── Trigger Auto Pilot execution flow ──
async function runAutoPilot() {
  const text = props.prompt.trim()
  if (text && (!props.description || !props.description.trim())) {
    emit('update:description', text)
    emit('update:prompt', '')
  }
  resetPipeline()
  nextTick(async () => {
    await runStep(0)
  })
}

function resetPipeline() {
  emit('update:pipelineStep', 0)
  emit('update:pipelineWarning', '')
  pipelineError.value = ''
  refineError.value = ''
}

// ── Pipeline actions ──
async function runStep(index) {
  const step = pipelineSteps[index]
  if (!step) return

  if (index === 0 && !props.description.trim()) {
    pipelineError.value = "Process description is required."
    return
  }
  if (index > 0 && (!props.description.trim() || !props.yamlSource.trim())) {
    pipelineError.value = "Existing blueprint definition is required. Run earlier steps first."
    return
  }

  loading.value = true
  loadingLabel.value = `Running Step ${index + 1}: ${step.label}…`
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

    // Log the step progression in the chat history
    const feedback = parseLLMFeedback(index, data.interactions)
    const newHistory = [
      ...props.history,
      { role: 'assistant', content: `⚙ [Step ${index + 1}/4] Complete: ${step.label}. Applied layout updates.`, isSystem: true }
    ]
    if (feedback) {
      newHistory.push({ role: 'assistant', content: feedback })
    }
    emit('update:history', newHistory)

    emit('update:interactions', data.interactions || [])
    emit('apply', { yamlStr: data.yaml, definition: data.definition })
    
    if (index === props.pipelineStep) {
      emit('update:pipelineStep', index + 1)
      
      // Auto-run next step if enabled
      if (autoAdvance.value && index + 1 < pipelineSteps.length) {
        nextTick(() => {
          setTimeout(() => {
            if (autoAdvance.value && !pipelineError.value && loading.value === false) {
              runStep(index + 1)
            }
          }, 1200)
        })
      }
    }

  } catch (e) {
    const errMsg = e.response?.data?.error ?? e.message
    pipelineError.value = errMsg
    emit('update:history', [
      ...props.history,
      { role: 'assistant', content: `❌ [Step ${index + 1}/4] Failed: ${step.label}. Error: ${errMsg}`, isSystem: true }
    ])
    if (e.response?.data?.interactions) emit('update:interactions', e.response.data.interactions)
  } finally {
    loading.value = false
  }
}

// ── Refine actions ──
async function applyRefinement() {
  if (!props.prompt.trim()) return

  const promptText = props.prompt
  loading.value = true
  loadingLabel.value = 'Applying refinement…'
  loadingSubLabel.value = promptText.slice(0, 80)
  refineError.value = ''

  try {
    const { data } = await api.post('/designer/process/analyse', {
      description:   promptText,
      history:       props.history,
      existing_yaml: props.yamlSource || '',
      provider:      props.provider || undefined,
    })

    emit('update:history', [
      ...props.history,
      { role: 'user',      content: promptText },
      { role: 'assistant', content: data.explanation || 'Workflow updated.' },
    ])
    emit('update:prompt', '')
    emit('update:interactions', data.interactions || [])
    emit('apply', { yamlStr: data.yaml, definition: data.definition })

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
  emit('update:description', '')
  resetPipeline()
}

// ── LLM providers ──
const enabledProviders = computed(() =>
  llmStore.PROVIDERS.filter(p => {
    const cfg = llmStore.configs.find(c => c.provider === p.id)
    return cfg && cfg.enabled
  })
)

// ── Feedback Parsers ──
function formatDecompose(obj) {
  let text = `Decomposed Workflow: ${obj.name || ''}\n`
  if (obj.abstract) text += `Summary: ${obj.abstract}\n`
  if (obj.detailed_description) text += `\nDescription:\n${obj.detailed_description}\n`
  
  if (Array.isArray(obj.states) && obj.states.length) {
    text += `\nIdentified States:\n`
    obj.states.forEach(s => {
      text += `• ${s.name}: ${s.instructions || ''}\n`
      if (s.data_contract) {
        const reads = s.data_contract.reads?.map(r => r.name).join(', ') || 'none'
        const writes = s.data_contract.writes?.map(w => w.name).join(', ') || 'none'
        text += `  (Reads: ${reads} | Writes: ${writes})\n`
      }
    })
  }
  return text
}

function formatCategorise(obj) {
  let text = `Categorized State Nodes:\n\n`
  if (Array.isArray(obj.states) && obj.states.length) {
    obj.states.forEach(s => {
      const typeStr = s.type ? `[${s.type.toUpperCase()}]` : ''
      const agent = s.agent_name ? ` (Agent: ${s.agent_name})` : ''
      const timeout = s.timeout ? ` [Timeout: ${s.timeout}]` : ''
      text += `• ${s.name} ${typeStr}${agent}${timeout}\n`
      if (s.instructions) {
        text += `  Instructions: ${s.instructions}\n`
      }
    })
  }
  return text
}

function formatWire(obj) {
  let text = `Wired Transitions & Guards:\n\n`
  if (Array.isArray(obj.transitions) && obj.transitions.length) {
    obj.transitions.forEach(t => {
      const guard = t.guard ? ` (if ${t.guard})` : ''
      text += `• ${t.from} ➔ ${t.to} [Trigger: ${t.trigger}]${guard}\n`
    })
  }
  return text
}

function formatImplement(obj) {
  let text = `Finalized Workflow Contract:\n\n`
  if (obj.metadata && obj.metadata.detailed_description) {
    text += `${obj.metadata.detailed_description}\n\n`
  }
  if (obj.blackboard_schema) {
    text += `Shared Blackboard Schema:\n`
    Object.entries(obj.blackboard_schema).forEach(([k, v]) => {
      const req = v.required ? ' (required)' : ''
      text += `• ${k} (${v.type || 'any'})${req}\n`
    })
  }
  if (Array.isArray(obj.inputs) && obj.inputs.length) {
    text += `\nInputs:\n`
    obj.inputs.forEach(i => {
      text += `• ${i.name} (${i.type || 'any'}): ${i.description || ''}\n`
    })
  }
  if (Array.isArray(obj.outputs) && obj.outputs.length) {
    text += `\nOutputs:\n`
    obj.outputs.forEach(o => {
      text += `• ${o.name} (${o.type || 'any'}): ${o.description || ''}\n`
    })
  }
  return text
}

function parseLLMFeedback(index, interactions) {
  if (!interactions || !interactions.length) return null
  const assistantMsg = [...interactions].reverse().find(m => m.role === 'assistant')
  if (!assistantMsg || !assistantMsg.content) return null

  const content = assistantMsg.content
  let obj = null
  try {
    obj = JSON.parse(content)
  } catch (e) {
    const match = content.match(/```(?:json)?\s*([\s\S]*?)\s*```/)
    if (match) {
      try {
        obj = JSON.parse(match[1])
      } catch (e2) {}
    }
    if (!obj) {
      const start = content.indexOf('{')
      const end = content.lastIndexOf('}')
      if (start !== -1 && end !== -1 && end > start) {
        try {
          obj = JSON.parse(content.slice(start, end + 1))
        } catch (e3) {}
      }
    }
  }

  if (!obj) return null

  try {
    if (index === 0) return formatDecompose(obj)
    if (index === 1) return formatCategorise(obj)
    if (index === 2) return formatWire(obj)
    if (index === 3) return formatImplement(obj)
  } catch (e) {
    console.error("Failed to format step feedback", e)
  }
  return null
}

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
