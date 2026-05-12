<template>
  <div class="flex flex-col h-full bg-surface-0 overflow-hidden">
    <!-- Header -->
    <header class="flex items-center justify-between px-8 py-6 bg-surface-1 border-b border-border shrink-0">
      <div>
        <h1 class="text-2xl font-bold text-text tracking-tight">MCP Marketplace</h1>
        <p class="text-sm text-text-muted mt-1">Discover and install community-powered Model Context Protocol servers.</p>
      </div>
      <div class="flex items-center gap-4">
        <div class="relative group">
          <svg class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted group-focus-within:text-accent transition-colors" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
          </svg>
          <input 
            v-model="search"
            type="text" 
            placeholder="Search servers..." 
            class="bg-surface-0 border border-border rounded-xl pl-10 pr-4 py-2 text-sm text-text outline-none focus:border-accent focus:ring-4 focus:ring-accent/10 transition-all w-64"
          />
        </div>
        <RouterLink to="/mcp-servers" class="btn-ghost text-xs gap-2">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/>
          </svg>
          My Servers
        </RouterLink>
      </div>
    </header>

    <!-- Categories -->
    <div class="flex gap-2 px-8 py-4 bg-surface-1/50 border-b border-border overflow-x-auto no-scrollbar shrink-0">
      <button 
        v-for="cat in categories" 
        :key="cat"
        @click="selectedCategory = cat"
        :class="[
          'px-4 py-1.5 rounded-full text-xs font-medium transition-all whitespace-nowrap border',
          selectedCategory === cat 
            ? 'bg-accent border-accent text-white shadow-lg shadow-accent/20' 
            : 'bg-surface-1 border-border text-text-muted hover:border-slate-500 hover:text-text'
        ]"
      >
        {{ cat }}
      </button>
    </div>

    <!-- Content -->
    <main class="flex-1 overflow-y-auto p-8 custom-scrollbar">
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        <div 
          v-for="item in filteredItems" 
          :key="item.id"
          class="group relative bg-surface-1 border border-border rounded-2xl p-6 hover:border-accent/40 hover:bg-surface-2 transition-all duration-300 flex flex-col shadow-sm hover:shadow-xl hover:shadow-accent/5 hover:-translate-y-1"
        >
          <!-- Status Badge -->
          <div v-if="isInstalled(item.name)" class="absolute top-4 right-4 bg-green-500/10 text-green-500 text-[10px] font-bold px-2 py-0.5 rounded-full border border-green-500/20">
            INSTALLED
          </div>

          <div class="flex items-center gap-4 mb-4">
            <div class="w-12 h-12 rounded-xl bg-surface-0 border border-border flex items-center justify-center overflow-hidden shrink-0 group-hover:scale-110 transition-transform duration-300 shadow-inner">
              <template v-if="item.icon.startsWith('http')">
                <img :src="item.icon" class="w-8 h-8 object-contain" />
              </template>
              <template v-else>
                <span class="text-2xl">{{ item.icon }}</span>
              </template>
            </div>
            <div>
              <h3 class="font-bold text-text tracking-tight">{{ item.name }}</h3>
              <span class="text-[10px] font-bold text-text-muted uppercase tracking-widest opacity-60">{{ item.category }}</span>
            </div>
          </div>

          <p class="text-sm text-text-muted leading-relaxed mb-6 flex-1">
            {{ item.description }}
          </p>

          <button 
            @click="install(item)"
            :disabled="installing === item.id || isInstalled(item.name)"
            :class="[
              'w-full py-2.5 rounded-xl font-bold text-xs tracking-wider transition-all duration-300 uppercase',
              isInstalled(item.name)
                ? 'bg-surface-0 border border-border text-text-muted cursor-default'
                : 'bg-accent text-white hover:bg-accent-hover shadow-lg shadow-accent/20 hover:shadow-accent/40'
            ]"
          >
            <template v-if="installing === item.id">
              <div class="flex items-center justify-center gap-2">
                <svg class="animate-spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                  <path d="M12 2v4m0 12v4M4.93 4.93l2.83 2.83m8.48 8.48l2.83 2.83M2 12h4m12 0h4M4.93 19.07l2.83-2.83m8.48-8.48l2.83-2.83"/>
                </svg>
                Installing...
              </div>
            </template>
            <template v-else-if="isInstalled(item.name)">
              Installed
            </template>
            <template v-else>
              Install Server
            </template>
          </button>
        </div>
      </div>

      <!-- Empty State -->
      <div v-if="filteredItems.length === 0" class="h-64 flex flex-col items-center justify-center text-center">
        <div class="w-16 h-16 bg-surface-1 rounded-full flex items-center justify-center mb-4 border border-border shadow-inner">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-text-muted opacity-30">
            <path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
          </svg>
        </div>
        <h3 class="text-lg font-bold text-text">No servers found</h3>
        <p class="text-sm text-text-muted mt-1">Try adjusting your search or category filter.</p>
      </div>
    </main>

    <!-- Success Modal -->
    <div v-if="installedServer" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-surface-0/80 backdrop-blur-md">
      <div class="bg-surface-1 border border-border rounded-3xl p-8 max-w-sm w-full shadow-2xl animate-in zoom-in duration-300">
        <div class="w-16 h-16 bg-green-500/10 text-green-500 rounded-full flex items-center justify-center mx-auto mb-6">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
        </div>
        <h2 class="text-xl font-bold text-center mb-2">Success!</h2>
        <p class="text-sm text-text-muted text-center mb-8">
          <strong>{{ installedServer.name }}</strong> has been installed. You can now configure its environment variables in the settings.
        </p>
        <div class="flex gap-3">
          <button @click="installedServer = null" class="btn-ghost flex-1">Dismiss</button>
          <RouterLink to="/mcp-servers" class="btn-primary flex-1">Configure</RouterLink>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '@/stores/auth'

