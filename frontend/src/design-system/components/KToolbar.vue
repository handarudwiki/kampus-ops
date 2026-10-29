<script setup lang="ts">
/** Filter row that sits above a table inside a KCard. Left: filters, right: actions. */
</script>

<template>
  <div class="flex flex-col gap-3 border-b border-line px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
    <div class="flex flex-wrap items-center gap-2"><slot /></div>
    <div v-if="$slots.right" class="flex flex-wrap items-center gap-2"><slot name="right" /></div>
  </div>
</template>
