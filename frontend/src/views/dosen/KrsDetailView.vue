<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Check, X } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Krs } from '@/api/types'
import { useAction, useAsync } from '@/composables/useAsync'
import { confirm, dateTime, ip, slot } from '@/design-system'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const id = route.params.id as string
const { data: krs, loading, reload } = useAsync<Krs | null>(() => api.get<Krs>(`/krs/${id}`), null)
const { run, busy } = useAction()
const canDecide = computed(() => krs.value?.status === 'diajukan' && auth.activeRole === 'dosen_wali')

async function approve() {
  if (await run(() => api.post(`/krs/${id}/approve`, {}), 'KRS disetujui')) router.push('/perwalian')
}
async function reject() {
  const r = await confirm({ title: 'Tolak KRS?', body: 'Catatan dikirim ke mahasiswa agar KRS bisa diperbaiki.', withReason: true, reasonLabel: 'Catatan untuk mahasiswa', danger: true, confirmLabel: 'Tolak KRS' })
  if (r.ok && (await run(() => api.post(`/krs/${id}/reject`, { catatan: r.reason }), 'KRS ditolak'))) reload()
}
</script>

<template>
  <div>
    <KPageHeader :title="krs ? krs.mahasiswa.nama : 'Detail KRS'" :description="krs ? `${krs.mahasiswa.nim} · ${krs.mahasiswa.prodi_nama} · angkatan ${krs.mahasiswa.angkatan}` : ''">
      <template #eyebrow><KButton variant="ghost" size="sm" @click="router.back()"><template #icon><ArrowLeft class="size-4" /></template>Kembali</KButton></template>
      <template #actions>
        <template v-if="canDecide">
          <KButton variant="secondary" :disabled="busy" @click="reject"><template #icon><X class="size-4" /></template>Tolak</KButton>
          <KButton :loading="busy" @click="approve"><template #icon><Check class="size-4" /></template>Setujui</KButton>
        </template>
      </template>
    </KPageHeader>
    <KSkeleton v-if="loading && !krs" :lines="6" />
    <div v-else-if="krs" class="grid gap-3 lg:grid-cols-3">
      <div class="grid gap-3 sm:grid-cols-3 lg:col-span-3">
        <KStatCard label="Total SKS" :value="`${krs.total_sks}/${krs.max_sks}`" :tone="krs.total_sks > krs.max_sks ? 'red' : 'blue'" :hint="krs.total_sks > krs.max_sks ? 'Melebihi batas' : 'Sesuai batas IPS'" />
        <KStatCard label="IPS lalu" :value="ip(krs.ips_lalu)" tone="violet" />
        <KStatCard label="IPK" :value="ip(krs.mahasiswa.ipk)" tone="green" />
      </div>
      <KCard title="Kelas diambil" :subtitle="`${krs.items.length} kelas`" flush class="lg:col-span-2">
        <KTable
          :columns="[{ key: 'mk', label: 'Mata kuliah' }, { key: 'slot', label: 'Jadwal', hideMobile: true }, { key: 'sks', label: 'SKS', align: 'right' }]"
          :rows="krs.items"
          :row-key="(i) => i.kelas.id"
        >
          <template #cell-mk="{ row }">
            <p class="font-semibold">{{ row.kelas.mk_nama }} <KBadge v-if="row.is_ulang" tone="amber">Ulang</KBadge></p>
            <p class="text-sm text-ink-3"><span class="font-mono">{{ row.kelas.mk_kode }}</span> · Kelas {{ row.kelas.nama }} · {{ row.kelas.dosen_nama }}</p>
          </template>
          <template #cell-slot="{ row }"><span class="text-sm text-ink-2">{{ slot(row.kelas.slot) }}</span></template>
          <template #cell-sks="{ row }"><span class="tabular font-semibold">{{ row.kelas.sks }}</span></template>
        </KTable>
      </KCard>
      <KCard title="Status">
        <template #actions><KStatus domain="krs" :value="krs.status" /></template>
        <KKeyValue :cols="1" :items="[{ label: 'Diajukan', value: dateTime(krs.submitted_at, auth.tenant?.timezone) }, { label: 'Diputuskan', value: dateTime(krs.decided_at, auth.tenant?.timezone) }, { label: 'Catatan', value: krs.catatan }]" />
      </KCard>
    </div>
  </div>
</template>
