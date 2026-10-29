<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Ban, MoreHorizontal, Pencil, Plus, UserRoundCog } from 'lucide-vue-next'
import { api, ApiError } from '@/api/client'
import type { Dosen, Kelas, Kurikulum, MataKuliah, Prodi, Ruang, Slot } from '@/api/types'
import { useAction } from '@/composables/useAsync'
import { confirm, errorText, HARI, slot, toast, toastError } from '@/design-system'
import { usePeriodeStore } from '@/stores/periode'

/** FR-14/15 schedule with clash detection, EC-12 lecturer change, EC-13 cancel cascade. */
const router = useRouter()
const periode = usePeriodeStore()
const prodi = ref<Prodi[]>([])
const dosen = ref<Dosen[]>([])
const ruang = ref<Ruang[]>([])
const prodiId = ref('')
const q = ref('')
const rows = ref<Kelas[]>([])
const loading = ref(false)

async function load() {
  if (!periode.active) await periode.load()
  loading.value = true
  try { rows.value = await api.get<Kelas[]>('/kelas', { periode_id: periode.active?.id, prodi_id: prodiId.value, q: q.value }) } catch (e) { toastError(e) } finally { loading.value = false }
}
onMounted(async () => {
  load()
  const [p, d, r] = await Promise.all([
    api.list<Prodi>('/prodi', { per_page: 100 }), api.list<Dosen>('/dosen', { per_page: 100 }), api.list<Ruang>('/ruang', { per_page: 100 }),
  ]).catch(() => [{ data: [] }, { data: [] }, { data: [] }] as never)
  prodi.value = p.data; dosen.value = d.data; ruang.value = r.data
})
watch([prodiId, q], () => { page.value = 1; load() })
const page = ref(1)
const PER = 20
const paged = computed(() => rows.value.slice((page.value - 1) * PER, page.value * PER))

// Form
type Form = { id?: string; prodi_id: string; mk_id: string; nama: string; dosen_id: string; ruang_id: string; slot: Slot; kuota: number }
const blank = (): Form => ({ prodi_id: prodiId.value, mk_id: '', nama: 'A', dosen_id: '', ruang_id: '', slot: { hari: 1, jam_mulai: '07:30', jam_selesai: '10:00' }, kuota: 40 })
const form = ref<Form>(blank())
const open = ref(false)
const conflicts = ref<{ code: string; items: Kelas[] } | null>(null)
const mkList = ref<MataKuliah[]>([])
watch(() => form.value.prodi_id, async (id) => {
  if (!id) { mkList.value = []; return }
  const kur = await api.get<Kurikulum[]>('/kurikulum', { prodi_id: id })
  const lists = await Promise.all(kur.filter((k) => k.aktif).map((k) => api.get<MataKuliah[]>(`/kurikulum/${k.id}/mata-kuliah`)))
  mkList.value = lists.flat()
})
function openForm(k?: Kelas) {
  conflicts.value = null
  form.value = k
    ? { id: k.id, prodi_id: prodiId.value, mk_id: k.mk_id, nama: k.nama, dosen_id: k.dosen_id ?? '', ruang_id: k.ruang_id ?? '', slot: { ...k.slot }, kuota: k.kuota }
    : blank()
  open.value = true
}
const { run, busy } = useAction()
async function save() {
  conflicts.value = null
  const body = { ...form.value, periode_id: periode.active?.id, dosen_id: form.value.dosen_id || null, ruang_id: form.value.ruang_id || null }
  busy.value = true
  try {
    if (form.value.id) await api.put(`/kelas/${form.value.id}`, body)
    else await api.post('/kelas', body)
    open.value = false
    toast('success', 'Kelas disimpan')
    load()
  } catch (e) {
    if (e instanceof ApiError && e.status === 409) conflicts.value = { code: e.code, items: (e.details as { conflicts?: Kelas[] })?.conflicts ?? [] }
    else toastError(e, 'Gagal menyimpan')
  } finally {
    busy.value = false
  }
}

