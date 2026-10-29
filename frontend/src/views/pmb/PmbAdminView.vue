<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { BadgeCheck, MoreHorizontal, Plus } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Gelombang, Pendaftar, PmbStatus } from '@/api/types'
import { useAction, useAsync } from '@/composables/useAsync'
import { confirm, date, rupiah, STATUS, toast } from '@/design-system'
import { PMB_NEXT } from './steps'

/** FR-08..10. Status pipeline + daftar ulang that issues NIM (EC-04 sequence on the server). */
const tab = ref<'pendaftar' | 'gelombang'>('pendaftar')
const status = ref<PmbStatus | ''>('')
const q = ref('')
const pendaftar = useAsync<Pendaftar[]>(() => api.get<Pendaftar[]>('/pmb/pendaftar', { status: status.value, q: q.value }), [])
const all = useAsync<Pendaftar[]>(() => api.get<Pendaftar[]>('/pmb/pendaftar'), [], { silent: true })
const gelombang = useAsync<Gelombang[]>(() => api.get<Gelombang[]>('/pmb/gelombang'), [])
watch([status, q], () => pendaftar.reload())

const counts = computed(() => {
  const c: Record<string, number> = {}
  for (const p of all.data.value) c[p.status] = (c[p.status] ?? 0) + 1
  return c
})
const statusTabs = computed(() => [
  { value: '' as const, label: 'Semua', count: all.data.value.length },
  ...(Object.keys(STATUS.pmb) as PmbStatus[]).map((k) => ({ value: k, label: STATUS.pmb[k].label, count: counts.value[k] ?? 0 })),
])

const { run, busy } = useAction()
function refresh() { pendaftar.reload(); all.reload() }
async function transition(p: Pendaftar, to: PmbStatus) {
  if (await run(() => api.post(`/pmb/pendaftar/${p.id}/transition`, { to }), `${p.nama}: ${STATUS.pmb[to].label}`)) refresh()
}
async function daftarUlang(p: Pendaftar) {
  const r = await confirm({ title: `Daftar ulang ${p.nama}?`, body: `Menerbitkan NIM untuk ${p.prodi_pilihan_nama} dan membuat akun mahasiswa aktif.`, confirmLabel: 'Terbitkan NIM' })
  if (!r.ok) return
  await run(async () => {
    const res = await api.post<Pendaftar>(`/pmb/pendaftar/${p.id}/daftar-ulang`)
    toast('success', 'NIM diterbitkan', `${res.nama}: ${res.nim}`)
    refresh()
  })
}
function menu(p: Pendaftar) {
  const items = PMB_NEXT[p.status].map((to) => ({ label: `Ubah ke ${STATUS.pmb[to].label}`, onClick: () => transition(p, to) }))
  if (p.status === 'lulus') items.push({ label: 'Daftar ulang & terbitkan NIM', onClick: () => daftarUlang(p) })
  return items
}

// Gelombang form
const gOpen = ref(false)
const g = ref({ nama: '', mulai: '', selesai: '', kuota: 100, biaya_formulir: 300000 })
async function saveGelombang() {
  if (await run(() => api.post('/pmb/gelombang', g.value), 'Gelombang dibuat')) { gOpen.value = false; gelombang.reload() }
}
</script>

<template>
  <div>
    <KPageHeader title="Penerimaan mahasiswa baru" description="Daftar → bayar formulir → lengkapi berkas → verifikasi → lulus seleksi → daftar ulang (NIM terbit).">
      <template #actions>
        <KSegmented v-model="tab" :options="[{ value: 'pendaftar', label: 'Pendaftar' }, { value: 'gelombang', label: 'Gelombang' }]" />
        <KButton v-if="tab === 'gelombang'" @click="gOpen = true"><template #icon><Plus class="size-4" /></template>Tambah gelombang</KButton>
      </template>
    </KPageHeader>

    <template v-if="tab === 'pendaftar'">
      <KCard flush>
        <KToolbar>
          <KSegmented v-model="status" :options="statusTabs" label="Filter status" />
          <template #right><KSearch v-model="q" placeholder="Cari nama / no. daftar" /></template>
        </KToolbar>
        <KTable
          :columns="[
            { key: 'p', label: 'Pendaftar' },
            { key: 'prodi', label: 'Prodi pilihan', hideMobile: true },
            { key: 'berkas', label: 'Berkas', hideMobile: true },
            { key: 'status', label: 'Status' },
            { key: 'menu', label: '', align: 'right', width: '56px' },
          ]"
          :rows="pendaftar.data.value"
          :loading="pendaftar.loading.value"
          empty-title="Tidak ada pendaftar"
        >
          <template #cell-p="{ row }"><p class="font-semibold">{{ row.nama }}</p><p class="text-sm text-ink-3"><span class="font-mono">{{ row.no_daftar }}</span> · {{ row.gelombang_nama }}</p></template>
          <template #cell-prodi="{ row }">{{ row.prodi_pilihan_nama }}</template>
          <template #cell-berkas="{ row }">
            <span class="tabular text-sm">{{ row.berkas.filter((b) => b.ok).length }}/{{ row.berkas.length }}</span>
          </template>
          <template #cell-status="{ row }">
            <KStatus domain="pmb" :value="row.status" />
            <p v-if="row.nim" class="mt-1 inline-flex items-center gap-1 font-mono text-sm text-green-ink"><BadgeCheck class="size-3.5" />{{ row.nim }}</p>
          </template>
          <template #cell-menu="{ row }">
            <KMenu v-if="menu(row).length" :items="menu(row)"><KIconButton label="Aksi" :class="busy && 'opacity-50'"><MoreHorizontal class="size-4" /></KIconButton></KMenu>
          </template>
        </KTable>
      </KCard>
    </template>

    <div v-else class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      <KCard v-for="x in gelombang.data.value" :key="x.id" :title="x.nama" :subtitle="`${date(x.mulai)} – ${date(x.selesai)}`">
        <div class="flex flex-col gap-3">
          <div class="flex items-end justify-between">
            <p class="tabular text-xl font-bold">{{ x.terisi }}<span class="text-md text-ink-3">/{{ x.kuota }}</span></p>
            <KBadge tone="violet">Formulir {{ rupiah(x.biaya_formulir) }}</KBadge>
          </div>
          <KProgress :value="x.terisi" :max="x.kuota" :tone="x.terisi >= x.kuota ? 'red' : 'blue'" />
        </div>
      </KCard>
      <KCard v-if="!gelombang.data.value.length" class="sm:col-span-2 xl:col-span-3"><KEmpty title="Belum ada gelombang" /></KCard>
    </div>

    <KModal v-model="gOpen" title="Tambah gelombang" size="md">
      <div class="grid gap-4 sm:grid-cols-2">
        <KField label="Nama" required class="sm:col-span-2"><KInput v-model="g.nama" placeholder="Gelombang 1 2027" /></KField>
        <KField label="Mulai" required><KInput v-model="g.mulai" type="date" /></KField>
        <KField label="Selesai" required><KInput v-model="g.selesai" type="date" /></KField>
        <KField label="Kuota" required><KInput v-model.number="g.kuota" type="number" /></KField>
        <KField label="Biaya formulir (Rp)" required><KInput v-model.number="g.biaya_formulir" type="number" /></KField>
      </div>
      <template #footer>
        <KButton variant="secondary" @click="gOpen = false">Batal</KButton>
        <KButton :loading="busy" :disabled="!g.nama || !g.mulai || !g.selesai" @click="saveGelombang">Simpan</KButton>
      </template>
    </KModal>
  </div>
</template>
