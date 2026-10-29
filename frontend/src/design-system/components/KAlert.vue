<script setup lang="ts">
import { computed } from 'vue'
import { AlertTriangle, CheckCircle2, Info, XCircle } from 'lucide-vue-next'
import { toneClass } from '../tone'
/** Inline banner. Tinted like the reference's AI-suggestion cards. */
const props = withDefaults(defineProps<{ level?: 'info' | 'success' | 'warning' | 'error'; title?: string }>(), { level: 'info' })
const map = {
  info: { tone: 'blue', icon: Info },
  success: { tone: 'green', icon: CheckCircle2 },
  warning: { tone: 'amber', icon: AlertTriangle },
  error: { tone: 'red', icon: XCircle },
} as const
const cfg = computed(() => map[props.level])
</script>

<template>
  <div :class="['flex gap-3 rounded-control p-4', toneClass[cfg.tone].tint]" :role="level === 'error' ? 'alert' : 'status'">
    <component :is="cfg.icon" :class="['mt-0.5 size-4 shrink-0', toneClass[cfg.tone].text]" />
    <div class="min-w-0 flex-1 text-sm text-ink-2">
      <p v-if="title" class="mb-0.5 font-bold text-ink">{{ title }}</p>
      <slot />
    </div>
    <div v-if="$slots.action" class="shrink-0 self-center"><slot name="action" /></div>
  </div>
</template>
