<template>
  <div class="flex flex-col h-full overflow-hidden">

    <!-- ── Top bar ──────────────────────────────────────────────────────── -->
    <header class="flex items-center gap-3 px-4 py-2.5 border-b border-border shrink-0 bg-surface-1">
      <RouterLink to="/dashboard" class="btn-ghost p-1.5">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/>
        </svg>
      </RouterLink>
      <div class="w-px h-4 bg-border"/>

      <span class="text-sm font-medium text-text">{{ run?.workflow_name ?? 'Monitor' }}</span>
      <span v-if="run" class="text-xs font-mono text-text-muted">{{ run.id.slice(0, 8) }}</span>
      <span v-if="run" :class="`status-${run.status} ml-1`">{{ run.status }}</span>

      <div class="flex-1"/>

      <div class="flex items-center gap-2 mr-4">
        <button @click="onEditClick" class="btn-ghost text-xs gap-1">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
          </svg>
          Edit
        </button>
        <button @click="showRunModal = true" class="btn-ghost text-xs gap-1">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/>
          </svg>
          Re-run
        </button>
      </div>

      <!-- HITL controls (only for hitl-type states, not wait/join nodes) -->
      <template v-if="run?.status === 'waiting' && currentStateType === 'hitl'">
        <button @click="resolve('approved')" class="btn bg-green-500/20 text-green-400 hover:bg-green-500/30 text-xs">✓ Approve</button>
        <button @click="resolve('rejected')" class="btn bg-red-500/20 text-red-400 hover:bg-red-500/30 text-xs">✕ Reject</button>
      </template>
      <!-- Wait/join node indicator -->
      <span v-else-if="run?.status === 'waiting' && currentStateType === 'wait'"
            class="text-xs text-indigo-400/80 font-mono px-2 py-1 rounded bg-indigo-500/10 border border-indigo-500/20">
        ⏳ awaiting condition…
      </span>

      <!-- Manual trigger -->
      <button v-if="run?.status === 'running'" @click="showTriggerModal = true" class="btn-ghost text-xs">
        Fire Trigger
      </button>

      <!-- Termination -->
      <button v-if="canTerminate" @click="stopRun" class="btn text-xs bg-red-500/10 text-red-500 hover:bg-red-500/20 border border-red-500/20">
        <svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor" class="mr-1">
          <rect x="6" y="6" width="12" height="12"/>
        </svg>
        Stop
      </button>

      <button @click="loadRun" class="btn-ghost p-1.5" title="Refresh">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="23 4 23 10 17 10"/>
          <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
        </svg>
      </button>
    </header>

    <!-- Failure reason banner -->
    <div v-if="run?.status === 'failed' && run?.failure_reason"
         class="mx-4 mt-3 mb-1 px-4 py-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-xs font-mono shrink-0 flex items-center justify-between">
      <div class="flex-1">
        <span class="font-semibold">Run failed:</span> {{ run.failure_reason }}
      </div>
      <button 
        v-if="workflowYAML"
        @click="showDebugModal = true" 
        class="btn bg-violet-600 hover:bg-violet-500 text-white text-[10px] px-3 py-1.5 ml-4 shadow-lg shadow-violet-600/20 gap-1.5 uppercase font-bold tracking-wider"
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <path d="M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z"/>
          <path d="M12 8v4"/><path d="M12 16h.01"/>
        </svg>
        Debug with AI
      </button>
    </div>

    <!-- AI Debug Modal -->
    <AIDebugModal
      v-if="showDebugModal && run"
      :run-id="run.id"
      :original-YAML="workflowYAML"
      :failed-node="run.current_state"
      :error-message="run.failure_reason"
      :blackboard="run.blackboard"
      @close="showDebugModal = false"
    />

    <!-- HITL resolve error banner -->
    <div v-if="resolveError"
         class="mx-4 mt-3 mb-1 px-4 py-3 rounded-lg bg-orange-500/10 border border-orange-500/30 text-orange-400 text-xs font-mono shrink-0 flex items-start gap-2">
      <span class="font-semibold shrink-0">HITL signal failed:</span>
      <span class="break-all">{{ resolveError }}</span>
      <button @click="resolveError = ''" class="ml-auto shrink-0 opacity-60 hover:opacity-100">✕</button>
    </div>

    <div class="flex flex-1 overflow-hidden">

      <!-- ── Canvas + thinking + event log ────────────────────────────── -->
      <div class="flex-1 flex flex-col overflow-hidden">

        <!-- State machine canvas (read-only) -->
        <VueFlow
          v-if="flowNodes.length"
          :nodes="flowNodes"
          :edges="flowEdges"
          :node-types="nodeTypes"
          :nodes-draggable="false"
          :nodes-connectable="false"
          :elements-selectable="false"
          fit-view-on-init
          class="bg-surface-0 flex-1"
        >
          <Background pattern-color="#2a2d3a" :gap="24" :size="1" />
          <Controls />
        </VueFlow>
        <div v-else class="flex-1 flex items-center justify-center text-text-muted text-sm">
          Loading workflow…
        </div>

        <!-- Agent thinking stream -->
        <div v-if="thinkingText" class="border-t border-border bg-surface-1 px-4 py-2 max-h-32 overflow-y-auto">
          <div class="flex items-center gap-2 mb-1">
            <div class="w-1.5 h-1.5 rounded-full bg-violet-400 animate-pulse"/>
            <span class="text-xs text-text-muted font-semibold">Agent reasoning</span>
          </div>
          <p class="text-xs font-mono text-text-muted whitespace-pre-wrap">{{ thinkingText }}</p>
        </div>

        <!-- Event log -->
        <div class="h-48 border-t border-border">
          <EventLog :events="filteredEvents" />
        </div>
      </div>

      <!-- ── Right panel ───────────────────────────────────────────────── -->
      <aside class="w-72 border-l border-border flex flex-col overflow-hidden bg-surface-1">
        <div class="flex border-b border-border">
          <button
            v-for="tab in visibleTabs" :key="tab.id"
            @click="activeTab = tab.id"
            :class="[
              'flex-1 py-2 text-xs font-medium transition-colors capitalize',
              activeTab === tab.id ? 'text-text border-b-2 border-accent' : 'text-text-muted hover:text-text',
            ]"
          >
            <span v-if="tab.id === 'debug'" class="flex items-center justify-center gap-1">
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="shrink-0">
                <path d="M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z"/>
                <line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
              </svg>
              LLM Debug
              <span
                v-if="runLLMCalls.length"
                class="text-[9px] bg-accent/20 text-accent px-1 rounded-full"
              >{{ runLLMCalls.length }}</span>
            </span>
            <span v-else-if="tab.id === 'mcp'" class="flex items-center justify-center gap-1">
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="shrink-0">
                <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
              </svg>
              MCP Debug
              <span
                v-if="runMCPLogs.length"
                class="text-[9px] bg-accent/20 text-accent px-1 rounded-full"
              >{{ runMCPLogs.length }}</span>
            </span>
            <span v-else-if="tab.id === 'task'" class="flex items-center justify-center gap-1">
              <span class="w-2 h-2 rounded-full bg-amber-500 animate-pulse"/>
              Task
            </span>
            <span v-else-if="tab.id === 'snapshot'" class="flex items-center justify-center gap-1">
               <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="shrink-0">
                  <circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/>
               </svg>
               Snapshot
            </span>
            <span v-else>{{ tab.label }}</span>
          </button>
        </div>

        <!-- Blackboard tab (Live) -->
        <BlackboardPanel
          v-if="activeTab === 'blackboard'"
          :blackboard="run?.blackboard"
          class="flex-1 overflow-hidden"
        />

        <!-- Snapshot tab (Historical) -->
        <div v-else-if="activeTab === 'snapshot' && selectedTransition" class="flex-1 flex flex-col overflow-hidden">
           <div class="px-4 py-2 border-b border-border bg-indigo-500/5 flex items-center justify-between">
              <span class="text-[10px] font-bold text-indigo-400 uppercase tracking-widest">Historical State</span>
              <button @click="selectedTransition = null; activeTab = 'blackboard'" class="text-[9px] text-text-muted hover:text-text">Close</button>
           </div>
           <div v-if="selectedTransition.agent_output?.reasoning" class="p-3 border-b border-border bg-surface-2">
              <h4 class="text-[10px] font-bold text-text-muted uppercase tracking-widest mb-1.5">Agent Reasoning</h4>
              <p class="text-[11px] font-mono text-text-muted whitespace-pre-wrap leading-relaxed">{{ selectedTransition.agent_output.reasoning }}</p>
           </div>
           <BlackboardPanel
             :blackboard="selectedTransition.blackboard_snapshot"
             class="flex-1 overflow-hidden"
           />
        </div>

        <!-- Debug tabs (admin only) -->
        <LLMDebugPanel
          v-else-if="activeTab === 'debug'"
          :calls="runLLMCalls"
          class="flex-1 overflow-hidden"
        />

        <MCPAuditLogPanel
          v-else-if="activeTab === 'mcp'"
          :logs="runMCPLogs"
          class="flex-1 overflow-hidden"
        />
        
        <!-- Task tab (HITL) -->
        <div v-else-if="activeTab === 'task'" class="flex-1 flex flex-col overflow-hidden bg-surface-2/30">
          <div class="p-4 space-y-4 flex-1 overflow-y-auto">
            <div class="space-y-1 mb-4">
              <h3 class="text-xs font-semibold text-text uppercase tracking-wider">Input Required</h3>
              <p class="text-xs text-text-muted">{{ currentStateDef?.instructions || 'A human must review this step to continue.' }}</p>
            </div>

            <JsonSchemaForm
              v-if="currentFormSchema"
              ref="hitlForm"
              :schema="currentFormSchema"
              :initial-data="run?.blackboard"
            />
            <div v-else class="py-4 text-center">
              <p class="text-xs text-text-muted">No additional data required. Please approve or reject above.</p>
            </div>

            <!-- Blackboard Tweak (Optional) -->
            <div class="mt-8 pt-4 border-t border-border">
              <div class="flex items-center justify-between mb-3">
                <h4 class="text-[10px] font-bold text-text-muted uppercase tracking-widest">Advanced Tweaks</h4>
                <button 
                  @click="showBlackboardTweak = !showBlackboardTweak" 
                  class="text-[9px] text-indigo-400 hover:text-indigo-300"
                >
                  {{ showBlackboardTweak ? 'Hide Context' : 'Edit Context' }}
                </button>
              </div>

              <div v-if="showBlackboardTweak && blackboardSchema" class="flex p-1 bg-surface-0 border border-slate-800/40 rounded-lg w-fit mb-3">
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
              <div v-if="showBlackboardTweak" class="space-y-4">
                <div v-if="tweakView === 'form' && blackboardSchema" class="max-h-[300px] overflow-y-auto pr-2 scrollbar-thin scrollbar-thumb-slate-800">
                  <JsonSchemaForm
                    :schema="blackboardSchema"
                    :initial-data="editedBlackboard"
                    @update:modelValue="syncFromTweakForm"
                  />
                </div>
                <BlackboardEditor 
                  v-else
                  v-model="editedBlackboard"
                  class="max-h-[300px]"
                  @update:modelValue="syncFromTweakJson"
                />
              </div>
            </div>
          </div>
          
          <div class="p-4 border-t border-border bg-surface-1 flex gap-2">
            <button @click="resolve('rejected')" class="flex-1 btn-danger justify-center">Reject</button>
            <button @click="resolve('approved')" class="flex-1 btn-primary justify-center">Approve</button>
          </div>
        </div>

        <!-- Timeline tab -->
        <div v-else class="flex-1 overflow-y-auto p-3">
          <div v-if="!history.length" class="text-text-muted text-xs text-center py-8">
            No transitions yet
          </div>

          <div v-else class="relative">
            <!-- Vertical guide line -->
            <div class="absolute left-[11px] top-4 bottom-0 w-px bg-border"/>

            <div
              v-for="(t, i) in history"
              :key="i"
              class="group relative pl-8 pb-4 last:pb-2 cursor-pointer"
              @click="inspectTransition(t)"
            >
              <!-- Timeline dot -->
              <div 
                :class="[
                  'absolute left-1.5 top-1.5 w-[11px] h-[11px] rounded-full border-2 z-10 transition-all',
                  selectedTransition?.id === t.id ? 'border-accent bg-accent scale-125' : 'border-indigo-500 bg-surface-1 group-hover:border-accent'
                ]"
              />

              <!-- Content -->
              <div :class="['space-y-1 transition-opacity', selectedTransition && selectedTransition.id !== t.id ? 'opacity-40' : 'opacity-100']">
                <!-- State arrived at -->
                <div class="flex items-baseline gap-1.5 font-display">
                  <span class="text-xs font-bold text-text">{{ t.to_state }}</span>
                </div>

                <!-- Trigger badge + origin -->
                <div class="flex items-center gap-1.5 flex-wrap">
                  <span class="inline-flex items-center gap-1 text-[10px] font-mono bg-indigo-500/15 text-indigo-300 px-1.5 py-0.5 rounded border border-indigo-500/25">
                    {{ t.trigger }}
                  </span>
                  <span class="text-[10px] text-text-muted">from</span>
                  <span class="text-[10px] font-mono text-text-muted">{{ t.from_state }}</span>
                </div>

                <!-- Timestamp + time spent in this state -->
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="text-[10px] text-text-muted">{{ formatTime(t.timestamp) }}</span>
                  <span
                    v-if="stateDuration(i) !== null"
                    class="inline-flex items-center gap-1 text-[10px] font-mono text-amber-600/80"
                    :title="`Time spent in ${t.to_state}`"
                  >
                    <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                      <circle cx="12" cy="12" r="9"/><polyline points="12 7 12 12 15 15"/>
                    </svg>
                    {{ stateDuration(i) }}
                  </span>
                </div>
              </div>
            </div>

            <!-- Current active state (live indicator) -->
            <div v-if="run?.current_state" class="relative pl-8 pb-1">
              <div class="absolute left-1.5 top-1.5 w-[11px] h-[11px] rounded-full bg-green-500 border-2 border-green-400 z-10 animate-pulse"/>
              <div class="space-y-0.5">
                <div class="flex items-center gap-2">
                  <span class="text-xs font-mono font-medium text-green-400">{{ run.current_state }}</span>
                  <span class="text-[10px] text-text-muted">← active</span>
                </div>
                <!-- Time spent in the current (still-active) state -->
                <div v-if="history.length" class="flex items-center gap-1 text-[10px] font-mono text-amber-600/60">
                  <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <circle cx="12" cy="12" r="9"/><polyline points="12 7 12 12 15 15"/>
                  </svg>
                  {{ activeStateDuration }}
                  <span v-if="run.status === 'running'" class="text-text-muted font-sans">and counting</span>
                </div>
                <div v-if="run.status !== 'running'" :class="`status-${run.status} text-[10px]`">
                  {{ run.status }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </aside>
    </div>

    <!-- ── Trigger modal ─────────────────────────────────────────────────── -->
    <div
      v-if="showTriggerModal"
      class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50"
      @click.self="showTriggerModal = false"
    >
      <div class="card w-80 p-5 space-y-3">
        <h3 class="font-semibold text-text text-sm">Fire Trigger</h3>
        <input v-model="triggerName" class="input font-mono" placeholder="trigger_name" @keydown.enter="fireTrigger"/>
        <div class="flex gap-2 justify-end">
          <button class="btn-ghost" @click="showTriggerModal = false">Cancel</button>
          <button class="btn-primary" @click="fireTrigger">Fire</button>
        </div>
      </div>
    </div>
    <!-- Start Run Modal -->
    <StartRunModal
      v-if="showRunModal && workflowDef"
      :definition="workflowDef"
      :initial-data="rerunInitialData"
      @close="showRunModal = false"
      @started="(newRun) => router.push(`/monitor/${newRun.id}`).then(() => location.reload())"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, shallowRef, watch } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { VueFlow } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls }   from '@vue-flow/controls'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/controls/dist/style.css'

