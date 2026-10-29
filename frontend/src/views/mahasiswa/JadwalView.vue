<script setup lang="ts">
import { computed } from 'vue'
import { api } from '@/api/client'
import type { JadwalItem } from '@/api/types'
import { useAsync } from '@/composables/useAsync'
import { HARI, toneClass, toneFor } from '@/design-system'

const { data, loading } = useAsync<JadwalItem[]>(() => api.get<JadwalItem[]>('/akademik/me/jadwal'), [])
const days = [1, 2, 3, 4, 5, 6] as const
const byDay = computed(() =>
  days.map((d) => ({
    hari: d,
    items: data.value.filter((j) => j.kelas.slot.hari === d).sort((a, b) => a.kelas.slot.jam_mulai.localeCompare(b.kelas.slot.jam_mulai)),
  })),
)
const totalSks = computed(() => data.value.reduce((a, j) => a + j.kelas.sks, 0))
</script>

<template>
  <div>
    <KPageHeader title="Jadwal kuliah" :description="`${data.length} kelas · ${totalSks} SKS dari KRS yang disetujui`" />
    <div v-if="loading" class="h-80 animate-pulse rounded-card bg-surface" />
    <KCard v-else-if="!data.length"><KEmpty title="Belum ada jadwal" description="Jadwal muncul setelah KRS disetujui dosen wali."><KButton to="/krs" size="sm">Buka KRS</KButton></KEmpty></KCard>
    <div v-else class="grid gap-3 sm:grid-cols-2 xl:grid-cols-6">
      <KCard v-for="d in byDay" :key="d.hari" :title="HARI[d.hari]" :subtitle="d.items.length ? `${d.items.length} kelas` : 'Libur'">
        <ul class="flex flex-col gap-2">
          <li v-for="j in d.items" :key="j.kelas.id" :class="['rounded-control p-3', toneClass[toneFor(j.kelas.mk_kode)].tint]">
            <p class="tabular text-xs font-bold text-ink-2">{{ j.kelas.slot.jam_mulai }}–{{ j.kelas.slot.jam_selesai }}</p>
            <p class="mt-1 font-bold leading-snug">{{ j.kelas.mk_nama }}</p>
            <p class="mt-1 text-xs text-ink-2">{{ j.kelas.ruang_nama ?? '—' }} · {{ j.kelas.dosen_nama ?? '—' }}</p>
          </li>
        </ul>
      </KCard>
    </div>
  </div>
</template>
