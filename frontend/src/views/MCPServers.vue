<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Top bar -->
    <header class="flex items-center gap-4 px-6 py-4 border-b border-border shrink-0">
      <h1 class="text-base font-semibold text-slate-200">MCP Servers</h1>
      <p class="text-xs text-slate-500">Manage and monitor Model Context Protocol servers</p>
      <div class="flex-1"/>
      <button @click="pingAll" :disabled="pingingAll" class="btn-ghost text-xs gap-1.5 border border-border">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
        </svg>
        {{ pingingAll ? 'Checking...' : 'Check All' }}
      </button>
      <button v-if="isAdmin" @click="openModal(null)" class="btn-primary text-xs gap-1.5">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Add Server
      </button>
    </header>

    <div class="flex-1 overflow-y-auto p-6">
      <!-- Non-admin notice -->
      <div v-if="!isAdmin" class="px-4 py-3 rounded-lg bg-amber-500/10 border border-amber-500/30 text-amber-400 text-sm mb-6">
        MCP server management requires admin privileges. You can view registered servers and their status.
      </div>

      <div v-if="loading" class="text-slate-600 text-sm py-12 text-center">Loading...</div>

      <div v-else-if="!servers.length" class="card p-12 text-center">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" class="mx-auto mb-4 text-slate-700">
          <rect x="2" y="2" width="20" height="8" rx="2"/>
          <rect x="2" y="14" width="20" height="8" rx="2"/>
          <circle cx="6" cy="6" r="1" fill="currentColor"/>
          <circle cx="6" cy="18" r="1" fill="currentColor"/>
        </svg>
        <p class="text-slate-500 text-sm mb-1">No MCP servers registered</p>
        <p class="text-slate-600 text-xs mb-4">Add an MCP server to make its tools available to workflow agents.</p>
        <button v-if="isAdmin" @click="openModal(null)" class="btn-primary text-xs">Add Server</button>
      </div>

      <!-- Server cards -->
      <div v-else class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div v-for="srv in servers" :key="srv.id || srv.name" class="card p-5 space-y-3">
          <!-- Header row -->
          <div class="flex items-start gap-3">
            <!-- Status indicator -->
            <div class="mt-0.5 shrink-0" :title="statusTitle(srv)">
              <div v-if="healthStatus[srv.id] === undefined" class="w-2.5 h-2.5 rounded-full bg-slate-600 ring-2 ring-slate-600/30" title="Not checked"/>
              <div v-else-if="healthStatus[srv.id] === 'checking'" class="w-2.5 h-2.5 rounded-full bg-blue-400 ring-2 ring-blue-400/30 animate-pulse"/>
              <div v-else-if="healthStatus[srv.id] === 'online'" class="w-2.5 h-2.5 rounded-full bg-green-400 ring-2 ring-green-400/30"/>
              <div v-else-if="healthStatus[srv.id] === 'offline'" class="w-2.5 h-2.5 rounded-full bg-red-400 ring-2 ring-red-400/30"/>
              <div v-else class="w-2.5 h-2.5 rounded-full bg-amber-400 ring-2 ring-amber-400/30"/>
            </div>

            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-0.5">
                <span class="font-mono text-sm font-medium text-indigo-400">{{ srv.name }}</span>
                <span v-if="healthLatency[srv.id]" class="text-[10px] text-slate-600">{{ healthLatency[srv.id] }}ms</span>
              </div>
              <p class="text-xs text-slate-500 leading-relaxed">{{ srv.description || 'No capabilities discovered yet' }}</p>
            </div>

            <!-- Actions -->
            <div class="flex gap-1 shrink-0">
              <button @click="pingServer(srv)" :disabled="healthStatus[srv.id] === 'checking'" class="btn-ghost text-xs py-1 px-2" title="Check availability">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
                </svg>
              </button>
              <template v-if="isAdmin">
                <button @click="openModal(srv)" class="btn-ghost text-xs py-1 px-2">Edit</button>
                <button @click="confirmDelete(srv)" class="btn-danger text-xs py-1 px-2">Delete</button>
              </template>
            </div>
          </div>

          <!-- Connection details -->
          <div class="flex flex-wrap gap-x-4 gap-y-1 text-[11px]">
            <div v-if="srv.url" class="flex items-center gap-1.5 text-slate-400">
              <span class="text-slate-600">URL</span>
              <span class="font-mono">{{ srv.url }}</span>
            </div>
            <div v-if="srv.env_var" class="flex items-center gap-1.5 text-slate-400">
              <span class="text-slate-600">ENV</span>
              <span class="font-mono" v-text="'$' + srv.env_var"></span>
            </div>
            <div v-if="srv.created_by" class="flex items-center gap-1.5 text-slate-400">
              <span class="text-slate-600">By</span>
              <span>{{ srv.created_by }}</span>
            </div>
          </div>

          <!-- Health error message -->
          <div v-if="healthError[srv.id]" class="text-[11px] text-red-400/80 bg-red-500/5 rounded px-2.5 py-1.5 font-mono break-all">
            {{ healthError[srv.id] }}
          </div>
        </div>
      </div>
    </div>

    <!-- Add/Edit Modal -->
    <MCPServerModal v-if="modalServer !== undefined" :server="modalServer" @close="modalServer = undefined" @saved="onSaved" />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useAuthStore, api } from '@/stores/auth'