import BlackboardEditor from '@/components/monitor/BlackboardEditor.vue'

import StateNode      from '@/components/designer/StateNode.vue'
import BlackboardPanel from '@/components/monitor/BlackboardPanel.vue'
import EventLog        from '@/components/monitor/EventLog.vue'
import LLMDebugPanel   from '@/components/monitor/LLMDebugPanel.vue'
import MCPAuditLogPanel from '@/components/monitor/MCPAuditLogPanel.vue'
import JsonSchemaForm  from '@/components/monitor/JsonSchemaForm.vue'
import AIDebugModal    from '@/components/monitor/AIDebugModal.vue'
import StartRunModal from '@/components/shared/StartRunModal.vue'
import { useExecutionStore } from '@/stores/execution'
import { useWorkflowStore }  from '@/stores/workflows'
import { useAuthStore }      from '@/stores/auth'
import { useLayout }         from '@/composables/useLayout'
import { convertPhaxaSchemaToJsonSchema } from '@/utils/schema'

const route     = useRoute()
const router    = useRouter()
const execStore  = useExecutionStore()
const wfStore    = useWorkflowStore()
const authStore  = useAuthStore()
const { layout } = useLayout()

const activeTab    = ref('blackboard')
const showTriggerModal = ref(false)
const showRunModal = ref(false)
const showDebugModal = ref(false)
const triggerName  = ref('')
const workflowDef  = ref(null)
const workflowYAML = ref('')
const nodeTypes    = shallowRef({ stateNode: StateNode })
const hitlForm     = ref(null)
const showBlackboardTweak = ref(false)
const tweakView = ref('form')
const editedBlackboard = ref({})
const resolveError = ref('')
const selectedTransition = ref(null)

