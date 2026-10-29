<script setup lang="ts">
/** Filter row that sits above a table inside a KCard. Left: filters, right: actions. Wraps instead of overflowing. */
</script>

<template>
  <div class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-5 py-4">
    <div class="flex min-w-0 max-w-full flex-wrap items-center gap-2"><slot /></div>
    <div v-if="$slots.right" class="ml-auto flex flex-wrap items-center gap-2"><slot name="right" /></div>
  </div>
</template>
