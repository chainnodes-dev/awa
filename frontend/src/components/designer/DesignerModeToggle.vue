<template>
  <div
    class="flex bg-surface-0 rounded-lg p-0.5 border border-border shrink-0"
    role="group"
    aria-label="Designer mode"
  >
    <button
      type="button"
      @click="select('ai')"
      :class="[
        'btn py-1 px-2.5 text-xs gap-1.5 whitespace-nowrap transition-colors',
        modelValue === 'ai' ? 'bg-surface-2 text-violet-300 font-medium shadow-sm' : 'text-text-muted hover:text-text'
      ]"
      title="AI Designer — Vertical layout (Canvas top, AI Chat bottom)"
    >
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="shrink-0">
        <rect x="3" y="3" width="18" height="18" rx="2"/>
        <line x1="3" y1="14" x2="21" y2="14"/>
      </svg>
      <span>AI · Vertical</span>
    </button>

    <button
      type="button"
      @click="select('classic')"
      :class="[
        'btn py-1 px-2.5 text-xs gap-1.5 whitespace-nowrap transition-colors',
        modelValue === 'classic' ? 'bg-surface-2 text-text font-medium shadow-sm' : 'text-text-muted hover:text-text'
      ]"
      title="Designer — Horizontal layout (Canvas left, Inspector / YAML right)"
    >
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="shrink-0">
        <rect x="3" y="3" width="18" height="18" rx="2"/>
        <line x1="14" y1="3" x2="14" y2="21"/>
      </svg>
      <span>Designer · Horizontal</span>
    </button>
  </div>
</template>

<script setup>
const props = defineProps({
  modelValue: {
    type: String,
    required: true,
    validator: v => ['ai', 'classic'].includes(v),
  },
})

const emit = defineEmits(['switch'])

function select(target) {
  if (target !== props.modelValue) {
    emit('switch', target)
  }
}
</script>

