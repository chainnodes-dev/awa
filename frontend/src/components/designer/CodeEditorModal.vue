<template>
  <Teleport to="body">
    <div
      class="fixed inset-0 z-[200] bg-black/80 backdrop-blur-sm flex flex-col"
      @keydown.esc="$emit('close')"
      tabindex="-1"
      ref="overlay"
    >
      <!-- Header bar -->
      <div class="flex items-center gap-3 px-5 py-3 border-b border-border bg-surface shrink-0">
        <!-- Code icon -->
        <svg class="text-teal-400 shrink-0" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
        </svg>
        <span class="text-sm font-semibold text-slate-200 flex-1">
          {{ stateName || 'Code' }}
          <span class="ml-2 text-[10px] font-mono text-slate-600 uppercase tracking-widest bg-teal-500/10 px-1.5 py-0.5 rounded">JavaScript</span>
        </span>

        <!-- Keyboard hint -->
        <span class="text-[10px] text-slate-600 mr-2 hidden sm:block">Esc to close</span>

        <!-- Done button -->
        <button
          class="btn-primary text-xs px-4 py-1.5"
          @click="$emit('close')"
        >Done</button>
      </div>

      <!-- Editor fills remaining height -->
      <div class="flex-1 overflow-hidden p-4">
        <CodeEditor
          :model-value="modelValue"
          :bb-schema="bbSchema"
          :height="editorHeight"
          @update:model-value="$emit('update:modelValue', $event)"
        />
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import CodeEditor from './CodeEditor.vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  stateName:  { type: String, default: '' },
  bbSchema:   { type: Object, default: () => ({}) },
})
defineEmits(['update:modelValue', 'close'])

const overlay = ref(null)

// Dynamically fill the space below the header (~56px) minus padding (32px)
const editorHeight = computed(() => 'calc(100vh - 88px)')

onMounted(() => {
  overlay.value?.focus()
})
</script>
