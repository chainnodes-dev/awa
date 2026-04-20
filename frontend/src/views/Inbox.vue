<template>
  <div class="flex h-full overflow-hidden bg-surface-0">
    <!-- ── Task List ─────────────────────────────────────────────────── -->
    <aside class="w-80 border-r border-border flex flex-col bg-surface-1 shrink-0">
      <div class="px-4 py-3 border-b border-border flex items-center justify-between">
        <h2 class="text-xs font-semibold text-slate-400 uppercase tracking-widest">Task Queue</h2>
        <span class="badge bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
          {{ pendingTasks.length }}
        </span>
      </div>

      <div class="flex-1 overflow-y-auto p-2 space-y-2">
        <div
          v-for="task in pendingTasks"
          :key="task.id"
          @click="selectedTaskId = task.id"
          :class="[
            'cursor-pointer p-3 rounded-xl border transition-all duration-200 group',
            selectedTaskId === task.id
              ? 'bg-accent/10 border-accent/40 shadow-lg shadow-accent/5'
              : 'border-transparent hover:bg-white/5'
          ]"
        >
          <div class="flex items-start justify-between mb-1.5">
            <span class="text-xs font-bold text-slate-200 truncate pr-2">{{ task.workflow_name || 'Workflow' }}</span>
            <span class="text-[9px] font-mono text-slate-600 bg-surface-0 px-1.5 py-0.5 rounded border border-border">
              HITL
            </span>
          </div>
          <div class="text-[10px] text-slate-500 font-medium mb-2">{{ task.state_name }}</div>
          
          <div class="flex items-center gap-2 text-[9px] text-slate-600">
            <span class="flex items-center gap-1">
              <svg width="8" height="8" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
              </svg>
              {{ formatTime(task.created_at) }}
            </span>
            <span v-if="task.assignee" class="flex items-center gap-1">
              <svg width="8" height="8" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
              </svg>
              {{ task.assignee }}
            </span>
          </div>
        </div>

        <div v-if="!pendingTasks.length" class="flex flex-col items-center justify-center h-full py-12 text-center opacity-40">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="mb-2">
            <path d="M22 13h-4l-3 9L9 3l-3 9H2"/>
          </svg>
          <p class="text-xs">No pending tasks</p>
        </div>
      </div>
    </aside>

    <!-- ── Task Detail ───────────────────────────────────────────────── -->
    <main class="flex-1 flex flex-col min-w-0 bg-surface-0 relative">
      <template v-if="selectedTask">
        <header class="px-6 py-4 border-b border-border flex items-center justify-between shrink-0 bg-surface-1/50 backdrop-blur">
          <div>
            <div class="flex items-center gap-2 mb-1">
              <span class="text-sm font-semibold text-slate-200">{{ selectedTask.workflow_name }}</span>
              <span class="text-xs text-slate-600 font-mono">/ {{ selectedTask.run_id.slice(0, 8) }}</span>
            </div>
            <p class="text-[10px] text-slate-500 font-mono tracking-tight">{{ selectedTask.state_name }} state</p>
          </div>

          <div class="flex items-center gap-3">
            <RouterLink :to="`/monitor/${selectedTask.run_id}`" class="btn-ghost text-xs">
              View Run
            </RouterLink>
            <div class="w-px h-4 bg-border"/>
            <button @click="resolve('rejected')" class="btn-danger text-xs px-4">Reject</button>
            <button @click="resolve('approved')" class="btn-primary text-xs px-6">Approve</button>
          </div>
        </header>

        <div class="flex-1 overflow-hidden flex divide-x divide-border">
          <!-- Main Form Section -->
          <div class="flex-1 overflow-y-auto p-6 space-y-8">
            <section class="max-w-xl">
              <div class="mb-6">
                <h3 class="text-xs font-bold text-slate-400 uppercase tracking-widest mb-1">Task Resolution</h3>
                <p class="text-xs text-slate-500">Provide the required information to continue the workflow execution.</p>
              </div>

              <JsonSchemaForm
                v-if="selectedTask.form_schema"
                ref="hitlForm"
                :schema="selectedTask.form_schema"
                :initial-data="selectedTask.blackboard"
              />
              <div v-else class="py-12 flex flex-col items-center text-center border border-dashed border-border rounded-xl">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-slate-700 mb-2">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
                </svg>
                <p class="text-xs text-slate-600">No specific input structure defined for this node.</p>
              </div>
            </section>
          </div>

          <!-- Blackboard Tweak Section -->
          <div class="w-96 flex flex-col overflow-hidden bg-surface-1/30">
            <div class="px-4 py-3 border-b border-border flex items-center justify-between shrink-0">
               <h3 class="text-xs font-bold text-slate-400 uppercase tracking-widest">Advanced Tweaks</h3>
               <button 
                 @click="showBlackboard = !showBlackboard" 
                 class="text-[10px] text-indigo-400 hover:text-indigo-300 transition-colors"
                >
                 {{ showBlackboard ? 'Hide Context' : 'Edit Context' }}
               </button>
            </div>

            <div v-if="showBlackboard && blackboardSchema" class="px-4 py-2 bg-surface-0 border-b border-border flex gap-2">
               <button
                 @click="tweakView = 'form'"
                 :class="[
                   'px-2 py-0.5 text-[9px] font-bold uppercase rounded transition-colors',
                   tweakView === 'form' ? 'bg-accent/20 text-accent' : 'text-slate-600 hover:text-slate-500'
                 ]"
               > Form </button>
               <button
                 @click="tweakView = 'json'"
                 :class="[
                   'px-2 py-0.5 text-[9px] font-bold uppercase rounded transition-colors',
                   tweakView === 'json' ? 'bg-accent/20 text-accent' : 'text-slate-600 hover:text-slate-500'
                 ]"
               > JSON </button>
            </div>
            
            <div v-if="showBlackboard" class="flex-1 overflow-hidden flex flex-col p-3 pt-0">
              <p class="text-[10px] text-slate-600 mb-3 mt-3 leading-relaxed">
                Directly modify any field in the workflow's blackboard context. Be careful with manual overrides.
              </p>
              
              <div v-if="tweakView === 'form' && blackboardSchema" class="flex-1 overflow-y-auto pr-2 scrollbar-thin scrollbar-thumb-slate-800">
                <JsonSchemaForm
                  :schema="blackboardSchema"
                  :initial-data="editedBlackboard"
                  @update:modelValue="syncFromTweakForm"
                />
              </div>
              <BlackboardEditor 
                v-else
                v-model="editedBlackboard"
                class="flex-1"
                @update:modelValue="syncFromTweakJson"
              />
            </div>
            <div v-else class="flex-1 flex flex-col items-center justify-center opacity-30 p-6 text-center">
               <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="mb-2">
                 <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/>
               </svg>
               <span class="text-[10px]">Blackboard editor hidden</span>
            </div>
          </div>
        </div>
      </template>

      <div v-else class="flex-1 flex flex-col items-center justify-center text-center opacity-30">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <rect x="4" y="4" width="16" height="16" rx="2" ry="2"/><rect x="9" y="9" width="6" height="6"/><line x1="9" y1="1" x2="9" y2="4"/><line x1="15" y1="1" x2="15" y2="4"/><line x1="9" y1="20" x2="9" y2="23"/><line x1="15" y1="20" x2="15" y2="23"/><line x1="20" y1="9" x2="23" y2="9"/><line x1="20" y1="15" x2="23" y2="15"/><line x1="1" y1="9" x2="4" y2="9"/><line x1="1" y1="15" x2="4" y2="15"/>
        </svg>
        <p class="mt-4 text-sm font-medium">Select a task from the queue to resolve it</p>
      </div>
      
      <!-- Error Banner -->
      <div v-if="resolveError" class="absolute bottom-4 left-1/2 -translate-x-1/2 max-w-lg w-full px-6 py-3 rounded-xl bg-red-500/20 border border-red-500/40 text-red-200 text-xs text-center shadow-2xl backdrop-blur-md flex items-center gap-3">
        <span class="flex-1">{{ resolveError }}</span>
        <button @click="resolveError = ''" class="text-white hover:opacity-100 opacity-60">✕</button>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { useExecutionStore } from '@/stores/execution'
