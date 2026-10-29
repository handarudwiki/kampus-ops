<script setup lang="ts">
import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/vue'
/** Dropdown menu. items: [{ label, onClick, danger?, icon? }] */
defineProps<{
  items: { label: string; onClick: () => void; danger?: boolean; icon?: unknown; active?: boolean; hint?: string }[]
  align?: 'left' | 'right'
  up?: boolean
}>()
</script>

<template>
  <Menu as="div" class="relative inline-block text-left">
    <MenuButton as="template"><slot /></MenuButton>
    <transition enter-active-class="transition duration-100 ease-out" enter-from-class="opacity-0 scale-95" leave-active-class="transition duration-75 ease-in" leave-to-class="opacity-0 scale-95">
      <MenuItems
        :class="[
          'absolute z-40 min-w-52 rounded-control border border-line bg-surface p-1.5 shadow-pop focus:outline-none',
          align === 'left' ? 'left-0' : 'right-0',
          up ? 'bottom-full mb-2 origin-bottom' : 'mt-2 origin-top-right',
        ]"
      >
        <slot name="header" />
        <MenuItem v-for="it in items" :key="it.label" v-slot="{ active: hover }">
          <button
            type="button"
            :class="[
              'flex w-full items-center gap-2.5 rounded-[8px] px-2.5 py-2 text-left text-base font-medium',
              hover && 'bg-surface-2',
              it.danger ? 'text-red-ink' : 'text-ink',
            ]"
            @click="it.onClick"
          >
            <component :is="it.icon" v-if="it.icon" class="size-4 text-ink-3" />
            <span class="flex-1">{{ it.label }}</span>
            <span v-if="it.hint" class="text-xs text-ink-3">{{ it.hint }}</span>
            <span v-if="it.active" class="size-2 rounded-pill bg-accent" />
          </button>
        </MenuItem>
      </MenuItems>
    </transition>
  </Menu>
</template>
