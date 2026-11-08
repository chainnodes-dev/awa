<template>
  <div class="flex flex-col h-full overflow-y-auto bg-surface-0">

      <header class="px-6 py-4 border-b border-border shrink-0 bg-surface-1 flex items-center gap-6">
        <div class="flex flex-col">
          <h1 class="text-base font-bold text-text tracking-tight uppercase">Settings</h1>
          <p class="text-[10px] text-text-muted font-bold uppercase tracking-widest opacity-60">Platform Identity & AI Designer</p>
        </div>

        <div class="flex-1">
          <!-- Tab Switcher -->
          <div class="flex p-1 bg-surface-2 rounded-xl border border-border w-fit">
            <button @click="activeTab = 'general'" 
                    :class="['px-6 py-1.5 text-[11px] font-bold uppercase rounded-lg transition-all', 
                             activeTab === 'general' ? 'bg-accent text-white shadow-lg shadow-indigo-500/20' : 'text-text-muted hover:text-text']"
                    style="background-color: activeTab === 'general' ? 'var(--color-accent)' : 'transparent'">
              General
            </button>
            <button @click="activeTab = 'local_models'" 
                    :class="['px-6 py-1.5 text-[11px] font-bold uppercase rounded-lg transition-all', 
                             activeTab === 'local_models' ? 'bg-accent text-white shadow-lg shadow-indigo-500/20' : 'text-text-muted hover:text-text']"
                    style="background-color: activeTab === 'local_models' ? 'var(--color-accent)' : 'transparent'">
              Local Models
            </button>
            <button @click="activeTab = 'secrets'" 
                    :class="['px-6 py-1.5 text-[11px] font-bold uppercase rounded-lg transition-all', 
                             activeTab === 'secrets' ? 'bg-accent text-white shadow-lg shadow-indigo-500/20' : 'text-text-muted hover:text-text']"
                    style="background-color: activeTab === 'secrets' ? 'var(--color-accent)' : 'transparent'">
              Secrets
            </button>
            <button @click="activeTab = 'prompts'" 
                    :class="['px-6 py-1.5 text-[11px] font-bold uppercase rounded-lg transition-all', 
                             activeTab === 'prompts' ? 'bg-accent text-white shadow-lg shadow-indigo-500/20' : 'text-text-muted hover:text-text']"
                    style="background-color: activeTab === 'prompts' ? 'var(--color-accent)' : 'transparent'">
              System Prompts
            </button>
          </div>
        </div>

        <div v-if="entStore.status" class="flex items-center gap-2">
           <span v-if="entStore.status.company_name" class="text-[10px] font-bold tracking-wider text-text-muted">Licensed to <strong class="text-text">{{ entStore.status.company_name }}</strong></span>
           <span class="text-[10px] font-bold uppercase tracking-widest text-text-muted">Tier</span>
           <span :class="tierBadgeClass(entStore.status.tier)">{{ entStore.status.tier }}</span>
        </div>
      </header>

      <div v-if="activeTab === 'general'" class="flex-1 px-6 py-6 max-w-3xl w-full mx-auto space-y-6">
        <!-- Appearance (Theme Switcher) -->
        <section class="space-y-4">
           <div class="flex items-center gap-2 px-1">
             <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-indigo-400">
               <circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
             </svg>
             <h2 class="text-sm font-bold text-text uppercase tracking-wider">Platform Appearance</h2>
           </div>

           <div class="card p-5">
              <div class="flex items-center gap-4">
                 <button 
                   v-for="mode in ['light', 'dark', 'system']" 
                   :key="mode"
                   @click="setTheme(mode)"
                   :class="[
                     'flex-1 flex flex-col items-center gap-2 p-3 rounded-lg border transition-all',
                     theme === mode ? 'border-accent bg-accent/5 text-accent' : 'border-border hover:bg-text/5 text-text-muted'
                   ]"
                 >
                   <span class="text-[10px] font-bold uppercase tracking-widest">{{ mode }}</span>
                 </button>
              </div>
           </div>
        </section>

        <!-- Branding -->
        <section v-if="isAdmin" class="space-y-4">
           <div class="flex items-center gap-2 px-1">
             <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-indigo-400">
               <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
             </svg>
              <h2 class="text-sm font-bold text-text uppercase tracking-wider">Tenant Identity & Branding</h2>
              <span v-if="!entStore.hasFeature('branding')" class="badge bg-amber-500/10 text-amber-500 text-[9px] font-bold tracking-widest">Enterprise Feature</span>
            </div>

            <div class="card p-5 space-y-4" :class="{'opacity-40 grayscale pointer-events-none select-none': !entStore.hasFeature('branding')}">
               <div class="grid grid-cols-2 gap-4">
                  <div>
                     <label class="text-[10px] font-bold text-text-muted uppercase tracking-widest block mb-1">Company Name</label>
                     <input v-model="brandingForm.name" class="input text-sm w-full" placeholder="e.g. Acme Corp" />
                  </div>
                  <div>
                     <label class="text-[10px] font-bold text-text-muted uppercase tracking-widest block mb-1">Logo URL</label>
                     <input v-model="brandingForm.logo_url" class="input text-sm w-full" placeholder="https://..." autocomplete="off" />
                  </div>
               </div>
               <div class="flex justify-end">
                  <button @click="saveBranding" :disabled="brandingBusy" class="btn-primary text-xs px-4">
                     {{ brandingBusy ? 'Saving...' : 'Save Branding' }}
                  </button>
               </div>
            </div>
        </section>


        <!-- LLM Configs -->
        <section v-if="isAdmin" class="space-y-4">
           <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 px-1">
             <div class="flex items-center gap-2">
               <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-indigo-400">
                 <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
               </svg>
               <h2 class="text-sm font-bold text-text uppercase tracking-wider">LLM Providers & Routing</h2>
               <span class="text-[9px] bg-indigo-500/10 text-indigo-400 px-2 py-0.5 rounded-full font-medium">
                 {{ showAllProviders ? 'All Providers' : 'Standard (OpenRouter + Local)' }}
               </span>
             </div>
             <div class="flex items-center gap-2">
               <!-- Category Filter Pills (shown when direct providers are expanded) -->
               <div v-if="showAllProviders" class="flex items-center gap-1 bg-surface-2 p-1 rounded-lg border border-border text-[10px]">
                 <button @click="llmFilter = 'all'" :class="['px-2.5 py-1 rounded-md font-medium transition-all', llmFilter === 'all' ? 'bg-surface-0 text-text shadow-sm' : 'text-text-muted hover:text-text']">All ({{ llmStore.PROVIDERS.length }})</button>
                 <button @click="llmFilter = 'active'" :class="['px-2.5 py-1 rounded-md font-medium transition-all', llmFilter === 'active' ? 'bg-surface-0 text-text shadow-sm' : 'text-text-muted hover:text-text']">Active</button>
                 <button @click="llmFilter = 'gateway'" :class="['px-2.5 py-1 rounded-md font-medium transition-all flex items-center gap-1', llmFilter === 'gateway' ? 'bg-indigo-500/20 text-indigo-400 shadow-sm' : 'text-text-muted hover:text-text']">
                   <span class="w-1.5 h-1.5 rounded-full bg-indigo-400"></span> OpenRouter
                 </button>
                 <button @click="llmFilter = 'cloud'" :class="['px-2.5 py-1 rounded-md font-medium transition-all', llmFilter === 'cloud' ? 'bg-surface-0 text-text shadow-sm' : 'text-text-muted hover:text-text']">Direct APIs</button>
                 <button @click="llmFilter = 'local'" :class="['px-2.5 py-1 rounded-md font-medium transition-all', llmFilter === 'local' ? 'bg-surface-0 text-text shadow-sm' : 'text-text-muted hover:text-text']">Local</button>
               </div>

               <!-- Toggle Direct Providers Button -->
               <button
                 type="button"
                 @click="showAllProviders = !showAllProviders"
                 :class="['text-xs px-2.5 py-1 rounded-lg border font-medium transition-all flex items-center gap-1.5', showAllProviders ? 'bg-surface-2 border-accent text-accent' : 'bg-surface-1 border-border text-text-muted hover:text-text']"
                 :title="showAllProviders ? 'Hide direct cloud APIs' : 'Show direct cloud APIs (OpenAI, Anthropic, Gemini, Grok...)'"
               >
                 <span>{{ showAllProviders ? 'Hide Direct APIs' : 'Show Direct APIs' }}</span>
                 <span class="text-[10px] opacity-70">({{ showAllProviders ? llmStore.PROVIDERS.length : '2' }})</span>
               </button>
             </div>
           </div>

           <!-- OpenRouter Spotlight Banner -->
           <div v-if="llmFilter === 'all' || llmFilter === 'gateway'" class="bg-gradient-to-r from-indigo-500/10 via-purple-500/10 to-transparent border border-indigo-500/20 rounded-xl p-4 flex items-start justify-between gap-4">
             <div class="space-y-1">
               <div class="flex items-center gap-2">
                 <span class="text-xs font-bold text-indigo-400 uppercase tracking-wider">Universal Router (Recommended)</span>
                 <span class="text-[9px] bg-indigo-500/20 text-indigo-300 px-2 py-0.5 rounded-full font-semibold">One Key for 300+ Models</span>
               </div>
               <p class="text-[11px] text-text-muted max-w-xl">
                 Use <strong>OpenRouter</strong> to route requests across Claude 3.5 Sonnet, GPT-4o, DeepSeek R1, Llama 3.3, and Gemini without managing separate subscriptions.
               </p>
             </div>
             <a href="https://openrouter.ai/keys" target="_blank" rel="noopener noreferrer" class="btn-ghost text-[10px] text-indigo-400 hover:text-indigo-300 font-bold shrink-0">
               Get API Key ↗
             </a>
           </div>

           <!-- Semantic Model Profiles (Task-based Routing) -->
           <div class="space-y-3">
             <div class="flex items-center justify-between px-1">
               <div>
                 <h3 class="text-xs font-bold text-text uppercase tracking-wider flex items-center gap-2">
                   <span>Task Model Profiles</span>
                   <span class="text-[9px] bg-indigo-500/10 text-indigo-400 px-2 py-0.5 rounded-full font-medium">Automatic Workflow Routing</span>
                 </h3>
                 <p class="text-[11px] text-text-muted mt-0.5">Workflow state nodes select these profiles. Assign OpenRouter or Local Ollama models to each role.</p>
               </div>
               <button @click="llmStore.fetchProfiles()" class="btn-ghost text-xs p-1.5 text-text-muted hover:text-text" title="Refresh Profiles">
                 <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.2"/></svg>
               </button>
             </div>

             <!-- Profile Cards Grid -->
             <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
               <div
                 v-for="(prof, key) in displayProfiles"
                 :key="key"
                 class="card p-4 space-y-3 border-border hover:border-accent/30 transition-all flex flex-col justify-between"
               >
                 <div class="space-y-1.5">
                   <div class="flex items-center justify-between">
                     <span class="text-xs font-bold text-text flex items-center gap-1.5">
                       <span>{{ profileIcon(key) }}</span>
                       <span>{{ prof.label || key }}</span>
                     </span>
                     <span :class="['text-[9px] px-2 py-0.5 rounded-full font-mono font-bold uppercase tracking-wider', prof.provider === 'ollama' ? 'bg-emerald-500/10 text-emerald-400' : 'bg-indigo-500/10 text-indigo-400']">
                       {{ prof.provider === 'ollama' ? 'Local' : 'OpenRouter' }}
                     </span>
                   </div>
                   <p class="text-[10px] text-text-muted leading-relaxed line-clamp-2">{{ prof.description }}</p>
                 </div>

                 <div class="pt-2 border-t border-border/50 flex items-center justify-between gap-2">
                   <div class="truncate">
                     <span class="text-[9px] text-text-muted block font-mono uppercase tracking-widest">Assigned Model</span>
                     <span class="text-xs font-mono font-medium text-text truncate block" :title="prof.model">{{ prof.model || 'Not assigned' }}</span>
                   </div>
                   <button
                     @click="openAssignModal(key)"
                     class="btn-ghost text-xs px-2.5 py-1 text-accent hover:bg-accent/10 shrink-0 font-medium rounded-lg"
                   >
                     Change
                   </button>
                 </div>
               </div>
             </div>
           </div>

           <!-- Process Generation Profiles (AI Designer) -->
           <div class="space-y-3">
             <div class="flex items-center justify-between px-1">
               <div>
                 <h3 class="text-xs font-bold text-text uppercase tracking-wider flex items-center gap-2">
                   <span>Process Generation Profiles</span>
                   <span class="text-[9px] bg-purple-500/10 text-purple-400 px-2 py-0.5 rounded-full font-medium">AI Designer & Refine</span>
                 </h3>
                 <p class="text-[11px] text-text-muted mt-0.5">Tiered models used by the AI Designer to architect, decompose, wire, and synthesize workflows.</p>
               </div>
               <button @click="llmStore.fetchDesignerProfiles()" class="btn-ghost text-xs p-1.5 text-text-muted hover:text-text" title="Refresh Designer Profiles">
                 <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.2"/></svg>
               </button>
             </div>

             <!-- Designer Profile Cards Grid (Frontier, Balanced, Fast) -->
             <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
               <div
                 v-for="(prof, key) in displayDesignerProfiles"
                 :key="key"
                 class="card p-4 space-y-3 border-border hover:border-purple-500/30 transition-all flex flex-col justify-between"
               >
                 <div class="space-y-1.5">
                   <div class="flex items-center justify-between">
                     <span class="text-xs font-bold text-text flex items-center gap-1.5">
                       <span>{{ designerProfileIcon(key) }}</span>
                       <span>{{ prof.label || key }}</span>
                     </span>
                     <span v-if="key === 'balanced'" class="text-[9px] px-2 py-0.5 rounded-full bg-accent/20 text-accent font-semibold uppercase tracking-wider">Default</span>
                     <span v-else class="text-[9px] px-2 py-0.5 rounded-full bg-purple-500/10 text-purple-400 font-mono font-bold uppercase tracking-wider">
                       {{ prof.provider === 'ollama' ? 'Local' : 'OpenRouter' }}
                     </span>
                   </div>
                   <p class="text-[10px] text-text-muted leading-relaxed line-clamp-2">{{ prof.description }}</p>
                 </div>

                 <div class="pt-2 border-t border-border/50 flex items-center justify-between gap-2">
                   <div class="truncate">
                     <span class="text-[9px] text-text-muted block font-mono uppercase tracking-widest">Assigned Model</span>
                     <span class="text-xs font-mono font-medium text-text truncate block" :title="prof.model">{{ prof.model || 'Not assigned' }}</span>
                   </div>
                   <button
                     @click="openAssignModalForDesigner(key)"
                     class="btn-ghost text-xs px-2.5 py-1 text-accent hover:bg-accent/10 shrink-0 font-medium rounded-lg"
                   >
                     Change
                   </button>
                 </div>
               </div>
             </div>
           </div>

           <!-- Provider Cards -->
           <div v-for="prov in filteredProviders" :key="prov.id" :class="['card p-5 space-y-4 transition-all', prov.id === 'openrouter' ? 'border-indigo-500/40 shadow-sm shadow-indigo-500/5' : '']">
             <div class="flex items-center gap-3">
               <div class="flex-1">
                 <div class="flex items-center gap-2">
                   <h2 class="text-sm font-semibold text-text">{{ prov.label }}</h2>
                   <span v-if="prov.badge" class="text-[9px] px-2 py-0.5 rounded-full bg-indigo-500/20 text-indigo-300 font-semibold uppercase tracking-wider">{{ prov.badge }}</span>
                   <span v-if="isDefaultProvider(prov.id)" class="text-xs px-2 py-0.5 rounded-full bg-accent/20 text-accent font-medium">default</span>
                 </div>
                 <p class="text-xs text-text-muted mt-0.5">{{ prov.description || providerEndpoint(prov.id) || 'Local server' }}</p>
               </div>
               <label class="flex items-center gap-2 cursor-pointer">
                 <input type="checkbox" v-model="forms[prov.id].enabled" class="rounded" />
                 <span class="text-xs text-text-muted">Enabled</span>
               </label>
             </div>

             <div class="grid grid-cols-2 gap-3">
               <div v-if="prov.needsKey" class="col-span-2">
                 <label class="text-xs text-text-muted mb-1 block">API Key</label>
                 <input v-model="forms[prov.id].api_key" type="password" :placeholder="hasExistingKey(prov.id) ? 'Configured (leave blank to keep)' : 'Enter API key'" class="input text-sm w-full font-mono" autocomplete="new-password" />
               </div>
               <div class="col-span-2">
                 <div class="flex items-center justify-between mb-1">
                   <label class="text-xs text-text-muted block">Endpoint URL</label>
                   <span v-if="!prov.needsURL && !forms[prov.id].base_url" class="text-[10px] text-text-muted opacity-60">Default: {{ providerEndpoint(prov.id) }}</span>
                 </div>
                 <input v-model="forms[prov.id].base_url" type="text" :placeholder="prov.needsURL ? 'http://localhost:11434' : ('https://' + providerEndpoint(prov.id) + ' (default)')" class="input text-sm w-full font-mono" />
               </div>
               <div v-if="prov.id === 'openrouter' || prov.id === 'ollama'">
                 <label class="text-xs text-text-muted mb-1 block">Default Model</label>
                 <div class="flex items-center justify-between gap-2 px-3 py-1.5 rounded-lg bg-surface-1 border border-border min-h-[38px]">
                   <div class="min-w-0 flex-1">
                     <span class="text-xs font-mono font-medium text-text truncate block" :title="forms[prov.id].default_model || llmStore.DEFAULT_MODELS[prov.id]">
                       {{ forms[prov.id].default_model || llmStore.DEFAULT_MODELS[prov.id] || 'Not selected' }}
                     </span>
                     <span v-if="!forms[prov.id].default_model" class="text-[9px] text-text-muted">
                       (Preset default)
                     </span>
                   </div>
                   <button
                     type="button"
                     @click="openModelPickerForProvider(prov.id)"
                     class="btn-ghost text-xs px-2.5 py-1 text-accent hover:bg-accent/10 shrink-0 font-medium rounded-lg border border-accent/20"
                   >
                     Change
                   </button>
                 </div>
               </div>
               <div v-else>
                 <label class="text-xs text-text-muted mb-1 block">Default Model</label>
                 <input v-model="forms[prov.id].default_model" type="text" :placeholder="llmStore.DEFAULT_MODELS[prov.id]" class="input text-sm w-full font-mono text-xs" />
               </div>
               <div>
                 <label class="text-xs text-text-muted mb-1 block">Max Output Tokens</label>
                 <input v-model.number="forms[prov.id].max_output_tokens" type="number" placeholder="4096" class="input text-sm w-full" />
               </div>
             </div>

             <div v-if="statuses[prov.id]" :class="['text-xs px-3 py-2 rounded', statuses[prov.id].ok ? 'bg-green-500/10 text-green-400' : 'bg-red-500/10 text-red-400']">
               {{ statuses[prov.id].ok ? '✓ Connected' : '✗ ' + statuses[prov.id].error }}
             </div>

             <div class="flex gap-2 pt-1">
               <button @click="save(prov.id)" :disabled="saving[prov.id]" class="btn-primary text-xs px-3 py-1.5">{{ saving[prov.id] ? 'Saving…' : 'Save' }}</button>
               <button @click="test(prov.id)" :disabled="testing[prov.id]" class="btn-ghost text-xs px-3 py-1.5">{{ testing[prov.id] ? 'Testing…' : 'Test Connection' }}</button>
               <button v-if="forms[prov.id].enabled && !isDefaultProvider(prov.id)" @click="saveDefault(prov.id)" class="btn-ghost text-violet-400 text-xs px-3 py-1.5 font-bold uppercase">Set as Default</button>
             </div>
           </div>
        </section>

        <!-- Infrastructure & Telemetry -->
        <section v-if="isAdmin" class="space-y-4">
           <div class="flex items-center gap-2 px-1">
             <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-purple-400">
               <rect x="4" y="4" width="16" height="16" rx="2"/><path d="M9 9h6v6H9z"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="M2 12h2"/><path d="M20 12h2"/>
             </svg>
             <h2 class="text-sm font-bold text-text uppercase tracking-wider">Infrastructure</h2>
           </div>

           <div class="card p-5 flex items-start justify-between">
              <div class="space-y-1">
                <h3 class="text-xs font-bold text-text">Anonymized Telemetry</h3>
                <p class="text-[11px] text-text-muted max-w-sm">
                  Share platform-wide usage metrics with Chain Nodes HQ.
                </p>
              </div>
              <button @click="toggleTelemetry" :class="['w-10 h-5 rounded-full relative transition-colors duration-200 outline-none', telemetryEnabled ? 'bg-emerald-500' : 'bg-surface-3']">
                <span :class="['absolute top-1 left-1 bg-white w-3 h-3 rounded-full transition-transform', telemetryEnabled ? 'translate-x-5' : 'translate-x-0']"></span>
              </button>
           </div>
         </section>
      </div>

      <!-- Secrets Management (Enterprise Feature) -->
      <div v-if="activeTab === 'secrets'" class="flex-1 px-6 py-6 max-w-3xl w-full mx-auto space-y-6">
        <section v-if="isAdmin" class="space-y-4">
           <div class="flex items-center gap-2 px-1">
             <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-indigo-400">
               <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
             </svg>
              <h2 class="text-sm font-bold text-text uppercase tracking-wider">Environment Secrets</h2>
              <span v-if="!entStore.hasFeature('secrets')" class="badge bg-amber-500/10 text-amber-500 text-[9px] font-bold tracking-widest">Enterprise Feature</span>
            </div>

            <div class="card p-5 space-y-4" :class="{'opacity-40 grayscale pointer-events-none select-none': !entStore.hasFeature('secrets')}">
              <div v-for="(val, idx) in secretsForm" :key="idx" class="flex gap-2 items-center">
                 <input v-model="secretsForm[idx].name" class="input text-xs w-1/3" placeholder="KEY_NAME" />
                 <input v-model="secretsForm[idx].value" type="password" class="input text-xs flex-1" placeholder="Value (hidden)" />
                 <button @click="removeSecret(idx)" class="btn-ghost text-red-400 p-2">✕</button>
              </div>
              <div class="flex justify-between items-center pt-2">
                 <button @click="addSecret" class="btn-ghost text-[10px] font-bold uppercase">+ Add Secret</button>
                 <button @click="saveSecrets" :disabled="secretsBusy" class="btn-primary text-xs px-4">
                    {{ secretsBusy ? 'Saving...' : 'Save Secrets' }}
                 </button>
              </div>
            </div>

            <div v-if="!entStore.hasFeature('secrets')" class="text-center p-6 bg-surface-2 rounded-xl border border-border">
              <p class="text-xs text-text-muted">
                Secrets Management is an Enterprise feature. Please upload an enterprise license token or upgrade your plan to unlock encrypted vault storage for keys and API credentials.
              </p>
            </div>
        </section>
      </div>

      <!-- Local Models Management Tab -->
      <div v-if="activeTab === 'local_models'" class="flex-1 px-6 py-6 max-w-3xl w-full mx-auto space-y-6">
        <section class="space-y-4">
          <div class="flex items-center gap-2 px-1">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-indigo-400">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4a2 2 0 0 0 1-1.73z"/>
              <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
              <line x1="12" y1="22.08" x2="12" y2="12"/>
            </svg>
            <h2 class="text-sm font-bold text-text uppercase tracking-wider">Local Model Weight Manager</h2>
          </div>

          <!-- Ollama Server Status & Models List -->
          <div class="card p-5 space-y-4">
            <div class="flex items-center justify-between border-b border-border/50 pb-3">
              <div>
                <h3 class="text-xs font-bold text-text uppercase tracking-widest">Ollama Status</h3>
                <p class="text-[11px] text-text-muted mt-0.5">Manage local weights served via http://localhost:11434</p>
              </div>
              <span :class="['px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-widest', modelsError ? 'bg-red-500/10 text-red-400' : 'bg-green-500/10 text-green-400']">
                {{ modelsError ? 'Disconnected' : 'Connected' }}
              </span>
            </div>

            <!-- Error state -->
            <div v-if="modelsError" class="bg-red-500/10 text-red-400 text-xs p-4 rounded border border-red-500/20">
              <p class="font-bold">Cannot connect to Ollama server:</p>
              <p class="mt-1 font-mono text-[10px]">{{ modelsError }}</p>
              <p class="mt-2 text-text-muted text-[10px]">
                Please ensure Ollama is running on your machine (`ollama serve`) and is accessible.
              </p>
            </div>

            <!-- Models list -->
            <div v-else class="space-y-3">
              <h4 class="text-[10px] font-bold text-text-muted uppercase tracking-widest">Downloaded Models</h4>
              
              <div v-if="localModels.length === 0" class="text-center py-6 text-xs text-text-muted">
                No local models downloaded yet.
              </div>
              
              <div v-else class="divide-y divide-border/50">
                <div v-for="model in localModels" :key="model.name" class="flex items-center justify-between py-3 first:pt-0 last:pb-0">
                  <div class="flex flex-col gap-0.5">
                    <span class="text-xs font-bold text-text font-mono">{{ model.name }}</span>
                    <span class="text-[10px] text-text-muted font-medium">
                      Size: {{ formatSize(model.size) }} | Format: {{ model.details?.format || 'GGUF' }}
                    </span>
                  </div>
                  <button @click="deleteModel(model.name)" class="btn-ghost text-red-400 text-xs px-2.5 py-1 font-bold uppercase hover:bg-red-500/10 rounded">
                    Delete
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- Download Model Form -->
          <div v-if="!modelsError" class="card p-5 space-y-4">
            <h3 class="text-xs font-bold text-text uppercase tracking-widest border-b border-border/50 pb-2">Download Local Weights</h3>
            
            <div class="flex gap-2">
              <div class="flex-1 relative">
                <input 
                  v-model="downloadModelName" 
                  type="text" 
                  placeholder="e.g. gemma2:2b, gemma2:9b, llama3.2" 
                  class="input text-sm w-full font-mono"
                  :disabled="downloadProgress.active"
                  @keyup.enter="downloadModel"
                />
                <!-- Suggestions -->
                <div class="flex flex-wrap gap-1.5 mt-2">
                  <span 
                    v-for="suggest in ['gemma2:2b', 'gemma2:9b', 'llama3.2', 'qwen2.5-coder:7b']"
                    :key="suggest"
                    @click="!downloadProgress.active && (downloadModelName = suggest)"
                    :class="['text-[9px] font-mono font-bold uppercase tracking-wide px-2 py-0.5 rounded cursor-pointer transition-all border',
                             downloadModelName === suggest ? 'bg-accent/20 border-accent text-accent' : 'bg-surface-2 border-border text-text-muted hover:text-text']"
                  >
                    {{ suggest }}
                  </span>
                </div>
              </div>
              <button 
                @click="downloadModel" 
                :disabled="downloadProgress.active || !downloadModelName" 
                class="btn-primary text-xs px-5 h-9 shrink-0 flex items-center justify-center font-bold uppercase tracking-wider"
              >
                {{ downloadProgress.active ? 'Downloading...' : 'Pull Weights' }}
              </button>
            </div>

            <!-- Download Progress Display -->
            <div v-if="downloadProgress.active" class="space-y-2 bg-surface-2 border border-border/80 rounded-xl p-4">
              <div class="flex items-center justify-between text-[11px] font-bold text-text uppercase tracking-wider">
                <span>{{ downloadProgress.status }}</span>
                <span>{{ downloadProgress.percent }}%</span>
              </div>
              
              <!-- Progress Bar -->
              <div class="w-full bg-surface-3 h-2 rounded-full overflow-hidden">
                <div 
                  class="bg-accent h-full transition-all duration-150" 
                  :style="{ width: downloadProgress.percent + '%' }"
                ></div>
              </div>
              
              <div class="flex justify-between text-[10px] text-text-muted font-mono">
                <span>{{ downloadProgress.completed }} / {{ downloadProgress.total }}</span>
              </div>
            </div>
          </div>
        </section>
      </div>

      <div v-if="activeTab === 'prompts'" class="flex-1 px-6 py-6 max-w-5xl w-full mx-auto space-y-6">
        <section class="space-y-6">
           <div class="flex items-center justify-between px-1">
             <div class="flex items-center gap-2">
               <svg class="text-violet-400" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                 <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
               </svg>
               <h2 class="text-sm font-bold text-text uppercase tracking-wider">AI Designer System Prompts</h2>
             </div>
             <p class="text-[10px] text-text-muted uppercase tracking-widest font-bold">Overrides reset on server restart</p>
           </div>

          <div v-for="(content, id) in systemPrompts" :key="id" class="space-y-2 group">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <span class="text-[10px] font-mono text-violet-400/80 uppercase tracking-widest font-bold">{{ id.replace(/workflow_/g, '').replace(/_/g, ' ') }}</span>
                <button 
                  @click="openPopout(id)"
                  class="p-1 rounded hover:bg-surface-2 text-text-muted hover:text-violet-400 transition-all opacity-0 group-hover:opacity-100"
                  title="Full-screen Editor"
                >
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                    <path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/>
                  </svg>
                </button>
              </div>
              <button @click="savePrompt(id)" :disabled="savingPrompt[id]" class="text-[10px] bg-violet-500/10 text-violet-400 hover:bg-violet-500/20 px-2 py-1 rounded font-bold uppercase transition-all">
                {{ savingPrompt[id] ? 'Saving...' : 'Save Override' }}
              </button>
            </div>
            
            <div class="h-80 shadow-inner rounded-lg border border-border/50 overflow-hidden">
              <CodeEditor
                v-model="systemPrompts[id]"
                language="text"
                height="100%"
                :show-badge="false"
              />
            </div>
          </div>
        </section>
      </div>

    <CodeEditorModal
      v-if="activePopoutId"
      v-model="systemPrompts[activePopoutId]"
      :title="activePopoutId.replace(/_/g, ' ').toUpperCase()"
      language="text"
      @close="activePopoutId = null"
    />

    <!-- Assign Model to Profile Modal -->
    <div v-if="assignModalOpen" class="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4">
      <div class="card max-w-2xl w-full max-h-[85vh] flex flex-col overflow-hidden border-border shadow-2xl">
        <!-- Modal Header -->
        <div class="p-4 border-b border-border flex items-center justify-between bg-surface-1 shrink-0">
          <div class="flex items-center gap-2">
            <span class="text-base">{{ getModalIcon() }}</span>
            <div>
              <h3 class="text-sm font-bold text-text">{{ modalTitle }}</h3>
              <p class="text-[11px] text-text-muted">{{ modalSubtitle }}</p>
            </div>
          </div>
          <button @click="assignModalOpen = false" class="btn-ghost p-1 text-text-muted hover:text-text">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
          </button>
        </div>

        <!-- Modal Source Tabs -->
        <div class="p-3 border-b border-border bg-surface-0 flex items-center justify-between gap-3 shrink-0">
          <div v-if="assignTargetMode === 'profile'" class="flex items-center gap-1 bg-surface-2 p-1 rounded-lg border border-border text-xs">
            <button
              @click="assignSourceTab = 'openrouter'"
              :class="['px-3 py-1 rounded-md font-medium transition-all flex items-center gap-1.5', assignSourceTab === 'openrouter' ? 'bg-indigo-500/20 text-indigo-400 font-bold shadow-sm' : 'text-text-muted hover:text-text']"
            >
              <span>🌐 OpenRouter Catalog</span>
              <span v-if="llmStore.openRouterCatalog.length" class="text-[10px] opacity-75">({{ llmStore.openRouterCatalog.length }})</span>
            </button>
            <button
              @click="assignSourceTab = 'ollama'"
              :class="['px-3 py-1 rounded-md font-medium transition-all flex items-center gap-1.5', assignSourceTab === 'ollama' ? 'bg-emerald-500/20 text-emerald-400 font-bold shadow-sm' : 'text-text-muted hover:text-text']"
            >
              <span>💻 Local Weights (Ollama)</span>
              <span v-if="localModels.length" class="text-[10px] opacity-75">({{ localModels.length }})</span>
            </button>
          </div>
          <div v-else class="flex items-center gap-2 text-xs font-semibold px-2.5 py-1 bg-surface-2 rounded-lg border border-border">
            <span v-if="assignSourceTab === 'openrouter'" class="text-indigo-400 flex items-center gap-1.5">
              <span>🌐 OpenRouter Catalog</span>
              <span v-if="llmStore.openRouterCatalog.length" class="text-[10px] opacity-75">({{ llmStore.openRouterCatalog.length }} models)</span>
            </span>
            <span v-else class="text-emerald-400 flex items-center gap-1.5">
              <span>💻 Local Weights (Ollama)</span>
              <span v-if="localModels.length" class="text-[10px] opacity-75">({{ localModels.length }} downloaded)</span>
            </span>
          </div>

          <!-- Live Search Bar -->
          <div v-if="assignSourceTab === 'openrouter'" class="relative flex-1 max-w-xs">
            <input
              v-model="modelSearchQuery"
              type="text"
              placeholder="Search 300+ models (e.g. gemini, r1, sonnet)..."
              class="input text-xs w-full pl-8"
              autofocus
            />
            <svg class="absolute left-2.5 top-2.5 text-text-muted" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
          </div>
        </div>

        <!-- Modal Body (Scrollable List) -->
        <div class="flex-1 overflow-y-auto p-4 space-y-2">
          <!-- OpenRouter Models List -->
          <template v-if="assignSourceTab === 'openrouter'">
            <div v-if="llmStore.catalogLoading" class="text-center py-12 text-text-muted text-xs">
              <div class="inline-block animate-spin mb-2">↻</div>
              <div>Fetching live models from OpenRouter...</div>
            </div>

            <div v-else-if="filteredOpenRouterModels.length === 0" class="text-center py-12 text-text-muted text-xs">
              No models match "{{ modelSearchQuery }}".
            </div>

            <div
              v-for="m in filteredOpenRouterModels"
              :key="m.id"
              :class="['p-3 rounded-xl border transition-all flex items-center justify-between gap-3', currentSelectedModel === m.id ? 'border-accent bg-accent/5' : 'border-border bg-surface-1 hover:border-border-hover']"
            >
              <div class="space-y-1 min-w-0 flex-1">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="text-xs font-semibold text-text truncate">{{ m.name }}</span>
                  <span class="text-[9px] font-mono text-text-muted bg-surface-2 px-1.5 py-0.5 rounded">{{ m.id }}</span>
                  <span v-if="currentSelectedModel === m.id" class="text-[9px] font-bold text-accent bg-accent/10 px-2 py-0.5 rounded-full">CURRENT</span>
                </div>
                <p v-if="m.description" class="text-[10px] text-text-muted line-clamp-1">{{ m.description }}</p>
                <div class="flex items-center gap-3 text-[10px] font-mono text-text-muted">
                  <span v-if="m.context_length">Context: {{ formatContext(m.context_length) }}</span>
                  <span v-if="m.prompt_price !== undefined">Prompt: ${{ m.prompt_price.toFixed(2) }}/M</span>
                  <span v-if="m.completion_price !== undefined">Comp: ${{ m.completion_price.toFixed(2) }}/M</span>
                </div>
              </div>

              <button
                @click="assignModel('openrouter', m.id)"
                :disabled="savingProfile"
                :class="['text-xs px-3 py-1.5 rounded-lg font-bold transition-all shrink-0', currentSelectedModel === m.id ? 'btn-ghost text-text-muted' : 'btn-primary']"
              >
                {{ currentSelectedModel === m.id ? 'Selected' : 'Assign' }}
              </button>
            </div>
          </template>

          <!-- Ollama Local Models List -->
          <template v-else-if="assignSourceTab === 'ollama'">
            <div v-if="localModels.length === 0" class="text-center py-12 text-text-muted text-xs space-y-2">
              <p>No local Ollama models currently downloaded.</p>
              <p class="text-[10px]">Use the "Local Model Weight Manager" section below to pull weights (e.g. `llama3.2`).</p>
            </div>

            <div
              v-for="m in localModels"
              :key="m.name"
              :class="['p-3 rounded-xl border transition-all flex items-center justify-between gap-3', currentSelectedModel === m.name ? 'border-accent bg-accent/5' : 'border-border bg-surface-1 hover:border-border-hover']"
            >
              <div class="space-y-1 min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="text-xs font-semibold text-text">{{ m.name }}</span>
                  <span class="text-[9px] font-mono text-emerald-400 bg-emerald-500/10 px-1.5 py-0.5 rounded">Local</span>
                  <span v-if="currentSelectedModel === m.name" class="text-[9px] font-bold text-accent bg-accent/10 px-2 py-0.5 rounded-full">CURRENT</span>
                </div>
                <div class="text-[10px] font-mono text-text-muted">
                  Size: {{ formatSize(m.size) }} | Format: {{ m.details?.format || 'GGUF' }}
                </div>
              </div>

              <button
                @click="assignModel('ollama', m.name)"
                :disabled="savingProfile"
                :class="['text-xs px-3 py-1.5 rounded-lg font-bold transition-all shrink-0', currentSelectedModel === m.name ? 'btn-ghost text-text-muted' : 'btn-primary']"
              >
                {{ currentSelectedModel === m.name ? 'Selected' : 'Assign' }}
              </button>
            </div>
          </template>
        </div>

        <!-- Custom Model Input Footer -->
        <div class="p-3 border-t border-border bg-surface-1 flex items-center justify-between gap-2 shrink-0">
          <div class="flex-1 flex items-center gap-2">
            <span class="text-[10px] font-mono text-text-muted whitespace-nowrap">Or custom slug:</span>
            <input
              v-model="customSlugInput"
              type="text"
              placeholder="e.g. meta-llama/llama-3.3-70b-instruct"
              class="input text-xs font-mono flex-1"
            />
          </div>
          <button
            @click="assignModel(assignSourceTab, customSlugInput)"
            :disabled="!customSlugInput || savingProfile"
            class="btn-primary text-xs px-3 py-1.5 shrink-0"
          >
            {{ assignTargetMode === 'provider' ? 'Select Custom' : 'Assign Custom' }}
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import { useAuthStore, api } from '@/stores/auth'
import { useLLMStore } from '@/stores/llm'
import { useEnterpriseStore } from '@/stores/enterprise'
import { useTheme } from '@/composables/useTheme'
import CodeEditor from '@/components/designer/CodeEditor.vue'
import CodeEditorModal from '@/components/designer/CodeEditorModal.vue'
const activeTab = ref('general')

