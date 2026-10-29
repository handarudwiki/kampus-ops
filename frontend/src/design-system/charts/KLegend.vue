<script setup lang="ts">
import type { Tone } from '@/api/types'
import KDot from '../components/KDot.vue'
defineProps<{ items: { label: string; tone: Tone }[] }>()
</script>

<template>
  <ul class="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm font-medium text-ink-2">
    <li v-for="it in items" :key="it.label" class="flex items-center gap-2"><KDot :tone="it.tone" />{{ it.label }}</li>
  </ul>
</template>