import { convertAsmSchemaToJsonSchema } from '@/utils/schema'
import JsonSchemaForm from '@/components/monitor/JsonSchemaForm.vue'
import BlackboardEditor from '@/components/monitor/BlackboardEditor.vue'

const execStore = useExecutionStore()
const pendingTasks = computed(() => execStore.pendingHITL)
const selectedTaskId = ref(null)
const selectedTask = computed(() => pendingTasks.value.find(t => t.id === selectedTaskId.value))

const editedBlackboard = ref({})
const showBlackboard = ref(false)
const tweakView = ref('form')
const resolveError = ref('')
const hitlForm = ref(null)

const blackboardSchema = computed(() => {
  return convertAsmSchemaToJsonSchema(selectedTask.value?.blackboard_schema)
})

// When selecting a task, clone its blackboard for editing
watch(selectedTask, (task) => {
  if (task) {
    editedBlackboard.value = JSON.parse(JSON.stringify(task.blackboard || {}))
    // Default to form if schema exists, otherwise JSON
    tweakView.value = task.blackboard_schema && Object.keys(task.blackboard_schema).length > 0 ? 'form' : 'json'
  }
}, { immediate: true })

function syncFromTweakForm(data) {
  // Only update if data actually changed to avoid loop
  if (JSON.stringify(editedBlackboard.value) !== JSON.stringify(data)) {
    editedBlackboard.value = { ...data }
  }
}

function syncFromTweakJson(data) {
  // The multi_replace check logic isn't strictly needed here as editedBlackboard 
  // is the v-model for the editor, but we keep it for consistency.
}

onMounted(() => {
  execStore.fetchPendingHITL()
})

async function resolve(resolution) {
  if (!selectedTask.value) return
  resolveError.value = ''
  
  try {
    // Merge form inputs with manual blackboard tweaks
    // Form inputs take precedence if there's a conflict
    const formPayload = hitlForm.value ? hitlForm.value.formData : {}
    const finalPayload = {
      ...editedBlackboard.value,
      ...formPayload
    }

    await execStore.resolveHITL(selectedTask.value.run_id, resolution, 'user', finalPayload)
    
    // Select next task if available
    const currentIndex = pendingTasks.value.findIndex(t => t.id === selectedTaskId.value)
    if (pendingTasks.value.length > 1) {
      const next = pendingTasks.value[currentIndex + 1] || pendingTasks.value[currentIndex - 1]
      selectedTaskId.value = next?.id
    } else {
      selectedTaskId.value = null
    }
  } catch (err) {
    resolveError.value = err.response?.data?.error ?? err.message ?? 'Unknown error'
  }
}

function formatTime(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  return d.toLocaleTimeString('en', { hour12: false, hour: '2-digit', minute: '2-digit' })
}
</script>

<style scoped>
.badge {
  @apply px-1.5 py-0.5 rounded-full text-[10px] font-bold;
}
</style>
