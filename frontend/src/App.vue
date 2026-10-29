<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AppShell from '@/layouts/AppShell.vue'
import KToaster from '@/design-system/components/KToaster.vue'
import KConfirmHost from '@/design-system/components/KConfirmHost.vue'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'

useUiStore()
const route = useRoute()
const auth = useAuthStore()
const bare = computed(() => route.meta.bare || !auth.isAuthed)
</script>

<template>
  <RouterView v-if="bare" />
  <AppShell v-else>
    <RouterView v-slot="{ Component, route: r }">
      <component :is="Component" :key="r.path + (auth.activeRole ?? '')" />
    </RouterView>
  </AppShell>
  <KToaster />
  <KConfirmHost />
</template>
