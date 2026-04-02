<template>
  <div class="min-h-screen flex items-center justify-center bg-surface-0">
    <div class="w-full max-w-sm bg-surface-1 rounded-xl shadow-lg p-8 space-y-6">
      <!-- Logo / title -->
      <div class="text-center space-y-1">
        <h1 class="text-2xl font-bold text-content-primary">ASM Platform</h1>
        <p class="text-sm text-content-secondary">Agentic State Machine</p>
      </div>

      <!-- Form -->
      <form @submit.prevent="submit" class="space-y-4">
        <div class="space-y-1">
          <label class="block text-sm font-medium text-content-secondary" for="username">
            Username
          </label>
          <input
            id="username"
            v-model="form.username"
            type="text"
            autocomplete="username"
            required
            class="w-full px-3 py-2 rounded-lg border border-border bg-surface-0 text-content-primary
                   focus:outline-none focus:ring-2 focus:ring-accent-primary"
          />
        </div>

        <div class="space-y-1">
          <label class="block text-sm font-medium text-content-secondary" for="password">
            Password
          </label>
          <input
            id="password"
            v-model="form.password"
            type="password"
            autocomplete="current-password"
            required
            class="w-full px-3 py-2 rounded-lg border border-border bg-surface-0 text-content-primary
                   focus:outline-none focus:ring-2 focus:ring-accent-primary"
          />
        </div>

        <!-- Error message -->
        <p v-if="authStore.error" class="text-sm text-red-500">
          {{ authStore.error }}
        </p>

        <button
          type="submit"
          :disabled="authStore.loading"
          class="w-full py-2 px-4 rounded-lg bg-accent-primary text-white font-medium
                 hover:bg-accent-primary/90 disabled:opacity-50 disabled:cursor-not-allowed
                 transition-colors"
        >
          {{ authStore.loading ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const router = useRouter()

const form = reactive({ username: '', password: '' })

async function submit() {
  try {
    await authStore.login(form.username, form.password)
    router.push(router.currentRoute.value.query.redirect ?? '/dashboard')
  } catch {
    // Error already set in authStore.error
  }
}
</script>
