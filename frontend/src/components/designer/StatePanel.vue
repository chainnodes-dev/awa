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
    <template v-if="form.type !== 'script' && form.type !== 'wait' && form.type !== 'code'">
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
    <template v-if="form.type !== 'script' && form.agent">
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
          <input class="input font-mono text-xs" v-model="agentForm.mcp_servers" placeholder="{{ env.MCP_URL }}" />
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
</template>

<script setup>
import { reactive, ref, computed } from 'vue'
import { api } from '@/stores/auth'
import AgentCombobox from './AgentCombobox.vue'
import CodeEditor from './CodeEditor.vue'
import CodeEditorModal from './CodeEditorModal.vue'

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

const codegenLoading     = ref(false)
const codegenError       = ref('')
const codegenExplanation = ref('')

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

function onApply() {
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

  const nodeData = {
    name:         form.name,
    type:         form.type,
    instructions: form.instructions || undefined,
    agent:        (form.type !== 'script' && form.type !== 'wait' && form.type !== 'code') ? (form.agent || undefined) : undefined,
    script:       scriptDef,
    code:         codeDef,
    condition:    form.type === 'wait' ? (form.condition || undefined) : undefined,
    timeout:      form.timeout    || undefined,
    on_timeout:   form.on_timeout || undefined,
  }
  const agentData = (form.type !== 'script' && form.type !== 'code' && form.agent) ? {
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
