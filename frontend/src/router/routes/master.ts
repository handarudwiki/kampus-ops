import type { RouteRecordRaw } from 'vue-router'

const staff = ['operator' as const, 'admin_kampus' as const]
const master = () => import('@/views/master/MasterView.vue')
export default [
  { path: '/mahasiswa', component: master, props: { resource: 'mahasiswa' }, meta: { title: 'Mahasiswa', roles: staff } },
  { path: '/mahasiswa/:id', component: () => import('@/views/master/MahasiswaDetailView.vue'), meta: { title: 'Mahasiswa', roles: [...staff, 'kaprodi'] } },
  { path: '/dosen', component: master, props: { resource: 'dosen' }, meta: { title: 'Dosen', roles: staff } },
  { path: '/ruang', component: master, props: { resource: 'ruang' }, meta: { title: 'Ruang', roles: staff } },
  { path: '/periode', component: master, props: { resource: 'periode' }, meta: { title: 'Periode akademik', roles: staff } },
  { path: '/prodi', component: () => import('@/views/master/ProdiView.vue'), meta: { title: 'Prodi & fakultas', roles: staff } },
] satisfies RouteRecordRaw[]
