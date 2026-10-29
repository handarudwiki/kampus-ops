<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { Loader2 } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger' | 'soft'
    size?: 'sm' | 'md'
    loading?: boolean
    disabled?: boolean
    to?: string
    type?: 'button' | 'submit'
    block?: boolean
  }>(),
  { variant: 'primary', size: 'md', type: 'button' },
)

const cls = computed(() => [
  'inline-flex items-center justify-center gap-2 rounded-pill font-semibold whitespace-nowrap select-none',
  'transition-colors duration-ko ease-ko disabled:cursor-not-allowed disabled:opacity-50',
  props.size === 'sm' ? 'h-8 px-3 text-sm' : 'h-10 px-4 text-base',
  props.block && 'w-full',
  {
    primary: 'bg-ink text-ink-on hover:bg-ink/85',
    secondary: 'bg-surface text-ink border border-line hover:bg-surface-2',
    ghost: 'text-ink-2 hover:bg-surface-2 hover:text-ink',
    danger: 'bg-red text-white hover:bg-red/90',
    soft: 'bg-accent-soft text-accent hover:bg-accent/15',
  }[props.variant],
])
</script>

<template>
  <RouterLink v-if="to" :to="to" :class="cls"><slot name="icon" /><slot /></RouterLink>
  <button v-else :type="type" :class="cls" :disabled="disabled || loading">
    <Loader2 v-if="loading" class="size-4 animate-spin" />
    <slot v-else name="icon" />
    <slot />
  </button>
</template>
