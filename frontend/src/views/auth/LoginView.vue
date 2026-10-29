<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { ArrowRight, KeyRound, ShieldCheck, Zap } from 'lucide-vue-next'
import { api, ApiError } from '@/api/client'
import type { Tenant } from '@/api/types'
import { errorText } from '@/design-system'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const tenants = ref<Tenant[]>([])
const tenant = ref('UNIKA')
const identifier = ref('')
const password = ref('')
const error = ref<string | null>(null)
const busy = ref(false)

const DEMO = [
  { label: 'Mahasiswa', id: '2024110001' },
  { label: 'Mhs belum bayar', id: '2024110002' },
  { label: 'Dosen · Wali · Kaprodi', id: '0011223344' },
  { label: 'Dosen', id: '0055667788' },
  { label: 'Operator BAAK', id: 'baak@unika.ac.id' },
  { label: 'Keuangan', id: 'keuangan@unika.ac.id' },
  { label: 'Admin kampus', id: 'admin@unika.ac.id' },
  { label: 'Super admin', id: 'super@kampusops.id' },
  { label: 'Calon mahasiswa', id: 'calon@mail.com' },
]

onMounted(async () => {
  try { tenants.value = await api.get<Tenant[]>('/tenants/public') } catch { tenants.value = [] }
})

function fill(id: string) {
  tenant.value = 'UNIKA'
  identifier.value = id
  password.value = 'password123'
}

async function submit() {
  error.value = null
  busy.value = true
  try {
    await auth.login(tenant.value, identifier.value.trim(), password.value)
    router.replace((route.query.next as string) || '/')
  } catch (e) {
    error.value = e instanceof ApiError ? errorText(e.code, e.message) : String(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="grid min-h-dvh lg:grid-cols-[1.1fr_1fr]">
    <!-- Brand panel -->
    <section class="relative hidden flex-col justify-between overflow-hidden bg-surface p-10 lg:flex">
      <div class="flex items-center gap-2.5">
        <span class="relative block size-8">
          <span class="absolute left-0 top-0 size-4 rounded-pill bg-blue" />
          <span class="absolute right-0 top-0.5 size-3.5 rounded-pill bg-green" />
          <span class="absolute bottom-0 left-2 size-4 rounded-pill bg-amber" />
        </span>
        <span class="text-md font-extrabold">KampusOps</span>
      </div>
      <div class="max-w-lg">
        <h1 class="text-2xl font-extrabold tracking-tight">Satu sumber data akademik dan keuangan kampus.</h1>
        <p class="mt-4 text-md text-ink-2">Dari PMB sampai lapor PDDikti. Aturan akademik ditegakkan sistem, bukan dicek manual.</p>
        <div class="mt-8 grid gap-3">
          <div class="flex gap-3 rounded-control bg-blue-soft p-4">
            <Zap class="mt-0.5 size-5 text-blue-ink" />
            <div><p class="font-bold">KRS tanpa overbooking</p><p class="text-sm text-ink-2">Kuota kelas diupdate atomik, aman untuk ratusan request bersamaan.</p></div>
          </div>
          <div class="flex gap-3 rounded-control bg-green-soft p-4">
            <ShieldCheck class="mt-0.5 size-5 text-green-ink" />
            <div><p class="font-bold">Siap lapor PDDikti</p><p class="text-sm text-ink-2">Validator menunjuk record yang salah sebelum ekspor.</p></div>
          </div>
          <div class="flex gap-3 rounded-control bg-pink-soft p-4">
            <KeyRound class="mt-0.5 size-5 text-pink-ink" />
            <div><p class="font-bold">Multi-kampus, multi-peran</p><p class="text-sm text-ink-2">Data tiap kampus terisolasi; satu akun bisa punya banyak peran.</p></div>
          </div>
        </div>
      </div>
      <p class="text-sm text-ink-3">Demo portofolio · data sintetis</p>
    </section>

    <!-- Form -->
    <section class="flex items-center justify-center p-4 sm:p-10">
      <div class="w-full max-w-md">
        <h2 class="text-xl font-extrabold tracking-tight">Masuk</h2>
        <p class="mt-1 text-ink-3">Gunakan email, NIM, atau NIDN.</p>

        <form class="mt-8 flex flex-col gap-4" @submit.prevent="submit">
          <KField label="Kampus" for="tenant">
            <KSelect
              id="tenant"
              v-model="tenant"
              :options="tenants.length ? tenants.map((t) => ({ value: t.code, label: `${t.name} (${t.code})` })) : [{ value: 'UNIKA', label: 'UNIKA' }, { value: 'POLTEK', label: 'POLTEK' }]"
            />
          </KField>
          <KField label="Email / NIM / NIDN" for="identifier">
            <KInput id="identifier" v-model="identifier" autocomplete="username" required autofocus />
          </KField>
          <KField label="Kata sandi" for="password">
            <KInput id="password" v-model="password" type="password" autocomplete="current-password" required />
          </KField>
          <KAlert v-if="error" level="error">{{ error }}</KAlert>
          <KButton type="submit" :loading="busy" block>Masuk <ArrowRight class="size-4" /></KButton>
        </form>

        <div class="mt-8">
          <p class="ko-eyebrow mb-3">Akun demo · kata sandi password123</p>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="d in DEMO" :key="d.id" type="button"
              class="rounded-pill border border-line bg-surface px-3 py-1.5 text-sm font-semibold text-ink-2 transition-colors hover:border-line-strong hover:text-ink"
              @click="fill(d.id)"
            >{{ d.label }}</button>
          </div>
        </div>

        <p class="mt-8 text-sm text-ink-3">
          Calon mahasiswa baru?
          <RouterLink to="/daftar" class="font-semibold text-accent hover:underline">Daftar PMB</RouterLink>
        </p>
      </div>
    </section>
  </div>
</template>
