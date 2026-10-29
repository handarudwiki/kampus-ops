import { ref, shallowRef, type Ref } from 'vue'
import { toastError } from '@/design-system/feedback'

/**
 * Standard load pattern: data + loading + error + reload().
 * Every page uses this so loading skeletons and error toasts behave the same everywhere.
 */
export function useAsync<T>(fn: () => Promise<T>, initial: T, opts: { immediate?: boolean; silent?: boolean } = {}) {
  const data = shallowRef(initial) as Ref<T>
  const loading = ref(false)
  const error = ref<unknown>(null)
  async function reload() {
    loading.value = true
    error.value = null
    try {
      data.value = await fn()
    } catch (e) {
      error.value = e
      if (!opts.silent) toastError(e, 'Gagal memuat data')
    } finally {
      loading.value = false
    }
  }
  if (opts.immediate !== false) reload()
  return { data, loading, error, reload }
}

/** Wrap a mutation: sets busy, shows success toast, reports errors. Returns true on success. */
export function useAction() {
  const busy = ref(false)
  async function run(fn: () => Promise<unknown>, success?: string, failTitle = 'Gagal'): Promise<boolean> {
    busy.value = true
    try {
      await fn()
      if (success) (await import('@/design-system/feedback')).toast('success', success)
      return true
    } catch (e) {
      toastError(e, failTitle)
      return false
    } finally {
      busy.value = false
    }
  }
  return { busy, run }
}
