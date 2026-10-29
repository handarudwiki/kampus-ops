<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { TrendingDown, TrendingUp } from 'lucide-vue-next'
import type { Tone } from '@/api/types'
import { rupiahShort, splitFigure } from '../format'
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
/** Long rupiah amounts are compacted (Rp 5,7 M) so every KPI fits one line at every width. */
const display = computed(() => {
  const m = props.value.match(/^Rp\s?([\d.]+)$/)
  if (m) {
    const n = Number(m[1]!.replace(/\./g, ''))
    if (n >= 1e6) return rupiahShort(n)
  }
  return props.value
})
const parts = computed(() => splitFigure(display.value))
const long = computed(() => display.value.length > 9)
</script>

<template>
  <component :is="to ? RouterLink : 'div'" :to="to" class="flex min-h-[148px] flex-col gap-3 rounded-card bg-surface p-5 shadow-card transition-colors duration-ko" :class="to && 'hover:bg-surface-2/60'">
    <div class="flex items-center gap-2 text-base font-semibold text-ink-2">
      <KDot :tone="tone ?? 'blue'" size="md" />{{ label }}
    </div>
    <div :class="['tabular font-bold tracking-tight', long ? 'text-lg sm:text-xl' : 'text-xl sm:text-2xl']" :title="value">
      {{ parts[0] }}<span class="text-ink-3">{{ parts[1] }}</span>
    </div>
    <div v-if="hint" class="mt-auto inline-flex w-fit items-center gap-1.5 rounded-pill bg-surface-2 px-2.5 py-1 text-xs font-semibold text-ink-2">
      <TrendingUp v-if="trend === 'up'" class="size-3.5 text-green" />
      <TrendingDown v-else-if="trend === 'down'" class="size-3.5 text-red" />
      {{ hint }}
    </div>
  </component>
</template>
