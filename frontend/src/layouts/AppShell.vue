<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { Bell, ChevronsUpDown, LogOut, Menu as MenuIcon, Monitor, Moon, Sun, X } from 'lucide-vue-next'
import { NAV, NAV_FOOTER } from '@/config/navigation'
import { ROLE_LABEL } from '@/config/roles'
import { useAuthStore } from '@/stores/auth'
import { usePeriodeStore } from '@/stores/periode'
import { useUiStore, type Theme } from '@/stores/ui'
import { api } from '@/api/client'
import type { Notifikasi, Role } from '@/api/types'
import { relative } from '@/design-system'

const auth = useAuthStore()
const periode = usePeriodeStore()
const ui = useUiStore()
const route = useRoute()
const router = useRouter()

const groups = computed(() => (auth.activeRole ? NAV[auth.activeRole] : []))
const isActive = (to: string) => (to === '/' ? route.path === '/' : route.path === to || route.path.startsWith(to + '/'))

const roleItems = computed(() =>
  (auth.user?.roles ?? []).map((r) => ({
    label: ROLE_LABEL[r.role],
    hint: r.scope_prodi_name ?? undefined,
    active: r.role === auth.activeRole,
    onClick: () => switchRole(r.role),
  })),
)
function switchRole(role: Role) {
  auth.switchRole(role)
  router.push('/')
}
async function logout() {
  await auth.logout()
  router.push('/login')
}

const themes: { value: Theme; icon: typeof Sun; label: string }[] = [
  { value: 'light', icon: Sun, label: 'Terang' },
  { value: 'system', icon: Monitor, label: 'Sistem' },
  { value: 'dark', icon: Moon, label: 'Gelap' },
]

const notif = ref<Notifikasi[]>([])
const unread = computed(() => notif.value.filter((n) => !n.read).length)
async function loadNotif() {
  try { notif.value = await api.get<Notifikasi[]>('/notifikasi') } catch { notif.value = [] }
}
const notifItems = computed(() =>
  notif.value.slice(0, 8).map((n) => ({
    label: n.judul,
    hint: relative(n.created_at),
    active: !n.read,
    onClick: async () => {
      try { await api.post(`/notifikasi/${n.id}/read`) } catch { /* ignore */ }
      n.read = true
      if (n.link) router.push(n.link)
    },
  })),
)

onMounted(() => {
  periode.load()
  loadNotif()
})
watch(() => route.fullPath, () => (ui.navOpen = false))
</script>

