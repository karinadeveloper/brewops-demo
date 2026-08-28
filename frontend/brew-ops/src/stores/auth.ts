import { defineStore } from 'pinia'
import router from '../router'
import { useApi } from '../composables/useApi'

const ACCESS_TOKEN_STORAGE_KEY = 'brewops_access_token'
const REFRESH_TOKEN_STORAGE_KEY = 'brewops_refresh_token'
const USER_STORAGE_KEY = 'brewops_user'

export interface AuthUser {
  email: string
  // BrewOps has no multi-tenant roles — every account is ADMIN (see
  // CLAUDE.md's User domain model). POST /auth/login doesn't return a user
  // profile, so this is set locally from the login form rather than
  // fetched from the API.
  role: 'ADMIN'
}

interface AuthState {
  accessToken: string | null
  refreshToken: string | null
  user: AuthUser | null
}

interface LoginResponse {
  access_token: string
  refresh_token: string
}

function readStoredUser(): AuthUser | null {
  const raw = localStorage.getItem(USER_STORAGE_KEY)
  if (!raw) {
    return null
  }
  try {
    return JSON.parse(raw) as AuthUser
  } catch {
    return null
  }
}

export const useAuthStore = defineStore('auth', {
  // Reading from localStorage here (rather than in a separate on-mount
  // composable) means the store already has the session restored the
  // moment it's first used, so a page reload never bounces the user to
  // /login while data is still loading.
  state: (): AuthState => ({
    accessToken: localStorage.getItem(ACCESS_TOKEN_STORAGE_KEY),
    refreshToken: localStorage.getItem(REFRESH_TOKEN_STORAGE_KEY),
    user: readStoredUser(),
  }),

  getters: {
    isAuthenticated: (state) => state.accessToken !== null,
  },

  actions: {
    async login(email: string, password: string) {
      const { post } = useApi()
      const response = await post<LoginResponse>('/auth/login', { email, password })
      this.setTokens(response.access_token, response.refresh_token)
      this.user = { email, role: 'ADMIN' }
      localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(this.user))
    },

    // Used both by login (both tokens are new) and by useApi's refresh
    // interceptor. POST /auth/refresh only ever returns a new access
    // token — the refresh token itself never rotates — so the interceptor
    // always passes the existing refreshToken back in unchanged.
    setTokens(accessToken: string, refreshToken: string) {
      this.accessToken = accessToken
      this.refreshToken = refreshToken
      localStorage.setItem(ACCESS_TOKEN_STORAGE_KEY, accessToken)
      localStorage.setItem(REFRESH_TOKEN_STORAGE_KEY, refreshToken)
    },

    // reason is set to "expired" when useApi's interceptor forces a logout
    // after a failed refresh, so LoginView can show why the user landed
    // back here. A manual "cerrar sesión" click passes no reason.
    logout(reason?: string) {
      this.accessToken = null
      this.refreshToken = null
      this.user = null
      localStorage.removeItem(ACCESS_TOKEN_STORAGE_KEY)
      localStorage.removeItem(REFRESH_TOKEN_STORAGE_KEY)
      localStorage.removeItem(USER_STORAGE_KEY)
      router.push({ path: '/login', query: reason ? { reason } : undefined })
    },
  },
})
