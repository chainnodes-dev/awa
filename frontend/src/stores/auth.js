import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'
import router from '@/router'

// Shared axios instance used by ALL stores.
// Automatically injects the Bearer token and handles 401 → refresh → retry.
export const api = axios.create({ baseURL: '/api/v1' })

// Track whether a refresh is in flight to avoid concurrent refresh races.
let refreshPromise = null

api.interceptors.request.use(config => {
  const token = sessionStorage.getItem('access_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

api.interceptors.response.use(
  res => res,
  async err => {
    const original = err.config
    if (err.response?.status === 401 && !original._retry) {
      original._retry = true
      try {
        if (!refreshPromise) {
          refreshPromise = useAuthStore().refresh().finally(() => { refreshPromise = null })
        }
        await refreshPromise
        const token = sessionStorage.getItem('access_token')
        original.headers.Authorization = `Bearer ${token}`
        return api(original)
      } catch {
        useAuthStore().logout()
        router.push('/login')
        return Promise.reject(err)
      }
    }
    return Promise.reject(err)
  }
)

export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)   // { id, username, role }
  const loading = ref(false)
  const error = ref(null)

  const isAuthenticated = computed(() => !!user.value && !!sessionStorage.getItem('access_token'))
  const role = computed(() => user.value?.role ?? null)

  function hasRole(allowed) {
    if (!isAuthenticated.value) return false
    const myRole = role.value
    if (myRole === 'super_admin') return true

    const check = (r) => {
      if (r === 'viewer') return true
      if (r === 'runner') return ['runner', 'operator', 'admin'].includes(myRole)
      if (r === 'editor') return ['editor', 'operator', 'admin'].includes(myRole)
      if (r === 'admin')  return myRole === 'admin'
      return myRole === r
    }

    if (Array.isArray(allowed)) return allowed.some(check)
    return check(allowed)
  }

  async function login(username, password) {
    loading.value = true
    error.value = null
    try {
      const { data } = await axios.post('/api/v1/auth/login', { username, password })
      sessionStorage.setItem('access_token', data.access_token)
      localStorage.setItem('refresh_token', data.refresh_token)
      user.value = data.user
    } catch (e) {
      error.value = e.response?.data?.error ?? 'Login failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function refresh() {
    const rt = localStorage.getItem('refresh_token')
    if (!rt) throw new Error('no refresh token')
    const { data } = await axios.post('/api/v1/auth/refresh', { refresh_token: rt })
    sessionStorage.setItem('access_token', data.access_token)
    localStorage.setItem('refresh_token', data.refresh_token)
  }

  async function fetchMe() {
    try {
      const { data } = await api.get('/users/me')
      user.value = data
    } catch {
      logout()
    }
  }

  function logout() {
    // Best-effort server-side revocation.
    const token = sessionStorage.getItem('access_token')
    if (token) {
      axios.post('/api/v1/auth/logout', null, {
        headers: { Authorization: `Bearer ${token}` }
      }).catch(() => {})
    }
    sessionStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    user.value = null
  }

  // On app boot, restore session from stored tokens.
  async function init() {
    if (sessionStorage.getItem('access_token')) {
      await fetchMe()
    } else if (localStorage.getItem('refresh_token')) {
      try {
        await refresh()
        await fetchMe()
      } catch {
        logout()
      }
    }
  }

  async function checkStatus() {
    const { data } = await axios.get('/api/v1/auth/status')
    return data
  }

  async function setup(username, password) {
    loading.value = true
    error.value = null
    try {
      await axios.post('/api/v1/auth/setup', { username, password })
    } catch (e) {
      error.value = e.response?.data?.error ?? 'Setup failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  return { user, loading, error, isAuthenticated, role, hasRole, login, refresh, logout, init, checkStatus, setup }
})
