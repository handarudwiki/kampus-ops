<script setup lang="ts">
import { ref } from 'vue'
import { Plus } from 'lucide-vue-next'
import type { Tone } from '@/api/types'
import { confirm, STATUS, toast, toneClass, type StatusDomain } from '@/design-system'

/** Living reference for the design system. Public route: /design-system */
const tones: Tone[] = ['blue', 'green', 'amber', 'red', 'violet', 'pink', 'gray']
const surfaces = [
  { name: 'canvas', cls: 'bg-canvas' }, { name: 'surface', cls: 'bg-surface' },
  { name: 'surface-2', cls: 'bg-surface-2' }, { name: 'surface-3', cls: 'bg-surface-3' },
  { name: 'line', cls: 'bg-line' }, { name: 'ink', cls: 'bg-ink' }, { name: 'ink-2', cls: 'bg-ink-2' },
  { name: 'ink-3', cls: 'bg-ink-3' }, { name: 'accent', cls: 'bg-accent' },
]
const type = [
  { cls: 'text-2xl font-bold', name: 'display · 40/44 bold', sample: '253.056' },
  { cls: 'text-xl font-bold', name: 'page title · 28/34 bold', sample: 'Persetujuan KRS' },
  { cls: 'text-lg font-bold', name: 'heading · 20/28 bold', sample: 'Halo, Rina' },
  { cls: 'text-md font-bold', name: 'card title · 16/22 bold', sample: 'Kelas hari ini' },
  { cls: 'text-base', name: 'body · 14/20', sample: 'Mahasiswa memilih kelas di jendela KRS.' },
  { cls: 'text-sm text-ink-3', name: 'caption · 13/18 ink-3', sample: 'Diperbarui 3 menit lalu' },
  { cls: 'ko-eyebrow', name: 'eyebrow · 12 caps', sample: 'Mata kuliah' },
]
const seg = ref<'a' | 'b' | 'c'>('a')
const text = ref('')
const sel = ref('aktif')
const modal = ref(false)
const rows = [
  { id: '1', kode: 'IF201', nama: 'Struktur Data', sks: 3, status: 'disetujui' },
  { id: '2', kode: 'IF202', nama: 'Basis Data', sks: 3, status: 'diajukan' },
  { id: '3', kode: 'IF203', nama: 'Jaringan Komputer', sks: 2, status: 'ditolak' },
]
const domains = Object.keys(STATUS) as StatusDomain[]
async function ask() {
  const r = await confirm({ title: 'Tolak KRS?', body: 'Mahasiswa akan diberi tahu dan bisa memperbaiki KRS.', withReason: true, danger: true, confirmLabel: 'Tolak' })
  if (r.ok) toast('success', 'KRS ditolak', r.reason)
}
</script>

