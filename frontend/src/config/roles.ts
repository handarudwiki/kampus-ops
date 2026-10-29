import type { Role } from '@/api/types'

export const ROLE_LABEL: Record<Role, string> = {
  super_admin: 'Super Admin',
  admin_kampus: 'Admin Kampus',
  operator: 'Operator Akademik',
  kaprodi: 'Kaprodi',
  dosen_wali: 'Dosen Wali',
  dosen: 'Dosen Pengampu',
  keuangan: 'Bagian Keuangan',
  mahasiswa: 'Mahasiswa',
  calon_mahasiswa: 'Calon Mahasiswa',
}

export const ALL_ROLES = Object.keys(ROLE_LABEL) as Role[]
/** Roles that operate on the whole campus's academic data */
export const STAFF: Role[] = ['admin_kampus', 'operator']
