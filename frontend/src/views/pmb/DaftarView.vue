<script setup lang="ts">
import { ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { CheckCircle2 } from 'lucide-vue-next'
import { api, ApiError } from '@/api/client'
import type { Gelombang, Pendaftar, Prodi, Tenant } from '@/api/types'
import { errorText } from '@/design-system'
import { useAuthStore } from '@/stores/auth'

/** Public PMB registration. Options come from the unauthenticated /pmb/public-options for the selected campus. */
const auth = useAuthStore()
const tenants = ref<Tenant[]>([])
const gelombang = ref<Gelombang[]>([])
const prodi = ref<Prodi[]>([])
const f = ref({ tenant_code: 'UNIKA', nama: '', email: '', nik: '', gelombang_id: '', prodi_pilihan_id: '', password: '' })
const err = ref<string | null>(null)
const busy = ref(false)
const done = ref<Pendaftar | null>(null)

api.get<Tenant[]>('/tenants/public').then((t) => (tenants.value = t)).catch(() => {})
watch(() => f.value.tenant_code, async (code) => {
  try {
    const o = await api.get<{ gelombang: Gelombang[]; prodi: Prodi[] }>('/pmb/public-options', { tenant_code: code })
    gelombang.value = o.gelombang
    prodi.value = o.prodi
  } catch {
    gelombang.value = []
    prodi.value = []
  }
}, { immediate: true })

async function submit() {
  err.value = null
  busy.value = true
  try {
    done.value = await api.post<Pendaftar>('/pmb/register', f.value)
    await auth.login(f.value.tenant_code, f.value.email, f.value.password).catch(() => {})
  } catch (e) {
    err.value = e instanceof ApiError ? errorText(e.code, e.message) : String(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="mx-auto flex min-h-dvh max-w-xl flex-col justify-center p-4 sm:p-8">
    <KCard v-if="done">
      <div class="flex flex-col items-center gap-3 py-6 text-center">
        <CheckCircle2 class="size-12 text-green" />
        <p class="text-lg font-bold">Pendaftaran berhasil</p>
        <p class="text-ink-2">No. pendaftaran <b class="font-mono">{{ done.no_daftar }}</b>. Langkah berikutnya: bayar formulir.</p>
        <KButton to="/">Lihat status pendaftaran</KButton>
      </div>
    </KCard>
    <template v-else>
      <h1 class="text-xl font-extrabold tracking-tight">Daftar mahasiswa baru</h1>
      <p class="mb-6 mt-1 text-ink-3">Isi data diri. Setelah mendaftar, Anda bisa memantau status seleksi.</p>
      <KCard>
        <form class="grid gap-4 sm:grid-cols-2" @submit.prevent="submit">
          <KField label="Kampus" required class="sm:col-span-2"><KSelect v-model="f.tenant_code" :options="tenants.length ? tenants.map((t) => ({ value: t.code, label: t.name })) : [{ value: 'UNIKA', label: 'UNIKA' }]" /></KField>
          <KField label="Nama lengkap" required class="sm:col-span-2"><KInput v-model="f.nama" required /></KField>
          <KField label="Email" required><KInput v-model="f.email" type="email" required /></KField>
          <KField label="NIK" required hint="16 digit"><KInput v-model="f.nik" inputmode="numeric" maxlength="16" pattern="\d{16}" required /></KField>
          <KField label="Gelombang" required><KSelect v-model="f.gelombang_id" :options="gelombang.map((g) => ({ value: g.id, label: g.nama }))" placeholder="Pilih…" required /></KField>
          <KField label="Prodi pilihan" required><KSelect v-model="f.prodi_pilihan_id" :options="prodi.map((p) => ({ value: p.id, label: `${p.jenjang} ${p.nama}` }))" placeholder="Pilih…" required /></KField>
          <KField label="Kata sandi" required class="sm:col-span-2"><KInput v-model="f.password" type="password" minlength="8" required /></KField>
          <KAlert v-if="err" level="error" class="sm:col-span-2">{{ err }}</KAlert>
          <KButton type="submit" :loading="busy" block class="sm:col-span-2">Daftar</KButton>
        </form>
      </KCard>
      <p class="mt-6 text-center text-sm text-ink-3">Sudah punya akun? <RouterLink to="/login" class="font-semibold text-accent hover:underline">Masuk</RouterLink></p>
    </template>
  </div>
</template>
