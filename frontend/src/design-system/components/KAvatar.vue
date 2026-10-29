<script setup lang="ts">
import { computed } from 'vue'
import { initials } from '../format'
import { toneClass, toneFor } from '../tone'
const props = withDefaults(defineProps<{ name: string; size?: 'sm' | 'md' | 'lg' }>(), { size: 'md' })
const tone = computed(() => toneFor(props.name))
</script>

<template>
  <span
    :class="[
      'inline-grid shrink-0 place-items-center rounded-pill font-bold',
      toneClass[tone].soft,
      { sm: 'size-7 text-xs', md: 'size-9 text-sm', lg: 'size-14 text-md' }[size],
    ]"
    :aria-label="name"
  >{{ initials(name) }}</span>
</template>
