import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { public: true, title: 'Sign In' }
  },
  {
    path: '/',
    redirect: '/dashboard'
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('@/views/Dashboard.vue'),
    meta: { title: 'Dashboard' }
  },
  {
    path: '/designer',
    name: 'DesignerNew',
    component: () => import('@/views/Designer.vue'),
    meta: { title: 'Designer' }
  },
  {
    path: '/designer/:name/:version',
    name: 'Designer',
    component: () => import('@/views/Designer.vue'),
    meta: { title: 'Designer' }
  },
  {
    path: '/monitor/:id',
    name: 'Monitor',
    component: () => import('@/views/Monitor.vue'),
    meta: { title: 'Monitor' }
  },
  {
    path: '/mcp-servers',
    name: 'MCPServers',
    component: () => import('@/views/MCPServers.vue'),
    meta: { title: 'MCP Servers' }
  },
  {
    path: '/settings',
    name: 'Settings',
    component: () => import('@/views/Settings.vue'),
    meta: { title: 'Settings' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Navigation guard: redirect unauthenticated users to /login.
router.beforeEach(async (to) => {
  if (to.meta.public) return true

  const authStore = useAuthStore()

  // First navigation: try to restore session from stored tokens.
  if (!authStore.isAuthenticated) {
    await authStore.init()
  }

  if (!authStore.isAuthenticated) {
    return { name: 'Login', query: { redirect: to.fullPath } }
  }

  return true
})

export default router
