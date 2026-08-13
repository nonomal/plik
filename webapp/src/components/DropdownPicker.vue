<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const props = defineProps({
    /** HTML id for the toggle button (e.g. "theme-picker-toggle") */
    id: { type: String, required: true },
    /** Array of { name, label, flag? } */
    items: { type: Array, required: true },
    /** Currently selected item name (for checkmark) */
    current: { type: String, default: '' },
    /** Prefix for each option's id attribute (e.g. "theme-option-") */
    itemIdPrefix: { type: String, required: true },
    /** CSS class for the trigger button */
    buttonClass: { type: String, default: 'btn-ghost text-sm' },
    /** Tooltip for the trigger button */
    title: { type: String, default: '' },
    /** Tailwind width class for the dropdown panel */
    dropdownWidth: { type: String, default: 'w-52' },
    /** CSS class(es) applied to the trigger button when the dropdown is open */
    activeClass: { type: String, default: 'bg-surface-700/50 text-surface-100' },
})

const emit = defineEmits(['select'])

const open = ref(false)
const pickerRef = ref(null)

function toggle() {
    open.value = !open.value
}

function select(name) {
    emit('select', name)
    open.value = false
}

function onClickOutside(e) {
    if (pickerRef.value && !pickerRef.value.contains(e.target)) {
        open.value = false
    }
}

onMounted(() => document.addEventListener('click', onClickOutside, true))
onUnmounted(() => document.removeEventListener('click', onClickOutside, true))
</script>

<template>
  <div ref="pickerRef" class="relative w-full">
    <!-- Trigger button -->
    <button
        :id="id"
        :class="[buttonClass, open ? activeClass : '']"
        @click.stop="toggle"
        :title="title">
      <!-- Icon slot -->
      <slot name="icon" />
      <!-- Label slot -->
      <slot />
    </button>

    <!-- Dropdown -->
    <Transition name="dropdown">
      <div v-if="open"
           :class="['absolute right-0 top-full mt-2',
                     dropdownWidth,
                     'bg-surface-900 border border-surface-700/50',
                     'rounded-lg shadow-xl overflow-hidden z-50']">
        <div class="py-1 max-h-80 overflow-y-auto">
          <button
              v-for="item in items"
              :key="item.name"
              :id="itemIdPrefix + item.name"
              class="w-full flex items-center gap-3 px-3 py-2 text-sm text-left
                     transition-colors hover:bg-surface-700/30"
              :class="current === item.name
                ? 'text-accent-400'
                : 'text-surface-300 hover:text-surface-100'"
              @click="select(item.name)">

            <!-- Check mark for active item -->
            <svg v-if="current === item.name"
                 class="w-4 h-4 shrink-0 text-accent-400"
                 fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M5 13l4 4L19 7" />
            </svg>
            <span v-else class="w-4 h-4 shrink-0"></span>

            <!-- Optional flag image -->
            <img v-if="item.flag" :src="item.flag" class="w-5 h-3.5 shrink-0 rounded-[2px]" alt="" />

            <span class="truncate">{{ item.label }}</span>
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.dropdown-enter-active,
.dropdown-leave-active {
    transition: opacity 0.15s ease, transform 0.15s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
    opacity: 0;
    transform: translateY(-4px);
}
</style>
