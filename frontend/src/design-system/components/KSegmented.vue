<script setup lang="ts" generic="T extends string | number">
/** Pill segmented control ("This month · Last month · Custom"). Use for 2–5 views or filters. */
const model = defineModel<T>({ required: true })
defineProps<{ options: { value: T; label: string; count?: number }[]; label?: string }>()
</script>

<template>
  <div class="inline-flex max-w-full overflow-x-auto rounded-pill bg-surface-3 p-1 ko-scroll" role="tablist" :aria-label="label">
    <button
      v-for="o in options"
      :key="o.value"
      type="button"
      role="tab"
      :aria-selected="model === o.value"
      :class="[
        'flex h-7 items-center gap-1.5 whitespace-nowrap rounded-pill px-3 text-sm font-semibold transition-all duration-ko',
        model === o.value ? 'bg-surface text-ink shadow-card' : 'text-ink-3 hover:text-ink',
      ]"
      @click="model = o.value"
    >
      {{ o.label }}
      <span v-if="o.count != null" class="tabular rounded-pill bg-surface-3 px-1.5 text-xs text-ink-2">{{ o.count }}</span>
    </button>
  </div>
</template>
