<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Minus, Plus, Send } from 'lucide-vue-next'
import { useKrsStore } from '@/stores/krs'
import { confirm, dateTime, errorText, slot, STATUS, toast, toastError } from '@/design-system'
import { useAuthStore } from '@/stores/auth'
import type { Offering } from '@/api/types'

const store = useKrsStore()
const auth = useAuthStore()
onMounted(() => store.load().catch((e) => toastError(e, 'Gagal memuat KRS')))

const q = ref('')
const filter = ref<'semua' | 'bisa' | 'diambil'>('semua')
const rows = computed(() => {
  const term = q.value.toLowerCase()
  return store.offerings.filter((o) => {
    if (term && !`${o.mk_kode} ${o.mk_nama} ${o.dosen_nama ?? ''}`.toLowerCase().includes(term)) return false
    if (filter.value === 'bisa') return o.eligible && !o.in_krs
    if (filter.value === 'diambil') return o.in_krs
    return true
  })
})
const counts = computed(() => ({
  bisa: store.offerings.filter((o) => o.eligible && !o.in_krs).length,
  diambil: store.offerings.filter((o) => o.in_krs).length,
}))

const steps = [
  { key: 'draft', label: STATUS.krs.draft.label },
  { key: 'diajukan', label: STATUS.krs.diajukan.label },
  { key: 'disetujui', label: STATUS.krs.disetujui.label },
]
const phaseLabel = { krs: 'Masa KRS', revisi: 'Masa revisi', closed: 'Ditutup' }

async function add(o: Offering) {
  try {
    await store.add(o.id)
    toast('success', `${o.mk_nama} ditambahkan`)
  } catch (e) {
    toastError(e, 'Tidak bisa mengambil kelas')
    store.refreshOfferings().catch(() => {})
  }
}
async function remove(id: string, nama: string) {
  try { await store.remove(id); toast('info', `${nama} dihapus dari KRS`) } catch (e) { toastError(e) }
}
const submitting = ref(false)
async function submit() {
  const k = store.krs
  if (!k) return
  const r = await confirm({ title: 'Ajukan KRS ke dosen wali?', body: `${k.items.length} kelas · ${k.total_sks} SKS. KRS tidak bisa diubah sampai diproses dosen wali.`, confirmLabel: 'Ajukan' })
  if (!r.ok) return
  submitting.value = true
  try { await store.submit(); toast('success', 'KRS diajukan', 'Menunggu persetujuan dosen wali.') } catch (e) { toastError(e, 'Gagal mengajukan') } finally { submitting.value = false }
}
const tz = computed(() => auth.tenant?.timezone)
</script>

