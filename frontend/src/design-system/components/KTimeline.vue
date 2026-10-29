<script setup lang="ts">
import { computed } from 'vue'
import { date } from '../format'

/** Horizontal schedule of phases with a "today" marker (semester calendar). */
const props = defineProps<{ phases: { key: string; label: string; start: string; end: string }[]; current?: string | null }>()
const day = (s: string) => new Date(s.slice(0, 10) + 'T00:00:00').getTime()
const today = day(new Date().toISOString())
const idx = computed(() => props.phases.findIndex((p) => p.key === props.current))
const state = (i: number, p: { end: string }) => (i === idx.value ? 'now' : day(p.end) < today ? 'done' : 'next')
</script>

<template>
  <ol class="ko-scroll -mx-1 flex gap-2 overflow-x-auto px-1 pb-1">
    <li
      v-for="(p, i) in phases"
      :key="p.key"
      :class="[
        'relative flex min-w-[128px] flex-1 flex-col gap-2 rounded-control p-3 transition-colors',
        state(i, p) === 'now' ? 'bg-ink text-ink-on' : 'bg-surface-2',
      ]"
    >
      <div class="flex items-center gap-2">
        <span
          :class="[
            'size-2 rounded-pill',
            state(i, p) === 'now' ? 'ko-live animate-pulse bg-green' : state(i, p) === 'done' ? 'bg-ink-3' : 'bg-line-strong',
          ]"
        />
        <span :class="['text-xs font-semibold uppercase tracking-[0.06em]', state(i, p) === 'now' ? 'opacity-70' : 'text-ink-3']">
          {{ state(i, p) === 'now' ? 'Sekarang' : state(i, p) === 'done' ? 'Selesai' : 'Berikutnya' }}
        </span>
      </div>
      <p class="font-bold" :class="state(i, p) === 'done' && 'text-ink-3'">{{ p.label }}</p>
      <p :class="['tabular text-xs', state(i, p) === 'now' ? 'opacity-70' : 'text-ink-3']">
        {{ date(p.start, { day: 'numeric', month: 'short' }) }}<template v-if="p.end !== p.start"> – {{ date(p.end, { day: 'numeric', month: 'short' }) }}</template>
      </p>
    </li>
  </ol>
</template>
