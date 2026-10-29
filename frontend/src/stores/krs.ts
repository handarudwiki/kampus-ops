import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '@/api/client'
import type { Krs, KrsMe, Offering } from '@/api/types'

/** Student-side KRS state (FR-16..20). Mutations send Idempotency-Key (EC-02). */
export const useKrsStore = defineStore('krs', () => {
  const me = ref<KrsMe | null>(null)
  const offerings = ref<Offering[]>([])
  const loading = ref(false)
  const pending = ref<string | null>(null) // kelas_id being added/removed

  const krs = computed(() => me.value?.krs ?? null)
  const editable = computed(() => !!me.value?.window.open && (!krs.value || ['draft', 'ditolak'].includes(krs.value.status)))
  const sksLeft = computed(() => (krs.value ? krs.value.max_sks - krs.value.total_sks : 0))

  async function load() {
    loading.value = true
    try {
      const [m, o] = await Promise.all([api.get<KrsMe>('/krs/me'), api.get<Offering[]>('/krs/me/offerings')])
      me.value = m
      offerings.value = o
    } finally {
      loading.value = false
    }
  }
  async function refreshOfferings() {
    offerings.value = await api.get<Offering[]>('/krs/me/offerings')
  }
  function setKrs(k: Krs) {
    if (me.value) me.value = { ...me.value, krs: k }
  }

  async function add(kelasId: string) {
    pending.value = kelasId
    try {
      setKrs(await api.post<Krs>('/krs/me/items', { kelas_id: kelasId }, { idempotent: true }))
      await refreshOfferings()
    } finally {
      pending.value = null
    }
  }
  async function remove(kelasId: string) {
    pending.value = kelasId
    try {
      setKrs(await api.del<Krs>(`/krs/me/items/${kelasId}`, { idempotent: true }))
      await refreshOfferings()
    } finally {
      pending.value = null
    }
  }
  async function submit() {
    setKrs(await api.post<Krs>('/krs/me/submit', {}, { idempotent: true }))
  }

  return { me, offerings, loading, pending, krs, editable, sksLeft, load, refreshOfferings, add, remove, submit }
})
