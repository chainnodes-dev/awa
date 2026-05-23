<template>
  <div class="p-4 space-y-3 flex-1 overflow-y-auto">
    <div class="flex items-center gap-2 mb-1">
      <span class="text-xs font-semibold text-text-muted uppercase tracking-wider flex-1">New Transition</span>
      <button class="text-text-muted hover:text-text text-sm leading-none p-1" @click="$emit('cancel')">✕</button>
    </div>
    <div class="flex items-center gap-2 py-2 px-3 rounded-lg bg-surface-0 border border-border">
      <span class="text-xs font-mono text-indigo-400 truncate">{{ connection.source }}</span>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#4b5563" stroke-width="2">
        <line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/>
      </svg>
      <span class="text-xs font-mono text-indigo-400 truncate">{{ connection.target }}</span>
    </div>
    <div class="space-y-1">
      <label class="text-xs text-text-muted">Trigger name *</label>
      <TriggerCombobox v-model="trigger" :options="options" placeholder="e.g. validation_passed" />
    </div>
    <div class="space-y-1">
      <label class="text-xs text-text-muted">Guard expression</label>
      <input class="input font-mono" v-model="guard" placeholder="e.g. amount > 0" @keydown.enter="onConfirm" />
    </div>
    <p class="text-[10px] text-text-muted">Leave guard empty to always follow this transition.</p>
    <div class="flex gap-2 pt-1">
      <button class="btn-ghost text-xs flex-1" @click="$emit('cancel')">Cancel</button>
      <button class="btn-primary text-xs flex-1" :class="{ 'opacity-40 cursor-not-allowed': !trigger }" :disabled="!trigger" @click="onConfirm">Create</button>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import TriggerCombobox from './TriggerCombobox.vue'

const props = defineProps({
  connection: Object,
  options:    { type: Array, default: () => [] }
})
const emit  = defineEmits(['confirm', 'cancel'])
const trigger = ref('')
const guard   = ref('')
function onConfirm() {
  if (!trigger.value.trim()) return
  emit('confirm', { trigger: trigger.value.trim(), guard: guard.value.trim() })
}
</script>
