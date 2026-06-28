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
      <span class="text-sm font-medium text-text">{{ workflowName || 'New Workflow' }}</span>
      <!-- Version selector -->
      <template v-if="versions.length > 1">
        <select
          :value="meta.versionNumber"
          @change="switchVersion(Number($event.target.value))"
          class="bg-surface-0 border border-border rounded px-2 py-1 text-xs text-text-muted font-mono cursor-pointer hover:border-slate-500 outline-none"
        >
          <option v-for="v in versions" :key="v.version_number" :value="v.version_number">
            v{{ v.version_number }}
          </option>
        </select>
      </template>
      <span v-else-if="meta.versionNumber" class="text-xs text-text-muted font-mono">v{{ meta.versionNumber }}</span>
      <div v-if="isDirty" class="w-1.5 h-1.5 rounded-full bg-amber-500" title="Unsaved changes"/>

      <div class="flex-1"/>

      <!-- Auto-layout -->
      <button
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

      <!-- Timeout edge toggle -->
      <button
        @click="showTimeoutEdges = !showTimeoutEdges"
        :class="['btn-ghost text-xs gap-1', !showTimeoutEdges && 'opacity-40']"
        :title="showTimeoutEdges ? 'Hide timeout edges' : 'Show timeout edges'"
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="9"/>
          <polyline points="12 7 12 12 15 15"/>
        </svg>
        Timeouts
      </button>

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

      <!-- Left Node Palette Sidebar -->
      <aside class="w-56 border-r border-border flex flex-col bg-surface-1 shrink-0 select-none overflow-y-auto">
        <div class="px-4 py-3 border-b border-border flex items-center justify-between bg-surface-1 shrink-0">
          <span class="text-xs font-semibold text-text-muted uppercase tracking-wider">Node Palette</span>
        </div>
        <div class="p-3 space-y-4">
          <!-- reasoning & logic -->
          <div class="space-y-1.5">
            <div class="text-[10px] text-text-muted font-bold uppercase tracking-wider px-1">AI & Logic</div>
            <div class="grid grid-cols-1 gap-1.5">
              <div
                v-for="node in paletteLogic"
                :key="node.type"
                draggable="true"
                @dragstart="onDragStart($event, node)"
                class="flex items-center gap-2.5 p-2 rounded-lg border border-border bg-surface-0 transition-colors group text-xs text-text hover:bg-surface-2 cursor-grab active:cursor-grabbing"
              >
                <div :class="['w-2.5 h-2.5 rounded-full shrink-0', node.dot]"/>
                <span class="truncate flex-1">{{ node.label }}</span>
              </div>
            </div>
          </div>
          <!-- triggers (low code) -->
          <div class="space-y-1.5">
            <div class="text-[10px] text-text-muted font-bold uppercase tracking-wider px-1">Triggers (Low-Code)</div>
            <div class="grid grid-cols-1 gap-1.5">
              <div
                v-for="node in paletteTriggers"
                :key="node.triggerType"
                draggable="true"
                @dragstart="onDragStart($event, node)"
                class="flex items-center gap-2.5 p-2 rounded-lg border border-border bg-surface-0 hover:bg-surface-2 cursor-grab active:cursor-grabbing text-xs text-text transition-colors group"
              >
                <div :class="['w-2.5 h-2.5 rounded-full shrink-0 bg-emerald-400']"/>
                <span class="truncate flex-1">{{ node.label }}</span>
              </div>
            </div>
          </div>
        </div>
      </aside>

      <!-- Center Column: Visual Canvas + Generate Panel (Split view style) -->
      <div class="flex-1 flex flex-col min-w-0 overflow-hidden bg-surface-0 ai-designer-center-col">
        <!-- Top Canvas (Dynamic height) -->
        <div 
          class="relative overflow-hidden border-b border-border shrink-0"
          :style="{ height: canvasHeightPercent + '%' }"
          @dragover.prevent
          @drop="onDrop"
        >
          <VueFlow
            :key="flowKey"
            v-model:nodes="nodes"
            v-model:edges="edges"
            :node-types="nodeTypes"
            :edge-types="edgeTypes"
            :default-viewport="{ zoom: 1 }"
            fit-view-on-init
            :class="['bg-surface-0 ai-designer-mode', !showTimeoutEdges && 'hide-timeout-edges']"
            @node-click="onNodeClick"
            @edge-click="onEdgeClick"
            @pane-click="onPaneClick"
            @connect="onConnect"
            @nodes-change="onNodesChange"
            @edges-change="changes => { if (changes.some(c => c.type !== 'select')) isDirty = true }"
          >
            <Background pattern-color="#2a2d3a" :gap="24" :size="1" />
            <Controls />
            <MiniMap node-color="#1a1d27" mask-color="rgba(12,14,20,0.7)" />
          </VueFlow>
        </div>

        <!-- Resizer Divider -->
        <div 
          class="h-1.5 shrink-0 bg-border hover:bg-violet-500/80 active:bg-violet-500 cursor-row-resize transition-colors select-none z-10"
          @mousedown="startResize"
        />

        <!-- Bottom Generate Panel (Takes remaining space) -->
        <div class="flex-1 flex flex-col min-h-0 min-w-0">
          <AIGeneratePanel
            :yaml-source="yamlSource"
            v-model:description="meta.processDescription"
            v-model:prompt="generatePrompt"
            v-model:history="generateHistory"
            v-model:interactions="generateInteractions"
            v-model:provider="generateProvider"
            v-model:pipeline-step="generatePipelineStep"
            v-model:pipeline-warning="generatePipelineWarning"
            :metadata-abstract="meta.description"
            @apply="onAIApply"
            @request-description-modal="showDescriptionModal = true"
          />
        </div>
      </div>

      <!-- Resizer Divider for Right Sidebar -->
      <div 
        class="w-1.5 shrink-0 bg-border hover:bg-violet-500/80 active:bg-violet-500 cursor-col-resize transition-colors select-none z-10"
        @mousedown="startSidebarResize"
      />

      <!-- Right Sidebar: Properties & Config Panel -->
      <aside :style="{ width: sidebarWidth + 'px' }" class="border-l border-border flex flex-col overflow-hidden bg-surface-1 shrink-0">
        <div class="flex-1 flex flex-col overflow-y-auto custom-scrollbar">
          
          <!-- Pending connection -->
          <PendingConnectionPanel
            v-if="pendingConnection"
            :connection="pendingConnection"
            :options="availableTriggers"
            @confirm="confirmConnection"
            @cancel="cancelConnection"
          />

          <!-- Workflow metadata (no selection) -->
          <div v-else-if="!selected" class="p-4 space-y-4 overflow-y-auto flex-1">
            <div class="space-y-3">
              <h3 class="text-xs font-semibold text-text-muted uppercase tracking-wider">Workflow</h3>
              <div class="space-y-2">
                <input v-model="meta.name" class="input" placeholder="Name" @input="isDirty=true"/>
                <div v-if="meta.versionNumber" class="flex items-center gap-2 px-3 py-2 rounded-lg bg-surface-0 border border-border">
                  <span class="text-xs text-text-muted">Version</span>
                  <span class="text-xs font-mono text-text">v{{ meta.versionNumber }}</span>
                  <span class="text-[10px] text-text-muted flex-1 text-right">auto-incremented on save</span>
                </div>
                <textarea v-model="meta.description" class="input h-16 resize-none" placeholder="Short Abstract (e.g. Invoice Processing)" @input="isDirty=true"/>
                <div class="space-y-2">
                  <label class="text-[10px] text-text-muted font-semibold uppercase tracking-wider">Cron Schedule</label>
                  <CronEditor v-model="meta.schedule" @change="isDirty=true" />
                </div>
                <div class="space-y-2">
                  <div class="flex items-center justify-between px-1">
                    <label class="text-[10px] text-text-muted font-semibold uppercase tracking-wider">Modular Workflow</label>
                  </div>
                  <label class="flex items-center gap-2 p-3 rounded-lg bg-indigo-500/5 border border-indigo-500/20 cursor-pointer hover:bg-indigo-500/10 transition-colors group">
                    <input type="checkbox" v-model="meta.reusable" class="accent-indigo-500" @change="isDirty=true" />
                    <div class="flex-1">
                      <div class="text-xs font-semibold text-indigo-300 group-hover:text-indigo-200 transition-colors">Mark as Reusable</div>
                      <div class="text-[10px] text-text-muted leading-tight">Enables this workflow to be called as a sub-process by other workflows.</div>
                    </div>
                  </label>
                </div>
              </div>
            </div>

            <div class="space-y-4">
              <BlackboardSchemaEditor
                :schema="bbSchema"
                @change="isDirty = true"
                @open-popout="onOpenSchemaPopout"
              />
            </div>

            <!-- Agents summary -->
            <div v-if="agents.length" class="space-y-2">
              <h3 class="text-xs font-semibold text-text-muted uppercase tracking-wider">Agents ({{ agents.length }})</h3>
              <div v-for="a in agents" :key="a.name" class="text-xs text-text-muted flex items-center gap-2 py-1">
                <span class="font-mono text-text-muted truncate flex-1">{{ a.name }}</span>
                <span v-if="a.model" class="text-[10px] text-text-muted truncate">{{ a.model.split('/').pop() }}</span>
              </div>
            </div>

            <!-- Workflow Prompt -->
            <div class="space-y-1 pt-2 border-t border-border/50">
              <label class="text-[10px] text-text-muted font-semibold uppercase tracking-wider">Workflow Prompt (Global Personality)</label>
              <textarea v-model="meta.systemPrompt" class="input h-32 font-mono text-sm leading-relaxed resize-none" placeholder="You are a helpful assistant…" @input="isDirty=true"/>
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
            :agents="agents"
            @update="updateNode"
            @delete="deleteSelected"
          />

          <!-- Edge selected -->
          <EdgePanel
            v-else-if="selected?.type === 'edge'"
            :key="'edge-' + selected.id"
            :edge="selected.data"
            :options="availableTriggers"
            @update="updateEdge"
            @delete="deleteSelected"
          />
        </div>
      </aside>
    </div>

    <!-- ── Modals ──────────────────────────────────────────────────────── -->
    <AddStateModal v-if="showAddState" @close="showAddState = false" @add="onAddState" />
    
    <!-- Description Pop-out Modal -->
    <CodeEditorModal
      v-if="showDescriptionModal"
      v-model="meta.processDescription"
      title="Process Description"
      language="text"
      @close="showDescriptionModal = false"
    />

    <!-- Start Run Modal -->
    <StartRunModal
      v-if="showRunModal"
      :definition="buildDefinition()"
      @close="showRunModal = false"
      @started="(run) => router.push(`/monitor/${run.id}`)"
    />

    <!-- Schema Pop-out Modal -->
    <SchemaEditorModal
      v-if="showSchemaModal"
      :name="activeSchemaName"
      :schema="activeSchemaRef"
      @close="showSchemaModal = false"
      @change="isDirty = true"
      @open-popout="onOpenSchemaPopout"
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch, nextTick, shallowRef, markRaw } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { Background }  from '@vue-flow/background'
import { Controls }    from '@vue-flow/controls'
import { MiniMap }     from '@vue-flow/minimap'

