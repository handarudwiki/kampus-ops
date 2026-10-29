<script setup lang="ts">
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { X } from 'lucide-vue-next'
import KIconButton from './KIconButton.vue'
/** Centered dialog on desktop, bottom sheet on mobile. Footer slot holds actions (right-aligned). */
const open = defineModel<boolean>({ required: true })
withDefaults(defineProps<{ title: string; description?: string; size?: 'sm' | 'md' | 'lg' }>(), { size: 'md' })
</script>

<template>
  <TransitionRoot :show="open" as="template">
    <Dialog class="relative z-50" @close="open = false">
      <TransitionChild enter="duration-150 ease-out" enter-from="opacity-0" leave="duration-100 ease-in" leave-to="opacity-0">
        <div class="fixed inset-0 bg-ink/30 backdrop-blur-[2px]" />
      </TransitionChild>
      <div class="fixed inset-0 flex items-end justify-center sm:items-center sm:p-4">
        <TransitionChild
          enter="duration-200 ease-out" enter-from="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
          leave="duration-100 ease-in" leave-to="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
          as="template"
        >
          <DialogPanel
            :class="[
              'flex max-h-[90vh] w-full flex-col rounded-t-card bg-surface shadow-pop sm:rounded-card',
              { sm: 'sm:max-w-md', md: 'sm:max-w-xl', lg: 'sm:max-w-3xl' }[size],
            ]"
          >
            <div class="flex items-start justify-between gap-4 px-6 pt-6">
              <div>
                <DialogTitle class="text-md font-bold">{{ title }}</DialogTitle>
                <p v-if="description" class="mt-1 text-sm text-ink-3">{{ description }}</p>
              </div>
              <KIconButton label="Tutup" @click="open = false"><X class="size-4" /></KIconButton>
            </div>
            <div class="ko-scroll overflow-y-auto px-6 py-5"><slot /></div>
            <div v-if="$slots.footer" class="flex flex-col-reverse gap-2 border-t border-line px-6 py-4 sm:flex-row sm:justify-end">
              <slot name="footer" />
            </div>
          </DialogPanel>
        </TransitionChild>
      </div>
    </Dialog>
  </TransitionRoot>
</template>
