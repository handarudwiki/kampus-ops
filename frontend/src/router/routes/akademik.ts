import type { RouteRecordRaw } from 'vue-router'

export default [
  { path: '/kurikulum', component: () => import('@/views/akademik/KurikulumView.vue'), meta: { title: 'Kurikulum', roles: ['kaprodi', 'operator'] } },
  { path: '/kelas', component: () => import('@/views/akademik/KelasView.vue'), meta: { title: 'Jadwal & kelas', roles: ['kaprodi', 'operator'] } },
  { path: '/masa-studi', component: () => import('@/views/akademik/MasaStudiView.vue'), meta: { title: 'Masa studi', roles: ['kaprodi'] } },
] satisfies RouteRecordRaw[]
