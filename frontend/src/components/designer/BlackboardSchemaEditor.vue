<template>
  <div class="space-y-3 pt-3 border-t border-border">
    <div class="flex items-center gap-2">
      <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider flex-1">Blackboard Schema</h4>
      <button class="text-[10px] text-violet-400 hover:text-violet-300 transition-colors" @click="addField">+ Add Field</button>
    </div>
    
    <p v-if="!Object.keys(schema).length" class="text-[10px] text-text-muted italic">No fields defined.</p>
    
    <div v-for="(field, name) in schema" :key="name" class="space-y-2 p-2 rounded bg-surface-0 border border-border/40 relative group">
      <div class="flex gap-2">
        <!-- We use a local proxy for the name to handle the map key rename -->
        <input 
          class="input font-mono text-[10px] py-1 flex-1" 
          :value="name"
          placeholder="field_name"
          @change="renameField(name, $event.target.value)"
        />
        <select class="input text-[10px] py-1 w-20 shrink-0" v-model="field.type" @change="$emit('change')">
          <option>string</option>
          <option>number</option>
          <option>bool</option>
          <option>object</option>
        </select>
        <button class="text-text-muted hover:text-red-400 text-xs px-1" @click="removeField(name)">✕</button>
      </div>

      <div class="flex items-center gap-3">
        <label class="flex items-center gap-1.5 text-[9px] text-text-muted cursor-pointer hover:text-text transition-colors">
          <input type="checkbox" v-model="field.required" class="accent-violet-500" @change="$emit('change')" />
          <span>Mandatory Input</span>
        </label>
        
        <label class="flex items-center gap-1.5 text-[9px] text-text-muted cursor-pointer hover:text-text transition-colors">
          <input type="checkbox" v-model="field.is_output" class="accent-teal-500" @change="$emit('change')" />
          <span>Workflow Output</span>
        </label>
      </div>
    </div>
  </div>
</template>

<script setup>
const props = defineProps({
  schema: { type: Object, required: true }
})
const emit = defineEmits(['change'])

function addField() {
  const baseName = 'new_field'
  let name = baseName
  let counter = 1
  while (props.schema[name]) {
    name = `${baseName}_${counter++}`
  }
  props.schema[name] = { type: 'string', required: false, is_output: false }
  emit('change')
}

function removeField(name) {
  delete props.schema[name]
  emit('change')
}

function renameField(oldName, newName) {
  if (!newName || oldName === newName || props.schema[newName]) return
  const val = props.schema[oldName]
  delete props.schema[oldName]
  props.schema[newName] = val
  emit('change')
}
</script>