// Ganti dosen
const gOpen = ref(false)
const gForm = ref({ id: '', dosen_id: '', efektif: new Date().toISOString().slice(0, 10) })
async function gantiDosen() {
  if (await run(() => api.post(`/kelas/${gForm.value.id}/ganti-dosen`, gForm.value), 'Dosen pengampu diganti')) { gOpen.value = false; load() }
}
async function cancel(k: Kelas) {
  const r = await confirm({ title: `Batalkan ${k.mk_nama} · ${k.nama}?`, body: `${k.terisi} KRS terkait dibatalkan otomatis, SKS dikembalikan, dan mahasiswa diberi tahu.`, withReason: true, danger: true, confirmLabel: 'Batalkan kelas' })
  if (r.ok && (await run(() => api.post(`/kelas/${k.id}/cancel`, { alasan: r.reason }), 'Kelas dibatalkan'))) load()
}
const menu = (k: Kelas) => [
  { label: 'Ubah jadwal', icon: Pencil, onClick: () => openForm(k) },
  { label: 'Ganti dosen', icon: UserRoundCog, onClick: () => { gForm.value = { id: k.id, dosen_id: '', efektif: new Date().toISOString().slice(0, 10) }; gOpen.value = true } },
  ...(k.status === 'dibuka' ? [{ label: 'Batalkan kelas', icon: Ban, danger: true, onClick: () => cancel(k) }] : []),
]
const hariOpts = [1, 2, 3, 4, 5, 6].map((h) => ({ value: h, label: HARI[h]! }))
const dosenOpts = computed(() => dosen.value.map((d) => ({ value: d.id, label: d.nama })))
const ruangOpts = computed(() => ruang.value.map((r) => ({ value: r.id, label: `${r.kode} · ${r.nama} (${r.kapasitas})` })))
</script>

