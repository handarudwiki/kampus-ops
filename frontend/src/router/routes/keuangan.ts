import type { RouteRecordRaw } from 'vue-router'

const roles = ['keuangan' as const]
export default [
  { path: '/tagihan', component: () => import('@/views/keuangan/TagihanSayaView.vue'), meta: { title: 'Tagihan', roles: ['mahasiswa'] } },
  { path: '/keuangan/tagihan', component: () => import('@/views/keuangan/TagihanAdminView.vue'), meta: { title: 'Tagihan mahasiswa', roles } },
  { path: '/keuangan/dispensasi', component: () => import('@/views/keuangan/DispensasiView.vue'), meta: { title: 'Dispensasi', roles: ['keuangan', 'operator'] } },
  { path: '/keuangan/piutang', component: () => import('@/views/keuangan/PiutangView.vue'), meta: { title: 'Piutang', roles } },
] satisfies RouteRecordRaw[]
