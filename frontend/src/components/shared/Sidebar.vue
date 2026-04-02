<template>
  <aside class="w-14 flex flex-col items-center py-4 gap-1 border-r border-border bg-surface-1 shrink-0">
    <!-- Logo -->
    <div class="mb-4 w-8 h-8 rounded-lg bg-accent flex items-center justify-center shrink-0">
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <circle cx="8" cy="3" r="2" fill="white"/>
        <circle cx="3" cy="13" r="2" fill="white"/>
        <circle cx="13" cy="13" r="2" fill="white"/>
        <line x1="8" y1="5" x2="3" y2="11" stroke="white" stroke-width="1.5"/>
        <line x1="8" y1="5" x2="13" y2="11" stroke="white" stroke-width="1.5"/>
      </svg>
    </div>

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

    <div class="flex-1"/>

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
