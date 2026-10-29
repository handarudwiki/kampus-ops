<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ArrowRight, Pencil, Plus } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Kurikulum, MataKuliah, Prodi } from '@/api/types'
import { useAction } from '@/composables/useAsync'
import { toastError } from '@/design-system'

/** FR-11..13: curriculum per prodi & cohort, prerequisites, equivalence mapping (EC-05). */
const prodi = ref<Prodi[]>([])
const prodiId = ref<string>('')
const kurikulum = ref<Kurikulum[]>([])
const kurId = ref<string>('')
const mk = ref<MataKuliah[]>([])
const loading = ref(false)
const tab = ref<'mk' | 'ekuivalensi'>('mk')

onMounted(async () => {
  try {
    prodi.value = (await api.list<Prodi>('/prodi', { per_page: 100 })).data
    prodiId.value = prodi.value[0]?.id ?? ''
  } catch (e) { toastError(e) }
})
watch(prodiId, async (id) => {
  if (!id) return
  kurikulum.value = await api.get<Kurikulum[]>('/kurikulum', { prodi_id: id })
  kurId.value = kurikulum.value.find((k) => k.aktif)?.id ?? kurikulum.value[0]?.id ?? ''
})
async function loadMk() {
  if (!kurId.value) { mk.value = []; return }
  loading.value = true
  try { mk.value = await api.get<MataKuliah[]>(`/kurikulum/${kurId.value}/mata-kuliah`) } finally { loading.value = false }
}
watch(kurId, () => { loadMk(); loadEkv() })

const bySemester = computed(() => {
  const m = new Map<number, MataKuliah[]>()
  for (const x of mk.value) m.set(x.semester, [...(m.get(x.semester) ?? []), x])
  return [...m.entries()].sort((a, b) => a[0] - b[0])
})

// MK form
const open = ref(false)
const form = ref<{ id?: string; kode: string; nama: string; sks: number; semester: number; sifat: 'wajib' | 'pilihan'; prasyarat: { mk_id: string; nilai_min: string | null }[] }>({ kode: '', nama: '', sks: 3, semester: 1, sifat: 'wajib', prasyarat: [] })
const { run, busy } = useAction()
function openForm(x?: MataKuliah) {
  form.value = x
    ? { id: x.id, kode: x.kode, nama: x.nama, sks: x.sks, semester: x.semester, sifat: x.sifat, prasyarat: x.prasyarat.map((p) => ({ mk_id: p.mk_id, nilai_min: p.nilai_min })) }
    : { kode: '', nama: '', sks: 3, semester: 1, sifat: 'wajib', prasyarat: [] }
  open.value = true
}
async function save() {
  const body = { ...form.value, kurikulum_id: kurId.value }
  const ok = await run(() => (form.value.id ? api.put(`/mata-kuliah/${form.value.id}`, body) : api.post('/mata-kuliah', body)), 'Mata kuliah disimpan')
  if (ok) { open.value = false; loadMk() }
}
const mkOpts = computed(() => mk.value.filter((x) => x.id !== form.value.id).map((x) => ({ value: x.id, label: `${x.kode} · ${x.nama}` })))
const nilaiOpts = ['A', 'AB', 'B', 'BC', 'C', 'D'].map((v) => ({ value: v, label: `Minimal ${v}` }))

// Ekuivalensi
interface Ekv { id: string; mk_lama_kode: string; mk_lama_nama: string; mk_baru_kode: string; mk_baru_nama: string }
const ekv = ref<Ekv[]>([])
async function loadEkv() {
  if (!kurId.value) return
  try { ekv.value = await api.get<Ekv[]>('/ekuivalensi', { kurikulum_id: kurId.value }) } catch { ekv.value = [] }
}
const ekvOpen = ref(false)
const ekvForm = ref({ mk_lama_id: '', mk_baru_id: '' })
const lamaOpts = ref<{ value: string; label: string }[]>([])
async function openEkv() {
  ekvForm.value = { mk_lama_id: '', mk_baru_id: '' }
  const others = kurikulum.value.filter((k) => k.id !== kurId.value)
  const lists = await Promise.all(others.map((k) => api.get<MataKuliah[]>(`/kurikulum/${k.id}/mata-kuliah`).then((l) => l.map((x) => ({ value: x.id, label: `${k.nama} · ${x.kode} ${x.nama}` })))))
  lamaOpts.value = lists.flat()
  ekvOpen.value = true
}
async function saveEkv() {
  if (await run(() => api.post('/ekuivalensi', ekvForm.value), 'Ekuivalensi ditambahkan')) { ekvOpen.value = false; loadEkv() }
}
</script>

