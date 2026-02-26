<template>
  <div
    v-if="visible"
    class="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4"
  >
    <div
      class="bg-slate-800 rounded-xl shadow-2xl border border-slate-700 w-full max-w-sm overflow-hidden"
    >
      <div
        class="bg-slate-900 p-4 border-b border-slate-700 flex justify-between items-center"
      >
        <h3 class="font-bold text-slate-200">Login</h3>
      </div>
      <div class="p-6 space-y-4">
        <p class="text-xs text-slate-400">
          Enter your web panel credentials<br>
          <small>WEB_USER / WEB_PASS from <code>.env</code></small>.
        </p>
        <div class="space-y-2">
          <label class="text-xs text-slate-400 uppercase font-bold"
            >Username</label
          >
          <input
            v-model="loginUser"
            type="text"
            class="w-full bg-slate-900 border border-slate-600 text-slate-200 px-3 py-2 rounded text-sm focus:outline-none focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500"
          />
        </div>
        <div class="space-y-2">
          <label class="text-xs text-slate-400 uppercase font-bold"
            >Password</label
          >
          <input
            v-model="loginPass"
            type="password"
            class="w-full bg-slate-900 border border-slate-600 text-slate-200 px-3 py-2 rounded text-sm focus:outline-none focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500"
            @keyup.enter="doLogin"
          />
        </div>
        <div v-if="loginError" class="text-xs text-red-400">
          {{ loginError }}
        </div>
      </div>
      <div
        class="bg-slate-900/50 p-4 border-t border-slate-700 flex justify-end gap-3"
      >
        <button
          @click="onCancel"
          class="px-4 py-2 rounded text-sm text-slate-400 hover:text-white hover:bg-slate-700"
        >
          Cancel
        </button>
        <button
          @click="doLogin"
          :disabled="loginLoading"
          class="px-4 py-2 rounded text-sm bg-cyan-600 hover:bg-cyan-500 text-white shadow-lg disabled:opacity-50"
        >
          <i
            v-if="loginLoading"
            class="fa-solid fa-circle-notch fa-spin"
          ></i>
          <span v-else>Login</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  defaultUser: { type: String, default: '' },
})

const emit = defineEmits(['update:visible', 'login-success'])

const loginUser = ref(props.defaultUser || '')
const loginPass = ref('')
const loginLoading = ref(false)
const loginError = ref('')

watch(
  () => props.defaultUser,
  (val) => {
    if (!loginUser.value) loginUser.value = val || ''
  }
)

const close = () => {
  emit('update:visible', false)
  loginPass.value = ''
  loginError.value = ''
}

const onCancel = () => {
  close()
}

const doLogin = async () => {
  loginError.value = ''
  const u = loginUser.value.trim()
  const p = loginPass.value
  if (!u || !p) {
    loginError.value = 'Enter username and password'
    return
  }

  loginLoading.value = true
  try {
    const res = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ user: u, pass: p }),
    })
    if (!res.ok) {
      if (res.status === 401) {
        loginError.value = 'Invalid credentials'
      } else {
        loginError.value = 'Login failed'
      }
      return
    }
    emit('login-success')
    close()
  } catch (e) {
    console.error(e)
    loginError.value = 'Network error'
  } finally {
    loginLoading.value = false
  }
}
</script>