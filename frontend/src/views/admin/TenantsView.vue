<script setup lang="ts">
import { ref } from 'vue'
import { Building, Plus } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { Tenant } from '@/api/types'
import { useAction, useAsync } from '@/composables/useAsync'
import { num, toneClass, toneFor } from '@/design-system'

/** Vendor view: tenants & plans. No access to student grades by design. */
type T = Tenant & { mahasiswa_count: number; plan: string }
const { data, loading, reload } = useAsync<T[]>(() => api.get<T[]>('/tenants'), [])
const { run, busy } = useAction()
const open = ref(false)
const f = ref({ code: '', name: '', timezone: 'Asia/Jakarta', plan: 'standard' })
const TZ = [{ value: 'Asia/Jakarta', label: 'WIB · Asia/Jakarta' }, { value: 'Asia/Makassar', label: 'WITA · Asia/Makassar' }, { value: 'Asia/Jayapura', label: 'WIT · Asia/Jayapura' }]
async function save() {
  if (await run(() => api.post('/tenants', f.value), 'Tenant dibuat')) { open.value = false; reload() }
}
</script>

<template>
  <div>
    <KPageHeader title="Tenant kampus" description="Setiap kampus terisolasi penuh; semua unique index diawali tenant_id.">
      <template #actions><KButton @click="open = true"><template #icon><Plus class="size-4" /></template>Tambah kampus</KButton></template>
    </KPageHeader>
    <div v-if="loading" class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3"><div v-for="i in 2" :key="i" class="h-40 animate-pulse rounded-card bg-surface" /></div>
    <div v-else class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      <KCard v-for="t in data" :key="t.id">
        <div class="flex items-start gap-4">
          <span :class="['grid size-11 place-items-center rounded-control', toneClass[toneFor(t.code)].soft]"><Building class="size-5" /></span>
          <div class="min-w-0 flex-1">
            <p class="truncate text-md font-bold">{{ t.name }}</p>
            <p class="text-sm text-ink-3"><span class="font-mono">{{ t.code }}</span> · {{ t.timezone }}</p>
          </div>
          <KBadge tone="violet">{{ t.plan }}</KBadge>
        </div>
        <p class="tabular mt-5 text-xl font-bold">{{ num(t.mahasiswa_count) }} <span class="text-base font-medium text-ink-3">mahasiswa</span></p>
      </KCard>
    </div>
    <KModal v-model="open" title="Tambah kampus" size="sm">
      <div class="flex flex-col gap-4">
        <KField label="Kode" required hint="Dipakai saat login"><KInput v-model="f.code" /></KField>
        <KField label="Nama kampus" required><KInput v-model="f.name" /></KField>
        <KField label="Zona waktu" required><KSelect v-model="f.timezone" :options="TZ" /></KField>
        <KField label="Paket" required><KSelect v-model="f.plan" :options="[{ value: 'standard', label: 'Standard' }, { value: 'pro', label: 'Pro' }]" /></KField>
      </div>
      <template #footer>
        <KButton variant="secondary" @click="open = false">Batal</KButton>
        <KButton :loading="busy" :disabled="!f.code || !f.name" @click="save">Simpan</KButton>
      </template>
    </KModal>
  </div>
</template>
