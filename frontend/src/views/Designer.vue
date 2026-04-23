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
              @click="rightTab = 'generate'"
              :class="['flex-1 py-2 text-xs font-medium transition-colors flex items-center justify-center gap-1.5', rightTab === 'generate' ? 'text-slate-200 border-b-2 border-violet-400' : 'text-slate-500 hover:text-slate-400']"
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
                  <textarea v-model="meta.description" class="input h-16 resize-none" placeholder="Description (short abstract)" @input="isDirty=true"/>
                  <label class="flex items-center gap-2 p-3 rounded-lg bg-indigo-500/5 border border-indigo-500/20 cursor-pointer hover:bg-indigo-500/10 transition-colors group">
                    <input type="checkbox" v-model="meta.reusable" class="accent-indigo-500" @change="isDirty=true" />
                    <div class="flex-1">
                      <div class="text-xs font-semibold text-indigo-300 group-hover:text-indigo-200 transition-colors">Mark as Reusable</div>
                      <div class="text-[10px] text-slate-600">Enables this workflow to be called as a sub-process by others.</div>
                    </div>
                  </label>
                </div>
              </div>

              <!-- Reusable configuration -->
              <template v-if="meta.reusable">
                <div class="space-y-4">
                  <IOPortEditor
                    title="Inputs"
                    :ports="meta.inputs"
                    show-required
                    @change="isDirty = true"
                  />
                  <IOPortEditor
                    title="Outputs"
                    :ports="meta.outputs"
                    @change="isDirty = true"
                  />
                </div>
              </template>

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
              :agents="agents"
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
import { ref, reactive, computed, onMounted, nextTick, shallowRef } from 'vue'
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
import StatePanel             from '@/components/designer/StatePanel.vue'
import EdgePanel              from '@/components/designer/EdgePanel.vue'
import BBFieldModal           from '@/components/designer/BBFieldModal.vue'
import AddStateModal          from '@/components/designer/AddStateModal.vue'
import GeneratePanel          from '@/components/designer/GeneratePanel.vue'
import IOPortEditor           from '@/components/designer/IOPortEditor.vue'

import { useWorkflowStore } from '@/stores/workflows'
import { useWorkflowIO }    from '@/composables/useWorkflowIO'
import { useCanvasSync }    from '@/composables/useCanvasSync'
import { useCanvasActions } from '@/composables/useCanvasActions'
import { useLayout }        from '@/composables/useLayout'

const route   = useRoute()
const router  = useRouter()
const wfStore = useWorkflowStore()
const { fitView } = useVueFlow()
const { layout }  = useLayout()

// ── Shared reactive state ────────────────────────────────────────────────────

const nodes   = ref([])
const edges   = ref([])
const agents  = ref([])
const meta    = reactive({ 
  name: '', version: '1.0.0', versionNumber: 0, description: '', systemPrompt: '', processDescription: '',
  reusable: false, inputs: [], outputs: [], capabilities: []
})
const bbSchema = reactive({})
const versions = ref([])

const view              = ref('canvas')
const rightTab          = ref('properties')
const yamlSource        = ref('')
const yamlError         = ref('')
const saving            = ref(false)
const isDirty           = ref(false)
const showRunModal      = ref(false)
const selected          = ref(null)
const pendingConnection = ref(null)
const showBBModal       = ref(false)
const showAddState      = ref(false)
const flowKey           = ref(0)
const showTimeoutEdges  = ref(true)

// -- Generation Tab State (lifted for persistence) --
const generatePrompt       = ref('')
const generateHistory      = ref([])
const generateInteractions = ref([])
const generateProvider     = ref('')
const generatePipelineStep    = ref(0)
const generatePipelineWarning = ref('')

const nodeTypes = shallowRef({ stateNode: StateNode })
const edgeTypes = shallowRef({ transitionEdge: TransitionEdge })
const workflowName = computed(() => meta.name || '')

// ── Composables ──────────────────────────────────────────────────────────────

const {
  fromLoad, loadDefinition, buildDefinition,
  saveWorkflow, switchVersion, onRunClick, onAIApply: _onAIApply, importYAML,
} = useWorkflowIO({ nodes, edges, meta, bbSchema, agents, versions, isDirty, yamlSource, yamlError, saving, flowKey, showRunModal, view, wfStore, router })

const { onYamlInput } = useCanvasSync({ nodes, edges, meta, isDirty, view, yamlSource, yamlError, flowKey, fromLoad, loadDefinition, buildDefinition })

const {
  applyLayout,
  onNodeClick, onEdgeClick, onPaneClick, onNodesChange,
  onConnect, confirmConnection, cancelConnection,
  getAgent,
  updateNode, updateEdge, deleteSelected,
  addState, onAddState,
  addBBField, removeBBField,
} = useCanvasActions({ nodes, edges, agents, isDirty, selected, pendingConnection, showAddState, bbSchema, fitView, layout })

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

// ── Load on mount ────────────────────────────────────────────────────────────

onMounted(async () => {
  const { name, version } = route.params
  if (name && version) {
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
  }
})

</script>

<style>
/* Hide timeout/condition exception edges when the toggle is off.
   VueFlow sets the edge object's `class` on the wrapping <g> element,
   so `.timeout-edge` targets that wrapper — no need to pierce shadow DOM. */
.hide-timeout-edges .timeout-edge { display: none; }
</style>
