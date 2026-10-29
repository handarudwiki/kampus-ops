import { onBeforeUnmount, onMounted, ref, type Ref } from 'vue'

/** Track an element's width so SVG charts draw in real pixels instead of stretching a viewBox. */
export function useWidth(el: Ref<HTMLElement | null>, fallback = 600) {
  const width = ref(fallback)
  let ro: ResizeObserver | null = null
  onMounted(() => {
    if (!el.value) return
    width.value = el.value.clientWidth || fallback
    ro = new ResizeObserver(([e]) => { if (e) width.value = Math.max(120, e.contentRect.width) })
    ro.observe(el.value)
  })
  onBeforeUnmount(() => ro?.disconnect())
  return width
}
