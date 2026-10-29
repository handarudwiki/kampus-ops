import type { RouteRecordRaw } from 'vue-router'

export default [
  { path: '/pddikti', component: () => import('@/views/pelaporan/PddiktiView.vue'), meta: { title: 'Validator PDDikti', roles: ['operator', 'admin_kampus'] } },
  { path: '/audit-log', component: () => import('@/views/pelaporan/AuditLogView.vue'), meta: { title: 'Audit log', roles: ['operator', 'admin_kampus', 'keuangan'] } },
] satisfies RouteRecordRaw[]
