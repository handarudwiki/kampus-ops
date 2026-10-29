<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Dialog, DialogPanel, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { CornerDownLeft, Moon, Repeat, Search, Sun } from 'lucide-vue-next'
import { NAV, NAV_FOOTER } from '@/config/navigation'
import { ROLE_LABEL } from '@/config/roles'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'

/** ⌘K / Ctrl+K: jump to any page of the active role, switch role, or change theme. */
const open = defineModel<boolean>({ required: true })
const auth = useAuthStore()
const ui = useUiStore()
const router = useRouter()
const q = ref('')
const cursor = ref(0)
const input = ref<HTMLInputElement | null>(null)

interface Cmd { id: string; label: string; group: string; icon: unknown; hint?: string; run: () => void }

const commands = computed<Cmd[]>(() => {
  const role = auth.activeRole
  const pages = role
    ? NAV[role].flatMap((g) => g.items.map((it) => ({ id: it.to, label: it.label, group: 'Halaman', icon: it.icon, hint: g.label, run: () => router.push(it.to) })))
    : []
  const extras = NAV_FOOTER.map((it) => ({ id: it.to, label: it.label, group: 'Halaman', icon: it.icon, run: () => router.push(it.to) }))
  const roles = (auth.user?.roles ?? [])
    .filter((r) => r.role !== role)
    .map((r) => ({ id: 'role:' + r.role, label: `Beralih ke ${ROLE_LABEL[r.role]}`, group: 'Peran', icon: Repeat, run: () => { auth.switchRole(r.role); router.push('/') } }))
  const theme = [
    { id: 'theme:light', label: 'Tema terang', group: 'Tampilan', icon: Sun, run: () => (ui.theme = 'light') },
    { id: 'theme:dark', label: 'Tema gelap', group: 'Tampilan', icon: Moon, run: () => (ui.theme = 'dark') },
  ]
  return [...pages, ...extras, ...roles, ...theme]
})

const results = computed(() => {
  const term = q.value.trim().toLowerCase()
  return term ? commands.value.filter((c) => `${c.label} ${c.hint ?? ''}`.toLowerCase().includes(term)) : commands.value
})
const grouped = computed(() => {
  const m = new Map<string, { cmd: Cmd; i: number }[]>()
  results.value.forEach((cmd, i) => m.set(cmd.group, [...(m.get(cmd.group) ?? []), { cmd, i }]))
  return [...m.entries()]
})

watch(open, async (v) => {
  if (!v) return
  q.value = ''
  cursor.value = 0
  await nextTick()
  input.value?.focus()
})
watch(q, () => (cursor.value = 0))

function exec(c?: Cmd) {
  if (!c) return
  open.value = false
  c.run()
}
function onKey(e: KeyboardEvent) {
  if (e.key === 'ArrowDown') { e.preventDefault(); cursor.value = Math.min(results.value.length - 1, cursor.value + 1) }
  else if (e.key === 'ArrowUp') { e.preventDefault(); cursor.value = Math.max(0, cursor.value - 1) }
  else if (e.key === 'Enter') { e.preventDefault(); exec(results.value[cursor.value]) }
}
</script>

<template>
  <TransitionRoot :show="open" as="template">
    <Dialog class="relative z-50" @close="open = false">
      <TransitionChild enter="duration-150 ease-out" enter-from="opacity-0" leave="duration-100" leave-to="opacity-0">
        <div class="fixed inset-0 bg-ink/30 backdrop-blur-[2px]" />
      </TransitionChild>
      <div class="fixed inset-0 flex items-start justify-center p-4 pt-[12vh]">
        <TransitionChild enter="duration-150 ease-out" enter-from="opacity-0 scale-[0.98]" leave="duration-100" leave-to="opacity-0 scale-[0.98]" as="template">
          <DialogPanel class="w-full max-w-xl overflow-hidden rounded-card border border-line bg-surface shadow-pop">
            <div class="flex items-center gap-3 border-b border-line px-5">
              <Search class="size-5 text-ink-3" />
              <input
                ref="input"
                v-model="q"
                class="h-14 flex-1 bg-transparent text-md outline-none placeholder:text-ink-3 focus-visible:ring-0 focus-visible:ring-offset-0"
                placeholder="Cari halaman atau perintah…"
                aria-label="Cari perintah"
                @keydown="onKey"
              />
              <kbd class="rounded-[6px] border border-line px-1.5 py-0.5 font-mono text-xs text-ink-3">esc</kbd>
            </div>
            <div class="ko-scroll max-h-[50vh] overflow-y-auto p-2">
              <p v-if="!results.length" class="px-3 py-8 text-center text-sm text-ink-3">Tidak ada hasil untuk “{{ q }}”.</p>
              <div v-for="[group, items] in grouped" :key="group" class="mb-1">
                <p class="ko-eyebrow px-3 pb-1 pt-2">{{ group }}</p>
                <button
                  v-for="{ cmd, i } in items"
                  :key="cmd.id"
                  type="button"
                  :class="['flex w-full items-center gap-3 rounded-control px-3 py-2.5 text-left', i === cursor ? 'bg-surface-3' : 'hover:bg-surface-2']"
                  @mouseenter="cursor = i"
                  @click="exec(cmd)"
                >
                  <span class="grid size-8 place-items-center rounded-[10px] bg-surface-2 text-ink-2"><component :is="cmd.icon" class="size-4" /></span>
                  <span class="flex-1 font-semibold">{{ cmd.label }}</span>
                  <span v-if="cmd.hint" class="text-xs text-ink-3">{{ cmd.hint }}</span>
                  <CornerDownLeft v-if="i === cursor" class="size-4 text-ink-3" />
                </button>
              </div>
            </div>
          </DialogPanel>
        </TransitionChild>
      </div>
    </Dialog>
  </TransitionRoot>
</template>
