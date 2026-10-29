import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import KStatus from '@/design-system/components/KStatus.vue'
import KStatCard from '@/design-system/components/KStatCard.vue'

describe('design-system components', () => {
  it('KStatus renders the label and tone from the status map', () => {
    const w = mount(KStatus, { props: { domain: 'krs', value: 'disetujui' } })
    expect(w.text()).toBe('Disetujui')
    expect(w.html()).toContain('bg-green-soft')
  })
  it('KStatCard mutes the number tail', () => {
    const w = mount(KStatCard, { props: { label: 'Mahasiswa', value: '12.375' } })
    expect(w.find('.text-ink-3').text()).toBe('.375')
  })
})
