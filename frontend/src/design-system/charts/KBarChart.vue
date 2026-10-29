<script setup lang="ts">
import { computed, ref } from 'vue'
import { SERIES_TONES, toneClass } from '../tone'
import KLegend from './KLegend.vue'

/** Rounded vertical bars. Multiple series render grouped. Values are plain numbers. */
const props = withDefaults(
  defineProps<{ labels: string[]; series: { name: string; values: number[] }[]; height?: number; format?: (n: number) => string }>(),
  { height: 200, format: (n: number) => n.toLocaleString('id-ID') },
)
const W = 600
const pad = { t: 8, b: 26, l: 4, r: 4 }
const max = computed(() => Math.max(1, ...props.series.flatMap((s) => s.values)))
const groupW = computed(() => (W - pad.l - pad.r) / Math.max(1, props.labels.length))
const barW = computed(() => Math.max(3, Math.min(22, (groupW.value * 0.7) / props.series.length)))
const hover = ref<number | null>(null)
const innerH = computed(() => props.height - pad.t - pad.b)
function bar(si: number, i: number) {
  const v = props.series[si]!.values[i] ?? 0
  const h = Math.max(2, (v / max.value) * innerH.value)
  const gx = pad.l + i * groupW.value + groupW.value / 2 - (barW.value * props.series.length + 2 * (props.series.length - 1)) / 2
  return { x: gx + si * (barW.value + 2), y: pad.t + innerH.value - h, h }
}
const every = computed(() => Math.ceil(props.labels.length / 8))
</script>

<template>
  <div class="flex flex-col gap-3">
    <div class="relative">
      <svg :viewBox="`0 0 ${W} ${height}`" class="w-full" :style="{ height: height + 'px' }" preserveAspectRatio="none" role="img" :aria-label="series.map((s) => s.name).join(', ')">
        <line :x1="0" :x2="W" :y1="pad.t + innerH" :y2="pad.t + innerH" class="stroke-line" stroke-width="1" />
        <g v-for="(_, i) in labels" :key="i" @mouseenter="hover = i" @mouseleave="hover = null">
          <rect :x="pad.l + i * groupW" :y="0" :width="groupW" :height="height" fill="transparent" />
          <rect
            v-for="(s, si) in series"
            :key="s.name"
            v-bind="{ x: bar(si, i).x, y: bar(si, i).y, height: bar(si, i).h }"
            :width="barW"
            :rx="Math.min(4, barW / 2)"
            :class="[toneClass[SERIES_TONES[si % SERIES_TONES.length]!].fill, 'transition-opacity duration-ko']"
            :opacity="hover === null || hover === i ? 1 : 0.35"
          />
        </g>
      </svg>
      <div class="pointer-events-none absolute inset-x-0 bottom-0 flex text-xs text-ink-3" :style="{ paddingLeft: '0.6%', paddingRight: '0.6%' }">
        <span v-for="(l, i) in labels" :key="i" class="flex-1 truncate text-center">{{ i % every === 0 ? l : '' }}</span>
      </div>
      <div v-if="hover !== null" class="pointer-events-none absolute left-1/2 top-0 -translate-x-1/2 rounded-control border border-line bg-surface px-3 py-2 text-xs shadow-pop">
        <p class="font-bold">{{ labels[hover] }}</p>
        <p v-for="s in series" :key="s.name" class="tabular text-ink-2">{{ s.name }}: <b class="text-ink">{{ format(s.values[hover] ?? 0) }}</b></p>
      </div>
    </div>
    <KLegend v-if="series.length > 1" :items="series.map((s, i) => ({ label: s.name, tone: SERIES_TONES[i % SERIES_TONES.length]! }))" />
  </div>
</template>
