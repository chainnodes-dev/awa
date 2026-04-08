import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/stores/auth'

export const useWorkflowStore = defineStore('workflows', () => {
  const definitions = ref([])
  const loading = ref(false)
  const error = ref(null)

  async function fetchAll() {
    loading.value = true
    error.value = null
    try {
      const { data } = await api.get('/workflows')
      definitions.value = data ?? []
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function fetchOne(name, version) {
    const { data } = await api.get(`/workflows/${name}/${version}`)
    return data
  }

  async function save(yaml) {
    const { data } = await api.post('/workflows', { yaml })
    await fetchAll()
    return data
  }

  async function remove(name, version) {
    await api.delete(`/workflows/${name}/${version}`)
    definitions.value = definitions.value.filter(
      d => !(d.metadata.name === name && d.metadata.version === version)
    )
  }

  async function fetchVersions(name) {
    const { data } = await api.get(`/workflows/${name}/versions`)
    return data ?? []
  }

  async function fetchByVersion(name, versionNumber) {
    const { data } = await api.get(`/workflows/${name}/v/${versionNumber}`)
    return data
  }

  async function startRun(workflowName, workflowVersion, input = {}) {
    const { data } = await api.post('/runs', { workflow_name: workflowName, workflow_version: workflowVersion, input })
    return data
  }

  return { definitions, loading, error, fetchAll, fetchOne, fetchVersions, fetchByVersion, save, remove, startRun }
})
