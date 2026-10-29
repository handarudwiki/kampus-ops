<script setup lang="ts">
withDefaults(defineProps<{ lines?: number; class?: string }>(), { lines: 1 })
</script>

<template>
  <div class="flex flex-col gap-2" aria-busy="true" aria-label="Memuat">
    <div v-for="i in lines" :key="i" :class="['h-4 animate-pulse rounded-pill bg-surface-3', $props.class]" :style="{ width: i === lines && lines > 1 ? '60%' : '100%' }" />
  </div>
</template>
