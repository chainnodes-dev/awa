<template>
  <div class="space-y-1.5">
    <label v-if="schema.title || name" class="block text-[10px] font-semibold text-text-muted uppercase tracking-wider">
      {{ schema.title || name }}
      <span v-if="required" class="text-red-500/80 ml-0.5">*</span>
    </label>

    <!-- Enum / Select -->
    <select
      v-if="schema.enum"
      :value="modelValue"
      @input="$emit('update:modelValue', $event.target.value)"
      class="input"
      :required="required"
    >
      <option v-if="!required" value="">(Select...)</option>
      <option v-for="opt in schema.enum" :key="opt" :value="opt">{{ opt }}</option>
    </select>

    <!-- Boolean / Checkbox -->
    <div v-else-if="schema.type === 'boolean'" class="flex items-center gap-2 py-1">
      <input
        type="checkbox"
        :checked="modelValue"
        @change="$emit('update:modelValue', $event.target.checked)"
        class="w-4 h-4 rounded border-border bg-surface-1 text-accent focus:ring-accent/30"
      />
      <span class="text-xs text-text-muted">{{ schema.description || 'Enable' }}</span>
    </div>

    <!-- Object / Nested Group -->
    <div v-else-if="schema.type === 'object'" class="space-y-4 p-3 rounded-xl bg-surface-2/30 border border-border/50 shadow-inner">
      <SchemaField
        v-for="(propSchema, propName) in (schema.properties || {})"
        :key="propName"
        :name="propName"
        :schema="propSchema"
        :required="Array.isArray(schema.required) && schema.required.includes(propName)"
        :modelValue="modelValue?.[propName]"
        @update:modelValue="v => updateObjectProperty(propName, v)"
      />
      <div v-if="!Object.keys(schema.properties || {}).length" class="space-y-2 py-2">
        <div class="flex items-center justify-between text-[10px] text-text-muted italic px-1">
          <span>No sub-properties defined</span>
          <span class="font-mono opacity-50">JSON Mode</span>
        </div>
        <textarea
          class="input font-mono text-xs min-h-[80px] w-full bg-surface-1/50"
          :value="typeof modelValue === 'object' ? JSON.stringify(modelValue, null, 2) : modelValue"
          @input="onJsonInput"
          placeholder='{ "key": "value" }'
        />
      </div>
    </div>

    <!-- Array / List -->
    <div v-else-if="schema.type === 'array'" class="space-y-3">
      <div v-for="(item, index) in (modelValue || [])" :key="index" class="relative group p-3 rounded-xl bg-surface-2/20 border border-border/30 transition-all hover:border-border/60">
        <div class="absolute right-2 top-2 z-10 opacity-0 group-hover:opacity-100 transition-opacity">
          <button @click="removeArrayItem(index)" class="p-1.5 text-red-500/60 hover:text-red-500 transition-colors bg-surface-1 rounded-lg shadow-lg border border-border/50" title="Remove item">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
              <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
            </svg>
          </button>
        </div>
        <SchemaField
          :schema="schema.items || { type: 'string' }"
          :modelValue="item"
          @update:modelValue="v => updateArrayItem(index, v)"
        />
      </div>
      <button 
        @click="addArrayItem" 
        class="btn-ghost text-[10px] gap-1.5 px-3 py-2 border border-dashed border-border/50 w-full justify-center hover:border-accent/40 hover:text-accent hover:bg-accent/5 transition-all rounded-xl"
      >
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Add {{ schema.items?.title || 'Item' }}
      </button>
    </div>

    <!-- Number -->
    <input
      v-else-if="schema.type === 'number' || schema.type === 'integer'"
      type="number"
      :value="modelValue"
      @input="$emit('update:modelValue', Number($event.target.value))"
      class="input"
      :placeholder="schema.description || ''"
      :required="required"
    />

    <!-- Text / Default -->
    <input
      v-else-if="schema.type !== 'file'"
      type="text"
      :value="modelValue"
      @input="$emit('update:modelValue', $event.target.value)"
      class="input"
      :placeholder="schema.description || ''"
      :required="required"
    />

    <!-- File Upload -->
    <div v-else class="space-y-2">
      <div v-if="modelValue?.name" class="flex items-center gap-2 p-2 rounded-lg bg-surface-1 border border-border/50">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-indigo-400">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
        </svg>
        <div class="flex-1 min-w-0">
          <div class="text-xs font-medium truncate">{{ modelValue.name }}</div>
          <div class="text-[10px] text-text-muted">{{ formatSize(modelValue.size) }}</div>
        </div>
        <button @click="$emit('update:modelValue', null)" class="text-text-muted hover:text-red-400 p-1">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>
      <div v-else class="relative">
        <input
          type="file"
          class="hidden"
          ref="fileInput"
          @change="handleFileUpload"
        />
        <button 
          @click="$refs.fileInput.click()"
          class="btn-ghost w-full justify-center border-dashed border-2 py-4 text-xs gap-2"
          :disabled="uploading"
        >
          <svg v-if="!uploading" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/>
          </svg>
          <svg v-else class="animate-spin h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
             <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
             <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          {{ uploading ? 'Uploading...' : 'Click to upload PDF' }}
        </button>
      </div>
      <p v-if="uploadError" class="text-[10px] text-red-400 mt-1">{{ uploadError }}</p>
    </div>

    <p v-if="schema.description && !['boolean', 'object', 'array'].includes(schema.type)" class="text-[10px] text-text-muted italic">
      {{ schema.description }}
    </p>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { api } from '@/stores/auth'
