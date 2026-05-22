<template>
  <div class="min-h-screen flex items-center justify-center bg-gradient-to-r from-white to-black">
    <div class="w-full max-w-sm bg-white/60 dark:bg-zinc-900/60 backdrop-blur-xl border border-white/30 dark:border-zinc-800/30 rounded-2xl shadow-2xl p-8 space-y-6">
      <!-- Logo / title -->
      <div class="text-center space-y-2">
        <img src="/logo.png" class="w-16 h-16 mx-auto mb-2 object-contain" />
        <h1 class="text-2xl font-bold text-blue-600 dark:text-blue-400 uppercase tracking-widest">Chain Nodes</h1>
        <p class="text-[10px] text-blue-500 dark:text-blue-300 uppercase tracking-[0.2em] font-semibold opacity-90">Platform Setup</p>
      </div>

      <form @submit.prevent="submit" class="space-y-4">
        <div class="space-y-1">
          <label class="block text-sm font-semibold text-blue-600 dark:text-blue-400" for="username">
            Admin Username
          </label>
          <input
            id="username"
            v-model="form.username"
            type="text"
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
            required
            class="input"
          />
        </div>

        <div class="space-y-1">
          <label class="block text-sm font-semibold text-blue-600 dark:text-blue-400" for="confirm">
            Confirm Password
          </label>
          <input
            id="confirm"
            v-model="form.confirm"
            type="password"
            required
            class="input"
          />
        </div>

        <p v-if="error" class="text-sm text-red-500 font-medium">
          {{ error }}
        </p>

        <button
          type="submit"
          :disabled="loading"
          class="w-full py-2.5 px-4 rounded-lg bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 text-white font-semibold
                 hover:brightness-110 active:scale-[0.98] disabled:opacity-50
                 transition-all shadow-sm flex items-center justify-center gap-2"
        >
          {{ loading ? 'Configuring…' : 'Initialize Platform' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const router = useRouter()
const loading = ref(false)
const error = ref(null)

const form = reactive({ username: '', password: '', confirm: '' })

async function submit() {
  if (form.password !== form.confirm) {
    error.value = "Passwords do not match"
    return
  }
  loading.value = true
  error.value = null
  try {
    await authStore.setup(form.username, form.password)
    router.push('/login')
  } catch (e) {
    error.value = e.response?.data?.error ?? 'Setup failed'
  } finally {
    loading.value = false
  }
}
</script>
