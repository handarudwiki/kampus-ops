<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { CalendarPlus, CheckCheck } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Pertemuan, PresensiRow, PresensiStatus } from '@/api/types'
import { useAction, useAsync } from '@/composables/useAsync'
import { date, STATUS, toneClass } from '@/design-system'

/** FR-21. Lecturer is recorded per meeting, so a mid-semester swap keeps history intact (EC-12). */
const props = defineProps<{ kelasId: string; canEdit: boolean }>()
const pertemuan = useAsync<Pertemuan[]>(() => api.get<Pertemuan[]>(`/kelas/${props.kelasId}/pertemuan`), [])
const selected = ref<string | null>(null)
watch(pertemuan.data, (p) => { if (!selected.value && p.length) selected.value = p[p.length - 1]!.id })

const rows = ref<PresensiRow[]>([])
const loadingRows = ref(false)
watch(selected, async (id) => {
  if (!id) return
  loadingRows.value = true
  try { rows.value = await api.get<PresensiRow[]>(`/pertemuan/${id}/presensi`) } finally { loadingRows.value = false }
})

const { run, busy } = useAction()
const codes: PresensiStatus[] = ['H', 'I', 'S', 'A']
const allHadir = () => rows.value.forEach((r) => (r.status = 'H'))
async function save() {
  const items = rows.value.filter((r) => r.status).map((r) => ({ mahasiswa_id: r.mahasiswa_id, status: r.status }))
  if (await run(() => api.put(`/pertemuan/${selected.value}/presensi`, { items }), 'Presensi disimpan')) pertemuan.reload()
}

const open = ref(false)
const form = ref({ tanggal: new Date().toISOString().slice(0, 10), topik: '' })
async function create() {
  const ok = await run(async () => {
    const p = await api.post<Pertemuan>(`/kelas/${props.kelasId}/pertemuan`, form.value)
    await pertemuan.reload()
    selected.value = p.id
  }, 'Pertemuan dibuka')
  if (ok) { open.value = false; form.value.topik = '' }
}
const current = computed(() => pertemuan.data.value.find((p) => p.id === selected.value))
</script>

<template>
  <div class="grid gap-3 lg:grid-cols-[280px_1fr]">
    <KCard title="Pertemuan" flush>
      <template #actions><KButton v-if="canEdit" size="sm" variant="soft" @click="open = true"><template #icon><CalendarPlus class="size-4" /></template>Buka</KButton></template>
      <ul class="ko-scroll max-h-[520px] overflow-y-auto p-2">
        <li v-for="p in pertemuan.data.value" :key="p.id">
          <button
            :class="['flex w-full flex-col gap-1 rounded-control px-3 py-2.5 text-left transition-colors', selected === p.id ? 'bg-surface-3' : 'hover:bg-surface-2']"
            @click="selected = p.id"
          >
            <span class="flex items-center justify-between"><span class="font-semibold">Pertemuan {{ p.ke }}</span><span class="text-xs text-ink-3">{{ date(p.tanggal) }}</span></span>
            <span class="truncate text-sm text-ink-3">{{ p.topik }}</span>
            <span class="tabular text-xs text-ink-2">H {{ p.rekap.hadir }} · I {{ p.rekap.izin }} · S {{ p.rekap.sakit }} · A {{ p.rekap.alpa }}</span>
          </button>
        </li>
        <li v-if="!pertemuan.data.value.length"><KEmpty title="Belum ada pertemuan" /></li>
      </ul>
    </KCard>

    <KCard :title="current ? `Pertemuan ${current.ke} · ${current.topik}` : 'Presensi'" :subtitle="current ? `${date(current.tanggal)} · dicatat oleh ${current.dosen_nama}` : undefined" flush>
      <template #actions>
        <template v-if="canEdit && current">
          <KButton size="sm" variant="secondary" @click="allHadir"><template #icon><CheckCheck class="size-4" /></template>Semua hadir</KButton>
          <KButton size="sm" :loading="busy" @click="save">Simpan</KButton>
        </template>
      </template>
      <KTable
        v-if="current"
        :columns="[{ key: 'mhs', label: 'Mahasiswa' }, { key: 'status', label: 'Kehadiran', align: 'right' }]"
        :rows="rows"
        :row-key="(r) => r.mahasiswa_id"
        :loading="loadingRows"
      >
        <template #cell-mhs="{ row }"><p class="font-semibold">{{ row.nama }}</p><p class="font-mono text-sm text-ink-3">{{ row.nim }}</p></template>
        <template #cell-status="{ row }">
          <div class="inline-flex gap-1" role="radiogroup" :aria-label="`Kehadiran ${row.nama}`">
            <button
              v-for="c in codes" :key="c" type="button" role="radio" :aria-checked="row.status === c" :disabled="!canEdit" :title="STATUS.presensi[c].label"
              :class="['size-8 rounded-pill text-xs font-bold transition-colors', row.status === c ? toneClass[STATUS.presensi[c].tone].soft + ' ring-1 ring-current' : 'bg-surface-2 text-ink-3 hover:text-ink']"
              @click="row.status = c"
            >{{ c }}</button>
          </div>
        </template>
      </KTable>
      <KEmpty v-else title="Pilih atau buka pertemuan" />
    </KCard>

    <KModal v-model="open" title="Buka pertemuan" size="sm">
      <div class="flex flex-col gap-4">
        <KField label="Tanggal" required><KInput v-model="form.tanggal" type="date" /></KField>
        <KField label="Topik" required><KInput v-model="form.topik" placeholder="Pengantar struktur data" /></KField>
      </div>
      <template #footer>
        <KButton variant="secondary" @click="open = false">Batal</KButton>
        <KButton :loading="busy" :disabled="!form.topik" @click="create">Buka pertemuan</KButton>
      </template>
    </KModal>
  </div>
</template>
