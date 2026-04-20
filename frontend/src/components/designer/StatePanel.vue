<template>
  <div class="p-4 space-y-2.5 flex-1 overflow-y-auto">
    <!-- Header -->
    <div class="flex items-center gap-2 mb-1">
      <span class="text-xs font-semibold text-slate-400 uppercase tracking-wider flex-1">State</span>
      <button class="btn-danger text-xs py-1" @click="$emit('delete')">Delete</button>
    </div>

    <div class="space-y-1">
      <label class="text-xs text-slate-500">Name</label>
      <input class="input" v-model="form.name" />
    </div>
    <div class="space-y-1">
      <label class="text-xs text-slate-500">Type</label>
      <select class="input" v-model="form.type">
        <option value="initial">initial</option>
        <option value="terminal">terminal</option>
        <optgroup label="──────────────"></optgroup>
        <option value="intermediate">intermediate</option>
        <option value="script">script</option>
        <option value="code">code</option>
        <optgroup label="──────────────"></optgroup>
        <option value="skill_call">skill_call</option>
        <optgroup label="──────────────"></optgroup>
        <option value="hitl">hitl</option>
        <option value="wait">wait</option>
      </select>
    </div>
    <div class="space-y-1">
      <label class="text-xs text-slate-500">Instructions</label>
      <textarea class="input text-xs leading-relaxed resize-none h-16" v-model="form.instructions" placeholder="Describe what should happen in this state…" />
      <!-- Generate Code button — only shown for code nodes -->
      <div v-if="form.type === 'code'" class="pt-0.5">
        <button
          :class="['w-full flex items-center justify-center gap-1.5 text-xs py-1.5 px-3 rounded font-medium transition-colors border',
            codegenLoading
              ? 'bg-teal-500/10 text-teal-500/60 border-teal-500/20 cursor-not-allowed'
              : 'bg-teal-500/10 hover:bg-teal-500/20 text-teal-400 border-teal-500/30 hover:border-teal-500/60']"
          :disabled="codegenLoading || !form.instructions.trim()"
          @click="generateCode"
        >
          <svg v-if="!codegenLoading" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
          </svg>
          <svg v-else class="animate-spin" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
          </svg>
          {{ codegenLoading ? 'Generating…' : 'Generate Code' }}
        </button>
        <p v-if="codegenError" class="text-[10px] text-red-400 font-mono mt-1">{{ codegenError }}</p>
        <p v-if="codegenExplanation" class="text-[10px] text-teal-400/70 mt-1 leading-relaxed">✓ {{ codegenExplanation }}</p>
      </div>
    </div>
    <!-- Wait condition -->
    <template v-if="form.type === 'wait'">
      <div class="space-y-1">
        <label class="text-xs text-slate-500">Condition (expr-lang)</label>
        <input class="input font-mono text-xs" v-model="form.condition" placeholder="e.g. blackboard.count >= 2" />
      </div>
    </template>
    
    <!-- HITL Form Schema -->
    <template v-if="form.type === 'hitl'">
      <div class="pt-3 border-t border-border space-y-2.5">
        <h4 class="text-xs font-semibold text-amber-500/80 uppercase tracking-wider">HITL Form Schema</h4>
        <p class="text-[10px] text-slate-600 leading-relaxed">
          Define a JSON Schema to render a custom input form in the dashboard.
        </p>
        <CodeEditor
          v-model="form.form_schema"
          mode="json"
          height="200px"
          @update:modelValue="validateJson"
        />
        <div v-if="jsonError" class="text-[10px] text-red-400 font-mono mt-1 break-all bg-red-500/5 p-1.5 rounded border border-red-500/10">
          ⚠ {{ jsonError }}
        </div>
        <div class="flex items-center gap-3">
          <button 
            class="text-[10px] text-slate-500 hover:text-slate-300 flex items-center gap-1"
            @click="form.form_schema = JSON.stringify({type: 'object', properties: { field_1: {type: 'string', title: 'Example Field'}}}, null, 2); validateJson()"
          >
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
            </svg>
            Insert Template
          </button>
          <div class="w-px h-3 bg-border"/>
          <button 
            class="text-[10px] text-indigo-400 hover:text-indigo-300 flex items-center gap-1 disabled:opacity-30 disabled:cursor-not-allowed"
            :disabled="!!jsonError || !form.form_schema"
            @click="showPreview = true"
          >
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/>
            </svg>
            Preview Form
          </button>
        </div>
      </div>
    </template>

    <!-- Code section -->
    <template v-if="form.type === 'code'">
      <div class="pt-3 border-t border-border space-y-2.5">
        <div class="flex items-center gap-2">
          <h4 class="text-xs font-semibold text-teal-400/80 uppercase tracking-wider flex-1">Code</h4>
          <span class="text-[9px] font-mono text-slate-600 uppercase tracking-wider bg-teal-500/10 px-1.5 py-0.5 rounded">JavaScript</span>
          <!-- Pop-out button -->
          <button
            class="text-slate-500 hover:text-teal-400 transition-colors"
            title="Open in fullscreen editor"
            @click="showCodeModal = true"
          >
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/>
              <line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/>
            </svg>
          </button>
        </div>
        <p class="text-[10px] text-slate-600 leading-relaxed">
          Read &amp; write <code class="text-teal-400/80">bb</code> directly. Return
          <code class="text-teal-400/80">&#123; trigger, blackboard_updates?, reasoning? &#125;</code>
          or call <code class="text-teal-400/80">trigger('name')</code> to exit early.
        </p>
        <CodeEditor
          v-model="form.code"
          :bb-schema="bbSchema"
          height="300px"
        />
      </div>
    </template>

    <!-- Agent field -->
    <template v-if="form.type !== 'script' && form.type !== 'wait' && form.type !== 'code' && form.type !== 'skill_call'">
      <div class="space-y-1">
        <label class="text-xs text-slate-500">Agent</label>
        <div class="space-y-1.5">
          <AgentCombobox
            v-model="form.agent"
            :agents="agents"
            placeholder="agent_name (required for automation)"
          />
          <p v-if="(form.type === 'initial' || form.type === 'intermediate') && !form.agent" class="text-[10px] text-amber-500 font-medium">
            ⚠ Agent required for automated reasoning.
          </p>
        </div>
      </div>
    </template>

    <div class="space-y-1">
      <label class="text-xs text-slate-500">Timeout</label>
      <input class="input" v-model="form.timeout" placeholder="e.g. 30m" />
    </div>
    <template v-if="form.timeout">
      <div class="space-y-1">
        <label class="text-xs text-slate-500">On Timeout</label>
        <input class="input font-mono" v-model="form.on_timeout" placeholder="trigger_name" />
      </div>
    </template>

    <!-- Script section -->
    <template v-if="form.type === 'script'">
      <div class="pt-3 border-t border-border space-y-2.5">
        <h4 class="text-xs font-semibold text-amber-500/80 uppercase tracking-wider">Script</h4>
        <p class="text-[10px] text-slate-600 leading-relaxed">Expressions use expr-lang. Blackboard fields are available as variables.</p>
        <div class="space-y-1">
          <label class="text-xs text-slate-500">Trigger Expression *</label>
          <input class="input font-mono text-xs" v-model="form.script_trigger" placeholder='amount > 1000 ? "needs_review" : "auto_approve"' />
        </div>
        <div class="space-y-1">
          <div class="flex items-center justify-between">
            <label class="text-xs text-slate-500">Blackboard Updates</label>
            <button class="text-[10px] text-indigo-400 hover:text-indigo-300" @click="form.script_updates.push({ key: '', expr: '' })">+ Add</button>
          </div>
          <div v-for="(row, i) in form.script_updates" :key="i" class="flex gap-1.5 items-center">
            <input class="input font-mono text-xs flex-1" v-model="row.key" placeholder="field" />
            <span class="text-slate-600 text-xs shrink-0">=</span>
            <input class="input font-mono text-xs flex-[2]" v-model="row.expr" placeholder="amount * 0.2" />
            <button class="text-slate-600 hover:text-red-400 text-xs shrink-0 px-1" @click="form.script_updates.splice(i, 1)">✕</button>
          </div>
        </div>
      </div>
    </template>

    <!-- Skill Call section -->
    <template v-if="form.type === 'skill_call'">
      <div class="pt-3 border-t border-border space-y-2.5">
        <h4 class="text-xs font-semibold text-violet-400/80 uppercase tracking-wider">Skill Call</h4>
        <p class="text-[10px] text-slate-600 leading-relaxed">
          Invoke another workflow as a child skill. The child runs to completion
          before this state transitions.
        </p>

        <div class="space-y-1">
          <label class="text-xs text-slate-500">Skill</label>
          <SkillPicker
            v-model="form.skill_ref"
            @skill-selected="onSkillSelected"
          />
        </div>

        <div class="space-y-1">
          <label class="text-xs text-slate-500">Completion Trigger</label>
          <input
            class="input font-mono text-xs"
            v-model="form.skill_completion_trigger"
            placeholder="skill_done"
          />
        </div>

        <div class="space-y-1">
          <label class="text-xs text-slate-500">Failure Trigger <span class="text-slate-600 font-normal">(optional, default: skill_failed)</span></label>
          <input
            class="input font-mono text-xs"
            v-model="form.skill_failure_trigger"
            placeholder="skill_failed"
          />
        </div>

        <SkillInputMappingEditor
          label="Input Mappings — child port ← parent bb field"
          :mappings="form.skill_input_mappings"
          @update:mappings="form.skill_input_mappings = $event"
        />

        <SkillInputMappingEditor
          label="Output Mappings — parent bb field ← child port"
          :mappings="form.skill_output_mappings"
          @update:mappings="form.skill_output_mappings = $event"
        />
      </div>
    </template>

    <!-- Transitions -->
    <div class="pt-3 border-t border-border space-y-3">
      <h4 class="text-xs font-semibold text-slate-500 uppercase tracking-wider">Transitions</h4>
      <div class="space-y-1.5">
        <div class="text-[10px] text-slate-500 font-medium">Incoming</div>
        <div v-if="incomingTransitions.length" class="space-y-1">
          <div v-for="t in incomingTransitions" :key="`${t.from}__${t.label}`"
               class="flex items-center gap-1.5 px-2 py-1.5 rounded bg-surface-0 border border-border text-[10px]">
            <span class="text-slate-400 font-mono truncate max-w-[80px]">{{ t.from }}</span>
            <span class="text-slate-600">→</span>
            <span class="text-indigo-400 font-mono font-medium truncate flex-1">{{ t.label }}</span>
          </div>
        </div>
        <p v-else class="text-[10px] text-slate-600 italic">None</p>
      </div>
      <div class="space-y-1.5">
        <div class="text-[10px] text-slate-500 font-medium">Outgoing</div>
        <div v-if="outgoingTransitions.length" class="space-y-1">
          <div v-for="t in outgoingTransitions" :key="`${t.label}__${t.to}`"
               class="flex items-center gap-1.5 px-2 py-1.5 rounded bg-surface-0 border border-border text-[10px]">
            <span class="text-indigo-400 font-mono font-medium truncate flex-1">{{ t.label }}</span>
            <span class="text-slate-600">→</span>
            <span class="text-slate-400 font-mono truncate max-w-[80px]">{{ t.to }}</span>
          </div>
        </div>
        <p v-else class="text-[10px] text-slate-600 italic">None</p>
      </div>
    </div>

    <!-- Agent config section -->
    <template v-if="form.type !== 'script' && form.type !== 'skill_call' && form.agent">
      <div class="pt-3 border-t border-border space-y-2.5">
        <h4 class="text-xs font-semibold text-slate-500 uppercase tracking-wider">Agent Config</h4>
        <div class="space-y-1">
          <label class="text-xs text-slate-500">Provider</label>
          <select class="input text-xs" v-model="agentForm.provider">
            <option value="">Default (global setting)</option>
            <option value="anthropic">Anthropic</option>
            <option value="openai">OpenAI</option>
            <option value="grok">Grok (xAI)</option>
            <option value="deepseek">Deepseek</option>
            <option value="gemini">Gemini</option>
            <option value="ollama">Ollama (local)</option>
          </select>
        </div>
        <div class="space-y-1">
          <label class="text-xs text-slate-500">Model</label>
          <input class="input text-xs" v-model="agentForm.model" placeholder="Leave blank to use provider default" />
        </div>
        <div class="space-y-1">
          <label class="text-xs text-slate-500">Task Queue</label>
          <input class="input font-mono text-xs" v-model="agentForm.task_queue" placeholder="asm-workers (default)" />
        </div>
        <div class="space-y-1">
          <label class="text-xs text-slate-500">MCP Servers</label>
          <MCPServerPicker v-model="agentForm.mcp_servers" />
        </div>
      </div>

      <!-- Convert to Script section -->
      <div class="pt-3 border-t border-border space-y-2">
        <div class="flex items-center gap-2">
          <h4 class="text-xs font-semibold text-slate-500 uppercase tracking-wider flex-1">Convert to Script</h4>
          <button
            :class="['text-xs px-2.5 py-1 rounded font-medium transition-colors', scaffoldLoading ? 'bg-indigo-500/20 text-indigo-400 cursor-not-allowed' : 'bg-indigo-600 hover:bg-indigo-500 text-white']"
            :disabled="scaffoldLoading"
            @click="generateScript"
          >{{ scaffoldLoading ? 'Generating…' : 'Generate Script' }}</button>
        </div>
        <p class="text-[10px] text-slate-600 leading-relaxed">Convert this LLM agent into deterministic expr-lang logic for zero-latency execution.</p>
        <p v-if="scaffoldError" class="text-[10px] text-red-400 font-mono">{{ scaffoldError }}</p>
        <template v-if="scaffoldScript">
          <div class="bg-surface-0 border border-border rounded p-2 space-y-2">
            <div class="space-y-0.5">
              <span class="text-[9px] text-slate-500 font-bold uppercase">Trigger</span>
              <div class="text-[10px] font-mono text-indigo-400">{{ scaffoldScript.trigger }}</div>
            </div>
            <div v-if="Object.keys(scaffoldScript.updates || {}).length" class="space-y-0.5">
              <span class="text-[9px] text-slate-500 font-bold uppercase">Updates</span>
              <div v-for="(v, k) in scaffoldScript.updates" :key="k" class="text-[10px] font-mono text-slate-400 flex gap-2">
                <span class="text-slate-500 shrink-0">{{ k }}</span>
                <span class="text-slate-600 shrink-0">=</span>
                <span class="truncate">{{ v }}</span>
              </div>
            </div>
          </div>
          <div class="flex gap-2">
            <button class="btn-primary text-[10px] flex-1 py-1.5" @click="applyScript">Apply to State</button>
            <button class="btn-ghost text-[10px] px-2 py-1.5" @click="copyScript">Copy</button>
          </div>
        </template>
      </div>
    </template>

    <button class="btn-primary w-full mt-1" @click="onApply">Apply</button>
  </div>

  <!-- Fullscreen code editor pop-out -->
  <CodeEditorModal
    v-if="showCodeModal"
    v-model="form.code"
    :state-name="form.name"
    :bb-schema="bbSchema"
    @close="showCodeModal = false"
  />

  <!-- HITL Form Preview Modal -->
  <div v-if="showPreview" class="fixed inset-0 z-[60] flex items-center justify-center p-6 bg-black/60 backdrop-blur-sm" @click.self="showPreview = false">
    <div class="bg-surface-1 border border-border rounded-2xl w-full max-w-lg shadow-2xl flex flex-col overflow-hidden animate-in zoom-in duration-200">
      <header class="px-6 py-4 border-b border-border flex items-center justify-between">
        <h3 class="text-sm font-bold text-slate-200">Form Preview</h3>
        <button @click="showPreview = false" class="text-slate-500 hover:text-slate-300 transition-colors">✕</button>
      </header>
      <div class="p-8 overflow-y-auto max-h-[70vh]">
        <div class="mb-6 pb-6 border-b border-border/50">
          <p class="text-xs text-slate-400">This is how the form will appear to operators during HITL resolution.</p>
        </div>
        <JsonSchemaForm
          :schema="parsedFormSchema"
          :initial-data="{}"
        />
      </div>
      <footer class="px-6 py-4 bg-surface-0 border-t border-border flex justify-end">
        <button @click="showPreview = false" class="btn-primary px-6">Close Preview</button>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, computed, watch } from 'vue'
