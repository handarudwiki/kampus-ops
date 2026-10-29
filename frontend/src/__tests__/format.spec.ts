import { describe, expect, it } from 'vitest'
import { bobot, ip, rupiah, slot, splitFigure } from '@/design-system/format'

describe('format', () => {
  it('renders bobot_x100 with integer math (EC-10)', () => {
    expect(bobot(375)).toBe('3,75')
    expect(bobot(400)).toBe('4,00')
    expect(bobot(5)).toBe('0,05')
    expect(bobot(null)).toBe('—')
  })
  it('shows IPK with a decimal comma', () => {
    expect(ip('3.42')).toBe('3,42')
    expect(ip(null)).toBe('—')
  })
  it('formats rupiah without decimals', () => {
    expect(rupiah(4500000).replace(/\s/g, ' ')).toMatch(/Rp\s?4\.500\.000/)
  })
  it('splits a figure into strong head and muted tail', () => {
    expect(splitFigure('12.375')).toEqual(['12', '.375'])
    expect(splitFigure('3,42')).toEqual(['3', ',42'])
    expect(splitFigure('24')).toEqual(['24', ''])
  })
  it('formats a schedule slot', () => {
    expect(slot({ hari: 1, jam_mulai: '07:30', jam_selesai: '10:00' })).toBe('Senin, 07:30–10:00')
  })
})