<template>
  <div>
    <KPageHeader title="Jadwal & kelas" :description="periode.active ? `Periode ${periode.active.tipe} ${periode.active.tahun}/${periode.active.tahun + 1}. Bentrok ruang dan dosen dicek saat menyimpan.` : ''">
      <template #actions><KButton @click="openForm()"><template #icon><Plus class="size-4" /></template>Tambah kelas</KButton></template>
    </KPageHeader>
    <KCard flush>
      <KToolbar>
        <KSearch v-model="q" placeholder="Cari mata kuliah / dosen" />
        <KSelect v-model="prodiId" :options="prodi.map((p) => ({ value: p.id, label: `${p.jenjang} ${p.nama}` }))" placeholder="Semua prodi" class="w-full sm:w-52" />
        <template #right><span class="tabular text-sm text-ink-3">{{ rows.length }} kelas</span></template>
      </KToolbar>
      <KTable
        :columns="[
          { key: 'mk', label: 'Kelas' },
          { key: 'dosen', label: 'Dosen', hideMobile: true },
          { key: 'slot', label: 'Jadwal & ruang', hideMobile: true },
          { key: 'kuota', label: 'Terisi', width: '130px' },
          { key: 'status', label: 'Status', hideMobile: true },
          { key: 'menu', label: '', align: 'right', width: '56px' },
        ]"
        :rows="paged"
        :loading="loading"
        clickable
        @row-click="(k) => router.push(`/kelas/${k.id}`)"
      >
        <template #cell-mk="{ row }"><p class="font-semibold">{{ row.mk_nama }} · {{ row.nama }}</p><p class="text-sm text-ink-3"><span class="font-mono">{{ row.mk_kode }}</span> · {{ row.sks }} SKS</p></template>
        <template #cell-dosen="{ row }"><span v-if="row.dosen_nama">{{ row.dosen_nama }}</span><KBadge v-else tone="red">Belum ada dosen</KBadge></template>
        <template #cell-slot="{ row }"><p class="text-sm">{{ slot(row.slot) }}</p><p class="text-sm text-ink-3">{{ row.ruang_nama ?? '—' }}</p></template>
        <template #cell-kuota="{ row }">
          <div class="flex flex-col gap-1.5"><span class="tabular text-sm font-semibold">{{ row.terisi }}/{{ row.kuota }}</span><KProgress :value="row.terisi" :max="row.kuota" :tone="row.terisi >= row.kuota ? 'red' : 'blue'" /></div>
        </template>
        <template #cell-status="{ row }"><KStatus domain="kelas" :value="row.status" variant="dot" /></template>
        <template #cell-menu="{ row }"><div @click.stop><KMenu :items="menu(row)"><KIconButton label="Aksi"><MoreHorizontal class="size-4" /></KIconButton></KMenu></div></template>
      </KTable>
      <KPagination v-if="rows.length > PER" v-model="page" :total="rows.length" :per-page="PER" />
    </KCard>

    <KModal v-model="open" :title="form.id ? 'Ubah kelas' : 'Tambah kelas'" size="lg">
      <KAlert v-if="conflicts" level="error" :title="errorText(conflicts.code)" class="mb-4">
        <ul class="mt-1 flex flex-col gap-1">
          <li v-for="c in conflicts.items" :key="c.id">{{ c.mk_nama }} · {{ c.nama }} — {{ slot(c.slot) }}, {{ c.ruang_nama }}, {{ c.dosen_nama }}</li>
        </ul>
      </KAlert>
      <div class="grid gap-4 sm:grid-cols-2">
        <KField v-if="!form.id" label="Prodi" required><KSelect v-model="form.prodi_id" :options="prodi.map((p) => ({ value: p.id, label: `${p.jenjang} ${p.nama}` }))" placeholder="Pilih…" /></KField>
        <KField v-if="!form.id" label="Mata kuliah" required><KSelect v-model="form.mk_id" :options="mkList.map((m) => ({ value: m.id, label: `${m.kode} · ${m.nama} (${m.sks} SKS)` }))" placeholder="Pilih…" /></KField>
        <KField label="Nama kelas" required><KInput v-model="form.nama" /></KField>
        <KField label="Kuota" required><KInput v-model.number="form.kuota" type="number" min="1" /></KField>
        <KField label="Dosen pengampu"><KSelect v-model="form.dosen_id" :options="dosenOpts" placeholder="Belum ditentukan" /></KField>
        <KField label="Ruang"><KSelect v-model="form.ruang_id" :options="ruangOpts" placeholder="Belum ditentukan" /></KField>
        <KField label="Hari" required><KSelect v-model.number="form.slot.hari" :options="hariOpts" /></KField>
        <div class="grid grid-cols-2 gap-3">
          <KField label="Mulai" required><KInput v-model="form.slot.jam_mulai" type="time" /></KField>
          <KField label="Selesai" required><KInput v-model="form.slot.jam_selesai" type="time" /></KField>
        </div>
      </div>
      <template #footer>
        <KButton variant="secondary" @click="open = false">Batal</KButton>
        <KButton :loading="busy" :disabled="!form.id && !form.mk_id" @click="save">Simpan</KButton>
      </template>
    </KModal>

    <KModal v-model="gOpen" title="Ganti dosen pengampu" description="Presensi sebelumnya tetap atas nama dosen lama. Hak input nilai pindah sejak tanggal efektif." size="sm">
      <div class="flex flex-col gap-4">
        <KField label="Dosen baru" required><KSelect v-model="gForm.dosen_id" :options="dosenOpts" placeholder="Pilih…" /></KField>
        <KField label="Tanggal efektif" required><KInput v-model="gForm.efektif" type="date" /></KField>
      </div>
      <template #footer>
        <KButton variant="secondary" @click="gOpen = false">Batal</KButton>
        <KButton :loading="busy" :disabled="!gForm.dosen_id" @click="gantiDosen">Simpan</KButton>
      </template>
    </KModal>
  </div>
</template>
