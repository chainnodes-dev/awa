<template>
  <div class="flex flex-col h-full overflow-hidden">

    <!-- ── Top bar ──────────────────────────────────────────────────────── -->
    <header class="flex items-center gap-3 px-4 py-2.5 border-b border-border shrink-0 bg-surface-1">
      <RouterLink to="/dashboard" class="btn-ghost p-1.5">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/>
        </svg>
      </RouterLink>
      <div class="w-px h-4 bg-border"/>

      <span class="text-sm font-medium text-text">{{ run?.workflow_name ?? 'Monitor' }}</span>
      <span v-if="run?.id" class="text-xs font-mono text-text-muted">{{ run.id.slice(0, 8) }}</span>
      <span v-if="run" :class="`status-${run.status} ml-1`">{{ run.status }}</span>

      <div class="flex-1"/>

      <div class="flex items-center gap-2 mr-4">
        <button @click="onEditClick" class="btn-ghost text-xs gap-1">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
          </svg>
          Edit
        </button>
        <button @click="showRunModal = true" class="btn-ghost text-xs gap-1">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/>
          </svg>
          Re-run
        </button>
      </div>

      <!-- HITL controls (only for hitl-type states, not wait/join nodes) -->
      <template v-if="currentStateType === 'hitl' && !hitlResolved">
        <button @click="resolve('approved')" class="btn bg-green-500/20 text-green-400 hover:bg-green-500/30 text-xs">✓ Approve</button>
        <button @click="resolve('rejected')" class="btn bg-red-500/20 text-red-400 hover:bg-red-500/30 text-xs">✕ Reject</button>
      </template>
      <!-- Wait/join node indicator -->
      <span v-else-if="run?.status === 'waiting' && currentStateType === 'wait'"
            class="text-xs text-indigo-400/80 font-mono px-2 py-1 rounded bg-indigo-500/10 border border-indigo-500/20">
        ⏳ awaiting condition…
      </span>

      <!-- Manual trigger -->
      <button v-if="run?.status === 'running'" @click="showTriggerModal = true" class="btn-ghost text-xs">
        Fire Trigger
      </button>

      <!-- Termination -->
      <button v-if="canTerminate" @click="stopRun" class="btn text-xs bg-red-500/10 text-red-500 hover:bg-red-500/20 border border-red-500/20">
        <svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor" class="mr-1">
          <rect x="6" y="6" width="12" height="12"/>
        </svg>
        Stop
      </button>

      <button @click="loadRun" class="btn-ghost p-1.5" title="Refresh">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="23 4 23 10 17 10"/>
          <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
        </svg>
      </button>
    </header>

    <!-- Failure reason banner -->
    <div v-if="run?.status === 'failed' && run?.failure_reason"
         class="mx-4 mt-3 mb-1 px-4 py-4 rounded-xl bg-red-500/10 border border-red-500/20 text-red-400 shrink-0 flex items-start justify-between gap-4">
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-2 mb-2">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
          <span class="text-xs font-bold uppercase tracking-wider">Run Failed</span>
        </div>
        <div class="text-[11px] font-mono whitespace-pre-wrap break-all leading-relaxed opacity-90 max-h-48 overflow-y-auto pr-4 scrollbar-thin scrollbar-thumb-red-500/30">
          {{ run.failure_reason }}
        </div>
      </div>
      <button 
        v-if="workflowYAML"
        @click="showDebugModal = true" 
        class="btn bg-violet-600 hover:bg-violet-500 text-white text-[10px] px-3 py-1.5 shrink-0 shadow-lg shadow-violet-600/20 gap-1.5 uppercase font-bold tracking-wider"
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <path d="M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z"/>
          <path d="M12 8v4"/><path d="M12 16h.01"/>
        </svg>
        Debug with AI
      </button>
    </div>

    <!-- AI Debug Modal -->
    <AIDebugModal
      v-if="showDebugModal && run"
      :run-id="run.id"
      :original-YAML="workflowYAML"
      :failed-node="run.current_state"
      :error-message="run.failure_reason"
      :blackboard="run.blackboard"
      @close="showDebugModal = false"
    />

    <!-- HITL resolve error banner -->
    <div v-if="resolveError"
         class="mx-4 mt-3 mb-1 px-4 py-3 rounded-lg bg-orange-500/10 border border-orange-500/30 text-orange-400 text-xs font-mono shrink-0 flex items-start gap-2">
      <span class="font-semibold shrink-0">HITL signal failed:</span>
      <span class="break-all">{{ resolveError }}</span>
      <button @click="resolveError = ''" class="ml-auto shrink-0 opacity-60 hover:opacity-100">✕</button>
    </div>

    <div class="flex flex-1 overflow-hidden">

      <!-- ── Canvas + thinking + event log ────────────────────────────── -->
      <div class="flex-1 flex flex-col overflow-hidden">

        <!-- State machine canvas (read-only) -->
        <div class="relative flex-1 overflow-hidden">
          <VueFlow
            v-if="flowNodes.length"
            :nodes="flowNodes"
            :edges="flowEdges"
            :node-types="nodeTypes"
            :nodes-draggable="false"
            :nodes-connectable="false"
            :elements-selectable="false"
            fit-view-on-init
            class="bg-surface-0"
          >
            <Background pattern-color="#2a2d3a" :gap="24" :size="1" />
            <Controls />

            <!-- High-quality HTML Edge Labels -->
            <template #edge-label="{ labelX, labelY, data }">
              <EdgeLabelRenderer>
                <div 
                  v-if="data?.trigger"
                  :style="{
                    position: 'absolute',
                    transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`,
                    pointerEvents: 'all'
                  }"
                  class="px-2 py-1 rounded border border-border bg-surface-2/80 text-[9px] font-mono font-medium text-text shadow-xl backdrop-blur-sm whitespace-nowrap text-center leading-tight"
                >
                  <div class="flex items-center justify-center gap-1">
                    <span v-if="data.isTimeout">⏱</span>
                    <span>{{ data.trigger }}</span>
                  </div>
                  <div v-if="data.guard" class="text-[8px] text-text-muted opacity-90 mt-0.5">
                    {{ data.guard }}
                  </div>
                </div>
              </EdgeLabelRenderer>
            </template>
          </VueFlow>
          <div v-else class="h-full flex items-center justify-center text-text-muted text-sm">
            Loading workflow…
          </div>

          <!-- Time Travel Scrubber -->
          <div v-if="history.length" class="absolute bottom-6 left-1/2 -translate-x-1/2 z-20 flex items-center gap-4 px-5 py-3 rounded-2xl bg-surface-1/80 border border-border/50 backdrop-blur-xl shadow-2xl min-w-[400px]">
            <button 
              @click="isLive = !isLive"
              :class="[
                'flex items-center gap-2 px-3 py-1.5 rounded-lg text-[10px] font-bold uppercase tracking-wider transition-all border',
                isLive ? 'bg-green-500/10 text-green-500 border-green-500/20' : 'bg-surface-2 text-text-muted border-border hover:text-text'
              ]"
            >
              <div v-if="isLive" class="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse"/>
              {{ isLive ? 'Live' : 'Travel' }}
            </button>

            <div class="flex-1 flex flex-col gap-1.5">
              <div class="flex justify-between text-[9px] font-mono text-text-muted uppercase tracking-tighter">
                <span>Start</span>
                <span class="text-accent">{{ isLive ? 'Current' : 'Step ' + (scrubIndex + 1) }}</span>
                <span>End</span>
              </div>
              <div class="flex items-center gap-2">
                <button 
                  @click="stepBack" 
                  :disabled="isLive || scrubIndex <= 0"
                  class="p-1 rounded bg-surface-2 border border-border text-text-muted hover:text-text disabled:opacity-30 disabled:hover:text-text-muted cursor-pointer disabled:cursor-not-allowed transition-all"
                  title="Step Back"
                >
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <polyline points="15 18 9 12 15 6"/>
                  </svg>
                </button>
                
                <input 
                  type="range" 
                  min="0" 
                  :max="history.length" 
                  v-model.number="scrubIndex"
                  @input="isLive = false"
                  class="flex-1 h-1.5 bg-surface-2 rounded-lg appearance-none cursor-pointer accent-accent"
                />

                <button 
                  @click="stepForward" 
                  :disabled="isLive || scrubIndex >= history.length"
                  class="p-1 rounded bg-surface-2 border border-border text-text-muted hover:text-text disabled:opacity-30 disabled:hover:text-text-muted cursor-pointer disabled:cursor-not-allowed transition-all"
                  title="Step Forward"
                >
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <polyline points="9 18 15 12 9 6"/>
                  </svg>
                </button>
              </div>
            </div>

            <div class="text-[10px] font-mono text-text-muted bg-surface-2 px-2 py-1 rounded-md border border-border">
              {{ formatDuration(scrubDuration) }}
            </div>
          </div>
        </div>


        <!-- Bottom Log/Debug Tabs -->
        <div class="h-64 border-t border-border flex flex-col overflow-hidden bg-surface-1">
          <div class="flex border-b border-border bg-surface-2 shrink-0">
            <button
              v-for="tab in bottomTabs" :key="tab.id"
              @click="bottomTab = tab.id"
              :class="[
                'px-4 py-2 text-[10px] font-bold uppercase tracking-wider transition-colors border-r border-border',
                bottomTab === tab.id ? 'text-accent bg-surface-1' : 'text-text-muted hover:text-text bg-surface-2',
              ]"
            >
              {{ tab.label }}
              <span
                v-if="tab.count"
                class="text-[9px] bg-accent/20 text-accent px-1 rounded-full ml-1"
              >{{ tab.count }}</span>
            </button>
          </div>
          <div class="flex-1 overflow-hidden">
            <EventLog v-if="bottomTab === 'events'" :events="filteredEvents" />
            <LLMDebugPanel
              v-else-if="bottomTab === 'llm'"
              :calls="displayLLMCalls"
              class="h-full"
            />
            <MCPAuditLogPanel
              v-else-if="bottomTab === 'mcp'"
              :logs="runMCPLogs"
              class="h-full"
            />
          </div>
        </div>
      </div>

      <!-- ── Right panel ───────────────────────────────────────────────── -->
      <aside class="w-72 border-l border-border flex flex-col overflow-hidden bg-surface-1">
        <div class="flex border-b border-border">
          <button
            v-for="tab in visibleTabs" :key="tab.id"
            @click="activeTab = tab.id"
            :class="[
              'flex-1 py-2 text-xs font-medium transition-colors capitalize',
              activeTab === tab.id ? 'text-text border-b-2 border-accent' : 'text-text-muted hover:text-text',
            ]"
          >
            <span v-if="tab.id === 'debug'" class="flex items-center justify-center gap-1">
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="shrink-0">
                <path d="M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z"/>
                <line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
              </svg>
              LLM Debug
              <span
                v-if="runLLMCalls.length"
                class="text-[9px] bg-accent/20 text-accent px-1 rounded-full"
              >{{ runLLMCalls.length }}</span>
            </span>
            <span v-else-if="tab.id === 'mcp'" class="flex items-center justify-center gap-1">
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="shrink-0">
                <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
              </svg>
              MCP Debug
              <span
                v-if="runMCPLogs.length"
                class="text-[9px] bg-accent/20 text-accent px-1 rounded-full"
              >{{ runMCPLogs.length }}</span>
            </span>
            <span v-else-if="tab.id === 'task'" class="flex items-center justify-center gap-1">
              <span class="w-2 h-2 rounded-full bg-amber-500 animate-pulse"/>
              Task
            </span>
            <span v-else-if="tab.id === 'snapshot'" class="flex items-center justify-center gap-1">
               <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="shrink-0">
                  <circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/>
               </svg>
               Snapshot
            </span>
            <span v-else>{{ tab.label }}</span>
          </button>
        </div>

        <!-- Blackboard tab -->
        <div v-if="activeTab === 'blackboard'" class="flex-1 flex flex-col overflow-hidden">
          <BlackboardPanel
            :blackboard="displayBlackboard"
            :previous-blackboard="displayPreviousBlackboard"
            class="flex-1 overflow-hidden"
          />

          <!-- Thought Trace (Live) -->
          <div v-if="displayReasoning" class="border-t border-border p-4 bg-surface-2 shrink-0 max-h-[40%] overflow-y-auto shadow-inner">
            <div class="flex items-center gap-2 mb-2">
              <div v-if="isThinking" class="w-1.5 h-1.5 rounded-full bg-violet-400 animate-pulse"/>
              <h4 class="text-[10px] font-bold text-text-muted uppercase tracking-widest">Thought Trace</h4>
            </div>
            <p class="text-xs text-text leading-relaxed whitespace-pre-wrap font-sans">{{ displayReasoning }}</p>
          </div>
        </div>

        <!-- Thoughts tab -->
        <div v-else-if="activeTab === 'thoughts'" class="flex-1 flex flex-col overflow-hidden p-4 space-y-4">
           <div v-if="displayReasoning" class="space-y-2">
             <div class="flex items-center gap-2">
                <div v-if="isThinking" class="w-2 h-2 rounded-full bg-violet-500 animate-pulse"/>
                <h3 class="text-xs font-bold text-text uppercase tracking-wider">Active Monologue</h3>
             </div>
             <div class="p-4 rounded-xl bg-violet-500/5 border border-violet-500/10 text-sm text-text leading-relaxed whitespace-pre-wrap italic opacity-90 shadow-inner">
               "{{ displayReasoning }}"
             </div>
           </div>
           
           <div class="space-y-3">
             <h3 class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border pb-1">Historical Context</h3>
             <div class="space-y-4">
               <div v-for="(t, idx) in displayHistory.slice().reverse()" :key="idx" class="relative pl-4 border-l border-border/50">
                 <div v-if="t.agent_output?.reasoning" class="space-y-1">
                   <div class="text-[10px] font-mono text-accent">{{ t.to_state }}</div>
                   <p class="text-xs text-text-muted leading-snug">{{ t.agent_output.reasoning }}</p>
                 </div>
               </div>
             </div>
           </div>
        </div>

        <!-- Snapshot tab (Historical) -->
        <div v-else-if="activeTab === 'snapshot' && selectedTransition" class="flex-1 flex flex-col overflow-hidden">
           <div class="px-4 py-2 border-b border-border bg-indigo-500/5 flex items-center justify-between">
              <span class="text-[10px] font-bold text-indigo-400 uppercase tracking-widest">Historical State</span>
              <button @click="selectedTransition = null; activeTab = 'blackboard'" class="text-[9px] text-text-muted hover:text-text">Close</button>
           </div>
           <div v-if="selectedTransition.agent_output?.reasoning" class="p-3 border-b border-border bg-surface-2">
              <h4 class="text-[10px] font-bold text-text-muted uppercase tracking-widest mb-1.5">Agent Reasoning</h4>
              <p class="text-[11px] font-mono text-text-muted whitespace-pre-wrap leading-relaxed">{{ selectedTransition.agent_output.reasoning }}</p>
           </div>
           <BlackboardPanel
             :blackboard="selectedTransition.blackboard_snapshot"
             class="flex-1 overflow-hidden"
           />
        </div>


        <!-- Task tab (HITL) -->
        <div v-else-if="activeTab === 'task'" class="flex-1 flex flex-col overflow-hidden bg-surface-2/30">
          <div class="p-4 space-y-4 flex-1 overflow-y-auto">
            <div class="space-y-1 mb-4">
              <h3 class="text-xs font-semibold text-text uppercase tracking-wider">Input Required</h3>
              <p class="text-xs text-text-muted">{{ currentStateDef?.instructions || 'A human must review this step to continue.' }}</p>
            </div>

            <JsonSchemaForm
              v-if="currentFormSchema"
              ref="hitlForm"
              :schema="currentFormSchema"
              :initial-data="run?.blackboard"
            />
            <div v-else class="py-4 text-center">
              <p class="text-xs text-text-muted">No additional data required. Please approve or reject above.</p>
            </div>

            <div class="mt-8 pt-4 border-t border-border flex-1 flex flex-col min-h-0">
              <div class="flex items-center justify-between mb-3">
                <h4 class="text-xs font-bold text-text-muted uppercase tracking-widest">Advanced Tweaks</h4>
                <button 
                  @click="showBlackboardTweak = !showBlackboardTweak" 
                  class="text-[11px] text-indigo-400 hover:text-indigo-300 font-medium"
                >
                  {{ showBlackboardTweak ? 'Hide Context' : 'Edit Context' }}
                </button>
              </div>

              <div v-if="showBlackboardTweak && blackboardSchema" class="flex p-1 bg-surface-0 border border-slate-800/40 rounded-lg w-fit mb-3">
                <button
                  @click="tweakView = 'form'"
                  :class="[
                    'px-2 py-1 text-[9px] font-bold uppercase rounded transition-all',
                    tweakView === 'form' ? 'bg-accent/10 text-accent' : 'text-text-muted hover:text-text-muted'
                  ]"
                > Form </button>
                <button
                  @click="tweakView = 'json'"
                  :class="[
                    'px-2 py-1 text-[9px] font-bold uppercase rounded transition-all',
                    tweakView === 'json' ? 'bg-accent/10 text-accent' : 'text-text-muted hover:text-text-muted'
                  ]"
                > JSON </button>
              </div>
              <div v-if="showBlackboardTweak" class="space-y-4 flex-1 flex flex-col min-h-0">
                <div v-if="tweakView === 'form' && blackboardSchema" class="flex-1 overflow-y-auto pr-2 scrollbar-thin scrollbar-thumb-slate-800">
                  <JsonSchemaForm
                    :schema="blackboardSchema"
                    :initial-data="editedBlackboard"
                    @update:modelValue="syncFromTweakForm"
                  />
                </div>
                <BlackboardEditor 
                  v-else
                  v-model="editedBlackboard"
                  class="flex-1"
                  @update:modelValue="syncFromTweakJson"
                />
              </div>
            </div>
          </div>
          
          <div class="p-4 border-t border-border bg-surface-1/50 flex gap-3">
            <button @click="resolve('rejected')" class="flex-1 btn-danger-solid justify-center py-2.5">Reject</button>
            <button @click="resolve('approved')" class="flex-1 btn-primary justify-center py-2.5">Approve</button>
          </div>
        </div>

        <!-- Timeline tab -->
        <div v-else class="flex-1 overflow-y-auto p-3">
          <div v-if="!history.length" class="text-text-muted text-xs text-center py-8">
            No transitions yet
          </div>

          <div v-else class="relative">
            <!-- Vertical guide line -->
            <div class="absolute left-[11px] top-4 bottom-0 w-px bg-border"/>

            <div
              v-for="(t, i) in history"
              :key="i"
              class="group relative pl-8 pb-4 last:pb-2 cursor-pointer"
              @click="inspectTransition(t)"
            >
              <!-- Timeline dot -->
              <div 
                :class="[
                  'absolute left-1.5 top-1.5 w-[11px] h-[11px] rounded-full border-2 z-10 transition-all',
                  selectedTransition?.id === t.id ? 'border-accent bg-accent scale-125' : 'border-indigo-500 bg-surface-1 group-hover:border-accent'
                ]"
              />

              <!-- Content -->
              <div :class="['space-y-1 transition-opacity', selectedTransition && selectedTransition.id !== t.id ? 'opacity-40' : 'opacity-100']">
                <!-- State arrived at -->
                <div class="flex items-baseline gap-1.5 font-display">
                  <span class="text-xs font-bold text-text">{{ t.to_state }}</span>
                </div>

                <!-- Trigger badge + origin -->
                <div class="flex items-center gap-1.5 flex-wrap">
                  <span class="inline-flex items-center gap-1 text-[9px] font-mono bg-indigo-500/15 text-indigo-300 px-1.5 py-0.5 rounded border border-indigo-500/25">
                    {{ t.trigger }}
                  </span>
                  <span class="text-[10px] text-text-muted">from</span>
                  <span class="text-[10px] font-mono text-text-muted">{{ t.from_state }}</span>
                </div>

                <!-- Timestamp + time spent in this state -->
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="text-[10px] text-text-muted">{{ formatTime(t.timestamp) }}</span>
                  <span
                    v-if="stateDuration(i) !== null"
                    class="inline-flex items-center gap-1 text-[10px] font-mono text-amber-600/80"
                    :title="`Time spent in ${t.to_state}`"
                  >
                    <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                      <circle cx="12" cy="12" r="9"/><polyline points="12 7 12 12 15 15"/>
                    </svg>
                    {{ stateDuration(i) }}
                  </span>
                </div>
              </div>
            </div>

            <!-- Current active state (live indicator) -->
            <div v-if="run?.current_state" class="relative pl-8 pb-1">
              <div class="absolute left-1.5 top-1.5 w-[11px] h-[11px] rounded-full bg-green-500 border-2 border-green-400 z-10 animate-pulse"/>
              <div class="space-y-0.5">
                <div class="flex items-center gap-2">
                  <span class="text-xs font-mono font-medium text-green-400">{{ run.current_state }}</span>
                  <span class="text-[10px] text-text-muted">← active</span>
                </div>
                <!-- Time spent in the current (still-active) state -->
                <div v-if="history.length" class="flex items-center gap-1 text-[10px] font-mono text-amber-600/60">
                  <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <circle cx="12" cy="12" r="9"/><polyline points="12 7 12 12 15 15"/>
                  </svg>
                  {{ activeStateDuration }}
                  <span v-if="run.status === 'running'" class="text-text-muted font-sans">and counting</span>
                </div>
                <div v-if="run.status !== 'running'" :class="`status-${run.status} text-[10px]`">
                  {{ run.status }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </aside>
    </div>

    <!-- ── Trigger modal ─────────────────────────────────────────────────── -->
    <div
      v-if="showTriggerModal"
      class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50"
      @click.self="showTriggerModal = false"
    >
      <div class="card w-80 p-5 space-y-3">
        <h3 class="font-semibold text-text text-sm">Fire Trigger</h3>
        <input v-model="triggerName" class="input font-mono" placeholder="trigger_name" @keydown.enter="fireTrigger"/>
        <div class="flex gap-2 justify-end">
          <button class="btn-ghost" @click="showTriggerModal = false">Cancel</button>
          <button class="btn-primary" @click="fireTrigger">Fire</button>
        </div>
      </div>
    </div>
    <!-- Start Run Modal -->
    <StartRunModal
      v-if="showRunModal && workflowDef"
      :definition="workflowDef"
      :initial-data="rerunInitialData"
      @close="showRunModal = false"
      @started="(newRun) => router.push(`/monitor/${newRun.id}`).then(() => location.reload())"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, shallowRef, watch, markRaw } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { VueFlow, EdgeLabelRenderer } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls }   from '@vue-flow/controls'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/controls/dist/style.css'

import BlackboardEditor from '@/components/monitor/BlackboardEditor.vue'

import StateNode      from '@/components/designer/StateNode.vue'
import BlackboardPanel from '@/components/monitor/BlackboardPanel.vue'
import EventLog        from '@/components/monitor/EventLog.vue'
import LLMDebugPanel   from '@/components/monitor/LLMDebugPanel.vue'
import MCPAuditLogPanel from '@/components/monitor/MCPAuditLogPanel.vue'
import JsonSchemaForm  from '@/components/monitor/JsonSchemaForm.vue'
import AIDebugModal    from '@/components/monitor/AIDebugModal.vue'
import StartRunModal from '@/components/shared/StartRunModal.vue'
import { useExecutionStore } from '@/stores/execution'
import { useWorkflowStore }  from '@/stores/workflows'
import { useAuthStore }      from '@/stores/auth'
import { useLayout }         from '@/composables/useLayout'
import { convertBlackboardSchemaToJsonSchema } from '@/utils/schema'

const route     = useRoute()
const router    = useRouter()
const execStore  = useExecutionStore()
const wfStore    = useWorkflowStore()
const authStore  = useAuthStore()
const { layout } = useLayout()

const activeTab    = ref('blackboard')
const showTriggerModal = ref(false)
const showRunModal = ref(false)
const showDebugModal = ref(false)
const triggerName  = ref('')
const workflowDef  = ref(null)
const workflowYAML = ref('')
const nodeTypes    = shallowRef({ stateNode: markRaw(StateNode) })
const hitlForm     = ref(null)
const showBlackboardTweak = ref(true)
const tweakView = ref('form')
const editedBlackboard = ref({})
const resolveError = ref('')
const hitlResolved = ref(false)
const selectedTransition = ref(null)

const scrubIndex = ref(0)
const isLive     = ref(true)
const showDebugTabs = ref(false)

const run          = computed(() => execStore.activeRun)
const history      = computed(() => execStore.transitions)
const runLLMCalls  = computed(() => execStore.llmCalls[route.params.id] ?? [])
const runMCPLogs   = computed(() => execStore.mcpLogs[route.params.id]  ?? [])

// Time Travel Logic
const displayHistory = computed(() => {
  if (isLive.value || scrubIndex.value >= history.value.length) return history.value
  return history.value.slice(0, scrubIndex.value + 1)
})

const displayBlackboard = computed(() => {
  if (isLive.value || scrubIndex.value >= history.value.length) return run.value?.blackboard ?? {}
  return history.value[scrubIndex.value]?.blackboard_snapshot ?? {}
})

const displayPreviousBlackboard = computed(() => {
  const currentIdx = isLive.value || scrubIndex.value >= history.value.length ? history.value.length - 1 : scrubIndex.value
  if (currentIdx <= 0) return null
  return history.value[currentIdx - 1]?.blackboard_snapshot ?? {}
})

const displayCurrentState = computed(() => {
  if (isLive.value || scrubIndex.value >= history.value.length) return run.value?.current_state
  return history.value[scrubIndex.value]?.to_state
})

const scrubDuration = computed(() => {
  if (!history.value.length) return 0
  if (isLive.value || scrubIndex.value >= history.value.length) {
    return Date.now() - new Date(history.value[0].timestamp)
  }
  return new Date(history.value[scrubIndex.value].timestamp) - new Date(history.value[0].timestamp)
})

// Auto-advance scrubber
watch(() => history.value.length, (newLen) => {
  if (isLive.value) scrubIndex.value = newLen
})

const currentStateType = computed(() => {
  if (!workflowDef.value || !displayCurrentState.value) return null
  return workflowDef.value.states?.find(s => s.name === displayCurrentState.value)?.type ?? null
})

const currentStateDef = computed(() => {
  if (!workflowDef.value || !displayCurrentState.value) return null
  return workflowDef.value.states?.find(s => s.name === displayCurrentState.value)
})

const canTerminate = computed(() => {
  if (!run.value) return false
  if (currentStateType.value === 'terminal') return false
  return ['pending', 'running', 'waiting'].includes(run.value.status)
})

const currentFormSchema = computed(() => currentStateDef.value?.form_schema)

const blackboardSchema = computed(() => {
  return convertBlackboardSchemaToJsonSchema(workflowDef.value?.blackboard?.schema)
})

const rerunInitialData = computed(() => {
  if (!run.value?.blackboard || !workflowDef.value?.blackboard?.schema) return {}
  const schema = workflowDef.value.blackboard.schema
  const bb = run.value.blackboard
  const filtered = {}
  for (const k of Object.keys(schema)) {
    if (bb[k] !== undefined) {
      filtered[k] = bb[k]
    }
  }
  return filtered
})

const visibleTabs  = computed(() => {
  const tabs = [
    { id: 'blackboard', label: 'Blackboard' },
    { id: 'thoughts',   label: 'Thoughts'   },
    { id: 'timeline',   label: 'Timeline'   },
  ]
  if (currentStateType.value === 'hitl') {
    tabs.push({ id: 'task', label: 'Task' })
  }
  if (selectedTransition.value) {
    tabs.push({ id: 'snapshot', label: 'Snapshot' })
  }
  return tabs
})

const bottomTab = ref('events')
const displayLLMCalls = computed(() => {
  const historicalCalls = []
  for (const t of displayHistory.value) {
    if (t.agent_output?.llm_calls) {
      for (const call of t.agent_output.llm_calls) {
        historicalCalls.push({
          id: `${call.stateName}-${call.agentName}-${call.timestamp}`,
          stateName: call.stateName,
          agentName: call.agentName,
          system: call.system,
          messages: call.messages,
          response: call.response,
          timestamp: call.timestamp,
        })
      }
    }
  }

  if (isLive.value && run.value?.status === 'running') {
    const liveCalls = runLLMCalls.value
    for (const lc of liveCalls) {
      const exists = historicalCalls.some(
        hc => hc.stateName === lc.stateName && hc.agentName === lc.agentName && hc.timestamp === lc.timestamp
      )
      if (!exists) {
        historicalCalls.push(lc)
      }
    }
  }
  return historicalCalls
})

const bottomTabs = computed(() => {
  return [
    { id: 'events', label: 'Event Log' },
    { id: 'llm', label: 'LLM Debug', count: displayLLMCalls.value.length },
    { id: 'mcp', label: 'MCP Debug', count: runMCPLogs.value.length },
  ]
})

// Auto-switch to Task tab when waiting at a HITL node
watch([() => run.value?.status, () => currentStateType.value], ([status, type]) => {
  if (status === 'waiting' && type === 'hitl' && !hitlResolved.value) {
    activeTab.value = 'task'
  }
}, { immediate: true })

// Reset HITL resolved state when switching to a different state
watch(() => displayCurrentState.value, () => {
  hitlResolved.value = false
})

// Capture blackboard when HITL is active
watch([() => run.value?.blackboard, () => run.value?.status], ([bb, status]) => {
  if (bb && status === 'waiting') {
    editedBlackboard.value = JSON.parse(JSON.stringify(bb))
    // Default to JSON if no schema is defined to show "full set of inputs"
    if (!blackboardSchema.value || Object.keys(blackboardSchema.value.properties || {}).length === 0) {
      tweakView.value = 'json'
    } else {
      tweakView.value = 'form'
    }
  }
}, { immediate: true, deep: true })

function syncFromTweakForm(data) {
  if (JSON.stringify(editedBlackboard.value) !== JSON.stringify(data)) {
    editedBlackboard.value = { ...data }
  }
}

function syncFromTweakJson(data) {
  // Bi-directional updates work through v-model
}

// ── Data loading ─────────────────────────────────────────────────────────────

async function loadRun() {
  const r = await execStore.fetchRun(route.params.id)
  await execStore.fetchHistory(route.params.id)
  if (authStore.hasRole('admin')) {
    await execStore.fetchMCPLogs(route.params.id)
  }
  if (r) {
    const { definition, yaml: src } = await wfStore
      .fetchOne(r.workflow_name, r.workflow_version)
      .catch(() => ({ definition: null, yaml: '' }))
    workflowDef.value = definition
    workflowYAML.value = src
  }
}

function handleKeydown(e) {
  // Option+Shift+D to toggle debug tabs
  if (e.altKey && e.shiftKey && e.code === 'KeyD') {
    e.preventDefault()
    showDebugTabs.value = !showDebugTabs.value
  }
  // Option+Shift+A to toggle AI Debug modal (if run failed)
  if (e.altKey && e.shiftKey && e.code === 'KeyA') {
    if (run.value?.status === 'failed' && workflowYAML.value) {
      e.preventDefault()
      showDebugModal.value = !showDebugModal.value
    }
  }
}

onMounted(() => {
  loadRun()
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})

// ── Vue Flow graph ────────────────────────────────────────────────────────────

// Edges don't depend on node positions — compute separately so layout() can use them
const flowEdges = computed(() => {
  if (!workflowDef.value) return []
  const edges = []
  
  for (const t of (workflowDef.value.transitions ?? [])) {
    // Handle both single target ('to') and parallel targets ('to_nodes')
    const targets = t.to_nodes ?? (t.to ? [t.to] : [])
    
    for (const target of targets) {
      const isLast = displayHistory.value.length > 0 && 
                     displayHistory.value[displayHistory.value.length - 1].from_state === t.from &&
                     displayHistory.value[displayHistory.value.length - 1].to_state === target &&
                     displayHistory.value[displayHistory.value.length - 1].trigger === t.trigger

      const isTimeout = t.trigger === 'timeout'

      edges.push({
        id:      `${t.from}__${t.trigger}__${target}`,
        source:  t.from,
        target:  target,
        animated: isLast || (isLive.value && run.value?.status === 'running'),
        class:   isLast ? 'edge-active' : '',
        data:    { 
          trigger: t.trigger,
          guard: t.guard,
          isTimeout
        },
      })
    }
  }
  return edges
})

const flowNodes = computed(() => {
  if (!workflowDef.value) return []
  const states       = workflowDef.value.states ?? []
  const currentState = displayCurrentState.value

  const rawNodes = states.map(s => ({
    id:       s.name,
    type:     'stateNode',
    position: s.position ?? { x: 200, y: 100 },
    data:     { ...s, isActive: s.name === currentState },
  }))

  // Apply dagre layout if the workflow has no explicitly saved positions
  const hasExplicit = states.some(
    s => s.position && (s.position.x !== 200 || s.position.y !== 100)
  )
  if (!hasExplicit && rawNodes.length > 1) {
    return layout(rawNodes, flowEdges.value)
  }
  return rawNodes
})

// ── Events & thinking ─────────────────────────────────────────────────────────

const filteredEvents = computed(() =>
  execStore.eventLog.filter(e => {
    const d = e.data ?? {}
    return d.run_id === route.params.id || d.run?.id === route.params.id
  })
)

const thinkingText = computed(() => execStore.thinkingTokens[route.params.id] ?? '')

const isThinking = computed(() => !!thinkingText.value && run.value?.status === 'running')

const lastReasoning = computed(() => {
  if (!isLive.value && scrubIndex.value < history.value.length) {
    return history.value[scrubIndex.value]?.agent_output?.reasoning
  }

  const calls = runLLMCalls.value
  if (!calls || calls.length === 0) return null
  // The calls are already sorted by time (added via push in store)
  // We want the most recent one that has a response with reasoning
  for (let i = calls.length - 1; i >= 0; i--) {
    if (calls[i].response?.reasoning) {
      return calls[i].response.reasoning
    }
  }
  return null
})

const displayReasoning = computed(() => {
  if (thinkingText.value) {
    // If it looks like a JSON response being streamed, try to extract the reasoning field
    const trimmed = thinkingText.value.trim()
    if (trimmed.startsWith('{')) {
      const match = thinkingText.value.match(/"reasoning":\s*"((?:[^"\\]|\\.)*)"/)
      if (match) {
        // Unescape the string if it's a partial JSON match
        try {
          return JSON.parse(`"${match[1]}"`)
        } catch (e) {
          return match[1]
        }
      }
      
      // If we see the JSON structure but not the reasoning value yet
      if (thinkingText.value.includes('"reasoning"')) return "Extracting reasoning..."
      
      // If it's a JSON block, don't show the raw JSON, wait for the field
      return null
    }
    return thinkingText.value
  }
  return lastReasoning.value
})

// ── Actions ───────────────────────────────────────────────────────────────────

async function resolve(resolution) {
  if (!run.value) return
  resolveError.value = ''
  try {
    const formPayload = hitlForm.value ? hitlForm.value.formData : {}
    const finalPayload = {
      ...editedBlackboard.value,
      ...formPayload
    }
    hitlResolved.value = true
    await execStore.resolveHITL(route.params.id, resolution, 'user', finalPayload)
    
    // Switch tab and clean up before refetching to ensure smooth transition
    activeTab.value = 'blackboard'
    showBlackboardTweak.value = false
    
    await loadRun()
  } catch (err) {
    hitlResolved.value = false
    const msg = err?.response?.data?.error ?? err?.message ?? 'Unknown error'
    resolveError.value = msg
    await loadRun()
  }
}

async function fireTrigger() {
  if (!triggerName.value) return
  await execStore.sendTrigger(route.params.id, triggerName.value)
  showTriggerModal.value = false
  triggerName.value = ''
}

async function stopRun() {
  if (!confirm('Are you sure you want to stop this run? This action cannot be undone.')) return
  try {
    await execStore.terminateRun(route.params.id)
  } catch (err) {
    const msg = err?.response?.data?.error ?? err?.message ?? 'Failed to stop run'
    alert(msg)
  }
}

function onEditClick() {
  if (!run.value) return
  router.push(`/designer/${run.value.workflow_name}/${run.value.workflow_version}`)
}

function inspectTransition(t) {
  if (selectedTransition.value?.id === t.id) {
    selectedTransition.value = null
    activeTab.value = 'blackboard'
  } else {
    selectedTransition.value = t
    activeTab.value = 'snapshot'
  }
}

function stepBack() {
  if (isLive.value || scrubIndex.value <= 0) return
  scrubIndex.value--
}

function stepForward() {
  if (isLive.value || scrubIndex.value >= history.value.length) return
  scrubIndex.value++
}

watch(isLive, (live) => {
  if (live) {
    scrubIndex.value = history.value.length
  }
})

// ── Formatting ────────────────────────────────────────────────────────────────

function formatTime(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  const now = Date.now()
  const diff = now - d.getTime()
  if (diff < 60_000)  return 'just now'
  if (diff < 3600_000) {
    const m = Math.floor(diff / 60_000)
    return `${m}m ago`
  }
  return d.toLocaleTimeString('en', { hour12: false, hour: '2-digit', minute: '2-digit' })
}

function formatDuration(ms) {
  if (ms < 1000)    return '< 1s'
  const s = Math.floor(ms / 1000)
  if (s < 60)       return `${s}s`
  const m = Math.floor(s / 60)
  const rs = s % 60
  if (m < 60)       return rs ? `${m}m ${rs}s` : `${m}m`
  const h = Math.floor(m / 60)
  const rm = m % 60
  return rm ? `${h}h ${rm}m` : `${h}h`
}

// Time spent in history[i].to_state = next transition timestamp − this timestamp.
// Returns null for the very last completed entry (duration belongs to activeStateDuration).
function stateDuration(i) {
  const next = history.value[i + 1]
  if (!next) return null   // last entry — shown separately as activeStateDuration
  const ms = new Date(next.timestamp) - new Date(history.value[i].timestamp)
  return formatDuration(ms)
}

// Duration the run has been in its current state (last transition → now).
const activeStateDuration = computed(() => {
  const last = history.value[history.value.length - 1]
  if (!last) return null
  return formatDuration(Date.now() - new Date(last.timestamp))
})
</script>