const authStore = useAuthStore()
const llmStore = useLLMStore()
const entStore = useEnterpriseStore()
const { theme, setTheme } = useTheme()

const activePopoutId = ref(null)

function openPopout(id) {
  activePopoutId.value = id
}

const isAdmin = computed(() => authStore.user?.role === 'admin' || authStore.user?.role === 'super_admin')

// Build a reactive form object per provider
const forms = reactive({})
llmStore.PROVIDERS.forEach(p => {
  forms[p.id] = { api_key: '', base_url: '', default_model: '', max_output_tokens: 4096, enabled: false }
})

const saving  = reactive({})
const testing = reactive({})
const statuses = reactive({})

const selectedDefault = ref('')
const llmFilter = ref('all')

const enabledProviders = computed(() =>
  llmStore.PROVIDERS.filter(p => forms[p.id]?.enabled)
)

const showAllProviders = ref(false)

const filteredProviders = computed(() => {
  return llmStore.PROVIDERS.filter(p => {
    if (!showAllProviders.value) {
      if (p.id !== 'openrouter' && p.id !== 'ollama') {
        return false
      }
    }
    if (llmFilter.value === 'active') {
      return forms[p.id]?.enabled || hasExistingKey(p.id) || (p.id === 'ollama' && forms[p.id]?.base_url)
    }
    if (llmFilter.value === 'gateway') {
      return p.category === 'gateway'
    }
    if (llmFilter.value === 'local') {
      return p.category === 'local'
    }
    if (llmFilter.value === 'cloud') {
      return p.category === 'cloud'
    }
    return true
  })
})

