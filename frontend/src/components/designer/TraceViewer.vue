<template>
  <div class="flex flex-col h-full bg-surface-1">
    <div class="p-3 border-b border-border flex items-center gap-2 shrink-0">
      <svg class="text-text-muted" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
      </svg>
      <span class="text-xs font-semibold text-text-muted uppercase tracking-wider">LLM Interaction Trace</span>
      <div class="flex-1"/>
      <button @click="$emit('close')" class="text-text-muted hover:text-text-muted transition-colors">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
    </div>

    <div class="flex-1 overflow-y-auto p-4 space-y-6">
      <div v-for="(msg, i) in interactions" :key="i" class="space-y-1.5">
        <div class="flex items-center gap-2">
          <span :class="['text-[10px] font-bold uppercase tracking-widest px-1.5 py-0.5 rounded', roleClass(msg.role)]">
            {{ msg.role }}
          </span>
          <span v-if="msg.role === 'system'" class="text-[9px] text-text-muted font-medium">Core Instructions</span>
        </div>
        
        <div class="bg-surface-0 border border-border rounded-lg overflow-hidden">
          <pre class="p-3 text-sm font-mono leading-relaxed text-text overflow-x-auto whitespace-pre-wrap">{{ msg.content }}</pre>
        </div>
      </div>

      <div v-if="!interactions?.length" class="h-full flex flex-col items-center justify-center text-text-muted space-y-2 opacity-40">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
        </svg>
        <span class="text-xs">No interaction recorded for this turn.</span>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  interactions: { type: Array, default: () => [] }
})
defineEmits(['close'])

function roleClass(role) {
  switch (role) {
    case 'system':    return 'bg-slate-500/10 text-text-muted'
    case 'user':      return 'bg-indigo-500/10 text-indigo-400'
    case 'assistant': return 'bg-emerald-500/10 text-emerald-400'
    default:          return 'bg-slate-500/10 text-text-muted'
  }
}
</script>
