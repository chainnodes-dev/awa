<template>
  <div class="min-h-screen flex items-center justify-center bg-surface-0">
    <div class="w-full max-w-sm bg-surface-1 rounded-xl shadow-lg p-8 space-y-6">
      <!-- Logo / title -->
      <div class="text-center space-y-2">
        <img src="/logo.svg" class="w-12 h-12 mx-auto mb-2 object-contain" />
        <h1 class="text-2xl font-bold text-text uppercase tracking-widest">Phaxa</h1>
        <p class="text-[10px] text-text-muted uppercase tracking-[0.2em] opacity-60">Logic & Orchestration</p>
      </div>

      <!-- Form -->
      <form @submit.prevent="submit" class="space-y-4">
        <div class="space-y-1">
          <label class="block text-sm font-medium text-text-muted" for="username">
            Username
          </label>
          <input
            id="username"
            v-model="form.username"
            type="text"
            autocomplete="username"
            required
            class="input"
          />
        </div>

        <div class="space-y-1">
          <label class="block text-sm font-medium text-text-muted" for="password">
            Password
          </label>
          <input
            id="password"
            v-model="form.password"
            type="password"
            autocomplete="current-password"
            required
            class="input"
          />
        </div>

        <!-- Error message -->
        <p v-if="authStore.error" class="text-sm text-red-500">
          {{ authStore.error }}
        </p>

        <button
          type="submit"
          :disabled="authStore.loading"
          class="w-full py-2.5 px-4 rounded-lg bg-accent text-white font-semibold
                 hover:brightness-110 active:scale-[0.98] disabled:opacity-50
                 transition-all shadow-sm"
        >
          {{ authStore.loading ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const router = useRouter()

const form = reactive({ username: '', password: '' })

onMounted(async () => {
  const status = await authStore.checkStatus()
  if (!status.initialized) {
    router.push('/setup')
  }
})

async function submit() {
  try {
    await authStore.login(form.username, form.password)
    router.push(router.currentRoute.value.query.redirect ?? '/dashboard')
  } catch {
    // Error already set in authStore.error
  }
}
</script>
