<template>
  <div class="flex flex-col h-full">
    <div class="px-4 py-2.5 border-b border-border flex items-center gap-3">
      <span class="text-xs font-semibold text-text-muted uppercase tracking-wider">Event Log</span>
      <span class="text-xs text-text-muted">{{ events.length }} events</span>
      <button @click="clear" class="ml-auto btn-ghost text-xs py-0.5 px-2">Clear</button>
    </div>

    <div ref="logEl" class="flex-1 overflow-y-auto p-2 space-y-0.5 font-mono text-xs">
      <div
        v-for="(ev, i) in events" :key="i"
        class="flex items-start gap-2 px-2 py-1.5 rounded hover:bg-white/5 group"
      >
        <span class="text-text-muted shrink-0 w-[72px]">{{ formatTime(ev.timestamp) }}</span>
        <span :class="['shrink-0 w-32 truncate', eventColor(ev.type)]">{{ ev.type }}</span>
        <span :class="['text-text-muted', ev.type === 'run.failed' ? 'break-all' : 'truncate group-hover:whitespace-normal']">{{ summarise(ev) }}</span>
      </div>

      <div v-if="!events.length" class="text-text-muted text-center py-8">
        Waiting for events…
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'

const props = defineProps({
  events:  { type: Array, default: () => [] },
  autoScroll: { type: Boolean, default: true }
})

const emit = defineEmits(['clear'])
const logEl = ref(null)
const events = ref([...props.events])

watch(() => props.events, (v) => {
  events.value = [...v]
  if (props.autoScroll) nextTick(() => {
    if (logEl.value) logEl.value.scrollTop = 0
  })
}, { deep: true })

function clear() { events.value = []; emit('clear') }

function formatTime(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  return d.toLocaleTimeString('en', { hour12: false })
}

const EVENT_COLORS = {
  'run.created':        'text-green-400',
  'run.completed':      'text-green-500',
  'run.failed':         'text-red-400',
  'state.changed':      'text-indigo-400',
  'blackboard.updated': 'text-blue-400',
  'agent.thinking':     'text-text-muted',
  'agent.tool_call':    'text-cyan-600',
  'agent.prompt':       'text-violet-400',
  'agent.response':     'text-violet-300',
  'hitl.waiting':       'text-amber-400',
  'hitl.resolved':      'text-amber-300',
}
function eventColor(type) { return EVENT_COLORS[type] ?? 'text-text-muted' }

function summarise(ev) {
  const d = ev.data ?? {}
  switch (ev.type) {
    case 'state.changed':      return `${d.from_state} → ${d.to_state}  (${d.trigger})`
    case 'blackboard.updated': return `${d.key} = ${JSON.stringify(d.value)}`
    case 'agent.thinking':     return `[${d.agent_name}] ${(d.token ?? '').slice(0, 60)}`
    case 'agent.tool_call':    return `[${d.agent_name}] ${d.tool_name}${d.output !== undefined ? ' → done' : ''}`
    case 'agent.prompt':       return `[${d.agent_name}] → ${d.state_name} (${(d.system ?? '').length} chars system prompt)`
    case 'agent.response':     return `[${d.agent_name}] trigger=${d.trigger}${d.reasoning ? ' — ' + d.reasoning : ''}`
    case 'run.created':        return `run ${d.run?.id?.slice(0, 8)}`
    case 'hitl.waiting':       return `${d.state_name}${d.assignee ? ` → ${d.assignee}` : ''}`
    case 'run.failed': {
      const reason = d.run?.failure_reason ?? d.error ?? 'unknown error'
      // Strip Temporal activity envelope — keep only the innermost cause
      const clean = reason.replace(/^.*?: /g, '').replace(/\(type:.*?\)/g, '').trim()
      return clean || reason
    }
    case 'run.completed':      return d.run?.id ? `run ${d.run.id.slice(0, 8)} completed` : 'completed'
    default:                   return JSON.stringify(d).slice(0, 120)
  }
}
</script>
