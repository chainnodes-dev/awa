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
      <!-- Version selector -->
      <template v-if="versions.length > 1">
        <select
          :value="meta.versionNumber"
          @change="switchVersion(Number($event.target.value))"
          class="bg-surface-0 border border-border rounded px-2 py-1 text-xs text-slate-400 font-mono cursor-pointer hover:border-slate-500 outline-none"
        >
          <option v-for="v in versions" :key="v.version_number" :value="v.version_number">
            v{{ v.version_number }}
          </option>
        </select>
      </template>
      <span v-else-if="meta.versionNumber" class="text-xs text-slate-500 font-mono">v{{ meta.versionNumber }}</span>
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
      <button @click="onRunClick" class="btn-ghost text-xs gap-1 border border-border/50 hover:bg-indigo-500/10 hover:text-indigo-400 transition-colors">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
        </svg>
        Run
      </button>

      <button @click="saveWorkflow" :disabled="saving" class="btn-primary text-xs px-4">
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

        <!-- Right panel with tabs (canvas view only) -->
        <aside v-if="view === 'canvas'" class="w-80 border-l border-border flex flex-col overflow-hidden bg-surface-1">

          <!-- Tab bar -->
          <div class="flex border-b border-border shrink-0">
            <button
              @click="rightTab = 'properties'"
              :class="['flex-1 py-2 text-xs font-medium transition-colors', rightTab === 'properties' ? 'text-slate-200 border-b-2 border-accent' : 'text-slate-500 hover:text-slate-400']"
            >Properties</button>
            <button
              @click="rightTab = 'ai'"
              :class="['flex-1 py-2 text-xs font-medium transition-colors flex items-center justify-center gap-1.5', rightTab === 'ai' ? 'text-slate-200 border-b-2 border-violet-500' : 'text-slate-500 hover:text-slate-400']"
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 2l2.4 7.4H22l-6.2 4.5 2.4 7.4L12 17l-6.2 4.3 2.4-7.4L2 9.4h7.6z"/>
              </svg>
              AI Assistant
            </button>
          </div>

          <!-- Properties tab -->
          <template v-if="rightTab === 'properties'">
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
                  <div v-if="meta.versionNumber" class="flex items-center gap-2 px-3 py-2 rounded-lg bg-surface-0 border border-border">
                    <span class="text-xs text-slate-500">Version</span>
                    <span class="text-xs font-mono text-slate-300">v{{ meta.versionNumber }}</span>
                    <span class="text-[10px] text-slate-600 flex-1 text-right">auto-incremented on save</span>
                  </div>
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

              <!-- System Prompt (relocated to bottom) -->
              <div class="space-y-1 pt-2 border-t border-border/50">
                <label class="text-[10px] text-slate-500 font-semibold uppercase tracking-wider">System Prompt (Global Personality)</label>
                <textarea v-model="meta.systemPrompt" class="input h-32 font-mono text-[11px] leading-relaxed resize-none" placeholder="You are a helpful assistant…" @input="isDirty=true"/>
              </div>
            </div>

            <!-- State selected -->
            <StatePanel
              v-else-if="selected?.type === 'node'"
              :key="'node-' + selected.id"
              :node="selected.data"
              :edges="edges"
              :agent-def="getAgent(selected.data.agent)"
              :bb-schema="bbSchema"
              @update="updateNode"
              @delete="deleteSelected"
            />

            <!-- Edge selected -->
            <EdgePanel
              v-else-if="selected?.type === 'edge'"
              :key="'edge-' + selected.id"
              :edge="selected.data"
              @update="updateEdge"
              @delete="deleteSelected"
            />
          </template>

          <!-- AI Assistant tab -->
          <AIChatPanel
            v-else
            :yaml-source="yamlSource"
            :workflow-name="meta.name"
            :workflow-description="meta.description"
            @apply="onAIApply"
          />
        </aside>
      </template>
    </div>

    <!-- ── Modals ──────────────────────────────────────────────────────── -->
    <BBFieldModal  v-if="showBBModal"   @close="showBBModal = false"   @add="addBBField" />
    <AddStateModal v-if="showAddState" @close="showAddState = false" @add="onAddState" />
    <!-- Start Run Modal -->
    <StartRunModal
      v-if="showRunModal"
      :definition="buildDefinition()"
      @close="showRunModal = false"
      @started="(run) => router.push(`/monitor/${run.id}`)"
    />
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