const run          = computed(() => execStore.activeRun)
const history      = computed(() => execStore.transitions)
const runLLMCalls  = computed(() => execStore.llmCalls[route.params.id] ?? [])
const runMCPLogs   = computed(() => execStore.mcpLogs[route.params.id]  ?? [])

const currentStateType = computed(() => {
  if (!workflowDef.value || !run.value?.current_state) return null
  return workflowDef.value.states?.find(s => s.name === run.value.current_state)?.type ?? null
})

const currentStateDef = computed(() => {
  if (!workflowDef.value || !run.value?.current_state) return null
  return workflowDef.value.states?.find(s => s.name === run.value.current_state)
})

const canTerminate = computed(() => {
  if (!run.value) return false
  return ['pending', 'running', 'waiting'].includes(run.value.status)
})

const currentFormSchema = computed(() => currentStateDef.value?.form_schema)

const blackboardSchema = computed(() => {
  return convertPhaxaSchemaToJsonSchema(workflowDef.value?.blackboard?.schema)
})

const rerunInitialData = computed(() => {
  if (!run.value?.blackboard || !workflowDef.value?.blackboard?.schema) return {}
  const schema = workflowDef.value.blackboard.schema
  const bb = run.value.blackboard
  const filtered = {}
  for (const k of Object.keys(schema)) {
    if (bb[k] !== undefined) {
      filtered[k] = bb[k]
    }
  }
  return filtered
})

