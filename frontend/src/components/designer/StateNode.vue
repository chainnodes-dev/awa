<template>
  <div
    :class="[
      'state-node relative flex flex-col gap-1 px-4 py-3 rounded-2xl border-2 min-w-[150px] cursor-default select-none transition-shadow duration-300',
      nodeStyle,
      isActive && `glow-${data.type ?? 'prompt'}`,
    ]"
    @mouseenter="hovered = true"
    @mouseleave="hovered = false"
  >
    <!-- Instructions hover tooltip -->
    <Transition name="tooltip">
      <div
        v-if="hovered && data.instructions"
        class="absolute left-1/2 -translate-x-1/2 bottom-[calc(100%+8px)] z-50 w-64 p-2.5 rounded-lg border border-border bg-surface-0 shadow-xl pointer-events-none"
      >
        <p class="text-[10px] text-text leading-relaxed whitespace-pre-wrap">{{ data.instructions }}</p>
        <div class="absolute left-1/2 -translate-x-1/2 top-full w-2 h-2 rotate-45 bg-surface-0 border-r border-b border-border -mt-1"/>
      </div>
    </Transition>

    <div class="flex items-center gap-2">
      <div :class="['w-2.5 h-2.5 rounded-full shrink-0', dotColor]"/>
      <span class="text-sm font-semibold text-text leading-tight truncate">{{ data.name }}</span>
      <span v-if="data.type === 'initial' && data.triggerType" class="ml-auto flex items-center gap-1 text-emerald-400 text-[10px] font-bold uppercase tracking-wider bg-emerald-500/10 px-1.5 py-0.5 rounded border border-emerald-500/20">
        {{ data.triggerType }}
      </span>
      <span v-if="data.type === 'telegram_output'" class="ml-auto text-blue-400" title="Send Telegram">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <path d="M22 2L11 13M22 2l-7 20-4-9-9-4Z"/>
        </svg>
      </span>
      <span v-if="data.type === 'discord_output'" class="ml-auto text-indigo-400" title="Send Discord">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <path d="M18 8a3 3 0 0 0-3-3H5a3 3 0 0 0-3 3v8a3 3 0 0 0 3 3h10a3 3 0 0 0 3-3V8Z"/>
          <path d="M22 10v4"/>
        </svg>
      </span>
      <span v-if="data.type === 'hitl'" class="ml-auto text-amber-400">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
          <path d="M12 2a5 5 0 1 0 5 5 5 5 0 0 0-5-5zm0 8a3 3 0 1 1 3-3 3 3 0 0 1-3 3zm9 11v-1a7 7 0 0 0-7-7h-4a7 7 0 0 0-7 7v1h2v-1a5 5 0 0 1 5-5h4a5 5 0 0 1 5 5v1z"/>
        </svg>
      </span>
      <span v-if="data.type === 'wait'" class="ml-auto text-indigo-400">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
        </svg>
      </span>
      <span v-if="data.type === 'code'" class="ml-auto text-teal-400">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
        </svg>
      </span>
      <span v-if="data.type === 'subprocess'" class="ml-auto text-violet-400">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
        </svg>
      </span>
      <span v-if="data.type === 'timeout_node'" class="ml-auto text-red-500" title="Timeout Handler">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <circle cx="12" cy="12" r="10"/><path d="m12 6-2 6h4l-2 6"/>
        </svg>
      </span>
    </div>

    <!-- Agent badge -->
    <div v-if="data.agent" class="flex items-center gap-1">
      <span class="text-[10px] font-mono text-text-muted truncate">⚡ {{ data.agent }}</span>
    </div>

    <!-- Timeout badge -->
    <div v-if="data.timeout" class="text-[10px] text-amber-500/80">
      ⏱ {{ data.timeout }}
    </div>

    <!-- Active state blackboard preview -->
    <div v-if="isActive && bbPreview.length" class="mt-1 pt-1 border-t border-white/10 space-y-0.5">
      <div v-for="[k, v] in bbPreview" :key="k" class="flex gap-1 text-[10px]">
        <span class="text-text-muted shrink-0">{{ k }}:</span>
        <span class="text-text truncate font-mono">{{ formatVal(v) }}</span>
      </div>
    </div>

    <!-- Vue Flow handles -->
    <!-- Target handle (input) -->
    <Handle v-if="data.type !== 'initial'" type="target" :position="Position.Top"
            class="!w-3 !h-3 !bg-text-muted !border-text-muted hover:!bg-accent" />

    <!-- Source handle (primary output) -->
    <Handle v-if="data.type !== 'terminal' && data.type !== 'timeout_node'" type="source" :position="Position.Bottom"
            class="!w-3 !h-3 !bg-text-muted !border-text-muted hover:!bg-accent" />

    <!-- Dedicated Timeout handle (red, right side) -->
    <Handle v-if="data.type !== 'terminal' && data.type !== 'timeout_node' && data.timeout" type="source" :position="Position.Right"
            id="timeout"
            class="!w-3 !h-3 !bg-red-500 !border-red-600 hover:!bg-red-400" />
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { Handle, Position } from '@vue-flow/core'

