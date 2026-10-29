import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useNilaiStore } from '@/stores/nilai'
import type { NilaiRow } from '@/api/types'

const row = (komponen: Record<string, number | null>): NilaiRow => ({
  krs_detail_id: 'r1', nilai_id: null, mahasiswa_id: 'm', nim: '1', nama: 'A', komponen,
  angka: null, huruf: null, bobot_x100: null, final: false, koreksi_pending: false,
})

describe('nilai preview (EC-10)', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('computes the weighted score without float drift', () => {
    const s = useNilaiStore()
    s.server = { komponen: [{ key: 'tugas', label: 'Tugas', bobot_persen: 20 }, { key: 'uts', label: 'UTS', bobot_persen: 30 }, { key: 'uas', label: 'UAS', bobot_persen: 50 }], rows: [], finalized: false, deadline: '' }
    const r = row({ tugas: 85.5, uts: 72.25, uas: 90 })
    s.draft = { r1: r.komponen }
    // 85.5*.2 + 72.25*.3 + 90*.5 = 17.1 + 21.675 + 45 = 83.775 → 83.78
    expect(s.preview(r)).toBe('83.78')
  })

  it('returns null while a component is missing', () => {
    const s = useNilaiStore()
    s.server = { komponen: [{ key: 'uts', label: 'UTS', bobot_persen: 50 }, { key: 'uas', label: 'UAS', bobot_persen: 50 }], rows: [], finalized: false, deadline: '' }
    const r = row({ uts: 80, uas: null })
    s.draft = { r1: r.komponen }
    expect(s.preview(r)).toBeNull()
  })

  it('is stable for values that break naive float sums', () => {
    const s = useNilaiStore()
    s.server = { komponen: [{ key: 'a', label: 'A', bobot_persen: 10 }, { key: 'b', label: 'B', bobot_persen: 90 }], rows: [], finalized: false, deadline: '' }
    const r = row({ a: 0.1, b: 0.2 })
    s.draft = { r1: r.komponen }
    expect(s.preview(r)).toBe('0.19')
  })
})
