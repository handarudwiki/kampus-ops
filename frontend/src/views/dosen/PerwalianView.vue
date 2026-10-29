<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Check } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Krs, KrsStatus } from '@/api/types'
import { useAction, useAsync } from '@/composables/useAsync'
import { ip, relative } from '@/design-system'

/** Dosen wali queue. Goal from BRD: process one KRS in < 30 seconds, so approve is one click from the list. */
const router = useRouter()
const status = ref<KrsStatus | ''>('diajukan')
const q = ref('')
const { data, loading, reload } = useAsync<Krs[]>(() => api.get<Krs[]>('/krs/advisees', { status: status.value }), [])
watch(status, reload)
const { run, busy } = useAction()
const acting = ref<string | null>(null)

async function approve(k: Krs) {
  acting.value = k.id
  if (await run(() => api.post(`/krs/${k.id}/approve`, {}), `KRS ${k.mahasiswa.nama} disetujui`)) reload()
  acting.value = null
}
const filtered = (rows: Krs[]) => rows.filter((k) => !q.value || `${k.mahasiswa.nim} ${k.mahasiswa.nama}`.toLowerCase().includes(q.value.toLowerCase()))
</script>

<template>
  <div>
    <KPageHeader title="Persetujuan KRS" description="Mahasiswa bimbingan Anda. KRS yang diajukan tetap bisa diproses sampai H+3 setelah jendela tutup." />
    <KCard flush>
      <KToolbar>
        <KSegmented v-model="status" :options="[{ value: 'diajukan', label: 'Menunggu' }, { value: 'disetujui', label: 'Disetujui' }, { value: 'ditolak', label: 'Ditolak' }, { value: 'draft', label: 'Draft' }, { value: '', label: 'Semua' }]" />
        <template #right><KSearch v-model="q" placeholder="Cari NIM / nama" /></template>
      </KToolbar>
      <KTable
        :columns="[
          { key: 'mhs', label: 'Mahasiswa' },
          { key: 'ipk', label: 'IPK', align: 'right', hideMobile: true },
          { key: 'sks', label: 'SKS', align: 'right' },
          { key: 'submitted', label: 'Diajukan', hideMobile: true },
          { key: 'status', label: 'Status', hideMobile: true },
          { key: 'aksi', label: '', align: 'right' },
        ]"
        :rows="filtered(data)"
        :loading="loading"
        clickable
        empty-title="Tidak ada KRS"
        empty-description="Belum ada KRS dengan status ini."
        @row-click="(k) => router.push(`/krs/${k.id}`)"
      >
        <template #cell-mhs="{ row }">
          <div class="flex items-center gap-3">
            <KAvatar :name="row.mahasiswa.nama" size="sm" />
            <div><p class="font-semibold">{{ row.mahasiswa.nama }}</p><p class="font-mono text-sm text-ink-3">{{ row.mahasiswa.nim }}</p></div>
          </div>
        </template>
        <template #cell-ipk="{ row }"><span class="tabular">{{ ip(row.mahasiswa.ipk) }}</span></template>
        <template #cell-sks="{ row }"><span class="tabular font-semibold" :class="row.total_sks > row.max_sks && 'text-red-ink'">{{ row.total_sks }}/{{ row.max_sks }}</span></template>
        <template #cell-submitted="{ row }"><span class="text-sm text-ink-2">{{ row.submitted_at ? relative(row.submitted_at) : '—' }}</span></template>
        <template #cell-status="{ row }"><KStatus domain="krs" :value="row.status" variant="dot" /></template>
        <template #cell-aksi="{ row }">
          <KButton v-if="row.status === 'diajukan'" size="sm" :loading="busy && acting === row.id" @click.stop="approve(row)">
            <template #icon><Check class="size-4" /></template>Setujui
          </KButton>
        </template>
      </KTable>
    </KCard>
  </div>
</template>
