<template>
  <div class="flex flex-col h-full overflow-hidden">
    <header class="flex items-center gap-4 px-6 py-4 border-b border-border shrink-0">
      <h1 class="text-base font-semibold text-text">User Management</h1>
      <div class="flex-1"/>
      <button 
        @click="showAddModal = true" 
        :disabled="!entStore.hasFeature('user_management')"
        :class="entStore.hasFeature('user_management') ? 'btn-primary' : 'btn-disabled cursor-not-allowed opacity-50'"
        :title="!entStore.hasFeature('user_management') ? 'Multi-user setup requires a Pro or Enterprise license' : ''"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Add User
      </button>
    </header>

    <div v-if="!entStore.hasFeature('user_management')" class="mx-6 mt-4 p-3 bg-amber-500/10 border border-amber-500/20 rounded-lg flex items-center gap-3">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-amber-500">
        <path d="M12 15v2m0-8v4m0-6h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
      </svg>
      <div class="text-xs text-amber-200/80">
        <span class="font-semibold text-amber-500">Read-only Mode:</span> Multi-user management is an Enterprise feature. Please upgrade your license to add or manage team members.
      </div>
      <div class="flex-1" />
      <RouterLink to="/usage" class="text-xs font-semibold text-amber-500 hover:underline">Upgrade Now &rarr;</RouterLink>
    </div>

    <div class="flex-1 overflow-y-auto p-6">
      <div v-if="userStore.loading" class="text-text-muted text-sm py-8 text-center">Loading users…</div>
      
      <div v-else class="card overflow-hidden">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-border">
              <th class="text-left px-4 py-2.5 text-xs font-medium text-text-muted uppercase tracking-wider">Username</th>
              <th class="text-left px-4 py-2.5 text-xs font-medium text-text-muted uppercase tracking-wider">Role</th>
              <th class="text-left px-4 py-2.5 text-xs font-medium text-text-muted uppercase tracking-wider">User ID</th>
              <th/>
            </tr>
          </thead>
          <tbody>
            <tr v-for="user in userStore.users" :key="user.id" class="border-b border-border/50 hover:bg-white/3 transition-colors">
              <td class="px-4 py-3 text-text font-medium">{{ user.username }}</td>
              <td class="px-4 py-3">
                <span :class="roleBadgeClass(user.role)">{{ user.role }}</span>
              </td>
              <td class="px-4 py-3 font-mono text-[10px] text-text-muted">{{ user.id }}</td>
              <td class="px-4 py-3 text-right">
                <div class="flex justify-end gap-2">
                  <select 
                    v-if="user.id !== authStore.user?.id"
                    :value="user.role"
                    :disabled="!entStore.hasFeature('user_management')"
                    @change="e => updateRole(user, e.target.value)"
                    class="input text-xs py-1 px-2 w-28 bg-surface-2 disabled:opacity-40 disabled:cursor-not-allowed"
                  >
                    <option value="admin">admin</option>
                    <option value="editor">editor</option>
                    <option value="runner">runner</option>
                    <option value="viewer">viewer</option>
                  </select>
                  <button 
                    v-if="user.id !== authStore.user?.id"
                    @click="deleteUser(user)" 
                    :disabled="!entStore.hasFeature('user_management')"
                    class="btn-danger py-1 px-2 text-xs disabled:opacity-30 disabled:cursor-not-allowed disabled:grayscale"
                  >
                    Delete
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Add User Modal -->
    <div v-if="showAddModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm">
      <div class="card w-full max-w-sm p-6 shadow-2xl animate-in fade-in zoom-in duration-200">
        <h2 class="text-lg font-semibold text-slate-100 mb-6 font-display">Add New User</h2>
        <form @submit.prevent="createUser" class="space-y-4">
          <div class="space-y-1.5">
            <label class="text-xs font-medium text-text-muted">Username</label>
            <input v-model="newUser.username" type="text" required class="input" placeholder="e.g. alice_smith" />
          </div>
          <div class="space-y-1.5">
            <label class="text-xs font-medium text-text-muted">Temporary Password</label>
            <input v-model="newUser.password" type="password" required class="input" placeholder="••••••••" />
          </div>
          <div class="space-y-1.5">
            <label class="text-xs font-medium text-text-muted">Initial Role</label>
            <select v-model="newUser.role" class="input">
              <option value="viewer">Viewer (Read-only)</option>
              <option value="runner">Runner (Execute Workflows)</option>
              <option value="editor">Editor (Design Workflows)</option>
              <option value="admin">Admin (Full Control)</option>
            </select>
          </div>
          
          <div v-if="error" class="text-xs text-red-400 bg-red-400/10 p-2 rounded border border-red-400/20">
            {{ error }}
          </div>

          <div class="flex gap-3 pt-4">
            <button type="button" @click="showAddModal = false" class="btn-ghost flex-1">Cancel</button>
            <button type="submit" :disabled="creating" class="btn-primary flex-1 justify-center">
              {{ creating ? 'Creating...' : 'Create' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useUserStore } from '@/stores/users'
import { useAuthStore } from '@/stores/auth'
import { useEnterpriseStore } from '@/stores/enterprise'

const userStore = useUserStore()
const authStore = useAuthStore()
const entStore  = useEnterpriseStore()

const showAddModal = ref(false)
const creating = ref(false)
const error = ref(null)

const newUser = reactive({
  username: '',
  password: '',
  role: 'runner'
})

onMounted(() => {
  userStore.fetchAll()
})

async function createUser() {
  creating.value = true
  error.value = null
  try {
    await userStore.create({ ...newUser })
    showAddModal.value = false
    newUser.username = ''
    newUser.password = ''
    newUser.role = 'runner'
  } catch (e) {
    error.value = e.response?.data?.error ?? e.message
  } finally {
    creating.value = false
  }
}

async function updateRole(user, newRole) {
  try {
    await userStore.updateRole(user.id, newRole)
  } catch (e) {
    alert(e.response?.data?.error ?? e.message)
  }
}

async function deleteUser(user) {
  if (!confirm(`Are you sure you want to delete ${user.username}?`)) return
  try {
    await userStore.remove(user.id)
  } catch (e) {
    alert(e.response?.data?.error ?? e.message)
  }
}

function roleBadgeClass(role) {
  const base = 'badge'
  if (role === 'admin') return `${base} bg-red-500/10 text-red-400`
  if (role === 'super_admin') return `${base} bg-purple-500/10 text-purple-400`
  if (role === 'editor') return `${base} bg-blue-500/10 text-blue-400`
  if (role === 'runner') return `${base} bg-green-500/10 text-green-400`
  return `${base} bg-slate-500/10 text-text-muted`
}
</script>
