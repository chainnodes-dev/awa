<template>
  <aside class="w-14 flex flex-col items-center py-4 gap-1 border-r border-border bg-surface-1 shrink-0">
    <!-- Logo / Home -->
    <RouterLink to="/dashboard" :title="entStore.branding?.name || 'Home'" class="mb-8 w-10 h-10 flex items-center justify-center shrink-0 cursor-pointer hover:scale-105 transition-transform overflow-hidden">
      <img :src="entStore.branding?.logo_url || '/logo.svg'" class="w-full h-full object-contain" />
    </RouterLink>

    <NavItem to="/dashboard" title="Dashboard">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
        <rect x="3" y="3" width="7" height="7" rx="1"/>
        <rect x="14" y="3" width="7" height="7" rx="1"/>
        <rect x="3" y="14" width="7" height="7" rx="1"/>
        <rect x="14" y="14" width="7" height="7" rx="1"/>
      </svg>
    </NavItem>

    <NavItem to="/inbox" title="Inbox" class="relative">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
        <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
        <polyline points="22,6 12,13 2,6"/>
      </svg>
      <div v-if="pendingCount > 0" class="absolute -top-1 -right-1 min-w-[14px] h-[14px] bg-red-500 text-[9px] font-bold text-white flex items-center justify-center rounded-full px-0.5 border border-surface-1">
        {{ pendingCount }}
      </div>
    </NavItem>

    <NavItem to="/designer" title="Designer">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
        <circle cx="12" cy="5" r="2"/>
        <circle cx="5" cy="19" r="2"/>
        <circle cx="19" cy="19" r="2"/>
        <line x1="12" y1="7" x2="5" y2="17" />
        <line x1="12" y1="7" x2="19" y2="17"/>
        <line x1="7" y1="19" x2="17" y2="19"/>
      </svg>
    </NavItem>

    <NavItem to="/mcp-servers" title="MCP Servers">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
        <rect x="2" y="2" width="20" height="6" rx="2"/>
        <rect x="2" y="10" width="20" height="6" rx="2"/>
        <circle cx="6" cy="5" r="1" fill="currentColor"/>
        <circle cx="6" cy="13" r="1" fill="currentColor"/>
        <line x1="2" y1="20" x2="22" y2="20"/>
        <line x1="6" y1="20" x2="6" y2="22"/>
        <line x1="18" y1="20" x2="18" y2="22"/>
      </svg>
    </NavItem>

    <NavItem v-if="authStore.hasRole('admin')" to="/users" title="Users">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
        <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
        <circle cx="9" cy="7" r="4"/>
        <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
        <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
      </svg>
    </NavItem>

    <NavItem v-if="authStore.hasRole('admin')" to="/usage" title="Usage & Reporting">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
        <rect x="2" y="5" width="20" height="14" rx="2"/><line x1="2" y1="10" x2="22" y2="10"/>
      </svg>
    </NavItem>


    <div class="flex-1"/>

    <NavItem to="/settings" title="Settings">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
        <circle cx="12" cy="12" r="3"/>
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
      </svg>
    </NavItem>

    <button @click="onLogout" title="Sign Out" class="w-9 h-9 rounded-lg flex items-center justify-center text-text-muted hover:text-red-400 hover:bg-red-500/10 transition-all mb-2">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
        <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/>
      </svg>
    </button>

    <!-- WS indicator -->
    <div :title="connected ? 'Live' : 'Disconnected'"
         :class="connected ? 'bg-green-500' : 'bg-red-500'"
         class="w-2 h-2 rounded-full mb-2 transition-colors"/>
  </aside>
</template>

<script setup>
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { computed, onMounted } from 'vue'
import { useWebSocket }      from '@/composables/useWebSocket'
import { useExecutionStore } from '@/stores/execution'
import { useAuthStore }      from '@/stores/auth'
import { useEnterpriseStore } from '@/stores/enterprise'

const { connected } = useWebSocket()
const route      = useRoute()
const execStore  = useExecutionStore()
const authStore  = useAuthStore()
const entStore   = useEnterpriseStore()
const router     = useRouter()

function onLogout() {
  authStore.logout()
  router.push('/login')
}

onMounted(() => {
  entStore.fetchBranding()
  entStore.fetchStatus()
})

const pendingCount = computed(() => execStore.pendingHITL.length)
</script>

<script>
import { defineComponent, h } from 'vue'
import { RouterLink, useLink } from 'vue-router'

// Inline NavItem to keep the file self-contained
export const NavItem = defineComponent({
  props: { to: String, title: String },
  setup(props, { slots }) {
    return () => {
      const { isActive } = useLink({ to: props.to })
      return h(RouterLink, {
        to: props.to,
        title: props.title,
        class: [
          'w-9 h-9 rounded-lg flex items-center justify-center transition-all',
          isActive.value
            ? 'bg-accent text-white'
            : 'text-text-muted hover:text-text hover:bg-white/5'
        ]
      }, slots.default)
    }
  }
})
</script>
