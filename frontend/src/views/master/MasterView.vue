<script setup lang="ts">
import { RESOURCES, type ResourceKey } from './resources'
import ResourcePage from './ResourcePage.vue'
defineProps<{ resource: ResourceKey }>()
</script>

<template>
  <ResourcePage :key="resource" :config="RESOURCES[resource]" />
</template>