const authStore = useAuthStore()
const isAdmin = computed(() => authStore.hasRole('admin'))

const servers = ref([])
const loading = ref(true)
const pingingAll = ref(false)
const modalServer = ref(undefined)  // undefined = closed; null = add; object = edit

// Health state per server ID
const healthStatus  = reactive({})  // 'checking' | 'online' | 'offline' | 'unknown' | 'error'
const healthLatency = reactive({})  // ms number
const healthError   = reactive({})  // error string

async function fetchServers() {
  loading.value = true
  try {
    const { data } = await api.get('/mcp-servers')
    servers.value = data ?? []
  } catch { /* non-fatal */ } finally {
    loading.value = false
  }
}

async function pingServer(srv) {
  if (!srv.id) {
    healthStatus[srv.name] = 'unknown'
    healthError[srv.name] = 'Server not in DB (static YAML entry) — cannot ping'
    return
  }
  healthStatus[srv.id] = 'checking'
  delete healthError[srv.id]
  delete healthLatency[srv.id]
  try {
    const { data } = await api.post('/mcp-servers/' + srv.id + '/ping')
    healthStatus[srv.id]  = data.status  // 'online', 'offline', 'unknown', 'error'
    healthLatency[srv.id] = data.latency_ms
    if (data.error) healthError[srv.id] = data.error
  } catch (e) {
    healthStatus[srv.id] = 'error'
    healthError[srv.id] = e.response?.data?.error ?? e.message
  }
}

async function pingAll() {
  pingingAll.value = true
  await Promise.allSettled(servers.value.map(srv => pingServer(srv)))
  pingingAll.value = false
}

function statusTitle(srv) {
  const id = srv.id || srv.name
  const s = healthStatus[id]
  if (!s) return 'Not checked'
  if (s === 'checking') return 'Checking...'
  if (s === 'online') return 'Online (' + (healthLatency[id] || '?') + 'ms)'
  if (s === 'offline') return 'Offline'
  return 'Unknown'
}

function openModal(srv) {
  modalServer.value = srv  // null → add, object → edit
}

async function confirmDelete(srv) {
  if (!srv.id) return
  if (!confirm('Remove MCP server "' + srv.name + '"?')) return
  try {
    await api.delete('/mcp-servers/' + srv.id)
    servers.value = servers.value.filter(s => s.id !== srv.id)
  } catch (e) {
    alert(e.response?.data?.error ?? e.message)
  }
}

function onSaved(srv) {
  modalServer.value = undefined
  const idx = servers.value.findIndex(s => s.id === srv.id)
  if (idx !== -1) servers.value[idx] = srv
  else servers.value.push(srv)
}

onMounted(() => {
  fetchServers()
})
</script>

<script>
import { defineComponent, h, ref } from 'vue'
import { api } from '@/stores/auth'

