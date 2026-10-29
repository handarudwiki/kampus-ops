import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import type { Periode } from '@/api/types'

export interface Phase { key: string; label: string; start: string; end: string }

/** Ordered semester phases derived from periode.tahapan (FR-06). */
export function phasesOf(p: Periode): Phase[] {
  const t = p.tahapan
  const list: Phase[] = [
    { key: 'krs', label: 'KRS', start: t.krs_mulai, end: t.krs_selesai },
    ...(t.revisi_mulai && t.revisi_selesai ? [{ key: 'revisi', label: 'Revisi KRS', start: t.revisi_mulai, end: t.revisi_selesai }] : []),
    { key: 'kuliah', label: 'Perkuliahan', start: t.kuliah_mulai, end: t.kuliah_selesai },
    { key: 'uts', label: 'UTS', start: t.uts, end: t.uts },
    { key: 'uas', label: 'UAS', start: t.uas, end: t.uas },
    { key: 'nilai', label: 'Input nilai', start: t.uas, end: t.nilai_selesai },
  ]
  return list.filter((x) => x.start && x.end)
}

const day = (s: string) => new Date(s.slice(0, 10) + 'T00:00:00').getTime()

/** The phase "now" falls in; UTS/UAS (single days) win over the surrounding lecture period. */
export function currentPhase(phases: Phase[], now = Date.now()): Phase | null {
  const today = day(new Date(now).toISOString())
  const hits = phases.filter((p) => day(p.start) <= today && today <= day(p.end))
  return hits.find((p) => p.start === p.end) ?? hits.at(-1) ?? null
}

export const usePeriodeStore = defineStore('periode', () => {
  const active = ref<Periode | null>(null)
  const all = ref<Periode[]>([])

  const phases = computed(() => (active.value ? phasesOf(active.value) : []))
  const phase = computed(() => currentPhase(phases.value))
  const label = computed(() => (active.value ? `${active.value.tipe} ${active.value.tahun}/${active.value.tahun + 1}` : ''))

  async function load() {
    try { active.value = await api.get<Periode>('/periode/active') } catch { active.value = null }
  }
  async function loadAll() {
    try { all.value = (await api.list<Periode>('/periode', { per_page: 100 })).data } catch { all.value = [] }
  }
  return { active, all, phases, phase, label, load, loadAll }
})
