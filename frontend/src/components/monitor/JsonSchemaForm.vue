<template>
  <div class="json-schema-form space-y-6">
    <div v-if="title || description" class="space-y-1">
      <h3 v-if="title" class="text-sm font-semibold text-text">{{ title }}</h3>
      <p v-if="description" class="text-xs text-text-muted">{{ description }}</p>
    </div>

    <div class="grid grid-cols-1 gap-5">
      <SchemaField
        v-for="(propSchema, propName) in properties"
        :key="propName"
        :name="propName"
        :schema="propSchema"
        :required="isRequired(propName)"
        v-model="formData[propName]"
      />
    </div>

    <div v-if="showActions" class="flex items-center justify-end gap-3 pt-4 border-t border-border">
      <button v-if="showCancel" @click="$emit('cancel')" class="btn-ghost">
        Cancel
      </button>
      <button @click="submit" class="btn-primary px-6">
        {{ submitLabel }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { reactive, computed, onMounted, watch } from 'vue'
import SchemaField from './SchemaField.vue'

const props = defineProps({
  schema: { type: Object, default: () => ({ type: 'object', properties: {} }) },
  initialData: { type: Object, default: () => ({}) },
  submitLabel: { type: String, default: 'Submit' },
  showActions: { type: Boolean, default: false },
  showCancel: { type: Boolean, default: false }
})

const emit = defineEmits(['submit', 'cancel', 'update:modelValue'])

const formData = reactive({})

const title = computed(() => props.schema.title)
const description = computed(() => props.schema.description)
const properties = computed(() => props.schema.properties || {})

function isRequired(name) {
  return Array.isArray(props.schema.required) && props.schema.required.includes(name)
}

function initializeForm() {
  const propsObj = properties.value
  for (const key in propsObj) {
    formData[key] = props.initialData[key] ?? propsObj[key].default ?? undefined
    
    // Ensure booleans default to false if not specified
    if (propsObj[key].type === 'boolean' && formData[key] === undefined) {
      formData[key] = false
    }
  }
}

onMounted(initializeForm)
watch(() => props.schema, initializeForm, { deep: true })
watch(() => props.initialData, initializeForm, { deep: true })

watch(formData, (newVal) => {
  // Emit changes to parent for bi-directional sync
  emit('update:modelValue', { ...newVal })
}, { deep: true })

function submit() {
  emit('submit', { ...formData })
}

// Expose internal state for parent components that want to use their own buttons
defineExpose({
  formData,
  submit
})
</script>

<style scoped>
.json-schema-form {
  @apply text-left;
}
</style>
