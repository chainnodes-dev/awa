<template>
  <div class="flex flex-col h-full overflow-hidden bg-surface-0">
    <!-- Top bar -->
    <header class="sticky top-0 z-10 flex items-center gap-4 px-6 py-4 bg-surface-1/95 backdrop-blur-sm border-b border-border shrink-0">
      <div class="flex flex-col">
        <h1 class="text-base font-bold text-text tracking-tight uppercase">MCP Infrastructure</h1>
        <p class="text-[10px] text-text-muted font-bold uppercase tracking-widest opacity-60">Model Context Protocol</p>
      </div>
      
      <div class="flex-1 px-4">
        <!-- Tab Switcher -->
        <div class="flex p-1 bg-surface-2 rounded-xl border border-border w-fit">
          <button @click="activeTab = 'marketplace'" 
                  :class="['px-6 py-1.5 text-[11px] font-bold uppercase rounded-lg transition-all', 
                           activeTab === 'marketplace' ? 'bg-accent text-white shadow-lg shadow-indigo-500/20' : 'text-text-muted hover:text-text']"
                  style="background-color: activeTab === 'marketplace' ? 'var(--color-accent)' : 'transparent'">
            Marketplace
            <span class="px-1.5 py-0.5 rounded-full bg-amber-500/20 text-amber-500 text-[8px] ml-2">CURATED</span>
          </button>
          <button @click="activeTab = 'registered'" 
                  :class="['px-6 py-1.5 text-[11px] font-bold uppercase rounded-lg transition-all', 
                           activeTab === 'registered' ? 'bg-accent text-white shadow-lg shadow-indigo-500/20' : 'text-text-muted hover:text-text']"
                  style="background-color: activeTab === 'registered' ? 'var(--color-accent)' : 'transparent'">
            Registered Servers
          </button>
        </div>
      </div>



      <div class="flex items-center gap-3">
        <button v-if="activeTab === 'registered' && servers.length > 0" @click="exportToYaml" class="btn-ghost h-9 px-4 text-xs gap-2 border border-indigo-500/20 text-indigo-500 hover:bg-indigo-500/10">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
          Export YAML
        </button>
        <button @click="pingAll" :disabled="pingingAll" class="btn-ghost h-9 px-4 text-xs gap-2 border border-border bg-surface-1">
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
    <!-- Content Area -->
    <main class="flex-1 overflow-y-auto p-6 scroll-smooth custom-scrollbar">
      
      <!-- Registered Servers Section -->
      <section v-if="activeTab === 'registered'" class="space-y-6 animate-in fade-in duration-300">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <h2 class="text-sm font-bold text-text uppercase tracking-widest">My Registered Servers</h2>
            <span class="px-2 py-0.5 rounded-full bg-indigo-500/10 text-indigo-500 text-[10px] font-bold">{{ servers.length }}</span>
          </div>
        </div>

        <div v-if="loading" class="flex flex-col items-center justify-center py-20 gap-4">
          <div class="w-8 h-8 border-4 border-indigo-500/20 border-t-indigo-500 rounded-full animate-spin"/>
          <p class="text-xs text-text-muted font-mono uppercase tracking-widest animate-pulse">Syncing My Servers...</p>
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
          <div v-for="srv in servers" :key="srv.id" class="p-4 rounded-xl border border-border bg-surface-1 hover:border-accent/50 transition-all group">
            <div class="flex items-start justify-between mb-3">
              <div class="flex items-center gap-2.5 min-w-0">
                <div class="relative shrink-0">
                  <div v-if="healthStatus[srv.id] === 'online'" class="w-2 h-2 rounded-full bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.4)]"/>
                  <div v-else-if="healthStatus[srv.id] === 'offline'" class="w-2 h-2 rounded-full bg-red-500"/>
                  <div v-else-if="healthStatus[srv.id] === 'checking'" class="w-2 h-2 rounded-full bg-accent animate-pulse"/>
                  <div v-else class="w-2 h-2 rounded-full bg-slate-600"/>
                </div>
                <h4 class="font-bold text-[13px] text-text uppercase tracking-tight truncate">{{ srv.name }}</h4>
              </div>
              <div class="flex gap-1">
                <button @click="pingServer(srv)" class="p-1 text-text-muted hover:text-text transition-colors">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
                </button>
                <button v-if="isAdmin" @click="openModal(srv)" class="p-1 text-text-muted hover:text-accent transition-colors">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>
                </button>
              </div>
            </div>
            
            <p class="text-[11px] text-text-muted leading-relaxed line-clamp-2 mb-4 h-8">
              {{ srv.description || 'Custom MCP connector for local or remote tools.' }}
            </p>

            <div class="flex items-center justify-between pt-3 border-t border-border/30">
              <span class="text-[9px] font-bold text-accent/80 uppercase tracking-widest">{{ srv.transport }}</span>
              <span v-if="healthLatency[srv.id]" class="text-[9px] font-mono text-green-500/80">{{ healthLatency[srv.id] }}ms</span>
              <a v-if="srv.doc_url" :href="srv.doc_url" target="_blank" class="text-text-muted hover:text-accent">
                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
              </a>
            </div>
          </div>
        </div>
      </section>

      <!-- Marketplace Section -->
      <section v-if="activeTab === 'marketplace'" class="space-y-8 animate-in fade-in duration-300">
        
        <!-- Filters -->
        <div class="flex flex-col md:flex-row gap-4 items-center justify-between bg-surface-2 p-4 rounded-2xl border border-border">
          <div class="relative w-full md:w-96 group">
            <div class="absolute left-4 top-1/2 -translate-y-1/2 text-text-muted group-focus-within:text-indigo-500 transition-colors">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
            </div>
            <input v-model="marketSearch" 
                   type="text" 
                   placeholder="Search curated servers..." 
                   class="input !pl-10 h-10 text-xs">
          </div>
          
          <div class="flex gap-2 overflow-x-auto w-full md:w-auto pb-2 md:pb-0 scrollbar-none">
            <button v-for="cat in marketplaceCategories" 
                    :key="cat"
                    @click="selectedCategory = cat"
                    :class="['px-4 py-1.5 rounded-full text-[10px] font-bold uppercase tracking-wider transition-all whitespace-nowrap',
                             selectedCategory === cat ? 'bg-accent text-white shadow-lg shadow-indigo-500/20' : 'bg-surface-1 text-text-muted hover:text-text border border-border']"
                    :style="selectedCategory === cat ? 'background-color: var(--color-accent)' : ''">
              {{ cat }}
            </button>
          </div>
        </div>

        <div v-if="marketLoading" class="flex flex-col items-center justify-center py-20 gap-4">
          <div class="w-8 h-8 border-4 border-amber-500/20 border-t-amber-500 rounded-full animate-spin"/>
          <p class="text-xs text-text-muted font-mono uppercase tracking-widest animate-pulse">Syncing Curated List...</p>
        </div>

        <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 pb-12">
          <div v-for="item in filteredMarketplace" :key="item.id" 
               class="flex gap-6 p-6 rounded-2xl bg-surface-1 border border-border hover:border-indigo-500/40 hover:shadow-xl hover:shadow-indigo-500/5 transition-all duration-300 group">
            
            <!-- Icon / Brand -->
            <div class="w-16 h-16 shrink-0 rounded-2xl bg-surface-2 flex items-center justify-center border border-border shadow-sm group-hover:scale-105 transition-transform overflow-hidden">
              <img v-if="item.icon && item.icon.startsWith('http')" :src="item.icon" class="w-10 h-10 object-contain" />
              <svg v-else-if="item.id === 'sec-edgar' || item.icon === 'trending-up'" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-amber-500"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
              <svg v-else-if="item.id === 'brave-search' || item.icon === 'search'" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-indigo-500"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
              <svg v-else-if="item.id === 'slack' || item.icon === 'message-square'" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-pink-500"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
              <svg v-else-if="item.id === 'github' || item.icon === 'github'" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-text"><path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"/></svg>
              <svg v-else-if="item.icon === 'database'" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-indigo-400"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/><path d="M3 12c0 1.66 4 3 9 3s9-1.34 9-3"/></svg>
              <svg v-else-if="item.icon === 'hard-drive'" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-blue-400"><rect x="2" y="2" width="20" height="20" rx="2" ry="2"/><path d="M2 12h20"/><path d="M6 16h.01"/><path d="M10 16h.01"/></svg>
              <svg v-else-if="item.icon === 'table'" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-green-500"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="3" y1="15" x2="21" y2="15"/><line x1="9" y1="3" x2="9" y2="21"/><line x1="15" y1="3" x2="15" y2="21"/></svg>
              <svg v-else-if="item.icon === 'cloud-sun'" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-sky-400"><path d="M12 2v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="M20 12h2"/><path d="m19.07 4.93-1.41 1.41"/><path d="M15.947 12.65a4 4 0 0 0-5.925-4.128"/><path d="M13 22H7a5 5 0 1 1 4.9-6H13a3 3 0 0 1 0 6Z"/></svg>
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
      </section>
    </main>

    <!-- Modals -->
    <MCPServerModal v-if="modalServer !== undefined" :server="modalServer" @close="modalServer = undefined" @saved="onSaved" @deleted="onDeleted" />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useAuthStore, api } from '@/stores/auth'

