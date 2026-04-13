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

      <span class="text-sm font-medium text-slate-300">{{ run?.workflow_name ?? 'Monitor' }}</span>
      <span v-if="run" class="text-xs font-mono text-slate-600">{{ run.id.slice(0, 8) }}</span>
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

      <button @click="loadRun" class="btn-ghost p-1.5" title="Refresh">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="23 4 23 10 17 10"/>
          <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
        </svg>
      </button>
    </header>

    <!-- Failure reason banner -->
    <div v-if="run?.status === 'failed' && run?.failure_reason"
         class="mx-4 mt-3 mb-1 px-4 py-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-xs font-mono shrink-0">
      <span class="font-semibold">Run failed:</span> {{ run.failure_reason }}
    </div>

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
        <div v-else class="flex-1 flex items-center justify-center text-slate-600 text-sm">
          Loading workflow…
        </div>

        <!-- Agent thinking stream -->
        <div v-if="thinkingText" class="border-t border-border bg-surface-1 px-4 py-2 max-h-32 overflow-y-auto">
          <div class="flex items-center gap-2 mb-1">
            <div class="w-1.5 h-1.5 rounded-full bg-violet-400 animate-pulse"/>
            <span class="text-xs text-slate-500 font-semibold">Agent reasoning</span>
          </div>
          <p class="text-xs font-mono text-slate-400 whitespace-pre-wrap">{{ thinkingText }}</p>
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
              activeTab === tab.id ? 'text-slate-200 border-b-2 border-accent' : 'text-slate-500 hover:text-slate-300',
            ]"
          >
            <span v-if="tab.id === 'debug'" class="flex items-center justify-center gap-1">
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="shrink-0">
                <path d="M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z"/>
                <line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
              </svg>
              Debug
              <span
                v-if="runLLMCalls.length"
                class="text-[9px] bg-accent/20 text-accent px-1 rounded-full"
              >{{ runLLMCalls.length }}</span>
            </span>
            <span v-else>{{ tab.label }}</span>
          </button>
        </div>

        <!-- Blackboard tab -->
        <BlackboardPanel
          v-if="activeTab === 'blackboard'"
          :blackboard="run?.blackboard"
          class="flex-1 overflow-hidden"
        />

        <!-- Debug tab (admin only) -->
        <LLMDebugPanel
          v-else-if="activeTab === 'debug'"
          :calls="runLLMCalls"
          class="flex-1 overflow-hidden"
        />

        <!-- Timeline tab -->
        <div v-else class="flex-1 overflow-y-auto p-3">
          <div v-if="!history.length" class="text-slate-600 text-xs text-center py-8">
            No transitions yet
          </div>

          <div v-else class="relative">
            <!-- Vertical guide line -->
            <div class="absolute left-[11px] top-4 bottom-0 w-px bg-border"/>

            <div
              v-for="(t, i) in history"
              :key="i"
              class="relative pl-8 pb-4 last:pb-2"
            >
              <!-- Timeline dot -->
              <div class="absolute left-1.5 top-1.5 w-[11px] h-[11px] rounded-full border-2 border-indigo-500 bg-surface-1 z-10"/>

              <!-- Content -->
              <div class="space-y-1">
                <!-- State arrived at -->
                <div class="flex items-baseline gap-1.5">
                  <span class="text-xs font-mono font-medium text-slate-200">{{ t.to_state }}</span>
                </div>

                <!-- Trigger badge + origin -->
                <div class="flex items-center gap-1.5 flex-wrap">
                  <span class="inline-flex items-center gap-1 text-[10px] font-mono bg-indigo-500/15 text-indigo-300 px-1.5 py-0.5 rounded border border-indigo-500/25">
                    {{ t.trigger }}
                  </span>
                  <span class="text-[10px] text-slate-600">from</span>
                  <span class="text-[10px] font-mono text-slate-500">{{ t.from_state }}</span>
                </div>

                <!-- Timestamp + time spent in this state -->
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="text-[10px] text-slate-600">{{ formatTime(t.timestamp) }}</span>
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
                  <span class="text-[10px] text-slate-600">← active</span>
                </div>
                <!-- Time spent in the current (still-active) state -->
                <div v-if="history.length" class="flex items-center gap-1 text-[10px] font-mono text-amber-600/60">
                  <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <circle cx="12" cy="12" r="9"/><polyline points="12 7 12 12 15 15"/>
                  </svg>
                  {{ activeStateDuration }}
                  <span v-if="run.status === 'running'" class="text-slate-600 font-sans">and counting</span>
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
        <h3 class="font-semibold text-slate-200 text-sm">Fire Trigger</h3>
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
      :initial-data="run?.blackboard"
      @close="showRunModal = false"
      @started="(newRun) => router.push(`/monitor/${newRun.id}`).then(() => location.reload())"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, shallowRef } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { VueFlow } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls }   from '@vue-flow/controls'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/controls/dist/style.css'

