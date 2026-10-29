<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { MoreHorizontal, Pencil, Plus, Trash2 } from 'lucide-vue-next'
import { api, ApiError } from '@/api/client'
import { useAction } from '@/composables/useAsync'
import { confirm, toastError } from '@/design-system'
import { clearOptionCache, getPath, setPath, type Opt, type ResourceConfig } from './resource'

const props = defineProps<{ config: ResourceConfig }>()
const cfg = props.config
const router = useRouter()

// List state
const rows = ref<Record<string, unknown>[]>([])
const total = ref(0)
const page = ref(1)
const perPage = 20
const q = ref('')
const filters = reactive<Record<string, string>>({})
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const r = await api.list<Record<string, unknown>>(cfg.endpoint, { q: q.value, page: page.value, per_page: perPage, ...filters })
    rows.value = r.data
    total.value = r.meta?.total ?? r.data.length
  } catch (e) {
    toastError(e, 'Gagal memuat data')
  } finally {
    loading.value = false
  }
}
watch([q, () => ({ ...filters })], () => { page.value = 1; load() }, { deep: true })
watch(page, load)

// Option resolution (static or async)
const opts = reactive<Record<string, Opt[]>>({})
async function resolve(key: string, o?: Opt[] | (() => Promise<Opt[]>)) {
  if (!o) return
  opts[key] = typeof o === 'function' ? await o() : o
}
onMounted(() => {
  load()
  cfg.filters?.forEach((f) => resolve('f:' + f.key, f.options))
  cfg.fields.forEach((f) => resolve(f.key, f.options))
})

// Form
const open = ref(false)
const editing = ref<string | null>(null)
const form = ref<Record<string, unknown>>({})
const errors = ref<Record<string, string>>({})
const { run, busy } = useAction()

function openCreate() {
  editing.value = null
  form.value = structuredClone(cfg.defaults ?? {})
  errors.value = {}
  open.value = true
}
function openEdit(row: Record<string, unknown>) {
  editing.value = row.id as string
  form.value = structuredClone(JSON.parse(JSON.stringify(row)))
  errors.value = {}
  open.value = true
}
async function save() {
  errors.value = {}
  const body = form.value
  try {
    busy.value = true
    if (editing.value) await api.put(`${cfg.endpoint}/${editing.value}`, body)
    else await api.post(cfg.endpoint, body)
    open.value = false
    clearOptionCache()
    ;(await import('@/design-system')).toast('success', `${cfg.singular} ${editing.value ? 'diperbarui' : 'ditambahkan'}`)
    load()
  } catch (e) {
    if (e instanceof ApiError && e.code === 'VALIDATION_ERROR' && e.details && typeof e.details === 'object') errors.value = e.details as Record<string, string>
    else toastError(e, 'Gagal menyimpan')
  } finally {
    busy.value = false
  }
}
async function remove(row: Record<string, unknown>) {
  const r = await confirm({ title: `Hapus ${cfg.singular.toLowerCase()}?`, body: 'Perubahan tercatat di audit log.', danger: true, confirmLabel: 'Hapus' })
  if (r.ok && (await run(() => api.del(`${cfg.endpoint}/${row.id}`), `${cfg.singular} dihapus`))) load()
}

function menu(row: Record<string, unknown>) {
  const items: { label: string; icon: unknown; onClick: () => void; danger?: boolean }[] = [{ label: 'Ubah', icon: Pencil, onClick: () => openEdit(row) }]
  for (const a of cfg.rowActions ?? []) {
    if (a.show && !a.show(row)) continue
    items.push({
      label: a.label,
      icon: a.icon ?? Pencil,
      onClick: async () => {
        if (a.confirm && !(await confirm({ title: a.confirm })).ok) return
        if (await run(() => a.run(row), a.success)) load()
      },
    })
  }
  if (cfg.canDelete !== false) items.push({ label: 'Hapus', icon: Trash2, onClick: () => remove(row), danger: true })
  return items
}
const val = (row: Record<string, unknown>, key: string) => getPath(row, key)
const fval = (key: string) => getPath(form.value, key) as string | number | null
const fset = (key: string, v: unknown) => setPath(form.value, key, v)
</script>

<template>
  <div>
    <KPageHeader :title="cfg.title" :description="cfg.description">
      <template #actions>
        <slot name="actions" />
        <KButton @click="openCreate"><template #icon><Plus class="size-4" /></template>Tambah {{ cfg.singular.toLowerCase() }}</KButton>
      </template>
    </KPageHeader>
    <slot name="top" />
    <KCard flush>
      <KToolbar>
        <KSearch v-model="q" :placeholder="`Cari ${cfg.singular.toLowerCase()}…`" />
        <KSelect v-for="f in cfg.filters ?? []" :key="f.key" v-model="filters[f.key]" :options="opts['f:' + f.key] ?? []" :placeholder="f.label" class="w-full sm:w-44" />
        <template #right><span class="tabular text-sm text-ink-3">{{ total }} data</span></template>
      </KToolbar>
      <KTable
        :columns="[...cfg.columns, { key: '__menu', label: '', align: 'right', width: '56px' }]"
        :rows="rows"
        :loading="loading"
        :clickable="!!cfg.detail"
        @row-click="(r) => cfg.detail && router.push(cfg.detail(r.id as string))"
      >
        <template v-for="c in cfg.columns" :key="c.key" #[`cell-${c.key}`]="{ row }">
          <KStatus v-if="c.status" :domain="c.status" :value="val(row, c.key) as string" variant="dot" />
          <span v-else :class="[c.mono && 'font-mono text-sm', c.align === 'right' && 'tabular']">{{ c.format ? c.format(val(row, c.key), row) : (val(row, c.key) ?? '—') }}</span>
        </template>
        <template #cell-__menu="{ row }">
          <div @click.stop>
            <KMenu :items="menu(row)"><KIconButton label="Aksi"><MoreHorizontal class="size-4" /></KIconButton></KMenu>
          </div>
        </template>
      </KTable>
      <KPagination v-if="total > perPage" v-model="page" :total="total" :per-page="perPage" />
    </KCard>

    <KModal v-model="open" :title="editing ? `Ubah ${cfg.singular.toLowerCase()}` : `Tambah ${cfg.singular.toLowerCase()}`" size="lg">
      <form id="resource-form" class="grid gap-4 sm:grid-cols-2" @submit.prevent="save">
        <template v-for="f in cfg.fields" :key="f.key">
          <KField v-if="!(f.createOnly && editing)" :label="f.label" :required="f.required" :hint="f.hint" :error="errors[f.key]" :class="f.wide && 'sm:col-span-2'">
            <KSelect
              v-if="f.type === 'select'"
              :model-value="fval(f.key)"
              :options="opts[f.key] ?? []"
              placeholder="Pilih…"
              :required="f.required"
              @update:model-value="(v) => fset(f.key, v)"
            />
            <KInput
              v-else
              :model-value="fval(f.key)"
              :type="f.type ?? 'text'"
              :required="f.required"
              :invalid="!!errors[f.key]"
              @update:model-value="(v) => fset(f.key, f.type === 'number' && v !== '' ? Number(v) : v)"
            />
          </KField>
        </template>
      </form>
      <template #footer>
        <KButton variant="secondary" @click="open = false">Batal</KButton>
        <KButton type="submit" form="resource-form" :loading="busy">Simpan</KButton>
      </template>
    </KModal>
  </div>
</template>
