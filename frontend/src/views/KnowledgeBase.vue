<template>
  <div class="flex flex-col h-full overflow-hidden bg-surface-0">
    <!-- Header -->
    <header class="sticky top-0 z-10 flex items-center justify-between px-6 py-4 bg-surface-1/95 backdrop-blur-sm border-b border-border shrink-0">
      <div class="flex flex-col">
        <div class="flex items-center gap-2.5">
          <h1 class="text-base font-bold text-text tracking-tight uppercase">Knowledge Base</h1>
          <span class="px-2 py-0.5 rounded-full bg-accent/10 text-accent text-[10px] font-bold font-mono tracking-wider">VECTOR RAG</span>
        </div>
        <p class="text-[10px] text-text-muted font-bold uppercase tracking-widest opacity-60">Embedded Semantic Document Store</p>
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center gap-3">
        <button 
          @click="showPlayground = !showPlayground"
          :class="['btn-ghost h-9 px-4 text-xs gap-2 border transition-all', 
                   showPlayground ? 'bg-accent text-white border-accent' : 'border-border bg-surface-2/60 text-text']"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
          {{ showPlayground ? 'Close Search Playground' : 'Test Search Playground' }}
        </button>

        <button 
          v-if="canManage" 
          @click="showUploadModal = true" 
          class="btn-primary h-9 px-4 text-xs gap-2 shadow-xl shadow-indigo-500/20"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
          Upload Document
        </button>
      </div>
    </header>

    <!-- Main Content Area -->
    <main class="flex-1 overflow-y-auto p-6 scroll-smooth custom-scrollbar">
      <div class="max-w-7xl mx-auto space-y-6">

        <!-- Search Playground Drawer (When Open) -->
        <div v-if="showPlayground" class="p-6 bg-surface-1 border border-accent/30 rounded-2xl shadow-xl space-y-5 animate-in fade-in duration-200">
          <div class="flex items-center justify-between border-b border-border pb-3">
            <div class="flex items-center gap-2">
              <span class="w-2.5 h-2.5 rounded-full bg-accent animate-pulse"/>
              <h2 class="text-sm font-bold text-text uppercase tracking-wider">Semantic Search & Citation Playground</h2>
            </div>
            <span class="text-[10px] text-text-muted font-mono">Simulates deterministic `knowledge:search` tool</span>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-4 gap-4 items-end">
            <div class="md:col-span-2 space-y-1">
              <label class="text-[10px] font-bold uppercase text-text-muted tracking-widest">Query / Question</label>
              <input 
                v-model="searchQuery" 
                @keyup.enter="runSearch"
                type="text" 
                placeholder="e.g. What are the payment terms in vendor contract?" 
                class="input text-xs"
              />
            </div>

            <div class="space-y-1">
              <label class="text-[10px] font-bold uppercase text-text-muted tracking-widest">Collection Filter</label>
              <select v-model="searchCollection" class="input text-xs font-mono">
                <option value="">All Collections</option>
                <option v-for="c in collections" :key="c" :value="c">{{ c }}</option>
              </select>
            </div>

            <div class="flex items-center gap-2">
              <div class="flex-1 space-y-1">
                <label class="text-[10px] font-bold uppercase text-text-muted tracking-widest">Top K: {{ searchTopK }}</label>
                <input v-model.number="searchTopK" type="range" min="1" max="10" class="w-full accent-indigo-500" />
              </div>
              <button @click="runSearch" :disabled="searching || !searchQuery.trim()" class="btn-primary h-10 px-5 text-xs font-bold uppercase tracking-wider shrink-0">
                {{ searching ? 'Searching...' : 'Search' }}
              </button>
            </div>
          </div>

          <!-- Search Results View -->
          <div v-if="searchResults !== null" class="space-y-3 pt-2">
            <div class="flex items-center justify-between text-xs font-mono text-text-muted">
              <span>Matches: <strong class="text-text">{{ searchResults.length }}</strong></span>
              <span v-if="searchResults.length === 0" class="text-amber-400">No chunks matched above threshold</span>
            </div>

            <div v-if="searchResults.length > 0" class="grid grid-cols-1 gap-3">
              <div 
                v-for="res in searchResults" 
                :key="res.chunk_id"
                class="p-4 rounded-xl border border-border bg-surface-2/60 space-y-2 hover:border-accent/40 transition-colors"
              >
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-2">
                    <span class="px-2 py-0.5 rounded bg-accent/15 text-accent text-[11px] font-bold font-mono">
                      {{ res.citation }}
                    </span>
                    <span class="px-2 py-0.5 rounded bg-surface-1 border border-border text-text-muted text-[10px] uppercase font-bold">
                      {{ res.collection }}
                    </span>
                  </div>
                  <span class="text-[11px] font-mono font-bold text-green-400 bg-green-500/10 px-2 py-0.5 rounded">
                    Score: {{ Math.round(res.score * 100) }}%
                  </span>
                </div>
                <p class="text-xs text-text leading-relaxed font-sans whitespace-pre-wrap bg-surface-1/50 p-3 rounded-lg border border-border/40">
                  {{ res.content }}
                </p>
              </div>
            </div>
          </div>
        </div>

        <!-- Collections Tab Filter -->
        <div class="flex items-center justify-between gap-4">
          <div class="flex items-center gap-1.5 overflow-x-auto pb-1 scrollbar-none">
            <button 
              @click="activeCollection = 'all'" 
              :class="['px-3.5 py-1.5 rounded-lg text-xs font-bold uppercase tracking-wider transition-all',
                       activeCollection === 'all' ? 'bg-accent text-white shadow-md shadow-indigo-500/20' : 'bg-surface-1 text-text-muted hover:text-text border border-border']"
            >
              All Documents
              <span class="ml-1.5 px-1.5 py-0.2 rounded-full text-[9px] bg-black/20">{{ documents.length }}</span>
            </button>
            <button 
              v-for="c in collections" 
              :key="c"
              @click="activeCollection = c"
              :class="['px-3.5 py-1.5 rounded-lg text-xs font-bold uppercase tracking-wider transition-all',
                       activeCollection === c ? 'bg-accent text-white shadow-md shadow-indigo-500/20' : 'bg-surface-1 text-text-muted hover:text-text border border-border']"
            >
              {{ c }}
            </button>
          </div>

          <button @click="fetchDocuments" class="btn-ghost text-xs p-2 text-text-muted hover:text-text" title="Refresh">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/><path d="M16 16h5v5"/></svg>
          </button>
        </div>

        <!-- Documents List -->
        <div v-if="loading" class="flex flex-col items-center justify-center py-24 gap-3">
          <div class="w-8 h-8 border-4 border-accent/20 border-t-accent rounded-full animate-spin"/>
          <p class="text-xs text-text-muted font-mono uppercase tracking-widest animate-pulse">Loading Knowledge Base...</p>
        </div>

        <div v-else-if="filteredDocuments.length === 0" class="flex flex-col items-center justify-center py-24 text-center border-2 border-dashed border-border rounded-2xl bg-surface-1/40">
          <div class="w-16 h-16 rounded-full bg-surface-2 flex items-center justify-center mb-4 border border-border shadow-inner">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-text-muted opacity-40"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1-2.5-2.5Z"/><path d="M6 6h10"/><path d="M6 10h10"/></svg>
          </div>
          <h3 class="text-base font-bold text-text mb-1 uppercase tracking-tight">No Documents in Collection</h3>
          <p class="text-xs text-text-muted mb-6 max-w-sm">Upload company policies, contracts, or guidelines to ground agent execution with verified citations.</p>
          <button v-if="canManage" @click="showUploadModal = true" class="btn-primary px-5 py-2 text-xs">Upload First Document</button>
        </div>

        <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <div 
            v-for="doc in filteredDocuments" 
            :key="doc.id"
            class="p-5 rounded-2xl border border-border bg-surface-1 hover:border-accent/40 transition-all flex flex-col justify-between group"
          >
            <div>
              <!-- Card Header -->
              <div class="flex items-start justify-between gap-3 mb-3">
                <div class="flex items-center gap-3 min-w-0">
                  <div class="w-10 h-10 rounded-xl bg-surface-2 flex items-center justify-center border border-border shrink-0">
                    <svg v-if="doc.filename.endsWith('.pdf')" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-red-400"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>
                    <svg v-else width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="text-indigo-400"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>
                  </div>
                  <div class="min-w-0">
                    <h3 class="font-bold text-xs text-text truncate" :title="doc.filename">{{ doc.filename }}</h3>
                    <span class="text-[9px] font-bold px-1.5 py-0.5 rounded uppercase tracking-wider bg-surface-2 text-text-muted border border-border">
                      {{ doc.collection }}
                    </span>
                  </div>
                </div>

                <!-- Status Badge -->
                <div>
                  <span v-if="doc.status === 'ready'" class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[9px] font-bold uppercase bg-green-500/10 text-green-400 border border-green-500/20">
                    <span class="w-1.5 h-1.5 rounded-full bg-green-500"></span> Ready
                  </span>
                  <span v-else-if="doc.status === 'indexing'" class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[9px] font-bold uppercase bg-amber-500/10 text-amber-400 border border-amber-500/20 animate-pulse">
                    <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span> Indexing
                  </span>
                  <span v-else class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[9px] font-bold uppercase bg-red-500/10 text-red-400 border border-red-500/20" :title="doc.error_msg">
                    <span class="w-1.5 h-1.5 rounded-full bg-red-400"></span> Error
                  </span>
                </div>
              </div>

              <!-- Metadata Specs -->
              <div class="grid grid-cols-3 gap-2 py-3 border-y border-border/50 text-[10px] font-mono text-center">
                <div class="bg-surface-2/40 rounded p-1.5">
                  <div class="text-text font-bold">{{ doc.page_count }}</div>
                  <div class="text-text-muted text-[8px] uppercase">Pages</div>
                </div>
                <div class="bg-surface-2/40 rounded p-1.5">
                  <div class="text-text font-bold">{{ doc.chunk_count }}</div>
                  <div class="text-text-muted text-[8px] uppercase">Chunks</div>
                </div>
                <div class="bg-surface-2/40 rounded p-1.5">
                  <div class="text-text font-bold">{{ formatBytes(doc.byte_size) }}</div>
                  <div class="text-text-muted text-[8px] uppercase">Size</div>
                </div>
              </div>
            </div>

            <!-- Card Footer -->
            <div class="flex items-center justify-between pt-3 mt-3 text-[10px] text-text-muted">
              <span>Added {{ formatDate(doc.created_at) }}</span>
              <button 
                v-if="canManage" 
                @click="deleteDoc(doc)" 
                class="text-red-400 hover:text-red-300 p-1 rounded hover:bg-red-500/10 transition-colors"
                title="Delete document and chunks"
              >
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
              </button>
            </div>
          </div>
        </div>

      </div>
    </main>

    <!-- Upload Document Modal -->
    <div v-if="showUploadModal" class="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center z-50 p-4" @click.self="showUploadModal = false">
      <div class="card w-full max-w-lg p-6 space-y-5 bg-surface-1 border border-border rounded-2xl shadow-2xl animate-in fade-in zoom-in-95 duration-200">
        <div class="flex items-center justify-between border-b border-border pb-3">
          <div class="flex items-center gap-2.5">
            <div class="w-8 h-8 rounded-lg bg-accent/15 flex items-center justify-center text-accent">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
            </div>
            <h3 class="text-sm font-bold text-text uppercase tracking-tight">Upload Knowledge Document</h3>
          </div>
          <button @click="showUploadModal = false" class="text-text-muted hover:text-text p-1">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>

        <form @submit.prevent="submitUpload" class="space-y-4">
          <div class="space-y-1.5">
            <label class="text-[10px] uppercase font-bold text-text-muted tracking-widest">Document File (PDF, TXT, MD, CSV)</label>
            <input 
              ref="fileInput" 
              type="file" 
              required
              accept=".pdf,.txt,.md,.markdown,.csv,.json"
              @change="onFileSelected"
              class="block w-full text-xs text-text-muted file:mr-4 file:py-2 file:px-4 file:rounded-xl file:border-0 file:text-xs file:font-bold file:bg-surface-2 file:text-text hover:file:bg-surface-2/80 cursor-pointer"
            />
          </div>

          <div class="space-y-1.5">
            <label class="text-[10px] uppercase font-bold text-text-muted tracking-widest">Collection Tag</label>
            <input 
              v-model="uploadCollection" 
              type="text" 
              required
              placeholder="e.g. hr-policies, contracts, security, default" 
              class="input text-xs font-mono"
            />
            <p class="text-[9px] text-text-muted">Target collection partition for vector similarity searches</p>
          </div>

          <div v-if="uploadError" class="p-3 bg-red-500/10 border border-red-500/20 text-red-400 text-xs rounded-xl">
            {{ uploadError }}
          </div>

          <div class="flex justify-end gap-2 pt-2 border-t border-border">
            <button type="button" @click="showUploadModal = false" class="btn-ghost text-xs px-4 py-2">Cancel</button>
            <button type="submit" :disabled="uploading || !selectedFile" class="btn-primary text-xs px-6 py-2 font-bold uppercase tracking-wider flex items-center gap-1.5">
              {{ uploading ? 'Uploading...' : 'Ingest & Index' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { api, useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const canManage = computed(() => authStore.hasRole(['admin', 'editor', 'super_admin']))

const documents = ref([])
const collections = ref([])
const activeCollection = ref('all')
const loading = ref(true)

// Upload Modal State
const showUploadModal = ref(false)
const selectedFile = ref(null)
const uploadCollection = ref('default')
const uploading = ref(false)
const uploadError = ref('')

// Search Playground State
const showPlayground = ref(false)
const searchQuery = ref('')
const searchCollection = ref('')
const searchTopK = ref(4)
const searching = ref(false)
const searchResults = ref(null)

let pollTimer = null

const filteredDocuments = computed(() => {
  if (activeCollection.value === 'all') {
    return documents.value
  }
  return documents.value.filter(d => d.collection === activeCollection.value)
})

async function fetchDocuments() {
  try {
    const { data } = await api.get('/knowledge/documents')
    documents.value = data ?? []
  } catch (e) {
    console.error('Failed to load knowledge documents', e)
  } finally {
    loading.value = false
  }
}

async function fetchCollections() {
  try {
    const { data } = await api.get('/knowledge/collections')
    collections.value = data ?? []
  } catch (e) {
    console.error('Failed to load collections', e)
  }
}

function onFileSelected(e) {
  selectedFile.value = e.target.files[0] || null
}

async function submitUpload() {
  if (!selectedFile.value) return
  uploading.value = true
  uploadError.value = ''

  try {
    const formData = new FormData()
    formData.append('file', selectedFile.value)
    formData.append('collection', uploadCollection.value.trim() || 'default')

    await api.post('/knowledge/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })

    showUploadModal.value = false
    selectedFile.value = null
    uploadCollection.value = 'default'
    await fetchDocuments()
    await fetchCollections()
  } catch (e) {
    uploadError.value = e.response?.data?.error || e.message
  } finally {
    uploading.value = false
  }
}

async function deleteDoc(doc) {
  if (!confirm(`Are you sure you want to delete "${doc.filename}" and its vector index?`)) return
  try {
    await api.delete(`/knowledge/documents/${doc.id}`)
    documents.value = documents.value.filter(d => d.id !== doc.id)
    await fetchCollections()
  } catch (e) {
    alert('Failed to delete document: ' + (e.response?.data?.error || e.message))
  }
}

async function runSearch() {
  if (!searchQuery.value.trim()) return
  searching.value = true
  try {
    const { data } = await api.post('/knowledge/search', {
      query: searchQuery.value.trim(),
      collection: searchCollection.value,
      top_k: searchTopK.value,
      min_score: 0.20
    })
    searchResults.value = data.results ?? []
  } catch (e) {
    alert('Search error: ' + (e.response?.data?.error || e.message))
  } finally {
    searching.value = false
  }
}

function formatBytes(bytes) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(1) + ' ' + sizes[i]
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric'
  })
}

onMounted(() => {
  fetchDocuments()
  fetchCollections()
  // Poll periodically if any document is indexing
  pollTimer = setInterval(() => {
    if (documents.value.some(d => d.status === 'indexing')) {
      fetchDocuments()
    }
  }, 3000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>
