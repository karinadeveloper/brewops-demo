import { describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { createTestingPinia } from '@pinia/testing'
import { createMemoryHistory, createRouter } from 'vue-router'
import LoginView from './LoginView.vue'
import { useAuthStore } from '../stores/auth'

function buildTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/login', name: 'login', component: LoginView },
      { path: '/', name: 'dashboard', component: { template: '<div>dashboard</div>' } },
      { path: '/products', name: 'products', component: { template: '<div>products</div>' } },
    ],
  })
}

async function renderLoginView(initialPath = '/login') {
  const pinia = createTestingPinia({ stubActions: true })
  const router = buildTestRouter()
  await router.push(initialPath)
  await router.isReady()

  const view = render(LoginView, { global: { plugins: [pinia, router] } })
  return { ...view, router, authStore: useAuthStore(pinia) }
}

describe('LoginView', () => {
  it('renders the login form', async () => {
    // Arrange & Act
    await renderLoginView()

    // Assert
    expect(screen.getByLabelText('Correo electrónico')).toBeInTheDocument()
    expect(screen.getByLabelText('Contraseña')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Iniciar sesión' })).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('shows a generic error message when login fails', async () => {
    // Arrange
    const { authStore } = await renderLoginView()
    vi.mocked(authStore.login).mockRejectedValueOnce({ status: 401, message: 'invalid credentials' })
    const user = userEvent.setup()

    // Act
    await user.type(screen.getByLabelText('Correo electrónico'), 'owner@brewops.mx')
    await user.type(screen.getByLabelText('Contraseña'), 'wrong-password')
    await user.click(screen.getByRole('button', { name: 'Iniciar sesión' }))

    // Assert — generic message, never reveals which field was wrong.
    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent('Email o contraseña incorrectos.')
    })
  })

  it('shows the session-expired banner when reason=expired is in the URL', async () => {
    // Arrange & Act
    await renderLoginView('/login?reason=expired')

    // Assert
    expect(screen.getByRole('alert')).toHaveTextContent('Tu sesión expiró, ingresa de nuevo.')
  })

  it('disables the submit button while the login request is in flight', async () => {
    // Arrange
    const { authStore } = await renderLoginView()
    let resolveLogin!: () => void
    vi.mocked(authStore.login).mockReturnValueOnce(
      new Promise((resolve) => {
        resolveLogin = () => resolve(undefined)
      }),
    )
    const user = userEvent.setup()
    await user.type(screen.getByLabelText('Correo electrónico'), 'owner@brewops.mx')
    await user.type(screen.getByLabelText('Contraseña'), 'correct-password')

    // Act
    await user.click(screen.getByRole('button', { name: 'Iniciar sesión' }))

    // Assert — per-action loading state on the button, not a global overlay.
    const button = screen.getByRole('button', { name: /Ingresando/ })
    expect(button).toBeDisabled()

    resolveLogin()
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Iniciar sesión' })).not.toBeDisabled()
    })
  })
})
