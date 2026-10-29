<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { ArrowRight, Eye, EyeOff, ShieldCheck, Zap } from 'lucide-vue-next'
import { api, ApiError } from '@/api/client'
import type { Tenant, Tone } from '@/api/types'
import { errorText, toneClass } from '@/design-system'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const tenants = ref<Tenant[]>([])
const tenant = ref('UNIKA')
const identifier = ref('')
const password = ref('')
const show = ref(false)
const error = ref<string | null>(null)
const busy = ref(false)

const DEMO: { label: string; id: string; tone: Tone }[] = [
  { label: 'Mahasiswa', id: '2024110001', tone: 'blue' },
  { label: 'Mhs belum bayar', id: '2024110002', tone: 'red' },
  { label: 'Dosen · Wali · Kaprodi', id: '0011223344', tone: 'violet' },
  { label: 'Dosen', id: '0055667788', tone: 'pink' },
  { label: 'Operator BAAK', id: 'baak@unika.ac.id', tone: 'green' },
  { label: 'Keuangan', id: 'keuangan@unika.ac.id', tone: 'amber' },
  { label: 'Admin kampus', id: 'admin@unika.ac.id', tone: 'gray' },
  { label: 'Super admin', id: 'super@kampusops.id', tone: 'gray' },
  { label: 'Calon mahasiswa', id: 'calon@mail.com', tone: 'pink' },
]

onMounted(async () => {
  try { tenants.value = await api.get<Tenant[]>('/tenants/public') } catch { tenants.value = [] }
})

async function signIn(id: string, pw: string) {
  error.value = null
  busy.value = true
  try {
    await auth.login(tenant.value, id.trim(), pw)
    router.replace((route.query.next as string) || '/')
  } catch (e) {
    error.value = e instanceof ApiError ? errorText(e.code, e.message) : String(e)
  } finally {
    busy.value = false
  }
}
const submit = () => signIn(identifier.value, password.value)
/** Demo tiles sign in immediately — the fastest path through the acceptance demo. */
function demo(id: string) {
  tenant.value = 'UNIKA'
  identifier.value = id
  password.value = 'password123'
  signIn(id, 'password123')
}
</script>

