import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from './auth'

export const useEnterpriseStore = defineStore('enterprise', () => {
  const status = ref(null)
  const loading = ref(false)
  const branding = ref({ name: 'Phaxa', logo_url: '/logo.svg' })
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
      branding.value = data
    } catch (e) {
      console.error('Failed to fetch branding', e)
    }
  }

  async function updateBranding(payload) {
    await api.post('/enterprise/branding', payload)
    await fetchBranding()
  }

  async function fetchSecrets() {
    const { data } = await api.get('/enterprise/secrets')
    secrets.value = data
  }

  async function updateSecrets(payload) {
    await api.post('/enterprise/secrets', payload)
    await fetchSecrets()
  }

  async function fetchAuditLogs() {
    const { data } = await api.get('/enterprise/audit-logs')
    auditLogs.value = data
  }

  async function fetchAnalytics() {
    const { data } = await api.get('/enterprise/analytics')
    analytics.value = data
  }

  async function setLicense(token) {
    await api.post('/enterprise/license', { token })
    await fetchStatus()
  }

  return { 
    status, branding, secrets, auditLogs, analytics, loading, 
    fetchStatus, fetchBranding, updateBranding, 
    fetchSecrets, updateSecrets, fetchAuditLogs, fetchAnalytics,
    setLicense 
  }
})
