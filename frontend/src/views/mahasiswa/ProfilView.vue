<script setup lang="ts">
import { api } from '@/api/client'
import type { Mahasiswa, RiwayatStatus } from '@/api/types'
import { useAsync } from '@/composables/useAsync'
import { date, ip } from '@/design-system'

const { data, loading } = useAsync<{ mahasiswa: Mahasiswa; riwayat: RiwayatStatus[] } | null>(() => api.get('/akademik/me/profil'), null)
</script>

<template>
  <div>
    <KPageHeader title="Profil akademik" />
    <KSkeleton v-if="loading && !data" :lines="5" />
    <div v-else-if="data" class="grid gap-3 lg:grid-cols-3">
      <KCard class="lg:col-span-2">
        <div class="mb-6 flex items-center gap-4">
          <KAvatar :name="data.mahasiswa.nama" size="lg" />
          <div>
            <p class="text-lg font-bold">{{ data.mahasiswa.nama }}</p>
            <p class="text-ink-3"><span class="font-mono">{{ data.mahasiswa.nim }}</span> · {{ data.mahasiswa.prodi_nama }}</p>
          </div>
          <KStatus class="ml-auto" domain="mahasiswa" :value="data.mahasiswa.status" />
        </div>
        <KKeyValue
          :items="[
            { label: 'Angkatan', value: data.mahasiswa.angkatan },
            { label: 'Semester ke', value: data.mahasiswa.semester_ke },
            { label: 'IPK', value: ip(data.mahasiswa.ipk) },
            { label: 'SKS lulus', value: data.mahasiswa.sks_lulus },
            { label: 'Dosen wali', value: data.mahasiswa.dosen_wali_nama },
            { label: 'Email', value: data.mahasiswa.email },
            { label: 'NIK', value: data.mahasiswa.nik },
          ]"
        />
      </KCard>
      <KCard title="Riwayat status" subtitle="Dasar hitung masa studi dan cuti">
        <ol class="relative flex flex-col gap-4 border-l border-line pl-5">
          <li v-for="r in data.riwayat" :key="r.periode_kode + r.created_at" class="relative">
            <span class="absolute -left-[26px] top-1 size-3 rounded-pill border-2 border-surface bg-accent" />
            <div class="flex items-center gap-2"><span class="font-mono text-sm font-semibold">{{ r.periode_kode }}</span><KStatus domain="mahasiswa" :value="r.status" /></div>
            <p class="mt-0.5 text-sm text-ink-3">{{ r.alasan ?? 'Tanpa catatan' }} · {{ date(r.created_at) }}</p>
          </li>
        </ol>
      </KCard>
    </div>
  </div>
</template>