import StartRunModal from '@/components/shared/StartRunModal.vue'

const route  = useRoute()
const router = useRouter()
const wfStore = useWorkflowStore()
const { fitView } = useVueFlow()
const { layout }  = useLayout()

// ── Reactive state ──────────────────────────────────────────────────────────

const nodes   = ref([])
const edges   = ref([])
const agents  = ref([])   // [{name, model, task_queue, config: {prompt, mcp_servers}}]
const meta    = reactive({ name: '', version: '1.0.0', versionNumber: 0, description: '', systemPrompt: '' })
const bbSchema = reactive({})
const versions = ref([])   // [{version_number, version, created_at}]

const view          = ref('canvas')   // 'canvas' | 'split' | 'yaml'
const rightTab      = ref('properties') // 'properties' | 'ai'
const yamlSource    = ref('')
const yamlError     = ref('')
const saving        = ref(false)
const isDirty       = ref(false)
const showRunModal = ref(false)
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
    try {
      const { definition, yaml: src } = await wfStore.fetchOne(name, version)
      loadDefinition(definition, src)

      // Load version history for this workflow.
      try { versions.value = await wfStore.fetchVersions(name) } catch { /* ignore */ }

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
    } catch (e) {
      console.error('Failed to load workflow:', e)
      alert(`Could not load workflow "${name}@${version}". It may have been saved with an older version format. Try deleting and re-creating it.`)
    }
  }
})

function loadDefinition(def, src, updateYaml = true) {
  if (src && updateYaml) yamlSource.value = src

  // Clear existing state to avoid pollution
  meta.name          = def.metadata?.name           ?? ''
  meta.version       = def.metadata?.version        ?? '1.0.0'
  meta.versionNumber = def.metadata?.version_number ?? 0
  meta.description   = def.metadata?.description    ?? ''
  meta.systemPrompt  = def.metadata?.system_prompt   ?? ''

  // Reset blackboard
  for (const k in bbSchema) delete bbSchema[k]
  if (def.blackboard?.schema) {
    Object.assign(bbSchema, def.blackboard.schema)
  }

  // Map nodes with "data washing" to ensure clean state
  nodes.value = (def.states ?? []).map(s => ({
    id:       s.name,
    type:     'stateNode',
    position: s.position ?? { x: 200, y: 100 },
    data:     { ...s },   // instructions, script, agent, wait condition, etc.
  }))

  // Collect transitions from top-level or nested in states
  const rawTransitions = [...(def.transitions ?? [])]
  if (def.states) {
    def.states.forEach(s => {
      if (s.transitions) rawTransitions.push(...s.transitions.map(t => ({ ...t, from: s.name })))
    })
  }

  edges.value = rawTransitions.flatMap(t => {
    if (t.to_nodes && t.to_nodes.length > 0) {
      return t.to_nodes.map(target => ({
        id: `${t.from}__${t.trigger}__${target}`,
        source:  t.from,
        target:  target,
        type:    'transitionEdge',
        label:   t.trigger,
        data:    { guard: t.guard ?? '' },
      }))
    }
    const target = (t.to || t.target || '')
    return [{
      id: `${t.from}__${t.trigger}__${target}`,
      source:  t.from,
      target:  target,
      type:    'transitionEdge',
      label:   t.trigger,
      data:    { guard: t.guard ?? '' },
    }]
  })

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
  const currentSelected = selected.value
  if (!currentSelected || currentSelected.type !== 'node') return

  const idx = nodes.value.findIndex(n => n.id === currentSelected.id)
  if (idx === -1) return

  const newId = node.name
  const oldId = nodes.value[idx].id

  // If renaming, check for collisions
  if (newId !== oldId && nodes.value.some(n => n.id === newId)) {
    alert(`A state named "${newId}" already exists.`)
    return
  }

  // Update the node in the array.
  // We preserve the existing object properties (like position, selected, etc.)
  // but update the id and data.
  if (newId === oldId) {
    // If ID hasn't changed, just update the data property to maintain object reference
    nodes.value[idx] = { ...nodes.value[idx], data: { ...node } }
  } else {
    // If renaming, we must replace the object but we keep all other props
    const updatedNode = { 
      ...nodes.value[idx], 
      id: newId, 
      data: { ...node },
      selected: true
    }
    nodes.value[idx] = updatedNode

    // Update all connected edges
    edges.value = edges.value.map(e => {
      if (e.source === oldId || e.target === oldId) {
        return {
          ...e,
          id: `${e.source === oldId ? newId : e.source}__${e.label}__${e.target === oldId ? newId : e.target}`,
          source: e.source === oldId ? newId : e.source,
          target: e.target === oldId ? newId : e.target,
        }
      }
      return e
    })
  }

  upsertAgent(agentDef)

  // Use nextTick to ensure Vue Flow has processed the array change,
  // then update our own selection ref.
  nextTick(() => {
    selected.value = { type: 'node', id: newId, data: { ...node } }
    isDirty.value = true
  })
}