// ── MCPServerModal ────────────────────────────────────────────────────────────
export const MCPServerModal = defineComponent({
  props: { server: { default: null } },
  emits: ['close', 'saved'],
  setup(p, { emit }) {
    const isEdit = !!p.server?.id
    const form = ref({
      name:        p.server?.name        ?? '',
      url:         p.server?.url         ?? '',
      env_var:     p.server?.env_var     ?? '',
      description: p.server?.description ?? '',
    })
    const saving      = ref(false)
    const discovering = ref(false)
    const error       = ref('')

    async function discover() {
      if (!form.value.url) { error.value = 'Enter a URL first to discover tools'; return }
      discovering.value = true; error.value = ''
      try {
        const { data } = await api.post('/mcp-servers/discover', { url: form.value.url })
        if (data.error) {
          error.value = data.error
        } else {
          form.value.description = data.description
        }
      } catch (e) {
        error.value = e.response?.data?.error ?? e.message
      } finally {
        discovering.value = false
      }
    }

    async function save() {
      if (!form.value.name) { error.value = 'Name is required'; return }
      if (!form.value.url && !form.value.env_var) { error.value = 'Provide either a URL or an env var key'; return }
      saving.value = true; error.value = ''
      try {
        let res
        if (isEdit) {
          res = await api.put('/mcp-servers/' + p.server.id, {
            url: form.value.url,
            env_var: form.value.env_var,
            description: form.value.description,
          })
        } else {
          res = await api.post('/mcp-servers', form.value)
        }
        emit('saved', res.data)
      } catch (e) {
        error.value = e.response?.data?.error ?? e.message
      } finally {
        saving.value = false
      }
    }

    const inputClass = 'input text-sm'
    const busy = () => saving.value || discovering.value

    return () => h('div', {
      class: 'fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50',
      onClick: e => e.target === e.currentTarget && emit('close'),
    }, [
      h('div', { class: 'card w-[520px] p-6 space-y-4' }, [
        h('h3', { class: 'font-semibold text-slate-200' }, isEdit ? 'Edit MCP Server' : 'Register MCP Server'),

        h('div', { class: 'space-y-1.5' }, [
          h('label', { class: 'text-xs text-slate-500' }, 'Name *'),
          h('input', {
            class: inputClass + ' font-mono',
            value: form.value.name,
            placeholder: 'invoice-parser',
            disabled: isEdit,
            onInput: e => form.value.name = e.target.value,
          }),
          !isEdit && h('p', { class: 'text-[10px] text-slate-600' }, 'Logical identifier used in workflow YAML agent configs.'),
        ]),

        // URL + Discover button
        h('div', { class: 'space-y-1.5' }, [
          h('label', { class: 'text-xs text-slate-500' }, 'URL'),
          h('div', { class: 'flex gap-2' }, [
            h('input', {
              class: inputClass + ' font-mono flex-1',
              value: form.value.url,
              placeholder: 'http://invoice-parser:8090',
              onInput: e => form.value.url = e.target.value,
            }),
            h('button', {
              class: 'btn-ghost text-xs border border-border px-3 shrink-0 gap-1.5 ' + (discovering.value ? 'animate-pulse' : ''),
              onClick: discover,
              disabled: busy() || !form.value.url,
              title: 'Connect to the server and discover its tools',
            }, [
              h('svg', { width: '12', height: '12', viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2' }, [
                h('circle', { cx: '11', cy: '11', r: '8' }),
                h('line', { x1: '21', y1: '21', x2: '16.65', y2: '16.65' }),
              ]),
              discovering.value ? 'Discovering...' : 'Discover',
            ]),
          ]),
        ]),

        h('div', { class: 'space-y-1.5' }, [
          h('label', { class: 'text-xs text-slate-500' }, 'Env Var Key'),
          h('input', {
            class: inputClass + ' font-mono',
            value: form.value.env_var,
            placeholder: 'MCP_INVOICE_PARSER_URL',
            onInput: e => form.value.env_var = e.target.value,
          }),
          h('p', { class: 'text-[10px] text-slate-600' }, 'Provide URL or env var (or both). Env vars are resolved at runtime by the worker.'),
        ]),

        // Description — read-only, populated by Discover
        h('div', { class: 'space-y-1.5' }, [
          h('div', { class: 'flex items-center gap-2' }, [
            h('label', { class: 'text-xs text-slate-500' }, 'Capabilities'),
            h('span', { class: 'text-[10px] text-slate-600' }, '(auto-populated via Discover)'),
          ]),
          form.value.description
            ? h('div', {
                class: 'px-3 py-2.5 rounded-lg bg-white/3 border border-border text-xs text-slate-400 leading-relaxed whitespace-pre-wrap max-h-32 overflow-y-auto',
              }, form.value.description)
            : h('div', {
                class: 'px-3 py-2.5 rounded-lg bg-white/3 border border-border/50 text-xs text-slate-600 italic',
              }, 'No capabilities discovered yet. Enter a URL and click Discover.'),
        ]),

        error.value && h('p', { class: 'text-red-400 text-xs' }, error.value),

        h('div', { class: 'flex gap-3 justify-end' }, [
          h('button', { class: 'btn-ghost', onClick: () => emit('close'), disabled: busy() }, 'Cancel'),
          h('button', { class: 'btn-primary', onClick: save, disabled: busy() },
            saving.value ? 'Saving...' : (isEdit ? 'Save Changes' : 'Register')
          ),
        ]),
      ]),
    ])
  },
})
</script>
