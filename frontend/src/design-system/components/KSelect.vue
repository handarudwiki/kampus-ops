<script setup lang="ts">
import { ChevronDown } from 'lucide-vue-next'
import { controlClass } from './controlClass'
defineOptions({ inheritAttrs: false })
const model = defineModel<string | number | null>()
withDefaults(defineProps<{ options: { value: string | number; label: string }[]; placeholder?: string; size?: 'sm' | 'md' }>(), { size: 'md' })
</script>

<template>
  <div class="relative">
    <select v-model="model" v-bind="$attrs" :class="[controlClass, 'appearance-none pr-9', size === 'sm' ? 'h-8 text-sm' : 'h-10']">
      <option v-if="placeholder !== undefined" value="">{{ placeholder }}</option>
      <option v-for="o in options" :key="o.value" :value="o.value">{{ o.label }}</option>
    </select>
    <ChevronDown class="pointer-events-none absolute right-3 top-1/2 size-4 -translate-y-1/2 text-ink-3" />
  </div>
</template>
