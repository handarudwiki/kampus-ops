import type { RouteRecordRaw } from 'vue-router'

export default [
  { path: '/pmb', component: () => import('@/views/pmb/PmbAdminView.vue'), meta: { title: 'PMB', roles: ['operator', 'admin_kampus'] } },
  { path: '/daftar', component: () => import('@/views/pmb/DaftarView.vue'), meta: { title: 'Daftar PMB', public: true, bare: true } },
] satisfies RouteRecordRaw[]