const props = defineProps({
  name: String,
  schema: { type: Object, required: true },
  modelValue: [String, Number, Boolean, Array, Object],
  required: Boolean
})

const emit = defineEmits(['update:modelValue'])

function addArrayItem() {
  const current = Array.isArray(props.modelValue) ? [...props.modelValue] : []
  
  // Initialize with appropriate empty value based on item type
  let newItem = ''
  const itemType = props.schema.items?.type
  if (itemType === 'object') newItem = {}
  else if (itemType === 'number' || itemType === 'integer') newItem = 0
  else if (itemType === 'boolean') newItem = false
  else if (itemType === 'array') newItem = []
  
  current.push(newItem)
  emit('update:modelValue', current)
}

function removeArrayItem(index) {
  const current = Array.isArray(props.modelValue) ? [...props.modelValue] : []
  current.splice(index, 1)
  emit('update:modelValue', current)
}

function updateArrayItem(index, value) {
  const current = Array.isArray(props.modelValue) ? [...props.modelValue] : []
  current[index] = value
  emit('update:modelValue', current)
}

function updateObjectProperty(propName, value) {
  const current = props.modelValue && typeof props.modelValue === 'object' && !Array.isArray(props.modelValue) 
    ? { ...props.modelValue } 
    : {}
  current[propName] = value
  emit('update:modelValue', current)
}

function onJsonInput(e) {
  try {
    const val = JSON.parse(e.target.value)
    emit('update:modelValue', val)
  } catch (err) {
    // While typing, emit the raw string if it's not valid JSON yet
    // or just let it be. Usually it's better to only emit valid objects.
  }
}

const uploading = ref(false)
const uploadError = ref('')

async function handleFileUpload(e) {
  const file = e.target.files[0]
  if (!file) return

  uploading.value = true
  uploadError.value = ''

  try {
    const formData = new FormData()
    formData.append('file', file)
    
    const { data } = await api.post('/uploads', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
    
    emit('update:modelValue', data)
  } catch (err) {
    uploadError.value = 'Upload failed: ' + (err.response?.data?.error || err.message)
  } finally {
    uploading.value = false
    e.target.value = '' // Reset input
  }
}

function formatSize(bytes) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}
</script>