const items = ref([])
const search = ref('')
const selectedCategory = ref('All')
const installing = ref(null)
const installedServer = ref(null)
const installedList = ref([])

const categories = computed(() => {
  const cats = ['All', ...new Set(items.value.map(i => i.category))]
  return cats
})

const filteredItems = computed(() => {
  return items.value.filter(i => {
    const matchesSearch = i.name.toLowerCase().includes(search.value.toLowerCase()) || 
                         i.description.toLowerCase().includes(search.value.toLowerCase())
    const matchesCat = selectedCategory.value === 'All' || i.category === selectedCategory.value
    return matchesSearch && matchesCat
  })
})

async function fetchInstalled() {
  try {
    const res = await api.get('/mcp-servers')
    installedList.value = res.map(s => s.name)
  } catch (e) {
    console.error('Failed to fetch installed servers:', e)
  }
}

function isInstalled(name) {
  return installedList.value.includes(name)
}

async function install(item) {
  installing.value = item.id
  try {
    const payload = {
      name:        item.name,
      transport:   'stdio',
      command:     item.command,
      args:        item.args,
      description: item.description,
      env_vars:    (item.envVars || []).reduce((acc, v) => ({ ...acc, [v]: '' }), {})
    }
    
    await api.post('/mcp-servers', payload)
    installedServer.value = item
    await fetchInstalled()
  } catch (e) {
    alert('Failed to install server: ' + (e.response?.data?.error || e.message))
  } finally {
    installing.value = null
  }
}

async function fetchMarket() {
  try {
    const { data } = await api.get('/mcp-market')
    items.value = data
  } catch (e) {
    console.error('Failed to fetch marketplace:', e)
  }
}

onMounted(() => {
  fetchMarket()
  fetchInstalled()
})
</script>

<style scoped>
.no-scrollbar::-webkit-scrollbar {
  display: none;
}
.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>
