<template>
  <div class="flex flex-col h-full">
    <div class="px-4 py-3 border-b border-border flex items-center gap-2">
      <span class="text-xs font-semibold text-text-muted uppercase tracking-wider">Blackboard</span>
      <span class="ml-auto text-xs text-text-muted">{{ fieldCount }} fields</span>
    </div>

    <div class="flex-1 overflow-y-auto p-3 space-y-1">
      <div
        v-for="[key, value] in fields" :key="key"
        class="group flex items-start gap-2 px-2 py-1.5 rounded-lg hover:bg-white/5 transition-colors"
        :class="{ 'bg-accent/10 border border-accent/20': isNew(key) }"
      >
        <span class="text-xs font-mono text-text-muted shrink-0 mt-0.5 w-28 truncate">{{ key }}</span>
        <span class="text-xs font-mono break-all" :class="valueColor(value)">
          {{ formatValue(value) }}
        </span>
      </div>

      <div v-if="fieldCount === 0" class="text-text-muted text-xs text-center py-8">
        No blackboard data yet
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'

const props = defineProps({
  blackboard: { type: Object, default: () => ({}) }
})

const recentKeys = ref(new Set())
let clearTimer = null

// Track recently-updated keys for highlight
watch(() => props.blackboard, (newBB, oldBB) => {
  if (!oldBB) return
  for (const key of Object.keys(newBB)) {
    if (newBB[key] !== oldBB?.[key]) {
      recentKeys.value.add(key)
    }
  }
  clearTimer && clearTimeout(clearTimer)
  clearTimer = setTimeout(() => recentKeys.value.clear(), 2000)
}, { deep: true })

const fields = computed(() =>
  Object.entries(props.blackboard ?? {}).sort(([a], [b]) => a.localeCompare(b))
)

const fieldCount = computed(() => fields.value.length)

function isNew(key) { return recentKeys.value.has(key) }

function formatValue(v) {
  if (v === null || v === undefined) return 'null'
  if (typeof v === 'object') return JSON.stringify(v, null, 1)
  return String(v)
}

function valueColor(v) {
  if (v === null || v === undefined) return 'text-text-muted'
  if (typeof v === 'boolean') return v ? 'text-green-400' : 'text-red-400'
  if (typeof v === 'number') return 'text-blue-400'
  if (typeof v === 'string' && v === '') return 'text-text-muted'
  return 'text-text'
}
</script>
