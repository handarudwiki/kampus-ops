<script setup lang="ts">
import { ref, watch } from 'vue'
import { Search } from 'lucide-vue-next'
import KInput from './KInput.vue'
/** Debounced search box. v-model updates 250ms after typing stops. */
const model = defineModel<string>({ default: '' })
defineProps<{ placeholder?: string }>()
const local = ref(model.value)
let t: ReturnType<typeof setTimeout>
watch(local, (v) => { clearTimeout(t); t = setTimeout(() => (model.value = String(v ?? '')), 250) })
</script>

<template>
  <KInput v-model="local" type="search" :placeholder="placeholder ?? 'Cari…'" class="w-full sm:w-64">
    <template #icon><Search class="size-4" /></template>
  </KInput>
</template>
