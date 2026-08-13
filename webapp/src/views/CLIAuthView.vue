<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { auth } from '../authStore.js'
import { approveCLIAuth } from '../api.js'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const router = useRouter()
const { t: $t } = useI18n()

const code = ref('')
const comment = ref('')
const status = ref('pending')   // pending | approving | approved | error
const error = ref('')

onMounted(() => {
    // Pre-fill code from URL query
    if (route.query.code) {
        code.value = route.query.code
    }

    // Pre-fill token description with hostname
    const hostname = route.query.hostname || ''
    if (hostname) {
        comment.value = hostname
    } else {
        comment.value = $t('cliAuth.defaultComment')
    }
})

async function approve() {
    if (!code.value.trim()) return

    status.value = 'approving'
    error.value = ''
    try {
        await approveCLIAuth(code.value.trim(), comment.value.trim())
        status.value = 'approved'
    } catch (err) {
        status.value = 'error'
        error.value = err.message || $t('cliAuth.failedToApprove')
    }
}
</script>

<template>
  <div class="w-full min-h-[calc(100vh-3.5rem)] flex items-center justify-center p-4">
    <div class="glass-card p-8 max-w-md w-full text-center space-y-6">

      <!-- Header -->
      <div class="space-y-2">
        <div class="w-16 h-16 rounded-full bg-accent-500/20 flex items-center justify-center mx-auto">
          <svg class="w-8 h-8 text-accent-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round"
                  d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z" />
          </svg>
        </div>
        <h1 class="text-xl font-semibold text-surface-50">{{ $t('cliAuth.cliLogin') }}</h1>
        <p class="text-sm text-surface-400">
          {{ $t('cliAuth.authorizeDescription', { user: auth.user?.login || auth.user?.name }) }}
        </p>
      </div>

      <!-- Pending state: show code, description, and approve button -->
      <template v-if="status === 'pending' || status === 'approving'">
        <div class="space-y-4">
          <div>
            <label class="block text-xs text-surface-500 mb-1.5 text-left">{{ $t('cliAuth.verificationCode') }}</label>
            <input v-model="code"
                   type="text"
                   placeholder="XXXX-XXXX"
                   class="input-field text-center text-2xl font-mono tracking-widest"
                   :disabled="status === 'approving'"
                   @keydown.enter="approve" />
          </div>

          <div>
            <label class="block text-xs text-surface-500 mb-1.5 text-left">{{ $t('cliAuth.tokenDescription') }}</label>
            <input v-model="comment"
                   type="text"
                   :placeholder="$t('cliAuth.defaultComment')"
                   class="input-field text-sm"
                   :disabled="status === 'approving'"
                   @keydown.enter="approve" />
          </div>

          <button @click="approve"
                  :disabled="!code.trim() || status === 'approving'"
                  class="btn-primary w-full py-3">
            <template v-if="status === 'approving'">
              <span class="inline-flex items-center gap-2">
                <svg class="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
                </svg>
                {{ $t('cliAuth.authorizing') }}
              </span>
            </template>
            <template v-else>
              {{ $t('cliAuth.authorizeCli') }}
            </template>
          </button>
        </div>
      </template>

      <!-- Approved state -->
      <template v-if="status === 'approved'">
        <div class="space-y-4">
          <div class="w-14 h-14 rounded-full bg-emerald-500/20 flex items-center justify-center mx-auto">
            <svg class="w-7 h-7 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <h2 class="text-lg font-medium text-emerald-400">{{ $t('cliAuth.cliAuthorized') }}</h2>
          <p class="text-sm text-surface-400">
            {{ $t('cliAuth.authorizedDescription') }}
          </p>
          <button @click="router.push('/')"
                  class="text-sm text-accent-400 hover:text-accent-300 underline underline-offset-2 transition-colors">
            {{ $t('cliAuth.returnToPlik') }}
          </button>
        </div>
      </template>

      <!-- Error state -->
      <template v-if="status === 'error'">
        <div class="space-y-4">
          <div class="w-14 h-14 rounded-full bg-red-500/20 flex items-center justify-center mx-auto">
            <svg class="w-7 h-7 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </div>
          <h2 class="text-lg font-medium text-red-400">{{ $t('cliAuth.authorizationFailed') }}</h2>
          <p class="text-sm text-surface-400">{{ error }}</p>
          <button @click="status = 'pending'; error = ''"
                  class="text-sm text-accent-400 hover:text-accent-300 underline underline-offset-2 transition-colors">
            {{ $t('cliAuth.tryAgain') }}
          </button>
        </div>
      </template>

    </div>
  </div>
</template>
