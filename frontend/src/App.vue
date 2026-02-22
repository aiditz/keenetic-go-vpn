<template>
  <div
    class="h-screen flex flex-col font-sans selection:bg-cyan-500 selection:text-white bg-slate-900 text-slate-300"
  >
    <!-- Header -->
    <header
      class="bg-slate-800 border-b border-slate-700 p-4 flex justify-between items-center shadow-md z-10"
    >
      <div class="flex items-center gap-4">
        <div
          class="bg-cyan-600 p-2 rounded-lg text-white shadow-lg shadow-cyan-500/20"
        >
          <i class="fa-solid fa-network-wired text-xl"></i>
        </div>
        <div>
          <h1
            class="font-bold text-lg text-slate-100 tracking-wide leading-tight"
          >
            Keenetic Go VPN
          </h1>
          <div class="text-xs text-slate-500 font-mono">{{ host }}</div>
        </div>

        <!-- Tabs -->
        <nav class="ml-6 flex gap-2">
          <button
            type="button"
            :class="navClass('devices')"
            @click="setPage('devices')"
          >
            <i class="fa-solid fa-list mr-1.5"></i>
            Devices
          </button>
          <button
            type="button"
            :class="navClass('routes')"
            @click="setPage('routes')"
          >
            <i class="fa-solid fa-route mr-1.5"></i>
            Domain routes
          </button>
        </nav>
      </div>

      <div class="flex items-center gap-4">
        <div
          class="flex items-center gap-3 px-4 py-2 bg-slate-700/50 rounded-lg border border-slate-700"
        >
          <i class="fa-regular fa-user text-slate-400"></i>
          <span class="text-sm font-medium text-slate-200">{{ user }}</span>
        </div>
        <button
          @click="logout"
          class="text-slate-400 hover:text-red-400 transition duration-200"
          title="Logout"
        >
          <i class="fa-solid fa-right-from-bracket text-lg"></i>
        </button>
      </div>
    </header>

    <!-- Page content -->
    <main
      class="flex-1 overflow-hidden flex flex-col p-4 max-w-7xl mx-auto w-full"
    >
      <component :is="currentComponent" />
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import DevicesPage from './components/DevicesPage.vue'
import DomainRoutesPage from './components/DomainRoutesPage.vue'

// Server data from index.html
const user = window.SERVER_DATA?.User || 'Admin'
const host = window.SERVER_DATA?.Host || 'Keenetic'
const ttlStr = window.SERVER_DATA?.TTL || '24h'

// Page state
const currentPage = ref('devices') // default page is Devices

const currentComponent = computed(() =>
  currentPage.value === 'devices' ? DevicesPage : DomainRoutesPage
)

const navClass = (page) =>
  [
    'px-3 py-1.5 rounded-md text-sm font-medium transition',
    currentPage.value === page
      ? 'bg-cyan-600 text-white shadow shadow-cyan-500/30'
      : 'text-slate-400 hover:text-white hover:bg-slate-700/60',
  ].join(' ')

// Session / logout
const parseTTL = (s) => {
  if (s.endsWith('h')) return parseInt(s) * 60 * 60 * 1000
  if (s.endsWith('m')) return parseInt(s) * 60 * 1000
  return 24 * 60 * 60 * 1000
}
const SESSION_MS = parseTTL(ttlStr)
let sessionTimer = null

const logout = () => {
  const url = new URL(window.location.href)
  url.username = 'logout'
  url.password = 'logout'
  window.location.href = url.href
}

const resetSession = () => {
  if (sessionTimer) clearTimeout(sessionTimer)
  sessionTimer = setTimeout(logout, SESSION_MS)
}

const setPage = (page) => {
  if (page !== 'devices' && page !== 'routes') return
  currentPage.value = page
  resetSession()
}

const onUserActivity = () => {
  resetSession()
}

onMounted(() => {
  window.addEventListener('click', onUserActivity)
  window.addEventListener('keydown', onUserActivity)
  resetSession()
})

onBeforeUnmount(() => {
  window.removeEventListener('click', onUserActivity)
  window.removeEventListener('keydown', onUserActivity)
  if (sessionTimer) clearTimeout(sessionTimer)
})
</script>