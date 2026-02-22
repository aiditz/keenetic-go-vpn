<template>
  <div class="flex flex-col h-full space-y-4">
    <!-- Top controls -->
    <div
      class="flex flex-col md:flex-row justify-between gap-4 items-start md:items-center"
    >
      <!-- Interface selector + auto-refresh toggle -->
      <div class="space-y-3 w-full md:w-auto">
        <div class="space-y-2">
          <div class="text-sm text-slate-400 font-medium">
            Interface for domain routes
          </div>
          <div class="flex items-center gap-3">
            <select
              v-model="selectedInterface"
              @change="onInterfaceChange"
              class="bg-slate-900 border border-slate-600 text-slate-200 text-sm px-3 py-2 rounded-md focus:outline-none focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 min-w-[220px]"
            >
              <option value="" disabled>Select interface...</option>
              <option
                v-for="iface in interfaces"
                :key="iface.id"
                :value="iface.id"
              >
                {{ iface.name }} ({{ iface.id }})
              </option>
            </select>
            <span v-if="ifaceSaving" class="text-xs text-slate-500">
              <i class="fa-solid fa-circle-notch fa-spin text-cyan-500"></i>
              saving...
            </span>
          </div>
          <div
            class="text-xs text-slate-500"
            v-if="currentInterfaceDescription"
          >
            {{ currentInterfaceDescription }}
          </div>
        </div>

        <!-- Auto refresh toggle -->
        <div class="flex items-center gap-2 text-xs text-slate-400">
          <label class="inline-flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              v-model="autoRefresh"
              @change="onAutoRefreshChange"
              class="w-4 h-4 rounded border-slate-600 bg-slate-700 text-cyan-600"
            />
            <span>Auto refresh all routes</span>
          </label>
          <i
            class="fa-solid fa-circle-question text-slate-500"
            title="Once per day at 00:00 UTC, the backend looks up all domains and updates routes where IPs changed."
          ></i>
        </div>
      </div>

      <!-- Add domain + Sync All -->
      <div class="flex flex-col md:flex-row gap-2 w-full md:w-auto">
        <div class="flex items-center gap-2 flex-1">
          <input
            v-model="newDomainInput"
            type="text"
            placeholder="example.com or https://example.com/path"
            class="flex-1 bg-slate-900 border border-slate-600 text-slate-200 px-3 py-2 rounded-md focus:outline-none focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 placeholder-slate-600"
            @keyup.enter="addDomain"
          />
          <button
            @click="addDomain"
            :disabled="adding || !newDomainTrim || !selectedInterface"
            class="bg-cyan-600 hover:bg-cyan-500 text-white px-4 py-2 rounded-md shadow-lg shadow-cyan-600/20 text-sm flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <i v-if="adding" class="fa-solid fa-circle-notch fa-spin"></i>
            <i v-else class="fa-solid fa-plus"></i>
            <span>Add domain</span>
          </button>
        </div>
        <div class="flex items-center">
          <button
            @click="syncAll"
            :disabled="syncAllLoading || !domains.length || !selectedInterface"
            class="bg-emerald-600 hover:bg-emerald-500 text-white px-3 py-2 rounded-md shadow-lg shadow-emerald-600/20 text-xs flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed w-full md:w-auto justify-center"
          >
            <i
              v-if="syncAllLoading"
              class="fa-solid fa-circle-notch fa-spin"
            ></i>
            <i v-else class="fa-solid fa-arrows-rotate"></i>
            <span>Sync All</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Domains table -->
    <div
      class="bg-slate-800 rounded-xl shadow-xl border border-slate-700 flex-1 flex flex-col overflow-hidden"
    >
      <div
        class="flex items-center justify-between px-4 py-2 border-b border-slate-700 bg-slate-900/60 text-xs text-slate-400"
      >
        <span>Domains routing table</span>
        <span v-if="loading"
          ><i class="fa-solid fa-circle-notch fa-spin text-cyan-500"></i>
          Loading...</span
        >
      </div>

      <div class="flex-1 overflow-auto">
        <table class="w-full text-left border-collapse">
          <thead
            class="bg-slate-900/70 text-slate-400 text-xs uppercase font-bold tracking-wider sticky top-0 z-10 backdrop-blur"
          >
            <tr>
              <th class="p-2">Domain</th>
              <th class="p-2">IP addresses</th>
              <th class="p-2 w-40">Last lookup</th>
              <th class="p-2 w-48 text-center">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-700 text-sm">
            <tr v-if="!domains.length && !loading">
              <td
                colspan="4"
                class="p-4 text-center text-slate-500 text-xs"
              >
                No domains configured. Add one above to start routing by
                domain.
              </td>
            </tr>

            <tr
              v-for="entry in sortedDomains"
              :key="entry.domain"
              class="hover:bg-slate-700/40 transition group align-top"
            >
              <!-- Domain -->
              <td class="p-2 align-middle">
                <div class="font-medium text-slate-200 text-sm">
                  {{ entry.domain }}
                </div>
                <div
                  v-if="entry.applied_interface"
                  class="text-[10px] text-emerald-400 mt-0.5"
                >
                  Applied via: {{ entry.applied_interface }}
                </div>
              </td>

              <!-- IPs -->
              <td class="p-2 align-middle">
                <div class="flex flex-col gap-1">
                  <div
                    v-if="entry.ips && entry.ips.length"
                    class="flex flex-wrap gap-1"
                  >
                    <span
                      v-for="ip in entry.ips"
                      :key="ip"
                      class="bg-slate-900 border border-slate-600 text-cyan-300 text-xs font-mono px-2 py-0.5 rounded"
                    >
                      {{ ip }}
                    </span>
                  </div>
                  <div v-else class="text-xs text-slate-500 italic">
                    No IPs. Use sync or edit manually.
                  </div>

                  <div
                    v-if="entry.applied_ips && entry.applied_ips.length"
                    class="text-[10px] text-emerald-400 mt-1"
                  >
                    Applied IPs:
                    <span class="font-mono">
                      {{ entry.applied_ips.join(', ') }}
                    </span>
                  </div>
                </div>
              </td>

              <!-- Last lookup -->
              <td class="p-2 align-middle">
                <div class="text-xs text-slate-400">
                  <span v-if="entry.last_lookup">
                    {{ formatLastLookup(entry.last_lookup) }}
                  </span>
                  <span v-else class="italic text-slate-500">never</span>
                </div>
              </td>

              <!-- Actions -->
              <td class="p-2 align-middle">
                <div
                  class="flex items-center justify-center gap-2 text-xs"
                >
                  <!-- sync (lookup) -->
                  <button
                    class="px-2 py-1 rounded bg-slate-900 border border-slate-600 text-slate-200 hover:border-cyan-500 hover:text-cyan-300 transition"
                    @click="lookupDomain(entry)"
                    :disabled="isBusy(entry.domain)"
                    title="Lookup (nslookup) and update IPs"
                  >
                    <i class="fa-solid fa-arrows-rotate"></i>
                  </button>
                  <!-- edit -->
                  <button
                    class="px-2 py-1 rounded bg-slate-900 border border-slate-600 text-slate-200 hover:border-amber-500 hover:text-amber-300 transition"
                    @click="openEditModal(entry)"
                    :disabled="isBusy(entry.domain)"
                    title="Edit IPs manually"
                  >
                    <i class="fa-solid fa-pen"></i>
                  </button>
                  <!-- apply -->
                  <button
                    class="px-2 py-1 rounded bg-slate-900 border border-emerald-600 text-emerald-300 hover:bg-emerald-600/10 transition"
                    @click="applyDomain(entry)"
                    :disabled="isBusy(entry.domain) || !selectedInterface"
                    title="Apply routes on router"
                  >
                    <i class="fa-solid fa-upload"></i>
                  </button>
                  <!-- delete -->
                  <button
                    class="px-2 py-1 rounded bg-slate-900 border border-red-700 text-red-400 hover:bg-red-700/10 transition"
                    @click="deleteDomain(entry)"
                    :disabled="isBusy(entry.domain)"
                    title="Delete domain and its routes"
                  >
                    <i class="fa-solid fa-trash"></i>
                  </button>

                  <div
                    class="w-4 h-4 flex items-center justify-center"
                  >
                    <i
                      v-if="actionStatus[entry.domain]?.loading"
                      class="fa-solid fa-circle-notch fa-spin text-cyan-500 text-xs"
                    ></i>
                    <i
                      v-else-if="actionStatus[entry.domain]?.success"
                      class="fa-solid fa-check text-emerald-500 text-xs"
                    ></i>
                    <i
                      v-else-if="actionStatus[entry.domain]?.error"
                      class="fa-solid fa-circle-xmark text-red-500 text-xs"
                    ></i>
                  </div>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Edit IPs modal -->
    <div
      v-if="showEditModal"
      class="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4"
    >
      <div
        class="bg-slate-800 rounded-xl shadow-2xl border border-slate-700 w-full max-w-lg overflow-hidden"
      >
        <div
          class="bg-slate-900 p-4 border-b border-slate-700 flex justify-between items-center"
        >
          <h3 class="font-bold text-slate-200">Edit IP addresses</h3>
          <button
            @click="closeEditModal"
            class="text-slate-500 hover:text-slate-300"
          >
            <i class="fa-solid fa-xmark text-lg"></i>
          </button>
        </div>
        <div class="p-6 space-y-4">
          <div>
            <div class="text-sm text-slate-400">Domain</div>
            <div class="font-mono text-slate-200 text-sm">
              {{ editForm.domain }}
            </div>
          </div>
          <div>
            <label
              class="text-xs text-slate-400 uppercase font-bold"
              >IP addresses (one per line)</label
            >
            <textarea
              v-model="editForm.ipsText"
              rows="5"
              class="mt-1 w-full bg-slate-900 border border-slate-600 text-slate-200 px-3 py-2 rounded font-mono text-xs focus:outline-none focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500"
            ></textarea>
          </div>
        </div>
        <div
          class="bg-slate-900/50 p-4 border-t border-slate-700 flex justify-end gap-3"
        >
          <button
            @click="closeEditModal"
            class="px-4 py-2 rounded text-sm text-slate-400 hover:text-white hover:bg-slate-700"
          >
            Cancel
          </button>
          <button
            @click="saveEdit"
            :disabled="editSaving"
            class="px-4 py-2 rounded text-sm bg-cyan-600 hover:bg-cyan-500 text-white shadow-lg disabled:opacity-50"
          >
            <i
              v-if="editSaving"
              class="fa-solid fa-circle-notch fa-spin"
            ></i>
            <span v-else>Save</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'
