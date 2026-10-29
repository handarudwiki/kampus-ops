<script setup lang="ts">
import { computed } from 'vue'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'
import KIconButton from './KIconButton.vue'
import { num } from '../format'
const page = defineModel<number>({ required: true })
const props = defineProps<{ total: number; perPage: number }>()
const pages = computed(() => Math.max(1, Math.ceil(props.total / props.perPage)))
const from = computed(() => (props.total === 0 ? 0 : (page.value - 1) * props.perPage + 1))
const to = computed(() => Math.min(props.total, page.value * props.perPage))
</script>

<template>
  <div class="flex items-center justify-between gap-3 border-t border-line px-5 py-3 text-sm text-ink-3">
    <span class="tabular">{{ num(from) }}–{{ num(to) }} dari {{ num(total) }}</span>
    <div class="flex items-center gap-2">
      <KIconButton label="Sebelumnya" :class="page <= 1 && 'pointer-events-none opacity-40'" @click="page--"><ChevronLeft class="size-4" /></KIconButton>
      <span class="tabular font-semibold text-ink-2">{{ page }} / {{ pages }}</span>
      <KIconButton label="Berikutnya" :class="page >= pages && 'pointer-events-none opacity-40'" @click="page++"><ChevronRight class="size-4" /></KIconButton>
    </div>
  </div>
</template>
