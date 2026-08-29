import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { createMemoryHistory, createRouter } from 'vue-router'
import { createTestingPinia } from '@pinia/testing'
import MarketingTrashView from './MarketingTrashView.vue'
import type { MarketingAsset } from '../stores/marketing'

const { getMock, postMock } = vi.hoisted(() => ({ getMock: vi.fn(), postMock: vi.fn() }))

vi.mock('../composables/useApi', () => ({
  useApi: () => ({ get: getMock, post: postMock, patch: vi.fn(), delete: vi.fn(), postForm: vi.fn() }),
}))

function makeTrashedAsset(overrides: Partial<MarketingAsset> = {}): MarketingAsset {
  return {
    id: 'a1',
    name: 'Jugo de mango',
    image_url: 'https://cdn.example.com/a1.jpg',
    type: 'PRODUCT',
    created_at: '2026-01-01T00:00:00Z',
    deleted_at: '2026-01-05T12:00:00Z',
    deleted_by_email: 'owner@brewops.mx',
    ...overrides,
  }
}

async function renderTrashView() {
  const pinia = createTestingPinia({ stubActions: false })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/marketing/trash', name: 'marketing-trash', component: MarketingTrashView },
      { path: '/marketing', name: 'marketing', component: { template: '<div>marketing</div>' } },
    ],
  })
  await router.push('/marketing/trash')
  await router.isReady()
  return render(MarketingTrashView, { global: { plugins: [pinia, router] } })
}

describe('MarketingTrashView', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
  })

  it('lists soft-deleted assets with who and when deleted them', async () => {
    getMock.mockResolvedValueOnce([makeTrashedAsset()])

    await renderTrashView()

    await waitFor(() => expect(screen.getByText('Jugo de mango')).toBeInTheDocument())
    expect(screen.getByText(/owner@brewops\.mx/)).toBeInTheDocument()
  })

  it('shows an empty state when the trash is empty', async () => {
    getMock.mockResolvedValueOnce([])

    await renderTrashView()

    await waitFor(() => expect(screen.getByText('La papelera está vacía')).toBeInTheDocument())
  })

  // Unlike ProductsTrashView, restore here has no business invariant that
  // can reject it (no stock involved) — this is the simpler happy-path-only
  // flow the session's spec calls for.
  it('restoring successfully removes the asset from the trash list', async () => {
    getMock.mockResolvedValueOnce([makeTrashedAsset()])
    postMock.mockResolvedValueOnce(makeTrashedAsset({ deleted_at: null, deleted_by_email: null }))
    await renderTrashView()
    await waitFor(() => expect(screen.getByText('Jugo de mango')).toBeInTheDocument())
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'Restaurar' }))

    await waitFor(() => expect(screen.queryByText('Jugo de mango')).not.toBeInTheDocument())
    expect(postMock).toHaveBeenCalledWith('/marketing/assets/a1/restore')
  })

  it('shows a retry-safe error message when restore fails for transport reasons', async () => {
    getMock.mockResolvedValueOnce([makeTrashedAsset()])
    postMock.mockRejectedValueOnce({ status: 500, message: 'server error' })
    await renderTrashView()
    await waitFor(() => expect(screen.getByText('Jugo de mango')).toBeInTheDocument())
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'Restaurar' }))

    await waitFor(() =>
      expect(screen.getByText('No se pudo restaurar la imagen. Intentá de nuevo.')).toBeInTheDocument(),
    )
    expect(screen.getByText('Jugo de mango')).toBeInTheDocument()
  })
})
