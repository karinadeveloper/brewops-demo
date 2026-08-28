import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { postMock, pushMock } = vi.hoisted(() => ({
  postMock: vi.fn(),
  pushMock: vi.fn(),
}))

vi.mock('../router', () => ({
  default: { push: pushMock },
}))

vi.mock('../composables/useApi', () => ({
  useApi: () => ({ post: postMock, get: vi.fn(), patch: vi.fn(), delete: vi.fn() }),
}))

// Imported after the mocks above so the store picks up the mocked
// dependencies rather than the real router/useApi.
const { useAuthStore } = await import('./auth')

const ACCESS_TOKEN_KEY = 'brewops_access_token'
const REFRESH_TOKEN_KEY = 'brewops_refresh_token'
const USER_KEY = 'brewops_user'

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    postMock.mockReset()
    pushMock.mockReset()
  })

  it('login success stores both tokens and the user in state and localStorage', async () => {
    // Arrange
    postMock.mockResolvedValueOnce({ access_token: 'access-1', refresh_token: 'refresh-1' })
    const store = useAuthStore()

    // Act
    await store.login('owner@brewops.mx', 'correct-password')

    // Assert
    expect(store.accessToken).toBe('access-1')
    expect(store.refreshToken).toBe('refresh-1')
    expect(store.user).toEqual({ email: 'owner@brewops.mx', role: 'ADMIN' })
    expect(store.isAuthenticated).toBe(true)
    expect(localStorage.getItem(ACCESS_TOKEN_KEY)).toBe('access-1')
    expect(localStorage.getItem(REFRESH_TOKEN_KEY)).toBe('refresh-1')
    expect(JSON.parse(localStorage.getItem(USER_KEY) ?? '{}')).toEqual({
      email: 'owner@brewops.mx',
      role: 'ADMIN',
    })
  })

  it('login failure stores nothing in state or localStorage', async () => {
    // Arrange
    postMock.mockRejectedValueOnce({ status: 401, message: 'invalid credentials' })
    const store = useAuthStore()

    // Act
    await expect(store.login('owner@brewops.mx', 'wrong-password')).rejects.toBeTruthy()

    // Assert
    expect(store.accessToken).toBeNull()
    expect(store.refreshToken).toBeNull()
    expect(store.user).toBeNull()
    expect(store.isAuthenticated).toBe(false)
    expect(localStorage.getItem(ACCESS_TOKEN_KEY)).toBeNull()
    expect(localStorage.getItem(REFRESH_TOKEN_KEY)).toBeNull()
    expect(localStorage.getItem(USER_KEY)).toBeNull()
  })

  it('logout clears state and localStorage, and redirects to /login', () => {
    // Arrange
    const store = useAuthStore()
    store.$patch({
      accessToken: 'access-1',
      refreshToken: 'refresh-1',
      user: { email: 'owner@brewops.mx', role: 'ADMIN' },
    })
    localStorage.setItem(ACCESS_TOKEN_KEY, 'access-1')
    localStorage.setItem(REFRESH_TOKEN_KEY, 'refresh-1')
    localStorage.setItem(USER_KEY, JSON.stringify({ email: 'owner@brewops.mx', role: 'ADMIN' }))

    // Act
    store.logout()

    // Assert
    expect(store.accessToken).toBeNull()
    expect(store.refreshToken).toBeNull()
    expect(store.user).toBeNull()
    expect(store.isAuthenticated).toBe(false)
    expect(localStorage.getItem(ACCESS_TOKEN_KEY)).toBeNull()
    expect(localStorage.getItem(REFRESH_TOKEN_KEY)).toBeNull()
    expect(localStorage.getItem(USER_KEY)).toBeNull()
    expect(pushMock).toHaveBeenCalledWith({ path: '/login', query: undefined })
  })

  it('logout with a reason redirects with that reason as a query param', () => {
    // Arrange
    const store = useAuthStore()

    // Act
    store.logout('expired')

    // Assert
    expect(pushMock).toHaveBeenCalledWith({ path: '/login', query: { reason: 'expired' } })
  })

  it('setTokens updates state and localStorage', () => {
    // Arrange
    const store = useAuthStore()

    // Act
    store.setTokens('new-access', 'new-refresh')

    // Assert
    expect(store.accessToken).toBe('new-access')
    expect(store.refreshToken).toBe('new-refresh')
    expect(localStorage.getItem(ACCESS_TOKEN_KEY)).toBe('new-access')
    expect(localStorage.getItem(REFRESH_TOKEN_KEY)).toBe('new-refresh')
  })

  it('restores a persisted session from localStorage on initialization', () => {
    // Arrange
    localStorage.setItem(ACCESS_TOKEN_KEY, 'persisted-access')
    localStorage.setItem(REFRESH_TOKEN_KEY, 'persisted-refresh')
    localStorage.setItem(USER_KEY, JSON.stringify({ email: 'owner@brewops.mx', role: 'ADMIN' }))

    // Act — creating the store simulates what happens on a page reload.
    const store = useAuthStore()

    // Assert
    expect(store.accessToken).toBe('persisted-access')
    expect(store.refreshToken).toBe('persisted-refresh')
    expect(store.user).toEqual({ email: 'owner@brewops.mx', role: 'ADMIN' })
    expect(store.isAuthenticated).toBe(true)
  })
})
