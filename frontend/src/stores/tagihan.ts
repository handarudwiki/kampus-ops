import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api/client'
import type { Tagihan } from '@/api/types'

/** Bills (FR-25..27). Simulated payments go through the fake VA webhook (EC-03). */
export const useTagihanStore = defineStore('tagihan', () => {
  const mine = ref<Tagihan[]>([])
  async function loadMine() { mine.value = await api.get<Tagihan[]>('/tagihan/me') }
  async function issueVa(id: string) { return api.post<Tagihan>(`/tagihan/${id}/va`) }
  async function pay(va_number: string, amount: number) { return api.post<Tagihan>('/simulator/pay', { va_number, amount }, { idempotent: true }) }
  async function cicilan(id: string, jumlah_cicilan: number) { return api.post<Tagihan>(`/tagihan/${id}/cicilan`, { jumlah_cicilan }) }
  return { mine, loadMine, issueVa, pay, cicilan }
})
