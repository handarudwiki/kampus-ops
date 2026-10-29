<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { api } from '@/api/client'
import type { AuditEntry } from '@/api/types'
import { useAsync } from '@/composables/useAsync'
import { ROLE_LABEL } from '@/config/roles'
import { dateTime } from '@/design-system'
import { useAuthStore } from '@/stores/auth'

/** FR-32: append-only audit trail — who, when, before, after. */
const auth = useAuthStore()
const q = ref('')
const entitas = ref('')
const { data, loading, reload } = useAsync<AuditEntry[]>(() => api.get<AuditEntry[]>('/audit-log', { q: q.value, entitas: entitas.value }), [])
watch([q, entitas], () => { page.value = 1; reload() })
const page = ref(1)
const PER = 20
const paged = computed(() => data.value.slice((page.value - 1) * PER, page.value * PER))
const ENTITAS = ['krs', 'nilai', 'koreksi_nilai', 'kelas', 'mahasiswa', 'tagihan', 'pembayaran', 'dispensasi', 'periode', 'pendaftar_pmb'].map((v) => ({ value: v, label: v }))

const open = ref(false)
const sel = ref<AuditEntry | null>(null)
function show(e: AuditEntry) { sel.value = e; open.value = true }

/** Flatten before/after into one row per key, marking changes. */
const diff = computed(() => {
  const b = (sel.value?.before ?? {}) as Record<string, unknown>
  const a = (sel.value?.after ?? {}) as Record<string, unknown>
  const keys = [...new Set([...Object.keys(b ?? {}), ...Object.keys(a ?? {})])]
  return keys.map((k) => {
    const before = JSON.stringify(b?.[k] ?? null)
    const after = JSON.stringify(a?.[k] ?? null)
    return { key: k, before, after, changed: before !== after }
  }).sort((x, y) => Number(y.changed) - Number(x.changed))
})
const aksiTone = (a: string) => (a.includes('delete') || a.includes('reject') || a.includes('cancel') ? 'red' : a.includes('create') || a.includes('approve') ? 'green' : 'blue')
</script>

<template>
  <div>
    <KPageHeader title="Audit log" description="Catatan append-only untuk semua perubahan data akademik dan keuangan." />
    <KCard flush>
      <KToolbar>
        <KSearch v-model="q" placeholder="Cari aktor / ID record" />
        <KSelect v-model="entitas" :options="ENTITAS" placeholder="Semua entitas" class="w-full sm:w-48" />
      </KToolbar>
      <KTable
        :columns="[{ key: 'created_at', label: 'Waktu', width: '170px' }, { key: 'actor', label: 'Aktor' }, { key: 'aksi', label: 'Aksi' }, { key: 'entitas', label: 'Entitas', hideMobile: true }, { key: 'request_id', label: 'Request', hideMobile: true }]"
        :rows="paged"
        :loading="loading"
        clickable
        empty-title="Belum ada aktivitas"
        @row-click="show"
      >
        <template #cell-created_at="{ value }"><span class="tabular text-sm text-ink-2">{{ dateTime(value, auth.tenant?.timezone) }}</span></template>
        <template #cell-actor="{ row }"><p class="font-semibold">{{ row.actor_nama }}</p><p class="text-xs text-ink-3">{{ ROLE_LABEL[row.actor_role] ?? row.actor_role }}</p></template>
        <template #cell-aksi="{ value }"><KBadge :tone="aksiTone(value)">{{ value }}</KBadge></template>
        <template #cell-entitas="{ row }"><span class="font-mono text-sm">{{ row.entitas }}</span><span class="font-mono text-xs text-ink-3"> #{{ row.entitas_id.slice(0, 8) }}</span></template>
        <template #cell-request_id="{ value }"><span class="font-mono text-xs text-ink-3">{{ value?.slice(0, 12) }}</span></template>
      </KTable>
      <KPagination v-if="data.length > PER" v-model="page" :total="data.length" :per-page="PER" />
    </KCard>

    <KModal v-model="open" :title="sel ? `${sel.aksi} · ${sel.entitas}` : ''" :description="sel ? `${sel.actor_nama} · ${dateTime(sel.created_at, auth.tenant?.timezone)} · request ${sel.request_id}` : ''" size="lg">
      <KTable :columns="[{ key: 'key', label: 'Kolom' }, { key: 'before', label: 'Sebelum' }, { key: 'after', label: 'Sesudah' }]" :rows="diff" :row-key="(r) => r.key" empty-title="Tidak ada data">
        <template #cell-key="{ row }"><span :class="['font-mono text-sm', row.changed ? 'font-bold' : 'text-ink-3']">{{ row.key }}</span></template>
        <template #cell-before="{ row }"><code :class="['break-all font-mono text-xs', row.changed && 'rounded bg-red-soft px-1 text-red-ink']">{{ row.before }}</code></template>
        <template #cell-after="{ row }"><code :class="['break-all font-mono text-xs', row.changed && 'rounded bg-green-soft px-1 text-green-ink']">{{ row.after }}</code></template>
      </KTable>
    </KModal>
  </div>
</template>
