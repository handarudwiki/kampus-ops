import { describe, expect, it } from 'vitest'
import { ERROR_TEXT, STATUS, errorText, statusOf } from '@/design-system/status'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('status map', () => {
  it('uses one tone per meaning across domains', () => {
    expect(STATUS.krs.disetujui.tone).toBe('green')
    expect(STATUS.tagihan.lunas.tone).toBe('green')
    expect(STATUS.krs.ditolak.tone).toBe('red')
    expect(STATUS.tagihan.belum_bayar.tone).toBe('red')
    expect(STATUS.krs.diajukan.tone).toBe('amber')
  })
  it('falls back gracefully for unknown values', () => {
    expect(statusOf('krs', 'xyz')).toEqual({ label: 'xyz', tone: 'gray' })
    expect(statusOf('krs', null).label).toBe('—')
  })
  it('has a message for every business error code in the API contract', () => {
    const contract = readFileSync(resolve(__dirname, '../../../docs/API_CONTRACT.md'), 'utf8')
    const block = contract.split('### Error codes (business)')[1]!.split('---')[0]!
    const codes = block.match(/[A-Z][A-Z_]{3,}/g) ?? []
    expect(codes.length).toBeGreaterThan(10)
    for (const c of codes.filter((c) => c !== 'AUTH_TOKEN_EXPIRED')) expect(ERROR_TEXT[c], c).toBeTruthy()
    expect(errorText('NOPE', 'fallback')).toBe('fallback')
  })
})