const visibleTabs  = computed(() => {
  const tabs = [
    { id: 'blackboard', label: 'Blackboard' },
    { id: 'timeline',   label: 'Timeline'   },
  ]
  if (run.value?.status === 'waiting' && currentStateType.value === 'hitl') {
    tabs.push({ id: 'task', label: 'Task' })
  }
  if (authStore.hasRole('admin')) {
    tabs.push({ id: 'debug', label: 'LLM Debug' })
    tabs.push({ id: 'mcp',   label: 'MCP Debug' })
  }
  if (selectedTransition.value) {
    tabs.push({ id: 'snapshot', label: 'Snapshot' })
  }
  return tabs
})

// Auto-switch to Task tab when waiting at a HITL node
watch([() => run.value?.status, () => currentStateType.value], ([status, type]) => {
  if (status === 'waiting' && type === 'hitl') {
    activeTab.value = 'task'
  }
}, { immediate: true })

// Capture blackboard when HITL is active
watch(() => run.value?.blackboard, (bb) => {
  if (bb && run.value?.status === 'waiting') {
    editedBlackboard.value = JSON.parse(JSON.stringify(bb))
  }
}, { immediate: true })

function syncFromTweakForm(data) {
  if (JSON.stringify(editedBlackboard.value) !== JSON.stringify(data)) {
    editedBlackboard.value = { ...data }
  }
}

