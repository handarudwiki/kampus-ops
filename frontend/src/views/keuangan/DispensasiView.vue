<script setup lang="ts">
import { ref, watch } from 'vue'
import { Check, Plus, X } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Dispensasi, Mahasiswa } from '@/api/types'
import { useAction, useAsync } from '@/composables/useAsync'
import { confirm, date, statusOptions } from '@/design-system'
import { usePeriodeStore } from '@/stores/periode'

/** FR-27 / EC-15: dispensations unlock KRS despite unpaid bills; KRS susulan after the window. */
const periode = usePeriodeStore()
periode.load()
const status = ref('diajukan')
const { data, loading, reload } = useAsync<Dispensasi[]>(() => api.get<Dispensasi[]>('/dispensasi', { status: status.value }), [])
watch(status, reload)
const { run, busy } = useAction()

async function approve(d: Dispensasi) {
  if (await run(() => api.post(`/dispensasi/${d.id}/approve`), 'Dispensasi disetujui')) reload()
}
async function reject(d: Dispensasi) {
  const r = await confirm({ title: 'Tolak dispensasi?', withReason: true, danger: true, confirmLabel: 'Tolak' })
  if (r.ok && (await run(() => api.post(`/dispensasi/${d.id}/reject`, { catatan: r.reason }), 'Dispensasi ditolak'))) reload()
}

// Create
const open = ref(false)
const nim = ref('')
const found = ref<Mahasiswa | null>(null)
const form = ref({ jenis: 'cicilan', alasan: '' })
let t: ReturnType<typeof setTimeout>
watch(nim, (v) => {
  clearTimeout(t)
  found.value = null
  if (v.length < 4) return
  t = setTimeout(async () => {
    const r = await api.list<Mahasiswa>('/mahasiswa', { q: v, per_page: 1 }).catch(() => ({ data: [] }))
    found.value = r.data[0] ?? null
  }, 300)
})
async function create() {
  const ok = await run(() => api.post('/dispensasi', { mahasiswa_id: found.value!.id, periode_id: periode.active?.id, ...form.value }), 'Dispensasi diajukan')
  if (ok) { open.value = false; nim.value = ''; form.value = { jenis: 'cicilan', alasan: '' }; reload() }
}
</script>

<template>
  <div>
    <KPageHeader title="Dispensasi" description="Dispensasi yang disetujui membuka KRS walau tagihan belum lunas, atau memberi KRS susulan. Semua tercatat di audit log.">
      <template #actions><KButton @click="open = true"><template #icon><Plus class="size-4" /></template>Ajukan dispensasi</KButton></template>
    </KPageHeader>
    <KCard flush>
      <KToolbar><KSegmented v-model="status" :options="[...statusOptions('approval'), { value: '', label: 'Semua' }]" /></KToolbar>
      <KTable
        :columns="[{ key: 'mhs', label: 'Mahasiswa' }, { key: 'jenis', label: 'Jenis' }, { key: 'alasan', label: 'Alasan', hideMobile: true }, { key: 'status', label: 'Status', hideMobile: true }, { key: 'aksi', label: '', align: 'right' }]"
        :rows="data"
        :loading="loading"
        empty-title="Tidak ada dispensasi"
      >
        <template #cell-mhs="{ row }"><p class="font-semibold">{{ row.nama }}</p><p class="text-sm text-ink-3"><span class="font-mono">{{ row.nim }}</span> · {{ row.periode_kode }} · {{ date(row.created_at) }}</p></template>
        <template #cell-jenis="{ row }"><KStatus domain="jenisDispensasi" :value="row.jenis" /></template>
        <template #cell-alasan="{ value }"><p class="line-clamp-2 max-w-xs text-sm text-ink-2">{{ value }}</p></template>
        <template #cell-status="{ row }"><KStatus domain="approval" :value="row.status" variant="dot" /><p v-if="row.disetujui_oleh" class="text-xs text-ink-3">oleh {{ row.disetujui_oleh }}</p></template>
        <template #cell-aksi="{ row }">
          <div v-if="row.status === 'diajukan'" class="flex justify-end gap-1">
            <KIconButton label="Tolak" @click="reject(row)"><X class="size-4" /></KIconButton>
            <KButton size="sm" @click="approve(row)"><template #icon><Check class="size-4" /></template>Setujui</KButton>
          </div>
        </template>
      </KTable>
    </KCard>

    <KModal v-model="open" title="Ajukan dispensasi" size="sm">
      <div class="flex flex-col gap-4">
        <KField label="NIM mahasiswa" required :hint="found ? `${found.nama} · ${found.prodi_nama}` : 'Ketik minimal 4 karakter'"><KInput v-model="nim" /></KField>
        <KField label="Jenis" required><KSelect v-model="form.jenis" :options="statusOptions('jenisDispensasi')" /></KField>
        <KField label="Alasan" required><KTextarea v-model="form.alasan" /></KField>
      </div>
      <template #footer>
        <KButton variant="secondary" @click="open = false">Batal</KButton>
        <KButton :loading="busy" :disabled="!found || !form.alasan.trim()" @click="create">Ajukan</KButton>
      </template>
    </KModal>
  </div>
</template>