import StateNode      from '@/components/designer/StateNode.vue'
import BlackboardPanel from '@/components/monitor/BlackboardPanel.vue'
import EventLog        from '@/components/monitor/EventLog.vue'
import LLMDebugPanel   from '@/components/monitor/LLMDebugPanel.vue'
import StartRunModal from '@/components/shared/StartRunModal.vue'
import { useExecutionStore } from '@/stores/execution'
import { useWorkflowStore }  from '@/stores/workflows'
import { useAuthStore }      from '@/stores/auth'
import { useLayout }         from '@/composables/useLayout'

const route     = useRoute()
const router    = useRouter()
const execStore  = useExecutionStore()
const wfStore    = useWorkflowStore()
const authStore  = useAuthStore()
const { layout } = useLayout()

const run          = computed(() => execStore.activeRun)
const history      = computed(() => execStore.transitions)
const runLLMCalls  = computed(() => execStore.llmCalls[route.params.id] ?? [])
const visibleTabs  = computed(() => {
  const tabs = [
    { id: 'blackboard', label: 'Blackboard' },
    { id: 'timeline',   label: 'Timeline'   },
  ]
  if (authStore.hasRole('admin')) tabs.push({ id: 'debug', label: 'Debug' })
  return tabs
})
const currentStateType = computed(() => {
  if (!workflowDef.value || !run.value?.current_state) return null
  return workflowDef.value.states?.find(s => s.name === run.value.current_state)?.type ?? null
})
const activeTab    = ref('blackboard')
const showTriggerModal = ref(false)
const showRunModal = ref(false)
const triggerName  = ref('')
const workflowDef  = ref(null)
const nodeTypes    = shallowRef({ stateNode: StateNode })

// ── Data loading ─────────────────────────────────────────────────────────────

async function loadRun() {
  const r = await execStore.fetchRun(route.params.id)
  await execStore.fetchHistory(route.params.id)
  if (r) {
    const { definition } = await wfStore
      .fetchOne(r.workflow_name, r.workflow_version)
      .catch(() => ({ definition: null }))
    workflowDef.value = definition
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

const resolveError = ref('')

async function resolve(resolution) {
  resolveError.value = ''
  try {
    await execStore.resolveHITL(route.params.id, resolution, 'user')
    await loadRun()
  } catch (err) {
    const msg = err?.response?.data?.error ?? err?.message ?? 'Unknown error'
    resolveError.value = msg
    // Roll back optimistic status update so HITL buttons reappear
    await loadRun()
  }
}

async function fireTrigger() {
  if (!triggerName.value) return
  await execStore.sendTrigger(route.params.id, triggerName.value)
  showTriggerModal.value = false
  triggerName.value = ''
}

function onEditClick() {
  if (!run.value) return
  router.push(`/designer/${run.value.workflow_name}/${run.value.workflow_version}`)
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
