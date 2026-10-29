<script setup lang="ts">
/** Every page starts with this: title, one-line description, primary actions on the right. */
defineProps<{ title: string; description?: string }>()
</script>

<template>
  <div class="mb-5 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <div class="min-w-0">
      <div v-if="$slots.eyebrow" class="mb-1"><slot name="eyebrow" /></div>
      <h1 class="text-lg font-bold tracking-tight sm:text-xl">{{ title }}</h1>
      <p v-if="description" class="mt-1 text-base text-ink-3">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="flex flex-wrap items-center gap-2"><slot name="actions" /></div>
  </div>
</template>
