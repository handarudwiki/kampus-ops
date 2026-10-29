import type { RouteRecordRaw } from 'vue-router'

export default [
  { path: '/perwalian', component: () => import('@/views/dosen/PerwalianView.vue'), meta: { title: 'Persetujuan KRS', roles: ['dosen_wali'] } },
  { path: '/krs/:id', component: () => import('@/views/dosen/KrsDetailView.vue'), meta: { title: 'Detail KRS', roles: ['dosen_wali', 'operator', 'kaprodi'] } },
] satisfies RouteRecordRaw[]