import { api } from '@/stores/auth'
import AgentCombobox from './AgentCombobox.vue'
import CodeEditor from './CodeEditor.vue'
import CodeEditorModal from './CodeEditorModal.vue'
import MCPServerPicker from './MCPServerPicker.vue'
import SkillInputMappingEditor from './SkillInputMappingEditor.vue'
import SkillPicker from './SkillPicker.vue'
import JsonSchemaForm from '@/components/monitor/JsonSchemaForm.vue'

// Starter template shown when a new code state is created.
const defaultCodeTemplate = `// Blackboard fields are available via bb.*
// Modify bb directly and return a trigger.

// Example:
// bb.result = bb.value_1 + bb.value_2;
// return { trigger: 'done', reasoning: 'Computed result' };

return {
  blackboard_updates: {},
  trigger: 'done',
  reasoning: ''
};`

const props = defineProps({
  node:     Object,
  edges:    { type: Array,  default: () => [] },
  agentDef: { type: Object, default: null },
  bbSchema: { type: Object, default: () => ({}) },
  agents:   { type: Array,  default: () => [] },
})
const emit = defineEmits(['update', 'delete'])

const STATE_TYPES = ['initial', 'terminal', 'intermediate', 'script', 'code', 'hitl', 'wait']

// When a skill is selected from the picker, pre-populate the mapping rows
// with the skill's declared port names (bbField left blank for the user to fill).
// Only populates rows that aren't already filled to preserve manual edits.
function onSkillSelected(sk) {
  if (!sk) return
  if (sk.inputs?.length) {
    const existing = new Set(form.skill_input_mappings.map(m => m.name))
    sk.inputs.forEach(p => {
      if (!existing.has(p.name)) {
        form.skill_input_mappings.push({ name: p.name, bbField: '' })
      }
    })
  }
  if (sk.outputs?.length) {
    const existing = new Set(form.skill_output_mappings.map(m => m.name))
    sk.outputs.forEach(p => {
      if (!existing.has(p.name)) {
        form.skill_output_mappings.push({ name: p.name, bbField: '' })
      }
    })
  }
}

