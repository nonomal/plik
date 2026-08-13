<script setup>
import { useRouter } from 'vue-router'

const props = defineProps({
  message: { type: String, required: true },
  code: { type: String, default: '' },
  uri: { type: String, default: '' },
})

const router = useRouter()

function goHome() {
  router.push({ path: '/' })
}
</script>

<template>
  <div class="flex justify-center flex-1 min-h-0">
    <div class="flex flex-col items-center justify-center p-6">
      <div class="glass-card border-danger-500/50 p-8 max-w-lg w-full text-center space-y-4 animate-fade-in">
        <!-- Error icon -->
        <div class="w-16 h-16 mx-auto rounded-full bg-danger-500/10 flex items-center justify-center">
          <svg class="w-8 h-8 text-danger-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4.5c-.77-.833-2.694-.833-3.464 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z" />
          </svg>
        </div>

        <!-- Status code -->
        <div v-if="code" class="text-5xl font-bold text-danger-500/80">
          {{ code }}
        </div>

        <!-- Error message -->
        <p class="text-lg text-surface-200">
          {{ decodeURIComponent(message) }}
        </p>

        <!-- Original URI -->
        <p v-if="uri" class="text-sm text-surface-500 font-mono break-all">
          {{ decodeURIComponent(uri) }}
        </p>

        <!-- Back button -->
        <button class="btn-primary px-6 py-2 mt-4" @click="goHome">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
          </svg>
          Go to Upload
        </button>
      </div>
    </div>
  </div>
</template>
