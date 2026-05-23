<template>
  <div :class="['space-y-3', !isNested && 'pt-3 border-t border-border']">
    <div v-if="!isNested" class="flex items-center gap-2">
      <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider flex-1">{{ title }}</h4>
      <button class="text-[10px] text-violet-400 hover:text-violet-300 transition-colors" @click="addField">+ Add Field</button>
    </div>
    
    <p v-if="!Object.keys(schema).length" class="text-[10px] text-text-muted italic px-2">No fields defined.</p>
    
    <div v-for="(field, name) in schema" :key="name" 
         :class="['space-y-2 p-2 rounded bg-surface-0 border border-border/40 relative group transition-all', isNested && 'ml-4 border-l-2 border-l-violet-500/30']">
      <div class="flex gap-2">
        <input 
          class="input font-mono text-[10px] py-1 flex-1" 
          :value="name"
          placeholder="field_name"
          @change="renameField(name, $event.target.value)"
        />
        <select class="input text-[10px] py-1 w-20 shrink-0" v-model="field.type" @change="onTypeChange(field)">
          <option>string</option>
          <option>number</option>
          <option>bool</option>
          <option>object</option>
          <option>list</option>
          <option>file</option>
        </select>
        <button class="text-text-muted hover:text-red-400 text-xs px-1" @click="removeField(name)">✕</button>
      </div>

      <div class="flex items-center gap-3">
        <label class="flex items-center gap-1.5 text-[9px] text-text-muted cursor-pointer hover:text-text transition-colors">
          <input type="checkbox" v-model="field.required" class="accent-violet-500" @change="$emit('change')" />
          <span>Mandatory</span>
        </label>
        
        <label v-if="!isNested" class="flex items-center gap-1.5 text-[9px] text-text-muted cursor-pointer hover:text-text transition-colors">
          <input type="checkbox" v-model="field.is_output" class="accent-teal-500" @change="$emit('change')" />
          <span>Output</span>
        </label>

        <div class="flex-1" />

        <!-- Pop-out for complex types -->
        <button 
          v-if="field.type === 'object' || field.type === 'list'"
          @click="$emit('open-popout', { name, field })"
          class="text-text-muted hover:text-accent transition-colors opacity-0 group-hover:opacity-100"
          title="Open in fullscreen focus mode"
        >
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/>
            <line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/>
          </svg>
        </button>
      </div>

      <!-- Recursive Nesting for Object -->
      <div v-if="field.type === 'object'" class="mt-2 space-y-2">
        <div class="flex items-center gap-2 px-1">
          <span class="text-[9px] font-bold text-text-muted uppercase tracking-tighter">Properties</span>
          <div class="flex-1 h-px bg-border/20" />
        </div>
        <BlackboardSchemaEditor 
          v-if="field.properties"
          :schema="field.properties" 
          :level="level + 1"
          is-nested 
          @change="$emit('change')"
          @open-popout="$emit('open-popout', $event)"
        />
        <button 
          @click="addNestedField(field, 'properties')"
          class="text-[9px] text-violet-400/70 hover:text-violet-300 py-1 px-2 border border-dashed border-violet-500/20 rounded w-full transition-all hover:bg-violet-500/5"
        >
           + Add Property
        </button>
      </div>

      <!-- Recursive Nesting for List Items -->
      <div v-if="field.type === 'list'" class="mt-2 space-y-2">
        <div class="flex items-center gap-2 px-1">
          <span class="text-[9px] font-bold text-text-muted uppercase tracking-tighter">Item Structure</span>
          <div class="flex-1 h-px bg-border/20" />
        </div>
        
        <div v-if="!field.items" class="p-2 rounded bg-surface-1/50 border border-dashed border-border/40 flex flex-col items-center gap-2">
          <span class="text-[9px] text-text-muted italic">Defaulting to list of strings</span>
          <button @click="makeListComplex(field)" class="text-[9px] text-accent hover:underline">Define complex item structure</button>
        </div>
        <BlackboardSchemaEditor 
          v-else-if="field.items.type === 'object'"
          :schema="field.items.properties || {}" 
          :level="level + 1"
          is-nested 
          @change="$emit('change')"
          @open-popout="$emit('open-popout', $event)"
        />
        <div v-else class="flex items-center gap-2 p-1.5 rounded bg-surface-1 border border-border/30">
           <span class="text-[10px] text-text-muted">Type:</span>
           <select class="input py-0.5 text-[10px] w-20" v-model="field.items.type" @change="$emit('change')">
             <option>string</option>
             <option>number</option>
             <option>bool</option>
             <option>object</option>
           </select>
           <button v-if="field.items.type === 'object'" @click="makeListComplex(field)" class="text-[10px] text-accent">Define properties</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { nextTick } from 'vue'

const props = defineProps({
  schema:   { type: Object, required: true },
  isNested: { type: Boolean, default: false },
  title:    { type: String, default: 'Blackboard Schema' },
  level:    { type: Number, default: 0 }
})

const emit = defineEmits(['change', 'open-popout'])

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

function addNestedField(parent, collectionName) {
  if (!parent[collectionName]) parent[collectionName] = {}
  const schema = parent[collectionName]
  
  const baseName = 'prop'
  let name = baseName
  let counter = 1
  while (schema[name]) {
    name = `${baseName}_${counter++}`
  }
  schema[name] = { type: 'string', required: false }
  emit('change')
}

function onTypeChange(field) {
  if (field.type === 'object' && !field.properties) {
    field.properties = {}
  }
  if (field.type === 'list' && !field.items) {
    field.items = { type: 'string' }
  }
  emit('change')
}

function makeListComplex(field) {
  field.items = { type: 'object', properties: {} }
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
