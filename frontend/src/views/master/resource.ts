import type { Component } from 'vue'
import { api } from '@/api/client'
import type { StatusDomain } from '@/design-system'
import type { Column } from '@/design-system/components/KTable.vue'

export type Opt = { value: string | number; label: string }

export interface FieldDef {
  /** dot paths allowed: "tahapan.krs_mulai" */
  key: string
  label: string
  type?: 'text' | 'email' | 'number' | 'date' | 'select'
  required?: boolean
  hint?: string
  options?: Opt[] | (() => Promise<Opt[]>)
  /** column span in the 2-col form grid */
  wide?: boolean
  /** only on create */
  createOnly?: boolean
}

export interface ColumnDef extends Column {
  status?: StatusDomain
  mono?: boolean
  format?: (v: unknown, row: Record<string, unknown>) => string
}

export interface FilterDef { key: string; label: string; options: Opt[] | (() => Promise<Opt[]>) }

export interface RowAction {
  label: string
  icon?: Component
  show?: (row: Record<string, unknown>) => boolean
  run: (row: Record<string, unknown>) => Promise<unknown>
  confirm?: string
  success?: string
}

/** Declarative description of a master-data screen. ResourcePage renders it. */
export interface ResourceConfig {
  title: string
  description: string
  endpoint: string
  singular: string
  columns: ColumnDef[]
  fields: FieldDef[]
  filters?: FilterDef[]
  rowActions?: RowAction[]
  /** route to open on row click, e.g. id => `/mahasiswa/${id}` */
  detail?: (id: string) => string
  canDelete?: boolean
  defaults?: Record<string, unknown>
}

/** Lazy option loaders shared across resources */
const cache = new Map<string, Promise<Opt[]>>()
export function optionsFrom(endpoint: string, label: (r: Record<string, unknown>) => string): () => Promise<Opt[]> {
  return () => {
    if (!cache.has(endpoint)) {
      cache.set(endpoint, api.list<Record<string, unknown>>(endpoint, { per_page: 100 })
        .then((r) => r.data.map((x) => ({ value: x.id as string, label: label(x) })))
        .catch(() => { cache.delete(endpoint); return [] }))
    }
    return cache.get(endpoint)!
  }
}
export const clearOptionCache = () => cache.clear()

export function getPath(obj: Record<string, unknown>, path: string): unknown {
  return path.split('.').reduce<unknown>((o, k) => (o == null ? o : (o as Record<string, unknown>)[k]), obj)
}
export function setPath(obj: Record<string, unknown>, path: string, v: unknown) {
  const keys = path.split('.')
  let o = obj
  for (const k of keys.slice(0, -1)) o = (o[k] ??= {}) as Record<string, unknown>
  o[keys.at(-1)!] = v
}