import { formatTimeAgoFromISO } from '../utils'

const interfaces = ref([]) // {id, name, description}
const selectedInterface = ref('')
const autoRefresh = ref(false)
const domains = ref([])

const loading = ref(false)
const ifaceSaving = ref(false)
const adding = ref(false)
const syncAllLoading = ref(false)

const newDomainInput = ref('')
const actionStatus = reactive({})

const showEditModal = ref(false)
const editSaving = ref(false)
const editForm = reactive({
  domain: '',
  ipsText: '',
})

const newDomainTrim = computed(() => newDomainInput.value.trim())

const sortedDomains = computed(() =>
  [...domains.value].sort((a, b) =>
    a.domain.localeCompare(b.domain)
  )
)

const currentInterfaceDescription = computed(() => {
  const id = selectedInterface.value
  if (!id) return ''
  const iface = interfaces.value.find((i) => i.id === id)
  return iface?.description || ''
})

const isBusy = (domain) => actionStatus[domain]?.loading

const setActionStatus = (domain, status) => {
  actionStatus[domain] = {
    loading: !!status.loading,
    success: !!status.success,
    error: !!status.error,
  }
  if (!status.loading) {
    setTimeout(() => {
      if (actionStatus[domain]) delete actionStatus[domain]
    }, 3000)
  }
}

