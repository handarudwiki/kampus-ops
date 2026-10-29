<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Pencil, RefreshCw } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Mahasiswa, RiwayatStatus, StatusMhs } from '@/api/types'
import { useAction, useAsync } from '@/composables/useAsync'
import { date, ip, statusOptions } from '@/design-system'

const route = useRoute()
const router = useRouter()
const id = route.params.id as string
const mhs = useAsync<Mahasiswa | null>(() => api.get<Mahasiswa>(`/mahasiswa/${id}`), null)
const riwayat = useAsync<RiwayatStatus[]>(() => api.get<RiwayatStatus[]>(`/mahasiswa/${id}/riwayat-status`), [])
const { run, busy } = useAction()

// Quick fix for PDDikti issues (PV-05 NIK)
const editOpen = ref(false)
const edit = ref({ nik: '', email: '' })
function openEdit() {
  edit.value = { nik: mhs.data.value?.nik ?? '', email: mhs.data.value?.email ?? '' }
  editOpen.value = true
}
const nikOk = () => /^\d{16}$/.test(edit.value.nik)
async function saveEdit() {
  if (await run(() => api.put(`/mahasiswa/${id}`, { ...mhs.data.value, ...edit.value }), 'Data mahasiswa diperbarui')) {
    editOpen.value = false
    mhs.reload()
  }
}

// Status change (EC-08 cuti limit enforced by backend)
const stOpen = ref(false)
const st = ref<{ status: StatusMhs; alasan: string }>({ status: 'aktif', alasan: '' })
async function saveStatus() {
  if (await run(() => api.post(`/mahasiswa/${id}/status`, st.value), 'Status diperbarui')) {
    stOpen.value = false
    mhs.reload()
    riwayat.reload()
  }
}
</script>

<template>
  <div>
    <KPageHeader :title="mhs.data.value?.nama ?? 'Mahasiswa'" :description="mhs.data.value ? `${mhs.data.value.nim} · ${mhs.data.value.prodi_nama} · angkatan ${mhs.data.value.angkatan}` : ''">
      <template #eyebrow><KButton variant="ghost" size="sm" @click="router.back()"><template #icon><ArrowLeft class="size-4" /></template>Kembali</KButton></template>
      <template #actions>
        <KButton variant="secondary" @click="stOpen = true"><template #icon><RefreshCw class="size-4" /></template>Ubah status</KButton>
        <KButton @click="openEdit"><template #icon><Pencil class="size-4" /></template>Perbaiki data</KButton>
      </template>
    </KPageHeader>

    <KSkeleton v-if="mhs.loading.value && !mhs.data.value" :lines="5" />
    <div v-else-if="mhs.data.value" class="grid gap-3 lg:grid-cols-3">
      <KAlert v-if="!mhs.data.value.nik || !/^\d{16}$/.test(mhs.data.value.nik)" level="error" title="NIK tidak valid (PV-05)" class="lg:col-span-3">
        NIK kosong atau bukan 16 digit. Ekspor PDDikti diblokir sampai diperbaiki.
        <template #action><KButton size="sm" @click="openEdit">Perbaiki</KButton></template>
      </KAlert>
      <div class="grid gap-3 sm:grid-cols-3 lg:col-span-3">
        <KStatCard label="IPK" :value="ip(mhs.data.value.ipk)" tone="green" />
        <KStatCard label="SKS lulus" :value="String(mhs.data.value.sks_lulus)" tone="blue" />
        <KStatCard label="Semester ke" :value="String(mhs.data.value.semester_ke)" tone="violet" />
      </div>
      <KCard title="Data induk" class="lg:col-span-2">
        <template #actions><KStatus domain="mahasiswa" :value="mhs.data.value.status" /></template>
        <KKeyValue :items="[
          { label: 'NIM', value: mhs.data.value.nim },
          { label: 'NIK', value: mhs.data.value.nik },
          { label: 'Email', value: mhs.data.value.email },
          { label: 'Prodi', value: mhs.data.value.prodi_nama },
          { label: 'Angkatan', value: mhs.data.value.angkatan },
          { label: 'Dosen wali', value: mhs.data.value.dosen_wali_nama },
        ]" />
      </KCard>
      <KCard title="Riwayat status">
        <ol class="relative flex flex-col gap-4 border-l border-line pl-5">
          <li v-for="r in riwayat.data.value" :key="r.periode_kode + r.created_at" class="relative">
            <span class="absolute -left-[26px] top-1 size-3 rounded-pill border-2 border-surface bg-accent" />
            <div class="flex items-center gap-2"><span class="font-mono text-sm font-semibold">{{ r.periode_kode }}</span><KStatus domain="mahasiswa" :value="r.status" /></div>
            <p class="mt-0.5 text-sm text-ink-3">{{ r.alasan ?? 'Tanpa catatan' }} · {{ date(r.created_at) }}</p>
          </li>
        </ol>
      </KCard>
    </div>

    <KModal v-model="editOpen" title="Perbaiki data mahasiswa" size="sm">
      <div class="flex flex-col gap-4">
        <KField label="NIK" required :error="edit.nik && !nikOk() ? 'NIK harus 16 digit angka' : null"><KInput v-model="edit.nik" inputmode="numeric" maxlength="16" :invalid="!!edit.nik && !nikOk()" /></KField>
        <KField label="Email" required><KInput v-model="edit.email" type="email" /></KField>
      </div>
      <template #footer>
        <KButton variant="secondary" @click="editOpen = false">Batal</KButton>
        <KButton :loading="busy" :disabled="!nikOk()" @click="saveEdit">Simpan</KButton>
      </template>
    </KModal>

    <KModal v-model="stOpen" title="Ubah status mahasiswa" description="Dicatat di riwayat status periode aktif. Cuti dibatasi sesuai aturan kampus." size="sm">
      <div class="flex flex-col gap-4">
        <KField label="Status baru" required><KSelect v-model="st.status" :options="statusOptions('mahasiswa')" /></KField>
        <KField label="Alasan" required><KTextarea v-model="st.alasan" /></KField>
      </div>
      <template #footer>
        <KButton variant="secondary" @click="stOpen = false">Batal</KButton>
        <KButton :loading="busy" :disabled="!st.alasan.trim()" @click="saveStatus">Simpan</KButton>
      </template>
    </KModal>
  </div>
</template>
