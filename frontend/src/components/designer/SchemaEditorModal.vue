<template>
  <div class="fixed inset-0 z-[60] flex items-center justify-center p-6 bg-black/60 backdrop-blur-sm" @click.self="$emit('close')">
    <div class="bg-surface-1 border border-border rounded-2xl w-full max-w-2xl shadow-2xl flex flex-col overflow-hidden animate-in zoom-in duration-200">
      <header class="px-6 py-4 border-b border-border flex items-center justify-between shrink-0 bg-surface-2/50">
        <div class="flex items-center gap-3">
          <div class="p-2 rounded-lg bg-violet-500/10 text-violet-400">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
              <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
              <line x1="12" y1="22.08" x2="12" y2="12"/>
            </svg>
          </div>
          <div>
            <h3 class="text-sm font-bold text-text">Editing Schema: <span class="text-violet-400 font-mono">{{ name }}</span></h3>
            <p class="text-[10px] text-text-muted uppercase tracking-widest font-semibold opacity-70">Focus Mode</p>
          </div>
        </div>
        <button @click="$emit('close')" class="p-2 rounded-lg hover:bg-white/5 text-text-muted hover:text-text transition-all">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </header>

      <div class="flex-1 overflow-y-auto p-8 bg-surface-0/50">
        <div class="max-w-xl mx-auto">
          <div class="mb-8 p-4 rounded-xl bg-violet-500/5 border border-violet-500/20">
            <h4 class="text-xs font-bold text-violet-300 mb-1">Structural Overview</h4>
            <p class="text-xs text-text-muted leading-relaxed">
              You are defining the internal properties for <span class="font-mono text-text">{{ name }}</span>. 
              These fields will be available in the blackboard and can be used for data binding in HITL forms or scripts.
            </p>
          </div>

          <BlackboardSchemaEditor 
            :schema="schema" 
            :title="`Properties of ${name}`"
            @change="$emit('change')"
            @open-popout="$emit('open-popout', $event)"
          />
        </div>
      </div>

      <footer class="px-6 py-4 bg-surface-2/50 border-t border-border flex justify-between items-center shrink-0">
        <div class="flex items-center gap-2 text-[10px] text-text-muted italic">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/>
          </svg>
          Changes are saved to the workflow automatically.
        </div>
        <button @click="$emit('close')" class="btn-primary px-8">Done</button>
      </footer>
    </div>
  </div>
</template>

<script setup>
import BlackboardSchemaEditor from './BlackboardSchemaEditor.vue'

defineProps({
  name:   { type: String, required: true },
  schema: { type: Object, required: true }
})

defineEmits(['close', 'change', 'open-popout'])
</script>
