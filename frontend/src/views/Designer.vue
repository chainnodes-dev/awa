<template>
  <div class="flex flex-col h-full overflow-hidden">

    <!-- ── Toolbar ─────────────────────────────────────────────────────── -->
    <header class="flex items-center gap-2 px-4 py-2.5 border-b border-border shrink-0 bg-surface-1">
      <RouterLink to="/dashboard" class="btn-ghost p-1.5">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/>
        </svg>
      </RouterLink>
      <div class="w-px h-4 bg-border"/>
      <span class="text-sm font-medium text-slate-300">{{ workflowName || 'New Workflow' }}</span>
      <div v-if="isDirty" class="w-1.5 h-1.5 rounded-full bg-amber-500" title="Unsaved changes"/>

      <div class="flex-1"/>

      <!-- Auto-layout (canvas/split only) -->
      <button
        v-if="view !== 'yaml'"
        @click="applyLayout"
        class="btn-ghost text-xs gap-1"
        title="Auto-arrange nodes with dagre (A)"
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="3" width="7" height="7" rx="1"/>
          <rect x="14" y="3" width="7" height="7" rx="1"/>
          <rect x="8" y="14" width="8" height="7" rx="1"/>
          <line x1="6.5" y1="10" x2="6.5" y2="14"/>
          <line x1="17.5" y1="10" x2="17.5" y2="14"/>
          <line x1="6.5" y1="14" x2="12" y2="14"/>
          <line x1="17.5" y1="14" x2="12" y2="14"/>
        </svg>
        Layout
      </button>

      <!-- View toggle -->
      <div class="flex bg-surface-0 rounded-lg p-0.5 border border-border">
        <button
          @click="view = 'canvas'"
          :class="['btn py-1 px-2.5 text-xs', view === 'canvas' ? 'bg-surface-2 text-slate-200' : 'text-slate-500']"
        >Canvas</button>
        <button
          @click="view = 'split'"
          :class="['btn py-1 px-2.5 text-xs', view === 'split' ? 'bg-surface-2 text-slate-200' : 'text-slate-500']"
        >Split</button>
        <button
          @click="view = 'yaml'"
          :class="['btn py-1 px-2.5 text-xs', view === 'yaml' ? 'bg-surface-2 text-slate-200' : 'text-slate-500']"
        >YAML</button>
      </div>

      <button @click="addState" class="btn-ghost text-xs gap-1" title="Add state">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <circle cx="12" cy="12" r="9"/><line x1="12" y1="8" x2="12" y2="16"/><line x1="8" y1="12" x2="16" y2="12"/>
        </svg>
        Add State
      </button>
      <button @click="saveWorkflow" :disabled="saving" class="btn-primary text-xs">
        {{ saving ? 'Saving…' : 'Save' }}
      </button>
    </header>

    <!-- ── Main body ───────────────────────────────────────────────────── -->
    <div class="flex flex-1 overflow-hidden">

      <!-- YAML-only view -->
      <template v-if="view === 'yaml'">
        <div class="flex-1 flex flex-col overflow-hidden">
          <textarea
            v-model="yamlSource"
            @input="isDirty = true"
            class="flex-1 bg-surface-0 text-slate-300 font-mono text-xs p-4 resize-none outline-none border-none"
            spellcheck="false"
            placeholder="Paste or type your workflow YAML here…"
          />
          <div v-if="yamlError" class="px-4 py-2 bg-red-500/10 border-t border-red-500/30 text-red-400 text-xs font-mono">
            {{ yamlError }}
          </div>
          <div class="px-4 py-2 border-t border-border flex gap-3">
            <button @click="importYAML" class="btn-primary text-xs">Import to Canvas</button>
          </div>
        </div>
      </template>

      <!-- Canvas or Split view -->
      <template v-else>

        <!-- Canvas pane -->
        <div :class="['relative overflow-hidden', view === 'split' ? 'flex-[3]' : 'flex-1']">
          <VueFlow
            :key="flowKey"
            v-model:nodes="nodes"
            v-model:edges="edges"
            :node-types="nodeTypes"
            :edge-types="edgeTypes"
            :default-viewport="{ zoom: 1 }"
            fit-view-on-init
            class="bg-surface-0"
            @node-click="onNodeClick"
            @edge-click="onEdgeClick"
            @pane-click="onPaneClick"
            @connect="onConnect"
            @nodes-change="onNodesChange"
            @edges-change="isDirty = true"
          >
            <Background pattern-color="#2a2d3a" :gap="24" :size="1" />
            <Controls />
            <MiniMap v-if="view === 'canvas'" node-color="#1a1d27" mask-color="rgba(12,14,20,0.7)" />
          </VueFlow>
        </div>

        <!-- YAML pane (split view only) -->
        <div v-if="view === 'split'" class="flex-[2] border-l border-border flex flex-col min-h-0 min-w-0">
          <div class="px-3 py-2 border-b border-border flex items-center gap-2 shrink-0 bg-surface-1">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#64748b" stroke-width="2">
              <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
            </svg>
            <span class="text-xs text-slate-500 font-medium flex-1">YAML — live mirror</span>
            <span v-if="yamlError" class="text-[10px] text-red-400 truncate max-w-[160px]" :title="yamlError">
              ⚠ {{ yamlError }}
            </span>
          </div>
          <textarea
            v-model="yamlSource"
            @input="onYamlInput"
            class="flex-1 bg-surface-0 text-slate-300 font-mono text-[11px] leading-relaxed p-3 resize-none outline-none border-none min-h-0"
            spellcheck="false"
          />
        </div>

        <!-- Right property panel (canvas view only) -->
        <aside v-if="view === 'canvas'" class="w-72 border-l border-border flex flex-col overflow-hidden bg-surface-1">

          <!-- Pending connection: user dragged an edge, waiting for trigger name -->
          <PendingConnectionPanel
            v-if="pendingConnection"
            :connection="pendingConnection"
            @confirm="confirmConnection"
            @cancel="cancelConnection"
          />

          <!-- Workflow metadata (no selection) -->
          <div v-else-if="!selected" class="p-4 space-y-4 overflow-y-auto flex-1">
            <div class="space-y-3">
              <h3 class="text-xs font-semibold text-slate-400 uppercase tracking-wider">Workflow</h3>
              <div class="space-y-2">
                <input v-model="meta.name"        class="input" placeholder="Name" @input="isDirty=true"/>
                <input v-model="meta.version"     class="input" placeholder="Version (e.g. 1.0.0)" @input="isDirty=true"/>
                <textarea v-model="meta.description" class="input h-16 resize-none" placeholder="Description" @input="isDirty=true"/>
              </div>
            </div>

            <div class="space-y-2">
              <h3 class="text-xs font-semibold text-slate-400 uppercase tracking-wider">Blackboard Schema</h3>
              <div v-for="(field, key) in bbSchema" :key="key"
                   class="flex items-center gap-2 group">
                <span class="text-xs font-mono text-slate-300 w-28 truncate">{{ key }}</span>
                <span class="text-xs text-slate-500 flex-1">{{ field.type }}</span>
                <button @click="removeBBField(key)" class="opacity-0 group-hover:opacity-100 text-slate-600 hover:text-red-400 text-xs transition">✕</button>
              </div>
              <button @click="showBBModal = true" class="btn-ghost text-xs w-full justify-center border border-dashed border-border hover:border-accent/50">
                + Add Field
              </button>
            </div>

            <!-- Agents summary -->
            <div v-if="agents.length" class="space-y-2">
              <h3 class="text-xs font-semibold text-slate-400 uppercase tracking-wider">Agents ({{ agents.length }})</h3>
              <div v-for="a in agents" :key="a.name" class="text-xs text-slate-500 flex items-center gap-2 py-1">
                <span class="font-mono text-slate-400 truncate flex-1">{{ a.name }}</span>
                <span v-if="a.model" class="text-[10px] text-slate-600 truncate">{{ a.model.split('/').pop() }}</span>
              </div>
            </div>
          </div>

          <!-- State selected -->
          <!-- :key forces a full remount when the selected node changes,
               so StatePanel.setup() re-runs and the form is reinitialised. -->
          <StatePanel
            v-else-if="selected?.type === 'node'"
            :key="selected.id"
            :node="selected.data"
            :agent-def="getAgent(selected.data.agent)"
            @update="updateNode"
            @delete="deleteSelected"
          />

          <!-- Edge selected -->
          <EdgePanel
            v-else-if="selected?.type === 'edge'"
            :key="selected.id"
            :edge="selected.data"
            @update="updateEdge"
            @delete="deleteSelected"
          />
        </aside>
      </template>
    </div>

    <!-- ── Modals ──────────────────────────────────────────────────────── -->
    <BBFieldModal  v-if="showBBModal"   @close="showBBModal = false"   @add="addBBField" />
    <AddStateModal v-if="showAddState" @close="showAddState = false" @add="onAddState" />
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, nextTick, shallowRef } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { Background }  from '@vue-flow/background'
import { Controls }    from '@vue-flow/controls'
import { MiniMap }     from '@vue-flow/minimap'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

