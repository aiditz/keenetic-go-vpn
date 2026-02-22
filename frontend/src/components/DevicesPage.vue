<template>
  <div class="flex flex-col h-full">
    <!-- Top bar: refresh + search + last update -->
    <div class="flex flex-col md:flex-row justify-between items-end md:items-center mb-4 gap-4">
      <div class="text-sm text-slate-500 flex items-center gap-2">
        <span v-if="loading" class="text-cyan-400">
          <i class="fa-solid fa-circle-notch fa-spin"></i> Updating...
        </span>
        <span v-else>
          Updated <span class="text-slate-300 font-medium">{{ timeAgo }}</span>
        </span>
      </div>
      <div class="flex gap-3 w-full md:w-auto">
        <div class="relative flex-1 md:w-64">
          <i
            class="fa-solid fa-search absolute left-3 top-1/2 -translate-y-1/2 text-slate-500"
          ></i>
          <input
            v-model="search"
            type="text"
            placeholder="Search devices..."
            class="w-full bg-slate-800 border border-slate-600 text-slate-200 pl-10 pr-4 py-2 rounded-lg focus:outline-none focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 transition shadow-sm placeholder-slate-600"
          />
        </div>
        <button
          @click="fetchData"
          :disabled="loading"
          class="bg-cyan-600 hover:bg-cyan-500 text-white px-4 py-2 rounded-lg shadow-lg shadow-cyan-600/20 transition flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <i
            class="fa-solid fa-rotate"
            :class="{ 'fa-spin': loading }"
          ></i>
        </button>
      </div>
    </div>

    <!-- Devices table -->
    <div class="bg-slate-800 rounded-xl shadow-xl border border-slate-700 flex-1 flex flex-col overflow-hidden">
      <div class="overflow-auto flex-1">
        <table class="w-full text-left border-collapse">
          <thead
            class="bg-slate-900/50 text-slate-400 text-xs uppercase font-bold tracking-wider sticky top-0 z-10 backdrop-blur-sm"
          >
            <tr>
              <th class="p-2 w-12 text-center">St</th>
              <th class="p-2">Device Info</th>
              <th class="p-2">IP Address</th>
              <th class="p-2">Policy</th>
              <th class="p-2 w-16 text-center">Cfg</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-700 text-sm">
            <tr
              v-for="dev in filteredDevices"
              :key="dev.mac"
              class="hover:bg-slate-700/40 transition group"
            >
              <!-- Status -->
              <td class="p-2 text-center">
                <div
                  class="w-2.5 h-2.5 rounded-full inline-block"
                  :class="dev.online
                    ? 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]'
                    : 'bg-slate-600'"
                ></div>
              </td>

              <!-- Device info -->
              <td class="p-2">
                <div class="font-medium text-slate-200 text-sm">
                  {{ dev.name }}
                </div>
                <div class="text-[10px] text-slate-500 font-mono mt-0.5">
                  {{ dev.mac }}
                </div>
              </td>

              <!-- IP + static -->
              <td class="p-2">
                <div class="flex flex-col items-start gap-1">
                  <div class="flex items-center gap-2">
                    <span
                      v-if="dev.online && dev.ip && dev.ip !== '-'"
                      class="bg-slate-700 text-cyan-300 px-1.5 py-0.5 rounded text-xs font-mono border border-slate-600"
                    >
                      {{ dev.ip }}
                    </span>
                    <span v-else class="text-slate-600 text-xs italic">
                      Offline
                    </span>

                    <i
                      v-if="dev.static_ip && dev.static_ip === dev.ip"
                      class="fa-solid fa-thumbtack text-amber-500 text-[10px]"
                      title="Static IP Active"
                    ></i>
                  </div>

                  <span
                    v-if="dev.static_ip && dev.static_ip !== dev.ip"
                    class="text-[10px] text-amber-400 flex items-center gap-1 font-mono"
                    title="Reserved Static IP"
                  >
                    <i class="fa-solid fa-thumbtack"></i> {{ dev.static_ip }}
                  </span>
                </div>
              </td>

              <!-- Policy -->
              <td class="p-2">
                <div class="flex items-center gap-2">
                  <div class="relative flex-1">
                    <select
                      :value="getDevicePolicyValue(dev)"
                      @change="changePolicy(dev, $event)"
                      :disabled="actionStatus[dev.mac]?.loading"
                      class="appearance-none bg-slate-900 border border-slate-600 text-slate-300 text-xs py-1 pl-2 pr-2 rounded focus:outline-none focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 disabled:opacity-50 cursor-pointer hover:border-slate-500 transition w-full max-w-[150px]"
                      :class="{
                        'text-red-400 border-red-900': dev.access === 'DENY',
                      }"
                    >
                      <option value="">Default (Global)</option>
                      <option
                        v-for="p in policies"
                        :key="p.id"
                        :value="p.id"
                      >
                        {{ p.desc }}
                      </option>
                      <option disabled>──────────</option>
                      <option
                        value="_DENY_"
                        class="text-red-400 bg-slate-900 font-bold"
                      >
                        No Internet Access
                      </option>
                    </select>
                  </div>
                  <div class="w-4 h-4 flex items-center justify-center">
                    <i
                      v-if="actionStatus[dev.mac]?.loading"
                      class="fa-solid fa-circle-notch fa-spin text-cyan-500 text-xs"
                    ></i>
                    <i
                      v-else-if="actionStatus[dev.mac]?.success"
                      class="fa-solid fa-check text-emerald-500 text-xs"
                    ></i>
                    <i
                      v-else-if="actionStatus[dev.mac]?.error"
                      class="fa-solid fa-circle-xmark text-red-500 text-xs"
                    ></i>
                  </div>
                </div>
              </td>

              <!-- Config button -->
              <td class="p-2 text-center">
                <button
                  @click="openIpModal(dev)"
                  class="text-slate-500 hover:text-cyan-400 transition p-1.5 rounded-full hover:bg-slate-700/50"
                  title="Edit Static IP"
                >
                  <i class="fa-solid fa-pen text-xs"></i>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Static IP modal -->
    <div
      v-if="showModal"
      class="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4"
    >
      <div class="bg-slate-800 rounded-xl shadow-2xl border border-slate-700 w-full max-w-md overflow-hidden">
        <div class="bg-slate-900 p-4 border-b border-slate-700 flex justify-between items-center">
          <h3 class="font-bold text-slate-200">Static IP Settings</h3>
          <button
            @click="closeModal"
            class="text-slate-500 hover:text-slate-300"
          >
            <i class="fa-solid fa-xmark text-lg"></i>
          </button>
        </div>
        <div class="p-6 space-y-4">
          <div class="flex items-center gap-4">
            <div
              class="w-12 h-12 rounded-lg bg-slate-700 flex items-center justify-center text-cyan-400 text-xl"
            >
              <i class="fa-solid fa-desktop"></i>
            </div>
            <div>
              <div class="font-medium text-slate-200">
                {{ modalData.name }}
              </div>
              <div class="text-xs text-slate-500 font-mono">
                {{ modalData.mac }}
              </div>
            </div>
          </div>
          <div class="pt-2">
            <label class="flex items-center gap-2 cursor-pointer mb-3">
              <input
                type="checkbox"
                v-model="modalData.enabled"
                class="w-4 h-4 rounded border-slate-600 bg-slate-700 text-cyan-600"
              />
              <span class="text-sm text-slate-300">Enable Static IP</span>
            </label>
            <div v-if="modalData.enabled" class="space-y-1">
              <label class="text-xs text-slate-400 uppercase font-bold"
                >IP Address</label
              >
              <input
                v-model="modalData.ip"
                type="text"
                class="w-full bg-slate-900 border border-slate-600 text-slate-200 px-3 py-2 rounded focus:outline-none focus:border-cyan-500 placeholder-slate-600 font-mono"
                placeholder="192.168.1.x"
              />
            </div>
            <div
              v-else
              class="text-xs text-slate-500 bg-slate-900/50 p-3 rounded border border-slate-700 border-dashed"
            >
              Device will receive IP automatically via DHCP.
            </div>
          </div>
        </div>
        <div class="bg-slate-900/50 p-4 border-t border-slate-700 flex justify-end gap-3">
          <button
            @click="closeModal"
            class="px-4 py-2 rounded text-sm text-slate-400 hover:text-white hover:bg-slate-700"
          >
            Cancel
          </button>
          <button
            @click="saveStaticIp"
            :disabled="modalSaving"
            class="px-4 py-2 rounded text-sm bg-cyan-600 hover:bg-cyan-500 text-white shadow-lg disabled:opacity-50"
          >
            <i v-if="modalSaving" class="fa-solid fa-circle-notch fa-spin"></i>
            <span v-else>Save Changes</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, reactive } from 'vue'

