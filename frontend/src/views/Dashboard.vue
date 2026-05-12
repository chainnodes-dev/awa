<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Top bar -->
    <header class="flex items-center gap-4 px-6 py-4 border-b border-border shrink-0">
      <h1 class="text-base font-semibold text-text">Dashboard</h1>
      <div class="flex-1"/>
      <RouterLink to="/designer" class="btn-primary">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        New Workflow
      </RouterLink>
    </header>

    <div class="flex-1 overflow-y-auto p-6 space-y-8">

      <!-- HITL banner -->
      <div v-if="execStore.pendingHITL.length" class="card border-amber-500/40 bg-amber-500/5 p-4 flex items-center gap-4">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="#f59e0b">
          <path d="M12 2a5 5 0 1 0 5 5 5 5 0 0 0-5-5zm0 8a3 3 0 1 1 3-3 3 3 0 0 1-3 3zm9 11v-1a7 7 0 0 0-7-7h-4a7 7 0 0 0-7 7v1h2v-1a5 5 0 0 1 5-5h4a5 5 0 0 1 5 5v1z"/>
        </svg>
        <span class="text-amber-400 text-sm font-medium">
          {{ execStore.pendingHITL.length }} run{{ execStore.pendingHITL.length > 1 ? 's' : '' }} awaiting human approval
        </span>
        <div class="flex gap-2 ml-auto">
          <button
            v-for="req in execStore.pendingHITL.slice(0, 3)" :key="req.run_id"
            @click="openMonitor(req.run_id)"
            class="btn bg-amber-500/20 text-amber-300 hover:bg-amber-500/30 text-xs"
          >
            Review {{ req.run_id.slice(0, 8) }}
          </button>
        </div>
      </div>

      <!-- Stats row -->
      <div class="grid grid-cols-4 gap-4">
        <StatCard v-for="s in stats" :key="s.label" v-bind="s" />
      </div>

      <!-- Workflow Definitions -->
      <section>
        <div class="flex items-center gap-3 mb-3">
          <h2 class="text-sm font-semibold text-text">Workflow Definitions</h2>
          <span class="badge bg-white/5 text-text-muted">{{ wfStore.definitions.length }}</span>
        </div>

        <div v-if="wfStore.loading" class="text-text-muted text-sm py-8 text-center">Loading…</div>
        <div v-else-if="!wfStore.definitions.length" class="card p-8 text-center">
          <p class="text-text-muted text-sm mb-4">No workflows yet. Create your first one.</p>
          <RouterLink to="/designer" class="btn-primary">Create Workflow</RouterLink>
        </div>
        <div v-else class="grid grid-cols-1 gap-3">
          <WorkflowCard
            v-for="def in wfStore.definitions"
            :key="`${def.metadata.name}@${def.metadata.version}`"
            :definition="def"
            @run="startRun(def)"
            @edit="editWorkflow(def)"
            @delete="deleteWorkflow(def)"
          />
        </div>
      </section>

      <!-- Recent Runs -->
      <section>
        <div class="flex items-center gap-3 mb-3">
          <h2 class="text-sm font-semibold text-text">Recent Runs</h2>
          <span class="badge bg-white/5 text-text-muted">{{ execStore.runs.length }}</span>
        </div>

        <!-- Filters -->
        <div class="flex flex-wrap items-end gap-3 mb-3">
          <div class="space-y-1.5">
            <label class="text-[10px] text-text-muted uppercase tracking-wider">Workflow</label>
            <select v-model="runFilters.workflow" @change="applyFilters" class="input text-xs py-1.5 w-40">
              <option value="">All</option>
              <option v-for="d in wfStore.definitions" :key="d.metadata.name" :value="d.metadata.name">{{ d.metadata.name }}</option>
            </select>
          </div>
          <div class="space-y-1.5">
            <label class="text-[10px] text-text-muted uppercase tracking-wider">Status</label>
            <select v-model="runFilters.status" @change="applyFilters" class="input text-xs py-1.5 w-32">
              <option value="">All</option>
              <option value="pending">pending</option>
              <option value="running">running</option>
              <option value="waiting">waiting</option>
              <option value="complete">complete</option>
              <option value="failed">failed</option>
            </select>
          </div>
          <div class="space-y-1.5">
            <label class="text-[10px] text-text-muted uppercase tracking-wider">Started from</label>
            <input type="date" v-model="runFilters.startedFrom" @change="applyFilters" class="input text-xs py-1.5 w-36" />
          </div>
          <div class="space-y-1.5">
            <label class="text-[10px] text-text-muted uppercase tracking-wider">Started to</label>
            <input type="date" v-model="runFilters.startedTo" @change="applyFilters" class="input text-xs py-1.5 w-36" />
          </div>
          <button @click="resetFilters" class="btn-ghost text-xs py-1.5 px-2 text-text-muted">Reset</button>
          <button v-if="authStore.hasRole('admin') && execStore.runs.length" @click="confirmDeleteAll" class="btn-danger text-xs py-1.5 px-2 ml-auto">Delete All</button>
        </div>

        <div v-if="!execStore.runs.length" class="text-text-muted text-sm py-4 text-center">
          No runs match the current filters.
        </div>
        <div v-else class="card overflow-hidden">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-border">
                <th class="text-left px-4 py-2.5 text-xs font-medium text-text-muted">Run ID</th>
                <th class="text-left px-4 py-2.5 text-xs font-medium text-text-muted">Workflow</th>
                <th class="text-left px-4 py-2.5 text-xs font-medium text-text-muted">State</th>
                <th class="text-left px-4 py-2.5 text-xs font-medium text-text-muted">Status</th>
                <th class="text-left px-4 py-2.5 text-xs font-medium text-text-muted">Started</th>
                <th/>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="run in execStore.runs.slice(0, 20)" :key="run.id"
                class="border-b border-border/50 hover:bg-white/3 transition-colors"
              >
                <td class="px-4 py-2.5 font-mono text-xs text-text-muted">{{ run.id.slice(0, 8) }}</td>
                <td class="px-4 py-2.5 text-text">{{ run.workflow_name }}</td>
                <td class="px-4 py-2.5 font-mono text-xs text-indigo-400">{{ run.current_state }}</td>
                <td class="px-4 py-2.5">
                  <div class="flex items-center gap-2">
                    <div v-if="run.status === 'running'" class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse shadow-sm shadow-emerald-500/50"/>
                    <div v-else-if="run.status === 'waiting'" class="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse shadow-sm shadow-amber-500/50"/>
                    <div v-else-if="run.status === 'failed'" class="w-1.5 h-1.5 rounded-full bg-red-500 shadow-sm shadow-red-500/50"/>
                    <div v-else class="w-1.5 h-1.5 rounded-full bg-slate-500 opacity-30"/>
                    <span :class="`status-${run.status} text-[10px] font-bold uppercase tracking-wider`">{{ run.status }}</span>
                  </div>
                </td>
                <td class="px-4 py-2.5 text-xs text-text-muted">{{ relativeTime(run.started_at) }}</td>
                <td class="px-4 py-2.5 text-right flex gap-1 justify-end">
                  <button @click="openMonitor(run.id)" class="btn-ghost text-xs py-1 px-2">Monitor</button>
                  <button v-if="authStore.hasRole('admin')" @click="confirmDeleteRun(run)" class="btn-danger text-xs py-1 px-2">Delete</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <!-- Start Run Modal -->
    <StartRunModal v-if="selectedDef" :definition="selectedDef" @close="selectedDef = null" @started="onRunStarted" />

  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useWorkflowStore } from '@/stores/workflows'
