<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { confirmState } from '../feedback'
import KButton from './KButton.vue'
import KField from './KField.vue'
import KModal from './KModal.vue'
import KTextarea from './KTextarea.vue'

const req = computed(() => confirmState.current)
const reason = ref('')
const open = computed({
  get: () => !!req.value,
  set: (v) => { if (!v) finish(false) },
})
watch(req, () => (reason.value = ''))
function finish(ok: boolean) {
  const r = confirmState.current
  if (!r) return
  confirmState.current = null
  r.resolve({ ok, reason: reason.value })
}
</script>

<template>
  <KModal v-if="req" v-model="open" :title="req.title" :description="req.body" size="sm">
    <KField v-if="req.withReason" :label="req.reasonLabel ?? 'Alasan'" required>
      <KTextarea v-model="reason" />
    </KField>
    <template #footer>
      <KButton variant="secondary" @click="finish(false)">Batal</KButton>
      <KButton :variant="req.danger ? 'danger' : 'primary'" :disabled="req.withReason && !reason.trim()" @click="finish(true)">
        {{ req.confirmLabel ?? 'Ya, lanjutkan' }}
      </KButton>
    </template>
  </KModal>
</template>
