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

      <!-- Timeout edge toggle (canvas/split only) -->
      <button
        v-if="view !== 'yaml'"
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

      <!-- View toggle -->
      <div class="flex bg-surface-0 rounded-lg p-0.5 border border-border">
        <button
          @click="view = 'canvas'"
          :class="['btn py-1 px-2.5 text-xs', view === 'canvas' ? 'bg-surface-2 text-text' : 'text-text-muted']"
        >Canvas</button>
        <button
          @click="view = 'split'"
          :class="['btn py-1 px-2.5 text-xs', view === 'split' ? 'bg-surface-2 text-text' : 'text-text-muted']"
        >Split</button>
        <button
          @click="view = 'yaml'"
          :class="['btn py-1 px-2.5 text-xs', view === 'yaml' ? 'bg-surface-2 text-text' : 'text-text-muted']"
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
            class="flex-1 bg-surface-0 text-text font-mono text-xs p-4 resize-none outline-none border-none"
            spellcheck="false"
            placeholder="Paste or type your workflow YAML here…"
          />
          <div v-if="yamlErrors.length" class="px-4 py-2 bg-red-500/10 border-t border-red-500/30 text-red-400 text-xs font-mono space-y-1">
            <div v-for="(err, i) in yamlErrors" :key="i" class="flex gap-2">
              <span class="opacity-50 font-bold shrink-0">[{{ err.line || '!' }}]</span>
              <span>{{ err.message }}</span>
            </div>
          </div>
          <div class="px-4 py-2 border-t border-border flex gap-3">
            <button @click="handleImportYAML" class="btn-primary text-xs">Import to Canvas</button>
          </div>
        </div>
      </template>

      <!-- Canvas or Split view -->
      <template v-else>

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
                  :draggable="!(node.type === 'subprocess' && !entStore.hasFeature('subprocesses'))"
                  @dragstart="node.type === 'subprocess' && !entStore.hasFeature('subprocesses') ? $event.preventDefault() : onDragStart($event, node)"
                  class="flex items-center gap-2.5 p-2 rounded-lg border border-border bg-surface-0 transition-colors group text-xs text-text"
                  :class="[
                    (node.type === 'subprocess' && !entStore.hasFeature('subprocesses'))
                      ? 'opacity-40 grayscale cursor-not-allowed select-none'
                      : 'hover:bg-surface-2 cursor-grab active:cursor-grabbing'
                  ]"
                >
                  <div :class="['w-2 h-2 rounded-full shrink-0', node.dot]"/>
                  <span class="truncate flex-1">{{ node.label }}</span>
                  <span v-if="node.type === 'subprocess' && !entStore.hasFeature('subprocesses')" class="text-[9px] text-amber-500 font-bold uppercase shrink-0">Pro</span>
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
                  <div :class="['w-2 h-2 rounded-full shrink-0 bg-emerald-400']"/>
                  <span class="truncate flex-1">{{ node.label }}</span>
                </div>
              </div>
            </div>
            <!-- actions (low code) -->
            <div class="space-y-1.5">
              <div class="text-[10px] text-text-muted font-bold uppercase tracking-wider px-1">Actions (Low-Code)</div>
              <div class="grid grid-cols-1 gap-1.5">
                <div
                  v-for="node in paletteActions"
                  :key="node.type"
                  draggable="true"
                  @dragstart="onDragStart($event, node)"
                  class="flex items-center gap-2.5 p-2 rounded-lg border border-border bg-surface-0 hover:bg-surface-2 cursor-grab active:cursor-grabbing text-xs text-text transition-colors group"
                >
                  <div :class="['w-2 h-2 rounded-full shrink-0 bg-blue-400']"/>
                  <span class="truncate flex-1">{{ node.label }}</span>
                </div>
              </div>
            </div>
          </div>
        </aside>

        <!-- Canvas pane -->
        <div 
          :class="['relative overflow-hidden', view === 'split' ? 'flex-[3]' : 'flex-1']"
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
            :class="['bg-surface-0', !showTimeoutEdges && 'hide-timeout-edges']"
            @node-click="onNodeClick"
            @edge-click="onEdgeClick"
            @pane-click="onPaneClick"
            @connect="onConnect"
            @nodes-change="onNodesChange"
            @edges-change="changes => { if (changes.some(c => c.type !== 'select')) isDirty = true }"
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
            <span class="text-xs text-text-muted font-medium flex-1">YAML — live mirror</span>
            <span v-if="yamlErrors.length" class="text-[10px] text-red-400 truncate max-w-[160px]" :title="yamlErrors[0].message">
              ⚠ {{ yamlErrors.length }} error{{ yamlErrors.length > 1 ? 's' : '' }}
            </span>
          </div>
          <textarea
            v-model="yamlSource"
            @input="onYamlInput"
            class="flex-1 bg-surface-0 text-text font-mono text-[11px] leading-relaxed p-3 resize-none outline-none border-none min-h-0"
            spellcheck="false"
          />
        </div>

        <!-- Right panel with tabs (canvas view only) -->
        <aside v-if="view === 'canvas'" class="w-80 border-l border-border flex flex-col overflow-hidden bg-surface-1">

          <!-- Tab bar -->
          <div class="flex border-b border-border shrink-0">
            <button
              @click="rightTab = 'properties'"
              :class="['flex-1 py-2 text-xs font-medium transition-colors', rightTab === 'properties' ? 'text-text border-b-2 border-accent' : 'text-text-muted hover:text-text-muted']"
            >Properties</button>
            <button
              @click="rightTab = 'generate'"
              :class="['flex-1 py-2 text-xs font-medium transition-colors flex items-center justify-center gap-1.5', rightTab === 'generate' ? 'text-text border-b-2 border-violet-400' : 'text-text-muted hover:text-text-muted']"
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
              </svg>
              Generate
            </button>
          </div>

          <!-- Properties tab -->
          <template v-if="rightTab === 'properties'">
            <!-- Pending connection: user dragged an edge, waiting for trigger name -->
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
                  <input v-model="meta.name"        class="input" placeholder="Name" @input="isDirty=true"/>
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
                  <div class="space-y-1">
                    <div class="flex items-center justify-between">
                      <label class="text-[10px] text-text-muted font-semibold uppercase tracking-wider">Business Process Description</label>
                      <button @click="showDescriptionModal = true" class="text-text-muted hover:text-accent transition-colors" title="Expand editor">
                        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/>
                        </svg>
                      </button>
                    </div>
                    <textarea v-model="meta.processDescription" class="input h-32 resize-none text-sm" placeholder="Detailed natural language description of the process..." @input="isDirty=true"/>
                  </div>
                  <div class="space-y-1">
                    <div class="flex items-center justify-between px-1">
                      <label class="text-[10px] text-text-muted font-semibold uppercase tracking-wider">Modular Workflow</label>
                      <span v-if="!entStore.hasFeature('subprocesses')" class="badge bg-amber-500/10 text-amber-500 text-[9px] font-bold tracking-widest uppercase">Pro Feature</span>
                    </div>
                    <label 
                      class="flex items-center gap-2 p-3 rounded-lg bg-indigo-500/5 border border-indigo-500/20 cursor-pointer hover:bg-indigo-500/10 transition-colors group"
                      :class="{ 'opacity-40 grayscale pointer-events-none select-none': !entStore.hasFeature('subprocesses') }"
                    >
                      <input type="checkbox" v-model="meta.reusable" class="accent-indigo-500" @change="isDirty=true" :disabled="!entStore.hasFeature('subprocesses')" />
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

              <!-- Workflow Prompt (relocated to bottom) -->
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
          </template>

          <!-- Generate tab -->
          <GeneratePanel
            v-else-if="rightTab === 'generate'"
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
        </aside>
      </template>
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
import GeneratePanel          from '@/components/designer/GeneratePanel.vue'
import CodeEditorModal       from '@/components/designer/CodeEditorModal.vue'
import BBFieldModal         from '@/components/designer/BBFieldModal.vue'
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

