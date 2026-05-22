<template>
  <div class="p-4 space-y-4 flex-1 overflow-y-auto">
    <!-- Header with n8n aesthetics -->
    <div :class="['p-4 rounded-xl border flex items-start gap-3 shadow-sm transition-all duration-300', headerStyles.bg, headerStyles.border]">
      <div :class="['p-2.5 rounded-lg text-white shrink-0 shadow-md', headerStyles.iconBg]">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" v-html="headerStyles.iconSvg"></svg>
      </div>
      <div class="space-y-0.5 flex-1 min-w-0">
        <h3 class="text-sm font-bold text-text truncate">{{ headerStyles.title }}</h3>
        <p class="text-[10px] text-text-muted leading-tight">{{ headerStyles.description }}</p>
      </div>
      <button class="text-text-muted hover:text-red-400 p-1 rounded-lg hover:bg-red-500/10 transition-colors" @click="$emit('delete')" title="Delete State">
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/>
        </svg>
      </button>
    </div>

    <!-- Node Settings Card -->
    <div class="bg-surface-1 border border-border rounded-xl p-4 space-y-3">
      <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">General Configuration</h4>
      
      <div class="space-y-1">
        <label class="text-xs text-text-muted font-medium">State Name</label>
        <input class="input" v-model="form.name" />
      </div>
      
      <div class="space-y-1">
        <label class="text-xs text-text-muted font-medium">State Type</label>
        <select class="input" v-model="form.type">
          <option value="initial">Initial (Start)</option>
          <option value="terminal">Terminal (End)</option>
          <optgroup label="──────────────"></optgroup>
          <option value="prompt">Prompt (LLM Agent)</option>
          <option value="script">Script (Expressions)</option>
          <option value="code">Code (JavaScript)</option>
          <optgroup label="──────────────"></optgroup>
          <option value="subprocess" :disabled="!entStore.hasFeature('subprocesses')">
            Subprocess {{ entStore.hasFeature('subprocesses') ? '(Pro)' : '(Pro - Locked)' }}
          </option>
          <optgroup label="──────────────"></optgroup>
          <option value="telegram_output">Send Telegram message</option>
          <option value="discord_output">Send Discord message</option>
          <optgroup label="──────────────"></optgroup>
          <option value="hitl">Human-in-the-Loop (HITL)</option>
          <option value="wait">Wait Condition</option>
          <option value="timeout_node">Timeout Handler</option>
        </select>
      </div>
    </div>

    <!-- Type-specific parameters -->
    <div class="bg-surface-1 border border-border rounded-xl p-4 space-y-4">
      <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">Parameters</h4>

      <!-- INITIAL -->
      <template v-if="form.type === 'initial'">
        <div class="space-y-3">
          <div class="space-y-1">
            <label class="text-xs text-text-muted">Trigger Type</label>
            <select class="input" v-model="form.triggerType">
              <option value="">No External Trigger (Manual / Run button only)</option>
              <option value="telegram">Telegram Message</option>
              <option value="discord">Discord Event / Message</option>
              <option value="cron">Cron Cadence</option>
              <option value="webhook">Webhook Endpoint</option>
            </select>
          </div>
          <div v-if="form.triggerType" class="space-y-1">
            <label class="text-xs text-text-muted flex items-center justify-between">
              <span>Trigger Config JSON</span>
              <span class="text-[9px] text-text-muted italic">Variables / Token overrides</span>
            </label>
            <CodeEditor
              v-model="form.triggerConfig"
              language="json"
              height="120px"
            />
          </div>
        </div>
      </template>

      <!-- TERMINAL -->
      <template v-if="form.type === 'terminal'">
        <div class="text-center py-6 px-4 rounded-lg bg-surface-0/50 border border-dashed border-border/60">
          <div class="w-12 h-12 rounded-full bg-slate-500/10 flex items-center justify-center mx-auto mb-3">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-slate-400">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
            </svg>
          </div>
          <p class="text-xs text-text-muted font-medium">Terminal Node</p>
          <p class="text-[10px] text-text-muted/70 mt-1 max-w-[200px] mx-auto">
            This state represents a workflow final state. It completes processing and terminates the engine run.
          </p>
        </div>
      </template>

      <!-- PROMPT (Agent reasoning) -->
      <template v-if="form.type === 'prompt'">
        <div class="space-y-3">
          <div class="space-y-1">
            <label class="text-xs text-text-muted">Agent Assigned</label>
            <AgentCombobox
              v-model="form.agent"
              :agents="agents"
              placeholder="Select or enter agent name..."
            />
            <p v-if="!form.agent" class="text-[10px] text-amber-500 font-medium">
              ⚠ Agent required for automated reasoning.
            </p>
          </div>

          <div class="space-y-1">
            <div class="flex items-center justify-between">
              <label class="text-xs text-text-muted">Instructions / Task Prompt</label>
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
            <textarea class="input text-sm leading-relaxed resize-none h-24" v-model="form.instructions" placeholder="Instruct the agent on how to decide transitions..." />
          </div>
        </div>
      </template>

      <!-- SCRIPT -->
      <template v-if="form.type === 'script'">
        <div class="space-y-3">
          <div class="space-y-1">
            <label class="text-xs text-text-muted">Trigger Expression *</label>
            <input class="input font-mono text-xs" v-model="form.script_trigger" placeholder='amount > 1000 ? "needs_review" : "auto_approve"' />
          </div>
          
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <label class="text-xs text-text-muted">Blackboard Updates</label>
              <button class="text-[10px] text-indigo-400 hover:text-indigo-300 font-bold" @click="form.script_updates.push({ key: '', expr: '' })">+ Add Field</button>
            </div>
            
            <div v-if="form.script_updates.length" class="space-y-1.5 max-h-48 overflow-y-auto pr-1">
              <div v-for="(row, i) in form.script_updates" :key="i" class="flex gap-1.5 items-center">
                <input class="input font-mono text-xs flex-1" v-model="row.key" placeholder="field" />
                <span class="text-text-muted text-xs shrink-0">=</span>
                <input class="input font-mono text-xs flex-[2]" v-model="row.expr" placeholder="amount * 0.2" />
                <button class="text-text-muted hover:text-red-400 text-xs shrink-0 px-1" @click="form.script_updates.splice(i, 1)">✕</button>
              </div>
            </div>
            <p v-else class="text-[10px] text-text-muted italic">No blackboard updates configured.</p>
          </div>
        </div>
      </template>

      <!-- CODE (JavaScript Sandbox) -->
      <template v-if="form.type === 'code'">
        <div class="space-y-3">
          <div class="space-y-1">
            <div class="flex items-center justify-between">
              <label class="text-xs text-text-muted">Goal Instructions (AI Codegen Input)</label>
            </div>
            <textarea class="input text-xs leading-relaxed resize-none h-16" v-model="form.instructions" placeholder="Describe the javascript logic needed..." />
            
            <div class="pt-0.5">
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

          <div class="space-y-1.5 pt-2 border-t border-border">
            <div class="flex items-center gap-2">
              <span class="text-xs font-semibold text-teal-400 uppercase tracking-wider flex-1">JS Editor</span>
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
            <CodeEditor
              v-model="form.code"
              :bb-schema="bbSchema"
              height="200px"
            />
          </div>
        </div>
      </template>

      <!-- HITL (Human validation) -->
      <template v-if="form.type === 'hitl'">
        <div class="space-y-3">
          <div class="space-y-1">
            <label class="text-xs text-text-muted">Agent Assigned</label>
            <AgentCombobox
              v-model="form.agent"
              :agents="agents"
            />
          </div>

          <div class="space-y-1">
            <label class="text-xs text-text-muted">Task Description</label>
            <textarea class="input text-xs leading-relaxed resize-none h-16" v-model="form.instructions" placeholder="Instructions for the human operator..." />
          </div>

          <div class="space-y-2 pt-2 border-t border-border">
            <div class="text-xs font-semibold text-amber-500 uppercase tracking-wider">JSON Form Schema</div>
            <CodeEditor
              v-model="form.form_schema"
              language="json"
              height="150px"
              @update:modelValue="validateJson"
            />
            <div v-if="jsonError" class="text-[10px] text-red-400 font-mono mt-1 break-all bg-red-500/5 p-1.5 rounded border border-red-500/10">
              ⚠ {{ jsonError }}
            </div>
            <div class="flex items-center gap-3">
              <button 
                class="text-[10px] text-text-muted hover:text-text flex items-center gap-1 font-bold"
                @click="form.form_schema = JSON.stringify({type: 'object', properties: { field_1: {type: 'string', title: 'Example Field'}}}, null, 2); validateJson()"
              >
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
                </svg>
                Insert Template
              </button>
              <div class="w-px h-3 bg-border"/>
              <button 
                class="text-[10px] text-indigo-400 hover:text-indigo-300 flex items-center gap-1 font-bold disabled:opacity-30 disabled:cursor-not-allowed"
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
        </div>
      </template>

      <!-- WAIT -->
      <template v-if="form.type === 'wait'">
        <div class="space-y-1">
          <label class="text-xs text-text-muted">Wait Condition (expr-lang)</label>
          <input class="input font-mono text-xs" v-model="form.condition" placeholder="e.g. blackboard.count >= 2" />
          <p class="text-[9px] text-text-muted leading-relaxed">
            Execution pauses in a temporal state loop until this expression evaluates to true.
          </p>
        </div>
      </template>

      <!-- SUBPROCESS -->
      <template v-if="form.type === 'subprocess'">
        <div class="space-y-3">
          <!-- License lock check -->
          <div v-if="!entStore.hasFeature('subprocesses')" class="p-3 rounded-lg bg-amber-500/10 border border-amber-500/20 space-y-1">
            <span class="badge bg-amber-500/15 text-amber-500 text-[9px] font-bold tracking-widest uppercase">Pro Feature</span>
            <p class="text-[10px] text-amber-500/80 leading-relaxed font-medium">
              Subprocesses are locked on your current tier.
            </p>
          </div>

          <div :class="{ 'opacity-40 grayscale pointer-events-none select-none': !entStore.hasFeature('subprocesses') }">
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
                <select class="input font-mono text-xs" v-model="form.process_version">
                  <option value="">latest</option>
                  <option v-for="v in availableVersions" :key="v" :value="v">v{{ v }}</option>
                </select>
              </div>
            </div>

            <!-- Subprocess Public schema summary -->
            <div v-if="subprocessDef" class="rounded-lg bg-violet-500/5 border border-violet-500/20 p-2.5 mt-2.5 space-y-2">
              <div v-if="subprocessRequiredInputs.length" class="space-y-0.5">
                <div class="text-[9px] text-text-muted uppercase font-bold">Required Inputs</div>
                <div v-for="p in subprocessRequiredInputs" :key="p.name" class="flex gap-2 text-[10px]">
                  <span class="font-mono text-indigo-300 w-1/2 truncate">{{ p.name }}</span>
                  <span class="text-[9px] text-text-muted italic">{{ p.type }}</span>
                </div>
              </div>
              <div v-if="subprocessOutputs.length" class="space-y-0.5">
                <div class="text-[9px] text-text-muted uppercase font-bold">Outputs</div>
                <div v-for="p in subprocessOutputs" :key="p.name" class="flex gap-2 text-[10px]">
                  <span class="font-mono text-teal-300 w-1/2 truncate">{{ p.name }}</span>
                  <span class="text-[9px] text-text-muted italic">{{ p.type }}</span>
                </div>
              </div>
            </div>

            <div class="space-y-2 pt-2">
              <div class="space-y-1">
                <label class="text-xs text-text-muted">Completion Trigger</label>
                <input class="input font-mono text-xs" v-model="form.completion_trigger" placeholder="done" />
              </div>
              <div class="space-y-1">
                <label class="text-xs text-text-muted">Failure Trigger (optional)</label>
                <input class="input font-mono text-xs" v-model="form.failure_trigger" placeholder="failed" />
              </div>
            </div>

            <div class="pt-2 space-y-2">
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
        </div>
      </template>

      <!-- TELEGRAM OUTPUT -->
      <template v-if="form.type === 'telegram_output'">
        <div class="space-y-3">
          <div class="space-y-1">
            <label class="text-xs text-text-muted">Chat ID / Bind Reference</label>
            <input class="input font-mono text-xs" v-model="form.telegram_chat_id" placeholder="e.g. {{bb.chat_id}} or static id" />
            <div v-if="Object.keys(bbSchema).length" class="flex flex-wrap gap-1 items-center py-1">
              <span class="text-[9px] text-text-muted uppercase font-bold mr-1">Insert:</span>
              <button 
                v-for="k in Object.keys(bbSchema)" 
                :key="k"
                type="button"
                class="badge bg-indigo-500/10 hover:bg-indigo-500/20 text-indigo-400 text-[10px] font-mono border border-indigo-500/20 transition-all active:scale-95 px-1.5 py-0.5 rounded"
                @click="insertBBRef(k, 'telegram_chat_id')"
              >
                {{ k }}
              </button>
            </div>
          </div>
          <div class="space-y-1">
            <label class="text-xs text-text-muted">Message Template</label>
            <textarea class="input text-xs font-mono h-24 leading-relaxed resize-none" v-model="form.telegram_message_text" placeholder="e.g. Alert! User {{bb.name}} has requested review." />
            <div v-if="Object.keys(bbSchema).length" class="flex flex-wrap gap-1 items-center py-1">
              <span class="text-[9px] text-text-muted uppercase font-bold mr-1">Insert:</span>
              <button 
                v-for="k in Object.keys(bbSchema)" 
                :key="k"
                type="button"
                class="badge bg-indigo-500/10 hover:bg-indigo-500/20 text-indigo-400 text-[10px] font-mono border border-indigo-500/20 transition-all active:scale-95 px-1.5 py-0.5 rounded"
                @click="insertBBRef(k, 'telegram_message_text')"
              >
                {{ k }}
              </button>
            </div>
          </div>
        </div>
      </template>

      <!-- DISCORD OUTPUT -->
      <template v-if="form.type === 'discord_output'">
        <div class="space-y-3">
          <div class="space-y-1">
            <label class="text-xs text-text-muted">Channel ID / Bind Reference</label>
            <input class="input font-mono text-xs" v-model="form.discord_channel_id" placeholder="e.g. {{bb.channel_id}} or static id" />
            <div v-if="Object.keys(bbSchema).length" class="flex flex-wrap gap-1 items-center py-1">
              <span class="text-[9px] text-text-muted uppercase font-bold mr-1">Insert:</span>
              <button 
                v-for="k in Object.keys(bbSchema)" 
                :key="k"
                type="button"
                class="badge bg-indigo-500/10 hover:bg-indigo-500/20 text-indigo-400 text-[10px] font-mono border border-indigo-500/20 transition-all active:scale-95 px-1.5 py-0.5 rounded"
                @click="insertBBRef(k, 'discord_channel_id')"
              >
                {{ k }}
              </button>
            </div>
          </div>
          <div class="space-y-1">
            <label class="text-xs text-text-muted">Message Template</label>
            <textarea class="input text-xs font-mono h-24 leading-relaxed resize-none" v-model="form.discord_message_text" placeholder="e.g. Alert! User {{bb.name}} has requested review." />
            <div v-if="Object.keys(bbSchema).length" class="flex flex-wrap gap-1 items-center py-1">
              <span class="text-[9px] text-text-muted uppercase font-bold mr-1">Insert:</span>
              <button 
                v-for="k in Object.keys(bbSchema)" 
                :key="k"
                type="button"
                class="badge bg-indigo-500/10 hover:bg-indigo-500/20 text-indigo-400 text-[10px] font-mono border border-indigo-500/20 transition-all active:scale-95 px-1.5 py-0.5 rounded"
                @click="insertBBRef(k, 'discord_message_text')"
              >
                {{ k }}
              </button>
            </div>
          </div>
        </div>
      </template>

      <!-- TIMEOUT NODE -->
      <template v-if="form.type === 'timeout_node'">
        <div class="space-y-3">
          <div class="space-y-1">
            <label class="text-xs text-text-muted font-semibold">Default Timeout Duration</label>
            <input class="input font-mono text-sm border-red-500/20 focus:border-red-500/50" v-model="form.default_timeout" placeholder="e.g. 10s, 5m, 1h" />
            <p class="text-[10px] text-text-muted leading-relaxed mt-1">
              Changing this value automatically updates the default timeout duration on all eligible canvas nodes and ensures they connect to this Timeout Node.
            </p>
          </div>
        </div>
      </template>
    </div>

    <!-- Agent config section overrides (only for prompts and hitl with agents) -->
    <template v-if="['prompt', 'hitl'].includes(form.type) && form.agent">
      <div class="bg-surface-1 border border-border rounded-xl p-4 space-y-3">
        <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">Agent Parameter Overrides</h4>
        <div class="space-y-1">
          <label class="text-xs text-text-muted">LLM Provider</label>
          <select class="input text-xs" v-model="agentForm.provider">
            <option value="">Default (Global Settings)</option>
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
            <option value="ollama">Ollama (Local / Stdio)</option>
          </select>
        </div>
        
        <div class="space-y-1">
          <label class="text-xs text-text-muted">Model Identifier</label>
          <input class="input text-xs" v-model="agentForm.model" placeholder="Leave empty for global defaults..." />
        </div>

        <div class="space-y-1">
          <label class="text-xs text-text-muted">Task Queue</label>
          <input class="input font-mono text-xs" v-model="agentForm.task_queue" placeholder="chainnodes-workers" />
        </div>

        <div class="space-y-1">
          <div class="flex items-center justify-between">
            <label class="text-xs text-text-muted">Active MCP Tools</label>
            <div class="group relative">
               <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-text-muted cursor-help">
                 <circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/>
               </svg>
               <div class="absolute right-0 bottom-full mb-2 w-48 p-2 rounded bg-surface-2 border border-border text-[9px] text-text-muted leading-relaxed hidden group-hover:block z-50 shadow-2xl">
                 <p class="font-bold mb-1 text-accent uppercase">MCP Stdio Servers:</p>
                 <p class="opacity-70 italic">Add MCP server names and arguments to enable tools.</p>
               </div>
            </div>
          </div>
          <MCPServerPicker v-model="agentForm.mcp_servers" />
        </div>

        <!-- Convert to Script block -->
        <div class="pt-3 border-t border-border space-y-2">
          <div class="flex items-center gap-2">
            <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider flex-1">Scaffold Script</h4>
            <button
              :class="['text-xs px-2.5 py-1 rounded font-medium transition-colors', scaffoldLoading ? 'bg-indigo-500/20 text-indigo-400 cursor-not-allowed' : 'bg-indigo-600 hover:bg-indigo-500 text-white']"
              :disabled="scaffoldLoading"
              @click="generateScript"
            >{{ scaffoldLoading ? 'Generating…' : 'Convert' }}</button>
          </div>
          <p class="text-[10px] text-text-muted leading-relaxed">Scaffold expression logic based on LLM output history to bypass reasoning latency.</p>
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
              <button class="btn-primary text-[10px] flex-1 py-1.5" @click="applyScript">Apply Script</button>
              <button class="btn-ghost text-[10px] px-2 py-1.5" @click="copyScript">Copy</button>
            </div>
          </template>
        </div>
      </div>
    </template>

    <!-- Timeout & standard transition parameters (hidden for terminal/timeout nodes) -->
    <template v-if="form.type !== 'terminal' && form.type !== 'timeout_node'">
      <div class="bg-surface-1 border border-border rounded-xl p-4 space-y-3">
        <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">Timeout Boundaries</h4>
        <div class="space-y-1">
          <label class="text-xs text-text-muted">Timeout Limit</label>
          <input class="input" v-model="form.timeout" placeholder="e.g. 30s, 5m, 2h" />
        </div>
        <div v-if="form.timeout" class="space-y-1">
          <label class="text-xs text-text-muted">On Timeout Transition</label>
          <input class="input font-mono" v-model="form.on_timeout" placeholder="timeout" />
        </div>
      </div>
    </template>

    <!-- Transitions List Card -->
    <div class="bg-surface-1 border border-border rounded-xl p-4 space-y-3">
      <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">Transitions Map</h4>
      <div class="space-y-1.5">
        <div class="text-[10px] text-text-muted font-medium uppercase tracking-tight">Incoming</div>
        <div v-if="incomingTransitions.length" class="space-y-1">
          <div v-for="t in incomingTransitions" :key="`${t.from}__${t.label}`"
               class="flex items-center gap-1.5 px-2 py-1 rounded bg-surface-0 border border-border text-[10px]">
            <span class="text-text-muted font-mono truncate max-w-[80px]">{{ t.from }}</span>
            <span class="text-text-muted">→</span>
            <span class="text-indigo-400 font-mono font-medium truncate flex-1">{{ t.label || '(default)' }}</span>
          </div>
        </div>
        <p v-else class="text-[10px] text-text-muted italic">No incoming links</p>
      </div>
      <div class="space-y-1.5 pt-1.5 border-t border-border/40">
        <div class="text-[10px] text-text-muted font-medium uppercase tracking-tight">Outgoing</div>
        <div v-if="outgoingTransitions.length" class="space-y-1">
          <div v-for="t in outgoingTransitions" :key="`${t.label}__${t.to}`"
               class="flex items-center gap-1.5 px-2 py-1 rounded bg-surface-0 border border-border text-[10px]">
            <span class="text-indigo-400 font-mono font-medium truncate flex-1">{{ t.label || '(default)' }}</span>
            <span class="text-text-muted">→</span>
            <span class="text-text-muted font-mono truncate max-w-[80px]">{{ t.to }}</span>
          </div>
        </div>
        <p v-else class="text-[10px] text-text-muted italic">No outgoing links</p>
      </div>
    </div>

    <!-- Apply changes -->
    <button class="btn-primary w-full shadow-md py-2.5 font-bold tracking-wide" @click="onApply">Apply changes</button>
  </div>

  <!-- Fullscreen editors & previews -->
  <CodeEditorModal
    v-if="showCodeModal"
    v-model="form.code"
    :state-name="form.name"
    :bb-schema="bbSchema"
    @close="showCodeModal = false"
  />

  <CodeEditorModal
    v-if="showInstructionsModal"
    v-model="form.instructions"
    :state-name="form.name + ' Instructions'"
    language="text"
    @close="showInstructionsModal = false"
  />

  <!-- Preview HITL form modal -->
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
import { useEnterpriseStore } from '@/stores/enterprise'
import AgentCombobox from './AgentCombobox.vue'
import CodeEditor from './CodeEditor.vue'
import CodeEditorModal from './CodeEditorModal.vue'
import MCPServerPicker from './MCPServerPicker.vue'
import ProcessMappingEditor from './ProcessMappingEditor.vue'
import ProcessPicker from './ProcessPicker.vue'
import JsonSchemaForm from '@/components/monitor/JsonSchemaForm.vue'