// Extract domain from either plain domain or URL
const extractDomain = (input) => {
  const s = (input || '').trim()
  if (!s) return ''
  try {
    const url = s.includes('://') ? new URL(s) : new URL('http://' + s)
    return url.hostname.toLowerCase()
  } catch {
    // fallback: take part before first slash
    return s.split('/')[0].toLowerCase()
  }
}

const formatLastLookup = (iso) => {
  return formatTimeAgoFromISO(iso)
}

const upsertDomainEntry = (entry) => {
  const idx = domains.value.findIndex(
    (e) => e.domain.toLowerCase() === entry.domain.toLowerCase()
  )
  if (idx === -1) {
    domains.value.push(entry)
  } else {
    domains.value[idx] = entry
  }
}

const fetchData = async () => {
  loading.value = true
  try {
    const res = await fetch('/api/routes/data')
    if (!res.ok) throw new Error('Failed to fetch routes data')
    const data = await res.json()
    interfaces.value = data.interfaces || []
    selectedInterface.value = data.selected_interface || ''
    domains.value = data.domains || []
    autoRefresh.value = !!data.auto_refresh
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

const onInterfaceChange = async () => {
  const iface = selectedInterface.value
  if (!iface) return
  ifaceSaving.value = true
  try {
    const res = await fetch('/api/routes/interface', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ interface: iface }),
    })
    if (!res.ok) throw new Error('Failed to set interface')
  } catch (e) {
    console.error(e)
  } finally {
    ifaceSaving.value = false
  }
}

