import type { RouteRecordRaw } from 'vue-router'

const roles = ['mahasiswa' as const]
export default [
  { path: '/krs', component: () => import('@/views/mahasiswa/KrsView.vue'), meta: { title: 'KRS', roles } },
  { path: '/jadwal', component: () => import('@/views/mahasiswa/JadwalView.vue'), meta: { title: 'Jadwal kuliah', roles } },
  { path: '/khs', component: () => import('@/views/mahasiswa/KhsView.vue'), meta: { title: 'KHS', roles } },
  { path: '/transkrip', component: () => import('@/views/mahasiswa/TranskripView.vue'), meta: { title: 'Transkrip', roles } },
  { path: '/profil', component: () => import('@/views/mahasiswa/ProfilView.vue'), meta: { title: 'Profil', roles } },
] satisfies RouteRecordRaw[]
