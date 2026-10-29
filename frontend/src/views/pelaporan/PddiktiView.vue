<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ArrowUpRight, Download, Play } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { ValidationResult } from '@/api/types'
import { useAction } from '@/composables/useAsync'
import { dateTime, toastError } from '@/design-system'
import { usePeriodeStore } from '@/stores/periode'
import { useAuthStore } from '@/stores/auth'

/** FR-29..31: validator with readiness score, every issue links to its fix page; export blocked on errors. */
const periode = usePeriodeStore()
const auth = useAuthStore()
const periodeId = ref('')
const result = ref<ValidationResult | null>(null)
const loading = ref(false)
const level = ref<'' | 'error' | 'warning'>('')
const rule = ref('')

onMounted(async () => {
  await Promise.all([periode.loadAll(), periode.load()])
  periodeId.value = periode.active?.id ?? periode.all[0]?.id ?? ''
})
watch(periodeId, async (id) => {
  if (!id) return
  loading.value = true
  try { result.value = await api.get<ValidationResult | null>('/pddikti/validation', { periode_id: id }) } catch (e) { toastError(e) } finally { loading.value = false }
})

const { run, busy } = useAction()
async function validate() {
  await run(async () => { result.value = await api.post<ValidationResult>('/pddikti/validate', { periode_id: periodeId.value }) }, 'Validasi selesai')
}

const items = computed(() => (result.value?.items ?? []).filter((i) => (!level.value || i.level === level.value) && (!rule.value || i.rule === rule.value)))
const scoreTone = computed(() => { const s = result.value?.score ?? 0; return s >= 90 ? 'green' : s >= 70 ? 'amber' : 'red' })
const blocked = computed(() => !result.value || result.value.errors > 0)

const ENTITIES = [
  { key: 'mahasiswa', label: 'Mahasiswa' }, { key: 'kelas', label: 'Kelas kuliah' }, { key: 'krs', label: 'KRS' },
  { key: 'nilai', label: 'Nilai' }, { key: 'aktivitas', label: 'Aktivitas kuliah' },
]
const exporting = ref<string | null>(null)
async function download(entity: string) {
  exporting.value = entity
  try {
    const blob = await api.blob(`/pddikti/export/${entity}`, { periode_id: periodeId.value })
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `pddikti_${entity}_${periode.all.find((p) => p.id === periodeId.value)?.kode ?? ''}.csv`
    a.click()
    URL.revokeObjectURL(a.href)
  } catch (e) { toastError(e, 'Ekspor gagal') } finally { exporting.value = null }
}
</script>

<template>
  <div>
    <KPageHeader title="Validator PDDikti" description="Aturan PV-01 s.d. PV-08 dijalankan sebelum ekspor. Error memblokir ekspor; peringatan tidak.">
      <template #actions>
        <KSelect v-model="periodeId" :options="periode.all.map((p) => ({ value: p.id, label: `${p.kode} · ${p.tipe} ${p.tahun}` }))" class="w-52" />
        <KButton :loading="busy" :disabled="!periodeId" @click="validate"><template #icon><Play class="size-4" /></template>Jalankan validator</KButton>
      </template>
    </KPageHeader>

    <div v-if="loading" class="h-60 animate-pulse rounded-card bg-surface" />
    <KCard v-else-if="!result"><KEmpty title="Belum pernah divalidasi" description="Jalankan validator untuk melihat skor kesiapan lapor periode ini." /></KCard>
    <template v-else>
      <div class="grid gap-3 lg:grid-cols-3">
        <KCard title="Skor kesiapan" :subtitle="`Dijalankan ${dateTime(result.ran_at, auth.tenant?.timezone)}`">
          <div class="flex items-center gap-6">
            <KRing :value="result.score" :tone="scoreTone" :size="132">
              <p class="tabular text-xl font-bold leading-none">{{ result.score }}</p>
              <p class="text-xs text-ink-3">dari 100</p>
            </KRing>
            <div class="flex flex-col gap-3">
              <KBadge tone="red" variant="dot"><b class="tabular text-ink">{{ result.errors }}</b> error</KBadge>
              <KBadge tone="amber" variant="dot"><b class="tabular text-ink">{{ result.warnings }}</b> peringatan</KBadge>
            </div>
          </div>
        </KCard>
        <KCard title="Per aturan" class="lg:col-span-2">
          <div class="grid gap-2 sm:grid-cols-2">
            <button
              v-for="r in result.by_rule" :key="r.rule" type="button"
              :class="['flex items-center gap-3 rounded-control border p-3 text-left transition-colors', rule === r.rule ? 'border-accent bg-accent-soft' : 'border-line hover:bg-surface-2']"
              @click="rule = rule === r.rule ? '' : r.rule"
            >
              <span class="font-mono text-xs font-bold text-ink-3">{{ r.rule }}</span>
              <span class="min-w-0 flex-1 truncate text-sm font-medium">{{ r.label }}</span>
              <KBadge :tone="r.count === 0 ? 'green' : r.level === 'error' ? 'red' : 'amber'">{{ r.count }}</KBadge>
            </button>
          </div>
        </KCard>
      </div>

      <KCard title="Ekspor format Feeder" class="mt-3">
        <KAlert v-if="blocked" level="error" class="mb-4" title="Ekspor diblokir">Perbaiki {{ result.errors }} error di bawah lalu jalankan validator lagi.</KAlert>
        <div class="flex flex-wrap gap-2">
          <KButton v-for="e in ENTITIES" :key="e.key" variant="secondary" :disabled="blocked" :loading="exporting === e.key" @click="download(e.key)">
            <template #icon><Download class="size-4" /></template>{{ e.label }}.csv
          </KButton>
        </div>
      </KCard>

      <KCard title="Temuan" class="mt-3" flush>
        <KToolbar>
          <KSegmented v-model="level" :options="[{ value: '', label: 'Semua', count: result.items.length }, { value: 'error', label: 'Error', count: result.errors }, { value: 'warning', label: 'Peringatan', count: result.warnings }]" />
          <template #right><KBadge v-if="rule" tone="blue">{{ rule }} <button class="ml-1" aria-label="Hapus filter" @click="rule = ''">×</button></KBadge></template>
        </KToolbar>
        <KTable
          :columns="[{ key: 'rule', label: 'Aturan', width: '90px' }, { key: 'level', label: 'Tingkat', hideMobile: true }, { key: 'entity_label', label: 'Record' }, { key: 'message', label: 'Masalah', hideMobile: true }, { key: 'fix', label: '', align: 'right' }]"
          :rows="items"
          empty-title="Tidak ada temuan"
          empty-description="Data periode ini siap dilaporkan."
        >
          <template #cell-rule="{ value }"><span class="font-mono text-sm font-semibold">{{ value }}</span></template>
          <template #cell-level="{ row }"><KStatus domain="level" :value="row.level" variant="dot" /></template>
          <template #cell-entity_label="{ row }"><p class="font-semibold">{{ row.entity_label }}</p><p class="text-sm text-ink-3 sm:hidden">{{ row.message }}</p></template>
          <template #cell-message="{ value }"><span class="text-sm text-ink-2">{{ value }}</span></template>
          <template #cell-fix="{ row }"><KButton size="sm" variant="soft" :to="row.fix_url">Perbaiki <ArrowUpRight class="size-4" /></KButton></template>
        </KTable>
      </KCard>
    </template>
  </div>
</template>
