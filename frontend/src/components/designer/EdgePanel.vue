<template>
  <div class="p-4 space-y-3 flex-1 overflow-y-auto">
    <div class="flex items-center gap-2 mb-1">
      <span class="text-xs font-semibold text-text-muted uppercase tracking-wider flex-1">Transition</span>
      <button class="btn-danger text-xs py-1" @click="$emit('delete')">Delete</button>
    </div>
    <p class="text-[10px] font-mono text-text-muted bg-surface-0 px-2 py-1.5 rounded border border-border">
      {{ form.from }}  →  {{ form.to }}
    </p>
    <div class="space-y-1">
      <label class="text-xs text-text-muted">Trigger</label>
      <TriggerCombobox v-model="form.label" :options="options" />
    </div>
    <div class="space-y-1">
      <label class="text-xs text-text-muted">Guard</label>
      <input class="input font-mono" v-model="form.guard" placeholder="e.g. amount > 0" />
    </div>
    <p class="text-[10px] text-text-muted">Leave guard empty to always follow this transition.</p>
    <button class="btn-primary w-full mt-1" @click="$emit('update', { ...form })">Apply</button>
  </div>
</template>

<script setup>
import { reactive } from 'vue'
import TriggerCombobox from './TriggerCombobox.vue'

const props = defineProps({
  edge:    Object,
  options: { type: Array, default: () => [] }
})
defineEmits(['update', 'delete'])
const form = reactive({ ...props.edge })
</script>
