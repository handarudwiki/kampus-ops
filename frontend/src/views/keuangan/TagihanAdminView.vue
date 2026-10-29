<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { CreditCard, FilePlus2, Landmark, ListOrdered, MoreHorizontal, Receipt } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Pembayaran, Prodi, Tagihan } from '@/api/types'
import { useAction } from '@/composables/useAsync'
import { confirm, dateTime, rupiah, rupiahShort, STATUS, statusOptions, toastError } from '@/design-system'
import { usePeriodeStore } from '@/stores/periode'
import { useTagihanStore } from '@/stores/tagihan'
import PayModal from './PayModal.vue'

const periode = usePeriodeStore()
const store = useTagihanStore()
const rows = ref<Tagihan[]>([])
const prodi = ref<Prodi[]>([])
const loading = ref(false)
const status = ref('')
const prodiId = ref('')
const q = ref('')

async function load() {
  loading.value = true
  try { rows.value = await api.get<Tagihan[]>('/tagihan', { status: status.value, prodi_id: prodiId.value, q: q.value, periode_id: periode.active?.id }) } catch (e) { toastError(e) } finally { loading.value = false }
}
onMounted(async () => {
  await periode.load()
  load()
  prodi.value = (await api.list<Prodi>('/prodi', { per_page: 100 }).catch(() => ({ data: [] }))).data
})
watch([status, prodiId, q], load)

const sum = computed(() => {
  const s = { total: 0, paid: 0, lunas: 0, sebagian: 0, belum: 0 }
  for (const t of rows.value) {
    s.total += t.jumlah; s.paid += t.terbayar
    if (t.status === 'lunas') s.lunas++; else if (t.status === 'sebagian') s.sebagian++; else s.belum++
  }
  return s
})

const { run, busy } = useAction()
async function generate() {
  if (!periode.active) return
  const r = await confirm({ title: 'Buat tagihan periode aktif?', body: 'Tagihan dibuat dari tarif prodi dan angkatan. Mahasiswa cuti mendapat biaya cuti. Tagihan yang sudah ada tidak digandakan.', confirmLabel: 'Buat tagihan' })
  if (r.ok && (await run(() => api.post('/tagihan/generate', { periode_id: periode.active!.id }), 'Tagihan dibuat'))) load()
}

const payOpen = ref(false)
const paying = ref<Tagihan | null>(null)
const cOpen = ref(false)
const cTarget = ref<Tagihan | null>(null)
const cJumlah = ref(2)
async function saveCicilan() {
  if (await run(() => store.cicilan(cTarget.value!.id, cJumlah.value), 'Cicilan disetujui')) { cOpen.value = false; load() }
}
const hOpen = ref(false)
const history = ref<Pembayaran[]>([])
async function showHistory(t: Tagihan) {
  history.value = await api.get<Pembayaran[]>(`/tagihan/${t.id}/pembayaran`)
  hOpen.value = true
}
const menu = (t: Tagihan) => [
  ...(t.status !== 'lunas' ? [{ label: 'Simulasi bayar', icon: CreditCard, onClick: () => { paying.value = t; payOpen.value = true } }] : []),
  ...(!t.va_number ? [{ label: 'Terbitkan VA', icon: Landmark, onClick: async () => { if (await run(() => store.issueVa(t.id), 'VA diterbitkan')) load() } }] : []),
  ...(t.status !== 'lunas' && !t.cicilan.length ? [{ label: 'Atur cicilan', icon: ListOrdered, onClick: () => { cTarget.value = t; cJumlah.value = 2; cOpen.value = true } }] : []),
  { label: 'Riwayat pembayaran', icon: Receipt, onClick: () => showHistory(t) },
]
</script>