import '@vue-flow/core/dist/style.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'

import StateNode      from '@/components/designer/StateNode.vue'
import TransitionEdge from '@/components/designer/TransitionEdge.vue'
import StartRunModal  from '@/components/shared/StartRunModal.vue'

import PendingConnectionPanel from '@/components/designer/PendingConnectionPanel.vue'
import CronEditor from '@/components/designer/CronEditor.vue'
import StatePanel             from '@/components/designer/StatePanel.vue'
import EdgePanel              from '@/components/designer/EdgePanel.vue'
import BlackboardSchemaEditor from '@/components/designer/BlackboardSchemaEditor.vue'
import AddStateModal          from '@/components/designer/AddStateModal.vue'
import AIGeneratePanel          from '@/components/designer/AIGeneratePanel.vue'
import CodeEditorModal       from '@/components/designer/CodeEditorModal.vue'
import SchemaEditorModal    from '@/components/designer/SchemaEditorModal.vue'

import { useWorkflowStore } from '@/stores/workflows'
import { useEnterpriseStore } from '@/stores/enterprise'
import { useWorkflowIO }    from '@/composables/useWorkflowIO'
import { useCanvasSync }    from '@/composables/useCanvasSync'
import { useCanvasActions } from '@/composables/useCanvasActions'
import { useLayout }        from '@/composables/useLayout'