<template>
  <div>
    <KPageHeader title="Kurikulum" description="Mata kuliah, prasyarat, dan ekuivalensi antar kurikulum.">
      <template #actions>
        <KSelect v-model="prodiId" :options="prodi.map((p) => ({ value: p.id, label: `${p.jenjang} ${p.nama}` }))" class="w-56" />
        <KSelect v-model="kurId" :options="kurikulum.map((k) => ({ value: k.id, label: `${k.nama}${k.aktif ? ' (aktif)' : ''}` }))" class="w-48" />
      </template>
    </KPageHeader>

    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <KSegmented v-model="tab" :options="[{ value: 'mk', label: 'Mata kuliah', count: mk.length }, { value: 'ekuivalensi', label: 'Ekuivalensi', count: ekv.length }]" />
      <KButton v-if="tab === 'mk'" :disabled="!kurId" @click="openForm()"><template #icon><Plus class="size-4" /></template>Tambah mata kuliah</KButton>
      <KButton v-else :disabled="!kurId" @click="openEkv"><template #icon><Plus class="size-4" /></template>Tambah ekuivalensi</KButton>
    </div>

    <template v-if="tab === 'mk'">
      <KSkeleton v-if="loading" :lines="6" />
      <div v-else class="flex flex-col gap-3">
        <KCard v-for="[sem, list] in bySemester" :key="sem" :title="`Semester ${sem}`" :subtitle="`${list.reduce((a, x) => a + x.sks, 0)} SKS · ${list.length} mata kuliah`" flush>
          <KTable
            :columns="[{ key: 'kode', label: 'Kode', width: '100px' }, { key: 'nama', label: 'Mata kuliah' }, { key: 'sks', label: 'SKS', align: 'right', width: '70px' }, { key: 'sifat', label: 'Sifat', hideMobile: true }, { key: 'prasyarat', label: 'Prasyarat', hideMobile: true }, { key: 'aksi', label: '', align: 'right', width: '56px' }]"
            :rows="list"
          >
            <template #cell-kode="{ value }"><span class="font-mono text-sm">{{ value }}</span></template>
            <template #cell-sifat="{ value }"><KBadge :tone="value === 'wajib' ? 'blue' : 'gray'">{{ value === 'wajib' ? 'Wajib' : 'Pilihan' }}</KBadge></template>
            <template #cell-prasyarat="{ row }">
              <div class="flex flex-wrap gap-1">
                <KBadge v-for="p in row.prasyarat" :key="p.mk_id" tone="violet">{{ p.kode }}{{ p.nilai_min ? ` ≥ ${p.nilai_min}` : '' }}</KBadge>
                <span v-if="!row.prasyarat.length" class="text-ink-3">—</span>
              </div>
            </template>
            <template #cell-aksi="{ row }"><KIconButton label="Ubah" @click="openForm(row)"><Pencil class="size-4" /></KIconButton></template>
          </KTable>
        </KCard>
        <KCard v-if="!bySemester.length"><KEmpty title="Kurikulum masih kosong" /></KCard>
      </div>
    </template>

    <KCard v-else flush>
      <KTable :columns="[{ key: 'lama', label: 'Mata kuliah lama' }, { key: 'arrow', label: '', width: '40px', align: 'center' }, { key: 'baru', label: 'Setara dengan' }]" :rows="ekv" empty-title="Belum ada ekuivalensi" empty-description="Ekuivalensi dipakai agar prasyarat bisa dipenuhi lewat mata kuliah kurikulum lama.">
        <template #cell-lama="{ row }"><span class="font-mono text-sm">{{ row.mk_lama_kode }}</span> {{ row.mk_lama_nama }}</template>
        <template #cell-arrow><ArrowRight class="mx-auto size-4 text-ink-3" /></template>
        <template #cell-baru="{ row }"><span class="font-mono text-sm">{{ row.mk_baru_kode }}</span> {{ row.mk_baru_nama }}</template>
      </KTable>
    </KCard>

    <KModal v-model="open" :title="form.id ? 'Ubah mata kuliah' : 'Tambah mata kuliah'" size="lg">
      <div class="grid gap-4 sm:grid-cols-2">
        <KField label="Kode" required><KInput v-model="form.kode" /></KField>
        <KField label="Sifat" required><KSelect v-model="form.sifat" :options="[{ value: 'wajib', label: 'Wajib' }, { value: 'pilihan', label: 'Pilihan' }]" /></KField>
        <KField label="Nama" required class="sm:col-span-2"><KInput v-model="form.nama" /></KField>
        <KField label="SKS" required><KInput v-model.number="form.sks" type="number" min="1" max="6" /></KField>
        <KField label="Semester anjuran" required><KInput v-model.number="form.semester" type="number" min="1" max="14" /></KField>
        <div class="sm:col-span-2">
          <p class="mb-2 text-sm font-semibold text-ink-2">Prasyarat</p>
          <div class="flex flex-col gap-2">
            <div v-for="(p, i) in form.prasyarat" :key="i" class="flex gap-2">
              <KSelect v-model="p.mk_id" :options="mkOpts" placeholder="Pilih mata kuliah" class="flex-1" />
              <KSelect v-model="p.nilai_min" :options="nilaiOpts" placeholder="Cukup lulus" class="w-40" />
              <KButton variant="ghost" @click="form.prasyarat.splice(i, 1)">Hapus</KButton>
            </div>
            <KButton variant="soft" size="sm" class="self-start" @click="form.prasyarat.push({ mk_id: '', nilai_min: null })"><template #icon><Plus class="size-4" /></template>Tambah prasyarat</KButton>
          </div>
        </div>
      </div>
      <template #footer>
        <KButton variant="secondary" @click="open = false">Batal</KButton>
        <KButton :loading="busy" :disabled="!form.kode || !form.nama" @click="save">Simpan</KButton>
      </template>
    </KModal>

    <KModal v-model="ekvOpen" title="Tambah ekuivalensi" size="md">
      <div class="flex flex-col gap-4">
        <KField label="Mata kuliah lama" required><KSelect v-model="ekvForm.mk_lama_id" :options="lamaOpts" placeholder="Pilih…" /></KField>
        <KField label="Setara dengan (kurikulum ini)" required><KSelect v-model="ekvForm.mk_baru_id" :options="mk.map((x) => ({ value: x.id, label: `${x.kode} · ${x.nama}` }))" placeholder="Pilih…" /></KField>
      </div>
      <template #footer>
        <KButton variant="secondary" @click="ekvOpen = false">Batal</KButton>
        <KButton :loading="busy" :disabled="!ekvForm.mk_lama_id || !ekvForm.mk_baru_id" @click="saveEkv">Simpan</KButton>
      </template>
    </KModal>
  </div>
</template>
