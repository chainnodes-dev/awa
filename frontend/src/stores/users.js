import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from './auth'

export const useUserStore = defineStore('users', () => {
  const users = ref([])
  const loading = ref(false)

  async function fetchAll() {
    loading.value = true
    try {
      const { data } = await api.get('/users')
      users.value = data
    } finally {
      loading.value = false
    }
  }

  async function create(user) {
    const { data } = await api.post('/users', user)
    users.value.push(data)
    return data
  }

  async function updateRole(userId, role) {
    await api.put(`/users/${userId}/role`, { role })
    const user = users.value.find(u => u.id === userId)
    if (user) user.role = role
  }

  async function remove(userId) {
    await api.delete(`/users/${userId}`)
    users.value = users.value.filter(u => u.id !== userId)
  }

  return { users, loading, fetchAll, create, updateRole, remove }
})