function isDefaultProvider(id) {
  return llmStore.configs.find(c => c.provider === id)?.is_default === true
}

function hasExistingKey(id) {
  return llmStore.configs.find(c => c.provider === id)?.api_key === '***'
}

function providerEndpoint(id) {
  const map = {
    openrouter: 'openrouter.ai',
    openai:    'api.openai.com',
    grok:      'api.x.ai',
    deepseek:  'api.deepseek.com',
    gemini:    'generativelanguage.googleapis.com',
    anthropic: 'api.anthropic.com',
    mistral:   'api.mistral.ai',
    groq:      'api.groq.com',
    together:  'api.together.xyz',
    fireworks: 'api.fireworks.ai',
    cohere:    'api.cohere.ai',
    qwen:      'dashscope.aliyuncs.com',
    glm:       'open.bigmodel.cn',
    ollama:    'localhost:11434 (default)',
  }
  return map[id] ?? ''
}

// ── Model Profiles & Provider Model Selection ──
const assignModalOpen     = ref(false)
const assignTargetMode    = ref('profile') // 'profile' | 'provider'
const assignTargetKey     = ref('')        // profile key or provider id
const assigningProfileKey = ref('')
const modelSearchQuery    = ref('')
const assignSourceTab     = ref('openrouter') // 'openrouter' | 'ollama'
const customSlugInput     = ref('')
const savingProfile       = ref(false)