import { useExecutionStore } from '@/stores/execution'
import { useAuthStore } from '@/stores/auth'
import { useEnterpriseStore } from '@/stores/enterprise'

import StartRunModal from '@/components/shared/StartRunModal.vue'

const router = useRouter()
const wfStore = useWorkflowStore()
const execStore = useExecutionStore()
const authStore = useAuthStore()
const entStore = useEnterpriseStore()
const selectedDef = ref(null)

// Run filters — default "Started from" to today
const today = new Date().toISOString().slice(0, 10)
const runFilters = reactive({
  workflow: '',
  status: '',
  startedFrom: today,
  startedTo: '',
})

function buildFilterParams() {
  const params = {}
  if (runFilters.workflow) params.workflow = runFilters.workflow
  if (runFilters.status)   params.status = runFilters.status
  if (runFilters.startedFrom) params.started_from = new Date(runFilters.startedFrom).toISOString()
  if (runFilters.startedTo)   params.started_to = new Date(runFilters.startedTo + 'T23:59:59').toISOString()
  return params
}

function applyFilters() {
  execStore.fetchRuns(buildFilterParams())
}

function resetFilters() {
  runFilters.workflow = ''
  runFilters.status = ''
  runFilters.startedFrom = today
  runFilters.startedTo = ''
  applyFilters()
}

