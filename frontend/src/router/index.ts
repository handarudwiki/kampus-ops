import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import type { Role } from '@/api/types'
import { useAuthStore } from '@/stores/auth'
import mahasiswa from './routes/mahasiswa'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    /** roles allowed for the *active* role; omitted = any signed-in role */
    roles?: Role[]
    public?: boolean
    bare?: boolean
  }
}

const v = (name: string) => () => import(`../views/${name}.vue`)

const routes: RouteRecordRaw[] = [
  { path: '/login', component: v('auth/LoginView'), meta: { public: true, bare: true, title: 'Masuk' } },
  { path: '/', component: v('DashboardView'), meta: { title: 'Beranda' } },
  ...mahasiswa,
  { path: '/design-system', component: v('DesignSystemView'), meta: { title: 'Design system', public: true } },
  { path: '/:pathMatch(.*)*', component: v('NotFoundView'), meta: { title: 'Tidak ditemukan', public: true } },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  await auth.restore()
  if (to.meta.public) return true
  if (!auth.isAuthed) return { path: '/login', query: { next: to.fullPath } }
  if (to.meta.roles && auth.activeRole && !to.meta.roles.includes(auth.activeRole)) return { path: '/' }
  return true
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · KampusOps` : 'KampusOps'
})

/** Lets feature modules add their routes without editing this file's array. */
export function addRoutes(list: RouteRecordRaw[]) {
  for (const r of list) router.addRoute(r)
}
