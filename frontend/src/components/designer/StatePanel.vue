<template>
  <div class="p-4 space-y-2.5 flex-1 overflow-y-auto">
    <!-- Header -->
    <div class="flex items-center gap-2 mb-1">
      <span class="text-xs font-semibold text-text-muted uppercase tracking-wider flex-1">State</span>
      <button class="btn-danger text-xs py-1" @click="$emit('delete')">Delete</button>
    </div>

    <div class="space-y-1">
      <label class="text-xs text-text-muted">Name</label>
      <input class="input" v-model="form.name" />
    </div>
    <div class="space-y-1">
      <label class="text-xs text-text-muted">Type</label>
      <select class="input" v-model="form.type">
        <option value="initial">initial</option>
        <option value="terminal">terminal</option>
        <optgroup label="──────────────"></optgroup>
        <option value="prompt">prompt</option>
        <option value="script">script</option>
        <option value="code">code</option>
        <optgroup label="──────────────"></optgroup>
        <option value="subprocess">subprocess (Pro 🔒)</option>
        <optgroup label="──────────────"></optgroup>
        <option value="hitl">hitl</option>
        <option value="wait">wait</option>
      </select>
    </div>
    <div class="space-y-1">
      <div class="flex items-center justify-between">
        <label class="text-xs text-text-muted">Instructions</label>
        <button
          class="text-text-muted hover:text-accent transition-colors"
          title="Open in fullscreen editor"
          @click="showInstructionsModal = true"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/>
          </svg>
        </button>
      </div>
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
        <label class="text-xs text-text-muted">Condition (expr-lang)</label>
        <input class="input font-mono text-xs" v-model="form.condition" placeholder="e.g. blackboard.count >= 2" />
      </div>
    </template>
    
    <!-- HITL Form Schema -->
    <template v-if="form.type === 'hitl'">
      <div class="pt-3 border-t border-border space-y-2.5">
        <h4 class="text-xs font-semibold text-amber-500/80 uppercase tracking-wider">HITL Form Schema</h4>
        <p class="text-[10px] text-text-muted leading-relaxed">
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
            class="text-[10px] text-text-muted hover:text-text flex items-center gap-1"
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
          <span class="text-[9px] font-mono text-text-muted uppercase tracking-wider bg-teal-500/10 px-1.5 py-0.5 rounded">JavaScript</span>
          <!-- Pop-out button -->
          <button
            class="text-text-muted hover:text-teal-400 transition-colors"
            title="Open in fullscreen editor"
            @click="showCodeModal = true"
          >
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/>
              <line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/>
            </svg>
          </button>
        </div>
        <p class="text-[10px] text-text-muted leading-relaxed">
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
    <template v-if="form.type !== 'script' && form.type !== 'wait' && form.type !== 'code' && form.type !== 'subprocess'">
      <div class="space-y-1">
        <label class="text-xs text-text-muted">Agent</label>
        <div class="space-y-1.5">
          <AgentCombobox
            v-model="form.agent"
            :agents="agents"
            placeholder="agent_name (required for automation)"
          />
          <p v-if="(form.type === 'initial' || form.type === 'prompt') && !form.agent" class="text-[10px] text-amber-500 font-medium">
            ⚠ Agent required for automated reasoning.
          </p>
        </div>
      </div>
    </template>

    <div class="space-y-1">
      <label class="text-xs text-text-muted">Timeout</label>
      <input class="input" v-model="form.timeout" placeholder="e.g. 30m" />
    </div>
    <template v-if="form.timeout">
      <div class="space-y-1">
        <label class="text-xs text-text-muted">On Timeout</label>
        <input class="input font-mono" v-model="form.on_timeout" placeholder="trigger_name" />
      </div>
    </template>

    <!-- Script section -->
    <template v-if="form.type === 'script'">
      <div class="pt-3 border-t border-border space-y-2.5">
        <h4 class="text-xs font-semibold text-amber-500/80 uppercase tracking-wider">Script</h4>
        <p class="text-[10px] text-text-muted leading-relaxed">Expressions use expr-lang. Blackboard fields are available as variables.</p>
        <div class="space-y-1">
          <label class="text-xs text-text-muted">Trigger Expression *</label>
          <input class="input font-mono text-xs" v-model="form.script_trigger" placeholder='amount > 1000 ? "needs_review" : "auto_approve"' />
        </div>
        <div class="space-y-1">
          <div class="flex items-center justify-between">
            <label class="text-xs text-text-muted">Blackboard Updates</label>
            <button class="text-[10px] text-indigo-400 hover:text-indigo-300" @click="form.script_updates.push({ key: '', expr: '' })">+ Add</button>
          </div>
          <div v-for="(row, i) in form.script_updates" :key="i" class="flex gap-1.5 items-center">
            <input class="input font-mono text-xs flex-1" v-model="row.key" placeholder="field" />
            <span class="text-text-muted text-xs shrink-0">=</span>
            <input class="input font-mono text-xs flex-[2]" v-model="row.expr" placeholder="amount * 0.2" />
            <button class="text-text-muted hover:text-red-400 text-xs shrink-0 px-1" @click="form.script_updates.splice(i, 1)">✕</button>
          </div>
        </div>
      </div>
    </template>

    <!-- Sub-Process section -->
    <template v-if="form.type === 'subprocess'">
      <div class="pt-3 border-t border-border space-y-2.5">
        <div v-if="!entStore.hasFeature('subprocesses')" class="mb-4 p-4 rounded-xl bg-amber-500/10 border border-amber-500/20 space-y-3">
          <div class="flex items-start gap-3">
            <div class="p-2 rounded-lg bg-amber-500/20 text-amber-500 shrink-0">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
            </div>
            <div class="space-y-1">
              <h5 class="text-xs font-bold text-amber-500 uppercase tracking-tight">Enterprise Feature</h5>
              <p class="text-[10px] text-amber-500/80 leading-relaxed font-medium">
                Modular workflows (sub-processes) require a **Pro** or **Enterprise** license. 
                Free tier is limited to single-workflow execution.
              </p>
              <RouterLink to="/usage" class="inline-block pt-1 text-[10px] font-bold text-amber-400 hover:text-amber-300 underline underline-offset-2">
                Upgrade to Unlock &rarr;
              </RouterLink>
            </div>
          </div>
        </div>

        <div :class="{ 'opacity-50 grayscale pointer-events-none': !entStore.hasFeature('subprocesses') }">
          <h4 class="text-xs font-semibold text-violet-400/80 uppercase tracking-wider">Sub-Process Call</h4>
          <p class="text-[10px] text-text-muted leading-relaxed">
            Invoke another workflow as a child process. The child runs to completion
            before this state transitions.
          </p>

        <div class="flex gap-2">
          <div class="space-y-1 flex-[2]">
            <label class="text-xs text-text-muted">Reusable Process</label>
            <ProcessPicker
              v-model="form.process_ref"
              @process-selected="onProcessSelected"
            />
          </div>
          <div class="space-y-1 flex-1">
            <label class="text-xs text-text-muted">Version</label>
            <select
              class="input font-mono text-xs"
              v-model="form.process_version"
            >
              <option value="">latest</option>
              <option v-for="v in availableVersions" :key="v" :value="v">v{{ v }}</option>
            </select>
          </div>
        </div>

        <!-- Full-width Subprocess Summary -->
        <div v-if="subprocessDef" class="rounded-lg bg-violet-500/5 border border-violet-500/20 p-3 space-y-3 w-full">
          <div class="space-y-3">
            <div v-if="subprocessRequiredInputs.length" class="space-y-1">
              <div class="text-[9px] text-text-muted uppercase tracking-wider font-bold border-b border-violet-500/10 pb-0.5">Mandatory Inputs</div>
              <div class="grid grid-cols-1 gap-1">
                <div v-for="p in subprocessRequiredInputs" :key="p.name" class="flex items-center gap-2 text-[10px]">
                  <span class="font-mono text-indigo-300 w-1/2 truncate">{{ p.name }}</span>
                  <span class="text-[9px] text-text-muted italic">{{ p.type }}</span>
                </div>
              </div>
            </div>
            
            <div v-if="subprocessOutputs.length" class="space-y-1">
              <div class="text-[9px] text-text-muted uppercase tracking-wider font-bold border-b border-teal-500/10 pb-0.5">Workflow Results</div>
              <div class="grid grid-cols-1 gap-1">
                <div v-for="p in subprocessOutputs" :key="p.name" class="flex items-center gap-2 text-[10px]">
                  <span class="font-mono text-teal-300 w-1/2 truncate">{{ p.name }}</span>
                  <span class="text-[9px] text-text-muted italic">{{ p.type }}</span>
                </div>
              </div>
            </div>
            
            <div v-if="!subprocessRequiredInputs.length && !subprocessOutputs.length" class="text-[10px] text-text-muted italic bg-surface-0/30 p-2 rounded border border-dashed border-border/40 text-center">
              No public I/O contract declared
            </div>
          </div>
          <div class="flex items-center pt-1 border-t border-border/30">
            <span class="text-[9px] text-text-muted font-mono uppercase tracking-tighter">Reference: {{ subprocessDef.metadata.name }} v{{ subprocessDef.metadata.version }}</span>
          </div>
        </div>

        <div class="space-y-1">
          <label class="text-xs text-text-muted">Completion Trigger</label>
          <input
            class="input font-mono text-xs"
            v-model="form.completion_trigger"
            placeholder="done"
          />
        </div>

        <div class="space-y-1">
          <label class="text-xs text-text-muted">Failure Trigger <span class="text-text-muted font-normal">(optional)</span></label>
          <input
            class="input font-mono text-xs"
            v-model="form.failure_trigger"
            placeholder="subprocess_failed"
          />
        </div>

        <ProcessMappingEditor
          label="Input Mappings (to sub-process)"
          v-model:mappings="form.input_mappings"
          :child-options="subprocessRequiredInputs"
          :parent-options="Object.keys(bbSchema)"
        />
        <ProcessMappingEditor
          label="Output Mappings (from sub-process)"
          v-model:mappings="form.output_mappings"
          :child-options="subprocessOutputs"
          :parent-options="Object.keys(bbSchema)"
        />
      </div>
    </div>
  </template>

    <!-- Transitions -->
    <div class="pt-3 border-t border-border space-y-3">
      <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">Transitions</h4>
      <div class="space-y-1.5">
        <div class="text-[10px] text-text-muted font-medium">Incoming</div>
        <div v-if="incomingTransitions.length" class="space-y-1">
          <div v-for="t in incomingTransitions" :key="`${t.from}__${t.label}`"
               class="flex items-center gap-1.5 px-2 py-1.5 rounded bg-surface-0 border border-border text-[10px]">
            <span class="text-text-muted font-mono truncate max-w-[80px]">{{ t.from }}</span>
            <span class="text-text-muted">→</span>
            <span class="text-indigo-400 font-mono font-medium truncate flex-1">{{ t.label }}</span>
          </div>
        </div>
        <p v-else class="text-[10px] text-text-muted italic">None</p>
      </div>
      <div class="space-y-1.5">
        <div class="text-[10px] text-text-muted font-medium">Outgoing</div>
        <div v-if="outgoingTransitions.length" class="space-y-1">
          <div v-for="t in outgoingTransitions" :key="`${t.label}__${t.to}`"
               class="flex items-center gap-1.5 px-2 py-1.5 rounded bg-surface-0 border border-border text-[10px]">
            <span class="text-indigo-400 font-mono font-medium truncate flex-1">{{ t.label }}</span>
            <span class="text-text-muted">→</span>
            <span class="text-text-muted font-mono truncate max-w-[80px]">{{ t.to }}</span>
          </div>
        </div>
        <p v-else class="text-[10px] text-text-muted italic">None</p>
      </div>
    </div>

    <!-- Agent config section -->
    <template v-if="form.type !== 'script' && form.type !== 'subprocess' && form.agent">
      <div class="pt-3 border-t border-border space-y-2.5">
        <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">Agent Config</h4>
        <div class="space-y-1">
          <label class="text-xs text-text-muted">Provider</label>
          <select class="input text-xs" v-model="agentForm.provider">
            <option value="">Default (global setting)</option>
            <option value="anthropic">Anthropic</option>
            <option value="openai">OpenAI</option>
            <option value="grok">Grok (xAI)</option>
            <option value="deepseek">Deepseek</option>
            <option value="gemini">Gemini</option>
            <option value="mistral">Mistral AI</option>
            <option value="groq">Groq</option>
            <option value="together">Together AI</option>
            <option value="fireworks">Fireworks AI</option>
            <option value="cohere">Cohere</option>
            <option value="qwen">Qwen (Alibaba)</option>
            <option value="glm">GLM (Zhipu)</option>
            <option value="ollama">Ollama (local)</option>
          </select>
        </div>
        <div class="space-y-1">
          <label class="text-xs text-text-muted">Model</label>
          <input class="input text-xs" v-model="agentForm.model" placeholder="Leave blank to use provider default" />
        </div>
        <div class="space-y-1">
          <label class="text-xs text-text-muted">Task Queue</label>
          <input class="input font-mono text-xs" v-model="agentForm.task_queue" placeholder="phaxa-workers (default)" />
        </div>
        <div class="space-y-1">
          <div class="flex items-center justify-between">
            <label class="text-xs text-text-muted">MCP Servers</label>
            <div class="group relative">
               <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-text-muted cursor-help">
                 <circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/>
               </svg>
               <div class="absolute right-0 bottom-full mb-2 w-48 p-2 rounded bg-surface-2 border border-border text-[9px] text-text-muted leading-relaxed hidden group-hover:block z-50 shadow-2xl">
                 <p class="font-bold mb-1 text-accent uppercase">Port Hints (Local):</p>
                 <ul class="space-y-0.5 font-mono">
                   <li>GLEIF: 8081</li>
                   <li>Frankfurter: 8082</li>
                   <li>OpenFIGI: 8083</li>
                 </ul>
                 <p class="mt-1 opacity-70 italic">Example: http://localhost:8081</p>
               </div>
            </div>
          </div>
          <MCPServerPicker v-model="agentForm.mcp_servers" />
        </div>
      </div>

      <!-- Convert to Script section -->
      <div class="pt-3 border-t border-border space-y-2">
        <div class="flex items-center gap-2">
          <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider flex-1">Convert to Script</h4>
          <button
            :class="['text-xs px-2.5 py-1 rounded font-medium transition-colors', scaffoldLoading ? 'bg-indigo-500/20 text-indigo-400 cursor-not-allowed' : 'bg-indigo-600 hover:bg-indigo-500 text-white']"
            :disabled="scaffoldLoading"
            @click="generateScript"
          >{{ scaffoldLoading ? 'Generating…' : 'Generate Script' }}</button>
        </div>
        <p class="text-[10px] text-text-muted leading-relaxed">Convert this LLM agent into deterministic expr-lang logic for zero-latency execution.</p>
        <p v-if="scaffoldError" class="text-[10px] text-red-400 font-mono">{{ scaffoldError }}</p>
        <template v-if="scaffoldScript">
          <div class="bg-surface-0 border border-border rounded p-2 space-y-2">
            <div class="space-y-0.5">
              <span class="text-[9px] text-text-muted font-bold uppercase">Trigger</span>
              <div class="text-[10px] font-mono text-indigo-400">{{ scaffoldScript.trigger }}</div>
            </div>
            <div v-if="Object.keys(scaffoldScript.updates || {}).length" class="space-y-0.5">
              <span class="text-[9px] text-text-muted font-bold uppercase">Updates</span>
              <div v-for="(v, k) in scaffoldScript.updates" :key="k" class="text-[10px] font-mono text-text-muted flex gap-2">
                <span class="text-text-muted shrink-0">{{ k }}</span>
                <span class="text-text-muted shrink-0">=</span>
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

  <!-- Fullscreen instructions editor pop-out -->
  <CodeEditorModal
    v-if="showInstructionsModal"
    v-model="form.instructions"
    :state-name="form.name + ' Instructions'"
    language="text"
    @close="showInstructionsModal = false"
  />

  <!-- HITL Form Preview Modal -->
  <div v-if="showPreview" class="fixed inset-0 z-[60] flex items-center justify-center p-6 bg-black/60 backdrop-blur-sm" @click.self="showPreview = false">
    <div class="bg-surface-1 border border-border rounded-2xl w-full max-w-lg shadow-2xl flex flex-col overflow-hidden animate-in zoom-in duration-200">
      <header class="px-6 py-4 border-b border-border flex items-center justify-between">
        <h3 class="text-sm font-bold text-text">Form Preview</h3>
        <button @click="showPreview = false" class="text-text-muted hover:text-text transition-colors">✕</button>
      </header>
      <div class="p-8 overflow-y-auto max-h-[70vh]">
        <div class="mb-6 pb-6 border-b border-border/50">
          <p class="text-xs text-text-muted">This is how the form will appear to operators during HITL resolution.</p>
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
import ProcessMappingEditor from './ProcessMappingEditor.vue'
import ProcessPicker from './ProcessPicker.vue'
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