function updateEdge(data) {
  const currentSelected = selected.value
  if (!currentSelected || currentSelected.type !== 'edge') return

  const idx = edges.value.findIndex(e => e.id === currentSelected.id)
  if (idx === -1) return

  const oldId = edges.value[idx].id
  const newId = `${edges.value[idx].source}__${data.label}__${edges.value[idx].target}`

  edges.value[idx] = { 
    ...edges.value[idx], 
    id: newId,
    label: data.label, 
    data: { ...edges.value[idx].data, guard: data.guard },
    selected: true
  }

  nextTick(() => {
    selected.value = { 
      type: 'edge', 
      id: newId, 
      data: { label: data.label, guard: data.guard, from: edges.value[idx].source, to: edges.value[idx].target } 
    }
    isDirty.value = true
  })
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
      const prompt    = a.config?.prompt
      const mcp       = a.config?.mcp_servers
      const provider  = a.config?.provider
      if (prompt || mcp || provider) {
        obj.config = {}
        if (prompt)   obj.config.prompt      = prompt
        if (mcp)      obj.config.mcp_servers = mcp
        if (provider) obj.config.provider    = provider
      }
      return obj
    })

  return {
    apiVersion: 'asm/v1',
    kind:       'Workflow',
    metadata:   { 
      name: meta.name, 
      version: meta.version || 'draft', 
      description: meta.description,
      system_prompt: meta.systemPrompt
    },
    blackboard: { schema: { ...bbSchema } },
    states:     nodes.value.map(n => ({
      name:         n.id,
      type:         n.data.type         ?? 'intermediate',
      instructions: n.data.instructions || undefined,
      agent:        n.data.agent        || undefined,
      script:       n.data.script       || undefined,
      condition:    n.data.condition    || undefined,
      timeout:      n.data.timeout      || undefined,
      on_timeout:   n.data.on_timeout   || undefined,
      position:     n.position,
    })),
    transitions: Array.from(edges.value.reduce((map, e) => {
      const key = `${e.source}__${e.label}`
      if (!map.has(key)) {
        map.set(key, { 
          from: e.source, 
          trigger: e.label, 
          guard: e.data?.guard || undefined,
          to_nodes: [] 
        })
      }
      map.get(key).to_nodes.push(e.target)
      return map
    }, new Map()).values()).map(t => {
        if (t.to_nodes.length === 1) {
            return { from: t.from, to: t.to_nodes[0], trigger: t.trigger, guard: t.guard }
        }
        return { from: t.from, to_nodes: t.to_nodes, trigger: t.trigger, guard: t.guard }
    }),
    agents: cleanAgents,
  }
}