const route   = useRoute()
const router  = useRouter()
const wfStore = useWorkflowStore()
const entStore = useEnterpriseStore()
const { fitView, project } = useVueFlow()
const { layout }  = useLayout()

// ── Proxy router to keep A/B testing on /ai-designer routes ──
const customRouter = {
  ...router,
  replace(path) {
    if (path.startsWith('/designer/')) {
      const parts = path.split('/')
      router.replace(`/ai-designer/${parts[2]}/${parts[3]}`)
    } else {
      router.replace(path)
    }
  },
  push(path) {
    router.push(path)
  }
}

// ── Shared reactive state ────────────────────────────────────────────────────
const nodes   = ref([])
const edges   = ref([])
const agents  = ref([])
const meta    = reactive({ 
  name: '', version: '1.0.0', versionNumber: 0, description: '', systemPrompt: '', processDescription: '', schedule: '',
  reusable: false, inputs: [], outputs: [], capabilities: []
})
const bbSchema = reactive({})
const versions = ref([])

const view              = ref('canvas') // mock for useWorkflowIO
const rightTab          = ref('properties')
const yamlSource        = ref('')
const yamlErrors        = ref([])
const saving            = ref(false)
const isDirty           = ref(false)
const showRunModal      = ref(false)
const selected          = ref(null)
const pendingConnection = ref(null)
const showAddState         = ref(false)
const showDescriptionModal  = ref(false)
const flowKey              = ref(0)
const showTimeoutEdges  = ref(true)
const showSchemaModal   = ref(false)
const activeSchemaRef   = ref(null)
const activeSchemaName  = ref('')

