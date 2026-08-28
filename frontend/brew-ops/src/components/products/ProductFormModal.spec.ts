import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { createTestingPinia } from '@pinia/testing'
import ProductFormModal from './ProductFormModal.vue'
import { useProductsStore, type Product } from '../../stores/products'

const { postFormMock } = vi.hoisted(() => ({ postFormMock: vi.fn() }))

vi.mock('../../composables/useApi', () => ({
  useApi: () => ({ get: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn(), postForm: postFormMock }),
}))

function makeProduct(overrides: Partial<Product> = {}): Product {
  return {
    id: 'p1',
    name: 'Jugo de naranja 1L',
    category: 'juice',
    sale_price_cents: 4500,
    cost_cents: 2000,
    current_stock: 50,
    min_stock: 10,
    image_url: null,
    version: 3,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function renderModal(props: { mode: 'create' | 'edit'; product?: Product | null }) {
  const pinia = createTestingPinia({ stubActions: true })
  const view = render(ProductFormModal, { props, global: { plugins: [pinia] } })
  return { ...view, store: useProductsStore(pinia) }
}

describe('ProductFormModal', () => {
  beforeEach(() => {
    postFormMock.mockReset()
  })

  it('shows per-field validation errors and never calls createProduct when the form is empty', async () => {
    // Arrange
    const { store } = renderModal({ mode: 'create' })
    const user = userEvent.setup()

    // Act
    await user.click(screen.getByRole('button', { name: 'Crear producto' }))

    // Assert
    expect(screen.getByText('El nombre es obligatorio.')).toBeInTheDocument()
    expect(screen.getByText('El precio de venta no puede ser negativo.')).toBeInTheDocument()
    expect(screen.getByText('El costo no puede ser negativo.')).toBeInTheDocument()
    expect(screen.getByText('El stock actual debe ser un número entero, no negativo.')).toBeInTheDocument()
    expect(screen.getByText('El stock mínimo debe ser un número entero, no negativo.')).toBeInTheDocument()
    expect(store.createProduct).not.toHaveBeenCalled()
  })

  it('converts pesos input to integer cents before sending to the store', async () => {
    // Arrange
    const { store } = renderModal({ mode: 'create' })
    vi.mocked(store.createProduct).mockResolvedValueOnce(makeProduct())
    const user = userEvent.setup()

    // Act
    await user.type(screen.getByLabelText('Nombre'), 'Jugo de mango 1L')
    await user.type(screen.getByLabelText('Precio de venta (MXN)'), '48.50')
    await user.type(screen.getByLabelText('Costo (MXN)'), '22')
    await user.type(screen.getByLabelText('Stock actual'), '30')
    await user.type(screen.getByLabelText('Stock mínimo'), '5')
    await user.click(screen.getByRole('button', { name: 'Crear producto' }))

    // Assert
    await waitFor(() => expect(store.createProduct).toHaveBeenCalled())
    expect(store.createProduct).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'Jugo de mango 1L',
        sale_price_cents: 4850,
        cost_cents: 2200,
        current_stock: 30,
        min_stock: 5,
      }),
    )
  })

  it('sends the product version to updateProduct in edit mode', async () => {
    // Arrange
    const product = makeProduct()
    const { store } = renderModal({ mode: 'edit', product })
    vi.mocked(store.updateProduct).mockResolvedValueOnce(product)
    const user = userEvent.setup()

    // Act
    await user.click(screen.getByRole('button', { name: 'Guardar cambios' }))

    // Assert
    await waitFor(() => expect(store.updateProduct).toHaveBeenCalled())
    expect(store.updateProduct).toHaveBeenCalledWith(product.id, product.version, expect.any(Object))
  })

  it('shows a clear conflict message on a 409 and removes the Save action, without leaving the user retrying', async () => {
    // Arrange
    const product = makeProduct()
    const { store } = renderModal({ mode: 'edit', product })
    vi.mocked(store.updateProduct).mockRejectedValueOnce({ status: 409, message: 'stale version' })
    const user = userEvent.setup()

    // Act
    await user.click(screen.getByRole('button', { name: 'Guardar cambios' }))

    // Assert
    await waitFor(() => {
      expect(
        screen.getByText('Este producto fue modificado por otra sesión, recargá para ver los cambios más recientes.'),
      ).toBeInTheDocument()
    })
    expect(screen.queryByRole('button', { name: 'Guardar cambios' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Cerrar' })).toBeInTheDocument()
    expect(store.fetchProducts).toHaveBeenCalled()
  })

  it('shows an image preview after selecting a file and uploads it', async () => {
    // Arrange
    postFormMock.mockResolvedValueOnce({ url: 'https://example.com/uploaded.jpg' })
    renderModal({ mode: 'create' })
    const file = new File(['fake-image-bytes'], 'photo.png', { type: 'image/png' })
    const user = userEvent.setup()

    // Act
    await user.upload(screen.getByLabelText('Imagen'), file)

    // Assert
    await waitFor(() => expect(postFormMock).toHaveBeenCalled())
    expect(screen.getByAltText('Vista previa')).toBeInTheDocument()
  })

  it('shows an upload error and lets the user retry without losing other form data', async () => {
    // Arrange
    postFormMock.mockRejectedValueOnce({ status: 400, message: 'invalid content type' })
    renderModal({ mode: 'create' })
    const file = new File(['fake-image-bytes'], 'photo.png', { type: 'image/png' })
    const user = userEvent.setup()
    await user.type(screen.getByLabelText('Nombre'), 'Jugo de mango 1L')

    // Act
    await user.upload(screen.getByLabelText('Imagen'), file)

    // Assert
    await waitFor(() => {
      expect(
        screen.getByText(
          'No se pudo subir la imagen. Verificá que sea un archivo JPEG, PNG o WebP e intentá de nuevo.',
        ),
      ).toBeInTheDocument()
    })
    expect((screen.getByLabelText('Nombre') as HTMLInputElement).value).toBe('Jugo de mango 1L')

    // Act — retry
    postFormMock.mockResolvedValueOnce({ url: 'https://example.com/uploaded.jpg' })
    await user.click(screen.getByRole('button', { name: 'Reintentar subida' }))

    // Assert
    await waitFor(() => expect(postFormMock).toHaveBeenCalledTimes(2))
  })

  it('disables the save button while the image upload is in flight', async () => {
    // Arrange
    let resolveUpload!: (value: { url: string }) => void
    postFormMock.mockReturnValueOnce(
      new Promise((resolve) => {
        resolveUpload = resolve
      }),
    )
    renderModal({ mode: 'create' })
    const file = new File(['fake-image-bytes'], 'photo.png', { type: 'image/png' })
    const user = userEvent.setup()

    // Act
    await user.upload(screen.getByLabelText('Imagen'), file)

    // Assert
    await waitFor(() => expect(screen.getByText('Subiendo imagen…')).toBeInTheDocument())
    expect(screen.getByRole('button', { name: 'Crear producto' })).toBeDisabled()

    resolveUpload({ url: 'https://example.com/uploaded.jpg' })
    await waitFor(() => expect(screen.getByRole('button', { name: 'Crear producto' })).not.toBeDisabled())
  })
})