// Helper to convert skill_call mappings object { portName: bbField } → array of { name, bbField }
function mappingsToArray(obj) {
  if (!obj) return []
  return Object.entries(obj).map(([name, bbField]) => ({ name, bbField }))
}

const form = reactive({
  name:           props.node.name         ?? '',
  type:           props.node.type         ?? 'intermediate',
  instructions:   props.node.instructions ?? '',
  agent:          props.node.agent        ?? '',
  condition:      props.node.condition    ?? '',
  timeout:        props.node.timeout      ?? '',
  on_timeout:     props.node.on_timeout   ?? '',
  script_trigger: props.node.script?.trigger ?? '',
  script_updates: props.node.script?.updates
    ? Object.entries(props.node.script.updates).map(([k, v]) => ({ key: k, expr: v }))
    : [],
  code: props.node.code?.code ?? defaultCodeTemplate,
  // skill_call fields
  skill_ref:                 props.node.skill_ref            ?? '',
  skill_completion_trigger:  props.node.skill_completion_trigger ?? 'skill_done',
  skill_failure_trigger:     props.node.skill_failure_trigger    ?? '',
  skill_input_mappings:      mappingsToArray(props.node.skill_call?.input_mappings),
  skill_output_mappings:     mappingsToArray(props.node.skill_call?.output_mappings),
  form_schema: props.node.form_schema ? JSON.stringify(props.node.form_schema, null, 2) : '',
})