async function saveWorkflow() {
  if (!meta.name) {
    alert('Workflow name is required')
    return
  }

  // Validate agents for initial/intermediate states
  const unassigned = nodes.value.filter(n => 
    (n.data.type === 'initial' || n.data.type === 'intermediate') && !n.data.agent
  )
  if (unassigned.length > 0) {
    const names = unassigned.map(n => n.id).join(', ')
    if (!confirm(`Warning: The following reasoning states have no agent assigned: ${names}.\n\nThese states will wait for external triggers and will NOT be automated by an LLM. Continue?`)) {
      return
    }
  }

  saving.value = true
  try {
    const def = buildDefinition()
    const yamlStr = yaml.dump(def, { lineWidth: 120 })
    const result = await wfStore.save(yamlStr)
    // Server auto-increments version; update local state.
    const newDef = result.definition ?? result
    meta.version       = newDef.metadata?.version ?? meta.version
    meta.versionNumber = result.version_number ?? newDef.metadata?.version_number ?? meta.versionNumber
    isDirty.value = false
    // Refresh version list
    try { versions.value = await wfStore.fetchVersions(meta.name) } catch { /* ignore */ }
    router.replace(`/designer/${meta.name}/${meta.version}`)
    return result
  } catch (e) {
    alert(e.message)
    throw e
  } finally {
    saving.value = false
  }
}

async function onRunClick() {
  if (isDirty.value) {
    if (confirm('You have unsaved changes. Save them before running?')) {
      try {
        await saveWorkflow()
      } catch {
        return // save failed or cancelled
      }
    }
  }
  showRunModal.value = true
}

async function switchVersion(vnum) {
  if (isDirty.value && !confirm('You have unsaved changes. Discard and switch version?')) return
  try {
    const { definition, yaml: src } = await wfStore.fetchByVersion(meta.name, vnum)
    loadDefinition(definition, src)
    await nextTick()
    flowKey.value++
    router.replace(`/designer/${meta.name}/${meta.version}`)
  } catch (e) {
    alert(e.message)
  }
}

// ── AI Assistant integration ─────────────────────────────────────────────────

function onAIApply({ yamlStr, definition }) {
  try {
    const def = definition || yaml.load(yamlStr)
    if (!def) return
    loadDefinition(def, yamlStr)
    nextTick(() => { flowKey.value++ })
    isDirty.value = true
  } catch (e) {
    alert('Failed to apply AI-generated workflow: ' + e.message)
  }
}

// ── YAML import / split-view sync ────────────────────────────────────────────

