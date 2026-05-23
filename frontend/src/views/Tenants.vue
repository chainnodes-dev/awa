<template>
  <div class="flex flex-col h-full overflow-hidden">
    <header class="flex items-center gap-4 px-6 py-4 border-b border-border shrink-0">
      <h1 class="text-base font-semibold text-text uppercase tracking-wider">Tenant Administration</h1>
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
              <p class="text-[10px] font-mono text-text-muted uppercase tracking-widest mt-1">{{ tenant.slug }}</p>
            </div>
            <div class="w-8 h-8 rounded bg-surface-1 flex items-center justify-center font-mono text-[10px] text-text-muted shadow-inner">
              ID
            </div>
          </div>
          
          <div class="space-y-3 mt-auto">
             <div class="flex justify-between items-center text-xs">
                <span class="text-text-muted">License Status</span>
                <span :class="tenant.license_token ? 'text-green-400' : 'text-amber-400'">
                   <span class="inline-block w-1.5 h-1.5 rounded-full mr-2" :class="tenant.license_token ? 'bg-green-400' : 'bg-amber-400'"></span>
                  {{ tenant.license_token ? 'Active' : 'Missing' }}
                </span>
             </div>
             <div class="flex justify-between items-center text-[10px] font-mono text-text-muted">
                <span>Tenant ID</span>
                <span class="bg-black/20 px-1 rounded">{{ tenant.id }}</span>
             </div>
          </div>

          <div class="flex gap-2 mt-6 pt-4 border-t border-border/50">
            <button @click="openLicenseModal(tenant)" class="btn-ghost flex-1 justify-center text-[10px] uppercase font-bold tracking-tight">License</button>
            <button @click="deleteTenant(tenant)" class="btn-danger px-3 justify-center text-[10px] uppercase font-bold">Delete</button>
          </div>
        </div>
      </div>

      <div v-if="!tenantStore.tenants.length && !tenantStore.loading" class="card p-12 text-center">
        <div class="w-16 h-16 bg-surface-1 rounded-full flex items-center justify-center mx-auto mb-6 text-text-muted">
           <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
             <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
             <circle cx="9" cy="7" r="4"/>
             <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
             <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
           </svg>
        </div>
        <h3 class="text-text font-medium mb-2">No custom tenants found</h3>
        <p class="text-text-muted text-sm max-w-xs mx-auto mb-6">Create your first enterprise tenant to begin multi-tenant isolation testing.</p>
        <button @click="showAddModal = true" class="btn-primary">Create Tenant</button>
      </div>
    </div>

    <!-- Add Tenant Modal -->
    <div v-if="showAddModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-md">
      <div class="card w-full max-w-sm p-8 shadow-3xl animate-in slide-in-from-bottom-4 duration-300 relative overflow-hidden">
        <h2 class="text-xl font-bold text-slate-50 mb-1">Onboard Tenant</h2>
        <p class="text-xs text-text-muted mb-8">Initialize a new isolated environment.</p>
        
        <form @submit.prevent="createTenant" class="space-y-5">
          <div class="space-y-2">
            <label class="text-[10px] font-bold uppercase tracking-widest text-text-muted">Legal Name</label>
            <input v-model="newTenant.name" type="text" required class="input py-3" placeholder="e.g. Acme Corp" />
          </div>
          <div class="space-y-2">
            <label class="text-[10px] font-bold uppercase tracking-widest text-text-muted">Slug (URL identifier)</label>
            <input v-model="newTenant.slug" @input="newTenant.slug = newTenant.slug.toLowerCase().replace(/[^a-z0-9-]/g, '')" type="text" required class="input py-3 font-mono" placeholder="e.g. acme-prod" />
          </div>
          
          <div v-if="error" class="text-xs text-red-400 bg-red-400/10 p-3 rounded border border-red-500/20">
            {{ error }}
          </div>

          <div class="flex gap-3 pt-6">
            <button type="button" @click="showAddModal = false" class="btn-ghost flex-1 py-3">Cancel</button>
            <button type="submit" :disabled="creating" class="btn-primary flex-1 justify-center py-3">Deploy</button>
          </div>
        </form>
      </div>
    </div>

    <!-- License Modal -->
    <div v-if="selectedTenant" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-md">
      <div class="card w-full max-w-lg p-8 shadow-3xl animate-in zoom-in-95 duration-300">
        <div class="flex items-start justify-between mb-6">
          <div>
            <h2 class="text-xl font-bold text-slate-50">License Manager</h2>
            <p class="text-xs text-text-muted mt-1">Generating for: {{ selectedTenant.name }}</p>
          </div>
          <button @click="selectedTenant = null; generatedToken = null" class="text-text-muted hover:text-text">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>

        <div v-if="!generatedToken" class="space-y-6">
          <div class="grid grid-cols-2 gap-4">
            <div class="space-y-2">
              <label class="text-[10px] font-bold uppercase tracking-widest text-text-muted">Tier</label>
              <select v-model="licenseForm.tier" class="input">
                <option value="community">Community</option>
                <option value="pro">Pro</option>
                <option value="enterprise">Enterprise</option>
              </select>
            </div>
            <div class="space-y-2">
              <label class="text-[10px] font-bold uppercase tracking-widest text-text-muted">Validity (Days)</label>
              <input v-model.number="licenseForm.expires_in_days" type="number" class="input" />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div class="space-y-2">
              <label class="text-[10px] font-bold uppercase tracking-widest text-text-muted">Features (comma separated)</label>
              <input v-model="licenseForm.featuresInput" type="text" placeholder="e.g. sso, branding" class="input" />
            </div>
          </div>

          <div v-if="licenseError" class="text-xs text-red-400 bg-red-400/10 p-3 rounded">
            {{ licenseError }}
          </div>

          <button @click="generateLicense" :disabled="generating" class="btn-primary w-full justify-center py-3">
            {{ generating ? 'Generating...' : 'Sign & Issue License' }}
          </button>
        </div>

        <div v-else class="space-y-6">
          <div class="bg-green-500/10 border border-green-500/20 p-4 rounded-lg text-xs text-green-400 leading-relaxed">
            License generated successfully. Provide the token below to the tenant administrator.
          </div>
          
          <div class="space-y-2">
            <label class="text-[10px] font-bold uppercase tracking-widest text-text-muted">Signed Token</label>
            <div class="relative group">
              <textarea readonly class="input font-mono text-[10px] h-32 resize-none bg-black/40">{{ generatedToken }}</textarea>
              <button @click="copyToken" class="absolute top-2 right-2 p-2 bg-surface-2 rounded-lg opacity-0 group-hover:opacity-100 transition-opacity hover:text-accent">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
              </button>
            </div>
          </div>

          <button @click="selectedTenant = null; generatedToken = null" class="btn-ghost w-full justify-center py-3">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useTenantStore } from '@/stores/tenants'

