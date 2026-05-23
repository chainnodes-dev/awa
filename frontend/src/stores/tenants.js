import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from './auth'

export const useTenantStore = defineStore('tenants', () => {
  const tenants = ref([])
  const loading = ref(false)

  async function fetchAll() {
    loading.value = true
    try {
      const { data } = await api.get('/tenants')
      tenants.value = data
    } finally {
      loading.value = false
    }
  }

  async function create(tenant) {
    const { data } = await api.post('/tenants', tenant)
    tenants.value.push(data)
    return data
  }

  async function remove(id) {
    await api.delete(`/tenants/${id}`)
    tenants.value = tenants.value.filter(t => t.id !== id)
  }

  async function generateLicense(id, params) {
    const { data } = await api.post(`/tenants/${id}/license/generate`, params)
    return data
  }

  return { tenants, loading, fetchAll, create, remove, generateLicense }
})
