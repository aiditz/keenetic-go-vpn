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
          @click="doLogout"
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
      <component
        v-if="isAuthenticated"
        :is="currentComponent"
        :key="componentKey"
      />
      <div
        v-else
        class="flex-1 flex items-center justify-center text-slate-500 text-sm"
      >
        Please log in to access the panel.
      </div>
    </main>

    <!-- Login modal component -->
    <LoginModal
      v-model:visible="showLoginModal"
      :defaultUser="user"
      @login-success="onLoginSuccess"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import DevicesPage from './components/DevicesPage.vue'
import DomainRoutesPage from './components/DomainRoutesPage.vue'
import LoginModal from './components/LoginModal.vue'

const user = window.SERVER_DATA?.User || 'Admin'
const host = window.SERVER_DATA?.Host || 'Keenetic'
const ttlStr = window.SERVER_DATA?.TTL || '24h' // informational only

// Page switching
const currentPage = ref('devices') // default
const currentComponent = computed(() =>
  currentPage.value === 'devices' ? DevicesPage : DomainRoutesPage
)
const componentKey = ref(0)

const navClass = (page) =>
  [
    'px-3 py-1.5 rounded-md text-sm font-medium transition',
    currentPage.value === page
      ? 'bg-cyan-600 text-white shadow shadow-cyan-500/30'
      : 'text-slate-400 hover:text-white hover:bg-slate-700/60',
  ].join(' ')

const setPage = (page) => {
  if (page !== 'devices' && page !== 'routes') return
  currentPage.value = page
}

// Auth state
const isAuthenticated = ref(false)
const showLoginModal = ref(false)

// Global fetch interceptor: on 401 show login
const installFetchInterceptor = () => {
  if (!window.__KGOVPN_FETCH_PATCHED__) {
    const origFetch = window.fetch.bind(window)
    window.fetch = async (...args) => {
      const res = await origFetch(...args)
      if (res.status === 401) {
        isAuthenticated.value = false
        showLoginModal.value = true
      }
      return res
    }
    window.__KGOVPN_FETCH_PATCHED__ = true
  }
}

const checkAuth = async () => {
  try {
    const res = await fetch('/api/data')
    if (res.ok) {
      isAuthenticated.value = true
      showLoginModal.value = false
      componentKey.value++
    } else if (res.status === 401) {
      isAuthenticated.value = false
      showLoginModal.value = true
    } else {
      isAuthenticated.value = false
      showLoginModal.value = true
    }
  } catch (e) {
    console.error(e)
    isAuthenticated.value = false
    showLoginModal.value = true
  }
}

const onLoginSuccess = () => {
  isAuthenticated.value = true
  showLoginModal.value = false
  // Remount current page to trigger its onMounted fetch
  componentKey.value++
}

const doLogout = async () => {
  try {
    await fetch('/api/logout', { method: 'POST' })
  } catch (e) {
    console.error(e)
  } finally {
    isAuthenticated.value = false
    showLoginModal.value = true
    componentKey.value++
  }
}

onMounted(() => {
  installFetchInterceptor()
  checkAuth()
})
</script>