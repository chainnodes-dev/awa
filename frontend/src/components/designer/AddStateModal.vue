<template>
  <div class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50" @click.self="$emit('close')">
    <div class="card w-80 p-5 space-y-3">
      <h3 class="font-semibold text-text text-sm">Add State</h3>
      <input class="input" placeholder="State name (e.g. VALIDATING)" v-model="name" @keydown.enter="onAdd" />
      <select class="input" v-model="type">
        <option value="initial">initial</option>
        <option value="terminal">terminal</option>
        <optgroup label="──────────────"></optgroup>
        <option value="prompt">prompt</option>
        <option value="script">script</option>
        <option value="code">code</option>
        <optgroup label="──────────────"></optgroup>
        <option value="subprocess" :disabled="!entStore.hasFeature('subprocesses')">subprocess (Pro)</option>
        <optgroup label="──────────────"></optgroup>
        <option value="hitl">hitl</option>
        <option value="wait">wait</option>
      </select>
      <div class="flex gap-2 justify-end">
        <button class="btn-ghost" @click="$emit('close')">Cancel</button>
        <button class="btn-primary" @click="onAdd">Add</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useEnterpriseStore } from '@/stores/enterprise'
const emit = defineEmits(['close', 'add'])
const entStore = useEnterpriseStore()
const name = ref('')
const type = ref('prompt')
function onAdd() {
  if (!name.value) return
  emit('add', { name: name.value, type: type.value })
}
</script>
