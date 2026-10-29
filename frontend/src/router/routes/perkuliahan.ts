import type { RouteRecordRaw } from 'vue-router'

const kelasRoles = ['dosen' as const, 'operator' as const, 'kaprodi' as const, 'admin_kampus' as const]
export default [
  { path: '/mengajar', component: () => import('@/views/dosen/MengajarView.vue'), meta: { title: 'Kelas saya', roles: ['dosen'] } },
  { path: '/kelas/:id', component: () => import('@/views/dosen/KelasDetailView.vue'), meta: { title: 'Kelas', roles: kelasRoles } },
  { path: '/kelas/:id/nilai', component: () => import('@/views/dosen/KelasDetailView.vue'), meta: { title: 'Nilai kelas', roles: kelasRoles } },
  { path: '/koreksi-nilai', component: () => import('@/views/dosen/KoreksiNilaiView.vue'), meta: { title: 'Koreksi nilai', roles: ['dosen', 'kaprodi'] } },
] satisfies RouteRecordRaw[]
