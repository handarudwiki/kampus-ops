<script setup lang="ts">
const print = () => window.print()
import { ref, watch } from 'vue'
import { Printer } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Khs, Periode } from '@/api/types'
import { useAsync } from '@/composables/useAsync'
import { bobot, ip } from '@/design-system'

const periodes = useAsync<Periode[]>(() => api.get<Periode[]>('/akademik/me/periodes'), [])
const periodeId = ref<string | null>(null)
watch(periodes.data, (p) => { if (!periodeId.value && p.length) periodeId.value = p[0]!.id })

const khs = useAsync<Khs | null>(() => api.get<Khs>('/akademik/me/khs', { periode_id: periodeId.value }), null, { immediate: false })
watch(periodeId, (v) => v && khs.reload())

const cols = [
  { key: 'mk_kode', label: 'Kode', width: '100px' },
  { key: 'mk_nama', label: 'Mata kuliah' },
  { key: 'sks', label: 'SKS', align: 'right' as const },
  { key: 'huruf', label: 'Nilai', align: 'center' as const },
  { key: 'bobot_x100', label: 'Bobot', align: 'right' as const },
]
</script>

<template>
  <div>
    <KPageHeader title="Kartu Hasil Studi" description="Nilai final per semester. Mata kuliah ulang menampilkan nilai semester tersebut.">
      <template #actions>
        <KSelect v-model="periodeId" :options="periodes.data.value.map((p) => ({ value: p.id, label: `${p.tipe} ${p.tahun}/${p.tahun + 1}` }))" class="w-56" />
        <KButton variant="secondary" @click="print"><template #icon><Printer class="size-4" /></template>Cetak</KButton>
      </template>
    </KPageHeader>
    <div class="mb-3 grid gap-3 sm:grid-cols-3">
      <KStatCard label="IPS" :value="ip(khs.data.value?.ips)" tone="blue" />
      <KStatCard label="IPK" :value="ip(khs.data.value?.ipk)" tone="green" />
      <KStatCard label="SKS semester" :value="String(khs.data.value?.total_sks ?? '—')" tone="violet" />
    </div>
    <KCard flush>
      <KTable :columns="cols" :rows="khs.data.value?.rows ?? []" :row-key="(r) => r.mk_kode" :loading="khs.loading.value" empty-title="Belum ada nilai final">
        <template #cell-mk_kode="{ value }"><span class="font-mono text-sm">{{ value }}</span></template>
        <template #cell-mk_nama="{ row }">{{ row.mk_nama }} <KBadge v-if="row.is_ulang" tone="amber">Ulang</KBadge></template>
        <template #cell-huruf="{ value }"><span class="font-bold">{{ value ?? '—' }}</span></template>
        <template #cell-bobot_x100="{ value }"><span class="tabular">{{ bobot(value as unknown as number) }}</span></template>
      </KTable>
    </KCard>
  </div>
</template>
