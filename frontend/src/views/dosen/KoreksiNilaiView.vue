<script setup lang="ts">
import { ref, watch } from 'vue'
import { ArrowRight, Check, X } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { KoreksiNilai } from '@/api/types'
import { useAction, useAsync } from '@/composables/useAsync'
import { confirm, relative } from '@/design-system'
import { useAuthStore } from '@/stores/auth'

/** EC-11: corrections after lock. Dosen sees their own requests; kaprodi decides. */
const auth = useAuthStore()
const isKaprodi = auth.activeRole === 'kaprodi'
const status = ref<string>('diajukan')
const { data, loading, reload } = useAsync<KoreksiNilai[]>(() => api.get<KoreksiNilai[]>('/koreksi-nilai', { status: status.value }), [])
watch(status, reload)
const { run } = useAction()

async function approve(k: KoreksiNilai) {
  const r = await confirm({ title: `Setujui koreksi ${k.mahasiswa_nama}?`, body: `${k.mk_nama}: ${k.huruf_lama} → ${k.huruf_baru}. IPS, IPK, dan transkrip dihitung ulang.`, confirmLabel: 'Setujui' })
  if (r.ok && (await run(() => api.post(`/koreksi-nilai/${k.id}/approve`), 'Koreksi disetujui'))) reload()
}
async function reject(k: KoreksiNilai) {
  const r = await confirm({ title: 'Tolak koreksi?', withReason: true, reasonLabel: 'Catatan untuk dosen', danger: true, confirmLabel: 'Tolak' })
  if (r.ok && (await run(() => api.post(`/koreksi-nilai/${k.id}/reject`, { catatan: r.reason }), 'Koreksi ditolak'))) reload()
}
</script>

<template>
  <div>
    <KPageHeader title="Koreksi nilai" :description="isKaprodi ? 'Permintaan koreksi nilai yang sudah dikunci di prodi Anda.' : 'Riwayat koreksi nilai yang Anda ajukan.'" />
    <KCard flush>
      <KToolbar>
        <KSegmented v-model="status" :options="[{ value: 'diajukan', label: 'Menunggu' }, { value: 'disetujui', label: 'Disetujui' }, { value: 'ditolak', label: 'Ditolak' }, { value: '', label: 'Semua' }]" />
      </KToolbar>
      <KTable
        :columns="[
          { key: 'mhs', label: 'Mahasiswa' },
          { key: 'mk', label: 'Kelas', hideMobile: true },
          { key: 'nilai', label: 'Perubahan', align: 'center' },
          { key: 'alasan', label: 'Alasan', hideMobile: true },
          { key: 'status', label: 'Status' },
          ...(isKaprodi ? [{ key: 'aksi', label: '', align: 'right' as const }] : []),
        ]"
        :rows="data"
        :loading="loading"
        empty-title="Tidak ada permintaan koreksi"
      >
        <template #cell-mhs="{ row }"><p class="font-semibold">{{ row.mahasiswa_nama }}</p><p class="font-mono text-sm text-ink-3">{{ row.nim }}</p></template>
        <template #cell-mk="{ row }"><p>{{ row.mk_nama }} · {{ row.kelas_nama }}</p><p class="text-sm text-ink-3">oleh {{ row.diajukan_oleh }} · {{ relative(row.created_at) }}</p></template>
        <template #cell-nilai="{ row }">
          <span class="inline-flex items-center gap-2 tabular">
            <span class="text-ink-3">{{ row.huruf_lama }} <span class="text-xs">({{ row.angka_lama }})</span></span>
            <ArrowRight class="size-3.5 text-ink-3" />
            <span class="font-bold">{{ row.huruf_baru }} <span class="text-xs font-normal text-ink-3">({{ row.angka_baru }})</span></span>
          </span>
        </template>
        <template #cell-alasan="{ value }"><p class="line-clamp-2 max-w-xs text-sm text-ink-2">{{ value }}</p></template>
        <template #cell-status="{ row }"><KStatus domain="approval" :value="row.status" variant="dot" /></template>
        <template #cell-aksi="{ row }">
          <div v-if="row.status === 'diajukan'" class="flex justify-end gap-1">
            <KIconButton label="Tolak" @click="reject(row)"><X class="size-4" /></KIconButton>
            <KButton size="sm" @click="approve(row)"><template #icon><Check class="size-4" /></template>Setujui</KButton>
          </div>
        </template>
      </KTable>
    </KCard>
  </div>
</template>
