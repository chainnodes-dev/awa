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
    path: '/setup',
    name: 'Setup',
    component: () => import('@/views/Setup.vue'),
    meta: { public: true, title: 'Initial Setup' }
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
    path: '/inbox',
    name: 'Inbox',
    component: () => import('@/views/Inbox.vue'),
    meta: { title: 'Inbox' }
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
  },
  {
    path: '/users',
    name: 'Users',
    component: () => import('@/views/Users.vue'),
    meta: { title: 'Users', role: 'admin' }
  },
  {
    path: '/usage',
    name: 'Usage',
    component: () => import('@/views/Usage.vue'),
    meta: { title: 'Usage & Reporting', role: 'admin' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Navigation guard: redirect unauthenticated users to /login or /setup.
router.beforeEach(async (to) => {
  const authStore = useAuthStore()

  // 1. If not authenticated, check if the system needs initial setup.
  // We skip this check ONLY if we are already going to /setup.
  if (!authStore.isAuthenticated && to.name !== 'Setup') {
    try {
      const { initialized } = await authStore.checkStatus()
      if (!initialized) return { name: 'Setup' }
    } catch (e) {
      console.error('Platform status check failed', e)
    }
  }

  // 2. Allow public pages (Login, Setup) once system is initialized.
  if (to.meta.public) return true

  // 3. Restore session from stored tokens.
  if (!authStore.isAuthenticated) {
    await authStore.init()
  }

  // 4. Force Login if still unauthenticated.
  if (!authStore.isAuthenticated) {
    return { name: 'Login', query: { redirect: to.fullPath } }
  }

  // 5. Role-based guarding.
  if (to.meta.role && !authStore.hasRole(to.meta.role)) {
    return { name: 'Dashboard' }
  }

  return true
})

// Need to import useEnterpriseStore
import { useEnterpriseStore } from '@/stores/enterprise'

export default router