const agentForm = reactive({
  provider:    props.agentDef?.config?.provider    ?? '',
  model:       props.agentDef?.model               ?? '',
  task_queue:  props.agentDef?.task_queue          ?? '',
  prompt:      props.agentDef?.config?.prompt      ?? '',
  mcp_servers: props.agentDef?.config?.mcp_servers ?? '',
})

const scaffoldScript  = ref(null)
const scaffoldLoading = ref(false)
const scaffoldError   = ref('')
const showCodeModal   = ref(false)

const codegenExplanation = ref('')

const jsonError = ref('')
const showPreview = ref(false)

const parsedFormSchema = computed(() => {
  try {
    return form.form_schema ? JSON.parse(form.form_schema) : null
  } catch {
    return null
  }
})

function validateJson() {
  if (!form.form_schema) {
    jsonError.value = ''
    return
  }
  try {
    JSON.parse(form.form_schema)
    jsonError.value = ''
  } catch (e) {
    jsonError.value = e.message
  }
}

// Initial validation
validateJson()

async function generateCode() {
  if (!form.instructions.trim()) return
  codegenLoading.value     = true
  codegenError.value       = ''
  codegenExplanation.value = ''
  try {
    const { data } = await api.post('/designer/codegen', {
      instructions:   form.instructions,
      state_name:     form.name,
      valid_triggers: outgoingTransitions.value.map(t => t.label),
      bb_schema:      Object.fromEntries(Object.entries(props.bbSchema).map(([k, v]) => [k, v.type])),
      existing_code:  form.code !== defaultCodeTemplate ? form.code : '',
    })
    form.code = data.code
    codegenExplanation.value = data.explanation ?? ''
  } catch (e) {
    codegenError.value = e.response?.data?.error ?? e.message
  } finally {
    codegenLoading.value = false
  }
}

