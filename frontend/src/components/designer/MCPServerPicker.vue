<template>
  <div class="space-y-2">
    <!-- Loading -->
    <div v-if="loading" class="text-[10px] text-text-muted italic">Loading servers…</div>

    <!-- Empty registry -->
    <div v-else-if="!servers.length" class="text-[10px] text-text-muted italic">
      No MCP servers registered. Add one in
      <span class="text-indigo-400">Settings → MCP Servers</span>.
    </div>

    <!-- Server chips -->
    <div v-else class="flex flex-wrap gap-1.5">
      <button
        v-for="srv in servers"
        :key="srv.name"
        type="button"
        :title="srv.description"
        :class="[
          'flex items-center gap-1.5 px-2 py-1 rounded-md border text-[10px] font-medium transition-colors',
          isSelected(srv)
            ? 'bg-accent/20 border-accent/40 text-accent'
            : 'bg-surface-0 border-border text-text-muted hover:border-accent/40 hover:text-text',
        ]"
        @click="toggle(srv)"
      >
        <!-- availability dot -->
        <span :class="['w-1.5 h-1.5 rounded-full shrink-0', srv.env_var || srv.url ? 'bg-green-500' : 'bg-slate-600']" />
        {{ srv.name }}
        <!-- check mark when selected -->
        <svg v-if="isSelected(srv)" width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
      </button>
    </div>

    <!-- Resolved value preview -->
    <div v-if="modelValue" class="flex items-start gap-1.5 mt-1">
      <span class="text-[9px] text-text-muted shrink-0 mt-px font-mono uppercase tracking-wider">value</span>
      <span class="text-[9px] font-mono text-text-muted break-all leading-relaxed">{{ modelValue }}</span>
      <button
        class="shrink-0 text-text-muted hover:text-red-400 transition-colors ml-auto"
        title="Clear selection"
        @click="$emit('update:modelValue', '')"
      >
        <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '@/stores/auth'

const props = defineProps({
  modelValue: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

const servers = ref([])
const loading = ref(true)

onMounted(async () => {
  try {
    const { data } = await api.get('/mcp-servers')
    servers.value = data ?? []
  } catch {
    servers.value = []
  } finally {
    loading.value = false
  }
})

// Convert a server entry to the template reference string used in workflow YAML.
function serverValue(srv) {
  if (srv.env_var) return `{{ env.${srv.env_var} }}`
  if (srv.url)     return srv.url
  return srv.name   // fallback — shouldn't happen in practice
}

// Parse the current modelValue into individual URL/template tokens.
const selectedValues = computed(() =>
  props.modelValue
    ? props.modelValue.split(',').map(s => s.trim()).filter(Boolean)
    : []
)

function isSelected(srv) {
  return selectedValues.value.includes(serverValue(srv))
}

function toggle(srv) {
  const val = serverValue(srv)
  const current = selectedValues.value
  const next = current.includes(val)
    ? current.filter(v => v !== val)
    : [...current, val]
  emit('update:modelValue', next.join(', '))
}
</script>