// Helper to convert subprocess mappings object { portName: bbField } → array of { name, bbField }
function mappingsToArray(obj) {
  if (!obj) return []
  return Object.entries(obj).map(([name, bbField]) => ({ name, bbField }))
}

// Supported state types: initial, terminal, prompt, script, code, hitl, wait, subprocess
const STATE_TYPES = ['initial', 'terminal', 'prompt', 'script', 'code', 'hitl', 'wait', 'subprocess']

// ── Reactive State ──────────────────────────────────────────────────────────

const form = reactive({
  name:           props.node.name         ?? '',
  type:           props.node.type         ?? 'prompt',
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
  // subprocess fields
  process_ref:        props.node.subprocess?.process_ref     ?? '',
  process_version:    props.node.subprocess?.process_version ?? '',
  completion_trigger: props.node.subprocess?.completion_trigger || (props.node.type === 'subprocess' ? 'done' : ''),
  failure_trigger:    props.node.subprocess?.failure_trigger    || '',
  input_mappings:     mappingsToArray(props.node.subprocess?.input_mappings),
  output_mappings:    mappingsToArray(props.node.subprocess?.output_mappings),
  form_schema: props.node.form_schema ? JSON.stringify(props.node.form_schema, null, 2) : '',
})

const agentForm = reactive({
  provider:    props.agentDef?.config?.provider    ?? '',
  model:       props.agentDef?.model               ?? '',
  task_queue:  props.agentDef?.task_queue          ?? '',
  prompt:      props.agentDef?.config?.prompt      ?? '',
  mcp_servers: props.agentDef?.config?.mcp_servers ?? '',
})