const view              = ref('canvas')
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
const showBBModal       = ref(false)

// -- Schema Pop-out state --
const showSchemaModal = ref(false)
const activeSchemaRef = ref(null)
const activeSchemaName = ref('')

function onOpenSchemaPopout({ name, field }) {
  activeSchemaName.value = name
  // For 'object' we edit 'properties', for 'list' we edit 'items.properties' (if it's an object)
  if (field.type === 'object') {
    if (!field.properties) field.properties = {}
    activeSchemaRef.value = field.properties
  } else if (field.type === 'list') {
    if (!field.items) field.items = { type: 'string' }
    if (field.items.type === 'object') {
      if (!field.items.properties) field.items.properties = {}
      activeSchemaRef.value = field.items.properties
    } else {
      // If it's a simple list, we don't have sub-properties to edit in a modal yet
      // but the editor handles simple types inline.
      return
    }
  }
  showSchemaModal.value = true
}

// -- Generation Tab State (lifted for persistence) --
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
  { type: 'emit_event', label: 'Emit Event', dot: 'bg-blue-400' },
  { type: 'timeout_node', label: 'Timeout Node', dot: 'bg-red-500' },
  { type: 'terminal', label: 'Terminal (End)', dot: 'bg-indigo-400' },
]

const paletteTriggers = [
  { isTrigger: true, triggerType: 'telegram', label: 'Telegram Trigger' },
  { isTrigger: true, triggerType: 'discord', label: 'Discord Trigger' },
  { isTrigger: true, triggerType: 'cron', label: 'Cron Trigger' },
  { isTrigger: true, triggerType: 'webhook', label: 'Webhook Trigger' },
]