const displayProfiles = computed(() => {
  if (Object.keys(llmStore.profiles).length > 0) {
    return llmStore.profiles
  }
  return {
    fast: { id: 'fast', provider: 'openrouter', model: '~deepseek/deepseek-flash-latest', label: 'Fast & Efficient', description: 'High-throughput classification, routing & structured extraction' },
    reasoning: { id: 'reasoning', provider: 'openrouter', model: 'deepseek/deepseek-r1', label: 'Deep Reasoning', description: 'Complex planning, multi-step deduction & architectural decisions' },
    coding: { id: 'coding', provider: 'openrouter', model: 'qwen/qwen-2.5-coder-32b-instruct', label: 'Code & Transform', description: 'Code generation, SQL querying & complex data transforms' },
    general: { id: 'general', provider: 'openrouter', model: '~deepseek/deepseek-flash-latest', label: 'Default Generalist', description: 'High-capability reasoning for general tool calling & orchestration' },
    local: { id: 'local', provider: 'ollama', model: 'llama3.2', label: 'On-Prem / Local', description: 'Zero data-leakage execution on local hardware via Ollama' },
  }
})

const displayDesignerProfiles = computed(() => {
  if (Object.keys(llmStore.designerProfiles).length > 0) {
    return llmStore.designerProfiles
  }
  return {
    frontier: { id: 'frontier', provider: 'openrouter', model: 'anthropic/claude-sonnet-4.5', label: 'Frontier', description: 'Maximum intelligence & reasoning for complex enterprise workflows', badge: 'Frontier' },
    balanced: { id: 'balanced', provider: 'openrouter', model: 'google/gemini-2.0-flash-001', label: 'Balanced', description: 'Optimal speed, quality & cost balance for standard workflows', badge: 'Default' },
    fast: { id: 'fast', provider: 'openrouter', model: '~deepseek/deepseek-flash-latest', label: 'Fast', description: 'Rapid prototyping & simple workflows with ultra-low latency', badge: 'Fast' },
  }
})