const availableVersions = ref([])
const subprocessDef    = ref(null)

const subprocessRequiredInputs = computed(() => {
  const s = subprocessDef.value?.blackboard?.schema
  if (!s) return []
  return Object.entries(s).filter(([_, f]) => f.required).map(([k, v]) => ({ name: k, type: v.type }))
})

const subprocessOutputs = computed(() => {
  const s = subprocessDef.value?.blackboard?.schema
  if (!s) return []
  return Object.entries(s).filter(([_, f]) => f.is_output).map(([k, v]) => ({ name: k, type: v.type }))
})

// Fetch versions when a process is selected or on mount if already set
watch(() => form.process_ref, async (name) => {
  if (!name) {
    availableVersions.value = []
    subprocessDef.value = null
    return
  }
  try {
    const { data } = await api.get(`/workflows/${name}/versions`)
    availableVersions.value = data.map(v => v.version_number).sort((a, b) => b - a)
  } catch {}
  fetchSubprocessDef()
}, { immediate: true })

watch(() => form.process_version, () => {
  fetchSubprocessDef()
})

async function fetchSubprocessDef() {
  if (!form.process_ref) return
  try {
    const version = form.process_version || 'latest'
    // Use the versioned API endpoint to get the full definition including blackboard schema
    const { data } = await api.get(`/workflows/${form.process_ref}/v/${version}`)
    subprocessDef.value = data
  } catch (err) {
    console.error("Failed to fetch subprocess def:", err)
  }
}

