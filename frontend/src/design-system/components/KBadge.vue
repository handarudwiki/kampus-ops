<script setup lang="ts">
import type { Tone } from '@/api/types'
import { toneClass } from '../tone'
import KDot from './KDot.vue'
/**
 * pill  → tinted background chip (filters, tags, inline status in dense tables)
 * dot   → "• Within Budget" style: colored dot + text, no background
 */
withDefaults(defineProps<{ tone?: Tone; variant?: 'pill' | 'dot' }>(), { tone: 'gray', variant: 'pill' })
</script>

<template>
  <span v-if="variant === 'dot'" class="inline-flex items-center gap-1.5 whitespace-nowrap text-sm font-medium text-ink-2">
    <KDot :tone="tone" /><slot />
  </span>
  <span v-else :class="['inline-flex h-6 items-center gap-1.5 whitespace-nowrap rounded-pill px-2.5 text-xs font-semibold', toneClass[tone].soft]">
    <slot />
  </span>
</template>
