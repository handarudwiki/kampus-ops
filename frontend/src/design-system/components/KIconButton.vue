<script setup lang="ts">
import { RouterLink } from 'vue-router'
/** Round icon button: card "↗" links, toolbar actions. Always pass a label for a11y. */
defineProps<{ label: string; to?: string; active?: boolean }>()
const cls = 'inline-grid size-8 place-items-center rounded-pill border border-line bg-surface text-ink-2 transition-colors duration-ko hover:bg-surface-2 hover:text-ink'
</script>

<template>
  <RouterLink v-if="to" :to="to" :class="cls" :aria-label="label" :title="label"><slot /></RouterLink>
  <button v-else type="button" :class="[cls, active && 'bg-surface-3 text-ink']" :aria-label="label" :title="label"><slot /></button>
</template>
