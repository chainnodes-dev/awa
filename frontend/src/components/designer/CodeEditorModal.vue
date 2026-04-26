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
        <!-- Icon — violet for text/skill, teal for code -->
        <svg v-if="language === 'text'" :class="['shrink-0', iconClass]" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
        </svg>
        <svg v-else :class="['shrink-0', iconClass]" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
        </svg>

        <span class="text-sm font-semibold text-text flex-1">
          {{ resolvedTitle }}
          <span :class="['ml-2 text-[10px] font-mono text-text-muted uppercase tracking-widest px-1.5 py-0.5 rounded', badgeClass]">
            {{ language === 'text' ? 'plain text' : 'JavaScript' }}
          </span>
        </span>

        <span class="text-[10px] text-text-muted mr-2 hidden sm:block">Esc to close</span>

        <button class="btn-primary text-xs px-4 py-1.5" @click="$emit('close')">Done</button>
      </div>

      <!-- Editor fills remaining height -->
      <div class="flex-1 overflow-hidden p-4">
        <CodeEditor
          :model-value="modelValue"
          :language="language"
          :bb-schema="bbSchema"
          :height="editorHeight"
          @update:model-value="$emit('update:modelValue', $event)"
        />
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import CodeEditor from './CodeEditor.vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  stateName:  { type: String, default: '' },
  title:      { type: String, default: '' },
  language:   { type: String, default: 'javascript' },
  bbSchema:   { type: Object, default: () => ({}) },
})
defineEmits(['update:modelValue', 'close'])

const overlay = ref(null)

const editorHeight = computed(() => 'calc(100vh - 88px)')

const resolvedTitle = computed(() =>
  props.title || props.stateName || (props.language === 'text' ? 'Process Description' : 'Code')
)

const iconClass   = computed(() => props.language === 'text' ? 'text-violet-400' : 'text-teal-400')
const badgeClass  = computed(() => props.language === 'text' ? 'bg-violet-500/10' : 'bg-teal-500/10')

onMounted(() => {
  overlay.value?.focus()
})
</script>
