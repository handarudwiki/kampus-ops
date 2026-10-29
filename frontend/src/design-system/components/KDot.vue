<script setup lang="ts">
import type { Tone } from '@/api/types'
import { toneClass } from '../tone'
defineProps<{ tone: Tone; size?: 'sm' | 'md' }>()
</script>

<template>
  <span :class="['inline-block shrink-0 rounded-pill', size === 'md' ? 'size-2.5' : 'size-2', toneClass[tone].dot]" aria-hidden="true" />
</template>
