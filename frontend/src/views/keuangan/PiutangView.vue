<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { api } from '@/api/client'
import type { Piutang } from '@/api/types'
import { useAsync } from '@/composables/useAsync'
import { rupiah, rupiahShort } from '@/design-system'

/** FR-28 receivables per prodi / cohort. */
const group = ref<'prodi' | 'angkatan'>('prodi')
const { data, loading, reload } = useAsync<Piutang[]>(() => api.get<Piutang[]>('/keuangan/piutang', { group: group.value }), [])
watch(group, reload)
const total = computed(() => data.value.reduce((a, r) => ({ t: a.t + r.total_tagihan, p: a.p + r.piutang }), { t: 0, p: 0 }))
</script>

<template>
  <div>
    <KPageHeader title="Piutang" description="Sisa tagihan yang belum dibayar.">
      <template #actions><KSegmented v-model="group" :options="[{ value: 'prodi', label: 'Per prodi' }, { value: 'angkatan', label: 'Per angkatan' }]" /></template>
    </KPageHeader>
    <div class="mb-3 grid gap-3 sm:grid-cols-2">
      <KStatCard label="Total piutang" :value="rupiahShort(total.p)" tone="red" :hint="total.t ? `${Math.round((total.p / total.t) * 100)}% dari total tagihan` : null" trend="down" />
      <KStatCard label="Total tagihan" :value="rupiahShort(total.t)" tone="blue" />
    </div>
    <div class="grid gap-3 xl:grid-cols-5">
      <KCard title="Tertagih vs piutang" class="xl:col-span-3">
        <KBarChart :labels="data.map((r) => r.group)" :series="[{ name: 'Terbayar', values: data.map((r) => r.total_terbayar) }, { name: 'Piutang', values: data.map((r) => r.piutang) }]" :format="rupiah" :height="240" />
      </KCard>
      <KCard flush class="xl:col-span-2">
        <KTable :columns="[{ key: 'group', label: group === 'prodi' ? 'Prodi' : 'Angkatan' }, { key: 'piutang', label: 'Piutang', align: 'right' }]" :rows="data" :row-key="(r) => r.group" :loading="loading">
          <template #cell-piutang="{ row }">
            <p class="tabular font-semibold">{{ rupiah(row.piutang) }}</p>
            <KProgress class="mt-1" :value="row.piutang" :max="row.total_tagihan" tone="red" />
          </template>
        </KTable>
      </KCard>
    </div>
  </div>
</template>
