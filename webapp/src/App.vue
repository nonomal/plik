<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AppHeader from './components/AppHeader.vue'
import { settings } from './settings.js'
import { config } from './config.js'

const { t } = useI18n()

const bgStyle = computed(() => {
    const style = {}
    if (settings.backgroundImage) {
        style.backgroundImage = `url(${settings.backgroundImage})`
        style.backgroundSize = 'cover'
        style.backgroundPosition = 'center center'
        style.backgroundAttachment = 'fixed'
        style.backgroundRepeat = 'no-repeat'
    }
    if (settings.backgroundColor) {
        style.backgroundColor = settings.backgroundColor
    }
    return style
})

const overlayStyle = computed(() => ({
    backgroundColor: `rgba(0, 0, 0, ${settings.overlayOpacity ?? 0.55})`,
}))

const hasBackground = computed(() => !!settings.backgroundImage)

const footerHTML = computed(() => {
    if (settings.footer) return settings.footer
    if (config.abuseContact) {
        const link = `<a href="mailto:${config.abuseContact}" class="underline hover:text-surface-200">${config.abuseContact}</a>`
        return t('footer.abuseContact', { link })
    }
    return ''
})
</script>

<template>
  <div class="min-h-screen flex flex-col relative" :style="bgStyle">
    <!-- Dark overlay for readability -->
    <div v-if="hasBackground"
         class="fixed inset-0 z-0 pointer-events-none"
         :style="overlayStyle"></div>

    <!-- Header -->
    <AppHeader class="relative z-50" />

    <!-- Main Content Area -->
    <div class="flex-1 flex relative z-10">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </div>

    <!-- Footer -->
    <footer v-if="footerHTML"
            class="relative z-10 text-center text-xs text-surface-400 py-3"
            v-html="footerHTML" />
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

