import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api/client'
import type { Periode } from '@/api/types'

export const usePeriodeStore = defineStore('periode', () => {
  const active = ref<Periode | null>(null)
  const all = ref<Periode[]>([])

  async function load() {
    try { active.value = await api.get<Periode>('/periode/active') } catch { active.value = null }
  }
  async function loadAll() {
    try { all.value = (await api.list<Periode>('/periode', { per_page: 100 })).data } catch { all.value = [] }
  }
  return { active, all, load, loadAll }
})
