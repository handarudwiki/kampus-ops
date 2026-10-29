<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { Plus, ShieldCheck } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Prodi, Role, User, UserRole } from '@/api/types'
import { useAction } from '@/composables/useAsync'
import { ALL_ROLES, ROLE_LABEL } from '@/config/roles'
import { toastError, toneFor } from '@/design-system'

/** FR-03: one account, many roles; kaprodi / dosen roles scoped to a prodi. */
const rows = ref<User[]>([])
const total = ref(0)
const page = ref(1)
const q = ref('')
const loading = ref(false)
const prodi = ref<Prodi[]>([])
const roles = ALL_ROLES.filter((r) => r !== 'super_admin' && r !== 'calon_mahasiswa')
const scoped: Role[] = ['kaprodi', 'dosen', 'dosen_wali']

async function load() {
  loading.value = true
  try {
    const r = await api.list<User>('/users', { q: q.value, page: page.value, per_page: 20 })
    rows.value = r.data
    total.value = r.meta?.total ?? r.data.length
  } catch (e) { toastError(e) } finally { loading.value = false }
}
onMounted(async () => {
  load()
  prodi.value = (await api.list<Prodi>('/prodi', { per_page: 100 }).catch(() => ({ data: [] }))).data
})
watch(q, () => { page.value = 1; load() })
watch(page, load)

const { run, busy } = useAction()
const open = ref(false)
const editing = ref<User | null>(null)
const form = ref({ name: '', email: '', password: '' })
const sel = ref<Record<string, { on: boolean; scope: string }>>({})
function initSel(rs: UserRole[]) {
  sel.value = Object.fromEntries(roles.map((r) => {
    const found = rs.find((x) => x.role === r)
    return [r, { on: !!found, scope: found?.scope_prodi_id ?? '' }]
  }))
}
function openCreate() { editing.value = null; form.value = { name: '', email: '', password: '' }; initSel([]); open.value = true }
function openRoles(u: User) { editing.value = u; initSel(u.roles); open.value = true }
function payloadRoles() {
  return Object.entries(sel.value).filter(([, v]) => v.on).map(([role, v]) => ({ role, scope_prodi_id: v.scope || null }))
}
async function save() {
  const ok = await run(() => editing.value
    ? api.put(`/users/${editing.value.id}/roles`, { roles: payloadRoles() })
    : api.post('/users', { ...form.value, roles: payloadRoles() }), 'Pengguna disimpan')
  if (ok) { open.value = false; load() }
}
</script>

<template>
  <div>
    <KPageHeader title="Pengguna & peran" description="Satu akun bisa punya beberapa peran. Akses dibatasi ke kampus dan prodi wewenangnya.">
      <template #actions><KButton @click="openCreate"><template #icon><Plus class="size-4" /></template>Tambah pengguna</KButton></template>
    </KPageHeader>
    <KCard flush>
      <KToolbar><KSearch v-model="q" placeholder="Cari nama / email" /></KToolbar>
      <KTable :columns="[{ key: 'user', label: 'Pengguna' }, { key: 'identifier', label: 'Login', hideMobile: true }, { key: 'roles', label: 'Peran' }, { key: 'aksi', label: '', align: 'right' }]" :rows="rows" :loading="loading">
        <template #cell-user="{ row }"><div class="flex items-center gap-3"><KAvatar :name="row.name" size="sm" /><div><p class="font-semibold">{{ row.name }}</p><p class="text-sm text-ink-3">{{ row.email }}</p></div></div></template>
        <template #cell-identifier="{ value }"><span class="font-mono text-sm">{{ value }}</span></template>
        <template #cell-roles="{ row }">
          <div class="flex flex-wrap gap-1">
            <KBadge v-for="r in row.roles" :key="r.role + r.scope_prodi_id" :tone="toneFor(r.role)">{{ ROLE_LABEL[r.role] }}<span v-if="r.scope_prodi_name" class="opacity-70">· {{ r.scope_prodi_name }}</span></KBadge>
          </div>
        </template>
        <template #cell-aksi="{ row }"><KButton size="sm" variant="secondary" @click="openRoles(row)"><template #icon><ShieldCheck class="size-4" /></template>Peran</KButton></template>
      </KTable>
      <KPagination v-if="total > 20" v-model="page" :total="total" :per-page="20" />
    </KCard>

    <KModal v-model="open" :title="editing ? `Peran ${editing.name}` : 'Tambah pengguna'" size="md">
      <div class="flex flex-col gap-4">
        <template v-if="!editing">
          <KField label="Nama" required><KInput v-model="form.name" /></KField>
          <div class="grid gap-4 sm:grid-cols-2">
            <KField label="Email" required><KInput v-model="form.email" type="email" /></KField>
            <KField label="Kata sandi awal" required><KInput v-model="form.password" type="password" /></KField>
          </div>
        </template>
        <div>
          <p class="mb-2 text-sm font-semibold text-ink-2">Peran</p>
          <ul class="divide-y divide-line rounded-control border border-line">
            <li v-for="r in roles" :key="r" class="flex flex-col gap-2 px-4 py-3 sm:flex-row sm:items-center">
              <label class="flex flex-1 cursor-pointer items-center gap-3">
                <input v-model="sel[r]!.on" type="checkbox" class="size-4 accent-[rgb(var(--ko-ink))]" />
                <span class="font-medium">{{ ROLE_LABEL[r] }}</span>
              </label>
              <KSelect v-if="scoped.includes(r) && sel[r]?.on" v-model="sel[r]!.scope" :options="prodi.map((p) => ({ value: p.id, label: p.nama }))" placeholder="Semua prodi" size="sm" class="sm:w-52" />
            </li>
          </ul>
        </div>
      </div>
      <template #footer>
        <KButton variant="secondary" @click="open = false">Batal</KButton>
        <KButton :loading="busy" :disabled="!editing && (!form.name || !form.email || !form.password)" @click="save">Simpan</KButton>
      </template>
    </KModal>
  </div>
</template>