const onAutoRefreshChange = async () => {
  const enabled = autoRefresh.value
  try {
    const res = await fetch('/api/routes/auto_refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    })
    if (!res.ok) throw new Error('Failed to set auto-refresh')
  } catch (e) {
    console.error(e)
    autoRefresh.value = !enabled
    alert('Failed to update auto refresh setting')
  }
}

const addDomain = async () => {
  const raw = newDomainTrim.value
  if (!raw || !selectedInterface.value) return
  const domain = extractDomain(raw)
  if (!domain) {
    alert('Invalid domain or URL')
    return
  }

  adding.value = true
  try {
    const res = await fetch('/api/routes/domain/add', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ domain }),
    })
    if (!res.ok) throw new Error('Failed to add domain')
    const data = await res.json()
    if (data.entry) {
      upsertDomainEntry(data.entry)
    } else {
      await fetchData()
    }
    newDomainInput.value = ''
  } catch (e) {
    console.error(e)
    alert('Failed to add domain')
  } finally {
    adding.value = false
  }
}

const lookupDomain = async (entry) => {
  const domain = entry.domain
  setActionStatus(domain, { loading: true })
  try {
    const res = await fetch('/api/routes/domain/lookup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ domain }),
    })
    if (!res.ok) throw new Error('Failed to lookup')
    const data = await res.json()
    if (data.entry) {
      upsertDomainEntry(data.entry)
    } else {
      await fetchData()
    }
    setActionStatus(domain, { loading: false, success: true })
  } catch (e) {
    console.error(e)
    setActionStatus(domain, { loading: false, error: true })
    alert('Lookup failed')
  }
}

const openEditModal = (entry) => {
  editForm.domain = entry.domain
  editForm.ipsText = (entry.ips || []).join('\n')
  showEditModal.value = true
}

const closeEditModal = () => {
  showEditModal.value = false
  editForm.domain = ''
  editForm.ipsText = ''
}

const saveEdit = async () => {
  if (!editForm.domain) return
  editSaving.value = true
  const domain = editForm.domain
  const ips = editForm.ipsText
    .split('\n')
    .map((s) => s.trim())
    .filter((s) => s.length > 0)

  try {
    const res = await fetch('/api/routes/domain/edit', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ domain, ips }),
    })
    if (!res.ok) throw new Error('Failed to edit')
    const data = await res.json()
    if (data.entry) {
      upsertDomainEntry(data.entry)
    } else {
      await fetchData()
    }
    showEditModal.value = false
  } catch (e) {
    console.error(e)
    alert('Failed to save changes')
  } finally {
    editSaving.value = false
  }
}

const applyDomain = async (entry) => {
  const domain = entry.domain
  if (!selectedInterface.value) {
    alert('Select interface first')
    return
  }
  setActionStatus(domain, { loading: true })
  try {
    const res = await fetch('/api/routes/domain/apply', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ domain }),
    })
    if (!res.ok) throw new Error('Failed to apply')
    const data = await res.json()
    if (data.entry) {
      upsertDomainEntry(data.entry)
    } else {
      await fetchData()
    }
    setActionStatus(domain, { loading: false, success: true })
  } catch (e) {
    console.error(e)
    setActionStatus(domain, { loading: false, error: true })
    alert('Failed to apply routes')
  }
}

const deleteDomain = async (entry) => {
  const domain = entry.domain
  if (!confirm(`Delete domain "${domain}" and remove its routes?`)) return
  setActionStatus(domain, { loading: true })
  try {
    const res = await fetch('/api/routes/domain/delete', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ domain }),
    })
    if (!res.ok) throw new Error('Failed to delete')
    domains.value = domains.value.filter(
      (e) => e.domain.toLowerCase() !== domain.toLowerCase()
    )
    setActionStatus(domain, { loading: false, success: true })
  } catch (e) {
    console.error(e)
    setActionStatus(domain, { loading: false, error: true })
    alert('Failed to delete domain')
  }
}

const syncAll = async () => {
  if (!selectedInterface.value || !domains.value.length) return
  syncAllLoading.value = true
  try {
    const res = await fetch('/api/routes/sync_all', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: '{}',
    })
    if (!res.ok) throw new Error('Failed to sync all')
    const data = await res.json()
    if (data.entries) {
      domains.value = data.entries
    } else {
      await fetchData()
    }
  } catch (e) {
    console.error(e)
    alert('Failed to sync all domains')
  } finally {
    syncAllLoading.value = false
  }
}

onMounted(() => {
  fetchData()
})
</script>