<template>
  <div class="grid min-h-dvh lg:grid-cols-[1.05fr_1fr]">
    <!-- Brand panel with a live mini-dashboard built from real components -->
    <section class="relative hidden overflow-hidden bg-surface p-10 lg:flex lg:flex-col">
      <div class="flex items-center gap-2.5">
        <span class="relative block size-8">
          <span class="absolute left-0 top-0 size-4 rounded-pill bg-blue" />
          <span class="absolute right-0 top-0.5 size-3.5 rounded-pill bg-green" />
          <span class="absolute bottom-0 left-2 size-4 rounded-pill bg-amber" />
        </span>
        <span class="text-md font-extrabold">KampusOps</span>
      </div>

      <div class="mt-14 max-w-xl">
        <span class="inline-flex items-center gap-2 rounded-pill bg-accent-soft px-3 py-1 text-xs font-bold text-accent">SIAKAD multi-kampus · SaaS</span>
        <h1 class="mt-5 text-[44px] font-extrabold leading-[1.05] tracking-tight">Satu sumber data akademik dan keuangan kampus.</h1>
        <p class="mt-4 max-w-md text-md text-ink-2">Dari PMB sampai lapor PDDikti. Aturan akademik ditegakkan sistem, bukan dicek manual.</p>
      </div>

      <!-- Collage -->
      <div class="pointer-events-none relative mt-12 flex-1 select-none" aria-hidden="true">
        <div class="absolute left-0 top-0 w-[260px] rounded-card bg-canvas p-1.5 shadow-pop">
          <KStatCard label="KRS disetujui" value="1.284" tone="green" hint="+212 hari ini" trend="up" />
        </div>
        <div class="absolute left-[280px] top-6 w-[230px] rounded-card bg-surface p-5 shadow-pop">
          <p class="mb-3 font-bold">Skor PDDikti</p>
          <div class="flex items-center gap-4">
            <KRing :value="96" tone="green" :size="84"><p class="tabular text-md font-bold">96</p></KRing>
            <div class="flex flex-col gap-2 text-sm">
              <KBadge tone="red" variant="dot">0 error</KBadge>
              <KBadge tone="amber" variant="dot">3 peringatan</KBadge>
            </div>
          </div>
        </div>
        <div class="absolute left-10 top-[180px] w-[440px] rounded-card bg-surface p-5 shadow-pop">
          <div class="mb-3 flex items-center justify-between"><p class="font-bold">Kuota kelas · detik pertama KRS</p><KBadge tone="blue">0 overbooking</KBadge></div>
          <div class="flex flex-col gap-3">
            <div v-for="k in [{ n: 'Struktur Data · A', v: 40, m: 40 }, { n: 'Basis Data · B', v: 33, m: 40 }, { n: 'Kecerdasan Buatan · A', v: 21, m: 35 }]" :key="k.n" class="flex items-center gap-3">
              <span class="w-40 truncate text-sm font-semibold">{{ k.n }}</span>
              <KProgress class="flex-1" :value="k.v" :max="k.m" :tone="k.v >= k.m ? 'red' : 'blue'" />
              <span class="tabular w-12 text-right text-sm text-ink-3">{{ k.v }}/{{ k.m }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="mt-6 flex gap-6 text-sm text-ink-3">
        <span class="inline-flex items-center gap-1.5"><Zap class="size-4 text-amber" />KRS p95 &lt; 300 ms</span>
        <span class="inline-flex items-center gap-1.5"><ShieldCheck class="size-4 text-green" />Data terisolasi per kampus</span>
      </div>
    </section>

    <!-- Form -->
    <section class="flex items-center justify-center p-4 sm:p-10">
      <div class="w-full max-w-[440px]">
        <div class="mb-8 flex items-center gap-2.5 lg:hidden">
          <span class="relative block size-8">
            <span class="absolute left-0 top-0 size-4 rounded-pill bg-blue" />
            <span class="absolute right-0 top-0.5 size-3.5 rounded-pill bg-green" />
            <span class="absolute bottom-0 left-2 size-4 rounded-pill bg-amber" />
          </span>
          <span class="text-md font-extrabold">KampusOps</span>
        </div>

        <div class="rounded-card bg-surface p-6 shadow-card sm:p-8">
          <h2 class="text-xl font-extrabold tracking-tight">Masuk</h2>
          <p class="mt-1 text-ink-3">Gunakan email, NIM, atau NIDN.</p>

          <form class="mt-6 flex flex-col gap-4" @submit.prevent="submit">
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
              <div class="relative">
                <KInput id="password" v-model="password" :type="show ? 'text' : 'password'" autocomplete="current-password" required class="[&_input]:pr-10" />
                <button type="button" class="absolute right-3 top-1/2 -translate-y-1/2 text-ink-3 hover:text-ink" :aria-label="show ? 'Sembunyikan' : 'Tampilkan'" @click="show = !show">
                  <EyeOff v-if="show" class="size-4" /><Eye v-else class="size-4" />
                </button>
              </div>
            </KField>
            <KAlert v-if="error" level="error">{{ error }}</KAlert>
            <KButton type="submit" :loading="busy" block>Masuk <ArrowRight class="size-4" /></KButton>
          </form>
        </div>

        <div class="mt-6">
          <div class="mb-3 flex items-center gap-3">
            <span class="h-px flex-1 bg-line" />
            <p class="ko-eyebrow">Masuk cepat · akun demo</p>
            <span class="h-px flex-1 bg-line" />
          </div>
          <div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
            <button
              v-for="d in DEMO" :key="d.id" type="button" :disabled="busy"
              class="group flex flex-col items-start gap-1 rounded-control border border-line bg-surface px-3 py-2.5 text-left transition-all hover:-translate-y-0.5 hover:border-line-strong hover:shadow-card disabled:opacity-50"
              @click="demo(d.id)"
            >
              <span class="flex items-center gap-1.5 text-sm font-bold"><span :class="['size-2 rounded-pill', toneClass[d.tone].dot]" />{{ d.label }}</span>
              <span class="w-full truncate font-mono text-[11px] text-ink-3">{{ d.id }}</span>
            </button>
          </div>
        </div>

        <p class="mt-6 text-center text-sm text-ink-3">
          Calon mahasiswa baru?
          <RouterLink to="/daftar" class="font-semibold text-accent hover:underline">Daftar PMB</RouterLink>
        </p>
      </div>
    </section>
  </div>
</template>
