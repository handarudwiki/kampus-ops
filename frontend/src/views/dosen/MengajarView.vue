<script setup lang="ts">
import { BookOpen, Users } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Kelas } from '@/api/types'
import { useAsync } from '@/composables/useAsync'
import { slot, toneClass, toneFor } from '@/design-system'

const { data, loading } = useAsync<Kelas[]>(() => api.get<Kelas[]>('/teaching/kelas'), [])
</script>

<template>
  <div>
    <KPageHeader title="Kelas saya" description="Kelas yang Anda ampu di periode aktif. Buka kelas untuk presensi dan nilai." />
    <div v-if="loading" class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3"><div v-for="i in 3" :key="i" class="h-44 animate-pulse rounded-card bg-surface" /></div>
    <KCard v-else-if="!data.length"><KEmpty title="Belum ada kelas" description="Hubungi kaprodi untuk plotting dosen." /></KCard>
    <div v-else class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      <RouterLink v-for="k in data" :key="k.id" :to="`/kelas/${k.id}`" class="group flex flex-col gap-4 rounded-card bg-surface p-5 shadow-card transition-colors hover:bg-surface-2/60">
        <div class="flex items-start justify-between gap-3">
          <span :class="['grid size-10 place-items-center rounded-control', toneClass[toneFor(k.mk_kode)].soft]"><BookOpen class="size-5" /></span>
          <KStatus domain="kelas" :value="k.status" />
        </div>
        <div>
          <p class="text-md font-bold">{{ k.mk_nama }} · {{ k.nama }}</p>
          <p class="text-sm text-ink-3"><span class="font-mono">{{ k.mk_kode }}</span> · {{ k.sks }} SKS</p>
        </div>
        <div class="flex items-center justify-between text-sm text-ink-2">
          <span>{{ slot(k.slot) }} · {{ k.ruang_nama }}</span>
          <span class="inline-flex items-center gap-1 tabular font-semibold"><Users class="size-4 text-ink-3" />{{ k.terisi }}</span>
        </div>
      </RouterLink>
    </div>
  </div>
</template>
