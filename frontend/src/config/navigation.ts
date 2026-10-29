import type { Component } from 'vue'
import {
  BadgePercent, BookOpen, Building2, CalendarDays, CalendarRange, ClipboardCheck, ClipboardList, DoorOpen,
  FileCheck2, FileClock, FileSpreadsheet, GraduationCap, HandCoins, History, Home, Landmark, Layers,
  ListChecks, Palette, Receipt, ScrollText, ShieldCheck, UserCircle, UserPlus, Users, UsersRound, Wallet,
} from 'lucide-vue-next'
import type { Role } from '@/api/types'

export interface NavItem { label: string; to: string; icon: Component }
export interface NavGroup { label?: string; items: NavItem[] }

const home: NavItem = { label: 'Beranda', to: '/', icon: Home }

/** Sidebar per active role. Order follows the semester cycle in the BRD. */
export const NAV: Record<Role, NavGroup[]> = {
  mahasiswa: [
    { items: [home] },
    { label: 'Akademik', items: [
      { label: 'KRS', to: '/krs', icon: ClipboardList },
      { label: 'Jadwal kuliah', to: '/jadwal', icon: CalendarDays },
      { label: 'KHS', to: '/khs', icon: FileCheck2 },
      { label: 'Transkrip', to: '/transkrip', icon: ScrollText },
    ] },
    { label: 'Keuangan', items: [{ label: 'Tagihan', to: '/tagihan', icon: Receipt }] },
    { label: 'Akun', items: [{ label: 'Profil', to: '/profil', icon: UserCircle }] },
  ],
  dosen_wali: [
    { items: [home] },
    { label: 'Perwalian', items: [{ label: 'Persetujuan KRS', to: '/perwalian', icon: ClipboardCheck }] },
  ],
  dosen: [
    { items: [home] },
    { label: 'Perkuliahan', items: [
      { label: 'Kelas saya', to: '/mengajar', icon: BookOpen },
      { label: 'Koreksi nilai', to: '/koreksi-nilai', icon: FileClock },
    ] },
  ],
  kaprodi: [
    { items: [home] },
    { label: 'Program studi', items: [
      { label: 'Kurikulum', to: '/kurikulum', icon: Layers },
      { label: 'Jadwal & kelas', to: '/kelas', icon: CalendarRange },
      { label: 'Koreksi nilai', to: '/koreksi-nilai', icon: FileClock },
      { label: 'Masa studi', to: '/masa-studi', icon: History },
    ] },
  ],
  operator: [
    { items: [home] },
    { label: 'Data induk', items: [
      { label: 'Mahasiswa', to: '/mahasiswa', icon: GraduationCap },
      { label: 'Dosen', to: '/dosen', icon: UsersRound },
      { label: 'Prodi & fakultas', to: '/prodi', icon: Building2 },
      { label: 'Ruang', to: '/ruang', icon: DoorOpen },
      { label: 'Kurikulum', to: '/kurikulum', icon: Layers },
    ] },
    { label: 'Semester', items: [
      { label: 'Jadwal & kelas', to: '/kelas', icon: CalendarRange },
      { label: 'PMB', to: '/pmb', icon: UserPlus },
      { label: 'Validator PDDikti', to: '/pddikti', icon: ShieldCheck },
      { label: 'Audit log', to: '/audit-log', icon: ListChecks },
    ] },
  ],
  keuangan: [
    { items: [home] },
    { label: 'Keuangan', items: [
      { label: 'Tagihan', to: '/keuangan/tagihan', icon: Wallet },
      { label: 'Dispensasi', to: '/keuangan/dispensasi', icon: BadgePercent },
      { label: 'Piutang', to: '/keuangan/piutang', icon: HandCoins },
      { label: 'Audit log', to: '/audit-log', icon: ListChecks },
    ] },
  ],
  admin_kampus: [
    { items: [home] },
    { label: 'Kampus', items: [
      { label: 'Pengguna & peran', to: '/pengguna', icon: Users },
      { label: 'Periode akademik', to: '/periode', icon: CalendarRange },
      { label: 'Prodi & fakultas', to: '/prodi', icon: Building2 },
      { label: 'Audit log', to: '/audit-log', icon: ListChecks },
    ] },
  ],
  super_admin: [
    { items: [home] },
    { label: 'Vendor', items: [{ label: 'Tenant kampus', to: '/tenants', icon: Landmark }] },
  ],
  calon_mahasiswa: [
    { items: [{ label: 'Status pendaftaran', to: '/', icon: FileSpreadsheet }] },
  ],
}

/** Always shown at the bottom of the sidebar for every role. */
export const NAV_FOOTER: NavItem[] = [{ label: 'Design system', to: '/design-system', icon: Palette }]