function syncFromTweakJson(data) {
  // Bi-directional updates work through v-model
}

// ── Data loading ─────────────────────────────────────────────────────────────

async function loadRun() {
  const r = await execStore.fetchRun(route.params.id)
  await execStore.fetchHistory(route.params.id)
  if (authStore.hasRole('admin')) {
    await execStore.fetchMCPLogs(route.params.id)
  }
  if (r) {
    const { definition, yaml: src } = await wfStore
      .fetchOne(r.workflow_name, r.workflow_version)
      .catch(() => ({ definition: null, yaml: '' }))
    workflowDef.value = definition
    workflowYAML.value = src
  }
}

onMounted(loadRun)

// ── Vue Flow graph ────────────────────────────────────────────────────────────

// Edges don't depend on node positions — compute separately so layout() can use them
const flowEdges = computed(() => {
  if (!workflowDef.value) return []
  const edges = []
  
  for (const t of (workflowDef.value.transitions ?? [])) {
    // Handle both single target ('to') and parallel targets ('to_nodes')
    const targets = t.to_nodes ?? (t.to ? [t.to] : [])
    
    for (const target of targets) {
      edges.push({
        id:     `${t.from}__${t.trigger}__${target}`,
        source:  t.from,
        target:  target,
        label:   t.trigger,
        data:   { guard: t.guard },
      })
    }
  }
  return edges
})