function importYAML() {
  yamlError.value = ''
  try {
    const def = yaml.load(yamlSource.value)
    if (!def) throw new Error('YAML is empty or invalid.')
    loadDefinition(def, null, false)
    view.value = 'canvas'
    nextTick(() => { flowKey.value++ })
    isDirty.value = true
  } catch (e) {
    yamlError.value = e.message
    alert('Failed to import YAML: ' + e.message)
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

// Switching views: sync YAML <-> Canvas
watch(view, (v, oldV) => {
  if (v === 'yaml' || v === 'split') {
    // Canvas -> YAML
    yamlSource.value = yaml.dump(buildDefinition(), { lineWidth: 120 })
  } else if (oldV === 'yaml' && v === 'canvas') {
    // YAML -> Canvas (user might have edited the YAML)
    try {
      const def = yaml.load(yamlSource.value)
      if (def) loadDefinition(def, null, false)
      yamlError.value = ''
    } catch (e) {
      yamlError.value = e.message
      alert('YAML Syntax Error: ' + e.message + '\n\nReturning to editor.')
      view.value = 'yaml'
    }
  }
})
</script>

<!-- ── Inline sub-components ────────────────────────────────────────────────── -->
<script>
import { defineComponent, h, ref, reactive } from 'vue'
import { api } from '@/stores/auth'

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
    edges:    { type: Array, default: () => [] },
    agentDef: { type: Object, default: null },
  },
  emits: ['update', 'delete'],
  setup(p, { emit }) {
    const STATE_TYPES = ['initial', 'intermediate', 'hitl', 'terminal', 'script', 'wait']

    const form = reactive({
      name:         p.node.name         ?? '',
      type:         p.node.type         ?? 'intermediate',
      instructions: p.node.instructions ?? '',
      agent:        p.node.agent        ?? '',
      condition:    p.node.condition    ?? '',
      timeout:      p.node.timeout      ?? '',
      on_timeout:   p.node.on_timeout   ?? '',
      // Script state fields
      script_trigger: p.node.script?.trigger ?? '',
      script_updates: p.node.script?.updates
        ? Object.entries(p.node.script.updates).map(([k, v]) => ({ key: k, expr: v }))
        : [],
    })

    const agentForm = reactive({
      provider:    p.agentDef?.config?.provider   ?? '',
      model:       p.agentDef?.model              ?? '',
      task_queue:  p.agentDef?.task_queue         ?? '',
      prompt:      p.agentDef?.config?.prompt     ?? '',
      mcp_servers: p.agentDef?.config?.mcp_servers ?? '',
    })

    // Scaffold (script generation) state
    const scaffoldScript  = ref(null) // { trigger, updates: {} }
    const scaffoldLoading = ref(false)
    const scaffoldError   = ref('')

    async function generateScript() {
      scaffoldLoading.value = true
      scaffoldError.value   = ''
      scaffoldScript.value  = null
      try {
        const { data } = await api.post('/designer/scaffold', {
          agent_name:  form.agent,
          state_name:  form.name,
          description: form.instructions || '',
          triggers:    outgoingTransitions.value.map(t => t.label),
          bb_schema:   Object.fromEntries(Object.entries(p.bbSchema).map(([k, v]) => [k, v.type])),
        })
        scaffoldScript.value = data
      } catch (e) {
        scaffoldError.value = e.response?.data?.error ?? e.message
      } finally {
        scaffoldLoading.value = false
      }
    }

    function applyScript() {
      if (!scaffoldScript.value) return
      
      form.type = 'script'
      form.agent = ''
      form.script_trigger = scaffoldScript.value.trigger
      form.script_updates = Object.entries(scaffoldScript.value.updates || {}).map(([key, expr]) => ({ key, expr }))
      
      onApply()
      scaffoldScript.value = null
    }

    function copyScript() {
      if (!scaffoldScript.value) return
      const text = JSON.stringify(scaffoldScript.value, null, 2)
      navigator.clipboard.writeText(text)
    }

    const incomingTransitions = computed(() => 
      p.edges.filter(e => e.target === p.node.name).map(e => ({ from: e.source, label: e.label }))
    )
    const outgoingTransitions = computed(() => 
      p.edges.filter(e => e.source === p.node.name).map(e => ({ to: e.target, label: e.label }))
    )

    function onApply() {
      // Build script definition when type is 'script'
      let scriptDef = undefined
      if (form.type === 'script') {
        scriptDef = {
          trigger: form.script_trigger || '',
          updates: form.script_updates.length
            ? Object.fromEntries(form.script_updates.filter(r => r.key).map(r => [r.key, r.expr]))
            : undefined,
        }
      }

      const nodeData = {
        name:         form.name,
        type:         form.type,
        instructions: form.instructions || undefined,
        agent:        form.type !== 'script' && form.type !== 'wait' ? (form.agent || undefined) : undefined,
        script:       scriptDef,
        condition:    form.type === 'wait' ? (form.condition || undefined) : undefined,
        timeout:      form.timeout    || undefined,
        on_timeout:   form.on_timeout || undefined,
      }
      const agentData = (form.type !== 'script' && form.agent) ? {
        name:       form.agent,
        model:      agentForm.model       || undefined,
        task_queue: agentForm.task_queue  || undefined,
        config: (agentForm.prompt || agentForm.mcp_servers || agentForm.provider) ? {
          prompt:      agentForm.prompt      || undefined,
          mcp_servers: agentForm.mcp_servers || undefined,
          provider:    agentForm.provider    || undefined,
        } : undefined,
      } : null
      emit('update', { node: nodeData, agentDef: agentData })
    }

    return () => {
      // Transitions list (incoming/outgoing)
      const transitionSection = h('div', { class: 'pt-3 border-t border-border space-y-3' }, [
        h('h4', { class: 'text-xs font-semibold text-slate-500 uppercase tracking-wider' }, 'Transitions'),
        
        // Incoming
        h('div', { class: 'space-y-1.5' }, [
          h('div', { class: 'text-[10px] text-slate-500 font-medium' }, 'Incoming'),
          incomingTransitions.value.length ? h('div', { class: 'space-y-1' }, 
            incomingTransitions.value.map(t => h('div', { 
              key: `${t.from}__${t.label}`,
              class: 'flex items-center gap-1.5 px-2 py-1.5 rounded bg-surface-0 border border-border text-[10px]'
            }, [
              h('span', { class: 'text-slate-400 font-mono truncate max-w-[80px]' }, t.from),
              h('span', { class: 'text-slate-600' }, '→'),
              h('span', { class: 'text-indigo-400 font-mono font-medium truncate flex-1' }, t.label),
            ]))
          ) : h('p', { class: 'text-[10px] text-slate-600 italic' }, 'None'),
        ]),

        // Outgoing
        h('div', { class: 'space-y-1.5' }, [
          h('div', { class: 'text-[10px] text-slate-500 font-medium' }, 'Outgoing'),
          outgoingTransitions.value.length ? h('div', { class: 'space-y-1' }, 
            outgoingTransitions.value.map(t => h('div', { 
              key: `${t.label}__${t.to}`,
              class: 'flex items-center gap-1.5 px-2 py-1.5 rounded bg-surface-0 border border-border text-[10px]'
            }, [
              h('span', { class: 'text-indigo-400 font-mono font-medium truncate flex-1' }, t.label),
              h('span', { class: 'text-slate-600' }, '→'),
              h('span', { class: 'text-slate-400 font-mono truncate max-w-[80px]' }, t.to),
            ]))
          ) : h('p', { class: 'text-[10px] text-slate-600 italic' }, 'None'),
        ]),
      ])
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
          field('Provider',
            h('select', {
              class: 'input text-xs',
              value: agentForm.provider,
              onChange: e => agentForm.provider = e.target.value,
            }, [
              h('option', { value: '' }, 'Default (global setting)'),
              h('option', { value: 'anthropic' }, 'Anthropic'),
              h('option', { value: 'openai' }, 'OpenAI'),
              h('option', { value: 'grok' }, 'Grok (xAI)'),
              h('option', { value: 'deepseek' }, 'Deepseek'),
              h('option', { value: 'gemini' }, 'Gemini'),
              h('option', { value: 'ollama' }, 'Ollama (local)'),
            ])
          ),
          field('Model',
            h('input', {
              class: 'input text-xs',
              value: agentForm.model,
              placeholder: 'Leave blank to use provider default',
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
        ]),

        // Script generation section (for non-script types with an agent name set)
        ...(form.type !== 'script' && form.agent ? [
          h('div', { class: 'pt-3 border-t border-border space-y-2' }, [
            h('div', { class: 'flex items-center gap-2' }, [
              h('h4', { class: 'text-xs font-semibold text-slate-500 uppercase tracking-wider flex-1' }, 'Convert to Script'),
              h('button', {
                class: `text-xs px-2.5 py-1 rounded font-medium transition-colors ${
                  scaffoldLoading.value
                    ? 'bg-indigo-500/20 text-indigo-400 cursor-not-allowed'
                    : 'bg-indigo-600 hover:bg-indigo-500 text-white'
                }`,
                disabled: scaffoldLoading.value,
                onClick: generateScript,
              }, scaffoldLoading.value ? 'Generating…' : 'Generate Script'),
            ]),
            h('p', { class: 'text-[10px] text-slate-600 leading-relaxed' },
              'Convert this LLM agent into deterministic expr-lang logic for zero-latency execution.'
            ),

            // Error
            scaffoldError.value && h('p', { class: 'text-[10px] text-red-400 font-mono' }, scaffoldError.value),

            // Script Preview
            scaffoldScript.value && h('div', { class: 'space-y-2' }, [
              h('div', { class: 'bg-surface-0 border border-border rounded p-2 space-y-2' }, [
                h('div', { class: 'space-y-0.5' }, [
                  h('span', { class: 'text-[9px] text-slate-500 font-bold uppercase' }, 'Trigger'),
                  h('div', { class: 'text-[10px] font-mono text-indigo-400' }, scaffoldScript.value.trigger),
                ]),
                Object.keys(scaffoldScript.value.updates || {}).length > 0 && h('div', { class: 'space-y-0.5' }, [
                  h('span', { class: 'text-[9px] text-slate-500 font-bold uppercase' }, 'Updates'),
                  Object.entries(scaffoldScript.value.updates).map(([k, v]) => h('div', { class: 'text-[10px] font-mono text-slate-400 flex gap-2' }, [
                    h('span', { class: 'text-slate-500 shrink-0' }, k),
                    h('span', { class: 'text-slate-600 shrink-0' }, '='),
                    h('span', { class: 'truncate' }, v),
                  ])),
                ]),
              ]),
              h('div', { class: 'flex gap-2' }, [
                h('button', { class: 'btn-primary text-[10px] flex-1 py-1.5', onClick: applyScript }, 'Apply to State'),
                h('button', { class: 'btn-ghost text-[10px] px-2 py-1.5', onClick: copyScript }, 'Copy'),
              ]),
            ]),
          ]),
        ] : [])
      ] : [];

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

        // Wait condition (only for wait states)
        ...(form.type === 'wait' ? [
          field('Condition (expr-lang)',
            h('input', {
              class:       'input font-mono text-xs',
              value:       form.condition,
              placeholder: 'e.g. blackboard.count >= 2',
              onInput:     e => form.condition = e.target.value,
            })
          ),
        ] : []),

        // Agent name field (hidden for script/wait states)
        ...(form.type !== 'script' && form.type !== 'wait' ? [
          field('Agent',
            h('div', { class: 'space-y-1.5' }, [
              h('input', {
                class:       'input font-mono',
                value:       form.agent,
                placeholder: 'agent_name (required for automation)',
                onInput:     e => form.agent = e.target.value,
              }),
              (form.type === 'initial' || form.type === 'intermediate') && !form.agent && h('p', { class: 'text-[10px] text-amber-500 font-medium' }, 
                '⚠ Agent required for automated reasoning.'
              ),
            ])
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
        transitionSection,
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
          ['initial', 'intermediate', 'hitl', 'terminal', 'script', 'wait'].map(t => h('option', { value: t }, t))
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

// ── AIChatPanel ─────────────────────────────────────────────────────────────
// Chat-like AI assistant sidebar for generating and refining workflows.
export const AIChatPanel = defineComponent({
  props: {
    yamlSource:           { type: String, default: '' },
    workflowName:         { type: String, default: '' },
    workflowDescription:  { type: String, default: '' },
  },
  emits: ['apply'],
  setup(p, { emit }) {
    const messages = ref([])   // { role: 'user'|'assistant', content: string, yaml?: string, definition?: object }
    const input = ref('')
    const loading = ref(false)
    const error = ref('')

    async function send() {
      const text = input.value.trim()
      if (!text || loading.value) return

      messages.value.push({ role: 'user', content: text })
      input.value = ''
      loading.value = true
      error.value = ''

      try {
        const payload = { description: text }

        // Always send the current YAML so the LLM sees the latest state
        if (p.yamlSource && p.yamlSource.trim()) {
          payload.existing_yaml = p.yamlSource
        }

        // Send the workflow's verbal description for additional context
        if (p.workflowDescription) {
          payload.workflow_description = p.workflowDescription
        }

        // Send conversation history (exclude the message we just pushed —
        // it becomes the Description on the server side). For assistant
        // messages, include the YAML they generated so the LLM sees the
        // correct output pattern (system prompt says "respond with ONLY YAML").
        const history = messages.value.slice(0, -1).map(m => ({
          role: m.role,
          content: m.role === 'assistant' && m.yaml
            ? '```yaml\n' + m.yaml + '\n```'
            : m.content,
        }))
        if (history.length) {
          payload.history = history
        }

        const { data } = await api.post('/designer/generate', payload)

        if (data.error) {
          error.value = data.error
          messages.value.push({ role: 'assistant', content: 'Error: ' + data.error })
        } else {
          messages.value.push({
            role: 'assistant',
            content: p.yamlSource ? 'I\'ve updated the workflow based on your request.' : 'I\'ve generated a workflow based on your description.',
            yaml: data.yaml,
            definition: data.definition,
          })
        }
      } catch (e) {
        const msg = e.response?.data?.error ?? e.message
        error.value = msg
        messages.value.push({ role: 'assistant', content: 'Error: ' + msg })
      } finally {
        loading.value = false
      }
    }

    function applyResult(msg) {
      emit('apply', { yamlStr: msg.yaml, definition: msg.definition })
    }

    // Auto-scroll to bottom
    function scrollContainer(el) {
      if (el) el.scrollTop = el.scrollHeight
    }

    return () => h('div', { class: 'flex flex-col flex-1 min-h-0' }, [

      // Messages area
      h('div', {
        class: 'flex-1 overflow-y-auto p-3 space-y-3',
        ref: el => { if (el) setTimeout(() => scrollContainer(el), 50) },
      }, [
        // Welcome message when empty
        !messages.value.length && h('div', { class: 'text-center py-8 space-y-3' }, [
          h('svg', { width: '32', height: '32', viewBox: '0 0 24 24', fill: 'none', stroke: '#8b5cf6', 'stroke-width': '1.5', class: 'mx-auto' }, [
            h('path', { d: 'M12 2l2.4 7.4H22l-6.2 4.5 2.4 7.4L12 17l-6.2 4.3 2.4-7.4L2 9.4h7.6z' }),
          ]),
          h('p', { class: 'text-sm text-slate-400' }, 'AI Workflow Assistant'),
          h('p', { class: 'text-xs text-slate-600 leading-relaxed max-w-[220px] mx-auto' },
            p.yamlSource
              ? 'Describe changes to make to this workflow. The AI will modify the existing definition.'
              : 'Describe a business process and the AI will generate a workflow for you.'
          ),
        ]),

        // Message list
        ...messages.value.map((msg, i) =>
          h('div', { key: i, class: 'space-y-1.5' }, [
            // Role label
            h('div', { class: 'flex items-center gap-1.5' }, [
              msg.role === 'user'
                ? h('span', { class: 'text-[10px] font-medium text-slate-500 uppercase' }, 'You')
                : h('span', { class: 'text-[10px] font-medium text-violet-500 uppercase' }, 'AI'),
            ]),
            // Content
            h('div', {
              class: msg.role === 'user'
                ? 'text-xs text-slate-300 bg-surface-0 rounded-lg px-3 py-2 border border-border'
                : 'text-xs text-slate-400 bg-violet-500/5 rounded-lg px-3 py-2 border border-violet-500/20',
            }, msg.content),
            // Apply button for assistant messages with YAML
            msg.yaml && h('button', {
              class: 'btn-primary text-xs gap-1.5 w-full',
              onClick: () => applyResult(msg),
            }, [
              h('svg', { width: '12', height: '12', viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2.5' }, [
                h('polyline', { points: '20 6 9 17 4 12' }),
              ]),
              'Apply to Workflow',
            ]),
          ])
        ),

        // Loading indicator
        loading.value && h('div', { class: 'flex items-center gap-2 text-xs text-violet-400 animate-pulse py-2' }, [
          h('svg', { width: '14', height: '14', viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2', class: 'animate-spin' }, [
            h('circle', { cx: '12', cy: '12', r: '10', opacity: '0.25' }),
            h('path', { d: 'M12 2a10 10 0 0 1 10 10', opacity: '0.75' }),
          ]),
          'Thinking...',
        ]),
      ]),

      // Input area
      h('div', { class: 'shrink-0 border-t border-border p-3' }, [
        h('div', { class: 'flex gap-2' }, [
          h('textarea', {
            class: 'input flex-1 text-xs resize-none min-h-[36px] max-h-[100px]',
            value: input.value,
            placeholder: p.yamlSource ? 'Describe a change...' : 'Describe a workflow...',
            rows: 1,
            onInput: e => {
              input.value = e.target.value
              // Auto-resize
              e.target.style.height = 'auto'
              e.target.style.height = Math.min(e.target.scrollHeight, 100) + 'px'
            },
            onKeydown: e => {
              if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send() }
            },
            disabled: loading.value,
          }),
          h('button', {
            class: 'btn-primary p-2 shrink-0 self-end',
            onClick: send,
            disabled: !input.value.trim() || loading.value,
            title: 'Send (Enter)',
          }, [
            h('svg', { width: '14', height: '14', viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2.5' }, [
              h('line', { x1: '22', y1: '2', x2: '11', y2: '13' }),
              h('polygon', { points: '22 2 15 22 11 13 2 9 22 2' }),
            ]),
          ]),
        ]),
        h('p', { class: 'text-[10px] text-slate-600 mt-1.5' }, 'Press Enter to send, Shift+Enter for new line'),
      ]),
    ])
  },
})
</script>