const props = defineProps({
  node:     Object,
  edges:    { type: Array,  default: () => [] },
  agentDef: { type: Object, default: null },
  bbSchema: { type: Object, default: () => ({}) },
  agents:   { type: Array,  default: () => [] },
})
const emit = defineEmits(['update', 'delete'])
const entStore = useEnterpriseStore()

// Codegen & script setup refs
const codegenLoading = ref(false)
const codegenError   = ref('')
const scaffoldScript  = ref(null)
const scaffoldLoading = ref(false)
const scaffoldError   = ref('')
const showCodeModal   = ref(false)
const showInstructionsModal = ref(false)
const codegenExplanation = ref('')
const jsonError = ref('')
const showPreview = ref(false)

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

// Helper to convert subprocess mappings object { portName: bbField } → array of { name, bbField }
function mappingsToArray(obj) {
  if (!obj) return []
  return Object.entries(obj).map(([name, bbField]) => ({ name, bbField }))
}

// Form parameters reactive setup
const form = reactive({
  name:           props.node.name         ?? '',
  type:           props.node.type         ?? 'prompt',
  instructions:   props.node.instructions ?? '',
  agent:          props.node.agent        ?? '',
  condition:      props.node.condition    ?? '',
  timeout:        props.node.timeout      ?? '',
  on_timeout:     props.node.on_timeout   ?? '',
  default_timeout: props.node.default_timeout ?? '10s',
  is_timeout_node: props.node.is_timeout_node ?? false,
  script_trigger: props.node.script?.trigger ?? '',
  script_updates: props.node.script?.updates
    ? Object.entries(props.node.script.updates).map(([k, v]) => ({ key: k, expr: v }))
    : [],
  code: props.node.code?.code ?? defaultCodeTemplate,
  process_ref:        props.node.subprocess?.process_ref     ?? '',
  process_version:    props.node.subprocess?.process_version ?? '',
  completion_trigger: props.node.subprocess?.completion_trigger || (props.node.type === 'subprocess' ? 'done' : ''),
  failure_trigger:    props.node.subprocess?.failure_trigger    || '',
  input_mappings:     mappingsToArray(props.node.subprocess?.input_mappings),
  output_mappings:    mappingsToArray(props.node.subprocess?.output_mappings),
  form_schema: props.node.form_schema ? JSON.stringify(props.node.form_schema, null, 2) : '',
  telegram_chat_id: props.node.telegram_output?.chat_id ?? '{{bb.chat_id}}',
  telegram_message_text: props.node.telegram_output?.message_text ?? '',
  discord_channel_id: props.node.discord_output?.channel_id ?? '{{bb.channel_id}}',
  discord_message_text: props.node.discord_output?.message_text ?? '',
  triggerType: props.node.triggerType ?? '',
  triggerConfig: props.node.triggerConfig ? JSON.stringify(props.node.triggerConfig, null, 2) : '{}',
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

// Header style configuration depending on node type
const headerStyles = computed(() => {
  const t = form.type
  if (t === 'initial') {
    return {
      title: 'Start / Trigger Node',
      description: 'Defines the entrypoint and start triggers for workflow execution.',
      bg: 'bg-emerald-500/5',
      border: 'border-emerald-500/20',
      iconBg: 'bg-emerald-500',
      iconSvg: '<polygon points="5 3 19 12 5 21 5 3"/>'
    }
  } else if (t === 'prompt') {
    return {
      title: 'Agent Prompt Node',
      description: 'Instructs an LLM agent to analyze blackboard data and determine the next step.',
      bg: 'bg-indigo-500/5',
      border: 'border-indigo-500/20',
      iconBg: 'bg-indigo-500',
      iconSvg: '<circle cx="12" cy="12" r="10"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/><line x1="9" y1="9" x2="9.01" y2="9"/><line x1="15" y1="9" x2="15.01" y2="9"/>'
    }
  } else if (t === 'script') {
    return {
      title: 'Script Node',
      description: 'Evaluates expr-lang expressions and updates the blackboard deterministically.',
      bg: 'bg-amber-500/5',
      border: 'border-amber-500/20',
      iconBg: 'bg-amber-500',
      iconSvg: '<polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/>'
    }
  } else if (t === 'code') {
    return {
      title: 'JavaScript Code Sandbox',
      description: 'Executes sandboxed JavaScript to parse API payloads, mutate data, or branch logic.',
      bg: 'bg-teal-500/5',
      border: 'border-teal-500/20',
      iconBg: 'bg-teal-500',
      iconSvg: '<polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>'
    }
  } else if (t === 'hitl') {
    return {
      title: 'Human-in-the-Loop Node',
      description: 'Pauses execution to present operators with a task form and request approval.',
      bg: 'bg-amber-500/5',
      border: 'border-amber-500/20',
      iconBg: 'bg-amber-500',
      iconSvg: '<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>'
    }
  } else if (t === 'wait') {
    return {
      title: 'Wait State',
      description: 'Pauses workflow execution until a specific event or condition is met.',
      bg: 'bg-slate-500/5',
      border: 'border-slate-500/20',
      iconBg: 'bg-slate-500',
      iconSvg: '<circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>'
    }
  } else if (t === 'subprocess') {
    return {
      title: 'Sub-Process State',
      description: 'Modular sub-workflow caller. Starts a child workflow run.',
      bg: 'bg-violet-500/5',
      border: 'border-violet-500/20',
      iconBg: 'bg-violet-500',
      iconSvg: '<rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><line x1="9" y1="3" x2="9" y2="21"/>'
    }
  } else if (t === 'telegram_output') {
    return {
      title: 'Telegram Action',
      description: 'Sends a Telegram message to a specified Chat ID using a message template.',
      bg: 'bg-blue-500/5',
      border: 'border-blue-500/20',
      iconBg: 'bg-blue-500',
      iconSvg: '<path d="M22 2L11 13M22 2l-7 20-4-9-9-4Z"/>'
    }
  } else if (t === 'discord_output') {
    return {
      title: 'Discord Action',
      description: 'Publishes a message to a Discord channel via API integrations.',
      bg: 'bg-indigo-500/5',
      border: 'border-indigo-500/20',
      iconBg: 'bg-indigo-600',
      iconSvg: '<path d="M18 8a3 3 0 0 0-3-3H5a3 3 0 0 0-3 3v8a3 3 0 0 0 3 3h10a3 3 0 0 0 3-3V8Z"/><path d="M22 10v4"/>'
    }
  } else if (t === 'timeout_node') {
    return {
      title: 'Timeout Handler',
      description: 'Acts as a central error boundary when any step fails to complete within limits.',
      bg: 'bg-red-500/5',
      border: 'border-red-500/20',
      iconBg: 'bg-red-500',
      iconSvg: '<circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>'
    }
  } else {
    return {
      title: 'Terminal State',
      description: 'Terminates the workflow execution and saves final output blackboard state.',
      bg: 'bg-slate-500/5',
      border: 'border-slate-500/20',
      iconBg: 'bg-slate-600',
      iconSvg: '<rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>'
    }
  }
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
    const { data } = await api.get(`/workflows/${form.process_ref}/v/${version}`)
    subprocessDef.value = data
  } catch (err) {
    console.error("Failed to fetch subprocess def:", err)
  }
}

function onProcessSelected(p) {
  if (!p) return
  subprocessDef.value = p
  form.process_version = ''
  
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
  fetchSubprocessDef()
}

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

function mappingsToObject(arr) {
  if (!arr || !arr.length) return undefined
  const filtered = arr.filter(m => m.name)
  if (!filtered.length) return undefined
  return Object.fromEntries(filtered.map(m => [m.name, m.bbField]))
}

function insertBBRef(key, field) {
  form[field] = (form[field] || '') + `{{bb.${key}}}`
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
  if (form.type === 'initial' && form.triggerType && form.triggerConfig) {
    try {
      JSON.parse(form.triggerConfig)
    } catch (e) {
      alert('Cannot apply changes: Invalid Trigger Configuration JSON.\n\n' + e.message)
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

  const telegramOutputDef = form.type === 'telegram_output' ? {
    chat_id:      form.telegram_chat_id || '',
    message_text: form.telegram_message_text || '',
  } : undefined

  const discordOutputDef = form.type === 'discord_output' ? {
    channel_id:   form.discord_channel_id || '',
    message_text: form.discord_message_text || '',
  } : undefined

  const nodeData = {
    name:            form.name,
    type:            form.type,
    instructions:    form.instructions || undefined,
    agent:           (!['script', 'wait', 'code', 'subprocess', 'telegram_output', 'discord_output', 'timeout_node'].includes(form.type)) ? (form.agent || undefined) : undefined,
    script:          scriptDef,
    code:            codeDef,
    condition:       form.type === 'wait' ? (form.condition || undefined) : undefined,
    timeout:         (form.type !== 'terminal' && form.type !== 'timeout_node') ? (form.timeout || undefined) : undefined,
    on_timeout:      (form.type !== 'terminal' && form.type !== 'timeout_node') ? (form.on_timeout || undefined) : undefined,
    subprocess:      subprocessDef,
    form_schema:     form.type === 'hitl' && form.form_schema ? JSON.parse(form.form_schema) : undefined,
    telegram_output: telegramOutputDef,
    discord_output:  discordOutputDef,
    triggerType:     form.type === 'initial' ? (form.triggerType || undefined) : undefined,
    triggerConfig:   (form.type === 'initial' && form.triggerType && form.triggerConfig) ? JSON.parse(form.triggerConfig) : undefined,
    is_timeout_node: form.type === 'timeout_node' ? true : undefined,
    default_timeout: form.type === 'timeout_node' ? (form.default_timeout || '10s') : undefined,
  }
  
  const agentData = (!['script', 'code', 'subprocess', 'telegram_output', 'discord_output', 'timeout_node'].includes(form.type) && form.agent) ? {
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
