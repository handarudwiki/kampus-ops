<script setup lang="ts">
import { computed } from 'vue'
import { statusOf, type StatusDomain } from '../status'
import KBadge from './KBadge.vue'
/** The only way to render a domain status. See design-system/status.ts. */
const props = defineProps<{ domain: StatusDomain; value: string | null | undefined; variant?: 'pill' | 'dot' }>()
const def = computed(() => statusOf(props.domain, props.value))
</script>

<template>
  <KBadge :tone="def.tone" :variant="variant ?? 'pill'">{{ def.label }}</KBadge>
</template>
