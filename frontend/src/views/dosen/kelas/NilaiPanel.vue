<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Lock, PencilLine, Save } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { NilaiRow } from '@/api/types'
import { useAction } from '@/composables/useAsync'
import { confirm, date, toastError } from '@/design-system'
import { useNilaiStore } from '@/stores/nilai'

/** FR-22/23 grade entry; after finalize only corrections via kaprodi (EC-11). */
const props = defineProps<{ kelasId: string; canEdit: boolean }>()
const store = useNilaiStore()
const loading = ref(true)
onMounted(() => store.load(props.kelasId).catch((e) => toastError(e)).finally(() => (loading.value = false)))

const { run, busy } = useAction()
const save = () => run(() => store.save(), 'Nilai disimpan')
async function finalize() {
  const r = await confirm({ title: 'Kunci nilai kelas ini?', body: 'Setelah dikunci, KHS/IPS/IPK dihitung dan perubahan hanya lewat koreksi nilai yang disetujui kaprodi.', confirmLabel: 'Kunci nilai', danger: true })
  if (!r.ok) return
  if (store.dirty) await store.save()
  await run(() => store.finalize(), 'Nilai dikunci')
}
function onInput(row: string, key: string, e: Event) {
  const raw = (e.target as HTMLInputElement).value
  const v = raw === '' ? null : Math.max(0, Math.min(100, Number(raw)))
  store.set(row, key, v)
}

// Correction request
const koreksi = ref<{ row: NilaiRow; komponen: Record<string, number | null> } | null>(null)
const alasan = ref('')
const kOpen = ref(false)
function openKoreksi(row: NilaiRow) {
  koreksi.value = { row, komponen: { ...row.komponen } }
  alasan.value = ''
  kOpen.value = true
}
async function submitKoreksi() {
  const k = koreksi.value!
  if (await run(() => api.post(`/nilai/${k.row.nilai_id}/koreksi`, { komponen: k.komponen, alasan: alasan.value }), 'Koreksi diajukan ke kaprodi')) {
    kOpen.value = false
    store.load(props.kelasId)
  }
}
</script>

<template>
  <KCard flush>
    <KToolbar>
      <template v-if="store.server">
        <KBadge v-for="k in store.server.komponen" :key="k.key" tone="blue">{{ k.label }} {{ k.bobot_persen }}%</KBadge>
        <span class="text-sm text-ink-3">Batas input {{ date(store.server.deadline) }}</span>
      </template>
      <template #right>
        <KBadge v-if="store.server?.finalized" tone="green"><Lock class="size-3" />Terkunci</KBadge>
        <template v-else-if="canEdit && store.server">
          <KButton size="sm" variant="secondary" :disabled="!store.dirty" :loading="busy" @click="save"><template #icon><Save class="size-4" /></template>Simpan</KButton>
          <KButton size="sm" :disabled="busy" @click="finalize"><template #icon><Lock class="size-4" /></template>Kunci nilai</KButton>
        </template>
      </template>
    </KToolbar>
    <KTable
      :columns="[
        { key: 'mhs', label: 'Mahasiswa' },
        ...(store.server?.komponen ?? []).map((k) => ({ key: 'k_' + k.key, label: k.label, align: 'center' as const, width: '96px' })),
        { key: 'angka', label: 'Angka', align: 'right', width: '80px' },
        { key: 'huruf', label: 'Huruf', align: 'center', width: '72px' },
        { key: 'aksi', label: '', align: 'right', width: '80px' },
      ]"
      :rows="store.server?.rows ?? []"
      :row-key="(r) => r.krs_detail_id"
      :loading="loading"
      empty-title="Belum ada peserta"
    >
      <template #cell-mhs="{ row }"><p class="font-semibold">{{ row.nama }}</p><p class="font-mono text-sm text-ink-3">{{ row.nim }}</p></template>
      <template v-for="k in store.server?.komponen ?? []" :key="k.key" #[`cell-k_${k.key}`]="{ row }">
        <input
          v-if="!store.server?.finalized && canEdit"
          type="number" min="0" max="100" inputmode="decimal"
          :value="store.draft[row.krs_detail_id]?.[k.key] ?? ''"
          :aria-label="`${k.label} ${row.nama}`"
          class="tabular h-9 w-20 rounded-control border border-line bg-surface-2 text-center focus:border-accent focus:bg-surface focus:outline-none focus:ring-4 focus:ring-accent/15"
          @input="onInput(row.krs_detail_id, k.key, $event)"
        />
        <span v-else class="tabular">{{ row.komponen[k.key] ?? '—' }}</span>
      </template>
      <template #cell-angka="{ row }"><span class="tabular font-semibold">{{ store.server?.finalized ? row.angka ?? '—' : store.preview(row) ?? '—' }}</span></template>
      <template #cell-huruf="{ row }"><span class="font-bold">{{ row.huruf ?? '—' }}</span></template>
      <template #cell-aksi="{ row }">
        <KBadge v-if="row.koreksi_pending" tone="amber">Koreksi</KBadge>
        <KIconButton v-else-if="store.server?.finalized && canEdit && row.nilai_id" label="Ajukan koreksi" @click="openKoreksi(row)"><PencilLine class="size-4" /></KIconButton>
      </template>
    </KTable>

    <KModal v-model="kOpen" title="Ajukan koreksi nilai" :description="koreksi ? `${koreksi.row.nama} · ${koreksi.row.nim}` : ''">
      <div v-if="koreksi" class="flex flex-col gap-4">
        <div class="grid grid-cols-3 gap-3">
          <KField v-for="k in store.server?.komponen ?? []" :key="k.key" :label="`${k.label} (${k.bobot_persen}%)`">
            <KInput v-model.number="koreksi.komponen[k.key]" type="number" min="0" max="100" />
          </KField>
        </div>
        <KField label="Alasan dan bukti" required hint="Disetujui kaprodi; IPS/IPK dihitung ulang dan record ditandai perlu lapor ulang bila periode sudah diekspor.">
          <KTextarea v-model="alasan" />
        </KField>
      </div>
      <template #footer>
        <KButton variant="secondary" @click="kOpen = false">Batal</KButton>
        <KButton :loading="busy" :disabled="!alasan.trim()" @click="submitKoreksi">Ajukan</KButton>
      </template>
    </KModal>
  </KCard>
</template>
