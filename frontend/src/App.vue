<template>
  <!-- Show the full shell layout only when authenticated -->
  <div v-if="authStore.isAuthenticated" class="flex h-screen overflow-hidden bg-surface-0">
    <Sidebar />
    <div class="flex-1 flex flex-col min-w-0">
      <RouterView />
    </div>
  </div>

  <!-- Public pages (Login) get the full viewport -->
  <RouterView v-else />
</template>

<script setup>
import { watch } from 'vue'
import { useRoute } from 'vue-router'
import Sidebar from '@/components/shared/Sidebar.vue'
import { useWebSocket } from '@/composables/useWebSocket'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'

const authStore = useAuthStore()
useTheme() // Initialize theme
const route = useRoute()

// Boot WebSocket only when authenticated; reconnect when auth state changes.
const { connect, disconnect } = useWebSocket()

watch(
  () => authStore.isAuthenticated,
  (authenticated) => {
    if (authenticated) {
      connect()
    } else {
      disconnect()
    }
  },
  { immediate: true }
)
</script>
