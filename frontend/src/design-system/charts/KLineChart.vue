<script setup lang="ts">
import { computed, ref } from 'vue'
import { SERIES_TONES, toneClass } from '../tone'
import { useWidth } from '../useWidth'
import KLegend from './KLegend.vue'

/** Smooth multi-series line chart with hover crosshair. */
const props = withDefaults(
  defineProps<{ labels: string[]; series: { name: string; values: number[] }[]; height?: number; format?: (n: number) => string }>(),
  { height: 200, format: (n: number) => n.toLocaleString('id-ID') },
)
const box = ref<HTMLElement | null>(null)
const width = useWidth(box)
const pad = { t: 12, b: 26, l: 8, r: 8 }
const all = computed(() => props.series.flatMap((s) => s.values))
const max = computed(() => Math.max(1, ...all.value))
const min = computed(() => Math.min(0, ...all.value))
const x = (i: number) => pad.l + (i * (width.value - pad.l - pad.r)) / Math.max(1, props.labels.length - 1)
const y = (v: number) => pad.t + (1 - (v - min.value) / (max.value - min.value || 1)) * (props.height - pad.t - pad.b)

function path(values: number[]) {
  const pts = values.map((v, i) => [x(i), y(v)] as const)
  if (!pts.length) return ''
  let d = `M${pts[0]![0]},${pts[0]![1]}`
  for (let i = 1; i < pts.length; i++) {
    const [x0, y0] = pts[i - 1]!
    const [x1, y1] = pts[i]!
    const cx = (x0 + x1) / 2
    d += ` C${cx},${y0} ${cx},${y1} ${x1},${y1}`
  }
  return d
}
const hover = ref<number | null>(null)
function move(e: MouseEvent) {
  const r = (e.currentTarget as SVGElement).getBoundingClientRect()
  const rel = ((e.clientX - r.left) / r.width) * width.value
  const step = (width.value - pad.l - pad.r) / Math.max(1, props.labels.length - 1)
  hover.value = Math.max(0, Math.min(props.labels.length - 1, Math.round((rel - pad.l) / step)))
}
const grid = [0, 0.25, 0.5, 0.75, 1]
</script>

<template>
  <div class="flex flex-col gap-3">
    <div ref="box" class="relative">
      <svg :viewBox="`0 0 ${width} ${height}`" class="w-full" :style="{ height: height + 'px' }" @mousemove="move" @mouseleave="hover = null">
        <line v-for="g in grid" :key="g" :x1="0" :x2="width" :y1="pad.t + g * (height - pad.t - pad.b)" :y2="pad.t + g * (height - pad.t - pad.b)" class="stroke-line" stroke-dasharray="3 4" stroke-width="1" />
        <line v-if="hover !== null" :x1="x(hover)" :x2="x(hover)" :y1="pad.t" :y2="height - pad.b" class="stroke-ink" stroke-dasharray="3 3" stroke-width="1" />
        <path v-for="(s, si) in series" :key="s.name" :d="path(s.values)" fill="none" :class="toneClass[SERIES_TONES[si % SERIES_TONES.length]!].stroke" stroke-width="3" stroke-linecap="round" vector-effect="non-scaling-stroke" />
      </svg>
      <div class="pointer-events-none absolute inset-x-0 bottom-0 flex justify-between px-1 text-xs text-ink-3">
        <span v-for="(l, i) in labels" :key="i">{{ l }}</span>
      </div>
      <div v-if="hover !== null" class="pointer-events-none absolute top-0 rounded-control border border-line bg-surface px-3 py-2 text-xs shadow-pop" :style="{ left: `min(calc(${(x(hover) / width) * 100}% + 8px), calc(100% - 160px))` }">
        <p class="font-bold">{{ labels[hover] }}</p>
        <p v-for="s in series" :key="s.name" class="tabular text-ink-2">{{ s.name }}: <b class="text-ink">{{ format(s.values[hover] ?? 0) }}</b></p>
      </div>
    </div>
    <KLegend :items="series.map((s, i) => ({ label: s.name, tone: SERIES_TONES[i % SERIES_TONES.length]! }))" />
  </div>
</template>
