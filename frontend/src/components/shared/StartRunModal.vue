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
        <button @click="$emit('close')" class="text-slate-500 hover:text-slate-300 transition-colors">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>

      <div class="space-y-1.5">
        <div class="flex items-center justify-between">
          <label class="text-xs font-semibold text-slate-500 uppercase tracking-wider">Initial Blackboard Data</label>
          <span class="text-[10px] text-slate-600 font-mono">JSON format</span>
        </div>
        <div class="relative group">
          <textarea
            v-model="input"
            class="input font-mono text-sm min-h-[160px] h-48 w-full resize-none bg-surface-0 border-slate-800/50 group-hover:border-slate-700/50 transition-colors py-3"
            placeholder='{ "key": "value" }'
            spellcheck="false"
          />
          <div class="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
             <button @click="formatJSON" class="text-[10px] bg-slate-800 hover:bg-slate-700 text-slate-400 px-2 py-1 rounded border border-slate-700">
               Format JSON
             </button>
          </div>
        </div>
        <p v-if="error" class="text-red-400 text-xs font-mono bg-red-400/5 p-2 rounded border border-red-400/20">
          {{ error }}
        </p>
      </div>

      <div class="pt-2 flex gap-3 justify-end">
        <button
          class="px-4 py-2 text-sm font-medium text-slate-400 hover:text-slate-200 transition-colors"
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
import { ref, watch, onMounted } from 'vue'
import { useWorkflowStore } from '@/stores/workflows'

const props = defineProps({
  definition: Object,
  initialData: [Object, String]
})

const emit = defineEmits(['close', 'started'])

const input   = ref('{}')
const loading = ref(false)
const error   = ref('')
const wfStore = useWorkflowStore()

onMounted(() => {
  if (props.initialData) {
    if (typeof props.initialData === 'object') {
      input.value = JSON.stringify(props.initialData, null, 2)
    } else {
      input.value = props.initialData
    }
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
