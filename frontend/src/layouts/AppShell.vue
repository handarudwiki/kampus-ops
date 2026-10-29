<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { Bell, ChevronsUpDown, LogOut, Menu as MenuIcon, Monitor, Moon, Search, Sun, X } from 'lucide-vue-next'
import { NAV, NAV_FOOTER } from '@/config/navigation'
import { ROLE_LABEL } from '@/config/roles'
import { useAuthStore } from '@/stores/auth'
import { usePeriodeStore } from '@/stores/periode'
import { useUiStore, type Theme } from '@/stores/ui'
import { api } from '@/api/client'
import type { Notifikasi, Role } from '@/api/types'
import { relative, SERIES_TONES, toneClass } from '@/design-system'
import CommandPalette from './CommandPalette.vue'

const auth = useAuthStore()
const periode = usePeriodeStore()
const ui = useUiStore()
const route = useRoute()
const router = useRouter()

const groups = computed(() => (auth.activeRole ? NAV[auth.activeRole] : []))
const flat = computed(() => groups.value.flatMap((g) => g.items))
const isActive = (to: string) => (to === '/' ? route.path === '/' : route.path === to || route.path.startsWith(to + '/'))
/** Mobile bottom bar: first four destinations of the role */
const tabs = computed(() => flat.value.slice(0, 4))

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