<template>
  <div class="min-h-dvh p-0 lg:p-3">
    <div class="flex min-h-dvh gap-3 lg:min-h-[calc(100dvh-24px)]">
      <!-- Mobile scrim -->
      <div v-if="ui.navOpen" class="fixed inset-0 z-30 bg-ink/30 lg:hidden" @click="ui.navOpen = false" />

      <!-- Sidebar -->
      <aside
        :class="[
          'fixed inset-y-0 left-0 z-40 flex w-[264px] flex-col bg-surface p-4 shadow-pop transition-transform duration-200 ease-ko',
          'lg:sticky lg:top-3 lg:h-[calc(100dvh-24px)] lg:translate-x-0 lg:rounded-card lg:shadow-card',
          ui.navOpen ? 'translate-x-0' : '-translate-x-full',
        ]"
      >
        <div class="mb-6 flex items-center justify-between px-2 pt-1">
          <RouterLink to="/" class="flex items-center gap-2.5">
            <span class="relative block size-8">
              <span class="absolute left-0 top-0 size-4 rounded-pill bg-blue" />
              <span class="absolute right-0 top-0.5 size-3.5 rounded-pill bg-green" />
              <span class="absolute bottom-0 left-2 size-4 rounded-pill bg-amber" />
            </span>
            <span class="text-md font-extrabold tracking-tight">KampusOps</span>
          </RouterLink>
          <KIconButton label="Tutup menu" class="lg:hidden" @click="ui.navOpen = false"><X class="size-4" /></KIconButton>
        </div>

        <!-- Role switcher (FR-03) -->
        <KMenu v-if="auth.user && auth.user.roles.length" :items="roleItems" align="left" class="mb-5 w-full">
          <button class="flex w-full items-center gap-3 rounded-control bg-surface-2 px-3 py-2.5 text-left transition-colors hover:bg-surface-3">
            <div class="min-w-0 flex-1">
              <p class="ko-eyebrow">{{ auth.tenant?.code }}</p>
              <p class="truncate font-bold">{{ auth.activeRole ? ROLE_LABEL[auth.activeRole] : '—' }}</p>
            </div>
            <ChevronsUpDown v-if="auth.user.roles.length > 1" class="size-4 text-ink-3" />
          </button>
        </KMenu>

        <nav class="ko-scroll -mx-1 flex-1 overflow-y-auto px-1" aria-label="Navigasi utama">
          <div v-for="(g, gi) in groups" :key="gi" class="mb-5">
            <p v-if="g.label" class="ko-eyebrow mb-2 px-3">{{ g.label }}</p>
            <ul class="flex flex-col gap-0.5">
              <li v-for="it in g.items" :key="it.to">
                <RouterLink
                  :to="it.to"
                  :class="[
                    'flex items-center gap-3 rounded-control px-3 py-2 font-semibold transition-colors duration-ko',
                    isActive(it.to) ? 'bg-surface-3 text-ink' : 'text-ink-2 hover:bg-surface-2 hover:text-ink',
                  ]"
                  :aria-current="isActive(it.to) ? 'page' : undefined"
                >
                  <component :is="it.icon" class="size-[18px]" :class="isActive(it.to) ? 'text-accent' : 'text-ink-3'" />
                  {{ it.label }}
                </RouterLink>
              </li>
            </ul>
          </div>
          <ul class="border-t border-line pt-3">
            <li v-for="it in NAV_FOOTER" :key="it.to">
              <RouterLink :to="it.to" class="flex items-center gap-3 rounded-control px-3 py-2 font-semibold text-accent hover:bg-accent-soft">
                <component :is="it.icon" class="size-[18px]" />{{ it.label }}
              </RouterLink>
            </li>
          </ul>
        </nav>

        <!-- User + theme (reference: avatar block + sun/moon segmented) -->
        <div class="mt-4 flex flex-col gap-3 border-t border-line pt-4">
          <div v-if="auth.user" class="flex items-center gap-3 px-1">
            <KAvatar :name="auth.user.name" />
            <div class="min-w-0 flex-1">
              <p class="truncate font-bold">{{ auth.user.name }}</p>
              <p class="truncate text-xs text-ink-3">{{ auth.user.identifier }}</p>
            </div>
            <KIconButton label="Keluar" @click="logout"><LogOut class="size-4" /></KIconButton>
          </div>
          <div class="grid grid-cols-3 rounded-control bg-surface-3 p-1" role="radiogroup" aria-label="Tema">
            <button
              v-for="t in themes" :key="t.value" type="button" role="radio" :aria-checked="ui.theme === t.value" :title="t.label"
              :class="['grid h-7 place-items-center rounded-[8px] transition-colors', ui.theme === t.value ? 'bg-surface text-ink shadow-card' : 'text-ink-3 hover:text-ink']"
              @click="ui.theme = t.value"
            ><component :is="t.icon" class="size-4" /></button>
          </div>
        </div>
      </aside>

      <!-- Main -->
      <div class="flex min-w-0 flex-1 flex-col">
        <header class="sticky top-0 z-20 flex items-center gap-3 bg-canvas/85 px-4 py-3 backdrop-blur lg:static lg:bg-transparent lg:px-2 lg:pt-1 lg:backdrop-blur-none">
          <KIconButton label="Buka menu" class="lg:hidden" @click="ui.navOpen = true"><MenuIcon class="size-4" /></KIconButton>
          <div class="flex min-w-0 flex-1 items-center gap-2">
            <span v-if="periode.active" class="inline-flex h-8 items-center gap-2 rounded-pill bg-surface px-3 text-sm font-semibold shadow-card">
              <KDot tone="green" />
              <span class="truncate">{{ periode.active.tipe }} {{ periode.active.tahun }}/{{ periode.active.tahun + 1 }}</span>
            </span>
            <span v-if="auth.tenant" class="hidden truncate text-sm text-ink-3 sm:inline">{{ auth.tenant.name }} · {{ auth.tenant.timezone }}</span>
          </div>
          <KMenu :items="notifItems" @click="loadNotif">
            <button class="relative inline-grid size-9 place-items-center rounded-pill bg-surface text-ink-2 shadow-card hover:text-ink" aria-label="Notifikasi">
              <Bell class="size-4" />
              <span v-if="unread" class="absolute -right-0.5 -top-0.5 grid min-w-4 place-items-center rounded-pill bg-red px-1 text-[10px] font-bold text-white">{{ unread }}</span>
            </button>
            <template #header>
              <p class="ko-eyebrow px-2.5 pb-1 pt-1.5">Notifikasi</p>
              <p v-if="!notif.length" class="px-2.5 py-3 text-sm text-ink-3">Belum ada notifikasi.</p>
            </template>
          </KMenu>
        </header>
        <main class="mx-auto w-full max-w-[1400px] flex-1 px-4 pb-10 pt-2 lg:px-2">
          <slot />
        </main>
      </div>
    </div>
  </div>
</template>
