import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from '../stores/auth'
import router from './index'

describe('router guard', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('redirects to /login when there is no active session', async () => {
    // Arrange
    const authStore = useAuthStore()
    expect(authStore.isAuthenticated).toBe(false)

    // Act
    await router.push('/')

    // Assert
    expect(router.currentRoute.value.path).toBe('/login')
  })

  it('preserves the originally requested route as a redirect query param', async () => {
    // Arrange
    const authStore = useAuthStore()
    expect(authStore.isAuthenticated).toBe(false)

    // Act
    await router.push('/')

    // Assert
    expect(router.currentRoute.value.query.redirect).toBe('/')
  })

  it('allows navigation to a protected route with a valid session', async () => {
    // Arrange
    const authStore = useAuthStore()
    authStore.$patch({ accessToken: 'valid-token' })

    // Act
    await router.push('/')

    // Assert
    expect(router.currentRoute.value.path).toBe('/')
    expect(router.currentRoute.value.name).toBe('dashboard')
  })

  it('redirects an already-authenticated user away from /login to the dashboard', async () => {
    // Arrange
    const authStore = useAuthStore()
    authStore.$patch({ accessToken: 'valid-token' })

    // Act
    await router.push('/login')

    // Assert
    expect(router.currentRoute.value.path).toBe('/')
  })

  it('allows an unauthenticated user to reach /login directly', async () => {
    // Arrange
    const authStore = useAuthStore()
    expect(authStore.isAuthenticated).toBe(false)

    // Act
    await router.push('/login')

    // Assert
    expect(router.currentRoute.value.path).toBe('/login')
  })
})
