import type { Tone } from '@/api/types'

/**
 * Tone → class names. Written out literally so Tailwind's scanner keeps them.
 * dot:  solid marker   soft: tinted chip   bar: fills   text: accent text   tint: large tinted panel
 */
export const toneClass: Record<Tone, { dot: string; soft: string; bar: string; text: string; tint: string; stroke: string; fill: string }> = {
  blue:   { dot: 'bg-blue',   soft: 'bg-blue-soft text-blue-ink',     bar: 'bg-blue',   text: 'text-blue-ink',   tint: 'bg-blue-soft',   stroke: 'stroke-blue',   fill: 'fill-blue' },
  green:  { dot: 'bg-green',  soft: 'bg-green-soft text-green-ink',   bar: 'bg-green',  text: 'text-green-ink',  tint: 'bg-green-soft',  stroke: 'stroke-green',  fill: 'fill-green' },
  amber:  { dot: 'bg-amber',  soft: 'bg-amber-soft text-amber-ink',   bar: 'bg-amber',  text: 'text-amber-ink',  tint: 'bg-amber-soft',  stroke: 'stroke-amber',  fill: 'fill-amber' },
  red:    { dot: 'bg-red',    soft: 'bg-red-soft text-red-ink',       bar: 'bg-red',    text: 'text-red-ink',    tint: 'bg-red-soft',    stroke: 'stroke-red',    fill: 'fill-red' },
  violet: { dot: 'bg-violet', soft: 'bg-violet-soft text-violet-ink', bar: 'bg-violet', text: 'text-violet-ink', tint: 'bg-violet-soft', stroke: 'stroke-violet', fill: 'fill-violet' },
  pink:   { dot: 'bg-pink',   soft: 'bg-pink-soft text-pink-ink',     bar: 'bg-pink',   text: 'text-pink-ink',   tint: 'bg-pink-soft',   stroke: 'stroke-pink',   fill: 'fill-pink' },
  gray:   { dot: 'bg-gray',   soft: 'bg-gray-soft text-gray-ink',     bar: 'bg-gray',   text: 'text-gray-ink',   tint: 'bg-gray-soft',   stroke: 'stroke-gray',   fill: 'fill-gray' },
}

/** Fixed order for chart series so "series 1" is always blue, "series 2" green, etc. */
export const SERIES_TONES: Tone[] = ['blue', 'green', 'amber', 'violet', 'pink', 'red']

/** Stable tone from a string (avatars, tags) */
export function toneFor(s: string): Tone {
  let h = 0
  for (const ch of s) h = (h * 31 + ch.charCodeAt(0)) | 0
  return SERIES_TONES[Math.abs(h) % SERIES_TONES.length]!
}
