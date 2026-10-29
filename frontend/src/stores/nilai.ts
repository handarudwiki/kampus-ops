import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api/client'
import type { NilaiKelas, NilaiRow } from '@/api/types'

/** Grade entry for one class (FR-22, FR-23). Keeps an editable draft separate from server state. */
export const useNilaiStore = defineStore('nilai', () => {
  const kelasId = ref<string | null>(null)
  const server = ref<NilaiKelas | null>(null)
  const draft = ref<Record<string, Record<string, number | null>>>({})
  const dirty = ref(false)

  function hydrate(n: NilaiKelas) {
    server.value = n
    draft.value = Object.fromEntries(n.rows.map((r) => [r.krs_detail_id, { ...r.komponen }]))
    dirty.value = false
  }
  async function load(id: string) {
    kelasId.value = id
    hydrate(await api.get<NilaiKelas>(`/kelas/${id}/nilai`))
  }
  function set(row: string, key: string, v: number | null) {
    draft.value[row] = { ...draft.value[row], [key]: v }
    dirty.value = true
  }
  /** Weighted preview with integer math (×100) so it matches the backend rounding (EC-10). */
  function preview(row: NilaiRow): string | null {
    const n = server.value
    if (!n) return null
    const vals = draft.value[row.krs_detail_id] ?? {}
    let total = 0
    for (const k of n.komponen) {
      const v = vals[k.key]
      if (v == null || Number.isNaN(v)) return null
      total += Math.round(v * 100) * k.bobot_persen
    }
    return (Math.round(total / 100) / 100).toFixed(2)
  }
  async function save() {
    const rows = Object.entries(draft.value).map(([krs_detail_id, komponen]) => ({ krs_detail_id, komponen }))
    hydrate(await api.put<NilaiKelas>(`/kelas/${kelasId.value}/nilai`, { rows }))
  }
  async function finalize() {
    await api.post(`/kelas/${kelasId.value}/nilai/finalize`)
    await load(kelasId.value!)
  }
  return { kelasId, server, draft, dirty, load, set, preview, save, finalize }
})
