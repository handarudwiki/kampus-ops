<script setup lang="ts">
import { controlClass } from './controlClass'
defineOptions({ inheritAttrs: false })
const model = defineModel<string | number | null>()
defineProps<{ invalid?: boolean }>()
</script>

<template>
  <div class="relative flex items-center">
    <span v-if="$slots.icon" class="pointer-events-none absolute left-3 text-ink-3"><slot name="icon" /></span>
    <input
      v-model="model"
      v-bind="$attrs"
      :class="[controlClass, 'h-10', $slots.icon && 'pl-9', invalid && '!border-red focus:!ring-red/15']"
      :aria-invalid="invalid || undefined"
    />
  </div>
</template>
