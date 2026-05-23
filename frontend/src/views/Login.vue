<template>
  <div class="min-h-screen flex items-center justify-center bg-gradient-to-r from-white to-black">
    <div class="w-full max-w-sm bg-white/60 dark:bg-zinc-900/60 backdrop-blur-xl border border-white/30 dark:border-zinc-800/30 rounded-2xl shadow-2xl p-8 space-y-6">
      <!-- Logo / title -->
      <div class="text-center space-y-2">
        <img src="/logo.png" class="w-16 h-16 mx-auto mb-2 object-contain" />
        <h1 class="text-2xl font-bold text-blue-600 dark:text-blue-400 uppercase tracking-widest">Chain Nodes</h1>
        <p class="text-[10px] text-blue-500 dark:text-blue-300 uppercase tracking-[0.2em] font-semibold opacity-90">Agentic Workflows</p>
      </div>

      <!-- Form -->
      <form @submit.prevent="submit" class="space-y-4">
        <div class="space-y-1">
          <label class="block text-sm font-semibold text-blue-600 dark:text-blue-400" for="username">
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
          <label class="block text-sm font-semibold text-blue-600 dark:text-blue-400" for="password">
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
        <p v-if="authStore.error" class="text-sm text-red-500 font-medium">
          {{ authStore.error }}
        </p>

        <button
          type="submit"
          :disabled="authStore.loading"
          class="w-full py-2.5 px-4 rounded-lg bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 text-white font-semibold
                 hover:brightness-110 active:scale-[0.98] disabled:opacity-50
                 transition-all shadow-sm flex items-center justify-center gap-2"
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
