<script setup lang="ts">
import { controlClass } from './controlClass'
const model = defineModel<string | null>()
</script>

<template>
  <textarea v-model="model" rows="3" :class="[controlClass, 'py-2.5']" />
</template>
