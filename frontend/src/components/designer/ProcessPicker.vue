<template>
  <div class="space-y-1.5">
    <!-- Selector row -->
    <div class="relative" ref="root">
      <button
        type="button"
        class="input w-full flex items-center gap-2 text-left"
        @click="open = !open"
      >
        <svg class="text-violet-400/60 shrink-0" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
        </svg>
        <span v-if="selected" class="font-mono text-xs text-slate-200 flex-1 truncate">{{ selected.metadata.name }}</span>
        <span v-else class="text-xs text-slate-600 flex-1">— select a process —</span>
        <svg class="text-slate-600 shrink-0 transition-transform" :class="open && 'rotate-180'" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="6 9 12 15 18 9"/>
        </svg>
      </button>

      <!-- Dropdown -->
      <div
        v-if="open"
        class="absolute z-50 left-0 right-0 top-full mt-1 bg-surface-1 border border-border rounded-lg shadow-xl overflow-hidden"
      >
        <!-- Search -->
        <div class="p-2 border-b border-border">
          <input
            ref="searchInput"
            v-model="query"
            class="input text-xs py-1 w-full"
            placeholder="Search reusable processes…"
            @keydown.esc="open = false"
            @keydown.enter.prevent="selectFirst"
          />
        </div>

        <div class="max-h-52 overflow-y-auto">
          <div v-if="loading" class="px-3 py-4 text-xs text-slate-600 text-center">Loading…</div>
          <div v-else-if="!filtered.length" class="px-3 py-4 text-xs text-slate-600 text-center italic">No reusable processes found</div>
          <button
            v-for="p in filtered"
            :key="p.metadata.name"
            type="button"
            class="w-full text-left px-3 py-2.5 hover:bg-white/5 transition-colors border-b border-border/40 last:border-0"
            @click="pick(p)"
          >
            <div class="flex items-center gap-2 mb-0.5">
              <span class="font-mono text-xs text-violet-400 font-medium">{{ p.metadata.name }}</span>
              <span v-if="p.inputs?.length || p.outputs?.length" class="text-[9px] text-slate-600">
                {{ p.inputs?.length || 0 }}↓ {{ p.outputs?.length || 0 }}↑
              </span>
            </div>
            <p v-if="p.metadata.description" class="text-[10px] text-slate-500 leading-snug line-clamp-1">{{ p.metadata.description }}</p>
          </button>
        </div>
      </div>
    </div>

    <!-- Selected process summary + clear -->
    <template v-if="selected">
      <div class="rounded-lg bg-violet-500/5 border border-violet-500/20 px-3 py-2.5 space-y-1.5">
        <!-- Port info -->
        <div class="flex gap-4">
          <div v-if="selected.inputs?.length" class="space-y-0.5">
            <div class="text-[9px] text-slate-600 uppercase tracking-wider font-semibold">Inputs</div>
            <div v-for="p in selected.inputs" :key="p.name" class="flex items-center gap-1.5 text-[10px]">
              <span class="font-mono text-indigo-300">{{ p.name }}</span>
              <span class="text-slate-600">{{ p.type }}</span>
              <span v-if="p.required" class="text-red-400/70">*</span>
            </div>
          </div>
          <div v-if="selected.outputs?.length" class="space-y-0.5">
            <div class="text-[9px] text-slate-600 uppercase tracking-wider font-semibold">Outputs</div>
            <div v-for="p in selected.outputs" :key="p.name" class="flex items-center gap-1.5 text-[10px]">
              <span class="font-mono text-teal-300">{{ p.name }}</span>
              <span class="text-slate-600">{{ p.type }}</span>
            </div>
          </div>
          <div v-if="!selected.inputs?.length && !selected.outputs?.length" class="text-[10px] text-slate-600 italic">No ports declared</div>
        </div>
        <!-- Workflow ref + clear -->
        <div class="flex items-center gap-2 pt-0.5">
          <span class="text-[10px] text-slate-500 font-mono flex-1 truncate">Ver: {{ selected.metadata.version }}</span>
          <button type="button" class="text-[10px] text-slate-600 hover:text-red-400 transition-colors" @click="clear">Clear</button>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { api } from '@/stores/auth'

const props = defineProps({
  modelValue: { type: String, default: '' }, // the resolved process name
})
const emit = defineEmits(['update:modelValue', 'process-selected'])

const open        = ref(false)
const query       = ref('')
const loading     = ref(false)
const processes   = ref([])
const root        = ref(null)
const searchInput = ref(null)

const selected = computed(() =>
  processes.value.find(p => p.metadata.name === props.modelValue) ?? null
)

const filtered = computed(() => {
  const q = query.value.toLowerCase()
  if (!q) return processes.value
  return processes.value.filter(p =>
    p.metadata.name.toLowerCase().includes(q) || p.metadata.description?.toLowerCase().includes(q)
  )
})

async function fetchProcesses() {
  loading.value = true
  try {
    const { data } = await api.get('/workflows?reusable=true')
    processes.value = data ?? []
  } catch { /* non-fatal */ } finally {
    loading.value = false
  }
}

function pick(p) {
  open.value  = false
  query.value = ''
  emit('update:modelValue', p.metadata.name)
  emit('process-selected', p)
}

function clear() {
  emit('update:modelValue', '')
  emit('process-selected', null)
}

function selectFirst() {
  if (filtered.value.length) pick(filtered.value[0])
}

// Focus search when dropdown opens
watch(open, async (v) => {
  if (v) {
    await nextTick()
    searchInput.value?.focus()
  }
})

// Close on outside click
function onClickOutside(e) {
  if (root.value && !root.value.contains(e.target)) open.value = false
}
onMounted(() => {
  document.addEventListener('mousedown', onClickOutside)
  fetchProcesses()
})
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onClickOutside)
})
</script>