function onProcessSelected(p) {
  if (!p) return
  subprocessDef.value = p
  form.process_version = '' // Reset to latest on new selection
  
  // Initial mapping suggestion
  if (p.blackboard?.schema) {
    const schema = p.blackboard.schema
    const required = Object.entries(schema).filter(([_, f]) => f.required).map(([k]) => k)
    const outputs  = Object.entries(schema).filter(([_, f]) => f.is_output).map(([k]) => k)

    const existingInput = new Set(form.input_mappings.map(m => m.name))
    required.forEach(name => {
      if (!existingInput.has(name)) form.input_mappings.push({ name, bbField: '' })
    })

    const existingOutput = new Set(form.output_mappings.map(m => m.name))
    outputs.forEach(name => {
      if (!existingOutput.has(name)) form.output_mappings.push({ name, bbField: '' })
    })
  }
  
  // Also fetch the specific version details to be sure we have everything
  fetchSubprocessDef()
}

const scaffoldScript  = ref(null)
const scaffoldLoading = ref(false)
const scaffoldError   = ref('')
const showCodeModal   = ref(false)
const showInstructionsModal = ref(false)

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

  const subprocessDef = form.type === 'subprocess' ? {
    process_ref:        form.process_ref        || '',
    process_version:    form.process_version    || undefined,
    completion_trigger: form.completion_trigger || 'done',
    failure_trigger:    form.failure_trigger    || undefined,
    input_mappings:     mappingsToObject(form.input_mappings),
    output_mappings:    mappingsToObject(form.output_mappings),
  } : undefined

  const nodeData = {
    name:         form.name,
    type:         form.type,
    instructions: form.instructions || undefined,
    agent:        (form.type !== 'script' && form.type !== 'wait' && form.type !== 'code' && form.type !== 'subprocess') ? (form.agent || undefined) : undefined,
    script:       scriptDef,
    code:         codeDef,
    condition:    form.type === 'wait' ? (form.condition || undefined) : undefined,
    timeout:      form.timeout    || undefined,
    on_timeout:   form.on_timeout || undefined,
    subprocess:   subprocessDef,
    form_schema:  form.type === 'hitl' && form.form_schema ? JSON.parse(form.form_schema) : undefined,
  }
  const agentData = (form.type !== 'script' && form.type !== 'code' && form.type !== 'subprocess' && form.agent) ? {
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
