<template>
  <div class="blackboard-editor flex flex-col h-full overflow-hidden">
    <div class="px-3 py-2 border-b border-border flex items-center justify-between shrink-0 bg-surface-2">
      <div class="flex items-center gap-2">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-indigo-400">
          <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
          <line x1="3" y1="9" x2="21" y2="9"/>
          <line x1="9" y1="21" x2="9" y2="9"/>
        </svg>
        <span class="text-[10px] font-bold text-text uppercase tracking-widest">Blackboard Context</span>
      </div>
      <div v-if="isValid === false" class="text-[9px] text-red-500 font-medium">
        Invalid JSON
      </div>
    </div>
    <div class="flex-1 min-h-0 bg-surface-0">
      <CodeEditor
        v-model="jsonValue"
        language="json"
        height="100%"
        class="border-none rounded-none"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import CodeEditor from '@/components/designer/CodeEditor.vue'

const props = defineProps({
  modelValue: { type: Object, default: () => ({}) }
})

const emit = defineEmits(['update:modelValue'])

const jsonValue = ref('')
const isValid = ref(true)

// Initialize from props
onMounted(() => {
  jsonValue.value = JSON.stringify(props.modelValue, null, 2)
})

// Update local text when props change (only if not active editing or different)
watch(() => props.modelValue, (newVal) => {
  try {
    const current = JSON.parse(jsonValue.value)
    if (JSON.stringify(current) !== JSON.stringify(newVal)) {
      jsonValue.value = JSON.stringify(newVal, null, 2)
    }
  } catch (e) {
    jsonValue.value = JSON.stringify(newVal, null, 2)
  }
}, { deep: true })

// Emit changes when valid JSON is typed
watch(jsonValue, (newVal) => {
  try {
    const parsed = JSON.parse(newVal)
    isValid.value = true
    emit('update:modelValue', parsed)
  } catch (e) {
    isValid.value = false
  }
})
</script>

<style scoped>
.blackboard-editor {
  @apply border border-border rounded-lg overflow-hidden transition-all duration-200;
}
.blackboard-editor:focus-within {
  @apply border-indigo-500/50 shadow-lg shadow-indigo-500/5;
}
</style>
