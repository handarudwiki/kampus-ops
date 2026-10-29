import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api, session } from '@/api/client'
import type { AuthResponse, Role, Tenant, User } from '@/api/types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const tenant = ref<Tenant | null>(null)
  const activeRole = ref<Role | null>((session.role as Role | null) ?? null)
  const loaded = ref(false)

  const roles = computed(() => user.value?.roles.map((r) => r.role) ?? [])
  const isAuthed = computed(() => !!user.value)

  function pickRole() {
    if (!activeRole.value || !roles.value.includes(activeRole.value)) {
      activeRole.value = roles.value[0] ?? null
    }
    if (activeRole.value) session.setRole(activeRole.value)
  }

  async function login(tenant_code: string, identifier: string, password: string) {
    const res = await api.post<AuthResponse>('/auth/login', { tenant_code, identifier, password })
    session.set(res)
    user.value = res.user
    tenant.value = res.tenant
    activeRole.value = null
    pickRole()
    loaded.value = true
  }

  /** Restore the session on page load. Silent if there is no token. */
  async function restore() {
    if (loaded.value) return
    loaded.value = true
    if (!session.access && !session.refresh) return
    try {
      const me = await api.get<{ user: User; tenant: Tenant }>('/me')
      user.value = me.user
      tenant.value = me.tenant
      pickRole()
    } catch {
      session.clear()
    }
  }

  function switchRole(role: Role) {
    activeRole.value = role
    session.setRole(role)
  }

  async function logout() {
    const token = session.refresh
    try { if (token) await api.post('/auth/logout', { refresh_token: token }) } catch { /* ignore */ }
    session.clear()
    user.value = null
    tenant.value = null
    activeRole.value = null
  }

  /** Called by the API client when refresh fails. */
  function expire() {
    user.value = null
    tenant.value = null
  }

  return { user, tenant, activeRole, roles, isAuthed, loaded, login, restore, switchRole, logout, expire }
})