function designerProfileIcon(key) {
  const icons = {
    frontier: '🚀',
    balanced: '⚖️',
    fast: '⚡',
  }
  return icons[key] || '⚙️'
}

const currentAssignProfile = computed(() => {
  if (assignTargetMode.value === 'provider') return {}
  if (assignTargetMode.value === 'designer_profile') {
    return llmStore.designerProfiles[assigningProfileKey.value] || displayDesignerProfiles.value[assigningProfileKey.value] || {}
  }
  return llmStore.profiles[assigningProfileKey.value] || displayProfiles.value[assigningProfileKey.value] || {}
})

const currentSelectedModel = computed(() => {
  if (assignTargetMode.value === 'provider') {
    const provId = assignTargetKey.value
    return forms[provId]?.default_model || llmStore.DEFAULT_MODELS[provId] || ''
  }
  return currentAssignProfile.value?.model || ''
})

const modalTitle = computed(() => {
  if (assignTargetMode.value === 'provider') {
    const prov = llmStore.PROVIDERS.find(p => p.id === assignTargetKey.value)
    return `Select Default Model for ${prov?.label || assignTargetKey.value}`
  }
  if (assignTargetMode.value === 'designer_profile') {
    return `Assign Model to ${currentAssignProfile.value.label || assigningProfileKey.value} Tier`
  }
  return `Assign Model to ${currentAssignProfile.value.label || assigningProfileKey.value}`
})

