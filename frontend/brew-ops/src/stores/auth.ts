import { defineStore } from 'pinia'

interface AuthUser {
  id: string
  email: string
}

interface AuthState {
  user: AuthUser | null
  accessToken: string | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    user: null,
    accessToken: null,
  }),
  getters: {
    isAuthenticated: (state) => state.accessToken !== null,
  },
  actions: {
    // Real login/logout/refresh logic lands in Session 5.
    setSession(user: AuthUser, accessToken: string) {
      this.user = user
      this.accessToken = accessToken
    },
    clearSession() {
      this.user = null
      this.accessToken = null
    },
  },
})