const tenantStore = useTenantStore()
const showAddModal = ref(false)
const selectedTenant = ref(null)
const generatedToken = ref(null)
const generating = ref(false)
const creating = ref(false)
const error = ref(null)
const licenseError = ref(null)

const newTenant = reactive({
  name: '',
  slug: ''
})

const licenseForm = reactive({
  tier: 'community',
  expires_in_days: 365,
  featuresInput: 'sso,branding,secrets,audit_logs,analytics'
})

onMounted(() => {
  tenantStore.fetchAll()
})

function openLicenseModal(tenant) {
  selectedTenant.value = tenant
  generatedToken.value = null
  licenseError.value = null
}

async function generateLicense() {
  generating.value = true
  licenseError.value = null
  try {
    const payload = {
      tier: licenseForm.tier,
      expires_in_days: licenseForm.expires_in_days,
      features: licenseForm.featuresInput.split(',').map(s => s.trim()).filter(Boolean)
    }
    const result = await tenantStore.generateLicense(selectedTenant.value.id, payload)
    generatedToken.value = result.token
  } catch (e) {
    licenseError.value = e.response?.data?.error ?? e.message
  } finally {
    generating.value = false
  }
}

function copyToken() {
  navigator.clipboard.writeText(generatedToken.value)
}

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
