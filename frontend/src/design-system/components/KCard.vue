<script setup lang="ts">
import { ArrowUpRight } from 'lucide-vue-next'
import KIconButton from './KIconButton.vue'
/** The one container. Title row + optional "↗" link, like the reference dashboard. */
withDefaults(defineProps<{ title?: string; subtitle?: string; to?: string; flush?: boolean; tone?: 'default' | 'inset' }>(), {
  tone: 'default',
})
</script>

<template>
  <section :class="['rounded-card shadow-card', tone === 'inset' ? 'bg-surface-2' : 'bg-surface']">
    <header v-if="title || $slots.actions || to" class="flex items-start justify-between gap-3 px-5 pt-5" :class="flush ? 'pb-4' : ''">
      <div class="min-w-0">
        <h2 v-if="title" class="truncate text-md font-bold">{{ title }}</h2>
        <p v-if="subtitle" class="mt-0.5 text-sm text-ink-3">{{ subtitle }}</p>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        <slot name="actions" />
        <KIconButton v-if="to" :to="to" label="Buka detail"><ArrowUpRight class="size-4" /></KIconButton>
      </div>
    </header>
    <div :class="flush ? '' : 'p-5'"><slot /></div>
  </section>
</template>
