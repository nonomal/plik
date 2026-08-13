<template>
  <component :is="viewComponent" v-bind="viewProps" />
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import UploadView from './UploadView.vue'
import DownloadView from './DownloadView.vue'
import ErrorView from './ErrorView.vue'

const route = useRoute()

const uploadId = computed(() => route.query.id)
const errorMessage = computed(() => route.query.err)

const viewComponent = computed(() => {
  if (uploadId.value) return DownloadView
  if (errorMessage.value) return ErrorView
  return UploadView
})

const viewProps = computed(() => {
  if (uploadId.value) return { id: uploadId.value }
  if (errorMessage.value) return {
    message: errorMessage.value,
    code: route.query.errcode || '',
    uri: route.query.uri || '',
  }
  return {}
})
</script>
