<script setup lang="ts">
import { computed } from 'vue'
import type { Tone } from '@/api/types'
import { toneClass } from '../tone'
import KLegend from './KLegend.vue'

/** Semicircle gauge split into segments (reference "Total Revenue"). Center slot for the figure. */
const props = defineProps<{ segments: { label: string; value: number; tone: Tone }[]; legend?: boolean }>()
const R = 80
const C = Math.PI * R
const total = computed(() => props.segments.reduce((a, s) => a + s.value, 0) || 1)
const arcs = computed(() => {
  let acc = 0
  const gap = props.segments.length > 1 ? 6 : 0
  return props.segments.map((s) => {
    const len = (s.value / total.value) * C
    const a = { ...s, dash: `${Math.max(0, len - gap)} ${C * 2}`, offset: -acc }
    acc += len
    return a
  })
})
</script>

<template>
  <div class="flex flex-col items-center gap-4">
    <div class="relative w-full max-w-[260px]">
      <svg viewBox="0 0 200 110" class="w-full">
        <path d="M20,100 A80,80 0 0 1 180,100" fill="none" class="stroke-surface-3" stroke-width="18" stroke-linecap="round" />
        <path
          v-for="a in arcs" :key="a.label"
          d="M20,100 A80,80 0 0 1 180,100" fill="none" :class="toneClass[a.tone].stroke"
          stroke-width="18" stroke-linecap="round" :stroke-dasharray="a.dash" :stroke-dashoffset="a.offset"
        />
      </svg>
      <div class="absolute inset-x-0 bottom-0 flex flex-col items-center"><slot /></div>
    </div>
    <KLegend v-if="legend !== false" :items="segments" />
  </div>
</template>
