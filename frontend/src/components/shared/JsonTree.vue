<template>
  <div class="json-tree font-mono text-[11px] select-text">
    <!-- Primitive values -->
    <div v-if="!isObject" :class="['inline-block py-0.5 px-1 rounded bg-surface-2/20', valueClass]">
      {{ formattedPrimitive }}
    </div>

    <!-- Objects / Arrays -->
    <div v-else class="flex flex-col">
      <div 
        @click="collapsed = !collapsed"
        class="flex items-center gap-1 cursor-pointer hover:bg-white/5 py-0.5 px-1 rounded transition-colors group w-fit select-none"
      >
        <svg 
          width="8" 
          height="8" 
          viewBox="0 0 24 24" 
          fill="none" 
          stroke="currentColor" 
          stroke-width="3"
          class="text-text-muted transition-transform duration-150"
          :class="{ '-rotate-90': collapsed }"
        >
          <polyline points="6 9 12 15 18 9"/>
        </svg>
        <span class="text-indigo-400 font-bold text-[10px]">{{ prefix }}</span>
        <span v-if="collapsed" class="text-text-muted text-[10px] font-medium px-1 bg-white/5 rounded border border-border/20">
          {{ childCount }} {{ childCount === 1 ? 'item' : 'items' }}
        </span>
        <span class="text-indigo-400 font-bold text-[10px]">{{ collapsed ? suffix : '' }}</span>
      </div>

      <div v-if="!collapsed" class="pl-4 border-l border-border/40 ml-2.5 py-0.5 space-y-1">
        <div v-for="{ key, val } in children" :key="key" class="flex flex-col sm:flex-row sm:items-start gap-1">
          <span class="text-text-muted text-[10px] font-semibold shrink-0 select-all">{{ key }}:</span>
          <JsonTree :data="val" />
        </div>
      </div>

      <div v-if="!collapsed" class="text-indigo-400 font-bold text-[10px] pl-1.5 select-none">{{ suffix }}</div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  data: { type: [Object, Array, String, Number, Boolean, null], default: null }
})

const collapsed = ref(false)

const isObject = computed(() => props.data !== null && typeof props.data === 'object')
const isArray = computed(() => Array.isArray(props.data))

const prefix = computed(() => isArray.value ? '[' : '{')
const suffix = computed(() => isArray.value ? ']' : '}')

const children = computed(() => {
  if (!isObject.value) return []
  if (isArray.value) {
    return props.data.map((v, i) => ({ key: String(i), val: v }))
  }
  return Object.entries(props.data).map(([k, v]) => ({ key: k, val: v }))
})

const childCount = computed(() => children.value.length)

const formattedPrimitive = computed(() => {
  if (props.data === null) return 'null'
  if (props.data === undefined) return 'undefined'
  if (typeof props.data === 'string') return `"${props.data}"`
  return String(props.data)
})

const valueClass = computed(() => {
  if (props.data === null || props.data === undefined) return 'text-slate-500 font-semibold'
  if (typeof props.data === 'number') return 'text-amber-400 font-medium'
  if (typeof props.data === 'boolean') return 'text-teal-400 font-medium'
  if (typeof props.data === 'string') return 'text-emerald-400'
  return 'text-text'
})
</script>
