<script setup lang="ts">
import { computed } from 'vue'
import type { Tone } from '@/api/types'
import { toneClass } from '../tone'
const props = withDefaults(defineProps<{ value: number; max?: number; tone?: Tone; label?: string }>(), { max: 100, tone: 'blue' })
const pct = computed(() => Math.max(0, Math.min(100, (props.value / (props.max || 1)) * 100)))
</script>

<template>
  <div class="h-2 w-full overflow-hidden rounded-pill bg-surface-3" role="progressbar" :aria-valuenow="value" :aria-valuemax="max" :aria-label="label">
    <div :class="['h-full rounded-pill transition-[width] duration-500 ease-ko', toneClass[tone].bar]" :style="{ width: pct + '%' }" />
  </div>
</template>
