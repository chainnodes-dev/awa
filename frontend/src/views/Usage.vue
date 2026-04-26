<template>
  <div class="flex flex-col h-full overflow-y-auto bg-surface-0">
    <header class="px-6 py-4 border-b border-border shrink-0 bg-surface-1 flex items-center justify-between">
      <div>
        <h1 class="text-base font-semibold text-text">Usage & Reporting</h1>
        <p class="text-xs text-text-muted mt-0.5">Monitor subscription limits, licenses, and system audit logs</p>
      </div>
      <div v-if="entStore.status" class="flex items-center gap-2">
        <span class="text-[10px] font-bold uppercase tracking-widest text-text-muted">Tier</span>
        <span :class="tierBadgeClass(entStore.status.tier)">{{ entStore.status.tier }}</span>
      </div>
    </header>

    <div class="flex-1 px-6 py-6 max-w-4xl w-full mx-auto space-y-8">
      <!-- Subscription & Usage -->
      <section class="space-y-4">
        <div class="flex items-center gap-2 px-1">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-indigo-400">
            <rect x="2" y="5" width="20" height="14" rx="2"/><line x1="2" y1="10" x2="22" y2="10"/>
          </svg>
          <h2 class="text-sm font-bold text-text uppercase tracking-wider">Subscription Limits</h2>
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div class="card p-5 space-y-3">
            <div class="flex justify-between items-baseline">
              <span class="text-xs font-semibold text-text-muted">Workflow Definitions</span>
              <span class="text-xs font-mono text-text">
                {{ entStore.status?.current_workflows ?? 0 }} / {{ entStore.status?.max_workflows || '∞' }}
              </span>
            </div>
            <div class="h-1.5 w-full bg-black/20 rounded-full overflow-hidden">
              <div class="h-full bg-indigo-500 transition-all duration-500" :style="{ width: usagePercent(entStore.status?.current_workflows, entStore.status?.max_workflows) + '%' }"/>
            </div>
          </div>

          <div class="card p-5 space-y-3">
            <div class="flex justify-between items-baseline">
              <span class="text-xs font-semibold text-text-muted">Monthly Runs</span>
              <span class="text-xs font-mono text-text">
                {{ entStore.status?.current_runs ?? 0 }} / {{ entStore.status?.max_runs || '∞' }}
              </span>
            </div>
            <div class="h-1.5 w-full bg-black/20 rounded-full overflow-hidden">
              <div class="h-full bg-emerald-500 transition-all duration-500" :style="{ width: usagePercent(entStore.status?.current_runs, entStore.status?.max_runs) + '%' }"/>
            </div>
          </div>
        </div>
      </section>

      <!-- License Key -->
      <section class="card p-5">
        <div class="flex items-center justify-between mb-4">
          <div>
            <h3 class="text-xs font-bold text-text uppercase tracking-wide">License Management</h3>
            <p class="text-[10px] text-text-muted">Manage your enterprise token and validity.</p>
          </div>
          <button @click="updatingLicense = !updatingLicense" class="btn-ghost text-[10px] uppercase font-bold">
            {{ updatingLicense ? 'Cancel' : 'Update Key' }}
          </button>
        </div>
        
        <div v-if="updatingLicense" class="space-y-3">
          <textarea v-model="licenseInput" class="input text-[10px] font-mono h-24 resize-none" placeholder="Paste signed JWT token..."></textarea>
          <div class="flex justify-end">
            <button @click="saveLicense" :disabled="licenseBusy || !licenseInput" class="btn-primary text-xs px-6">
              {{ licenseBusy ? 'Applying...' : 'Apply License' }}
            </button>
          </div>
        </div>
        <div v-else-if="entStore.status?.tier && entStore.status.tier !== 'invalid'" class="flex items-center justify-between bg-emerald-500/5 border border-emerald-500/20 p-3 rounded-lg">
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 rounded-full bg-emerald-500/10 flex items-center justify-center text-emerald-500">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><path d="M9 12l2 2 4-4"/></svg>
            </div>
            <div>
              <div class="text-[11px] font-bold text-text uppercase tracking-wider">License Active: {{ entStore.status.tier }}</div>
              <div class="text-[10px] text-text-muted">Valid until {{ formatDateTime(entStore.status.expires_at) }}</div>
            </div>
          </div>
          <div class="text-[10px] font-mono text-text-muted opacity-50">
            ID: {{ entStore.status.tenant_id?.slice(0,8) || '...' }}
          </div>
        </div>
        <div v-else class="flex items-center gap-2 text-[10px] font-mono text-text-muted bg-black/10 p-3 rounded-lg border border-dashed border-border">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
          {{ entStore.status?.tier === 'invalid' ? 'Invalid or Expired License Key' : 'No license key configured' }}
        </div>
      </section>

      <!-- Audit Logs -->
      <section class="space-y-4">
        <div class="flex items-center gap-2 px-1">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-indigo-400">
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
          </svg>
          <h2 class="text-sm font-bold text-text uppercase tracking-wider">Security Audit Trail</h2>
        </div>

        <div class="card overflow-hidden">
          <table class="w-full text-[11px]">
            <thead class="bg-surface-2 text-text-muted uppercase tracking-wider font-semibold">
              <tr>
                <th class="text-left px-4 py-2 w-1/4">Time</th>
                <th class="text-left px-4 py-2 w-1/4">Action</th>
                <th class="text-left px-4 py-2 w-1/4">User</th>
                <th class="text-left px-4 py-2">Details</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border/20">
              <tr v-for="log in entStore.auditLogs" :key="log.id" class="hover:bg-white/1">
                <td class="px-4 py-2.5 text-text-muted">{{ formatDateTime(log.created_at) }}</td>
                <td class="px-4 py-2.5">
                  <span class="px-1.5 py-0.5 rounded-full bg-slate-800 text-[10px] font-mono text-text">{{ log.action }}</span>
                </td>
                <td class="px-4 py-2.5 text-text-muted">{{ log.user_id?.slice(0, 8) || 'SYSTEM' }}</td>
                <td class="px-4 py-2.5 text-text-muted truncate" :title="JSON.stringify(log.details)">
                  {{ log.details ? JSON.stringify(log.details) : '-' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useEnterpriseStore } from '@/stores/enterprise'

const entStore = useEnterpriseStore()
const updatingLicense = ref(false)
const licenseInput = ref('')
const licenseBusy = ref(false)

async function saveLicense() {
  if (!licenseInput.value) return
  licenseBusy.value = true
  try {
    await entStore.setLicense(licenseInput.value)
    updatingLicense.value = false
    licenseInput.value = ''
  } catch (e) {
    alert('License update failed: ' + (e.response?.data?.error ?? e.message))
  } finally {
    licenseBusy.value = false
  }
}

function usagePercent(curr, max) {
  if (!max) return 0
  return Math.min(100, Math.round((curr / max) * 100))
}

function tierBadgeClass(tier) {
  if (tier === 'enterprise') return 'badge bg-purple-500/20 text-purple-400'
  if (tier === 'pro') return 'badge bg-indigo-500/20 text-indigo-400'
  return 'badge bg-slate-500/20 text-text-muted'
}

function formatDateTime(ts) {
  if (!ts) return '-'
  return new Date(ts).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

onMounted(() => {
  entStore.fetchStatus()
  entStore.fetchAuditLogs()
})
</script>