const flowNodes = computed(() => {
  if (!workflowDef.value) return []
  const states       = workflowDef.value.states ?? []
  const currentState = run.value?.current_state

  const rawNodes = states.map(s => ({
    id:       s.name,
    type:     'stateNode',
    position: s.position ?? { x: 200, y: 100 },
    data:     { ...s, isActive: s.name === currentState },
  }))

  // Apply dagre layout if the workflow has no explicitly saved positions
  const hasExplicit = states.some(
    s => s.position && (s.position.x !== 200 || s.position.y !== 100)
  )
  if (!hasExplicit && rawNodes.length > 1) {
    return layout(rawNodes, flowEdges.value)
  }
  return rawNodes
})

// ── Events & thinking ─────────────────────────────────────────────────────────

const filteredEvents = computed(() =>
  execStore.eventLog.filter(e => {
    const d = e.data ?? {}
    return d.run_id === route.params.id || d.run?.id === route.params.id
  })
)

const thinkingText = computed(() => execStore.thinkingTokens[route.params.id] ?? '')

// ── Actions ───────────────────────────────────────────────────────────────────

async function resolve(resolution) {
  if (!run.value) return
  resolveError.value = ''
  try {
    const formPayload = hitlForm.value ? hitlForm.value.formData : {}
    const finalPayload = {
      ...editedBlackboard.value,
      ...formPayload
    }
    await execStore.resolveHITL(route.params.id, resolution, 'user', finalPayload)
    
    // Switch tab and clean up before refetching to ensure smooth transition
    activeTab.value = 'blackboard'
    showBlackboardTweak.value = false
    
    await loadRun()
  } catch (err) {
    const msg = err?.response?.data?.error ?? err?.message ?? 'Unknown error'
    resolveError.value = msg
    await loadRun()
  }
}

async function fireTrigger() {
  if (!triggerName.value) return
  await execStore.sendTrigger(route.params.id, triggerName.value)
  showTriggerModal.value = false
  triggerName.value = ''
}

async function stopRun() {
  if (!confirm('Are you sure you want to stop this run? This action cannot be undone.')) return
  try {
    await execStore.terminateRun(route.params.id)
  } catch (err) {
    const msg = err?.response?.data?.error ?? err?.message ?? 'Failed to stop run'
    alert(msg)
  }
}

function onEditClick() {
  if (!run.value) return
  router.push(`/designer/${run.value.workflow_name}/${run.value.workflow_version}`)
}

function inspectTransition(t) {
  if (selectedTransition.value?.id === t.id) {
    selectedTransition.value = null
    activeTab.value = 'blackboard'
  } else {
    selectedTransition.value = t
    activeTab.value = 'snapshot'
  }
}

// ── Formatting ────────────────────────────────────────────────────────────────

function formatTime(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  const now = Date.now()
  const diff = now - d.getTime()
  if (diff < 60_000)  return 'just now'
  if (diff < 3600_000) {
    const m = Math.floor(diff / 60_000)
    return `${m}m ago`
  }
  return d.toLocaleTimeString('en', { hour12: false, hour: '2-digit', minute: '2-digit' })
}

function formatDuration(ms) {
  if (ms < 1000)    return '< 1s'
  const s = Math.floor(ms / 1000)
  if (s < 60)       return `${s}s`
  const m = Math.floor(s / 60)
  const rs = s % 60
  if (m < 60)       return rs ? `${m}m ${rs}s` : `${m}m`
  const h = Math.floor(m / 60)
  const rm = m % 60
  return rm ? `${h}h ${rm}m` : `${h}h`
}

// Time spent in history[i].to_state = next transition timestamp − this timestamp.
// Returns null for the very last completed entry (duration belongs to activeStateDuration).
function stateDuration(i) {
  const next = history.value[i + 1]
  if (!next) return null   // last entry — shown separately as activeStateDuration
  const ms = new Date(next.timestamp) - new Date(history.value[i].timestamp)
  return formatDuration(ms)
}

// Duration the run has been in its current state (last transition → now).
const activeStateDuration = computed(() => {
  const last = history.value[history.value.length - 1]
  if (!last) return null
  return formatDuration(Date.now() - new Date(last.timestamp))
})
</script>
