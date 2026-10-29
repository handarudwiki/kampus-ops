<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Tagihan } from '@/api/types'
import { useAction } from '@/composables/useAsync'
import { rupiah } from '@/design-system'
import { useTagihanStore } from '@/stores/tagihan'

/** Payment simulator: posts to the fake VA which fires the webhook server-side. */
const open = defineModel<boolean>({ required: true })
const props = defineProps<{ tagihan: Tagihan | null }>()
const emit = defineEmits<{ paid: [] }>()
const store = useTagihanStore()
const amount = ref(0)
watch(() => props.tagihan, (t) => {
  if (!t) return
  const next = t.cicilan.find((c) => !c.lunas)
  amount.value = next ? next.jumlah : t.jumlah - t.terbayar
}, { immediate: true })
const { run, busy } = useAction()
async function pay() {
  const t = props.tagihan!
  if (await run(async () => {
    const va = t.va_number ?? (await store.issueVa(t.id)).va_number!
    await store.pay(va, amount.value)
  }, 'Pembayaran diterima')) {
    open.value = false
    emit('paid')
  }
}
</script>

<template>
  <KModal v-model="open" title="Simulasi pembayaran" description="Mengirim pembayaran ke virtual account palsu; status tagihan dihitung ulang dari total pembayaran." size="sm">
    <div v-if="tagihan" class="flex flex-col gap-4">
      <div class="rounded-control bg-surface-2 p-4">
        <p class="ko-eyebrow">Virtual account</p>
        <p class="mt-1 font-mono text-md font-bold">{{ tagihan.va_number ?? 'Diterbitkan saat bayar' }}</p>
        <p class="mt-2 text-sm text-ink-3">Sisa tagihan {{ rupiah(tagihan.jumlah - tagihan.terbayar) }}</p>
      </div>
      <KField label="Jumlah dibayar (Rp)" required><KInput v-model.number="amount" type="number" min="1" /></KField>
    </div>
    <template #footer>
      <KButton variant="secondary" @click="open = false">Batal</KButton>
      <KButton :loading="busy" :disabled="amount <= 0" @click="pay">Bayar {{ rupiah(amount) }}</KButton>
    </template>
  </KModal>
</template>
