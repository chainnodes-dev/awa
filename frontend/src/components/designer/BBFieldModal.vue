<template>
  <div class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50" @click.self="$emit('close')">
    <div class="card w-80 p-5 space-y-3">
      <h3 class="font-semibold text-text text-sm">Add Blackboard Field</h3>
      <input class="input" placeholder="Field name" v-model="name" />
      <select class="input" v-model="type">
        <option v-for="t in ['string','number','bool','object']" :key="t" :value="t">{{ t }}</option>
      </select>
      <label class="flex items-center gap-2 text-sm text-text-muted cursor-pointer">
        <input type="checkbox" v-model="required" /> Required
      </label>
      <div class="flex gap-2 justify-end">
        <button class="btn-ghost" @click="$emit('close')">Cancel</button>
        <button class="btn-primary" @click="onAdd">Add</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
const emit = defineEmits(['close', 'add'])
const name     = ref('')
const type     = ref('string')
const required = ref(false)
function onAdd() {
  if (!name.value) return
  emit('add', { name: name.value, type: type.value, required: required.value })
}
</script>