const devices = ref([])
const policies = ref([])
const loading = ref(false)
const search = ref('')
const lastUpdateTs = ref(Date.now())
const timeAgo = ref('just now')
const actionStatus = reactive({})

const showModal = ref(false)
const modalSaving = ref(false)
const modalData = reactive({
  mac: '',
  name: '',
  enabled: false,
  ip: '',
})

const updateTimeAgo = () => {
  const diff = Math.floor((Date.now() - lastUpdateTs.value) / 1000)
  if (diff < 5) timeAgo.value = 'just now'
  else if (diff < 60) timeAgo.value = `${diff} seconds ago`
  else if (diff < 3600) timeAgo.value = `${Math.floor(diff / 60)} mins ago`
  else timeAgo.value = 'long time ago'
}

const fetchData = async () => {
  loading.value = true
  try {
    const res = await fetch('/api/data')
    if (!res.ok) throw new Error('Failed')
    const data = await res.json()
    devices.value = data.devices || []
    policies.value = data.policies || []
    lastUpdateTs.value = Date.now()
    updateTimeAgo()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

const filteredDevices = computed(() => {
  if (!search.value) return devices.value
  const s = search.value.toLowerCase()
  return devices.value.filter(
    (d) =>
      d.name.toLowerCase().includes(s) ||
      (d.ip && d.ip.includes(s)) ||
      (d.mac && d.mac.includes(s))
  )
})

const getDevicePolicyValue = (dev) => {
  if (dev.access === 'DENY') return '_DENY_'
  return dev.policy_id || ''
}

const changePolicy = async (dev, event) => {
  const newPolicyId = event.target.value
  const mac = dev.mac
  actionStatus[mac] = { loading: true, success: false, error: false }

  try {
    const res = await fetch('/api/policy', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mac, policy_id: newPolicyId }),
    })
    if (!res.ok) throw new Error('Err')

    if (newPolicyId === '_DENY_') {
      dev.access = 'DENY'
      dev.policy_id = ''
    } else {
      dev.access = 'PERMIT'
      dev.policy_id = newPolicyId
    }
    actionStatus[mac] = { loading: false, success: true, error: false }
  } catch (e) {
    console.error(e)
    actionStatus[mac] = { loading: false, success: false, error: true }
  } finally {
    setTimeout(() => {
      if (actionStatus[mac]) delete actionStatus[mac]
    }, 5000)
  }
}

const openIpModal = (dev) => {
  modalData.mac = dev.mac
  modalData.name = dev.name
  modalData.enabled = !!dev.static_ip
  modalData.ip = dev.static_ip || dev.ip || '192.168.1.'
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
}

const saveStaticIp = async () => {
  modalSaving.value = true
  try {
    const res = await fetch('/api/static_ip', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        mac: modalData.mac,
        ip: modalData.enabled ? modalData.ip : '',
      }),
    })
    if (!res.ok) throw new Error('Err')
    await fetchData()
    closeModal()
  } catch (e) {
    console.error(e)
    alert('Failed to set Static IP')
  } finally {
    modalSaving.value = false
  }
}

onMounted(() => {
  fetchData()
  setInterval(updateTimeAgo, 5000)
})
</script>