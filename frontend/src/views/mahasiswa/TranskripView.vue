<script setup lang="ts">
const print = () => window.print()
import { computed } from 'vue'
import { Printer } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Transkrip } from '@/api/types'
import { useAsync } from '@/composables/useAsync'
import { bobot, ip } from '@/design-system'

const { data, loading } = useAsync<Transkrip | null>(() => api.get<Transkrip>('/akademik/me/transkrip'), null)
const semesters = computed(() => {
  const m = new Map<number, Transkrip['rows']>()
  for (const r of data.value?.rows ?? []) m.set(r.semester, [...(m.get(r.semester) ?? []), r])
  return [...m.entries()].sort((a, b) => a[0] - b[0])
})
const cols = [
  { key: 'mk_kode', label: 'Kode', width: '100px' },
  { key: 'mk_nama', label: 'Mata kuliah' },
  { key: 'periode_kode', label: 'Periode', hideMobile: true },
  { key: 'sks', label: 'SKS', align: 'right' as const },
  { key: 'huruf', label: 'Nilai', align: 'center' as const },
  { key: 'bobot_x100', label: 'Bobot', align: 'right' as const },
]
</script>

<template>
  <div>
    <KPageHeader title="Transkrip akademik" description="Mata kuliah yang diulang memakai nilai terbaik.">
      <template #actions><KButton variant="secondary" @click="print"><template #icon><Printer class="size-4" /></template>Cetak</KButton></template>
    </KPageHeader>
    <KSkeleton v-if="loading && !data" :lines="6" />
    <template v-else-if="data">
      <div class="mb-3 grid gap-3 sm:grid-cols-3">
        <KStatCard label="IPK" :value="ip(data.ipk)" tone="green" />
        <KStatCard label="SKS lulus" :value="String(data.total_sks)" tone="blue" />
        <KStatCard label="Semester ke" :value="String(data.mahasiswa.semester_ke)" tone="violet" :hint="data.mahasiswa.prodi_nama" />
      </div>
      <div class="flex flex-col gap-3">
        <KCard v-for="[sem, rows] in semesters" :key="sem" :title="`Semester ${sem}`" :subtitle="`${rows.reduce((a, r) => a + r.sks, 0)} SKS`" flush>
          <KTable :columns="cols" :rows="rows" :row-key="(r) => r.mk_kode + r.periode_kode">
            <template #cell-mk_kode="{ value }"><span class="font-mono text-sm">{{ value }}</span></template>
            <template #cell-huruf="{ value }"><span class="font-bold">{{ value ?? '—' }}</span></template>
            <template #cell-bobot_x100="{ value }"><span class="tabular">{{ bobot(value as unknown as number) }}</span></template>
          </KTable>
        </KCard>
        <KCard v-if="!semesters.length"><KEmpty title="Transkrip masih kosong" /></KCard>
      </div>
    </template>
  </div>
</template>
