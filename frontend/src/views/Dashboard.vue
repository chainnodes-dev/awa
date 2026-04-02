<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Top bar -->
    <header class="flex items-center gap-4 px-6 py-4 border-b border-border shrink-0">
      <h1 class="text-base font-semibold text-slate-200">Dashboard</h1>
      <div class="flex-1"/>
      <button @click="showGenerateDialog = true" class="btn-ghost text-xs gap-1.5 border border-border">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 2l2.4 7.4H22l-6.2 4.5 2.4 7.4L12 17l-6.2 4.3 2.4-7.4L2 9.4h7.6z"/>
        </svg>
        Generate with AI
      </button>
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
          <h2 class="text-sm font-semibold text-slate-300">Workflow Definitions</h2>
          <span class="badge bg-white/5 text-slate-500">{{ wfStore.definitions.length }}</span>
        </div>

        <div v-if="wfStore.loading" class="text-slate-600 text-sm py-8 text-center">Loading…</div>
        <div v-else-if="!wfStore.definitions.length" class="card p-8 text-center">
          <p class="text-slate-500 text-sm mb-4">No workflows yet. Create your first one.</p>
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
          <h2 class="text-sm font-semibold text-slate-300">Recent Runs</h2>
          <span class="badge bg-white/5 text-slate-500">{{ execStore.runs.length }}</span>
        </div>

        <div v-if="!execStore.runs.length" class="text-slate-600 text-sm py-4 text-center">
          No runs yet.
        </div>
        <div v-else class="card overflow-hidden">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-border">
                <th class="text-left px-4 py-2.5 text-xs font-medium text-slate-500">Run ID</th>
                <th class="text-left px-4 py-2.5 text-xs font-medium text-slate-500">Workflow</th>
                <th class="text-left px-4 py-2.5 text-xs font-medium text-slate-500">State</th>
                <th class="text-left px-4 py-2.5 text-xs font-medium text-slate-500">Status</th>
                <th class="text-left px-4 py-2.5 text-xs font-medium text-slate-500">Started</th>
                <th/>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="run in execStore.runs.slice(0, 20)" :key="run.id"
                class="border-b border-border/50 hover:bg-white/3 transition-colors"
              >
                <td class="px-4 py-2.5 font-mono text-xs text-slate-400">{{ run.id.slice(0, 8) }}</td>
                <td class="px-4 py-2.5 text-slate-300">{{ run.workflow_name }}</td>
                <td class="px-4 py-2.5 font-mono text-xs text-indigo-400">{{ run.current_state }}</td>
                <td class="px-4 py-2.5">
                  <span :class="`status-${run.status}`">{{ run.status }}</span>
                </td>
                <td class="px-4 py-2.5 text-xs text-slate-500">{{ relativeTime(run.started_at) }}</td>
                <td class="px-4 py-2.5 text-right">
                  <button @click="openMonitor(run.id)" class="btn-ghost text-xs py-1 px-2">Monitor</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <!-- Start Run Modal -->
    <StartRunModal v-if="selectedDef" :definition="selectedDef" @close="selectedDef = null" @started="onRunStarted" />

    <!-- Generate Workflow Modal -->
    <GenerateWorkflowModal v-if="showGenerateDialog" @close="showGenerateDialog = false" @generated="onGenerated" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useWorkflowStore } from '@/stores/workflows'
import { useExecutionStore } from '@/stores/execution'

const router = useRouter()
const wfStore = useWorkflowStore()
const execStore = useExecutionStore()
const selectedDef = ref(null)
const showGenerateDialog = ref(false)

onMounted(() => {
  wfStore.fetchAll()
  execStore.fetchRuns()
  execStore.fetchPendingHITL()
})

