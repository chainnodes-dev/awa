<template>
  <div class="flex flex-col h-full overflow-hidden">
    <header class="flex items-center gap-4 px-6 py-4 border-b border-border shrink-0">
      <h1 class="text-base font-semibold text-slate-200 uppercase tracking-wider">Tenant Administration</h1>
      <div class="flex-1"/>
      <button @click="showAddModal = true" class="btn-primary">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        New Tenant
      </button>
    </header>

    <div class="flex-1 overflow-y-auto p-6">
      <div class="mb-8 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <div v-for="tenant in tenantStore.tenants" :key="tenant.id" class="card p-5 group flex flex-col hover:border-accent/40 transition-all duration-300">
          <div class="flex items-start justify-between mb-4">
            <div>
              <h3 class="font-bold text-slate-100 text-lg group-hover:text-accent transition-colors">{{ tenant.name }}</h3>
              <p class="text-[10px] font-mono text-slate-500 uppercase tracking-widest mt-1">{{ tenant.slug }}</p>
            </div>
            <div class="w-8 h-8 rounded bg-surface-1 flex items-center justify-center font-mono text-[10px] text-slate-600 shadow-inner">
              ID
            </div>
          </div>
          
          <div class="space-y-3 mt-auto">
             <div class="flex justify-between items-center text-xs">
                <span class="text-slate-500">License Status</span>
                <span :class="tenant.license_token ? 'text-green-400' : 'text-amber-400'">
                  {{ tenant.license_token ? 'Active' : 'Missing' }}
                </span>
             </div>
             <div class="flex justify-between items-center text-[10px] font-mono text-slate-600">
                <span>Tenant ID</span>
                <span class="bg-black/20 px-1 rounded">{{ tenant.id }}</span>
             </div>
          </div>

          <div class="flex gap-2 mt-6 pt-4 border-t border-border/50">
            <button @click="deleteTenant(tenant)" class="btn-danger flex-1 justify-center text-xs">Delete</button>
          </div>
        </div>
      </div>

      <div v-if="!tenantStore.tenants.length && !tenantStore.loading" class="card p-12 text-center">
        <div class="w-16 h-16 bg-surface-1 rounded-full flex items-center justify-center mx-auto mb-6 text-slate-600">
           <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
             <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
             <circle cx="9" cy="7" r="4"/>
             <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
             <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
           </svg>
        </div>
        <h3 class="text-slate-300 font-medium mb-2">No custom tenants found</h3>
        <p class="text-slate-500 text-sm max-w-xs mx-auto mb-6">Create your first enterprise tenant to begin multi-tenant isolation testing.</p>
        <button @click="showAddModal = true" class="btn-primary">Create Tenant</button>
      </div>
    </div>

    <!-- Add Tenant Modal -->
    <div v-if="showAddModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-md">
      <div class="card w-full max-w-sm p-8 shadow-3xl animate-in slide-in-from-bottom-4 duration-300 relative overflow-hidden">
        <div class="absolute top-0 right-0 p-8 opacity-5">
           <svg width="120" height="120" viewBox="0 0 24 24" fill="currentColor"><path d="M13 14h-2V9h2v5zm0 4h-2v-2h2v2zM1 21h22L12 2 1 21z"/></svg>
        </div>
        
        <h2 class="text-xl font-bold text-slate-50 mb-1">Onboard Tenant</h2>
        <p class="text-xs text-slate-500 mb-8">Initialize a new isolated environment.</p>
        
        <form @submit.prevent="createTenant" class="space-y-5">
          <div class="space-y-2">
            <label class="text-[10px] font-bold uppercase tracking-widest text-slate-500">Legal Name</label>
            <input v-model="newTenant.name" type="text" required class="input py-3" placeholder="e.g. Acme Corp" />
          </div>
          <div class="space-y-2">
            <label class="text-[10px] font-bold uppercase tracking-widest text-slate-500">Slug (URL identifier)</label>
            <input v-model="newTenant.slug" @input="newTenant.slug = newTenant.slug.toLowerCase().replace(/[^a-z0-9-]/g, '')" type="text" required class="input py-3 font-mono" placeholder="e.g. acme-prod" />
          </div>
          
          <div v-if="error" class="text-xs text-red-400 bg-red-400/10 p-3 rounded border border-red-500/20">
            {{ error }}
          </div>

          <div class="flex gap-3 pt-6">
            <button type="button" @click="showAddModal = false" class="btn-ghost flex-1 py-3">Cancel</button>
            <button type="submit" :disabled="creating" class="btn-primary flex-1 justify-center py-3 overflow-hidden group">
              <span v-if="!creating" class="flex items-center gap-2">
                Deploy Environment
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="group-hover:translate-x-0.5 transition-transform">
                   <line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/>
                </svg>
              </span>
              <span v-else>Deploying...</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useTenantStore } from '@/stores/tenants'

const tenantStore = useTenantStore()
const showAddModal = ref(false)
const creating = ref(false)
const error = ref(null)

const newTenant = reactive({
  name: '',
  slug: ''
})

onMounted(() => {
  tenantStore.fetchAll()
})

async function createTenant() {
  creating.value = true
  error.value = null
  try {
    await tenantStore.create({ ...newTenant })
    showAddModal.value = false
    newTenant.name = ''
    newTenant.slug = ''
  } catch (e) {
    error.value = e.response?.data?.error ?? e.message
  } finally {
    creating.value = false
  }
}

async function deleteTenant(tenant) {
  if (!confirm(`WARNING: Deleting tenant "${tenant.name}" will permanently erase all associated workflows, runs, and users. PROCEED?`)) return
  try {
    await tenantStore.remove(tenant.id)
  } catch (e) {
    alert(e.response?.data?.error ?? e.message)
  }
}
</script>
