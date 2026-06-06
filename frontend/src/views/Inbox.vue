<template>
  <div class="flex h-full overflow-hidden bg-surface-0">
    <!-- ── Task List ─────────────────────────────────────────────────── -->
    <aside class="w-80 border-r border-border flex flex-col bg-surface-1 shrink-0">
      <div class="px-4 py-3 border-b border-border flex items-center justify-between bg-surface-2/50">
        <h2 class="text-xs font-semibold text-text-muted uppercase tracking-widest">Task Queue</h2>
        <span class="badge bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
          {{ pendingTasks.length }}
        </span>
      </div>

      <div class="p-2 border-b border-border bg-surface-1 flex gap-1">
        <button 
          @click="filterType = 'all'"
          :class="['flex-1 py-1.5 text-[10px] font-bold uppercase rounded-lg transition-all', filterType === 'all' ? 'bg-accent/10 text-accent border border-accent/20' : 'text-text-muted hover:bg-white/5 border border-transparent']"
        >All</button>
        <button 
          @click="filterType = 'mine'"
          :class="['flex-1 py-1.5 text-[10px] font-bold uppercase rounded-lg transition-all', filterType === 'mine' ? 'bg-accent/10 text-accent border border-accent/20' : 'text-text-muted hover:bg-white/5 border border-transparent']"
        >My Tasks</button>
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
            <span class="text-xs font-bold text-text truncate pr-2">{{ task.workflow_name || 'Workflow' }}</span>
            <span class="text-[9px] font-mono text-text-muted bg-surface-0 px-1.5 py-0.5 rounded border border-border">
              HITL
            </span>
          </div>
          <div class="text-[10px] text-text-muted font-medium mb-2">{{ task.state_name }}</div>
          
          <div class="flex items-center gap-2 text-[9px] text-text-muted">
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
              <span class="text-sm font-semibold text-text">{{ selectedTask.workflow_name }}</span>
              <span class="text-xs text-text-muted font-mono">/ {{ selectedTask.run_id.slice(0, 8) }}</span>
            </div>
            <p class="text-[10px] text-text-muted font-mono tracking-tight">{{ selectedTask.state_name }} state</p>
          </div>

          <div class="flex items-center gap-3">
            <RouterLink :to="`/monitor/${selectedTask.run_id}`" class="btn-ghost text-xs">
              View Run
            </RouterLink>
          </div>
        </header>

        <div class="flex-1 overflow-hidden flex divide-x divide-border">
          <!-- Main Content Area: Social Chat only -->
          <div class="flex-1 flex flex-col overflow-hidden bg-surface-0">
            <div 
              ref="chatScroll"
              class="flex-1 overflow-y-auto p-6 space-y-4 scroll-smooth"
            >
              <div v-for="(msg, mi) in currentChat" :key="mi" 
                :class="['flex flex-col max-w-[85%]', msg.role === 'human' ? 'ml-auto items-end' : 'items-start']"
              >
                <div class="flex items-center gap-2 mb-1 px-1">
                  <span class="text-[9px] font-bold uppercase tracking-tighter text-text-muted">
                    {{ msg.sender || (msg.role === 'human' ? 'You' : 'Agent') }}
                  </span>
                  <span class="text-[8px] text-text-muted opacity-50">{{ formatTime(msg.timestamp) }}</span>
                </div>
                <div 
                  :class="[
                    'px-4 py-2.5 rounded-2xl text-xs leading-relaxed shadow-sm',
                    msg.role === 'human' 
                      ? 'bg-accent text-white rounded-tr-none' 
                      : 'bg-surface-2 text-text border border-border/50 rounded-tl-none'
                  ]"
                >
                  {{ msg.message }}
                </div>
              </div>
              
              <div v-if="!currentChat.length" class="h-full flex flex-col items-center justify-center text-center opacity-30">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="mb-3">
                  <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
                </svg>
                <p class="text-xs">No messages yet. Say hello to the agent!</p>
              </div>
            </div>

            <!-- Chat Input -->
            <div class="p-4 border-t border-border bg-surface-1/50">
              <div class="relative group">
                <input 
                  v-model="chatInput"
                  @keydown.enter.prevent="handleChatSend"
                  placeholder="Type a message to the agent..."
                  class="w-full bg-surface-2 border border-border rounded-xl px-4 py-3 text-xs focus:outline-none focus:ring-2 focus:ring-accent/50 focus:border-accent transition-all pr-12 shadow-inner"
                />
                <button 
                  @click="handleChatSend"
                  :disabled="!chatInput.trim()"
                  class="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 rounded-lg bg-accent text-white hover:bg-accent-hover transition-all disabled:opacity-30 disabled:cursor-not-allowed shadow-lg"
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/>
                  </svg>
                </button>
              </div>
            </div>
          </div>

          <!-- Task Resolution + Blackboard Tweak Sidebar -->
          <div class="w-96 flex flex-col overflow-hidden bg-surface-1/30 border-l border-border">
            <div class="flex-1 overflow-y-auto p-4 space-y-6">
              <!-- Task Form Section -->
              <div class="space-y-4">
                <div class="flex items-center justify-between">
                  <div class="space-y-0.5">
                    <h3 class="text-xs font-semibold text-text uppercase tracking-wider">Input Required</h3>
                    <p class="text-[11px] text-text-muted">Provide the required information to continue.</p>
                  </div>
                  <button 
                    v-if="selectedTask.form_schema"
                    @click="showPopOutModal = true" 
                    class="btn-ghost p-1.5 text-text-muted hover:text-text"
                    title="Pop out form"
                  >
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
                      <polyline points="15 3 21 3 21 9"/>
                      <line x1="10" y1="14" x2="21" y2="3"/>
                    </svg>
                  </button>
                </div>

                <JsonSchemaForm
                  v-if="selectedTask.form_schema"
                  ref="hitlForm"
                  :schema="selectedTask.form_schema"
                  :initial-data="selectedTask.blackboard"
                />
                <div v-else class="py-4 text-center border border-dashed border-border rounded-xl bg-surface-2/10">
                  <p class="text-xs text-text-muted">No specific input structure defined.</p>
                </div>
              </div>

              <!-- Advanced Tweaks collapsible -->
              <div class="pt-4 border-t border-border">
                <div class="flex items-center justify-between mb-3">
                  <h4 class="text-xs font-bold text-text-muted uppercase tracking-widest">Advanced Tweaks</h4>
                  <button 
                    @click="showBlackboard = !showBlackboard" 
                    class="text-[11px] text-indigo-400 hover:text-indigo-300 font-medium"
                  >
                    {{ showBlackboard ? 'Hide Context' : 'Edit Context' }}
                  </button>
                </div>

                <div v-if="showBlackboard && blackboardSchema" class="flex p-1 bg-surface-0 border border-slate-800/40 rounded-lg w-fit mb-3">
                  <button
                    @click="tweakView = 'form'"
                    :class="[
                      'px-2 py-1 text-[9px] font-bold uppercase rounded transition-all',
                      tweakView === 'form' ? 'bg-accent/10 text-accent' : 'text-text-muted hover:text-text-muted'
                    ]"
                  > Form </button>
                  <button
                    @click="tweakView = 'json'"
                    :class="[
                      'px-2 py-1 text-[9px] font-bold uppercase rounded transition-all',
                      tweakView === 'json' ? 'bg-accent/10 text-accent' : 'text-text-muted hover:text-text-muted'
                    ]"
                  > JSON </button>
                </div>

                <div v-if="showBlackboard" class="space-y-4">
                  <div v-if="tweakView === 'form' && blackboardSchema" class="max-h-64 overflow-y-auto pr-2 scrollbar-thin scrollbar-thumb-slate-800">
                    <JsonSchemaForm
                      :schema="blackboardSchema"
                      :initial-data="editedBlackboard"
                      @update:modelValue="syncFromTweakForm"
                    />
                  </div>
                  <BlackboardEditor 
                    v-else
                    v-model="editedBlackboard"
                    class="h-48"
                    @update:modelValue="syncFromTweakJson"
                  />
                </div>
              </div>
            </div>

            <!-- Resolution Buttons Footer -->
            <div class="p-4 border-t border-border bg-surface-1/50 flex gap-3">
              <button
                v-for="btn in currentTransitions"
                :key="btn.trigger"
                @click="resolve(btn.trigger)"
                :class="['flex-1 justify-center py-2.5', getButtonClass(btn.trigger)]"
              >
                {{ btn.label }}
              </button>
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

    <!-- Pop Out HITL Form Modal -->
    <div
      v-if="showPopOutModal && selectedTask?.form_schema"
      class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 px-4"
      @click.self="showPopOutModal = false"
    >
      <div class="card w-full max-w-[600px] p-6 space-y-5 shadow-2xl animate-in fade-in zoom-in duration-200">
        <div class="flex items-center justify-between">
          <h3 class="text-lg font-semibold text-slate-100 flex items-center gap-2">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-indigo-400">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
            </svg>
            Task Input Required
          </h3>
          <button @click="showPopOutModal = false" class="text-text-muted hover:text-text transition-colors">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>
        <p class="text-xs text-text-muted">Provide the required information below to resolve this step.</p>

        <div class="min-h-[150px] overflow-y-auto max-h-[400px] pr-2 scrollbar-thin scrollbar-thumb-slate-800">
          <JsonSchemaForm
            ref="popOutForm"
            :schema="selectedTask.form_schema"
            :initial-data="selectedTask.blackboard"
          />
        </div>

        <div class="pt-4 border-t border-border flex gap-3 justify-end">
          <button
            class="px-4 py-2 text-xs font-medium text-text-muted hover:text-text transition-colors border border-border rounded-lg bg-surface-2"
            @click="showPopOutModal = false"
          >
            Cancel
          </button>
          <button
            v-for="btn in currentTransitions"
            :key="btn.trigger"
            @click="resolveFromModal(btn.trigger)"
            :class="['px-5 py-2 text-xs font-semibold rounded-lg', getButtonClass(btn.trigger)]"
          >
            {{ btn.label }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { useExecutionStore } from '@/stores/execution'
import { useWorkflowStore } from '@/stores/workflows'
import { convertBlackboardSchemaToJsonSchema } from '@/utils/schema'
import JsonSchemaForm from '@/components/monitor/JsonSchemaForm.vue'
import BlackboardEditor from '@/components/monitor/BlackboardEditor.vue'

import { useAuthStore } from '@/stores/auth'
const execStore = useExecutionStore()
const wfStore = useWorkflowStore()
const authStore = useAuthStore()

const workflowDef = ref(null)

const filterType = ref('all')
const pendingTasks = computed(() => execStore.pendingHITL)
const selectedTaskId = ref(null)

watch(filterType, () => {
  fetchTasks()
})

async function fetchTasks() {
  const filter = {}
  if (filterType.value === 'mine' && authStore.user) {
    filter.assignee = authStore.user.username
  }
  await execStore.fetchPendingHITL(filter)
}

const selectedTask = computed(() => pendingTasks.value.find(t => t.id === selectedTaskId.value))

const editedBlackboard = ref({})
const showBlackboard = ref(true)
const tweakView = ref('form')
const resolveError = ref('')
const hitlForm = ref(null)
const showPopOutModal = ref(false)
const popOutForm = ref(null)

const activeMainTab = ref('chat')
const chatInput = ref('')
const chatScroll = ref(null)

const currentChat = computed(() => {
  if (!selectedTask.value) return []
  return execStore.chatMessages[selectedTask.value.run_id] ?? []
})

// Auto-scroll chat
watch(currentChat, () => {
  setTimeout(() => {
    if (chatScroll.value) {
      chatScroll.value.scrollTop = chatScroll.value.scrollHeight
    }
  }, 50)
}, { deep: true })

const blackboardSchema = computed(() => {
  return convertBlackboardSchemaToJsonSchema(selectedTask.value?.blackboard_schema)
})

// When selecting a task, clone its blackboard for editing and fetch workflow definition
watch(selectedTask, async (task) => {
  if (task) {
    editedBlackboard.value = JSON.parse(JSON.stringify(task.blackboard || {}))
    // Default to form if schema exists, otherwise JSON
    tweakView.value = task.blackboard_schema && Object.keys(task.blackboard_schema).length > 0 ? 'form' : 'json'
    
    // Fetch workflow definition to get transitions
    try {
      const res = await wfStore.fetchOne(task.workflow_name, task.workflow_version)
      workflowDef.value = res?.definition || null
    } catch (err) {
      console.error('Failed to fetch workflow definition:', err)
      workflowDef.value = null
    }
  } else {
    workflowDef.value = null
  }
}, { immediate: true })

const currentTransitions = computed(() => {
  if (!workflowDef.value || !selectedTask.value) {
    return [
      { trigger: 'accept', label: 'Accept' },
      { trigger: 'reject', label: 'Reject' }
    ]
  }
  const stateName = selectedTask.value.state_name
  const list = workflowDef.value.transitions?.filter(t => t.from === stateName && t.trigger) || []
  if (list.length > 0) {
    const triggers = [...new Set(list.map(t => t.trigger))]
    return triggers.map(trig => ({
      trigger: trig,
      label: formatTriggerLabel(trig)
    }))
  }
  return [
    { trigger: 'accept', label: 'Accept' },
    { trigger: 'reject', label: 'Reject' }
  ]
})

function formatTriggerLabel(trigger) {
  if (!trigger) return ''
  return trigger
    .replace(/[_-]/g, ' ')
    .split(' ')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}

function getButtonClass(trigger) {
  const t = trigger.toLowerCase()
  if (t.includes('reject') || t.includes('cancel') || t.includes('deny') || t.includes('fail') || t.includes('no')) {
    return 'btn-danger-solid'
  }
  return 'btn-primary'
}

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

async function resolveFromModal(resolution) {
  if (!selectedTask.value) return
  resolveError.value = ''
  
  try {
    const formPayload = popOutForm.value ? popOutForm.value.formData : {}
    const finalPayload = {
      ...editedBlackboard.value,
      ...formPayload
    }

    showPopOutModal.value = false
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

async function handleChatSend() {
  if (!selectedTask.value || !chatInput.value.trim()) return
  const msg = chatInput.value.trim()
  chatInput.value = ''
  
  try {
    await execStore.sendChat(selectedTask.value.run_id, msg)
  } catch (err) {
    resolveError.value = err.response?.data?.error ?? err.message ?? 'Failed to send message'
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
