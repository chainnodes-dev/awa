<template>
  <div class="flex flex-col h-full overflow-hidden bg-surface-0">
    <!-- Top bar -->
    <header class="flex items-center gap-4 px-6 py-4 border-b border-border shrink-0">
      <div class="flex flex-col">
        <h1 class="text-base font-bold text-text tracking-tight uppercase">MCP Infrastructure</h1>
        <p class="text-[10px] text-text-muted font-bold uppercase tracking-widest opacity-60">Model Context Protocol</p>
      </div>
      
      <div class="flex-1 px-4">
        <!-- Tab Switcher -->
        <div class="flex p-1 bg-surface-2 rounded-xl border border-border w-fit">
          <button @click="activeTab = 'registered'" 
                  :class="['px-6 py-1.5 text-[11px] font-bold uppercase rounded-lg transition-all', 
                           activeTab === 'registered' ? 'bg-indigo-600 text-white shadow-lg shadow-indigo-500/20' : 'text-text-muted hover:text-text']">
            Registered Servers
          </button>
          <button @click="activeTab = 'marketplace'" 
                  :class="['px-6 py-1.5 text-[11px] font-bold uppercase rounded-lg transition-all flex items-center gap-2', 
                           activeTab === 'marketplace' ? 'bg-indigo-600 text-white shadow-lg shadow-indigo-500/20' : 'text-text-muted hover:text-text']">
            Marketplace
            <span class="px-1.5 py-0.5 rounded-full bg-amber-500/20 text-amber-500 text-[8px]">CURATED</span>
          </button>
        </div>
      </div>

      <div class="flex items-center gap-3">
        <button v-if="activeTab === 'registered'" @click="pingAll" :disabled="pingingAll" class="btn-ghost h-9 px-4 text-xs gap-2 border border-border bg-surface-1">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
          {{ pingingAll ? 'Checking...' : 'Check All' }}
        </button>
        <button v-if="isAdmin" @click="openModal(null)" class="btn-primary h-9 px-4 text-xs gap-2 shadow-xl shadow-indigo-500/20">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          Add Custom
        </button>
      </div>
    </header>

    <!-- Content Area -->
    <main class="flex-1 overflow-y-auto p-6 scroll-smooth">
      
      <!-- Registered Servers Tab -->
      <div v-if="activeTab === 'registered'" class="space-y-6">
        <div v-if="loading" class="flex flex-col items-center justify-center py-20 gap-4">
          <div class="w-8 h-8 border-4 border-indigo-500/20 border-t-indigo-500 rounded-full animate-spin"/>
          <p class="text-xs text-text-muted font-mono uppercase tracking-widest animate-pulse">Syncing Registry...</p>
        </div>

        <div v-else-if="!servers.length" class="flex flex-col items-center justify-center py-24 text-center border-2 border-dashed border-border rounded-2xl bg-surface-1/50">
          <div class="w-16 h-16 rounded-full bg-surface-2 flex items-center justify-center mb-6 border border-border shadow-inner">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" class="text-text-muted opacity-40"><rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/><circle cx="6" cy="6" r="1" fill="currentColor"/><circle cx="6" cy="18" r="1" fill="currentColor"/></svg>
          </div>
          <h3 class="text-lg font-bold text-text mb-2 uppercase tracking-tight">No Active Servers</h3>
          <p class="text-sm text-text-muted mb-8 max-w-sm">Register a custom MCP server or browse the marketplace to enable tools for your workflows.</p>
          <button @click="activeTab = 'marketplace'" class="btn-primary px-8">Browse Marketplace</button>
        </div>

        <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          <div v-for="srv in servers" :key="srv.id" class="card group hover:border-indigo-500/50 transition-all duration-300">
            <div class="p-5 space-y-4">
              <!-- Header -->
              <div class="flex items-start justify-between">
                <div class="flex items-center gap-3">
                  <div class="relative">
                    <div v-if="healthStatus[srv.id] === 'online'" class="w-2.5 h-2.5 rounded-full bg-green-500 shadow-[0_0_10px_rgba(34,197,94,0.6)]"/>
                    <div v-else-if="healthStatus[srv.id] === 'offline'" class="w-2.5 h-2.5 rounded-full bg-red-500"/>
                    <div v-else-if="healthStatus[srv.id] === 'checking'" class="w-2.5 h-2.5 rounded-full bg-indigo-500 animate-pulse"/>
                    <div v-else class="w-2.5 h-2.5 rounded-full bg-slate-600"/>
                  </div>
                  <h4 class="font-display font-bold text-sm text-text uppercase tracking-tight truncate max-w-[150px]">{{ srv.name }}</h4>
                </div>
                <div class="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button @click="pingServer(srv)" class="p-1.5 rounded hover:bg-surface-2 text-text-muted hover:text-text" title="Refresh Status">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
                  </button>
                  <button v-if="isAdmin" @click="openModal(srv)" class="p-1.5 rounded hover:bg-surface-2 text-indigo-500" title="Configure Server">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
                  </button>
                </div>
              </div>

              <!-- Description -->
              <p class="text-xs text-text-muted leading-relaxed line-clamp-2 min-h-[32px]">
                {{ srv.description || 'No capabilities registered yet.' }}
              </p>

              <!-- Transport Badge -->
              <div class="flex items-center justify-between pt-4 border-t border-border/40">
                <div class="flex items-center gap-2">
                  <span v-if="srv.transport === 'stdio'" class="px-2 py-0.5 rounded bg-indigo-500/10 text-indigo-500 text-[9px] font-bold uppercase tracking-wider">LOCAL / STDIO</span>
                  <span v-else class="px-2 py-0.5 rounded bg-blue-500/10 text-blue-500 text-[9px] font-bold uppercase tracking-wider">REMOTE / SSE</span>
                  <span v-if="healthLatency[srv.id]" class="text-[10px] font-mono text-green-500 font-bold">{{ healthLatency[srv.id] }}ms</span>
                </div>
                <a v-if="srv.doc_url" :href="srv.doc_url" target="_blank" class="text-text-muted hover:text-indigo-500">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
                </a>
              </div>

              <!-- Error state -->
              <div v-if="healthError[srv.id]" class="mt-2 p-2 rounded bg-red-500/5 border border-red-500/10 text-[9px] text-red-500 font-mono truncate" :title="healthError[srv.id]">
                {{ healthError[srv.id] }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Marketplace Tab -->
      <div v-if="activeTab === 'marketplace'" class="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
        <!-- Hero Section -->
        <div class="relative p-10 rounded-3xl overflow-hidden bg-gradient-to-br from-indigo-900/40 to-slate-900 border border-indigo-500/20 shadow-2xl">
          <div class="absolute inset-0 bg-[radial-gradient(circle_at_top_right,_var(--tw-gradient-stops))] from-indigo-500/10 via-transparent to-transparent opacity-50"/>
          <div class="relative z-10 max-w-2xl">
            <h2 class="text-3xl font-display font-bold text-white mb-4 tracking-tight">Curated Marketplace</h2>
            <p class="text-indigo-100/70 text-base leading-relaxed">
              Explore "Known Good" MCP servers verified by the Phaxa team. These configurations are tested and ready to use in your enterprise workflows.
            </p>
          </div>
        </div>

        <div v-if="marketLoading" class="flex flex-col items-center justify-center py-20 gap-4">
          <div class="w-8 h-8 border-4 border-amber-500/20 border-t-amber-500 rounded-full animate-spin"/>
          <p class="text-xs text-text-muted font-mono uppercase tracking-widest animate-pulse">Syncing Curated List...</p>
        </div>

        <div v-else class="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div v-for="item in marketplace" :key="item.id" 
               class="flex gap-6 p-6 rounded-2xl bg-surface-1 border border-border hover:border-indigo-500/40 hover:shadow-xl hover:shadow-indigo-500/5 transition-all duration-300 group">
            
            <!-- Icon / Brand -->
            <div class="w-16 h-16 shrink-0 rounded-2xl bg-surface-2 flex items-center justify-center border border-border shadow-sm group-hover:scale-105 transition-transform">
              <svg v-if="item.id === 'sec-edgar'" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-amber-500"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
              <svg v-else-if="item.id === 'brave-search'" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-indigo-500"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
              <svg v-else-if="item.id === 'slack'" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-pink-500"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
              <svg v-else-if="item.id === 'github'" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-text"><path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"/></svg>
              <div v-else class="text-2xl font-bold text-indigo-500">{{ item.name[0] }}</div>
            </div>

            <!-- Info -->
            <div class="flex-1 space-y-3">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <h3 class="font-display font-bold text-lg text-text tracking-tight">{{ item.name }}</h3>
                  <span v-if="item.is_verified" class="px-1.5 py-0.5 rounded bg-green-500/10 text-green-500 text-[8px] font-bold uppercase tracking-tighter">Verified</span>
                </div>
                <span class="text-[10px] font-bold text-text-muted uppercase tracking-widest">{{ item.category }}</span>
              </div>
              <p class="text-sm text-text-muted leading-relaxed">{{ item.description }}</p>
              
              <div class="flex items-center gap-4 pt-2">
                <button @click="installFromMarket(item)" 
                        :disabled="isInstalled(item.name)"
                        class="btn-primary px-6 py-2 h-9 text-[11px] uppercase tracking-widest font-bold disabled:bg-surface-2 disabled:text-text-muted disabled:border-border">
                  {{ isInstalled(item.name) ? 'Already Installed' : 'Install Server' }}
                </button>
                <a v-if="item.doc_url" :href="item.doc_url" target="_blank" class="text-[11px] font-bold text-indigo-500 hover:underline">Documentation</a>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Modals -->
    <MCPServerModal v-if="modalServer !== undefined" :server="modalServer" @close="modalServer = undefined" @saved="onSaved" />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useAuthStore, api } from '@/stores/auth'

