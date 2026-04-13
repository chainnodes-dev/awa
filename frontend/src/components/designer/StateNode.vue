<template>
  <div
    :class="[
      'state-node relative flex flex-col gap-1 px-3 py-2.5 rounded-xl border-2 min-w-[140px] cursor-default select-none transition-shadow duration-300',
      nodeStyle,
      isActive && `glow-${data.type ?? 'intermediate'}`,
    ]"
  >
    <div class="flex items-center gap-2">
      <div :class="['w-2.5 h-2.5 rounded-full shrink-0', dotColor]"/>
      <span class="text-sm font-semibold text-slate-100 leading-tight truncate">{{ data.name }}</span>
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
    </div>

    <!-- Agent badge -->
    <div v-if="data.agent" class="flex items-center gap-1">
      <span class="text-[10px] font-mono text-slate-500 truncate">⚡ {{ data.agent }}</span>
    </div>

    <!-- Timeout badge -->
    <div v-if="data.timeout" class="text-[10px] text-amber-500/80">
      ⏱ {{ data.timeout }}
    </div>

    <!-- Active state blackboard preview -->
    <div v-if="isActive && bbPreview.length" class="mt-1 pt-1 border-t border-white/10 space-y-0.5">
      <div v-for="[k, v] in bbPreview" :key="k" class="flex gap-1 text-[10px]">
        <span class="text-slate-500 shrink-0">{{ k }}:</span>
        <span class="text-slate-300 truncate font-mono">{{ formatVal(v) }}</span>
      </div>
    </div>

    <!-- Vue Flow handles -->
    <Handle v-if="data.type !== 'initial'" type="target" :position="Position.Top"
            class="!w-3 !h-3 !bg-slate-600 !border-slate-500 hover:!bg-accent" />
    <Handle v-if="data.type !== 'terminal'" type="source" :position="Position.Bottom"
            class="!w-3 !h-3 !bg-slate-600 !border-slate-500 hover:!bg-accent" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'

const props = defineProps({
  data: { type: Object, required: true },
  selected: Boolean,
  isActive: { type: Boolean, default: false },
  blackboard: { type: Object, default: () => ({}) }
})

const TYPE_STYLES = {
  initial:      'border-green-500/60 bg-green-500/10',
  intermediate: 'border-indigo-500/60 bg-indigo-500/10',
  hitl:         'border-amber-500/60 bg-amber-500/10',
  terminal:     'border-slate-500/60 bg-slate-500/10',
  wait:         'border-indigo-400/60 bg-indigo-500/5',
  script:       'border-amber-500/50 bg-amber-500/5',
  code:         'border-teal-500/60 bg-teal-500/10',
}
const TYPE_DOTS = {
  initial:      'bg-green-500',
  intermediate: 'bg-indigo-500',
  hitl:         'bg-amber-500',
  terminal:     'bg-slate-500',
  wait:         'bg-indigo-400',
  script:       'bg-amber-500',
  code:         'bg-teal-400',
}
const TYPE_COLORS = {
  initial: '#22c55e', intermediate: '#6366f1', hitl: '#f59e0b', terminal: '#64748b', wait: '#818cf8',
  script: '#f59e0b', code: '#2dd4bf',
}

const nodeStyle = computed(() => {
  const base = TYPE_STYLES[props.data.type] ?? TYPE_STYLES.intermediate
  return props.selected
    ? base + ' ring-1 ring-accent/60 shadow-lg shadow-accent/20'
    : base
})
const dotColor = computed(() => TYPE_DOTS[props.data.type] ?? TYPE_DOTS.intermediate)
const typeColor = computed(() => TYPE_COLORS[props.data.type] ?? TYPE_COLORS.intermediate)

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