<template>
  <div class="mx-auto max-w-6xl p-4 sm:p-6">
    <KPageHeader title="KampusOps Design System" description="Token, komponen, dan pola yang dipakai semua halaman. Satu sumber, satu tampilan.">
      <template #actions><KButton to="/" variant="secondary">Kembali ke aplikasi</KButton></template>
    </KPageHeader>

    <div class="grid gap-3">
      <KCard title="Prinsip" subtitle="Aturan yang membuat UI konsisten">
        <ol class="grid gap-3 sm:grid-cols-2">
          <li v-for="(p, i) in [
            ['Token, bukan hex', 'Palet Tailwind dikunci ke token di tokens.css. bg-blue-500 tidak ada; pakai bg-blue, bg-blue-soft, text-blue-ink.'],
            ['Satu peta status', 'Semua status domain dirender lewat <KStatus>, label dan warnanya dari status.ts. Disetujui selalu hijau.'],
            ['Kartu adalah wadah', 'Semua konten duduk di <KCard> putih radius 20px di atas canvas abu. Tidak ada border tebal atau bayangan berat.'],
            ['Hitam untuk aksi utama', 'Tombol primer berwarna ink (hitam). Satu aksi primer per area; sisanya secondary atau ghost.'],
            ['Angka tabular', 'Angka memakai tabular-nums dan format.ts (Rupiah, IPK 3,42, tanggal id-ID). Ekor angka besar diredam.'],
            ['Pola halaman sama', 'KPageHeader → KCard (KToolbar + KTable) → KModal untuk form. Loading = skeleton, error = toast.'],
          ]" :key="i" class="rounded-control bg-surface-2 p-4">
            <p class="font-bold">{{ i + 1 }}. {{ p[0] }}</p>
            <p class="mt-1 text-sm text-ink-2">{{ p[1] }}</p>
          </li>
        </ol>
      </KCard>

      <div class="grid gap-3 lg:grid-cols-2">
        <KCard title="Surface & ink">
          <div class="grid grid-cols-3 gap-3">
            <div v-for="s in surfaces" :key="s.name" class="flex flex-col gap-1.5">
              <div :class="['h-12 rounded-control border border-line', s.cls]" />
              <code class="font-mono text-xs text-ink-2">{{ s.name }}</code>
            </div>
          </div>
        </KCard>
        <KCard title="Tone semantik" subtitle="solid · soft · ink">
          <div class="flex flex-col gap-2">
            <div v-for="t in tones" :key="t" class="flex items-center gap-3">
              <span :class="['size-6 rounded-pill', toneClass[t].dot]" />
              <span :class="['flex h-7 flex-1 items-center rounded-pill px-3 text-xs font-semibold', toneClass[t].soft]">{{ t }}</span>
            </div>
          </div>
        </KCard>
      </div>

      <KCard title="Tipografi" subtitle="Plus Jakarta Sans · JetBrains Mono untuk kode">
        <div class="flex flex-col divide-y divide-line">
          <div v-for="t in type" :key="t.name" class="flex flex-col gap-1 py-3 sm:flex-row sm:items-baseline sm:gap-6">
            <code class="w-56 shrink-0 font-mono text-xs text-ink-3">{{ t.name }}</code>
            <span :class="[t.cls, 'tabular']">{{ t.sample }}</span>
          </div>
        </div>
      </KCard>

      <div class="grid gap-3 sm:grid-cols-3">
        <KStatCard label="Mahasiswa aktif" value="1.284" tone="blue" hint="+4% dari semester lalu" trend="up" />
        <KStatCard label="Piutang" value="Rp 482,5 jt" tone="red" hint="18 jatuh tempo" trend="down" />
        <KStatCard label="Skor PDDikti" value="92,4" tone="green" hint="3 error tersisa" />
      </div>

      <div class="grid gap-3 lg:grid-cols-2">
        <KCard title="Tombol">
          <div class="flex flex-wrap items-center gap-2">
            <KButton><template #icon><Plus class="size-4" /></template>Primary</KButton>
            <KButton variant="secondary">Secondary</KButton>
            <KButton variant="soft">Soft</KButton>
            <KButton variant="ghost">Ghost</KButton>
            <KButton variant="danger">Danger</KButton>
            <KButton loading>Loading</KButton>
            <KButton size="sm">Small</KButton>
          </div>
        </KCard>
        <KCard title="Kontrol">
          <div class="flex flex-col gap-4">
            <KSegmented v-model="seg" :options="[{ value: 'a', label: 'Semester ini' }, { value: 'b', label: 'Semester lalu', count: 4 }, { value: 'c', label: 'Kustom' }]" />
            <div class="grid gap-3 sm:grid-cols-2">
              <KField label="Nama" hint="Sesuai KTP"><KInput v-model="text" placeholder="Rina Wulandari" /></KField>
              <KField label="Status"><KSelect v-model="sel" :options="[{ value: 'aktif', label: 'Aktif' }, { value: 'cuti', label: 'Cuti' }]" /></KField>
              <KField label="NIK" error="NIK harus 16 digit"><KInput model-value="12345" invalid /></KField>
              <KField label="Cari"><KSearch /></KField>
            </div>
          </div>
        </KCard>
      </div>

      <KCard title="Status" subtitle="Dari design-system/status.ts — gunakan <KStatus domain value>">
        <div class="flex flex-col gap-3">
          <div v-for="d in domains" :key="d" class="flex flex-col gap-2 sm:flex-row sm:items-center">
            <code class="w-36 shrink-0 font-mono text-xs text-ink-3">{{ d }}</code>
            <div class="flex flex-wrap gap-2">
              <KStatus v-for="k in Object.keys(STATUS[d])" :key="k" :domain="d" :value="k" />
            </div>
          </div>
          <div class="flex flex-wrap gap-4 border-t border-line pt-3">
            <KStatus domain="tagihan" value="lunas" variant="dot" />
            <KStatus domain="tagihan" value="belum_bayar" variant="dot" />
            <KStatus domain="krs" value="diajukan" variant="dot" />
          </div>
        </div>
      </KCard>

      <KCard title="Tabel" flush>
        <KToolbar>
          <KSearch />
          <template #right><KButton size="sm" variant="secondary">Ekspor</KButton></template>
        </KToolbar>
        <KTable :columns="[{ key: 'kode', label: 'Kode' }, { key: 'nama', label: 'Mata kuliah' }, { key: 'sks', label: 'SKS', align: 'right' }, { key: 'status', label: 'Status' }]" :rows="rows">
          <template #cell-kode="{ value }"><span class="font-mono text-sm">{{ value }}</span></template>
          <template #cell-status="{ value }"><KStatus domain="krs" :value="value" variant="dot" /></template>
        </KTable>
      </KCard>

      <div class="grid gap-3 lg:grid-cols-3">
        <KCard title="Bar chart">
          <KBarChart :labels="['Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab']" :series="[{ name: 'Hadir', values: [42, 38, 45, 40, 36, 20] }]" :height="160" />
        </KCard>
        <KCard title="Line chart">
          <KLineChart :labels="['W1', 'W2', 'W3', 'W4', 'W5']" :series="[{ name: 'Lunas', values: [120, 150, 110, 170, 200] }, { name: 'Sebagian', values: [80, 60, 90, 70, 85] }]" :height="160" />
        </KCard>
        <KCard title="Gauge & ring">
          <div class="flex items-center justify-around gap-4">
            <KGauge :segments="[{ label: 'Lunas', value: 62, tone: 'green' }, { label: 'Sebagian', value: 23, tone: 'amber' }, { label: 'Belum', value: 15, tone: 'red' }]" :legend="false">
              <p class="tabular text-lg font-bold">62%</p>
            </KGauge>
            <KRing :value="92" tone="green" :size="110"><p class="tabular text-lg font-bold">92</p></KRing>
          </div>
        </KCard>
      </div>

      <div class="grid gap-3 lg:grid-cols-2">
        <KCard title="Umpan balik">
          <div class="flex flex-col gap-2">
            <KAlert level="info" title="Jendela KRS dibuka">1–14 Agustus, zona waktu kampus.</KAlert>
            <KAlert level="warning">Sisa 2 semester sebelum batas masa studi.</KAlert>
            <KAlert level="error" title="Ekspor diblokir">Masih ada 3 error validasi.</KAlert>
            <div class="flex flex-wrap gap-2 pt-2">
              <KButton variant="secondary" @click="toast('success', 'KRS disetujui', '24 SKS · 8 mata kuliah')">Toast</KButton>
              <KButton variant="secondary" @click="ask">Confirm + alasan</KButton>
              <KButton variant="secondary" @click="modal = true">Modal</KButton>
            </div>
          </div>
        </KCard>
        <KCard title="Alur & progres">
          <div class="flex flex-col gap-6">
            <KStepper :steps="[{ key: 'draft', label: 'Draft' }, { key: 'diajukan', label: 'Diajukan' }, { key: 'disetujui', label: 'Disetujui' }]" current="diajukan" />
            <div class="flex flex-col gap-2">
              <div class="flex justify-between text-sm"><span class="text-ink-2">SKS terpakai</span><span class="tabular font-bold">18 / 24</span></div>
              <KProgress :value="18" :max="24" />
            </div>
            <div class="flex items-center gap-3"><KAvatar name="Rina Wulandari" /><KAvatar name="Budi Santoso" /><KAvatar name="Sari Dewi" /><KBadge tone="violet">Kaprodi</KBadge></div>
          </div>
        </KCard>
      </div>

      <KCard tone="inset" title="Kosong">
        <KEmpty title="Belum ada KRS" description="Pilih kelas dari daftar penawaran untuk mulai menyusun KRS."><KButton size="sm">Pilih kelas</KButton></KEmpty>
      </KCard>
    </div>

    <KModal v-model="modal" title="Tambah ruang" description="Ruang dipakai untuk deteksi bentrok jadwal.">
      <div class="grid gap-4 sm:grid-cols-2">
        <KField label="Kode" required><KInput placeholder="R-301" /></KField>
        <KField label="Kapasitas" required><KInput type="number" placeholder="40" /></KField>
      </div>
      <template #footer>
        <KButton variant="secondary" @click="modal = false">Batal</KButton>
        <KButton @click="modal = false">Simpan</KButton>
      </template>
    </KModal>
  </div>
</template>