import StateNode       from '@/components/designer/StateNode.vue'
import TransitionEdge  from '@/components/designer/TransitionEdge.vue'
import { useWorkflowStore } from '@/stores/workflows'
import { useLayout }   from '@/composables/useLayout'
import yaml from 'js-yaml'

const route  = useRoute()
const router = useRouter()
const wfStore = useWorkflowStore()
const { fitView } = useVueFlow()
const { layout }  = useLayout()

// ── Reactive state ──────────────────────────────────────────────────────────

const nodes   = ref([])
const edges   = ref([])
const agents  = ref([])   // [{name, model, task_queue, config: {prompt, mcp_servers}}]
const meta    = reactive({ name: '', version: '1.0.0', description: '' })
const bbSchema = reactive({})

const view          = ref('canvas')   // 'canvas' | 'split' | 'yaml'
const yamlSource    = ref('')
const yamlError     = ref('')
const saving        = ref(false)
const isDirty       = ref(false)
const selected      = ref(null)       // { type: 'node'|'edge', id, data }
const pendingConnection = ref(null)   // { source, target } — drag completed, awaiting trigger

const showBBModal   = ref(false)
const showAddState  = ref(false)

const nodeTypes = shallowRef({ stateNode: StateNode })
const edgeTypes = shallowRef({ transitionEdge: TransitionEdge })