const stats = computed(() => {
  const runs = execStore.runs
  return [
    { label: 'Total Workflows', value: wfStore.definitions.length, color: 'indigo' },
    { label: 'Active Runs',     value: runs.filter(r => r.status === 'running').length,  color: 'blue' },
    { label: 'Awaiting HITL',   value: runs.filter(r => r.status === 'waiting').length,  color: 'amber' },
    { label: 'Completed Today', value: runs.filter(r => r.status === 'complete').length, color: 'green' },
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

function onGenerated({ name, version }) {
  showGenerateDialog.value = false
  wfStore.fetchAll()
  router.push(`/designer/${name}/${version}`)
}

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
      h('div', { class: 'text-xs text-slate-500' }, p.label)
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
          h('span', { class: 'font-medium text-slate-200 text-sm' }, def.metadata.name),
          h('span', { class: 'badge bg-white/5 text-slate-500 font-mono' }, `v${def.metadata.version}`),
        ]),
        h('p', { class: 'text-xs text-slate-500 truncate' }, def.metadata.description || `${def.states?.length ?? 0} states · ${def.agents?.length ?? 0} agents`),
      ]),
      h('div', { class: 'flex gap-2 shrink-0' }, [
        h('button', { class: 'btn-ghost text-xs', onClick: () => emit('edit') }, 'Edit'),
        h('button', { class: 'btn-primary text-xs', onClick: () => emit('run') }, 'Run'),
        h('button', { class: 'btn-danger text-xs', onClick: () => emit('delete') }, '✕'),
      ])
    ])
  }
})

export const StartRunModal = defineComponent({
  props: { definition: Object },
  emits: ['close', 'started'],
  setup(p, { emit }) {
    const input = ref('{}')
    const loading = ref(false)
    const error = ref('')
    const { startRun } = useWorkflowStore()

    async function run() {
      loading.value = true; error.value = ''
      try {
        const parsed = JSON.parse(input.value)
        const runData = await startRun(p.definition.metadata.name, p.definition.metadata.version, parsed)
        emit('started', runData)
        emit('close')
      } catch(e) {
        error.value = e.message
      } finally { loading.value = false }
    }

    return () => h('div', {
      class: 'fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50',
      onClick: (e) => e.target === e.currentTarget && emit('close')
    }, [
      h('div', { class: 'card w-[480px] p-6 space-y-4' }, [
        h('h3', { class: 'font-semibold text-slate-200' }, `Run: ${p.definition.metadata.name}`),
        h('div', [
          h('label', { class: 'text-xs text-slate-500 mb-1 block' }, 'Initial blackboard input (JSON)'),
          h('textarea', {
            class: 'input font-mono h-32 resize-none',
            value: input.value,
            onInput: e => input.value = e.target.value,
            spellcheck: false
          }),
        ]),
        error.value && h('p', { class: 'text-red-400 text-xs' }, error.value),
        h('div', { class: 'flex gap-3 justify-end' }, [
          h('button', { class: 'btn-ghost', onClick: () => emit('close') }, 'Cancel'),
          h('button', { class: 'btn-primary', onClick: run, disabled: loading.value },
            loading.value ? 'Starting…' : 'Start Run'
          ),
        ])
      ])
    ])
  }
})

import { useWorkflowStore } from '@/stores/workflows'
import { api } from '@/stores/auth'

