<template>
  <div class="flex flex-col h-full overflow-hidden">

    <!-- Empty state -->
    <div v-if="!calls.length" class="flex flex-col items-center justify-center h-full gap-2 text-text-muted text-xs">
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="opacity-40">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
      </svg>
      <span>No LLM calls recorded yet</span>
      <span class="text-slate-700 text-[10px]">Events appear here while a run is active</span>
    </div>

    <!-- Call list -->
    <div v-else class="flex-1 overflow-y-auto p-3 space-y-2">
      <div
        v-for="call in reversedCalls"
        :key="call.id"
        class="rounded-lg border border-border bg-surface-2 overflow-hidden"
      >
        <!-- Card header -->
        <button
          class="w-full flex items-center gap-2 px-3 py-2.5 text-left hover:bg-white/5 transition-colors"
          @click="toggle(call.id)"
        >
          <!-- Status dot -->
          <span
            :class="[
              'w-2 h-2 rounded-full shrink-0',
              call.response ? 'bg-green-500' : 'bg-amber-400 animate-pulse',
            ]"
          />

          <!-- State / agent labels -->
          <span class="text-xs font-mono text-text truncate">{{ call.stateName }}</span>
          <span class="text-[10px] text-text-muted shrink-0">·</span>
          <span class="text-xs font-mono text-accent/80 truncate">{{ call.agentName }}</span>

          <span class="flex-1"/>

          <!-- Trigger badge (once response is available) -->
          <span
            v-if="call.response?.trigger"
            class="text-[10px] font-mono bg-indigo-500/15 text-indigo-300 px-1.5 py-0.5 rounded border border-indigo-500/25 shrink-0"
          >{{ call.response.trigger }}</span>

          <!-- Timestamp -->
          <span class="text-[10px] text-text-muted shrink-0 ml-1">{{ fmtTime(call.timestamp) }}</span>

          <!-- Chevron -->
          <svg
            class="w-3 h-3 text-text-muted shrink-0 transition-transform duration-150 ml-1"
            :class="expanded.has(call.id) ? 'rotate-180' : ''"
            viewBox="0 0 20 20" fill="currentColor"
          >
            <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
          </svg>
        </button>

        <!-- Expanded body -->
        <div v-if="expanded.has(call.id)" class="border-t border-border divide-y divide-border/50">

          <!-- System prompt section -->
          <div class="px-3 py-2">
            <button
              class="flex items-center gap-1.5 text-[10px] text-text-muted hover:text-text font-semibold uppercase tracking-wider mb-1.5 transition-colors"
              @click="toggleSection(call.id, 'system')"
            >
              <svg
                class="w-2.5 h-2.5 transition-transform duration-100"
                :class="sectionOpen(call.id, 'system') ? 'rotate-90' : ''"
                viewBox="0 0 20 20" fill="currentColor"
              >
                <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd"/>
              </svg>
              System prompt
            </button>
            <pre
              v-if="sectionOpen(call.id, 'system')"
              class="text-xs font-mono text-text-muted whitespace-pre-wrap break-words leading-relaxed bg-black/20 rounded px-2.5 py-2 max-h-64 overflow-y-auto"
            >{{ call.system }}</pre>
          </div>

          <!-- User messages section -->
          <div class="px-3 py-2">
            <button
              class="flex items-center gap-1.5 text-[10px] text-text-muted hover:text-text font-semibold uppercase tracking-wider mb-1.5 transition-colors"
              @click="toggleSection(call.id, 'messages')"
            >
              <svg
                class="w-2.5 h-2.5 transition-transform duration-100"
                :class="sectionOpen(call.id, 'messages') ? 'rotate-90' : ''"
                viewBox="0 0 20 20" fill="currentColor"
              >
                <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd"/>
              </svg>
              Messages ({{ msgCount(call) }})
            </button>
            <div v-if="sectionOpen(call.id, 'messages')" class="space-y-1.5">
              <div
                v-for="(msg, mi) in call.messages"
                :key="mi"
                class="rounded bg-black/20 px-2.5 py-2"
              >
                <div class="flex items-center gap-1.5 mb-1">
                  <span
                    :class="[
                      'text-[10px] font-mono font-semibold px-1 py-0.5 rounded',
                      msg.role === 'user'      ? 'text-sky-400 bg-sky-500/10'
                      : msg.role === 'assistant' ? 'text-violet-400 bg-violet-500/10'
                      : 'text-text-muted bg-white/5',
                    ]"
                  >{{ msg.role }}</span>
                </div>
                <pre class="text-xs font-mono text-text-muted whitespace-pre-wrap break-words leading-relaxed max-h-48 overflow-y-auto">{{ msgContent(msg) }}</pre>
              </div>
            </div>
          </div>

          <!-- Response section -->
          <div v-if="call.response" class="px-3 py-2">
            <div class="text-[10px] text-text-muted font-semibold uppercase tracking-wider mb-1.5">Response</div>

            <!-- Reasoning -->
            <div v-if="call.response.reasoning" class="mb-2 rounded bg-black/20 px-2.5 py-2">
              <div class="text-[10px] text-text-muted mb-1">Reasoning</div>
              <p class="text-sm font-mono text-text-muted whitespace-pre-wrap break-words leading-relaxed">{{ call.response.reasoning }}</p>
            </div>

            <!-- Raw content -->
            <div class="rounded bg-black/20 px-2.5 py-2">
              <div class="text-[10px] text-text-muted mb-1">Raw content</div>
              <pre class="text-sm font-mono text-text-muted whitespace-pre-wrap break-words leading-relaxed max-h-64 overflow-y-auto">{{ call.response.content }}</pre>
            </div>
          </div>

          <!-- Awaiting response -->
          <div v-else class="px-3 py-2 flex items-center gap-2 text-xs text-amber-500/70">
            <div class="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse shrink-0"/>
            Awaiting response…
          </div>

        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, reactive } from 'vue'

const props = defineProps({
  calls: { type: Array, default: () => [] },
})

// Newest calls at the top
const reversedCalls = computed(() => [...props.calls].reverse())

// Which cards are expanded
const expanded = reactive(new Set())
function toggle(id) {
  expanded.has(id) ? expanded.delete(id) : expanded.add(id)
}

// Per-card section toggles: 'system' | 'messages'
const sections = reactive({})
function sectionOpen(callId, section) {
  return sections[`${callId}:${section}`] ?? (section === 'messages') // messages open by default
}
function toggleSection(callId, section) {
  const k = `${callId}:${section}`
  sections[k] = !sectionOpen(callId, section)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

function msgCount(call) {
  return call.messages?.length ?? 0
}

function msgContent(msg) {
  const c = msg.content
  if (typeof c === 'string') return c
  if (Array.isArray(c)) {
    return c.map(block => {
      if (block.type === 'text')        return block.text ?? ''
      if (block.type === 'tool_use')    return `[tool_use: ${block.name}] ${JSON.stringify(block.input, null, 2)}`
      if (block.type === 'tool_result') return `[tool_result: ${block.tool_use_id}] ${JSON.stringify(block.output, null, 2)}`
      return JSON.stringify(block)
    }).join('\n')
  }
  return JSON.stringify(c, null, 2)
}

function fmtTime(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  return d.toLocaleTimeString('en', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })
}
</script>