const authStore = useAuthStore()
const isAdmin = computed(() => authStore.hasRole('admin'))
const activeTab = ref('marketplace')

const servers = ref([])
const marketplace = ref([])
const loading = ref(true)
const marketLoading = ref(false)
const marketSearch = ref('')
const selectedCategory = ref('All')
const pingingAll = ref(false)
const modalServer = ref(undefined)

// Health state
const healthStatus  = reactive({})
const healthLatency = reactive({})
const healthError   = reactive({})

const marketplaceCategories = computed(() => {
  const cats = new Set(marketplace.value.map(m => m.category).filter(Boolean))
  return ['All', ...Array.from(cats).sort()]
})

const filteredMarketplace = computed(() => {
  return marketplace.value.filter(m => {
    const matchesSearch = !marketSearch.value || 
      m.name.toLowerCase().includes(marketSearch.value.toLowerCase()) ||
      m.description.toLowerCase().includes(marketSearch.value.toLowerCase())
    
    const matchesCategory = selectedCategory.value === 'All' || m.category === selectedCategory.value
    
    return matchesSearch && matchesCategory
  })
})

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
    modalServer.value = srv
  }
}

function onDeleted(srvID) {
  servers.value = servers.value.filter(s => s.id !== srvID)
}



function isInstalled(name) {
  return servers.value.some(s => s.name === name)
}