const authStore = useAuthStore()
const isAdmin = computed(() => authStore.hasRole('admin'))

const activeTab = ref('registered')
const servers = ref([])
const marketplace = ref([])
const loading = ref(true)
const marketLoading = ref(false)
const pingingAll = ref(false)
const modalServer = ref(undefined)

// Health state
const healthStatus  = reactive({})
const healthLatency = reactive({})
const healthError   = reactive({})

async function fetchServers() {
  loading.value = true
  try {
    const { data } = await api.get('/mcp-servers')
    servers.value = data ?? []
  } catch (e) { console.error('Failed to fetch servers', e) } finally {
    loading.value = false
  }
}

async function fetchMarketplace() {
  marketLoading.value = true
  try {
    const { data } = await api.get('/mcp-market')
    marketplace.value = data ?? []
  } catch (e) { console.error('Failed to fetch marketplace', e) } finally {
    marketLoading.value = false
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

function openModal(srv) { modalServer.value = srv }

function onSaved(srv) {
  const idx = servers.value.findIndex(s => s.id === srv.id)
  if (idx !== -1) servers.value[idx] = srv
  else {
    servers.value.push(srv)
    // If it was a new server, update the modalServer to the newly created one 
    // so the modal knows it has an ID now (switches to edit mode)
    modalServer.value = srv
  }
}

function isInstalled(name) {
  return servers.value.some(s => s.name === name)
}

async function installFromMarket(item) {
  if (!isAdmin.value) return
  
  // Pre-fill modal with market data
  const newSrv = {
    name: item.name,
    transport: item.transport,
    command: item.command,
    args: item.args || [],
    url: item.URL,
    description: item.description,
    doc_url: item.doc_url,
    env_vars: {}
  }
  
  // If there are required env vars, we open the modal for the user to fill them
  if (item.env_vars && item.env_vars.length > 0) {
    item.env_vars.forEach(ev => { newSrv.env_vars[ev.Name] = '' })
    openModal(newSrv)
  } else {
    // Just install it directly
    try {
      const { data } = await api.post('/mcp-servers', newSrv)
      servers.value.push(data)
      activeTab.value = 'registered'
    } catch (e) { alert(e.response?.data?.error || e.message) }
  }
}

watch(activeTab, (tab) => {
  if (tab === 'marketplace' && !marketplace.value.length) fetchMarketplace()
})

onMounted(() => {
  fetchServers()
})
</script>

<script>
// MCPServerModal is imported from the previous turn's structure 
// but we need to ensure it's exported correctly in this file if we're not using separate files.
import { defineComponent, h, ref } from 'vue'
import { api } from '@/stores/auth'

export const MCPServerModal = defineComponent({
  props: { server: { default: null } },
  emits: ['close', 'saved'],
  setup(p, { emit }) {
    const isEdit = !!p.server?.id
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
      args:        Array.isArray(p.server?.args) ? p.server.args.join(' ') : (p.server?.args ?? ''),
      description: p.server?.description ?? '',
      doc_url:     p.server?.doc_url     ?? '',
    })

    const discoveredTools = ref([])
    const saving      = ref(false)
    const savedSuccess = ref(false)
    const discovering = ref(false)
    const error       = ref('')

    function getEnvMap() {
      const m = {}
      envList.value.forEach(e => { if (e.key) m[e.key] = e.val })
      return m
    }

    async function discover() {
      discovering.value = true; error.value = ''
      try {
        const payload = {
          ...form.value,
          args: form.value.args.match(/(?:[^\s"]+|"[^"]*")+/g)?.map(s => s.replace(/^"|"$/g, '')) || [],
          env_vars: getEnvMap()
        }
        const { data } = await api.post('/mcp-servers/discover', payload)
        if (data.error) error.value = data.error
        else {
          form.value.description = data.description
          discoveredTools.value = data.raw_tools || []
        }
      } catch (e) { error.value = e.response?.data?.error ?? e.message } finally {
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
        
        savedSuccess.value = true
        setTimeout(() => { savedSuccess.value = false }, 3000)
        
        emit('saved', res.data)
      } catch (e) { error.value = e.response?.data?.error ?? e.message } finally {
        saving.value = false
      }
    }

    return () => h('div', { class: 'fixed inset-0 bg-black/80 backdrop-blur-md flex items-center justify-center z-[100] p-4' }, [
      h('div', { class: 'card w-full max-w-xl p-8 space-y-6 shadow-2xl border-border max-h-[90vh] overflow-y-auto' }, [
        h('div', { class: 'flex justify-between items-center' }, [
          h('h3', { class: 'text-lg font-bold text-text uppercase tracking-tight' }, isEdit ? 'Configure Server' : 'Install Server'),
          h('button', { class: 'text-text-muted hover:text-text', onClick: () => emit('close') }, '\u2715')
        ]),

        h('div', { class: 'grid grid-cols-2 gap-4' }, [
          h('div', { class: 'space-y-1.5' }, [
            h('label', { class: 'text-[10px] uppercase font-bold text-text-muted tracking-widest' }, 'Name'),
            h('input', { class: 'input font-mono text-sm', value: form.value.name, onInput: e => form.value.name = e.target.value }),
          ]),
          h('div', { class: 'space-y-1.5' }, [
            h('label', { class: 'text-[10px] uppercase font-bold text-text-muted tracking-widest' }, 'Transport'),
            h('select', { 
              class: 'input text-sm', 
              value: form.value.transport, 
              onChange: e => form.value.transport = e.target.value 
            }, [
              h('option', { value: 'stdio' }, 'Local (stdio)'),
              h('option', { value: 'sse' }, 'Remote (SSE)')
            ]),
          ]),
        ]),

        form.value.transport === 'stdio' ? [
          h('div', { class: 'space-y-1.5' }, [
            h('label', { class: 'text-[10px] uppercase font-bold text-text-muted tracking-widest' }, 'Executable'),
            h('input', { class: 'input font-mono text-sm', value: form.value.command, onInput: e => form.value.command = e.target.value }),
          ]),
          h('div', { class: 'space-y-1.5' }, [
            h('label', { class: 'text-[10px] uppercase font-bold text-text-muted tracking-widest' }, 'Arguments'),
            h('input', { class: 'input font-mono text-sm', value: form.value.args, onInput: e => form.value.args = e.target.value }),
          ]),
        ] : [
          h('div', { class: 'space-y-1.5' }, [
            h('label', { class: 'text-[10px] uppercase font-bold text-text-muted tracking-widest' }, 'Endpoint URL'),
            h('input', { class: 'input font-mono text-sm', value: form.value.url, onInput: e => form.value.url = e.target.value }),
          ]),
        ],

        h('div', { class: 'space-y-3' }, [
          h('div', { class: 'flex justify-between items-center' }, [
            h('label', { class: 'text-[10px] uppercase font-bold text-text-muted tracking-widest' }, 'Environment Variables'),
            h('button', { class: 'text-[10px] font-bold text-indigo-500', onClick: () => envList.value.push({key:'',val:''}) }, '+ Add')
          ]),
          envList.value.map((e, idx) => h('div', { class: 'flex gap-2' }, [
            h('input', { class: 'input text-xs font-mono flex-1', value: e.key, placeholder: 'KEY', onInput: v => e.key = v.target.value }),
            h('input', { class: 'input text-xs font-mono flex-1', type: 'password', value: e.val, placeholder: 'VALUE', onInput: v => e.val = v.target.value }),
            h('button', { class: 'p-2 text-text-muted hover:text-red-500', onClick: () => envList.value.splice(idx, 1) }, '\u2715')
          ]))
        ]),

        error.value && h('div', { class: 'p-3 rounded bg-red-500/10 border border-red-500/20 text-red-500 text-[10px] font-mono' }, error.value),

        discoveredTools.value.length > 0 && h('div', { class: 'space-y-2' }, [
          h('label', { class: 'text-[10px] uppercase font-bold text-text-muted tracking-widest' }, 'Discovered Tools'),
          h('div', { class: 'grid grid-cols-2 gap-2 max-h-[150px] overflow-y-auto p-1' }, 
            discoveredTools.value.map(t => h('div', { class: 'p-2 rounded bg-surface-2 border border-border flex flex-col gap-1' }, [
              h('span', { class: 'text-[10px] font-bold text-text truncate' }, t.name),
              h('span', { class: 'text-[8px] text-text-muted line-clamp-1' }, t.description)
            ]))
          )
        ]),

        h('div', { class: 'flex gap-3 pt-4 items-center' }, [
          h('button', { class: 'btn-ghost text-xs px-4 border border-border', onClick: discover, disabled: discovering.value }, discovering.value ? 'Discovering...' : 'Discover Tools'),
          h('div', { class: 'flex-1' }, [
            savedSuccess.value && h('span', { class: 'text-[10px] font-bold text-green-500 animate-pulse' }, '✓ Saved Successfully')
          ]),
          h('button', { class: 'btn-primary px-8 font-bold text-[10px] uppercase tracking-widest', onClick: save, disabled: saving.value }, saving.value ? 'Saving...' : 'Save Configuration'),
        ])
      ])
    ])
  }
})
</script>

<style scoped>
.font-display { font-family: 'Outfit', sans-serif; }
.card {
  @apply bg-surface-1 border border-border rounded-2xl overflow-hidden;
}
.btn-primary {
  @apply bg-indigo-600 hover:bg-indigo-700 text-white border-none transition-all;
}
.btn-ghost {
  @apply transition-colors;
}
.btn-ghost:hover {
  background-color: var(--color-surface-2);
}
.input {
  @apply w-full bg-surface-2 border border-border rounded-xl px-4 py-2 text-text focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all outline-none;
}
</style>
