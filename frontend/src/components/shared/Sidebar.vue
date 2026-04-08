<template>
  <aside class="w-14 flex flex-col items-center py-4 gap-1 border-r border-border bg-surface-1 shrink-0">
    <!-- Logo / Home -->
    <RouterLink to="/dashboard" title="Home" class="mb-6 w-9 h-9 rounded-lg bg-accent flex items-center justify-center shrink-0 cursor-pointer hover:opacity-80 transition-opacity">
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <circle cx="8" cy="3" r="2" fill="white"/>
        <circle cx="3" cy="13" r="2" fill="white"/>
        <circle cx="13" cy="13" r="2" fill="white"/>
        <line x1="8" y1="5" x2="3" y2="11" stroke="white" stroke-width="1.5"/>
        <line x1="8" y1="5" x2="13" y2="11" stroke="white" stroke-width="1.5"/>
      </svg>
    </RouterLink>

    <NavItem to="/dashboard" title="Dashboard">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
        <rect x="3" y="3" width="7" height="7" rx="1"/>
        <rect x="14" y="3" width="7" height="7" rx="1"/>
        <rect x="3" y="14" width="7" height="7" rx="1"/>
        <rect x="14" y="14" width="7" height="7" rx="1"/>
      </svg>
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

    <div class="flex-1"/>

    <NavItem to="/settings" title="Settings">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
        <circle cx="12" cy="12" r="3"/>
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>
      </svg>
    </NavItem>

    <!-- WS indicator -->
    <div :title="connected ? 'Live' : 'Disconnected'"
         :class="connected ? 'bg-green-500' : 'bg-red-500'"
         class="w-2 h-2 rounded-full mb-2 transition-colors"/>
  </aside>
</template>

<script setup>
import { RouterLink, useRoute } from 'vue-router'
import { useWebSocket } from '@/composables/useWebSocket'

const { connected } = useWebSocket()
const route = useRoute()
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
            : 'text-slate-500 hover:text-slate-200 hover:bg-white/5'
        ]
      }, slots.default)
    }
  }
})
</script>
