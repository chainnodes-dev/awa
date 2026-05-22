<template>
  <div
    class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 px-4"
    @click.self="$emit('close')"
  >
    <div class="card w-full max-w-[520px] p-6 space-y-5 shadow-2xl animate-in fade-in zoom-in duration-200">
      <div class="flex items-center justify-between">
        <h3 class="text-lg font-semibold text-slate-100 flex items-center gap-2">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-indigo-400">
            <polyline points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
          </svg>
          Run: {{ definition?.metadata?.name }}
        </h3>
        <button @click="$emit('close')" class="text-text-muted hover:text-text transition-colors">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>

      <!-- Tabs -->
      <div v-if="hasSchema" class="flex p-1 bg-surface-0 border border-slate-800/40 rounded-lg w-fit">
        <button
          @click="activeView = 'form'"
          :class="[
            'px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider rounded-md transition-all duration-200',
            activeView === 'form' ? 'bg-accent/10 text-accent' : 'text-text-muted hover:text-text-muted'
          ]"
        >
          Form
        </button>
        <button
          @click="activeView = 'json'"
          :class="[
            'px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider rounded-md transition-all duration-200',
            activeView === 'json' ? 'bg-accent/10 text-accent' : 'text-text-muted hover:text-text-muted'
          ]"
        >
          JSON
        </button>
      </div>

      <div class="space-y-1.5">
        <div v-if="activeView === 'form'" class="min-h-[200px] overflow-y-auto max-h-[400px] pr-2 scrollbar-thin scrollbar-thumb-slate-800">
          <JsonSchemaForm
             v-if="jsonSchema"
             :schema="jsonSchema"
             :initial-data="formData"
             @update:modelValue="syncFromForm"
          />
        </div>
        
        <div v-else class="space-y-1.5">
          <div class="flex items-center justify-between">
            <label class="text-xs font-semibold text-text-muted uppercase tracking-wider">Initial Blackboard Data</label>
            <span class="text-[10px] text-text-muted font-mono">JSON format</span>
          </div>
          <div class="relative group">
            <textarea
              v-model="input"
              class="input font-mono text-sm min-h-[160px] h-48 w-full resize-none bg-surface-0 border-slate-800/50 group-hover:border-slate-700/50 transition-colors py-3"
              placeholder='{ "key": "value" }'
              spellcheck="false"
              @input="syncFromJson"
            />
            <div class="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
               <button @click="formatJSON" class="text-[10px] bg-slate-800 hover:bg-slate-700 text-text-muted px-2 py-1 rounded border border-slate-700">
                 Format JSON
               </button>
            </div>
          </div>
        </div>

        <p v-if="error" class="text-red-400 text-xs font-mono bg-red-400/5 p-2 rounded border border-red-400/20">
          {{ error }}
        </p>
      </div>

      <div class="pt-2 flex gap-3 justify-end">
        <button
          class="px-4 py-2 text-sm font-medium text-text-muted hover:text-text transition-colors"
          @click="$emit('close')"
          :disabled="loading"
        >
          Cancel
        </button>
        <button
          class="btn-primary px-6 relative"
          @click="start"
          :disabled="loading"
        >
          <span :class="{ 'opacity-0': loading }">Start Workflow</span>
          <div v-if="loading" class="absolute inset-0 flex items-center justify-center">
            <svg class="animate-spin h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          </div>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, computed, reactive } from 'vue'
import { useWorkflowStore } from '@/stores/workflows'
import { convertBlackboardSchemaToJsonSchema } from '@/utils/schema'
import JsonSchemaForm from '@/components/monitor/JsonSchemaForm.vue'

const props = defineProps({
  definition: Object,
  initialData: [Object, String]
})

const emit = defineEmits(['close', 'started'])

const input   = ref('{}')
const loading = ref(false)
const error   = ref('')
const wfStore = useWorkflowStore()

const activeView = ref('form')
const formData   = reactive({})

const hasSchema = computed(() => {
  const inputs = props.definition?.inputs || []
  if (inputs.length > 0) return true

  const schema = props.definition?.blackboard?.schema || {}
  const outputs = props.definition?.outputs || []
  const outputNames = new Set(outputs.map(o => o.name))

  return Object.entries(schema).some(([key, field]) => {
    return !outputNames.has(key) && !field.is_output
  })
})

const jsonSchema = computed(() => {
  const schema = props.definition?.blackboard?.schema || {}
  const inputs = props.definition?.inputs || []
  const outputs = props.definition?.outputs || []

  const filtered = {}

  if (inputs.length > 0) {
    // Inputs first (respecting the order in definition.inputs)
    for (const port of inputs) {
      const field = schema[port.name] || { type: port.type }
      filtered[port.name] = { 
        ...field, 
        required: true,
        title: port.description || field.title || port.name
      }
    }
  } else {
    // Fallback to blackboard schema fields that are NOT outputs
    const outputNames = new Set(outputs.map(o => o.name))
    for (const [key, field] of Object.entries(schema)) {
      if (outputNames.has(key) || field.is_output) continue
      filtered[key] = field
    }
  }

  return convertBlackboardSchemaToJsonSchema(filtered, false)
})

function syncFromForm(data) {
  const currentStr = input.value.trim() || '{}'
  try {
    const current = JSON.parse(currentStr)
    // Only update if data actually changed to avoid cursor jumps in JSON view
    if (JSON.stringify(current) !== JSON.stringify(data)) {
      Object.assign(formData, data)
      input.value = JSON.stringify(data, null, 2)
    }
  } catch (e) {
    // If JSON is currently invalid, allow form updates to overwrite it
    Object.assign(formData, data)
    input.value = JSON.stringify(data, null, 2)
  }
}

function syncFromJson() {
  try {
    const parsed = JSON.parse(input.value)
    // Only update form if data actually changed
    if (JSON.stringify(parsed) !== JSON.stringify(formData)) {
      Object.keys(formData).forEach(key => delete formData[key])
      Object.assign(formData, parsed)
    }
    error.value = ''
  } catch (e) {
    // Ignore invalid JSON while typing
  }
}

onMounted(() => {
  if (props.initialData) {
    if (typeof props.initialData === 'object') {
      input.value = JSON.stringify(props.initialData, null, 2)
      Object.assign(formData, props.initialData)
    } else {
      input.value = props.initialData
      try {
        Object.assign(formData, JSON.parse(props.initialData))
      } catch (e) {}
    }
  }
  
  if (!hasSchema.value) {
    activeView.value = 'json'
  }
})

function formatJSON() {
  try {
    const parsed = JSON.parse(input.value)
    input.value = JSON.stringify(parsed, null, 2)
    error.value = ''
  } catch (e) {
    error.value = 'Cannot format: ' + e.message
  }
}

async function start() {
  if (!props.definition) return
  
  loading.value = true
  error.value   = ''
  
  try {
    let payload = {}
    if (input.value.trim()) {
      payload = JSON.parse(input.value)
    }
    
    const run = await wfStore.startRun(
      props.definition.metadata.name,
      props.definition.metadata.version,
      payload
    )
    emit('started', run)
    emit('close')
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>