// Command palette (⌘K / Ctrl+K)
const palette = ref(false)
function onKey(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    palette.value = !palette.value
  }
}
onMounted(() => {
  periode.load()
  loadNotif()
  window.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
watch(() => route.fullPath, () => (ui.navOpen = false))

const isMac = typeof navigator !== 'undefined' && /Mac/.test(navigator.platform)
</script>

<template>
  <div class="min-h-dvh lg:p-3">
    <div class="flex min-h-dvh gap-3 lg:min-h-[calc(100dvh-24px)]">
      <!-- Mobile scrim -->
      <Transition enter-active-class="transition-opacity duration-200" enter-from-class="opacity-0" leave-active-class="transition-opacity duration-150" leave-to-class="opacity-0">
        <div v-if="ui.navOpen" class="fixed inset-0 z-30 bg-ink/30 backdrop-blur-[2px] lg:hidden" @click="ui.navOpen = false" />
      </Transition>

      <!-- Sidebar -->
      <aside
        :class="[
          'fixed inset-y-0 left-0 z-40 flex w-[272px] flex-col bg-surface px-4 pb-4 pt-5 shadow-pop transition-transform duration-300 ease-ko',
          'lg:sticky lg:top-3 lg:h-[calc(100dvh-24px)] lg:translate-x-0 lg:rounded-card lg:shadow-card',
          ui.navOpen ? 'translate-x-0' : '-translate-x-full',
        ]"
        aria-label="Sidebar"
      >
        <div class="mb-5 flex items-center justify-between px-2">
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

        <!-- Search trigger, like the reference's "Search ⌘S" row -->
        <button
          type="button"
          class="mb-3 flex h-10 items-center gap-3 rounded-control px-3 text-ink-2 transition-colors hover:bg-surface-2 hover:text-ink"
          @click="palette = true"
        >
          <Search class="size-[18px] text-ink-3" />
          <span class="flex-1 text-left font-semibold">Cari</span>
          <kbd class="rounded-[6px] border border-line bg-surface-2 px-1.5 font-mono text-[11px] text-ink-3">{{ isMac ? '⌘' : 'Ctrl' }} K</kbd>
        </button>

        <!-- Role switcher (FR-03) -->
        <KMenu v-if="auth.user && auth.user.roles.length" :items="roleItems" align="left" class="mb-5 w-full">
          <button class="group flex w-full items-center gap-3 rounded-control border border-line px-3 py-2.5 text-left transition-colors hover:border-line-strong">
            <span class="grid size-8 place-items-center rounded-[10px] bg-accent-soft text-xs font-extrabold text-accent">{{ auth.tenant?.code.slice(0, 2) }}</span>
            <div class="min-w-0 flex-1">
              <p class="truncate text-xs font-semibold text-ink-3">{{ auth.tenant?.name }}</p>
              <p class="truncate font-bold">{{ auth.activeRole ? ROLE_LABEL[auth.activeRole] : '—' }}</p>
            </div>
            <ChevronsUpDown v-if="auth.user.roles.length > 1" class="size-4 text-ink-3 group-hover:text-ink" />
          </button>
        </KMenu>

        <nav class="ko-scroll -mx-1 flex-1 overflow-y-auto px-1" aria-label="Navigasi utama">
          <div v-for="(g, gi) in groups" :key="gi" class="mb-5">
            <p v-if="g.label" class="ko-eyebrow mb-2 flex items-center gap-2 px-3">
              <span :class="['size-1.5 rounded-pill', toneClass[SERIES_TONES[(gi - 1 + SERIES_TONES.length) % SERIES_TONES.length]!].dot]" />{{ g.label }}
            </p>
            <ul class="flex flex-col gap-0.5">
              <li v-for="it in g.items" :key="it.to">
                <RouterLink
                  :to="it.to"
                  :class="[
                    'relative flex items-center gap-3 rounded-control px-3 py-2.5 font-semibold transition-colors duration-ko',
                    isActive(it.to) ? 'bg-surface-2 text-ink' : 'text-ink-2 hover:bg-surface-2/70 hover:text-ink',
                  ]"
                  :aria-current="isActive(it.to) ? 'page' : undefined"
                >
                  <span v-if="isActive(it.to)" class="absolute -left-1 top-1/2 h-5 w-1 -translate-y-1/2 rounded-pill bg-accent" />
                  <component :is="it.icon" class="size-[18px]" :class="isActive(it.to) ? 'text-accent' : 'text-ink-3'" />
                  {{ it.label }}
                </RouterLink>
              </li>
            </ul>
          </div>
          <RouterLink
            v-for="it in NAV_FOOTER" :key="it.to" :to="it.to"
            class="flex items-center gap-3 rounded-control px-3 py-2.5 font-semibold text-accent transition-colors hover:bg-accent-soft"
          >
            <component :is="it.icon" class="size-[18px]" />{{ it.label }}
          </RouterLink>
        </nav>

        <!-- User + theme (reference: avatar block + sun/moon segmented) -->
        <div class="mt-4 flex flex-col gap-3 border-t border-line pt-4">
          <div v-if="auth.user" class="flex items-center gap-3 px-1">
            <KAvatar :name="auth.user.name" />
            <div class="min-w-0 flex-1">
              <p class="truncate font-bold">{{ auth.user.name }}</p>
              <p class="truncate text-xs text-ink-3">{{ auth.user.email }}</p>
            </div>
            <KIconButton label="Keluar" @click="logout"><LogOut class="size-4" /></KIconButton>
          </div>
          <div class="grid grid-cols-3 rounded-control bg-surface-2 p-1" role="radiogroup" aria-label="Tema">
            <button
              v-for="t in themes" :key="t.value" type="button" role="radio" :aria-checked="ui.theme === t.value" :title="t.label"
              :class="['grid h-8 place-items-center rounded-[9px] transition-all duration-ko', ui.theme === t.value ? 'bg-surface text-ink shadow-card' : 'text-ink-3 hover:text-ink']"
              @click="ui.theme = t.value"
            ><component :is="t.icon" class="size-4" /></button>
          </div>
        </div>
      </aside>

      <!-- Main -->
      <div class="flex min-w-0 flex-1 flex-col">
        <header class="sticky top-0 z-20 flex h-16 items-center gap-3 border-b border-line bg-canvas/80 px-4 backdrop-blur-md lg:static lg:h-auto lg:border-0 lg:bg-transparent lg:px-2 lg:pb-4 lg:pt-2 lg:backdrop-blur-none">
          <KIconButton label="Buka menu" class="lg:hidden" @click="ui.navOpen = true"><MenuIcon class="size-4" /></KIconButton>
          <div class="flex min-w-0 flex-1 items-center gap-2.5">
            <p class="truncate text-md font-bold lg:hidden">{{ route.meta.title }}</p>
            <span v-if="periode.active" class="hidden h-9 items-center gap-2 rounded-pill bg-surface pl-3 pr-1 text-sm font-semibold shadow-card sm:inline-flex">
              <KDot tone="green" />
              <span class="truncate">{{ periode.label }}</span>
              <span v-if="periode.phase" class="rounded-pill bg-ink px-2.5 py-1 text-xs font-bold text-ink-on">{{ periode.phase.label }}</span>
            </span>
            <span v-if="auth.tenant" class="hidden truncate text-sm text-ink-3 xl:inline">{{ auth.tenant.timezone }}</span>
          </div>
          <button
            type="button"
            class="hidden h-9 w-60 items-center gap-2 rounded-pill bg-surface px-3 text-sm text-ink-3 shadow-card transition-colors hover:text-ink md:flex"
            @click="palette = true"
          >
            <Search class="size-4" /><span class="flex-1 text-left">Cari halaman…</span>
            <kbd class="font-mono text-[11px]">{{ isMac ? '⌘' : 'Ctrl' }} K</kbd>
          </button>
          <KIconButton label="Cari" class="md:hidden" @click="palette = true"><Search class="size-4" /></KIconButton>
          <KMenu :items="notifItems" @click="loadNotif">
            <button class="relative inline-grid size-9 place-items-center rounded-pill bg-surface text-ink-2 shadow-card transition-colors hover:text-ink" aria-label="Notifikasi">
              <Bell class="size-4" />
              <span v-if="unread" class="absolute -right-0.5 -top-0.5 grid h-4 min-w-4 place-items-center rounded-pill bg-red px-1 text-[10px] font-bold text-white ring-2 ring-canvas">{{ unread }}</span>
            </button>
            <template #header>
              <div class="flex items-center justify-between px-2.5 pb-1 pt-1.5">
                <p class="ko-eyebrow">Notifikasi</p>
                <span v-if="unread" class="text-xs font-semibold text-accent">{{ unread }} baru</span>
              </div>
              <p v-if="!notif.length" class="px-2.5 py-4 text-sm text-ink-3">Belum ada notifikasi.</p>
            </template>
          </KMenu>
        </header>

        <main class="mx-auto w-full max-w-[1400px] flex-1 px-4 pb-28 pt-4 lg:px-2 lg:pb-10 lg:pt-0">
          <slot />
        </main>
      </div>
    </div>

    <!-- Mobile bottom tab bar (360px requirement: thumb-reachable primary nav) -->
    <nav class="fixed inset-x-0 bottom-0 z-20 border-t border-line bg-surface/90 px-2 pb-[max(env(safe-area-inset-bottom),8px)] pt-2 backdrop-blur-md lg:hidden" aria-label="Navigasi cepat">
      <ul class="grid grid-cols-5">
        <li v-for="t in tabs" :key="t.to">
          <RouterLink :to="t.to" :class="['flex flex-col items-center gap-1 rounded-control py-1.5 text-[11px] font-semibold', isActive(t.to) ? 'text-accent' : 'text-ink-3']">
            <component :is="t.icon" class="size-5" /><span class="max-w-full truncate px-1">{{ t.label }}</span>
          </RouterLink>
        </li>
        <li>
          <button type="button" class="flex w-full flex-col items-center gap-1 rounded-control py-1.5 text-[11px] font-semibold text-ink-3" @click="ui.navOpen = true">
            <MenuIcon class="size-5" /><span>Menu</span>
          </button>
        </li>
      </ul>
    </nav>

    <CommandPalette v-model="palette" />
  </div>
</template>
