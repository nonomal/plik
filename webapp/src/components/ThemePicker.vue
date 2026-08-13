<script setup>
import { computed } from 'vue'
import { getAvailableThemes, currentTheme, setUserTheme } from '../settings.js'
import DropdownPicker from './DropdownPicker.vue'

defineProps({
    buttonClass: {
        type: String,
        default: 'btn-ghost text-sm',
    },
})

const themes = computed(() => getAvailableThemes())
const showPicker = computed(() => themes.value.length > 1)
</script>

<template>
  <DropdownPicker
      v-if="showPicker"
      id="theme-picker-toggle"
      :items="themes"
      :current="currentTheme"
      item-id-prefix="theme-option-"
      :button-class="buttonClass"
      title="Switch theme"
      dropdown-width="w-52"
      @select="setUserTheme">
    <template #icon>
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012
                 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2
                 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0
                 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486
                 8.485M7 17h.01" />
      </svg>
    </template>
    <slot />
  </DropdownPicker>
</template>