async function confirmDeleteRun(run) {
  if (!confirm('Delete run ' + run.id.slice(0, 8) + '? This cannot be undone.')) return
  try {
    await execStore.deleteRun(run.id)
  } catch (e) {
    alert(e.response?.data?.error ?? e.message)
  }
}

async function confirmDeleteAll() {
  const count = execStore.runs.length
  if (!confirm('Delete all ' + count + ' visible runs? This cannot be undone.')) return
  const deletingRuns = true
  try {
    const ids = execStore.runs.map(r => r.id)
    await Promise.all(ids.map(id => execStore.deleteRun(id)))
  } catch (e) {
    alert(e.response?.data?.error ?? e.message)
  }
}

onMounted(async () => {
  await Promise.all([
    wfStore.fetchAll(),
    execStore.fetchRuns(),
    execStore.fetchPendingHITL(),
    entStore.fetchAnalytics()
  ])
})

const stats = computed(() => {
  const analytics = entStore.analytics || {}
  const total = Object.values(analytics).reduce((a, b) => a + b, 0)
  const completed = analytics.complete || 0
  const successRate = total > 0 ? Math.round((completed / total) * 100) : 0

  return [
    { label: 'Saved Designs', value: wfStore.definitions.length, color: 'indigo' },
    { label: 'Active Runs', value: analytics.running || 0, color: 'emerald' },
    { label: 'Avg Success', value: `${successRate}%`, color: 'blue' },
    { label: 'Open HITL', value: execStore.pendingHITL.length, color: 'amber' }
  ]
})

function startRun(def) { selectedDef.value = def }
function editWorkflow(def) { router.push(`/designer/${def.metadata.name}/${def.metadata.version}`) }
async function deleteWorkflow(def) {
  if (!confirm(`Delete ${def.metadata.name}@${def.metadata.version}?`)) return
  await wfStore.remove(def.metadata.name, def.metadata.version)
}
function openMonitor(id) { router.push(`/monitor/${id}`) }
function onRunStarted(run) { execStore.runs.unshift(run); router.push(`/monitor/${run.id}`) }

function relativeTime(ts) {
  const d = Math.floor((Date.now() - new Date(ts)) / 1000)
  if (d < 60) return `${d}s ago`
  if (d < 3600) return `${Math.floor(d/60)}m ago`
  return `${Math.floor(d/3600)}h ago`
}
</script>

<script>
// ---- Inline sub-components ----
import { defineComponent, h, ref } from 'vue'

export const StatCard = defineComponent({
  props: { label: String, value: [Number, String], color: String },
  setup(p) {
    const colors = { indigo: 'text-indigo-400 bg-indigo-500/10', blue: 'text-blue-400 bg-blue-500/10', amber: 'text-amber-400 bg-amber-500/10', green: 'text-green-400 bg-green-500/10' }
    return () => h('div', { class: 'card p-4' }, [
      h('div', { class: `text-2xl font-bold mb-1 ${colors[p.color]?.split(' ')[0]}` }, p.value),
      h('div', { class: 'text-xs text-text-muted' }, p.label)
    ])
  }
})

export const WorkflowCard = defineComponent({
  props: { definition: Object },
  emits: ['run', 'edit', 'delete'],
  setup(p, { emit }) {
    const def = p.definition
    return () => h('div', { class: 'card p-4 flex items-center gap-4 hover:border-border/80 transition-colors' }, [
      h('div', { class: 'flex-1 min-w-0' }, [
        h('div', { class: 'flex items-center gap-2 mb-1' }, [
          h('span', { class: 'font-medium text-text text-sm' }, def.metadata.name),
          h('span', { class: 'badge bg-white/5 text-text-muted font-mono' }, def.metadata.version?.toString().startsWith('v') ? def.metadata.version : `v${def.metadata.version}`),
          def.metadata.reusable ? h('span', { class: 'badge bg-violet-500/10 text-violet-400 font-bold tracking-wider text-[9px]' }, 'REUSABLE') : null
        ]),
        h('p', { class: 'text-xs text-text-muted truncate' }, def.metadata.description || `${def.states?.length ?? 0} states · ${def.agents?.length ?? 0} agents`),
      ]),
      h('div', { class: 'flex gap-2 shrink-0' }, [
        h('button', { class: 'btn-ghost text-xs', onClick: () => emit('edit') }, 'Edit'),
        h('button', { class: 'btn-primary text-xs', onClick: () => emit('run') }, 'Run'),
        h('button', { class: 'btn-danger text-xs', onClick: () => emit('delete') }, '✕'),
      ])
    ])
  }
})


</script>