// Bumped after async data load to remount VueFlow with nodes already populated,
// so fit-view-on-init fires when there is actually something to fit.
const flowKey = ref(0)

const workflowName = computed(() => meta.name || '')

// ── Load workflow ───────────────────────────────────────────────────────────

onMounted(async () => {
  const { name, version } = route.params
  if (name && version) {
    const { definition, yaml: src } = await wfStore.fetchOne(name, version)
    loadDefinition(definition, src)

    // VueFlow mounts immediately (canvas is the default view) with empty nodes,
    // so fit-view-on-init fires before data arrives and does nothing.
    // Bumping the key forces a remount now that nodes are populated.
    await nextTick()
    flowKey.value++

    const hasExplicit = (definition.states ?? []).some(
      s => s.position && (s.position.x !== 200 || s.position.y !== 100)
    )
    if (!hasExplicit && nodes.value.length > 1) {
      await nextTick()
      applyLayout(false)
    }
  }
})

function loadDefinition(def, src, updateYaml = true) {
  meta.name        = def.metadata?.name        ?? ''
  meta.version     = def.metadata?.version     ?? '1.0.0'
  meta.description = def.metadata?.description ?? ''

  // Clear and repopulate bbSchema
  Object.keys(bbSchema).forEach(k => delete bbSchema[k])
  Object.assign(bbSchema, def.blackboard?.schema ?? {})

  nodes.value = (def.states ?? []).map(s => ({
    id:       s.name,
    type:     'stateNode',
    position: s.position ?? { x: 200, y: 100 },
    data:     { ...s },   // instructions, script, agent, etc. all pass through
  }))

  edges.value = (def.transitions ?? []).map(t => ({
    id: `${t.from}__${t.trigger}__${t.to}`,
    source:  t.from,
    target:  t.to,
    type:    'transitionEdge',
    label:   t.trigger,
    data:    { guard: t.guard ?? '' },
  }))

  agents.value = def.agents ?? []

  if (updateYaml && src) yamlSource.value = src
  isDirty.value = false
}

// ── Auto-layout ─────────────────────────────────────────────────────────────

function applyLayout(markDirty = true) {
  if (!nodes.value.length) return
  nodes.value = layout(nodes.value, edges.value)
  nextTick(() => fitView({ padding: 0.15, duration: 300 }))
  if (markDirty) isDirty.value = true
}

// ── Canvas events ───────────────────────────────────────────────────────────

function onNodeClick({ node }) {
  pendingConnection.value = null
  selected.value = { type: 'node', id: node.id, data: { ...node.data } }
}

function onEdgeClick({ edge }) {
  pendingConnection.value = null
  selected.value = {
    type: 'edge', id: edge.id,
    data: { label: edge.label, guard: edge.data?.guard ?? '', from: edge.source, to: edge.target },
  }
}

function onPaneClick() {
  selected.value = null
  // Don't clear pendingConnection — user might have missed clicking a node
}

