<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import { AlertTriangle, ArrowUpRight, Info, XCircle } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Dashboard, Role } from '@/api/types'
import { useAsync } from '@/composables/useAsync'
import { ROLE_LABEL } from '@/config/roles'
import { toneClass } from '@/design-system'
import { useAuthStore } from '@/stores/auth'

/**
 * One dashboard for every role. The API returns the generic `Dashboard` shape,
 * so each role gets the same layout: KPI row → chart + alerts → table.
 */
const auth = useAuthStore()
const PmbStatus = defineAsyncComponent(() => import('./pmb/PmbStatusView.vue'))

const { data, loading } = useAsync<Dashboard | null>(() => api.get<Dashboard>('/dashboard'), null)

const QUICK: Partial<Record<Role, { label: string; to: string }[]>> = {
  mahasiswa: [{ label: 'Isi KRS', to: '/krs' }, { label: 'Lihat tagihan', to: '/tagihan' }],
  dosen_wali: [{ label: 'Proses KRS', to: '/perwalian' }],
  dosen: [{ label: 'Kelas saya', to: '/mengajar' }],
  kaprodi: [{ label: 'Koreksi nilai', to: '/koreksi-nilai' }, { label: 'Masa studi', to: '/masa-studi' }],
  operator: [{ label: 'Validator PDDikti', to: '/pddikti' }, { label: 'Jadwal kelas', to: '/kelas' }],
  keuangan: [{ label: 'Tagihan', to: '/keuangan/tagihan' }, { label: 'Dispensasi', to: '/keuangan/dispensasi' }],
  admin_kampus: [{ label: 'Periode', to: '/periode' }, { label: 'Pengguna', to: '/pengguna' }],
  super_admin: [{ label: 'Tenant', to: '/tenants' }],
}
const quick = computed(() => (auth.activeRole ? QUICK[auth.activeRole] ?? [] : []))
const firstName = computed(() => auth.user?.name.split(' ')[0] ?? '')
const alertIcon = { info: Info, warning: AlertTriangle, error: XCircle }
const alertTone = { info: 'blue', warning: 'amber', error: 'red' } as const
const statCols = computed(() => {
  const n = data.value?.stats.length ?? 3
  return n >= 4 ? 'sm:grid-cols-2 xl:grid-cols-4' : 'sm:grid-cols-3'
})
</script>

<template>
  <PmbStatus v-if="auth.activeRole === 'calon_mahasiswa'" />
  <div v-else>
    <KPageHeader :title="`Halo, ${firstName}`" :description="auth.activeRole ? `Ringkasan untuk ${ROLE_LABEL[auth.activeRole]}` : ''">
      <template #actions>
        <KButton v-for="(q, i) in quick" :key="q.to" :to="q.to" :variant="i === 0 ? 'primary' : 'secondary'">{{ q.label }}</KButton>
      </template>
    </KPageHeader>

    <!-- KPI row -->
    <div :class="['grid grid-cols-1 gap-3', statCols]">
      <template v-if="loading && !data">
        <div v-for="i in 4" :key="i" class="h-[150px] animate-pulse rounded-card bg-surface" />
      </template>
      <KStatCard v-for="s in data?.stats ?? []" :key="s.key" :label="s.label" :value="s.value" :tone="s.tone" :hint="s.hint" :trend="s.trend" />
    </div>

    <div v-if="data" class="mt-3 grid grid-cols-1 gap-3 xl:grid-cols-3">
      <!-- Chart -->
      <KCard v-if="data.chart" :title="data.chart.title" :class="data.alerts.length ? 'xl:col-span-2' : 'xl:col-span-3'">
        <KLineChart v-if="data.chart.labels.length > 6 && data.chart.series.length > 1" :labels="data.chart.labels" :series="data.chart.series" />
        <KBarChart v-else :labels="data.chart.labels" :series="data.chart.series" />
      </KCard>

      <!-- Alerts, styled like the reference's tinted suggestion cards -->
      <KCard v-if="data.alerts.length" title="Perlu perhatian" :class="!data.chart && 'xl:col-span-3'">
        <ul class="flex flex-col gap-2.5">
          <li v-for="(a, i) in data.alerts" :key="i">
            <component
              :is="a.link ? 'RouterLink' : 'div'"
              :to="a.link ?? undefined"
              :class="['group flex items-start gap-3 rounded-control p-3.5', toneClass[alertTone[a.level]].tint]"
            >
              <component :is="alertIcon[a.level]" :class="['mt-0.5 size-4 shrink-0', toneClass[alertTone[a.level]].text]" />
              <p class="flex-1 text-sm font-medium text-ink-2">{{ a.message }}</p>
              <ArrowUpRight v-if="a.link" class="size-4 shrink-0 text-ink-3 transition-transform group-hover:-translate-y-0.5 group-hover:translate-x-0.5" />
            </component>
          </li>
        </ul>
      </KCard>

      <!-- Table -->
      <KCard v-if="data.table" :title="data.table.title" :to="data.table.link ?? undefined" flush class="xl:col-span-3">
        <KTable :columns="data.table.columns" :rows="data.table.rows" :row-key="(r) => JSON.stringify(r)" />
      </KCard>
    </div>
  </div>
</template>