const paletteActions = [
  { type: 'telegram_output', label: 'Send Telegram' },
  { type: 'discord_output', label: 'Send Discord' },
]

function onDragStart(event, node) {
  event.dataTransfer.setData('application/vueflow', JSON.stringify(node))
  event.dataTransfer.effectAllowed = 'move'
}

function onDrop(event) {
  event.preventDefault()

  const nodeStr = event.dataTransfer.getData('application/vueflow')
  if (!nodeStr) return

  const paletteNode = JSON.parse(nodeStr)
  
  // Get bounds of the VueFlow container element
  const container = event.currentTarget.getBoundingClientRect()
  
  // Calculate client coordinates relative to container
  const position = project({
    x: event.clientX - container.left,
    y: event.clientY - container.top,
  })

  // Generate a unique name / id for the node
  let name = ''
  let type = paletteNode.type || 'prompt'

  if (type === 'subprocess' && !entStore.hasFeature('subprocesses')) {
    alert('Subprocess node is a Pro Feature and is locked in the Community Edition.')
    return
  }

  if (paletteNode.isTrigger) {
    type = 'initial'
    name = `trigger_${paletteNode.triggerType}`
  } else if (type === 'timeout_node') {
    name = 'timeout_node'
  } else {
    name = `${type}_${Math.random().toString(36).substring(2, 7)}`
  }

  // Ensure unique name
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

  // Auto-connect to Timeout Node if it exists
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

// ── Composables ──────────────────────────────────────────────────────────────

const {
  fromLoad, loadDefinition, buildDefinition,
  saveWorkflow, switchVersion, onRunClick, onAIApply: _onAIApply, importYAML,
} = useWorkflowIO({ nodes, edges, meta, bbSchema, agents, versions, isDirty, yamlSource, yamlErrors, saving, flowKey, showRunModal, view, wfStore, router })

const { onYamlInput } = useCanvasSync({ nodes, edges, meta, isDirty, view, yamlSource, yamlErrors, flowKey, fromLoad, loadDefinition, buildDefinition })

const {
  applyLayout,
  onNodeClick, onEdgeClick, onPaneClick, onNodesChange,
  onConnect, confirmConnection, cancelConnection,
  getAgent,
  updateNode, updateEdge, deleteSelected,
  addState, onAddState,
  addBBField, removeBBField,
} = useCanvasActions({ nodes, edges, agents, isDirty, selected, pendingConnection, showAddState, bbSchema, fitView, layout })

// ── Keyboard Shortcuts & Undo/Redo ───────────────────────────────────────────

const history = ref([])
const historyIndex = ref(-1)
let isRestoring = false

function takeSnapshot() {
  if (isRestoring) return
  // Clean forward history
  if (historyIndex.value < history.value.length - 1) {
    history.value = history.value.slice(0, historyIndex.value + 1)
  }
  // Max 50 steps
  if (history.value.length >= 50) history.value.shift()
  
  history.value.push(JSON.stringify({
    nodes: nodes.value,
    edges: edges.value,
    meta:  meta,
    bb:    bbSchema,
    agents: agents.value
  }))
  historyIndex.value = history.value.length - 1
}

function undo() {
  if (historyIndex.value <= 0) return
  historyIndex.value--
  restoreSnapshot(history.value[historyIndex.value])
}

function redo() {
  if (historyIndex.value >= history.value.length - 1) return
  historyIndex.value++
  restoreSnapshot(history.value[historyIndex.value])
}

function restoreSnapshot(snapshotJson) {
  isRestoring = true
  const data = JSON.parse(snapshotJson)
  nodes.value = data.nodes
  edges.value = data.edges
  Object.assign(meta, data.meta)
  Object.assign(bbSchema, data.bb)
  agents.value = data.agents
  isDirty.value = true
  nextTick(() => isRestoring = false)
}

// Watch for changes to trigger snapshots (debounced slightly or triggered by isDirty)
watch(isDirty, (newVal) => {
  if (newVal && !isRestoring) takeSnapshot()
})

watch(selected, (newSel) => {
  if (newSel) {
    rightTab.value = 'properties'
  }
})

function handleKeydown(e) {
  const isTyping = ['INPUT', 'TEXTAREA'].includes(e.target.tagName) || e.target.isContentEditable
  
  // Save: Cmd+S / Ctrl+S
  if ((e.metaKey || e.ctrlKey) && e.key === 's') {
    e.preventDefault()
    saveWorkflow()
  }
  
  // Undo: Cmd+Z
  if ((e.metaKey || e.ctrlKey) && !e.shiftKey && e.key === 'z') {
    if (isTyping) return
    e.preventDefault()
    undo()
  }
  
  // Redo: Cmd+Shift+Z or Cmd+Y
  if ((e.metaKey || e.ctrlKey) && ( (e.shiftKey && e.key === 'z') || e.key === 'y')) {
    if (isTyping) return
    e.preventDefault()
    redo()
  }

  // Delete: Backspace/Del
  if ((e.key === 'Backspace' || e.key === 'Delete') && !isTyping) {
    if (selected.value) {
      e.preventDefault()
      deleteSelected()
    }
  }

  // Layout: 'a'
  if (e.key === 'a' && !isTyping) {
    e.preventDefault()
    applyLayout()
  }
}

async function onAIApply(payload) {
  _onAIApply(payload)
  await nextTick()
  await nextTick()
  applyLayout(false)
  // Silently save after each pipeline step so intermediate results are never lost.
  // Skips agent-assignment checks — the workflow may not be fully wired yet.
  if (meta.name) {
    try { await saveWorkflow(true) } catch { /* best-effort */ }
  }
}

async function handleImportYAML() {
  const ok = await importYAML()
  if (ok) {
    await nextTick()
    await nextTick()
    applyLayout(false)
  }
}

// ── Load on mount ────────────────────────────────────────────────────────────

onMounted(async () => {
  // Check for AI-proposed fix from the Monitor
  const fixData = sessionStorage.getItem('chainnodes_ai_fix')
  if (fixData) {
    try {
      const { yaml: fixYaml, explanation } = JSON.parse(fixData)
      onAIApply({ yamlStr: fixYaml })
      sessionStorage.removeItem('chainnodes_ai_fix')
      // Show success toast or notification would be nice here
      console.log('AI fix applied:', explanation)
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
      // Always apply a fresh layout when opening the designer so the canvas
      // is cleanly arranged regardless of stored position data.
      if (nodes.value.length > 1) {
        await nextTick()
        applyLayout(false)
      }
    } catch (e) {
      console.error('Failed to load workflow:', e)
      alert(`Could not load workflow "${name}@${version}". It may have been saved with an older version format. Try deleting and re-creating it.`)
    }
  } else {
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
  
  // Initialize history with initial state
  takeSnapshot()

  window.addEventListener('keydown', handleKeydown)
})

import { onUnmounted } from 'vue'
onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})

watch(() => nodes.value, () => {
  // Vue Flow doesn't always trigger isDirty on internal mutations, so we nudge it
  if (!isRestoring) isDirty.value = true
}, { deep: true })

</script>

<style>
/* Hide timeout/condition exception edges when the toggle is off.
   VueFlow sets the edge object's `class` on the wrapping <g> element,
   so `.timeout-edge` targets that wrapper — no need to pierce shadow DOM. */
.hide-timeout-edges .timeout-edge { display: none; }
</style>