function onNodesChange() {
  isDirty.value = true
}

// Replace prompt() with a proper side panel
function onConnect(params) {
  pendingConnection.value = { source: params.source, target: params.target }
  selected.value = null
}

function confirmConnection({ trigger, guard }) {
  const { source, target } = pendingConnection.value
  edges.value.push({
    id:     `${source}__${trigger}__${target}`,
    source, target,
    type:   'transitionEdge',
    label:  trigger,
    data:   { guard: guard || '' },
  })
  pendingConnection.value = null
  isDirty.value = true
}

function cancelConnection() {
  pendingConnection.value = null
}

// ── Node / edge updates ─────────────────────────────────────────────────────

function getAgent(name) {
  if (!name) return null
  return agents.value.find(a => a.name === name) ?? null
}

function upsertAgent(def) {
  if (!def || !def.name) return
  const idx = agents.value.findIndex(a => a.name === def.name)
  if (idx !== -1) agents.value[idx] = def
  else agents.value.push(def)
}

// Receives { node, agentDef } from StatePanel
function updateNode({ node, agentDef }) {
  const idx = nodes.value.findIndex(n => n.id === selected.value.id)
  if (idx === -1) return

  const newId = node.name
  const oldId = nodes.value[idx].id
  nodes.value[idx] = { ...nodes.value[idx], id: newId, data: node }

  if (newId !== oldId) {
    edges.value = edges.value.map(e => ({
      ...e,
      source: e.source === oldId ? newId : e.source,
      target: e.target === oldId ? newId : e.target,
    }))
  }

  upsertAgent(agentDef)
  selected.value = null
  isDirty.value = true
}

function updateEdge(data) {
  const idx = edges.value.findIndex(e => e.id === selected.value.id)
  if (idx !== -1) {
    edges.value[idx] = { ...edges.value[idx], label: data.label, data: { guard: data.guard } }
    selected.value = null
    isDirty.value = true
  }
}

function deleteSelected() {
  if (!selected.value) return
  if (selected.value.type === 'node') {
    const id = selected.value.id
    nodes.value = nodes.value.filter(n => n.id !== id)
    edges.value = edges.value.filter(e => e.source !== id && e.target !== id)
  } else {
    edges.value = edges.value.filter(e => e.id !== selected.value.id)
  }
  selected.value = null
  isDirty.value = true
}

// ── Add state ───────────────────────────────────────────────────────────────

function addState() { showAddState.value = true }

function onAddState(stateData) {
  const existingX = nodes.value.map(n => n.position.x)
  const x = existingX.length ? Math.max(...existingX) + 200 : 200
  nodes.value.push({
    id:       stateData.name,
    type:     'stateNode',
    position: { x, y: 200 },
    data:     { ...stateData },
  })
  showAddState.value = false
  isDirty.value = true
}

// ── Blackboard ──────────────────────────────────────────────────────────────

function addBBField({ name, type, required }) {
  bbSchema[name] = { type, required }
  showBBModal.value = false
  isDirty.value = true
}
function removeBBField(key) {
  delete bbSchema[key]
  isDirty.value = true
}

// ── Build / save ─────────────────────────────────────────────────────────────

function buildDefinition() {
  const cleanAgents = agents.value
    .filter(a => a.name)
    .map(a => {
      const obj = { name: a.name }
      if (a.model)       obj.model      = a.model
      if (a.task_queue)  obj.task_queue = a.task_queue
      const prompt = a.config?.prompt
      const mcp    = a.config?.mcp_servers
      if (prompt || mcp) {
        obj.config = {}
        if (prompt) obj.config.prompt      = prompt
        if (mcp)    obj.config.mcp_servers = mcp
      }
      return obj
    })

  return {
    apiVersion: 'asm/v1',
    kind:       'Workflow',
    metadata:   { name: meta.name, version: meta.version, description: meta.description },
    blackboard: { schema: { ...bbSchema } },
    states:     nodes.value.map(n => ({
      name:         n.id,
      type:         n.data.type         ?? 'intermediate',
      instructions: n.data.instructions || undefined,
      agent:        n.data.agent        || undefined,
      script:       n.data.script       || undefined,
      timeout:      n.data.timeout      || undefined,
      on_timeout:   n.data.on_timeout   || undefined,
      position:     n.position,
    })),
    transitions: edges.value.map(e => ({
      from:    e.source,
      to:      e.target,
      trigger: e.label,
      guard:   e.data?.guard || undefined,
    })),
    agents: cleanAgents,
  }
}

