<template>
  <div class="flex flex-col h-full">
    <div class="px-4 py-3 border-b border-border flex items-center gap-2">
      <span class="text-xs font-semibold text-text-muted uppercase tracking-wider">Blackboard</span>
      <span class="ml-auto text-xs text-text-muted">{{ fieldCount }} fields</span>
    </div>

    <div class="flex-1 overflow-y-auto p-3 space-y-1">
      <div
        v-for="[key, value] in fields" :key="key"
        class="group flex flex-col gap-1 px-2 py-2 rounded-lg transition-all border border-transparent"
        :class="{ 'bg-accent/5 border-accent/10 shadow-sm': isChanged(key) }"
      >
        <div class="flex items-center gap-2">
          <span class="text-[10px] font-mono text-text-muted shrink-0 w-24 truncate">{{ key }}</span>
          <span v-if="isChanged(key)" class="text-[9px] font-bold text-accent uppercase tracking-tighter shrink-0 animate-pulse">Updated</span>
          <span class="flex-1"/>
          <span class="text-[10px] font-mono opacity-40 shrink-0">{{ typeof value }}</span>
        </div>
        
        <div class="flex flex-col">
          <div v-if="isChanged(key) && previousBlackboard?.[key] !== undefined" class="text-[10px] font-mono text-text-muted/40 line-through mb-0.5">
            {{ formatValue(previousBlackboard[key]) }}
          </div>
          <div class="text-xs font-mono break-all leading-relaxed" :class="valueColor(value)">
            {{ formatValue(value) }}
          </div>
        </div>
      </div>

      <div v-if="fieldCount === 0" class="text-text-muted text-xs text-center py-8">
        No blackboard data yet
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  blackboard: { type: Object, default: () => ({}) },
  previousBlackboard: { type: Object, default: null }
})

const fields = computed(() =>
  Object.entries(props.blackboard ?? {}).sort(([a], [b]) => a.localeCompare(b))
)

const fieldCount = computed(() => fields.value.length)

const changedKeys = computed(() => {
  if (!props.previousBlackboard) return new Set()
  const keys = new Set()
  for (const k of Object.keys(props.blackboard)) {
    if (JSON.stringify(props.blackboard[k]) !== JSON.stringify(props.previousBlackboard[k])) {
      keys.add(k)
    }
  }
  return keys
})

function isChanged(key) { return changedKeys.value.has(key) }

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
