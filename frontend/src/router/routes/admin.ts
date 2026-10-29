import type { RouteRecordRaw } from 'vue-router'

export default [
  { path: '/pengguna', component: () => import('@/views/admin/PenggunaView.vue'), meta: { title: 'Pengguna & peran', roles: ['admin_kampus'] } },
  { path: '/tenants', component: () => import('@/views/admin/TenantsView.vue'), meta: { title: 'Tenant kampus', roles: ['super_admin'] } },
] satisfies RouteRecordRaw[]