async function saveWorkflow() {
  if (!meta.name || !meta.version) {
    alert('Workflow name and version are required')
    return
  }
  saving.value = true
  try {
    const def = buildDefinition()
    const yamlStr = yaml.dump(def, { lineWidth: 120 })
    await wfStore.save(yamlStr)
    isDirty.value = false
    router.replace(`/designer/${meta.name}/${meta.version}`)
  } catch (e) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

// ── YAML import / split-view sync ────────────────────────────────────────────

function importYAML() {
  yamlError.value = ''
  try {
    const def = yaml.load(yamlSource.value)
    loadDefinition(def, null, false)
    view.value = 'canvas'
  } catch (e) {
    yamlError.value = e.message
  }
}

// Flag to prevent canvas → YAML watch from firing when the canvas was updated
// as a result of a YAML parse (would overwrite what the user typed)
let fromYamlUpdate = false
let yamlDebounce   = null

// Canvas → YAML: sync to YAML textarea when in split view
watch([nodes, edges], () => {
  if (fromYamlUpdate) return
  isDirty.value = true
  if (view.value === 'split') {
    yamlSource.value = yaml.dump(buildDefinition(), { lineWidth: 120 })
  }
}, { deep: true })

watch(meta, () => {
  if (fromYamlUpdate) return
  isDirty.value = true
  if (view.value === 'split') {
    yamlSource.value = yaml.dump(buildDefinition(), { lineWidth: 120 })
  }
}, { deep: true })

// YAML → Canvas: debounced parse when user edits the YAML textarea in split view
function onYamlInput() {
  isDirty.value = true
  if (view.value !== 'split') return
  yamlError.value = ''
  clearTimeout(yamlDebounce)
  yamlDebounce = setTimeout(() => {
    try {
      const def = yaml.load(yamlSource.value)
      if (!def) return
      fromYamlUpdate = true
      loadDefinition(def, null, false)  // false = don't overwrite YAML textarea
      nextTick(() => { fromYamlUpdate = false })
      yamlError.value = ''
    } catch (e) {
      yamlError.value = e.message
    }
  }, 400)
}

// Switching views: snapshot YAML from current canvas state
watch(view, v => {
  if (v === 'yaml' || v === 'split') {
    yamlSource.value = yaml.dump(buildDefinition(), { lineWidth: 120 })
  }
})
</script>

<!-- ── Inline sub-components ────────────────────────────────────────────────── -->
<script>
import { defineComponent, h, ref, reactive } from 'vue'

// ── Shared helper ────────────────────────────────────────────────────────────
function field(label, input) {
  return h('div', { class: 'space-y-1' }, [
    h('label', { class: 'text-xs text-slate-500' }, label),
    input,
  ])
}

// ── PendingConnectionPanel ───────────────────────────────────────────────────
// Shown in the right panel when the user has dragged an edge but hasn't named it yet
export const PendingConnectionPanel = defineComponent({
  props: { connection: Object },
  emits: ['confirm', 'cancel'],
  setup(p, { emit }) {
    const trigger = ref('')
    const guard   = ref('')

    function onConfirm() {
      if (!trigger.value.trim()) return
      emit('confirm', { trigger: trigger.value.trim(), guard: guard.value.trim() })
    }

    return () => h('div', { class: 'p-4 space-y-3 flex-1 overflow-y-auto' }, [
      h('div', { class: 'flex items-center gap-2 mb-1' }, [
        h('span', { class: 'text-xs font-semibold text-slate-400 uppercase tracking-wider flex-1' }, 'New Transition'),
        h('button', { class: 'text-slate-600 hover:text-slate-300 text-sm leading-none p-1', onClick: () => emit('cancel') }, '✕'),
      ]),

      // Arrow
      h('div', { class: 'flex items-center gap-2 py-2 px-3 rounded-lg bg-surface-0 border border-border' }, [
        h('span', { class: 'text-xs font-mono text-indigo-400 truncate' }, p.connection.source),
        h('svg', { width: '16', height: '16', viewBox: '0 0 24 24', fill: 'none', stroke: '#4b5563', 'stroke-width': '2' }, [
          h('line', { x1: '5', y1: '12', x2: '19', y2: '12' }),
          h('polyline', { points: '12 5 19 12 12 19' }),
        ]),
        h('span', { class: 'text-xs font-mono text-indigo-400 truncate' }, p.connection.target),
      ]),

      field('Trigger name *',
        h('input', {
          class: 'input font-mono autofocus',
          value: trigger.value,
          placeholder: 'e.g. validation_passed',
          onInput:   e => trigger.value = e.target.value,
          onKeydown: e => e.key === 'Enter' && onConfirm(),
        })
      ),

      field('Guard expression',
        h('input', {
          class: 'input font-mono',
          value: guard.value,
          placeholder: 'e.g. amount > 0',
          onInput:   e => guard.value = e.target.value,
          onKeydown: e => e.key === 'Enter' && onConfirm(),
        })
      ),

      h('p', { class: 'text-[10px] text-slate-600' }, 'Leave guard empty to always follow this transition.'),

      h('div', { class: 'flex gap-2 pt-1' }, [
        h('button', { class: 'btn-ghost text-xs flex-1', onClick: () => emit('cancel') }, 'Cancel'),
        h('button', {
          class:    `btn-primary text-xs flex-1 ${!trigger.value ? 'opacity-40 cursor-not-allowed' : ''}`,
          disabled: !trigger.value,
          onClick:  onConfirm,
        }, 'Create'),
      ]),
    ])
  },
})

// ── StatePanel ───────────────────────────────────────────────────────────────
// Shown when a state node is selected. Includes agent config when agent is set.
export const StatePanel = defineComponent({
  props: {
    node:     Object,
    agentDef: { type: Object, default: null },
  },
  emits: ['update', 'delete'],
  setup(p, { emit }) {
    const STATE_TYPES = ['initial', 'intermediate', 'hitl', 'terminal', 'script']

    const form = reactive({
      name:         p.node.name         ?? '',
      type:         p.node.type         ?? 'intermediate',
      instructions: p.node.instructions ?? '',
      agent:        p.node.agent        ?? '',
      timeout:      p.node.timeout      ?? '',
      on_timeout:   p.node.on_timeout   ?? '',
      // Script state fields
      script_trigger: p.node.script?.trigger ?? '',
      script_updates: p.node.script?.updates
        ? Object.entries(p.node.script.updates).map(([k, v]) => ({ key: k, expr: v }))
        : [],
    })

    const agentForm = reactive({
      model:      p.agentDef?.model             ?? '',
      task_queue: p.agentDef?.task_queue        ?? '',
      prompt:     p.agentDef?.config?.prompt    ?? '',
      mcp_servers: p.agentDef?.config?.mcp_servers ?? '',
    })

    function onApply() {
      // Build script definition when type is 'script'
      let scriptDef = undefined
      if (form.type === 'script' && form.script_trigger) {
        scriptDef = {
          trigger: form.script_trigger,
          updates: form.script_updates.length
            ? Object.fromEntries(form.script_updates.filter(r => r.key).map(r => [r.key, r.expr]))
            : undefined,
        }
      }

      const nodeData = {
        name:         form.name,
        type:         form.type,
        instructions: form.instructions || undefined,
        agent:        form.type !== 'script' ? (form.agent || undefined) : undefined,
        script:       scriptDef,
        timeout:      form.timeout    || undefined,
        on_timeout:   form.on_timeout || undefined,
      }
      const agentData = (form.type !== 'script' && form.agent) ? {
        name:       form.agent,
        model:      agentForm.model       || undefined,
        task_queue: agentForm.task_queue  || undefined,
        config: (agentForm.prompt || agentForm.mcp_servers) ? {
          prompt:      agentForm.prompt      || undefined,
          mcp_servers: agentForm.mcp_servers || undefined,
        } : undefined,
      } : null
      emit('update', { node: nodeData, agentDef: agentData })
    }

    return () => {
      // Script editor (for type === 'script')
      const scriptSection = form.type === 'script' ? [
        h('div', { class: 'pt-3 border-t border-border space-y-2.5' }, [
          h('h4', { class: 'text-xs font-semibold text-amber-500/80 uppercase tracking-wider' }, 'Script'),
          h('p', { class: 'text-[10px] text-slate-600 leading-relaxed' },
            'Expressions use expr-lang. Blackboard fields are available as variables.'
          ),
          field('Trigger Expression *',
            h('input', {
              class: 'input font-mono text-xs',
              value: form.script_trigger,
              placeholder: 'amount > 1000 ? "needs_review" : "auto_approve"',
              onInput: e => form.script_trigger = e.target.value,
            })
          ),
          h('div', { class: 'space-y-1' }, [
            h('div', { class: 'flex items-center justify-between' }, [
              h('label', { class: 'text-xs text-slate-500' }, 'Blackboard Updates'),
              h('button', {
                class: 'text-[10px] text-indigo-400 hover:text-indigo-300',
                onClick: () => form.script_updates.push({ key: '', expr: '' }),
              }, '+ Add'),
            ]),
            ...form.script_updates.map((row, i) =>
              h('div', { key: i, class: 'flex gap-1.5 items-center' }, [
                h('input', {
                  class: 'input font-mono text-xs flex-1',
                  value: row.key,
                  placeholder: 'field',
                  onInput: e => form.script_updates[i].key = e.target.value,
                }),
                h('span', { class: 'text-slate-600 text-xs shrink-0' }, '='),
                h('input', {
                  class: 'input font-mono text-xs flex-[2]',
                  value: row.expr,
                  placeholder: 'amount * 0.2',
                  onInput: e => form.script_updates[i].expr = e.target.value,
                }),
                h('button', {
                  class: 'text-slate-600 hover:text-red-400 text-xs shrink-0 px-1',
                  onClick: () => form.script_updates.splice(i, 1),
                }, '✕'),
              ])
            ),
          ]),
        ])
      ] : []

      // Agent config section (for non-script types with an agent name set)
      const agentSection = (form.type !== 'script' && form.agent) ? [
        h('div', { class: 'pt-3 border-t border-border space-y-2.5' }, [
          h('h4', { class: 'text-xs font-semibold text-slate-500 uppercase tracking-wider' }, 'Agent Config'),
          field('Model',
            h('input', {
              class: 'input text-xs',
              value: agentForm.model,
              placeholder: 'claude-sonnet-4-6',
              onInput: e => agentForm.model = e.target.value,
            })
          ),
          field('Task Queue',
            h('input', {
              class: 'input font-mono text-xs',
              value: agentForm.task_queue,
              placeholder: 'asm-workers (default)',
              onInput: e => agentForm.task_queue = e.target.value,
            })
          ),
          field('MCP Servers',
            h('input', {
              class: 'input font-mono text-xs',
              value: agentForm.mcp_servers,
              placeholder: '{{ env.MCP_URL }}',
              onInput: e => agentForm.mcp_servers = e.target.value,
            })
          ),
          field('System Prompt',
            h('textarea', {
              class: 'input font-mono text-[11px] leading-relaxed resize-none h-32',
              value: agentForm.prompt,
              placeholder: 'You are a helpful assistant…',
              onInput: e => agentForm.prompt = e.target.value,
            })
          ),
        ])
      ] : []

      return h('div', { class: 'p-4 space-y-2.5 flex-1 overflow-y-auto' }, [
        // Header
        h('div', { class: 'flex items-center gap-2 mb-1' }, [
          h('span', { class: 'text-xs font-semibold text-slate-400 uppercase tracking-wider flex-1' }, 'State'),
          h('button', { class: 'btn-danger text-xs py-1', onClick: () => emit('delete') }, 'Delete'),
        ]),

        field('Name',
          h('input', { class: 'input', value: form.name, onInput: e => form.name = e.target.value })
        ),
        field('Type',
          h('select', { class: 'input', value: form.type, onChange: e => form.type = e.target.value },
            STATE_TYPES.map(t => h('option', { value: t }, t))
          )
        ),

        // Instructions — state-level context for the LLM
        field('Instructions',
          h('textarea', {
            class:       'input text-xs leading-relaxed resize-none h-16',
            value:       form.instructions,
            placeholder: 'Describe what should happen in this state…',
            onInput:     e => form.instructions = e.target.value,
          })
        ),

        // Agent name field (hidden for script states)
        ...(form.type !== 'script' ? [
          field('Agent',
            h('input', {
              class:       'input font-mono',
              value:       form.agent,
              placeholder: 'agent_name (optional)',
              onInput:     e => form.agent = e.target.value,
            })
          ),
        ] : []),

        field('Timeout',
          h('input', {
            class:       'input',
            value:       form.timeout,
            placeholder: 'e.g. 30m',
            onInput:     e => form.timeout = e.target.value,
          })
        ),
        ...(form.timeout ? [
          field('On Timeout',
            h('input', {
              class:       'input font-mono',
              value:       form.on_timeout,
              placeholder: 'trigger_name',
              onInput:     e => form.on_timeout = e.target.value,
            })
          ),
        ] : []),

        ...scriptSection,
        ...agentSection,

        h('button', { class: 'btn-primary w-full mt-1', onClick: onApply }, 'Apply'),
      ])
    }
  },
})

// ── EdgePanel ────────────────────────────────────────────────────────────────
export const EdgePanel = defineComponent({
  props:  { edge: Object },
  emits:  ['update', 'delete'],
  setup(p, { emit }) {
    const form = reactive({ ...p.edge })
    return () => h('div', { class: 'p-4 space-y-3 flex-1 overflow-y-auto' }, [
      h('div', { class: 'flex items-center gap-2 mb-1' }, [
        h('span', { class: 'text-xs font-semibold text-slate-400 uppercase tracking-wider flex-1' }, 'Transition'),
        h('button', { class: 'btn-danger text-xs py-1', onClick: () => emit('delete') }, 'Delete'),
      ]),
      h('p', { class: 'text-[10px] font-mono text-slate-600 bg-surface-0 px-2 py-1.5 rounded border border-border' },
        `${form.from}  →  ${form.to}`
      ),
      field('Trigger',
        h('input', { class: 'input font-mono', value: form.label, onInput: e => form.label = e.target.value })
      ),
      field('Guard',
        h('input', {
          class:       'input font-mono',
          value:       form.guard ?? '',
          placeholder: 'e.g. amount > 0',
          onInput:     e => form.guard = e.target.value,
        })
      ),
      h('p', { class: 'text-[10px] text-slate-600' }, 'Leave guard empty to always follow this transition.'),
      h('button', { class: 'btn-primary w-full mt-1', onClick: () => emit('update', { ...form }) }, 'Apply'),
    ])
  },
})

// ── BBFieldModal ──────────────────────────────────────────────────────────────
export const BBFieldModal = defineComponent({
  emits: ['close', 'add'],
  setup(_, { emit }) {
    const name     = ref('')
    const type     = ref('string')
    const required = ref(false)
    return () => h('div', {
      class:   'fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50',
      onClick: e => e.target === e.currentTarget && emit('close'),
    }, [
      h('div', { class: 'card w-80 p-5 space-y-3' }, [
        h('h3', { class: 'font-semibold text-slate-200 text-sm' }, 'Add Blackboard Field'),
        h('input', { class: 'input', placeholder: 'Field name', value: name.value, onInput: e => name.value = e.target.value }),
        h('select', { class: 'input', value: type.value, onChange: e => type.value = e.target.value },
          ['string', 'number', 'bool', 'object'].map(t => h('option', { value: t }, t))
        ),
        h('label', { class: 'flex items-center gap-2 text-sm text-slate-400 cursor-pointer' }, [
          h('input', { type: 'checkbox', checked: required.value, onChange: e => required.value = e.target.checked }),
          'Required',
        ]),
        h('div', { class: 'flex gap-2 justify-end' }, [
          h('button', { class: 'btn-ghost', onClick: () => emit('close') }, 'Cancel'),
          h('button', {
            class:   'btn-primary',
            onClick: () => name.value && emit('add', { name: name.value, type: type.value, required: required.value }),
          }, 'Add'),
        ]),
      ]),
    ])
  },
})

// ── AddStateModal ────────────────────────────────────────────────────────────
export const AddStateModal = defineComponent({
  emits: ['close', 'add'],
  setup(_, { emit }) {
    const name = ref('')
    const type = ref('intermediate')
    return () => h('div', {
      class:   'fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50',
      onClick: e => e.target === e.currentTarget && emit('close'),
    }, [
      h('div', { class: 'card w-80 p-5 space-y-3' }, [
        h('h3', { class: 'font-semibold text-slate-200 text-sm' }, 'Add State'),
        h('input', {
          class:       'input',
          placeholder: 'State name (e.g. VALIDATING)',
          value:       name.value,
          onInput:     e => name.value = e.target.value,
          onKeydown:   e => e.key === 'Enter' && name.value && emit('add', { name: name.value, type: type.value }),
        }),
        h('select', { class: 'input', value: type.value, onChange: e => type.value = e.target.value },
          ['initial', 'intermediate', 'hitl', 'terminal'].map(t => h('option', { value: t }, t))
        ),
        h('div', { class: 'flex gap-2 justify-end' }, [
          h('button', { class: 'btn-ghost', onClick: () => emit('close') }, 'Cancel'),
          h('button', {
            class:   'btn-primary',
            onClick: () => name.value && emit('add', { name: name.value, type: type.value }),
          }, 'Add'),
        ]),
      ]),
    ])
  },
})
</script>
