<script setup lang="ts">
/** Label + control + hint/error. Wrap every form control in this. */
defineProps<{ label?: string; hint?: string; error?: string | null; for?: string; required?: boolean }>()
</script>

<template>
  <div class="flex flex-col gap-1.5">
    <label v-if="label" :for="$props.for" class="text-sm font-semibold text-ink-2">
      {{ label }}<span v-if="required" class="text-red"> *</span>
    </label>
    <slot />
    <p v-if="error" class="text-xs font-medium text-red-ink">{{ error }}</p>
    <p v-else-if="hint" class="text-xs text-ink-3">{{ hint }}</p>
  </div>
</template>
