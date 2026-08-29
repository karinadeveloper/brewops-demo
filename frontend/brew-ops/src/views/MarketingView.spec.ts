import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { createMemoryHistory, createRouter } from 'vue-router'
import { createTestingPinia } from '@pinia/testing'
import MarketingView from './MarketingView.vue'
import type { MarketingAsset } from '../stores/marketing'

const { getMock, postFormMock, deleteMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postFormMock: vi.fn(),
  deleteMock: vi.fn(),
}))

vi.mock('../composables/useApi', () => ({
  useApi: () => ({ get: getMock, post: vi.fn(), patch: vi.fn(), delete: deleteMock, postForm: postFormMock }),
}))

function makeAsset(overrides: Partial<MarketingAsset> = {}): MarketingAsset {
  return {
    id: 'a1',
    name: 'Jugo de mango',
    image_url: 'https://cdn.example.com/a1.jpg',
    type: 'PRODUCT',
    created_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

async function renderMarketingView() {
  const pinia = createTestingPinia({ stubActions: false })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/marketing', name: 'marketing', component: MarketingView },
      { path: '/marketing/trash', name: 'marketing-trash', component: { template: '<div>trash</div>' } },
    ],
  })
  await router.push('/marketing')
  await router.isReady()
  return render(MarketingView, { global: { plugins: [pinia, router] } })
}

describe('MarketingView', () => {
  beforeEach(() => {
    getMock.mockReset()
    postFormMock.mockReset()
    deleteMock.mockReset()
  })

  it('renders each asset with its type badge and creation date', async () => {
    getMock.mockResolvedValueOnce([makeAsset({ type: 'PROMOTION' })])

    await renderMarketingView()

    await waitFor(() => expect(screen.getByText('Jugo de mango')).toBeInTheDocument())
    expect(screen.getByText('Promoción')).toBeInTheDocument()
  })

  it('shows the empty state with an upload call-to-action when there are no images', async () => {
    getMock.mockResolvedValueOnce([])

    await renderMarketingView()

    await waitFor(() => expect(screen.getByText('Todavía no tenés imágenes')).toBeInTheDocument())
    expect(screen.getByRole('button', { name: 'Agregar tu primera imagen' })).toBeInTheDocument()
  })

  it('shows the load-error empty state with a retry action', async () => {
    getMock.mockRejectedValueOnce({ status: 500, message: 'server error' })

    await renderMarketingView()

    await waitFor(() => expect(screen.getByText('No se pudieron cargar las imágenes')).toBeInTheDocument())
    expect(screen.getByRole('button', { name: 'Reintentar' })).toBeInTheDocument()
  })

  it('uploading a new image posts name/type/file together and adds it to the grid', async () => {
    getMock.mockResolvedValueOnce([])
    postFormMock.mockResolvedValueOnce(makeAsset({ id: 'new-1', name: 'Promo de verano', type: 'PROMOTION' }))
    await renderMarketingView()
    await waitFor(() => expect(screen.getByText('Todavía no tenés imágenes')).toBeInTheDocument())
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'Agregar tu primera imagen' }))
    await user.type(screen.getByLabelText('Nombre'), 'Promo de verano')
    await user.selectOptions(screen.getByLabelText('Tipo'), 'PROMOTION')
    const file = new File(['data'], 'promo.jpg', { type: 'image/jpeg' })
    await user.upload(screen.getByLabelText('Imagen'), file)

    await user.click(screen.getByRole('button', { name: 'Subir imagen' }))

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(postFormMock).toHaveBeenCalledTimes(1)
    expect(screen.getByText('Promo de verano')).toBeInTheDocument()
  })

  it('shows a validation error and never submits when no file is selected', async () => {
    getMock.mockResolvedValueOnce([])
    await renderMarketingView()
    await waitFor(() => expect(screen.getByText('Todavía no tenés imágenes')).toBeInTheDocument())
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'Agregar tu primera imagen' }))
    await user.type(screen.getByLabelText('Nombre'), 'Sin imagen')
    await user.click(screen.getByRole('button', { name: 'Subir imagen' }))

    expect(screen.getByText('Seleccioná una imagen.')).toBeInTheDocument()
    expect(postFormMock).not.toHaveBeenCalled()
  })

  it('a Content-Type rejection from the backend shows a clear Spanish message', async () => {
    // The backend sniffs real file bytes (http.DetectContentType), so this
    // is the realistic failure mode: a file whose name/browser-reported
    // type look like an image (passing the file input's own accept filter,
    // and userEvent.upload's client-side enforcement of it) but whose
    // actual content the server determines isn't really one.
    getMock.mockResolvedValueOnce([])
    postFormMock.mockRejectedValueOnce({ status: 400, message: 'file must be a JPEG, PNG, or WebP image' })
    await renderMarketingView()
    await waitFor(() => expect(screen.getByText('Todavía no tenés imágenes')).toBeInTheDocument())
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: 'Agregar tu primera imagen' }))
    await user.type(screen.getByLabelText('Nombre'), 'Archivo raro')
    const file = new File(['not-really-an-image'], 'fake.jpg', { type: 'image/jpeg' })
    await user.upload(screen.getByLabelText('Imagen'), file)
    await user.click(screen.getByRole('button', { name: 'Subir imagen' }))

    await waitFor(() =>
      expect(screen.getByText('La imagen debe ser un archivo JPEG, PNG o WebP.')).toBeInTheDocument(),
    )
  })

  describe('deleting an asset', () => {
    let confirmSpy: ReturnType<typeof vi.spyOn>

    afterEach(() => {
      confirmSpy.mockRestore()
    })

    it('removes the asset after confirming', async () => {
      confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
      getMock.mockResolvedValueOnce([makeAsset()])
      deleteMock.mockResolvedValueOnce(undefined)
      await renderMarketingView()
      await waitFor(() => expect(screen.getByText('Jugo de mango')).toBeInTheDocument())
      const user = userEvent.setup()

      await user.click(screen.getByRole('button', { name: 'Eliminar' }))

      await waitFor(() => expect(screen.queryByText('Jugo de mango')).not.toBeInTheDocument())
      expect(deleteMock).toHaveBeenCalledWith('/marketing/assets/a1')
    })

    it('does nothing when the confirmation is dismissed', async () => {
      confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)
      getMock.mockResolvedValueOnce([makeAsset()])
      await renderMarketingView()
      await waitFor(() => expect(screen.getByText('Jugo de mango')).toBeInTheDocument())
      const user = userEvent.setup()

      await user.click(screen.getByRole('button', { name: 'Eliminar' }))

      expect(deleteMock).not.toHaveBeenCalled()
      expect(screen.getByText('Jugo de mango')).toBeInTheDocument()
    })
  })
})
