<script setup lang="ts" generic="T">
import KEmpty from './KEmpty.vue'
import KSkeleton from './KSkeleton.vue'

export interface Column {
  key: string
  label: string
  align?: 'left' | 'right' | 'center'
  width?: string
  /** hide below the sm breakpoint (360px mobile requirement) */
  hideMobile?: boolean
}

/**
 * Data table. Header in small caps eyebrow style, rows divided by hairlines.
 * Custom cells: <template #cell-status="{ row }">…</template>
 */
const props = withDefaults(
  defineProps<{
    columns: Column[]
    rows: T[]
    rowKey?: (row: T) => string
    loading?: boolean
    emptyTitle?: string
    emptyDescription?: string
    clickable?: boolean
  }>(),
  { rowKey: (r: T) => (r as { id: string }).id },
)
const emit = defineEmits<{ rowClick: [row: T] }>()
const align = (c: Column) => (c.align === 'right' ? 'text-right' : c.align === 'center' ? 'text-center' : 'text-left')
const get = (row: T, key: string) => (row as Record<string, unknown>)[key] as string
function click(row: T) {
  if (props.clickable) emit('rowClick', row)
}
</script>

<template>
  <div class="ko-scroll overflow-x-auto">
    <table class="w-full border-collapse text-base">
      <thead>
        <tr class="border-b border-line">
          <th
            v-for="c in columns"
            :key="c.key"
            scope="col"
            :style="{ width: c.width }"
            :class="['ko-eyebrow whitespace-nowrap px-5 py-3', align(c), c.hideMobile && 'hidden sm:table-cell']"
          >{{ c.label }}</th>
        </tr>
      </thead>
      <tbody>
        <template v-if="loading">
          <tr v-for="i in 5" :key="i" class="border-b border-line last:border-0">
            <td v-for="c in columns" :key="c.key" :class="['px-5 py-4', c.hideMobile && 'hidden sm:table-cell']"><KSkeleton /></td>
          </tr>
        </template>
        <tr
          v-for="row in rows"
          v-else
          :key="rowKey(row)"
          :class="['border-b border-line transition-colors duration-ko last:border-0', clickable && 'cursor-pointer hover:bg-surface-2']"
          :tabindex="clickable ? 0 : undefined"
          @click="click(row)"
          @keydown.enter="click(row)"
        >
          <td v-for="c in columns" :key="c.key" :class="['px-5 py-3.5 align-middle', align(c), c.hideMobile && 'hidden sm:table-cell']">
            <slot :name="`cell-${c.key}`" :row="row" :value="get(row, c.key)">{{ get(row, c.key) ?? '—' }}</slot>
          </td>
        </tr>
      </tbody>
    </table>
    <KEmpty v-if="!loading && rows.length === 0" :title="emptyTitle" :description="emptyDescription"><slot name="empty" /></KEmpty>
  </div>
</template>
