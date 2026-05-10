<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Top bar -->
    <header class="flex items-center gap-4 px-6 py-4 border-b border-border shrink-0">
      <h1 class="text-base font-semibold text-text">MCP Servers</h1>
      <p class="text-xs text-text-muted">Manage and monitor Model Context Protocol servers</p>
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

      <div v-if="loading" class="text-text-muted text-sm py-12 text-center">Loading...</div>

      <div v-else-if="!servers.length" class="card p-12 text-center">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" class="mx-auto mb-4 text-slate-700">
          <rect x="2" y="2" width="20" height="8" rx="2"/>
          <rect x="2" y="14" width="20" height="8" rx="2"/>
          <circle cx="6" cy="6" r="1" fill="currentColor"/>
          <circle cx="6" cy="18" r="1" fill="currentColor"/>
        </svg>
        <p class="text-text-muted text-sm mb-1">No MCP servers registered</p>
        <p class="text-text-muted text-xs mb-4">Add an MCP server to make its tools available to workflow agents.</p>
        <button v-if="isAdmin" @click="openModal(null)" class="btn-primary text-xs">Add Server</button>
      </div>

      <!-- Server cards -->
      <div v-else class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div v-for="srv in servers" :key="srv.id || srv.name" class="card p-5 space-y-3 relative overflow-hidden group">
          <!-- Background decoration -->
          <div class="absolute -right-4 -top-4 opacity-[0.02] group-hover:opacity-[0.05] transition-opacity">
            <svg width="120" height="120" viewBox="0 0 24 24" fill="currentColor"><rect x="2" y="2" width="20" height="20" rx="2"/></svg>
          </div>

          <!-- Header row -->
          <div class="flex items-start gap-3 relative z-10">
            <div class="mt-0.5 shrink-0" :title="statusTitle(srv)">
              <div v-if="healthStatus[srv.id] === undefined" class="w-2.5 h-2.5 rounded-full bg-slate-600 ring-2 ring-slate-600/30"/>
              <div v-else-if="healthStatus[srv.id] === 'checking'" class="w-2.5 h-2.5 rounded-full bg-blue-400 ring-2 ring-blue-400/30 animate-pulse"/>
              <div v-else-if="healthStatus[srv.id] === 'online'" class="w-2.5 h-2.5 rounded-full bg-green-400 ring-2 ring-green-400/30 shadow-[0_0_8px_rgba(74,222,128,0.4)]"/>
              <div v-else-if="healthStatus[srv.id] === 'offline'" class="w-2.5 h-2.5 rounded-full bg-red-400 ring-2 ring-red-400/30"/>
              <div v-else class="w-2.5 h-2.5 rounded-full bg-amber-400 ring-2 ring-amber-400/30"/>
            </div>

            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-1">
                <span class="font-display text-sm font-bold text-white tracking-tight uppercase">{{ srv.name }}</span>
                <span v-if="healthLatency[srv.id]" class="text-[10px] font-mono text-green-400/70">{{ healthLatency[srv.id] }}ms</span>
              </div>
              <p class="text-xs text-gray-400 leading-relaxed line-clamp-2">{{ srv.description || 'No capabilities discovered yet' }}</p>
            </div>

            <!-- Actions -->
            <div class="flex gap-1 shrink-0">
              <button @click="pingServer(srv)" :disabled="healthStatus[srv.id] === 'checking'" class="btn-ghost p-1.5" title="Check availability">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
                </svg>
              </button>
              <template v-if="isAdmin">
                <button @click="openModal(srv)" class="btn-ghost p-1.5 text-indigo-400 hover:text-indigo-300">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                </button>
                <button @click="confirmDelete(srv)" class="btn-ghost p-1.5 text-red-400 hover:text-red-300">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><line x1="10" y1="11" x2="10" y2="17"/><line x1="14" y1="11" x2="14" y2="17"/></svg>
                </button>
              </template>
            </div>
          </div>

          <!-- Connection details -->
          <div class="flex flex-wrap gap-x-4 gap-y-2 pt-2 border-t border-white/5">
            <div v-if="srv.transport === 'stdio'" class="flex items-center gap-1.5 text-[10px] text-gray-500">
              <span class="px-1.5 py-0.5 rounded-full bg-indigo-500/10 text-indigo-400 font-bold uppercase tracking-widest text-[8px]">Local</span>
              <span class="font-mono text-gray-300">{{ srv.command }} {{ srv.args?.join(' ') }}</span>
            </div>
            <div v-else-if="srv.url" class="flex items-center gap-1.5 text-[10px] text-gray-500">
              <span class="px-1.5 py-0.5 rounded-full bg-blue-500/10 text-blue-400 font-bold uppercase tracking-widest text-[8px]">Remote</span>
              <span class="font-mono text-gray-300">{{ srv.url }}</span>
            </div>
            
            <!-- Env vars list -->
            <div v-if="srv.env_vars && Object.keys(srv.env_vars).length" class="flex flex-wrap gap-1">
              <div v-for="(v, k) in srv.env_vars" :key="k" class="flex items-center gap-1 px-1.5 py-0.5 rounded bg-white/5 border border-white/5">
                <span class="text-[8px] text-gray-500 font-mono tracking-tighter">{{ k }}</span>
                <span class="w-1 h-1 rounded-full bg-indigo-500/50"/>
                <span class="text-[8px] text-indigo-400 font-mono">{{ v ? '••••' : 'ENV' }}</span>
              </div>
            </div>
          </div>

          <!-- Health error message -->
          <div v-if="healthError[srv.id]" class="mt-2 text-[10px] text-red-400 bg-red-500/5 rounded px-3 py-2 font-mono break-all border border-red-500/10">
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
  if (!srv.id) return
  healthStatus[srv.id] = 'checking'
  delete healthError[srv.id]
  delete healthLatency[srv.id]
  try {
    const { data } = await api.post('/mcp-servers/' + srv.id + '/ping')
    healthStatus[srv.id]  = data.status
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
  modalServer.value = srv
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
import { defineComponent, h, ref, reactive } from 'vue'
import { api } from '@/stores/auth'

// ── MCPServerModal ────────────────────────────────────────────────────────────
export const MCPServerModal = defineComponent({
  props: { server: { default: null } },
  emits: ['close', 'saved'],
  setup(p, { emit }) {
    const isEdit = !!p.server?.id
    
    // Convert map to array for easier form management
    const envList = ref([])
    if (p.server?.env_vars) {
      envList.value = Object.entries(p.server.env_vars).map(([key, val]) => ({ key, val }))
    }
    if (envList.value.length === 0) envList.value.push({ key: '', val: '' })

    const form = ref({
      name:        p.server?.name        ?? '',
      transport:   p.server?.transport   ?? 'stdio',
      url:         p.server?.url         ?? '',
      command:     p.server?.command     ?? '',
      args:        p.server?.args?.join(' ') ?? '',
      description: p.server?.description ?? '',
    })

    const saving      = ref(false)
    const discovering = ref(false)
    const error       = ref('')

    function addEnv() { envList.value.push({ key: '', val: '' }) }
    function removeEnv(idx) { envList.value.splice(idx, 1) }

    function getEnvMap() {
      const m = {}
      envList.value.forEach(e => { if (e.key) m[e.key] = e.val })
      return m
    }

    async function discover() {
      if (form.value.transport === 'sse' && !form.value.url) { 
        error.value = 'Enter a URL first'; return 
      }
      if (form.value.transport === 'stdio' && !form.value.command) {
        error.value = 'Enter an executable first'; return
      }
      
      discovering.value = true; error.value = ''
      try {
        const payload = {
          ...form.value,
          args: form.value.args.match(/(?:[^\s"]+|"[^"]*")+/g)?.map(s => s.replace(/^"|"$/g, '')) || [],
          env_vars: getEnvMap()
        }
        const { data } = await api.post('/mcp-servers/discover', payload)
        if (data.error) error.value = data.error
        else form.value.description = data.description
      } catch (e) {
        error.value = e.response?.data?.error ?? e.message
      } finally {
        discovering.value = false
      }
    }

    async function save() {
      if (!form.value.name) { error.value = 'Name is required'; return }
      saving.value = true; error.value = ''
      try {
        const payload = {
          ...form.value,
          args: form.value.args.match(/(?:[^\s"]+|"[^"]*")+/g)?.map(s => s.replace(/^"|"$/g, '')) || [],
          env_vars: getEnvMap()
        }
        let res
        if (isEdit) res = await api.put('/mcp-servers/' + p.server.id, payload)
        else res = await api.post('/mcp-servers', payload)
        emit('saved', res.data)
      } catch (e) {
        error.value = e.response?.data?.error ?? e.message
      } finally {
        saving.value = false
      }
    }

    const inputClass = 'input text-sm font-mono'

    return () => h('div', {
      class: 'fixed inset-0 bg-black/80 backdrop-blur-md flex items-center justify-center z-[100] p-4',
    }, [
      h('div', { class: 'card w-full max-w-xl p-8 space-y-6 shadow-2xl border-white/10 max-h-[90vh] overflow-y-auto' }, [
        h('div', { class: 'flex justify-between items-center' }, [
          h('h3', { class: 'text-lg font-bold text-white uppercase tracking-tight' }, isEdit ? 'Configure MCP Server' : 'Register New MCP Server'),
          h('button', { class: 'text-gray-500 hover:text-white', onClick: () => emit('close') }, '\u2715')
        ]),

        // Basic Info
        h('div', { class: 'grid grid-cols-2 gap-4' }, [
          h('div', { class: 'space-y-1.5' }, [
            h('label', { class: 'text-[10px] uppercase font-bold text-gray-500 tracking-widest' }, 'Logical Name'),
            h('input', {
              class: inputClass,
              value: form.value.name,
              placeholder: 'e.g. slack-main',
              onInput: e => form.value.name = e.target.value,
            }),
          ]),
          h('div', { class: 'space-y-1.5' }, [
            h('label', { class: 'text-[10px] uppercase font-bold text-gray-500 tracking-widest' }, 'Transport'),
            h('div', { class: 'flex p-1 bg-white/5 rounded-lg border border-white/5' }, [
              ['stdio', 'Local'], ['sse', 'Remote']
            ].map(([val, label]) => h('button', {
              class: 'flex-1 py-1.5 text-[10px] font-bold uppercase rounded transition-all ' + 
                     (form.value.transport === val ? 'bg-indigo-600 text-white shadow-lg shadow-indigo-500/20' : 'text-gray-500 hover:text-gray-300'),
              onClick: () => form.value.transport = val,
            }, label))),
          ]),
        ]),

        // Transport Specifics
        form.value.transport === 'stdio' ? [
          h('div', { class: 'space-y-1.5' }, [
            h('label', { class: 'text-[10px] uppercase font-bold text-gray-500 tracking-widest' }, 'Executable / Command'),
            h('input', {
              class: inputClass,
              value: form.value.command,
              placeholder: 'npx',
              onInput: e => form.value.command = e.target.value,
            }),
          ]),
          h('div', { class: 'space-y-1.5' }, [
            h('label', { class: 'text-[10px] uppercase font-bold text-gray-500 tracking-widest' }, 'Arguments'),
            h('input', {
              class: inputClass,
              value: form.value.args,
              placeholder: '-y @modelcontextprotocol/server-slack',
              onInput: e => form.value.args = e.target.value,
            }),
          ]),
        ] : [
          h('div', { class: 'space-y-1.5' }, [
            h('label', { class: 'text-[10px] uppercase font-bold text-gray-500 tracking-widest' }, 'SSE Endpoint URL'),
            h('input', {
              class: inputClass,
              value: form.value.url,
              placeholder: 'https://mcp.example.com/sse',
              onInput: e => form.value.url = e.target.value,
            }),
          ]),
        ],

        // Environment Variables
        h('div', { class: 'space-y-3' }, [
          h('div', { class: 'flex justify-between items-center' }, [
            h('label', { class: 'text-[10px] uppercase font-bold text-gray-500 tracking-widest' }, 'Environment Variables / Secrets'),
            h('button', { 
              class: 'text-[10px] font-bold text-indigo-400 hover:text-indigo-300',
              onClick: addEnv
            }, '+ Add Variable')
          ]),
          h('div', { class: 'space-y-2' }, envList.value.map((e, idx) => h('div', { class: 'flex gap-2' }, [
            h('input', {
              class: 'input text-xs font-mono flex-1',
              value: e.key,
              placeholder: 'KEY',
              onInput: v => e.key = v.target.value,
            }),
            h('input', {
              class: 'input text-xs font-mono flex-1',
              type: 'password',
              value: e.val,
              placeholder: 'VALUE',
              onInput: v => e.val = v.target.value,
            }),
            h('button', { 
              class: 'p-2 text-gray-600 hover:text-red-400',
              onClick: () => removeEnv(idx)
            }, '\u2715')
          ])))
        ]),

        // Capabilities / Discovery
        h('div', { class: 'space-y-1.5' }, [
          h('div', { class: 'flex justify-between items-end' }, [
            h('label', { class: 'text-[10px] uppercase font-bold text-gray-500 tracking-widest' }, 'Discovered Capabilities'),
            h('button', {
              class: 'btn-ghost text-[10px] px-3 h-7 gap-1.5 ' + (discovering.value ? 'animate-pulse' : ''),
              onClick: discover,
              disabled: saving.value || discovering.value,
            }, discovering.value ? 'Syncing...' : 'Discover Tools'),
          ]),
          h('div', {
            class: 'p-3 rounded-lg bg-white/3 border border-white/5 text-xs text-gray-400 leading-relaxed font-mono min-h-[60px] max-h-32 overflow-y-auto'
          }, form.value.description || 'No tools discovered yet.')
        ]),

        error.value && h('div', { class: 'p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-xs' }, error.value),

        h('div', { class: 'flex gap-3 pt-4' }, [
          h('button', { class: 'flex-1 btn-ghost py-3', onClick: () => emit('close') }, 'Cancel'),
          h('button', { 
            class: 'flex-[2] btn-primary py-3 font-bold uppercase tracking-widest text-xs', 
            onClick: save, 
            disabled: saving.value || discovering.value 
          }, saving.value ? 'Saving...' : 'Register Connection'),
        ]),
      ]),
    ])
  },
})
</script>