function onOpenSchemaPopout({ name, field }) {
  activeSchemaName.value = name
  if (field.type === 'object') {
    if (!field.properties) field.properties = {}
    activeSchemaRef.value = field.properties
  } else if (field.type === 'list') {
    if (!field.items) field.items = { type: 'string' }
    if (field.items.type === 'object') {
      if (!field.items.properties) field.items.properties = {}
      activeSchemaRef.value = field.items.properties
    }
  }
  showSchemaModal.value = true
}

// ── Canvas Resizing State ──
const canvasHeightPercent = ref(60)
let isDragging = false
let startY = 0
let startHeightPercent = 0

function startResize(e) {
  isDragging = true
  startY = e.clientY
  startHeightPercent = canvasHeightPercent.value
  document.addEventListener('mousemove', doResize)
  document.addEventListener('mouseup', stopResize)
  document.body.style.cursor = 'row-resize'
  document.body.style.userSelect = 'none'
}

function doResize(e) {
  if (!isDragging) return
  const container = document.querySelector('.ai-designer-center-col')
  if (!container) return
  const containerHeight = container.getBoundingClientRect().height
  if (containerHeight === 0) return
  
  const deltaY = e.clientY - startY
  const deltaPercent = (deltaY / containerHeight) * 100
  let newPercent = startHeightPercent + deltaPercent
  
  // Clamp between 25% and 80%
  if (newPercent < 25) newPercent = 25
  if (newPercent > 80) newPercent = 80
  
  canvasHeightPercent.value = newPercent
}

function stopResize() {
  isDragging = false
  document.removeEventListener('mousemove', doResize)
  document.removeEventListener('mouseup', stopResize)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}

// ── Generation Tab State ──
const generatePrompt       = ref('')
const generateHistory      = ref([])
const generateInteractions = ref([])
const generateProvider     = ref('')
const generatePipelineStep    = ref(0)
const generatePipelineWarning = ref('')

const nodeTypes = shallowRef({ stateNode: markRaw(StateNode) })
const edgeTypes = shallowRef({ transitionEdge: markRaw(TransitionEdge) })

const paletteLogic = [
  { type: 'initial', label: 'Initial (Start)', dot: 'bg-indigo-400' },
  { type: 'prompt', label: 'Prompt (Agent)', dot: 'bg-indigo-500' },
  { type: 'script', label: 'Script (Expr)', dot: 'bg-amber-400' },
  { type: 'code', label: 'Code (JS)', dot: 'bg-teal-400' },
  { type: 'hitl', label: 'HITL (Human)', dot: 'bg-amber-500' },
  { type: 'wait', label: 'Wait Node', dot: 'bg-slate-500' },
  { type: 'subprocess', label: 'Subprocess', dot: 'bg-violet-400' },
  { type: 'terminal', label: 'Terminal (End)', dot: 'bg-indigo-400' },
]