const hovered = ref(false)

const props = defineProps({
  id: { type: String, required: true },
  data: { type: Object, required: true },
  selected: Boolean,
  isActive: { type: Boolean, default: false },
  blackboard: { type: Object, default: () => ({}) }
})

const TYPE_STYLES = {
  initial:      'border-green-500/50 bg-green-500/5 backdrop-blur-md shadow-[0_0_12px_rgba(34,197,94,0.12)]',
  prompt:       'border-indigo-500/50 bg-indigo-500/5 backdrop-blur-md shadow-[0_0_12px_rgba(99,102,241,0.12)]',
  hitl:         'border-amber-500/40 bg-amber-500/5 backdrop-blur-md shadow-[0_0_12px_rgba(245,158,11,0.12)]',
  terminal:     'border-slate-500/40 bg-slate-500/5 backdrop-blur-md shadow-[0_0_12px_rgba(148,163,184,0.08)]',
  subprocess:   'border-violet-500/50 bg-violet-500/5 backdrop-blur-md shadow-[0_0_12px_rgba(167,139,250,0.12)]',
  wait:         'border-slate-500/40 bg-slate-500/5 backdrop-blur-md shadow-[0_0_12px_rgba(148,163,184,0.08)]',
  script:       'border-amber-500/40 bg-amber-500/5 backdrop-blur-md shadow-[0_0_12px_rgba(245,158,11,0.12)]',
  code:         'border-teal-500/50 bg-teal-500/5 backdrop-blur-md shadow-[0_0_12px_rgba(45,212,191,0.12)]',
  emit_event:   'border-blue-500/40 bg-blue-500/5 backdrop-blur-md shadow-[0_0_12px_rgba(59,130,246,0.12)]',
  telegram_output: 'border-blue-500/50 bg-blue-500/5 backdrop-blur-md shadow-[0_0_12px_rgba(59,130,246,0.12)]',
  discord_output:  'border-indigo-500/50 bg-indigo-500/5 backdrop-blur-md shadow-[0_0_12px_rgba(99,102,241,0.12)]',
  timeout_node:    'border-red-500/50 bg-red-500/5 backdrop-blur-md shadow-[0_0_12px_rgba(239,68,68,0.12)]',
}
const TYPE_DOTS = {
  initial:      'bg-indigo-400',
  terminal:     'bg-indigo-400',
  prompt:       'bg-indigo-500',
  script:       'bg-amber-400',
  code:         'bg-teal-400',
  subprocess:   'bg-violet-400',
  hitl:         'bg-amber-500',
  wait:         'bg-slate-500',
  emit_event:   'bg-blue-400',
  telegram_output: 'bg-blue-400',
  discord_output:  'bg-indigo-400',
  timeout_node:    'bg-red-500',
}
const TYPE_COLORS = {
  initial: '#818cf8', terminal: '#818cf8', prompt: '#6366f1',
  script: '#f59e0b', code: '#2dd4bf', subprocess: '#a78bfa',
  hitl: '#f59e0b', wait: '#94a3b8', emit_event: '#60a5fa',
  telegram_output: '#60a5fa', discord_output: '#818cf8',
  timeout_node:    '#ef4444',
}

const nodeStyle = computed(() => {
  const base = TYPE_STYLES[props.data.type] ?? TYPE_STYLES.prompt
  return props.selected
    ? base + ' ring-1 ring-accent/60 shadow-lg shadow-accent/20'
    : base
})
const dotColor = computed(() => TYPE_DOTS[props.data.type] ?? TYPE_DOTS.prompt)
const typeColor = computed(() => TYPE_COLORS[props.data.type] ?? TYPE_COLORS.prompt)

// Show top 3 blackboard fields when active
const bbPreview = computed(() =>
  props.isActive
    ? Object.entries(props.blackboard).slice(0, 3)
    : []
)

function formatVal(v) {
  if (v === null || v === undefined) return '—'
  if (typeof v === 'boolean') return v ? 'true' : 'false'
  if (typeof v === 'object') return JSON.stringify(v).slice(0, 20)
  return String(v).slice(0, 24)
}
</script>

<style scoped>
.tooltip-enter-active,
.tooltip-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.tooltip-enter-from,
.tooltip-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(4px);
}
.tooltip-enter-to,
.tooltip-leave-from {
  opacity: 1;
  transform: translateX(-50%) translateY(0);
}
</style>
