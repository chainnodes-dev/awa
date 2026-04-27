<template>
  <div class="space-y-1.5">
    <div class="text-[10px] font-semibold text-text-muted uppercase tracking-wider">{{ label }}</div>

    <div
      v-for="(port, i) in localPorts"
      :key="i"
      class="flex gap-1.5 items-center px-2 py-1.5 rounded bg-surface-0 border border-border"
    >
      <!-- Port name (child side) -->
      <select
        v-if="childOptions.length"
        class="input font-mono text-[10px] py-0.5 w-28 shrink-0"
        :value="port.name"
        @change="update(i, 'name', $event.target.value)"
      >
        <option value="">-- port --</option>
        <option v-for="opt in childOptions" :key="opt" :value="opt">{{ opt }}</option>
      </select>
      <input
        v-else
        class="input font-mono text-[10px] py-0.5 w-24 shrink-0"
        :value="port.name"
        placeholder="port"
        @input="update(i, 'name', $event.target.value)"
      />

      <span class="text-text-muted text-[10px] shrink-0">→</span>

      <!-- Blackboard field (parent side) -->
      <select
        v-if="parentOptions.length"
        class="input font-mono text-[10px] py-0.5 flex-1"
        :value="port.bbField"
        @change="update(i, 'bbField', $event.target.value)"
      >
        <option value="">-- bb field --</option>
        <option v-for="opt in parentOptions" :key="opt" :value="opt">{{ opt }}</option>
      </select>
      <input
        v-else
        class="input font-mono text-[10px] py-0.5 flex-1"
        :value="port.bbField"
        placeholder="bb.field"
        @input="update(i, 'bbField', $event.target.value)"
      />
      <button
        class="text-text-muted hover:text-red-400 text-xs shrink-0 px-1 transition-colors"
        @click="remove(i)"
      >✕</button>
    </div>

    <button
      class="w-full text-[10px] py-1 border border-dashed border-border hover:border-indigo-500/40
             text-text-muted hover:text-text-muted rounded transition-colors"
      @click="add"
    >
      + Add mapping
    </button>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  label:         { type: String, default: 'Mappings' },
  mappings:      { type: Array,  default: () => [] },
  childOptions:  { type: Array,  default: () => [] },
  parentOptions: { type: Array,  default: () => [] },
})
const emit = defineEmits(['update:mappings'])

// Work on a local copy so we don't mutate the prop directly.
const localPorts = ref(props.mappings.map(m => ({ ...m })))

watch(() => props.mappings, (v) => {
  localPorts.value = v.map(m => ({ ...m }))
}, { deep: true })

function update(i, key, value) {
  localPorts.value[i][key] = value
  emit('update:mappings', localPorts.value.map(p => ({ ...p })))
}

function remove(i) {
  localPorts.value.splice(i, 1)
  emit('update:mappings', localPorts.value.map(p => ({ ...p })))
}

function add() {
  localPorts.value.push({ name: '', bbField: '' })
  emit('update:mappings', localPorts.value.map(p => ({ ...p })))
}
</script>
