<template>
  <div class="flex flex-col h-full overflow-hidden">

    <!-- Empty state -->
    <div v-if="!logs.length" class="flex flex-col items-center justify-center h-full gap-2 text-text-muted text-xs">
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="opacity-40">
        <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
      </svg>
      <span>No MCP communication recorded</span>
      <span class="text-slate-700 text-[10px]">MCP server interactions appear here</span>
    </div>

    <!-- Log list -->
    <div v-else class="flex-1 overflow-y-auto p-3 space-y-2">
      <div
        v-for="log in reversedLogs"
        :key="log.id || log.created_at"
        class="rounded-lg border border-border bg-surface-2 overflow-hidden"
      >
        <!-- Card header -->
        <button
          class="w-full flex items-center gap-2 px-3 py-2.5 text-left hover:bg-white/5 transition-colors"
          @click="toggle(log.id || log.created_at)"
        >
          <!-- Status dot -->
          <span
            :class="[
              'w-2 h-2 rounded-full shrink-0',
              log.is_error ? 'bg-red-500' : 'bg-green-500',
            ]"
          />

          <!-- Method / Tool labels -->
          <span class="text-xs font-mono text-text-muted shrink-0">{{ log.method }}</span>
          <span v-if="log.tool_name" class="text-[10px] text-text-muted shrink-0">·</span>
          <span v-if="log.tool_name" class="text-xs font-mono text-accent/80 truncate">{{ log.tool_name }}</span>

          <span class="flex-1"/>

          <!-- Duration -->
          <span class="text-[10px] font-mono text-amber-500/70 shrink-0">{{ log.duration_ms }}ms</span>

          <!-- Timestamp -->
          <span class="text-[10px] text-text-muted shrink-0 ml-1">{{ fmtTime(log.created_at) }}</span>

          <!-- Chevron -->
          <svg
            class="w-3 h-3 text-text-muted shrink-0 transition-transform duration-150 ml-1"
            :class="expanded.has(log.id || log.created_at) ? 'rotate-180' : ''"
            viewBox="0 0 20 20" fill="currentColor"
          >
            <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
          </svg>
        </button>

        <!-- Expanded body -->
        <div v-if="expanded.has(log.id || log.created_at)" class="border-t border-border divide-y divide-border/50">
          
          <!-- Metadata -->
          <div class="px-3 py-2 bg-black/10">
             <div class="flex flex-col gap-1 text-[10px]">
                <div class="flex gap-2">
                   <span class="text-text-muted w-16">Server:</span>
                   <span class="text-text font-mono break-all">{{ log.server_url }}</span>
                </div>
                <div class="flex gap-2">
                   <span class="text-text-muted w-16">Context:</span>
                   <span class="text-text-muted font-mono">{{ log.state_name }} <span class="text-text-muted px-1">/</span> {{ log.agent_name }}</span>
                </div>
             </div>
          </div>

          <!-- Error message if applicable -->
          <div v-if="log.is_error" class="px-3 py-2 bg-red-500/5">
            <div class="text-[10px] text-red-400 font-semibold uppercase tracking-wider mb-1">Error</div>
            <pre class="text-sm font-mono text-red-300 whitespace-pre-wrap break-words">{{ log.error_msg }}</pre>
          </div>

          <!-- Input section -->
          <div v-if="log.input" class="px-3 py-2">
            <div class="text-[10px] text-text-muted font-semibold uppercase tracking-wider mb-1.5">Input</div>
            <pre class="text-sm font-mono text-text-muted whitespace-pre-wrap break-words leading-relaxed bg-black/20 rounded px-2.5 py-2 max-h-64 overflow-y-auto">{{ fmtJson(log.input) }}</pre>
          </div>

          <!-- Output section -->
          <div v-if="log.output" class="px-3 py-2">
            <div class="text-[10px] text-text-muted font-semibold uppercase tracking-wider mb-1.5">Output</div>
            <pre class="text-sm font-mono text-text-muted whitespace-pre-wrap break-words leading-relaxed bg-black/20 rounded px-2.5 py-2 max-h-96 overflow-y-auto">{{ fmtJson(log.output) }}</pre>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, reactive } from 'vue'

const props = defineProps({
  logs: { type: Array, default: () => [] },
})

const reversedLogs = computed(() => [...props.logs].reverse())

const expanded = reactive(new Set())
function toggle(id) {
  expanded.has(id) ? expanded.delete(id) : expanded.add(id)
}

function fmtJson(obj) {
  if (!obj) return ''
  return JSON.stringify(obj, null, 2)
}

function fmtTime(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  return d.toLocaleTimeString('en', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })
}
</script>
