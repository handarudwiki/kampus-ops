<script setup lang="ts">
import { Inbox } from 'lucide-vue-next'
/** Empty state: layered dot illustration (echoes the logo) + title + hint + optional action. */
defineProps<{ title?: string; description?: string }>()
</script>

<template>
  <div class="flex flex-col items-center justify-center gap-2 px-6 py-12 text-center">
    <div class="relative mb-2 grid size-20 place-items-center">
      <span class="absolute inset-0 rounded-pill bg-surface-2" />
      <span class="absolute inset-3 rounded-pill bg-surface-3" />
      <span class="absolute right-1 top-2 size-3 rounded-pill bg-blue/70" />
      <span class="absolute bottom-2 left-1 size-2.5 rounded-pill bg-amber/80" />
      <span class="absolute bottom-0 right-4 size-2 rounded-pill bg-green/80" />
      <span class="relative text-ink-3"><slot name="icon"><Inbox class="size-6" /></slot></span>
    </div>
    <p class="text-md font-bold">{{ title ?? 'Belum ada data' }}</p>
    <p v-if="description" class="max-w-sm text-sm text-ink-3">{{ description }}</p>
    <div v-if="$slots.default" class="mt-3"><slot /></div>
  </div>
</template>