const modalSubtitle = computed(() => {
  if (assignTargetMode.value === 'provider') {
    if (assignTargetKey.value === 'openrouter') {
      return 'Choose from 400+ OpenRouter cloud models or enter a custom model slug'
    }
    if (assignTargetKey.value === 'ollama') {
      return 'Choose from downloaded local Ollama models or enter a custom model slug'
    }
    return 'Select default model for this provider'
  }
  if (assignTargetMode.value === 'designer_profile') {
    return 'Select the model to power AI Designer workflow synthesis for this tier'
  }
  return 'Select an OpenRouter cloud model or on-premise Ollama model'
})

function getModalIcon() {
  if (assignTargetMode.value === 'provider') {
    return assignTargetKey.value === 'ollama' ? '💻' : '🌐'
  }
  if (assignTargetMode.value === 'designer_profile') {
    return designerProfileIcon(assigningProfileKey.value)
  }
  return profileIcon(assigningProfileKey.value)
}

const filteredOpenRouterModels = computed(() => {
  const q = modelSearchQuery.value.trim().toLowerCase()
  const list = llmStore.openRouterCatalog
  if (!q) return list.slice(0, 50)
  return list.filter(m => 
    m.id.toLowerCase().includes(q) || 
    (m.name && m.name.toLowerCase().includes(q)) ||
    (m.description && m.description.toLowerCase().includes(q))
  ).slice(0, 50)
})

