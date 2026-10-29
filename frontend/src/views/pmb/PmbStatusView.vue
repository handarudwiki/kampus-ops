<script setup lang="ts">
import { computed } from 'vue'
import { CheckCircle2, Circle } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Pendaftar } from '@/api/types'
import { useAsync } from '@/composables/useAsync'
import { date } from '@/design-system'
import { PMB_STEPS } from './steps'

const { data: p, loading } = useAsync<Pendaftar | null>(() => api.get<Pendaftar>('/pmb/me'), null)
const current = computed(() => (p.value?.status === 'tidak_lulus' ? 'lulus' : p.value?.status ?? 'daftar'))
</script>

<template>
  <div>
    <KPageHeader title="Status pendaftaran" description="Pantau tahapan seleksi penerimaan mahasiswa baru." />
    <KSkeleton v-if="loading && !p" :lines="4" />
    <div v-else-if="p" class="grid gap-3 lg:grid-cols-3">
      <KCard title="Tahapan" class="lg:col-span-3">
        <KStepper :steps="PMB_STEPS" :current="current" :failed="p.status === 'tidak_lulus'" />
        <KAlert v-if="p.status === 'tidak_lulus'" level="error" class="mt-5" title="Tidak lulus seleksi">Terima kasih telah mendaftar. Anda dapat mendaftar kembali di gelombang berikutnya.</KAlert>
        <KAlert v-else-if="p.nim" level="success" class="mt-5" :title="`NIM Anda: ${p.nim}`">Selamat! Daftar ulang selesai. Masuk kembali memakai NIM untuk mengakses portal mahasiswa.</KAlert>
        <KAlert v-else-if="p.status === 'lulus'" level="info" class="mt-5" title="Lulus seleksi">Silakan lakukan daftar ulang di bagian akademik untuk menerbitkan NIM.</KAlert>
      </KCard>
      <KCard title="Data pendaftar" class="lg:col-span-2">
        <KKeyValue
          :items="[
            { label: 'No. pendaftaran', value: p.no_daftar },
            { label: 'Nama', value: p.nama },
            { label: 'Email', value: p.email },
            { label: 'NIK', value: p.nik },
            { label: 'Gelombang', value: p.gelombang_nama },
            { label: 'Prodi pilihan', value: p.prodi_pilihan_nama },
            { label: 'Tanggal daftar', value: date(p.created_at) },
            { key: 'status', label: 'Status' },
          ]"
        >
          <template #value-status><KStatus domain="pmb" :value="p.status" /></template>
        </KKeyValue>
      </KCard>
      <KCard title="Kelengkapan berkas">
        <ul class="flex flex-col gap-3">
          <li v-for="b in p.berkas" :key="b.nama" class="flex items-center gap-3">
            <CheckCircle2 v-if="b.ok" class="size-5 text-green" />
            <Circle v-else class="size-5 text-ink-3" />
            <span :class="b.ok ? 'font-semibold' : 'text-ink-2'">{{ b.nama }}</span>
          </li>
        </ul>
      </KCard>
    </div>
    <KEmpty v-else title="Data pendaftaran tidak ditemukan" />
  </div>
</template>