<template>
  <div>
    <KPageHeader title="Kartu Rencana Studi" :description="store.me ? `${phaseLabel[store.me.window.phase]} · ${dateTime(store.me.window.start, tz)} – ${dateTime(store.me.window.end, tz)}` : 'Memuat…'">
      <template #actions>
        <KButton v-if="store.editable && store.krs?.items.length" :loading="submitting" @click="submit">
          <template #icon><Send class="size-4" /></template>Ajukan ke dosen wali
        </KButton>
      </template>
    </KPageHeader>

    <div v-if="store.loading && !store.me" class="grid gap-3 lg:grid-cols-3">
      <div class="h-96 animate-pulse rounded-card bg-surface lg:col-span-2" /><div class="h-96 animate-pulse rounded-card bg-surface" />
    </div>

    <template v-else-if="store.me">
      <div class="mb-3 flex flex-col gap-2">
        <KAlert v-if="!store.me.window.open" level="info" title="Jendela KRS tidak dibuka">KRS hanya bisa diubah di masa KRS atau masa revisi.</KAlert>
        <KAlert v-if="!store.me.eligibility.ok" level="error" title="Anda belum bisa mengisi KRS">
          <ul class="list-inside list-disc">
            <li v-for="r in store.me.eligibility.reasons" :key="r">{{ errorText(r) }}</li>
          </ul>
          <template v-if="store.me.eligibility.reasons.includes('KRS_UNPAID')" #action><KButton size="sm" to="/tagihan">Lihat tagihan</KButton></template>
        </KAlert>
        <KAlert v-if="store.krs?.status === 'ditolak'" level="warning" title="KRS ditolak dosen wali">{{ store.krs.catatan ?? 'Perbaiki lalu ajukan kembali.' }}</KAlert>
        <KAlert v-if="store.krs?.status === 'disetujui'" level="success" title="KRS disetujui">{{ store.krs.catatan ?? 'Selamat kuliah!' }}</KAlert>
      </div>

      <div class="grid gap-3 lg:grid-cols-3">
        <!-- Offerings -->
        <KCard title="Penawaran kelas" subtitle="Prasyarat, bentrok jadwal, dan kuota dicek saat Anda mengambil kelas" flush class="lg:col-span-2">
          <KToolbar>
            <KSearch v-model="q" placeholder="Cari mata kuliah / dosen" />
            <KSegmented v-model="filter" :options="[{ value: 'semua', label: 'Semua' }, { value: 'bisa', label: 'Bisa diambil', count: counts.bisa }, { value: 'diambil', label: 'Diambil', count: counts.diambil }]" />
          </KToolbar>
          <KTable
            :columns="[
              { key: 'mk', label: 'Mata kuliah' },
              { key: 'slot', label: 'Jadwal', hideMobile: true },
              { key: 'kuota', label: 'Kuota', width: '130px', hideMobile: true },
              { key: 'aksi', label: '', align: 'right', width: '110px' },
            ]"
            :rows="rows"
            empty-title="Tidak ada kelas"
          >
            <template #cell-mk="{ row }">
              <p class="font-semibold">{{ row.mk_nama }} <span class="font-normal text-ink-3">· {{ row.nama }}</span></p>
              <p class="text-sm text-ink-3"><span class="font-mono">{{ row.mk_kode }}</span> · {{ row.sks }} SKS · {{ row.dosen_nama ?? 'Dosen belum ditentukan' }}</p>
              <p class="text-sm text-ink-3 sm:hidden">{{ slot(row.slot) }}</p>
              <div v-if="!row.eligible && !row.in_krs && row.reasons.length" class="mt-1.5 flex flex-wrap gap-1">
                <KBadge v-for="r in row.reasons" :key="r" tone="red">{{ errorText(r) }}</KBadge>
              </div>
            </template>
            <template #cell-slot="{ row }"><span class="whitespace-nowrap text-sm text-ink-2">{{ slot(row.slot) }}</span><br /><span class="text-sm text-ink-3">{{ row.ruang_nama }}</span></template>
            <template #cell-kuota="{ row }">
              <div class="flex flex-col gap-1.5">
                <span class="tabular text-sm font-semibold">{{ row.terisi }}/{{ row.kuota }}</span>
                <KProgress :value="row.terisi" :max="row.kuota" :tone="row.terisi >= row.kuota ? 'red' : row.terisi / row.kuota > 0.8 ? 'amber' : 'green'" />
              </div>
            </template>
            <template #cell-aksi="{ row }">
              <KButton v-if="row.in_krs" size="sm" variant="secondary" :disabled="!store.editable" :loading="store.pending === row.id" @click="remove(row.id, row.mk_nama)">
                <template #icon><Minus class="size-4" /></template>Batal
              </KButton>
              <KButton v-else size="sm" :disabled="!store.editable || !row.eligible || !store.me.eligibility.ok" :loading="store.pending === row.id" @click="add(row)">
                <template #icon><Plus class="size-4" /></template>Ambil
              </KButton>
            </template>
          </KTable>
        </KCard>

        <!-- Summary -->
        <div class="flex flex-col gap-3">
          <KCard title="Ringkasan KRS">
            <template #actions><KStatus domain="krs" :value="store.krs?.status ?? 'draft'" /></template>
            <div class="flex items-center gap-5">
              <KRing :value="store.krs?.total_sks ?? 0" :max="store.krs?.max_sks ?? 24" :tone="store.sksLeft < 0 ? 'red' : 'blue'" :size="112">
                <p class="tabular text-lg font-bold leading-none">{{ store.krs?.total_sks ?? 0 }}</p>
                <p class="text-xs text-ink-3">dari {{ store.krs?.max_sks ?? '—' }} SKS</p>
              </KRing>
              <KKeyValue :cols="1" :items="[{ label: 'IPS semester lalu', value: store.krs?.ips_lalu?.replace('.', ',') ?? '—' }, { label: 'Sisa SKS', value: store.krs ? store.sksLeft : '—' }]" />
            </div>
            <div class="mt-5"><KStepper :steps="steps" :current="store.krs?.status === 'ditolak' ? 'disetujui' : store.krs?.status ?? 'draft'" :failed="store.krs?.status === 'ditolak'" /></div>
          </KCard>
          <KCard title="Kelas diambil" flush>
            <ul v-if="store.krs?.items.length" class="divide-y divide-line">
              <li v-for="it in store.krs.items" :key="it.kelas.id" class="flex items-center gap-3 px-5 py-3">
                <div class="min-w-0 flex-1">
                  <p class="truncate font-semibold">{{ it.kelas.mk_nama }} <KBadge v-if="it.is_ulang" tone="amber">Ulang</KBadge></p>
                  <p class="text-sm text-ink-3">{{ slot(it.kelas.slot) }}</p>
                </div>
                <span class="tabular text-sm font-bold">{{ it.kelas.sks }} SKS</span>
              </li>
            </ul>
            <KEmpty v-else title="Belum ada kelas" description="Ambil kelas dari daftar penawaran." />
          </KCard>
        </div>
      </div>
    </template>
  </div>
</template>
