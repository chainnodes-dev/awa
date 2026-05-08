<template>
  <div class="min-h-screen flex items-center justify-center bg-surface-0">
    <div class="w-full max-w-sm bg-surface-1 rounded-xl shadow-lg p-8 space-y-6">
      <div class="text-center space-y-1">
        <h1 class="text-2xl font-bold text-text">Platform Setup</h1>
        <p class="text-sm text-text-muted">Create the initial administrator account</p>
      </div>

      <form @submit.prevent="submit" class="space-y-4">
        <div class="space-y-1">
          <label class="block text-sm font-medium text-text-muted" for="username">
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
          <label class="block text-sm font-medium text-text-muted" for="password">
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
          <label class="block text-sm font-medium text-text-muted" for="confirm">
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

        <p v-if="error" class="text-sm text-red-500">
          {{ error }}
        </p>

        <button
          type="submit"
          :disabled="loading"
          class="btn-primary w-full justify-center py-2.5"
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