const paletteTriggers = [
  { isTrigger: true, triggerType: 'telegram', label: 'Telegram Trigger' },
  { isTrigger: true, triggerType: 'discord', label: 'Discord Trigger' },
  { isTrigger: true, triggerType: 'cron', label: 'Cron Trigger' },
  { isTrigger: true, triggerType: 'webhook', label: 'Webhook Trigger' },
]

const paletteActions = []

function onDragStart(event, node) {
  event.dataTransfer.setData('application/vueflow', JSON.stringify(node))
  event.dataTransfer.effectAllowed = 'move'
}

function onDrop(event) {
  event.preventDefault()

  const nodeStr = event.dataTransfer.getData('application/vueflow')
  if (!nodeStr) return

  const paletteNode = JSON.parse(nodeStr)
  const container = event.currentTarget.getBoundingClientRect()
  const position = project({
    x: event.clientX - container.left,
    y: event.clientY - container.top,
  })

  let name = ''
  let type = paletteNode.type || 'prompt'



  if (paletteNode.isTrigger) {
    type = 'initial'
    name = `trigger_${paletteNode.triggerType}`
  } else if (type === 'timeout_node') {
    name = 'timeout_node'
  } else {
    name = `${type}_${Math.random().toString(36).substring(2, 7)}`
  }

  let suffix = 1
  let finalName = name
  while (nodes.value.some(n => n.id === finalName)) {
    finalName = `${name}_${suffix++}`
  }

  const newNode = {
    id: finalName,
    type: 'stateNode',
    position,
    data: {
      name: finalName,
      type: type,
      instructions: '',
    }
  }

  if (paletteNode.isTrigger) {
    newNode.data.type = 'initial'
    newNode.data.triggerType = paletteNode.triggerType
    newNode.data.triggerConfig = {}
  } else if (type === 'telegram_output') {
    newNode.data.type = 'telegram_output'
    newNode.data.telegram_output = {
      chat_id: '{{bb.chat_id}}',
      message_text: 'Hello from Chain Nodes!',
    }
  } else if (type === 'discord_output') {
    newNode.data.type = 'discord_output'
    newNode.data.discord_output = {
      channel_id: '{{bb.channel_id}}',
      message_text: 'Hello from Chain Nodes!',
    }
  } else if (type === 'timeout_node') {
    newNode.data.type = 'timeout_node'
    newNode.data.is_timeout_node = true
    newNode.data.default_timeout = '10s'
  }

  const timeoutNode = nodes.value.find(n => n.data.type === 'timeout_node')
  if (timeoutNode && type !== 'initial' && type !== 'terminal' && type !== 'timeout_node') {
    newNode.data.timeout = timeoutNode.data.default_timeout || '10s'
    newNode.data.on_timeout = 'timeout'
    
    edges.value.push({
      id: `${finalName}__timeout__${timeoutNode.id}`,
      source: finalName,
      target: timeoutNode.id,
      sourceHandle: 'timeout',
      type: 'transitionEdge',
      label: 'timeout',
      class: 'timeout-edge',
      data: { guard: '', isTimeout: true }
    })
  }

  nodes.value.push(newNode)
  selected.value = { type: 'node', id: newNode.id, data: newNode.data }
  isDirty.value = true
}

const availableTriggers = computed(() => {
  let sourceId = null
  if (pendingConnection.value) {
    sourceId = pendingConnection.value.source
  } else if (selected.value?.type === 'edge') {
    sourceId = selected.value.data.from
  }
  if (!sourceId) return []

  const triggers = new Set()
  edges.value.forEach(e => {
    if (e.source === sourceId && e.label) triggers.add(e.label)
  })
  const node = nodes.value.find(n => n.id === sourceId)
  if (node?.data) {
    const data = node.data
    if (data.type === 'subprocess' && data.subprocess) {
      if (data.subprocess.completion_trigger) triggers.add(data.subprocess.completion_trigger)
      if (data.subprocess.failure_trigger) triggers.add(data.subprocess.failure_trigger)
    }
    if (data.on_timeout) triggers.add(data.on_timeout)
    if (data.type === 'wait' || data.timeout) triggers.add('timeout')
  }
  return Array.from(triggers).sort()
})

const workflowName = computed(() => meta.name || '')

