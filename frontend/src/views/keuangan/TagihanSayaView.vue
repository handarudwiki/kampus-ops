<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { CheckCircle2, Copy, CreditCard } from 'lucide-vue-next'
import type { Tagihan } from '@/api/types'
import { date, rupiah, STATUS, toast, toastError } from '@/design-system'
import { useTagihanStore } from '@/stores/tagihan'
import PayModal from './PayModal.vue'

const store = useTagihanStore()
const loading = ref(true)
const load = () => store.loadMine().catch((e) => toastError(e)).finally(() => (loading.value = false))
onMounted(load)
const sisa = computed(() => store.mine.reduce((a, t) => a + (t.jumlah - t.terbayar), 0))
const paying = ref<Tagihan | null>(null)
const payOpen = ref(false)
function pay(t: Tagihan) { paying.value = t; payOpen.value = true }
async function copy(v: string) {
  try { await navigator.clipboard.writeText(v); toast('info', 'Nomor VA disalin') } catch { /* ignore */ }
}
</script>

<template>
  <div>
    <KPageHeader title="Tagihan" description="KRS terbuka bila tagihan wajib lunas atau ada cicilan/dispensasi yang disetujui." />
    <div class="mb-3 grid gap-3 sm:grid-cols-2">
      <KStatCard label="Sisa tagihan" :value="rupiah(sisa)" :tone="sisa > 0 ? 'red' : 'green'" :hint="sisa > 0 ? 'Belum lunas' : 'Semua lunas'" />
      <KStatCard label="Jumlah tagihan" :value="String(store.mine.length)" tone="blue" />
    </div>
    <div v-if="loading" class="h-48 animate-pulse rounded-card bg-surface" />
    <div v-else class="grid gap-3 lg:grid-cols-2">
      <KCard v-for="t in store.mine" :key="t.id" :title="`${STATUS.jenisTagihan[t.jenis].label} · ${t.periode_kode}`" :subtitle="`Jatuh tempo ${date(t.jatuh_tempo)}`">
        <template #actions><KStatus domain="tagihan" :value="t.status" /></template>
        <div class="flex flex-col gap-4">
          <div>
            <p class="tabular text-xl font-bold">{{ rupiah(t.terbayar) }}<span class="text-md text-ink-3"> / {{ rupiah(t.jumlah) }}</span></p>
            <KProgress class="mt-2" :value="t.terbayar" :max="t.jumlah" :tone="t.status === 'lunas' ? 'green' : 'amber'" />
          </div>
          <div v-if="t.va_number" class="flex items-center justify-between rounded-control bg-surface-2 px-4 py-3">
            <div><p class="ko-eyebrow">Virtual account</p><p class="font-mono font-bold">{{ t.va_number }}</p></div>
            <KIconButton label="Salin VA" @click="copy(t.va_number)"><Copy class="size-4" /></KIconButton>
          </div>
          <div v-if="t.cicilan.length">
            <p class="ko-eyebrow mb-2">Cicilan</p>
            <ul class="flex flex-col gap-2">
              <li v-for="c in t.cicilan" :key="c.ke" class="flex items-center gap-3 text-sm">
                <CheckCircle2 :class="['size-4', c.lunas ? 'text-green' : 'text-ink-3']" />
                <span class="flex-1">Cicilan {{ c.ke }} · {{ date(c.jatuh_tempo) }}</span>
                <span class="tabular font-semibold">{{ rupiah(c.jumlah) }}</span>
              </li>
            </ul>
          </div>
          <KButton v-if="t.status !== 'lunas'" block @click="pay(t)"><template #icon><CreditCard class="size-4" /></template>Bayar (simulasi)</KButton>
        </div>
      </KCard>
      <KCard v-if="!store.mine.length" class="lg:col-span-2"><KEmpty title="Tidak ada tagihan" /></KCard>
    </div>
    <PayModal v-model="payOpen" :tagihan="paying" @paid="load" />
  </div>
</template>
