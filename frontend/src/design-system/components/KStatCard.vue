<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { TrendingDown, TrendingUp } from 'lucide-vue-next'
import type { Tone } from '@/api/types'
import { splitFigure } from '../format'
import KDot from './KDot.vue'
/** KPI tile: tone dot + label, big figure with a muted tail, hint chip. */
const props = defineProps<{
  label: string
  value: string
  tone?: Tone
  hint?: string | null
  trend?: 'up' | 'down' | null
  to?: string
}>()
const parts = computed(() => splitFigure(props.value))
</script>

<template>
  <component :is="to ? RouterLink : 'div'" :to="to" class="flex flex-col justify-between gap-4 rounded-card bg-surface p-5 shadow-card transition-colors duration-ko" :class="to && 'hover:bg-surface-2/60'">
    <div class="flex items-center gap-2 text-base font-semibold text-ink-2">
      <KDot :tone="tone ?? 'blue'" size="md" />{{ label }}
    </div>
    <div class="tabular text-xl font-bold tracking-tight sm:text-2xl">
      {{ parts[0] }}<span class="text-ink-3">{{ parts[1] }}</span>
    </div>
    <div v-if="hint" class="inline-flex w-fit items-center gap-1.5 rounded-pill bg-surface-2 px-2.5 py-1 text-xs font-semibold text-ink-2">
      <TrendingUp v-if="trend === 'up'" class="size-3.5 text-green" />
      <TrendingDown v-else-if="trend === 'down'" class="size-3.5 text-red" />
      {{ hint }}
    </div>
  </component>
</template>
