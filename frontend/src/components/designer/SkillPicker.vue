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
        <span v-if="selected" class="font-mono text-xs text-slate-200 flex-1 truncate">{{ selected.name }}</span>
        <span v-else class="text-xs text-slate-600 flex-1">— select a skill —</span>
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
            placeholder="Search skills…"
            @keydown.esc="open = false"
            @keydown.enter.prevent="selectFirst"
          />
        </div>

        <div class="max-h-52 overflow-y-auto">
          <div v-if="loading" class="px-3 py-4 text-xs text-slate-600 text-center">Loading…</div>
          <div v-else-if="!filtered.length" class="px-3 py-4 text-xs text-slate-600 text-center italic">No skills found</div>
          <button
            v-for="sk in filtered"
            :key="sk.name"
            type="button"
            class="w-full text-left px-3 py-2.5 hover:bg-white/5 transition-colors border-b border-border/40 last:border-0"
            @click="pick(sk)"
          >
            <div class="flex items-center gap-2 mb-0.5">
              <span class="font-mono text-xs text-violet-400 font-medium">{{ sk.name }}</span>
              <span v-if="sk.inputs?.length || sk.outputs?.length" class="text-[9px] text-slate-600">
                {{ sk.inputs?.length || 0 }}↓ {{ sk.outputs?.length || 0 }}↑
              </span>
            </div>
            <p v-if="sk.description" class="text-[10px] text-slate-500 leading-snug line-clamp-1">{{ sk.description }}</p>
          </button>
        </div>

        <!-- Footer link -->
        <div class="px-3 py-2 border-t border-border bg-surface-0">
          <RouterLink to="/skills" class="text-[10px] text-indigo-400/70 hover:text-indigo-400 transition-colors" @click="open = false">
            Manage skill registry ↗
          </RouterLink>
        </div>
      </div>
    </div>

    <!-- Selected skill summary + clear -->
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
          <span v-if="selected.workflow_ref" class="text-[10px] text-slate-500 font-mono flex-1 truncate">→ {{ selected.workflow_ref }}</span>
          <span v-else class="text-[10px] text-amber-500/70 flex-1 italic">no workflow linked</span>
          <button type="button" class="text-[10px] text-slate-600 hover:text-red-400 transition-colors" @click="clear">Clear</button>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { RouterLink } from 'vue-router'
import { api } from '@/stores/auth'

const props = defineProps({
  modelValue: { type: String, default: '' }, // the resolved skill_ref (workflow_ref)
})
const emit = defineEmits(['update:modelValue', 'skill-selected'])

const open        = ref(false)
const query       = ref('')
const loading     = ref(false)
const skills      = ref([])
const root        = ref(null)
const searchInput = ref(null)

const selected = computed(() =>
  skills.value.find(sk => sk.workflow_ref === props.modelValue || sk.name === props.modelValue) ?? null
)

const filtered = computed(() => {
  const q = query.value.toLowerCase()
  if (!q) return skills.value
  return skills.value.filter(sk =>
    sk.name.toLowerCase().includes(q) || sk.description?.toLowerCase().includes(q)
  )
})

async function fetchSkills() {
  loading.value = true
  try {
    const { data } = await api.get('/skills')
    skills.value = data ?? []
  } catch { /* non-fatal */ } finally {
    loading.value = false
  }
}

function pick(sk) {
  open.value  = false
  query.value = ''
  // Prefer workflow_ref as the ref value; fall back to skill name
  const ref = sk.workflow_ref || sk.name
  emit('update:modelValue', ref)
  emit('skill-selected', sk)
}

function clear() {
  emit('update:modelValue', '')
  emit('skill-selected', null)
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
  fetchSkills()
})
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onClickOutside)
})
</script>
