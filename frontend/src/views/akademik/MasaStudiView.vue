<script setup lang="ts">
import { useRouter } from 'vue-router'
import { api } from '@/api/client'
import type { Mahasiswa } from '@/api/types'
import { useAsync } from '@/composables/useAsync'
import { ip } from '@/design-system'

/** EC-09: warn at 2 semesters left; exceeded = DO candidate (decision stays manual). */
const router = useRouter()
const { data, loading } = useAsync<(Mahasiswa & { sisa_semester: number })[]>(() => api.get('/akademik/masa-studi-warning'), [])
</script>

<template>
  <div>
    <KPageHeader title="Masa studi" description="Mahasiswa dengan sisa masa studi ≤ 2 semester. Yang sudah melewati batas menjadi kandidat DO; keputusan tetap manual." />
    <KCard flush>
      <KTable
        :columns="[{ key: 'mhs', label: 'Mahasiswa' }, { key: 'angkatan', label: 'Angkatan', align: 'right', hideMobile: true }, { key: 'semester_ke', label: 'Semester', align: 'right' }, { key: 'ipk', label: 'IPK', align: 'right', hideMobile: true }, { key: 'sisa', label: 'Sisa', align: 'right' }]"
        :rows="data"
        :loading="loading"
        clickable
        empty-title="Tidak ada peringatan"
        empty-description="Semua mahasiswa masih dalam batas masa studi."
        @row-click="(m) => router.push(`/mahasiswa/${m.id}`)"
      >
        <template #cell-mhs="{ row }"><p class="font-semibold">{{ row.nama }}</p><p class="font-mono text-sm text-ink-3">{{ row.nim }}</p></template>
        <template #cell-ipk="{ row }"><span class="tabular">{{ ip(row.ipk) }}</span></template>
        <template #cell-sisa="{ row }">
          <KBadge v-if="row.sisa_semester < 0" tone="red">Kandidat DO</KBadge>
          <KBadge v-else :tone="row.sisa_semester === 0 ? 'red' : 'amber'">{{ row.sisa_semester }} semester</KBadge>
        </template>
      </KTable>
    </KCard>
  </div>
</template>