<template>
  <div>
    <KPageHeader title="Tagihan mahasiswa" description="Status dihitung dari total pembayaran, bukan dari event terakhir, sehingga webhook ganda atau terbalik tetap konsisten.">
      <template #actions><KButton :loading="busy" @click="generate"><template #icon><FilePlus2 class="size-4" /></template>Buat tagihan periode</KButton></template>
    </KPageHeader>
    <div class="mb-3 grid gap-3 lg:grid-cols-3">
      <KStatCard label="Total tagihan" :value="rupiahShort(sum.total)" tone="blue" :hint="`${rows.length} tagihan`" />
      <KStatCard label="Terbayar" :value="rupiahShort(sum.paid)" tone="green" :hint="sum.total ? `${Math.round((sum.paid / sum.total) * 100)}% tertagih` : null" trend="up" />
      <KCard title="Komposisi status">
        <KGauge :segments="[{ label: STATUS.tagihan.lunas.label, value: sum.lunas, tone: 'green' }, { label: STATUS.tagihan.sebagian.label, value: sum.sebagian, tone: 'amber' }, { label: STATUS.tagihan.belum_bayar.label, value: sum.belum, tone: 'red' }]">
          <p class="tabular text-lg font-bold">{{ rows.length ? Math.round((sum.lunas / rows.length) * 100) : 0 }}%</p>
          <p class="text-xs text-ink-3">lunas</p>
        </KGauge>
      </KCard>
    </div>
    <KCard flush>
      <KToolbar>
        <KSearch v-model="q" placeholder="Cari NIM / nama" />
        <KSelect v-model="status" :options="statusOptions('tagihan')" placeholder="Semua status" class="w-full sm:w-40" />
        <KSelect v-model="prodiId" :options="prodi.map((p) => ({ value: p.id, label: `${p.jenjang} ${p.nama}` }))" placeholder="Semua prodi" class="w-full sm:w-52" />
      </KToolbar>
      <KTable
        :columns="[
          { key: 'mhs', label: 'Mahasiswa' },
          { key: 'jenis', label: 'Jenis', hideMobile: true },
          { key: 'jumlah', label: 'Terbayar / tagihan', align: 'right' },
          { key: 'va', label: 'VA', hideMobile: true },
          { key: 'status', label: 'Status' },
          { key: 'menu', label: '', align: 'right', width: '56px' },
        ]"
        :rows="rows"
        :loading="loading"
      >
        <template #cell-mhs="{ row }"><p class="font-semibold">{{ row.nama }}</p><p class="text-sm text-ink-3"><span class="font-mono">{{ row.nim }}</span> · {{ row.prodi_nama }}</p></template>
        <template #cell-jenis="{ row }"><KStatus domain="jenisTagihan" :value="row.jenis" /> <KBadge v-if="row.cicilan.length" tone="gray">{{ row.cicilan.length }}× cicil</KBadge></template>
        <template #cell-jumlah="{ row }"><span class="tabular"><b>{{ rupiah(row.terbayar) }}</b><span class="text-ink-3"> / {{ rupiah(row.jumlah) }}</span></span></template>
        <template #cell-va="{ row }"><span class="font-mono text-sm text-ink-2">{{ row.va_number ?? '—' }}</span></template>
        <template #cell-status="{ row }"><KStatus domain="tagihan" :value="row.status" variant="dot" /></template>
        <template #cell-menu="{ row }"><KMenu :items="menu(row)"><KIconButton label="Aksi"><MoreHorizontal class="size-4" /></KIconButton></KMenu></template>
      </KTable>
    </KCard>

    <PayModal v-model="payOpen" :tagihan="paying" @paid="load" />

    <KModal v-model="cOpen" title="Atur cicilan" :description="cTarget ? `${cTarget.nama} · ${rupiah(cTarget.jumlah)}. Cicilan yang disetujui membuka akses KRS walau belum lunas.` : ''" size="sm">
      <KField label="Jumlah cicilan"><KSegmented v-model="cJumlah" :options="[{ value: 2, label: '2×' }, { value: 3, label: '3×' }, { value: 4, label: '4×' }]" /></KField>
      <p v-if="cTarget" class="mt-3 text-sm text-ink-3">± {{ rupiah(Math.ceil(cTarget.jumlah / cJumlah)) }} per cicilan</p>
      <template #footer>
        <KButton variant="secondary" @click="cOpen = false">Batal</KButton>
        <KButton :loading="busy" @click="saveCicilan">Setujui cicilan</KButton>
      </template>
    </KModal>

    <KModal v-model="hOpen" title="Riwayat pembayaran" size="md">
      <KTable :columns="[{ key: 'transaction_id', label: 'Transaksi' }, { key: 'paid_at', label: 'Waktu' }, { key: 'jumlah', label: 'Jumlah', align: 'right' }]" :rows="history" empty-title="Belum ada pembayaran">
        <template #cell-transaction_id="{ value }"><span class="font-mono text-sm">{{ value }}</span></template>
        <template #cell-paid_at="{ value }">{{ dateTime(value) }}</template>
        <template #cell-jumlah="{ value }"><span class="tabular font-semibold">{{ rupiah(value as unknown as number) }}</span></template>
      </KTable>
    </KModal>
  </div>
</template>
