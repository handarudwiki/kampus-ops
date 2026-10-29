<script setup lang="ts">
/** Definition list for detail panels. items: [{ label, value }] or use slots #value-<key>. */
defineProps<{ items: { key?: string; label: string; value?: string | number | null }[]; cols?: 1 | 2 | 3 }>()
</script>

<template>
  <dl :class="['grid gap-x-6 gap-y-4', { 1: 'grid-cols-1', 2: 'grid-cols-1 sm:grid-cols-2', 3: 'grid-cols-1 sm:grid-cols-3' }[cols ?? 2]]">
    <div v-for="it in items" :key="it.key ?? it.label" class="min-w-0">
      <dt class="ko-eyebrow mb-1">{{ it.label }}</dt>
      <dd class="truncate font-semibold">
        <slot :name="`value-${it.key}`">{{ it.value ?? '—' }}</slot>
      </dd>
    </div>
  </dl>
</template>
