import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from './auth'

export const useEnterpriseStore = defineStore('enterprise', () => {
  const status = ref(null)
  const loading = ref(false)
  const branding = ref({ name: 'Chain Nodes', logo_url: '/logo.png' })
  const secrets = ref({})
  const auditLogs = ref([])
  const analytics = ref({})

  async function fetchStatus() {
    loading.value = true
    try {
      const { data } = await api.get('/enterprise/status')
      status.value = data
    } finally {
      loading.value = false
    }
  }

  async function fetchBranding() {
    try {
      const { data } = await api.get('/enterprise/branding')
      if (data.logo_url) {
        branding.value = data
      } else {
        branding.value = { ...branding.value, name: data.name || 'Chain Nodes' }
      }
    } catch (e) {
      // Silence 403 (Forbidden) errors as they are expected for Free tier
      if (e.response?.status !== 403) {
        console.error('Failed to fetch branding', e)
      }
    }
  }

  async function updateBranding(payload) {
    await api.post('/enterprise/branding', payload)
    await fetchBranding()
  }

  async function fetchSecrets() {
    try {
      const { data } = await api.get('/enterprise/secrets')
      secrets.value = data
    } catch (e) {
      if (e.response?.status !== 403) {
        console.error('Failed to fetch secrets', e)
      }
    }
  }

  async function updateSecrets(payload) {
    await api.post('/enterprise/secrets', payload)
    await fetchSecrets()
  }

  async function fetchAuditLogs() {
    try {
      const { data } = await api.get('/enterprise/audit-logs')
      auditLogs.value = data
    } catch (e) {
      if (e.response?.status !== 403) {
        console.error('Failed to fetch audit logs', e)
      }
    }
  }

  async function fetchAnalytics() {
    try {
      const { data } = await api.get('/enterprise/analytics')
      analytics.value = data
    } catch (e) {
      if (e.response?.status !== 403) {
        console.error('Failed to fetch analytics', e)
      }
    }
  }

  async function setLicense(token) {
    await api.post('/enterprise/license', { token })
    await fetchStatus()
  }

  const hasFeature = (name) => {
    if (!status.value) return false
    if (status.value.tier === 'enterprise') return true
    return status.value.features?.includes(name) || false
  }

  return { 
    status, branding, secrets, auditLogs, analytics, loading, hasFeature,
    fetchStatus, fetchBranding, updateBranding, 
    fetchSecrets, updateSecrets, fetchAuditLogs, fetchAnalytics,
    setLicense 
  }
})
