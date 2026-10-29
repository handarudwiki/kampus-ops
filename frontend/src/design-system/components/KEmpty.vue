<script setup lang="ts">
import { Inbox } from 'lucide-vue-next'
defineProps<{ title?: string; description?: string }>()
</script>

<template>
  <div class="flex flex-col items-center justify-center gap-2 px-6 py-12 text-center">
    <div class="grid size-12 place-items-center rounded-pill bg-surface-2 text-ink-3">
      <slot name="icon"><Inbox class="size-5" /></slot>
    </div>
    <p class="text-md font-bold">{{ title ?? 'Belum ada data' }}</p>
    <p v-if="description" class="max-w-sm text-sm text-ink-3">{{ description }}</p>
    <div v-if="$slots.default" class="mt-2"><slot /></div>
  </div>
</template>
