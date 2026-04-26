<template>
  <div class="space-y-3 pt-3 border-t border-border">
    <div class="flex items-center gap-2">
      <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider flex-1">{{ title }}</h4>
      <button class="text-[10px] text-indigo-400 hover:text-indigo-300 transition-colors" @click="addPort">+ Add</button>
    </div>
    <p v-if="!ports.length" class="text-[10px] text-text-muted italic">No {{ title.toLowerCase() }} defined.</p>
    <div v-for="(p, i) in ports" :key="i" class="space-y-1.5 p-2 rounded bg-surface-0 border border-border/40 relative group">
      <div class="flex gap-2">
        <input class="input font-mono text-[10px] py-1 flex-1" v-model="p.name" placeholder="name" @input="$emit('change')" />
        <select class="input text-[10px] py-1 w-20 shrink-0" v-model="p.type" @change="$emit('change')">
          <option>string</option><option>number</option><option>bool</option><option>object</option>
        </select>
        <button class="text-text-muted hover:text-red-400 text-xs px-1" @click="removePort(i)">✕</button>
      </div>
      <input class="input text-[10px] py-1 w-full" v-model="p.description" placeholder="description" @input="$emit('change')" />
      <label v-if="showRequired" class="flex items-center gap-1.5 text-[9px] text-text-muted cursor-pointer">
        <input type="checkbox" v-model="p.required" class="accent-indigo-500" @change="$emit('change')" />
        <span>Mark as mandatory input</span>
      </label>
    </div>
  </div>
</template>

<script setup>
const props = defineProps({
  title:    { type: String, required: true },
  ports:    { type: Array,  required: true },
  showRequired: { type: Boolean, default: false }
})
const emit = defineEmits(['change'])

function addPort() {
  props.ports.push({ name: '', type: 'string', description: '', required: false })
  emit('change')
}
function removePort(i) {
  props.ports.splice(i, 1)
  emit('change')
}
</script>