// ── Composables ──
const {
  fromLoad, loadDefinition, buildDefinition,
  saveWorkflow, switchVersion, onRunClick, onAIApply: _onAIApply
} = useWorkflowIO({
  nodes, edges, meta, bbSchema, agents, versions, isDirty,
  yamlSource, yamlErrors, saving, flowKey, showRunModal, view,
  wfStore, router: customRouter
})

const { onYamlInput } = useCanvasSync({
  nodes, edges, meta, isDirty, view, yamlSource, yamlErrors,
  flowKey, fromLoad, loadDefinition, buildDefinition
})

const {
  applyLayout,
  onNodeClick, onEdgeClick, onPaneClick, onNodesChange,
  onConnect, confirmConnection, cancelConnection,
  getAgent,
  updateNode, updateEdge, deleteSelected,
  addState, onAddState
} = useCanvasActions({
  nodes, edges, agents, isDirty, selected, pendingConnection,
  showAddState, bbSchema, fitView, layout
})

async function onAIApply(payload) {
  _onAIApply(payload)
  await nextTick()
  await nextTick()
  applyLayout(false)
  if (meta.name) {
    try { await saveWorkflow(true) } catch { /* best-effort */ }
  }
}

const sidebarWidth = ref(320)
let isDraggingSidebar = false
let startSidebarX = 0
let startSidebarWidth = 0

function startSidebarResize(e) {
  isDraggingSidebar = true
  startSidebarX = e.clientX
  startSidebarWidth = sidebarWidth.value
  document.addEventListener('mousemove', doSidebarResize)
  document.addEventListener('mouseup', stopSidebarResize)
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}

function doSidebarResize(e) {
  if (!isDraggingSidebar) return
  const deltaX = startSidebarX - e.clientX
  let newWidth = startSidebarWidth + deltaX
  if (newWidth < 280) newWidth = 280
  if (newWidth > 600) newWidth = 600
  sidebarWidth.value = newWidth
}

function stopSidebarResize() {
  isDraggingSidebar = false
  document.removeEventListener('mousemove', doSidebarResize)
  document.removeEventListener('mouseup', stopSidebarResize)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}

// Keyboard shortcuts support
function handleKeydown(e) {
  const isTyping = ['INPUT', 'TEXTAREA'].includes(e.target.tagName) || e.target.isContentEditable
  if ((e.metaKey || e.ctrlKey) && e.key === 's') {
    e.preventDefault()
    saveWorkflow()
  }
}

onMounted(async () => {
  window.addEventListener('keydown', handleKeydown)
  sessionStorage.setItem('chainnodes_preferred_designer', 'ai')

  // Check for AI-proposed fix from the Monitor
  const fixData = sessionStorage.getItem('chainnodes_ai_fix')
  if (fixData) {
    try {
      const { yaml: fixYaml, explanation } = JSON.parse(fixData)
      await onAIApply({ yamlStr: fixYaml })
      sessionStorage.removeItem('chainnodes_ai_fix')
      // Show success toast or notification would be nice here
      console.log('AI fix applied:', explanation)
      return
    } catch (e) {
      console.error('Failed to apply stored AI fix:', e)
    }
  }

  const { name, version } = route.params
  if (name && version && name !== 'new') {
    try {
      const { definition, yaml: src } = await wfStore.fetchOne(name, version)
      loadDefinition(definition, src)
      try { versions.value = await wfStore.fetchVersions(name) } catch { /* ignore */ }
      await nextTick()
      flowKey.value++
      if (nodes.value.length > 1) {
        await nextTick()
        applyLayout(false)
      }
    } catch (e) {
      console.error('Failed to load workflow:', e)
    }
  } else {
    // Initial blank project nodes
    nodes.value = [
      {
        id: 'start',
        type: 'stateNode',
        position: { x: 250, y: 150 },
        data: { name: 'start', type: 'initial', instructions: '' }
      },
      {
        id: 'end',
        type: 'stateNode',
        position: { x: 250, y: 400 },
        data: { name: 'end', type: 'terminal', instructions: '' }
      }
    ]
    edges.value = []
    isDirty.value = false
    await nextTick()
    flowKey.value++
  }
})
</script>

<style scoped>
.custom-scrollbar::-webkit-scrollbar { width: 6px; }
.custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
.custom-scrollbar::-webkit-scrollbar-thumb { background: var(--color-border); border-radius: 3px; }
</style>
