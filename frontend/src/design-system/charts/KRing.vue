<script setup lang="ts">
import { computed } from 'vue'
import type { Tone } from '@/api/types'
import { toneClass } from '../tone'
/** Circular score (PDDikti readiness, KRS SKS usage). */
const props = withDefaults(defineProps<{ value: number; max?: number; tone?: Tone; size?: number }>(), { max: 100, tone: 'blue', size: 140 })
const R = 52
const C = 2 * Math.PI * R
const pct = computed(() => Math.max(0, Math.min(1, props.value / (props.max || 1))))
</script>

<template>
  <div class="relative grid place-items-center" :style="{ width: size + 'px', height: size + 'px' }">
    <svg viewBox="0 0 120 120" class="absolute inset-0 -rotate-90">
      <circle cx="60" cy="60" :r="R" fill="none" class="stroke-surface-3" stroke-width="12" />
      <circle cx="60" cy="60" :r="R" fill="none" :class="toneClass[tone].stroke" stroke-width="12" stroke-linecap="round" :stroke-dasharray="`${pct * C} ${C}`" class="transition-[stroke-dasharray] duration-700 ease-ko" />
    </svg>
    <div class="relative text-center"><slot /></div>
  </div>
</template>