const incomingTransitions = computed(() =>
  props.edges.filter(e => e.target === props.node.name).map(e => ({ from: e.source, label: e.label }))
)
const outgoingTransitions = computed(() =>
  props.edges.filter(e => e.source === props.node.name).map(e => ({ to: e.target, label: e.label }))
)

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
      bb_schema:   Object.fromEntries(Object.entries(props.bbSchema).map(([k, v]) => [k, v.type])),
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
  form.type           = 'script'
  form.agent          = ''
  form.script_trigger = scaffoldScript.value.trigger
  form.script_updates = Object.entries(scaffoldScript.value.updates || {}).map(([key, expr]) => ({ key, expr }))
  onApply()
  scaffoldScript.value = null
}

function copyScript() {
  if (!scaffoldScript.value) return
  navigator.clipboard.writeText(JSON.stringify(scaffoldScript.value, null, 2))
}

// Helper to convert mapping array [{ name, bbField }] → object { name: bbField }
function mappingsToObject(arr) {
  if (!arr || !arr.length) return undefined
  const filtered = arr.filter(m => m.name)
  if (!filtered.length) return undefined
  return Object.fromEntries(filtered.map(m => [m.name, m.bbField]))
}

function onApply() {
  if (form.type === 'hitl' && form.form_schema) {
    try {
      JSON.parse(form.form_schema)
    } catch (e) {
      alert('Cannot apply changes: Invalid HITL Form Schema JSON.\n\n' + e.message)
      return
    }
  }

  const scriptDef = form.type === 'script' ? {
    trigger: form.script_trigger || '',
    updates: form.script_updates.length
      ? Object.fromEntries(form.script_updates.filter(r => r.key).map(r => [r.key, r.expr]))
      : undefined,
  } : undefined

  const codeDef = form.type === 'code' ? {
    language: 'javascript',
    code:     form.code || '',
  } : undefined

  const skillCallDef = form.type === 'skill_call' ? {
    skill_ref:          form.skill_ref          || '',
    completion_trigger: form.skill_completion_trigger || 'skill_done',
    failure_trigger:    form.skill_failure_trigger   || undefined,
    input_mappings:     mappingsToObject(form.skill_input_mappings),
    output_mappings:    mappingsToObject(form.skill_output_mappings),
  } : undefined

  const nodeData = {
    name:         form.name,
    type:         form.type,
    instructions: form.instructions || undefined,
    agent:        (form.type !== 'script' && form.type !== 'wait' && form.type !== 'code' && form.type !== 'skill_call') ? (form.agent || undefined) : undefined,
    script:       scriptDef,
    code:         codeDef,
    condition:    form.type === 'wait' ? (form.condition || undefined) : undefined,
    timeout:      form.timeout    || undefined,
    on_timeout:   form.on_timeout || undefined,
    skill_call:   skillCallDef,
    form_schema:  form.type === 'hitl' && form.form_schema ? JSON.parse(form.form_schema) : undefined,
  }
  const agentData = (form.type !== 'script' && form.type !== 'code' && form.type !== 'skill_call' && form.agent) ? {
    name:       form.agent,
    model:      agentForm.model      || undefined,
    task_queue: agentForm.task_queue || undefined,
    config: (agentForm.prompt || agentForm.mcp_servers || agentForm.provider) ? {
      prompt:      agentForm.prompt      || undefined,
      mcp_servers: agentForm.mcp_servers || undefined,
      provider:    agentForm.provider    || undefined,
    } : undefined,
  } : null
  emit('update', { node: nodeData, agentDef: agentData })
}
</script>
