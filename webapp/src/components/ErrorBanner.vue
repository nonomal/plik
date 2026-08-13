<script setup>
defineProps({
  message: { type: String, required: true },
})
defineEmits(['dismiss'])
</script>

<template>
  <div class="glass-card border-danger-500/50 p-4 flex items-center gap-3 animate-fade-in">
    <svg class="w-5 h-5 text-danger-500 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
    <span class="text-sm text-danger-500">{{ message }}</span>
    <button class="ml-auto text-surface-400 hover:text-surface-100" @click="$emit('dismiss')">
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
      </svg>
    </button>
  </div>
</template>