// ── GenerateWorkflowModal ─────────────────────────────────────────────────────
// Shown when user clicks "Generate with AI".
// 1. Loads available MCP servers from the registry.
// 2. User writes a description and selects MCP servers.
// 3. Calls POST /api/v1/designer/generate, then saves the result via POST /api/v1/workflows.
// 4. Emits 'generated' so the parent can open the designer.
export const GenerateWorkflowModal = defineComponent({
  emits: ['close', 'generated'],
  setup(_, { emit }) {
    const description  = ref('')
    const mcpServers   = ref([])      // list of { name, description }
    const selectedMCP  = ref([])      // selected names
    const loading      = ref(false)
    const loadingMCP   = ref(true)
    const error        = ref('')
    const statusMsg    = ref('')

    // Load available MCP servers.
    api.get('/mcp-servers').then(res => {
      mcpServers.value = res.data
    }).catch(() => {
      // Non-fatal — just skip the server picker.
    }).finally(() => {
      loadingMCP.value = false
    })

    function toggleMCP(name) {
      const idx = selectedMCP.value.indexOf(name)
      if (idx !== -1) selectedMCP.value.splice(idx, 1)
      else selectedMCP.value.push(name)
    }

    async function generate() {
      if (!description.value.trim()) return
      loading.value = true
      error.value   = ''
      statusMsg.value = 'Asking Claude to design your workflow…'
      try {
        // 1. Generate the workflow.
        const genRes = await api.post('/designer/generate', {
          description: description.value.trim(),
          mcp_servers: selectedMCP.value,
        })
        const { yaml, definition } = genRes.data
        statusMsg.value = 'Saving workflow…'

        // 2. Save it.
        await api.post('/workflows', { yaml })

        emit('generated', {
          name:    definition.metadata.name,
          version: definition.metadata.version,
        })
      } catch (e) {
        error.value = e.response?.data?.error ?? e.message
        statusMsg.value = ''
      } finally {
        loading.value = false
      }
    }

    return () => h('div', {
      class: 'fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50',
      onClick: e => e.target === e.currentTarget && !loading.value && emit('close'),
    }, [
      h('div', { class: 'card w-[560px] max-h-[80vh] flex flex-col p-0 overflow-hidden' }, [

        // Header
        h('div', { class: 'flex items-center gap-3 px-5 py-4 border-b border-border shrink-0' }, [
          h('svg', { width: '16', height: '16', viewBox: '0 0 24 24', fill: 'none', stroke: '#a78bfa', 'stroke-width': '2' }, [
            h('path', { d: 'M12 2l2.4 7.4H22l-6.2 4.5 2.4 7.4L12 17l-6.2 4.3 2.4-7.4L2 9.4h7.6z' }),
          ]),
          h('h3', { class: 'font-semibold text-slate-200 flex-1' }, 'Generate Workflow with AI'),
          h('button', {
            class: 'text-slate-600 hover:text-slate-300 text-lg leading-none p-1',
            onClick: () => !loading.value && emit('close'),
          }, '✕'),
        ]),

        // Body
        h('div', { class: 'flex-1 overflow-y-auto p-5 space-y-4' }, [

          h('div', { class: 'space-y-1' }, [
            h('label', { class: 'text-xs text-slate-400 font-medium' }, 'Describe your business process *'),
            h('textarea', {
              class: 'input h-28 resize-none text-sm leading-relaxed',
              value: description.value,
              placeholder: 'e.g. Process customer invoices from email receipt through validation, enrichment, human review, and final posting to the ERP. Flag items over €10,000 for extra approval.',
              onInput: e => description.value = e.target.value,
              disabled: loading.value,
              autofocus: true,
            }),
          ]),

          // MCP server picker
          !loadingMCP.value && mcpServers.value.length ? h('div', { class: 'space-y-2' }, [
            h('label', { class: 'text-xs text-slate-400 font-medium' }, 'Available MCP Servers (optional)'),
            h('p', { class: 'text-[10px] text-slate-600' }, 'Select servers whose tools your agents should be able to use. Claude will decide which agents need which servers.'),
            h('div', { class: 'grid grid-cols-2 gap-1.5' }, mcpServers.value.map(s =>
              h('button', {
                key: s.name,
                class: `text-left px-2.5 py-2 rounded-lg border text-xs transition-colors ${
                  selectedMCP.value.includes(s.name)
                    ? 'border-indigo-500 bg-indigo-500/10 text-indigo-300'
                    : 'border-border bg-surface-0 text-slate-500 hover:border-border/80'
                }`,
                onClick: () => toggleMCP(s.name),
                disabled: loading.value,
              }, [
                h('div', { class: 'font-medium mb-0.5 font-mono' }, s.name),
                h('div', { class: 'text-[10px] opacity-70 leading-tight line-clamp-2' }, s.description),
              ])
            )),
          ]) : null,

          // Status / error
          loading.value && statusMsg.value
            ? h('div', { class: 'flex items-center gap-2 text-xs text-violet-400 animate-pulse' }, [
                h('span', {}, statusMsg.value),
              ])
            : null,

          error.value
            ? h('div', { class: 'px-3 py-2 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-xs' }, error.value)
            : null,
        ]),

        // Footer
        h('div', { class: 'flex gap-3 justify-end px-5 py-4 border-t border-border shrink-0' }, [
          h('button', {
            class: 'btn-ghost text-sm',
            onClick: () => emit('close'),
            disabled: loading.value,
          }, 'Cancel'),
          h('button', {
            class: `btn-primary text-sm gap-2 ${!description.value.trim() || loading.value ? 'opacity-40 cursor-not-allowed' : ''}`,
            onClick: generate,
            disabled: !description.value.trim() || loading.value,
          }, loading.value ? 'Generating…' : 'Generate'),
        ]),
      ]),
    ])
  },
})
</script>