function profileIcon(key) {
  const icons = {
    fast: '⚡',
    reasoning: '🧠',
    coding: '💻',
    general: '🛡️',
    local: '🔒',
  }
  return icons[key] || '⚙️'
}

function formatContext(tokens) {
  if (!tokens) return 'unknown'
  if (tokens >= 1000000) return (tokens / 1000000).toFixed(1).replace('.0', '') + 'M'
  if (tokens >= 1000) return Math.round(tokens / 1000) + 'k'
  return tokens
}

function openAssignModal(key) {
  assignTargetMode.value = 'profile'
  assignTargetKey.value = key
  assigningProfileKey.value = key
  modelSearchQuery.value = ''
  customSlugInput.value = ''
  const prof = currentAssignProfile.value
  assignSourceTab.value = prof?.provider === 'ollama' ? 'ollama' : 'openrouter'
  assignModalOpen.value = true
  llmStore.fetchOpenRouterModels()
}

function openAssignModalForDesigner(key) {
  assignTargetMode.value = 'designer_profile'
  assignTargetKey.value = key
  assigningProfileKey.value = key
  modelSearchQuery.value = ''
  customSlugInput.value = ''
  const prof = currentAssignProfile.value
  assignSourceTab.value = prof?.provider === 'ollama' ? 'ollama' : 'openrouter'
  assignModalOpen.value = true
  llmStore.fetchOpenRouterModels()
}

function openModelPickerForProvider(providerId) {
  assignTargetMode.value = 'provider'
  assignTargetKey.value = providerId
  assigningProfileKey.value = ''
  modelSearchQuery.value = ''
  customSlugInput.value = ''
  assignSourceTab.value = (providerId === 'ollama') ? 'ollama' : 'openrouter'
  assignModalOpen.value = true
  if (providerId === 'openrouter') {
    llmStore.fetchOpenRouterModels()
  } else if (providerId === 'ollama') {
    fetchLocalModels()
  }
}

async function assignModel(provider, modelSlug) {
  if (!modelSlug) return
  const trimmed = modelSlug.trim()

  if (assignTargetMode.value === 'provider') {
    const provId = assignTargetKey.value
    if (forms[provId]) {
      forms[provId].default_model = trimmed
      if (forms[provId].enabled || hasExistingKey(provId)) {
        await save(provId)
      }
    }
    assignModalOpen.value = false
    return
  }

  if (assignTargetMode.value === 'designer_profile') {
    savingProfile.value = true
    try {
      const key = assigningProfileKey.value
      const target = { ...displayDesignerProfiles.value }
      if (!target[key]) target[key] = { id: key }
      target[key].provider = provider
      target[key].model = trimmed
      await llmStore.saveDesignerProfiles(target)
      assignModalOpen.value = false
    } catch (e) {
      console.error('Failed to assign designer profile', e)
    } finally {
      savingProfile.value = false
    }
    return
  }

  // Profile assignment
  savingProfile.value = true
  try {
    const key = assigningProfileKey.value
    const target = { ...displayProfiles.value }
    if (!target[key]) target[key] = { id: key }
    target[key].provider = provider
    target[key].model = trimmed
    await llmStore.saveProfiles(target)
    assignModalOpen.value = false
  } catch (e) {
    console.error('Failed to assign model profile', e)
  } finally {
    savingProfile.value = false
  }
}

const updatingLicense = ref(false)
const licenseInput = ref('')
const licenseBusy = ref(false)

const brandingForm = reactive({ name: '', logo_url: '' })
const brandingBusy = ref(false)

async function saveBranding() {
  brandingBusy.value = true
  try {
    await entStore.updateBranding(brandingForm)
  } catch (e) {
    alert('Failed to update branding: ' + (e.response?.data?.error ?? e.message))
  } finally {
    brandingBusy.value = false
  }
}

const secretsForm = ref([])
const secretsBusy = ref(false)

function addSecret() {
  secretsForm.value.push({ name: '', value: '' })
}
function removeSecret(idx) {
  secretsForm.value.splice(idx, 1)
}

async function saveSecrets() {
  secretsBusy.value = true
  try {
    const payload = {}
    secretsForm.value.forEach(s => {
      if (s.name) payload[s.name] = s.value
    })
    await entStore.updateSecrets(payload)
    alert('Secrets saved successfully.')
  } catch (e) {
    alert('Failed to save secrets: ' + (e.response?.data?.error ?? e.message))
  } finally {
    secretsBusy.value = false
  }
}

async function saveLicense() {
  if (!licenseInput.value) return
  licenseBusy.value = true
  try {
    await entStore.setLicense(licenseInput.value)
    updatingLicense.value = false
    licenseInput.value = ''
  } catch (e) {
    alert('License update failed: ' + (e.response?.data?.error ?? e.message))
  } finally {
    licenseBusy.value = false
  }
}

function usagePercent(curr, max) {
  if (!max) return 0
  return Math.min(100, Math.round((curr / max) * 100))
}