function exportToYaml() {
  let yaml = 'mcp_servers:\n'
  servers.value.forEach(s => {
    yaml += `  - name: ${s.name}\n`
    if (s.description) yaml += `    description: "${s.description.replace(/"/g, '\\"')}"\n`
    if (s.transport)   yaml += `    transport: ${s.transport}\n`
    if (s.command)     yaml += `    command: ${s.command}\n`
    if (s.args && s.args.length > 0) {
      yaml += `    args: [${s.args.map(a => `"${a}"`).join(', ')}]\n`
    }
    if (s.url)         yaml += `    url: ${s.url}\n`
    if (s.env_vars && Object.keys(s.env_vars).length > 0) {
      const keys = Object.keys(s.env_vars)
      if (keys.length === 1) {
        yaml += `    env_var: ${keys[0]}\n`
      } else {
        yaml += `    env_vars: [${keys.map(k => `"${k}"`).join(', ')}]\n`
      }
    }
    yaml += '\n'
  })

  const blob = new Blob([yaml], { type: 'text/yaml' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'mcp_registry.yaml'
  a.click()
  URL.revokeObjectURL(url)
}

async function installFromMarket(item) {
  if (!isAdmin.value) return
  
  // Pre-fill modal with market data
  const newSrv = {
    name: item.name,
    transport: item.transport || 'stdio',
    command: item.command || '',
    args: item.args || [],
    url: item.url || '',
    description: item.description || '',
    doc_url: item.doc_url || '',
    env_vars: {}
  }
  
  if (item.env_vars && item.env_vars.length > 0) {
    item.env_vars.forEach(ev => { 
      const key = ev.name || ev.Name || '';
      if (key) newSrv.env_vars[key] = '' 
    })
  }
  
  openModal(newSrv)
}

onMounted(() => {
  fetchServers()
  fetchMarketplace()
})
</script>

<script>
// MCPServerModal is imported from the previous turn's structure 
// but we need to ensure it's exported correctly in this file if we're not using separate files.
import { defineComponent, h, ref } from 'vue'
import { api } from '@/stores/auth'

export const MCPServerModal = defineComponent({
  props: { server: { default: null } },
  emits: ['close', 'saved', 'deleted'],
  setup(p, { emit }) {
    console.log('MCPServerModal setup, server:', p.server)
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

    const discoveredTools = ref(p.server?.tools ?? [])
    const saving      = ref(false)
    const savedSuccess = ref(false)
    const discovering = ref(false)
    const error       = ref('')
    const showImport  = ref(false)
    const importText  = ref('')

    function handleImport() {
      try {
        let data = JSON.parse(importText.value)
        // Handle full Claude Desktop config { mcpServers: { ... } }
        if (data.mcpServers) {
          const keys = Object.keys(data.mcpServers)
          if (keys.length === 0) throw new Error('No servers found in JSON')
          const name = keys[0]
          const config = data.mcpServers[name]
          form.value.name = name
          form.value.command = config.command || ''
          form.value.args = Array.isArray(config.args) ? config.args.join(' ') : (config.args || '')
          if (config.env) {
            envList.value = Object.entries(config.env).map(([key, val]) => ({ key, val }))
          }
        } else {
          // Handle single server object
          form.value.command = data.command || form.value.command
          form.value.args = Array.isArray(data.args) ? data.args.join(' ') : (data.args || form.value.args)
          if (data.env) {
            envList.value = Object.entries(data.env).map(([key, val]) => ({ key, val }))
          }
        }
        showImport.value = false
        importText.value = ''
      } catch (e) { error.value = 'Invalid JSON: ' + e.message }
    }

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
          env_vars: getEnvMap(),
          tools: discoveredTools.value
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

    async function remove() {
      if (!confirm('Are you sure you want to delete this MCP server?')) return
      saving.value = true; error.value = ''
      try {
        await api.delete('/mcp-servers/' + p.server.id)
        emit('deleted', p.server.id)
        emit('close')
      } catch (e) { error.value = e.response?.data?.error ?? e.message } finally {
        saving.value = false
      }
    }

    return () => h('div', { class: 'fixed inset-0 bg-black/80 backdrop-blur-md flex items-center justify-center z-[100] p-4' }, [
      h('div', { class: 'card w-full max-w-xl p-8 space-y-6 shadow-2xl border-border max-h-[90vh] overflow-y-auto' }, [
        h('div', { class: 'flex justify-between items-center' }, [
          h('h3', { class: 'text-lg font-bold text-text uppercase tracking-tight' }, isEdit ? 'Configure Server' : 'Install Server'),
          h('div', { class: 'flex gap-2' }, [
            !isEdit && h('button', { class: 'text-[10px] font-bold text-indigo-500 border border-indigo-500/20 px-2 py-1 rounded hover:bg-indigo-500/10', onClick: () => showImport.value = !showImport.value }, 'Import JSON'),
            h('button', { class: 'text-text-muted hover:text-text p-1', onClick: () => emit('close') }, '\u2715')
          ])
        ]),

        showImport.value && h('div', { class: 'space-y-3 p-4 bg-surface-2 rounded-xl border border-indigo-500/30 animate-in fade-in slide-in-from-top-2' }, [
          h('label', { class: 'text-[10px] uppercase font-bold text-indigo-400 tracking-widest' }, 'Paste Claude Desktop JSON'),
          h('textarea', { 
            class: 'input font-mono text-[10px] h-32 leading-tight', 
            placeholder: '{\n  "mcpServers": {\n    "my-server": { ... }\n  }\n}',
            value: importText.value,
            onInput: e => importText.value = e.target.value
          }),
          h('div', { class: 'flex justify-end gap-2' }, [
            h('button', { class: 'text-[10px] font-bold text-text-muted', onClick: () => showImport.value = false }, 'Cancel'),
            h('button', { class: 'text-[10px] font-bold text-indigo-500', onClick: handleImport }, 'Apply Config')
          ])
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
            h('input', { 
              class: 'input text-xs font-mono flex-1', 
              value: e.key, 
              placeholder: 'KEY', 
              autocomplete: 'off',
              onInput: v => e.key = v.target.value 
            }),
            h('input', { 
              class: 'input text-xs font-mono flex-1', 
              type: 'password', 
              value: e.val, 
              placeholder: 'VALUE', 
              autocomplete: 'new-password',
              onInput: v => e.val = v.target.value 
            }),
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
          isEdit && h('button', { class: 'btn-ghost text-red-500 text-xs px-4 border border-red-500/20 hover:bg-red-500/10', onClick: remove, disabled: saving.value }, 'Delete'),
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
