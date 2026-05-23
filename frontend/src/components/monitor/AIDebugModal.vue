<template>
  <div class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-6">
    <div class="card w-full max-w-4xl max-h-[90vh] flex flex-col shadow-2xl border-violet-500/20">
      <header class="px-6 py-4 border-b border-border flex items-center justify-between shrink-0 bg-surface-1">
        <div class="flex items-center gap-2">
          <div class="p-2 rounded-lg bg-violet-500/10 text-violet-400">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z"/>
              <path d="M12 8v4"/><path d="M12 16h.01"/>
            </svg>
          </div>
          <div>
            <h3 class="text-base font-bold text-text">AI Diagnostic Assistant</h3>
            <p class="text-xs text-text-muted mt-0.5">Analysing failure and proposing corrections</p>
          </div>
        </div>
        <button @click="$emit('close')" class="btn-ghost p-2 text-text-muted hover:text-text transition-colors">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </header>

      <div class="flex-1 overflow-y-auto p-6 space-y-6 bg-surface-0">
        <!-- Initial Loading State -->
        <div v-if="loading && !proposal" class="flex flex-col items-center justify-center py-20 space-y-4">
          <div class="w-12 h-12 rounded-full border-2 border-violet-500/20 border-t-violet-500 animate-spin"/>
          <div class="text-center">
            <p class="text-sm font-medium text-text">Consulting Diagnostic Agent...</p>
            <p class="text-xs text-text-muted mt-1">Reviewing workflow YAML and blackboard state</p>
          </div>
        </div>

        <!-- Error State -->
        <div v-else-if="error" class="bg-red-500/10 border border-red-500/30 rounded-lg p-4 flex items-start gap-3">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-red-400 shrink-0 mt-0.5">
            <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
          <div>
            <p class="text-xs font-bold text-red-400 uppercase tracking-widest mb-1">Diagnostic Failed</p>
            <p class="text-xs text-red-400/80 font-mono">{{ error }}</p>
            <button @click="runDiagnostic" class="btn-ghost text-[10px] font-bold text-violet-400 uppercase tracking-widest mt-3 p-0 hover:bg-transparent">Retry Diagnostic</button>
          </div>
        </div>

        <!-- Proposal State -->
        <div v-if="proposal" class="space-y-6 animate-in fade-in slide-in-from-bottom-4 duration-500">
          <!-- Explanation -->
          <div class="bg-violet-500/5 border border-violet-500/20 rounded-xl p-5 space-y-3 shadow-sm">
            <div class="flex items-center gap-2">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#a78bfa" stroke-width="2.5">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
              </svg>
              <h4 class="text-[10px] font-bold text-violet-400 uppercase tracking-widest">Diagnostic Report</h4>
            </div>
            <p class="text-sm text-text leading-relaxed">{{ proposal.explanation }}</p>
          </div>

          <!-- Comparison / Diff View -->
          <div class="grid grid-cols-2 gap-4 h-[400px]">
            <div class="flex flex-col min-w-0">
              <div class="flex items-center justify-between mb-2 px-1">
                <span class="text-[10px] font-bold text-text-muted uppercase tracking-widest">Current (Broken)</span>
              </div>
              <div class="flex-1 bg-surface-1 rounded-lg border border-border overflow-hidden grayscale opacity-60">
                <CodeEditor
                  :model-value="originalYAML"
                  language="yaml"
                  readonly
                  height="100%"
                  :show-badge="false"
                />
              </div>
            </div>
            <div class="flex flex-col min-w-0">
              <div class="flex items-center justify-between mb-2 px-1">
                <span class="text-[10px] font-bold text-emerald-400 uppercase tracking-widest">AI Correction</span>
              </div>
              <div class="flex-1 bg-surface-1 rounded-lg border border-emerald-500/30 overflow-hidden shadow-lg shadow-emerald-500/5">
                <CodeEditor
                  :model-value="proposal.yaml"
                  language="yaml"
                  readonly
                  height="100%"
                  :show-badge="false"
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      <footer class="px-6 py-4 border-t border-border bg-surface-1 flex items-center justify-between shrink-0">
        <div class="flex items-center gap-2">
          <button @click="$emit('close')" class="btn-ghost text-xs px-4 py-2">Keep Current</button>
        </div>
        <div class="flex items-center gap-2">
          <button 
            v-if="proposal"
            @click="applyFix" 
            class="btn bg-emerald-600 hover:bg-emerald-500 text-white text-xs px-6 py-2 shadow-lg shadow-emerald-600/20"
          >
            Apply Fix & Open Designer
          </button>
        </div>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/stores/auth'
import CodeEditor from '@/components/designer/CodeEditor.vue'

const props = defineProps({
  runId: { type: String, required: true },
  originalYAML: { type: String, required: true },
  failedNode: { type: String, required: true },
  errorMessage: { type: String, required: true },
  blackboard: { type: Object, default: () => ({}) }
})

const emit = defineEmits(['close'])
const router = useRouter()

const loading = ref(true)
const error = ref('')
const proposal = ref(null)

async function runDiagnostic() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.post('/designer/debug', {
      workflow_yaml: props.originalYAML,
      failed_node_name: props.failedNode,
      error_message: props.errorMessage,
      blackboard_snapshot: props.blackboard
    })
    proposal.value = data
  } catch (e) {
    error.value = e.response?.data?.error ?? e.message
  } finally {
    loading.value = false
  }
}

function applyFix() {
  if (!proposal.value) return
  // We store the fix in session storage so the designer can pick it up
  sessionStorage.setItem('chainnodes_ai_fix', JSON.stringify({
    yaml: proposal.value.yaml,
    explanation: proposal.value.explanation
  }))
  
  // Close and navigate to designer
  emit('close')
  router.push('/designer?fix=true')
}

onMounted(runDiagnostic)
</script>