function tierBadgeClass(tier) {
  const base = 'badge'
  if (tier === 'enterprise') return `${base} bg-purple-500/20 text-purple-400`
  if (tier === 'pro') return `${base} bg-indigo-500/20 text-indigo-400`
  return `${base} bg-slate-500/20 text-text-muted`
}

function syncFormsFromStore() {
  llmStore.PROVIDERS.forEach(p => {
    const cfg = llmStore.configs.find(c => c.provider === p.id)
    if (cfg) {
      forms[p.id].enabled            = cfg.enabled
      forms[p.id].default_model      = cfg.default_model
      forms[p.id].max_output_tokens  = cfg.max_output_tokens || 4096
      forms[p.id].base_url           = cfg.base_url ?? ''
      forms[p.id].api_key            = ''  // never pre-fill masked keys
    } else {
      forms[p.id].enabled       = false
      forms[p.id].default_model = ''
      forms[p.id].base_url      = ''
      forms[p.id].api_key       = ''
    }
  })
  const def = llmStore.configs.find(c => c.is_default)
  selectedDefault.value = def?.provider ?? ''
}

async function save(provider) {
  saving[provider] = true
  statuses[provider] = null
  try {
    await llmStore.upsert(provider, {
      api_key:           forms[provider].api_key,
      base_url:          forms[provider].base_url,
      default_model:     forms[provider].default_model,
      max_output_tokens: forms[provider].max_output_tokens,
      enabled:           forms[provider].enabled,
    })
    syncFormsFromStore()
  } catch (e) {
    statuses[provider] = { ok: false, error: e.response?.data?.error ?? e.message }
  } finally {
    saving[provider] = false
  }
}

async function test(provider) {
  testing[provider] = true
  statuses[provider] = null
  try {
    const result = await llmStore.testConnection(provider)
    statuses[provider] = result
  } catch (e) {
    statuses[provider] = { ok: false, error: e.response?.data?.error ?? e.message }
  } finally {
    testing[provider] = false
  }
}

async function saveDefault(provider) {
  const p = provider || selectedDefault.value
  if (!p) return
  try {
    await llmStore.setDefault(p)
  } catch (e) {
    console.error('Failed to set default provider', e)
  }
}

// -- System Prompts --
const systemPrompts = reactive({
  workflow_generator_base: '',
  skill_analyser_preamble: '',
  workflow_refinement_addendum: '',
  workflow_decompose: '',
  workflow_categorise: '',
  workflow_wire: '',
  workflow_implement_finish: '',
  workflow_debugger: '',
})
const savingPrompt = reactive({})

async function fetchPrompts() {
  try {
    const { data } = await api.get('/designer/prompts')
    Object.assign(systemPrompts, data)
  } catch (e) {
    console.error('Failed to fetch system prompts', e)
  }
}

async function savePrompt(id) {
  savingPrompt[id] = true
  try {
    await api.put(`/designer/prompts/${id}`, { content: systemPrompts[id] })
  } catch (e) {
    alert('Failed to save prompt: ' + (e.response?.data?.error ?? e.message))
  } finally {
    savingPrompt[id] = false
  }
}


// -- Local Models Manager --
const localModels = ref([])
const modelsLoading = ref(false)
const modelsError = ref(null)
const downloadModelName = ref('')
const downloadProgress = reactive({
  active: false,
  status: '',
  percent: 0,
  completed: '0 B',
  total: '0 B'
})

function formatSize(bytes) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

async function fetchLocalModels() {
  modelsLoading.value = true
  modelsError.value = null
  try {
    const { data } = await api.get('/llm-configs/local/models')
    localModels.value = data.models ?? []
  } catch (e) {
    modelsError.value = e.response?.data?.error ?? e.message
  } finally {
    modelsLoading.value = false
  }
}

async function deleteModel(modelName) {
  if (!confirm(`Are you sure you want to delete ${modelName}?`)) return
  try {
    await api.delete(`/llm-configs/local/models?model=${encodeURIComponent(modelName)}`)
    await fetchLocalModels()
  } catch (e) {
    alert('Failed to delete model: ' + (e.response?.data?.error ?? e.message))
  }
}

async function downloadModel() {
  const modelName = downloadModelName.value.trim()
  if (!modelName) return

  downloadProgress.active = true
  downloadProgress.status = 'Connecting to Ollama...'
  downloadProgress.percent = 0
  downloadProgress.completed = '0 B'
  downloadProgress.total = '0 B'

  try {
    const token = sessionStorage.getItem('access_token')
    const response = await fetch('/api/v1/llm-configs/local/models/pull', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({ model: modelName })
    })

    if (!response.ok) {
      const errText = await response.text()
      let errMsg = response.statusText
      try {
        const parsed = JSON.parse(errText)
        errMsg = parsed.error || errMsg
      } catch (e) {}
      throw new Error(`Failed with status ${response.status}: ${errMsg}`)
    }

    const reader = response.body.getReader()
    const decoder = new TextDecoder("utf-8")
    let buffer = ""

    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split("\n")
      buffer = lines.pop() // keep last incomplete line
      
      for (const line of lines) {
        if (line.trim()) {
          const data = JSON.parse(line)
          if (data.status) {
            downloadProgress.status = data.status
          }
          if (data.total) {
            downloadProgress.total = formatSize(data.total)
            if (data.completed) {
              downloadProgress.completed = formatSize(data.completed)
              downloadProgress.percent = Math.min(100, Math.round((data.completed / data.total) * 100))
            }
          }
          if (data.status === 'success') {
            downloadProgress.percent = 100
          }
        }
      }
    }

    downloadProgress.status = 'Success!'
    downloadProgress.percent = 100
    downloadModelName.value = ''
    await fetchLocalModels()
    alert('Model downloaded successfully!')
  } catch (e) {
    alert('Failed to download model: ' + e.message)
  } finally {
    downloadProgress.active = false
  }
}

onMounted(async () => {
  if (!isAdmin.value) return

  // Load core settings
  try {
    await Promise.allSettled([
      llmStore.fetchAll(),
      llmStore.fetchProfiles(),
      llmStore.fetchDesignerProfiles(),
      fetchPrompts(),
      fetchLocalModels()
    ])
    syncFormsFromStore()
  } catch (e) { console.error('Failed to load core settings', e) }

  // Load Enterprise/SuperAdmin settings (may fail if not licensed)
  try {
    await Promise.allSettled([
      entStore.fetchBranding().then(() => Object.assign(brandingForm, entStore.branding)),
      entStore.fetchSecrets().then(() => {
        secretsForm.value = Object.entries(entStore.secrets).map(([k, v]) => ({ name: k, value: '' }))
      }),
      entStore.fetchAuditLogs(),
      entStore.fetchStatus()
    ])
  } catch (e) { console.error('Failed to load enterprise settings', e) }

  if (authStore.hasRole('super_admin')) {
    try {
      await fetchTelemetry()
    } catch (e) { console.error('Failed to fetch telemetry', e) }
  }
})

const telemetryEnabled = ref(false)

async function fetchTelemetry() {
  try {
    const { data } = await api.get('/admin/telemetry')
    telemetryEnabled.value = data.enabled
  } catch (e) { console.error('Failed to fetch telemetry', e) }
}

async function toggleTelemetry() {
  const next = !telemetryEnabled.value
  try {
    await api.post('/admin/telemetry', { enabled: next })
    telemetryEnabled.value = next
  } catch (e) { alert('Failed to toggle telemetry') }
}

function formatDateTime(ts) {
  if (!ts) return '-'
  return new Date(ts).toLocaleString(undefined, {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
  })
}
</script>
