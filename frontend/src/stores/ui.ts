import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type Theme = 'system' | 'light' | 'dark'

export const useUiStore = defineStore('ui', () => {
  const theme = ref<Theme>(read() ?? 'system')
  const navOpen = ref(false)

  watch(theme, (t) => {
    const root = document.documentElement
    if (t === 'system') delete root.dataset.theme
    else root.dataset.theme = t
    try {
      if (t === 'system') localStorage.removeItem('ko-theme')
      else localStorage.setItem('ko-theme', t)
    } catch { /* ignore */ }
  }, { immediate: true })

  return { theme, navOpen }
})

function read(): Theme | null {
  try { return localStorage.getItem('ko-theme') as Theme | null } catch { return null }
}
