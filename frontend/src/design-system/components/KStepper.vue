<script setup lang="ts">
import { computed } from 'vue'
import { Check, X } from 'lucide-vue-next'
/** Horizontal status flow (PMB pipeline, KRS lifecycle). */
const props = defineProps<{ steps: { key: string; label: string }[]; current: string; failed?: boolean }>()
const idx = computed(() => props.steps.findIndex((s) => s.key === props.current))
</script>

<template>
  <ol class="ko-scroll flex items-start gap-1 overflow-x-auto pb-1">
    <li v-for="(s, i) in steps" :key="s.key" class="flex min-w-[88px] flex-1 flex-col items-center gap-2 text-center">
      <div class="flex w-full items-center">
        <div :class="['h-0.5 flex-1 rounded-pill', i === 0 ? 'bg-transparent' : i <= idx ? 'bg-ink' : 'bg-line']" />
        <span
          :class="[
            'grid size-7 shrink-0 place-items-center rounded-pill text-xs font-bold',
            i < idx ? 'bg-ink text-ink-on' : i === idx ? (failed ? 'bg-red text-white' : 'bg-accent text-white ring-4 ring-accent/20') : 'bg-surface-3 text-ink-3',
          ]"
        >
          <Check v-if="i < idx" class="size-3.5" />
          <X v-else-if="i === idx && failed" class="size-3.5" />
          <template v-else>{{ i + 1 }}</template>
        </span>
        <div :class="['h-0.5 flex-1 rounded-pill', i === steps.length - 1 ? 'bg-transparent' : i < idx ? 'bg-ink' : 'bg-line']" />
      </div>
      <span :class="['text-xs font-semibold', i <= idx ? 'text-ink' : 'text-ink-3']">{{ s.label }}</span>
    </li>
  </ol>
</template>
