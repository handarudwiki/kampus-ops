<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Kelas } from '@/api/types'
import { useAsync } from '@/composables/useAsync'
import { slot } from '@/design-system'
import { useAuthStore } from '@/stores/auth'
import NilaiPanel from './kelas/NilaiPanel.vue'
import PresensiPanel from './kelas/PresensiPanel.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const id = route.params.id as string
type Tab = 'peserta' | 'presensi' | 'nilai'
const tab = ref<Tab>(route.path.endsWith('/nilai') ? 'nilai' : auth.activeRole === 'dosen' ? 'presensi' : 'peserta')

const kelas = useAsync<Kelas | null>(() => api.get<Kelas>(`/kelas/${id}`), null)
const peserta = useAsync<{ mahasiswa_id: string; nim: string; nama: string }[]>(() => api.get(`/kelas/${id}/peserta`), [])
const canEdit = computed(() => auth.activeRole === 'dosen')
</script>

<template>
  <div>
    <KPageHeader
      :title="kelas.data.value ? `${kelas.data.value.mk_nama} · ${kelas.data.value.nama}` : 'Kelas'"
      :description="kelas.data.value ? `${kelas.data.value.mk_kode} · ${kelas.data.value.sks} SKS · ${slot(kelas.data.value.slot)} · ${kelas.data.value.ruang_nama ?? 'Ruang belum ditentukan'}` : ''"
    >
      <template #eyebrow><KButton variant="ghost" size="sm" @click="router.back()"><template #icon><ArrowLeft class="size-4" /></template>Kembali</KButton></template>
      <template #actions>
        <KStatus v-if="kelas.data.value" domain="kelas" :value="kelas.data.value.status" />
      </template>
    </KPageHeader>

    <KAlert v-if="kelas.data.value && !kelas.data.value.dosen_id" level="error" class="mb-3" title="Kelas tanpa dosen pengampu">Error PV-03: tetapkan dosen di menu Jadwal & kelas sebelum ekspor PDDikti.</KAlert>

    <KSegmented v-model="tab" class="mb-3" :options="[{ value: 'peserta', label: 'Peserta', count: peserta.data.value.length }, { value: 'presensi', label: 'Presensi' }, { value: 'nilai', label: 'Nilai' }]" />

    <KCard v-if="tab === 'peserta'" flush>
      <KTable :columns="[{ key: 'nim', label: 'NIM', width: '140px' }, { key: 'nama', label: 'Nama' }]" :rows="peserta.data.value" :row-key="(r) => r.mahasiswa_id" :loading="peserta.loading.value">
        <template #cell-nim="{ value }"><span class="font-mono text-sm">{{ value }}</span></template>
      </KTable>
    </KCard>
    <PresensiPanel v-else-if="tab === 'presensi'" :kelas-id="id" :can-edit="canEdit" />
    <NilaiPanel v-else :kelas-id="id" :can-edit="canEdit" />
  </div>
</template>
