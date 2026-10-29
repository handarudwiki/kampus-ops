<script setup lang="ts">
import { CheckCircle2, Info, X, XCircle } from 'lucide-vue-next'
import { dismiss, toasts } from '../feedback'
const icon = { success: CheckCircle2, error: XCircle, info: Info }
const color = { success: 'text-green', error: 'text-red', info: 'text-blue' }
</script>

<template>
  <div class="pointer-events-none fixed inset-x-4 bottom-4 z-[60] flex flex-col items-center gap-2 sm:inset-x-auto sm:right-6 sm:items-end" aria-live="polite">
    <TransitionGroup enter-active-class="transition duration-200 ease-out" enter-from-class="opacity-0 translate-y-2" leave-active-class="transition duration-150" leave-to-class="opacity-0">
      <div v-for="t in toasts" :key="t.id" class="pointer-events-auto flex w-full max-w-sm gap-3 rounded-control border border-line bg-surface p-4 shadow-pop">
        <component :is="icon[t.level]" :class="['mt-0.5 size-4 shrink-0', color[t.level]]" />
        <div class="min-w-0 flex-1">
          <p class="font-bold">{{ t.title }}</p>
          <p v-if="t.body" class="mt-0.5 text-sm text-ink-2">{{ t.body }}</p>
        </div>
        <button class="text-ink-3 hover:text-ink" aria-label="Tutup" @click="dismiss(t.id)"><X class="size-4" /></button>
      </div>
    </TransitionGroup>
  </div>
</template>
