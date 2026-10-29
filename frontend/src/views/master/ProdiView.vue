<script setup lang="ts">
import { ref } from 'vue'
import { RESOURCES } from './resources'
import ResourcePage from './ResourcePage.vue'
const tab = ref<'prodi' | 'fakultas'>('prodi')
</script>

<template>
  <div>
    <KSegmented v-model="tab" class="mb-4" :options="[{ value: 'prodi', label: 'Program studi' }, { value: 'fakultas', label: 'Fakultas' }]" />
    <ResourcePage :key="tab" :config="RESOURCES[tab]" />
  </div>
</template>
