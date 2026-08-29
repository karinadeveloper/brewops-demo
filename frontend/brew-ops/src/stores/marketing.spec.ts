import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { getMock, postMock, postFormMock, deleteMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
  postFormMock: vi.fn(),
  deleteMock: vi.fn(),
}))

vi.mock('../composables/useApi', () => ({
  useApi: () => ({ get: getMock, post: postMock, patch: vi.fn(), delete: deleteMock, postForm: postFormMock }),
}))

const { useMarketingStore } = await import('./marketing')

function makeAsset(overrides: Partial<import('./marketing').MarketingAsset> = {}) {
  return {
    id: 'a1',
    name: 'Jugo de mango',
    image_url: 'https://cdn.example.com/a1.jpg',
    type: 'PRODUCT' as const,
    created_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('useMarketingStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getMock.mockReset()
    postMock.mockReset()
    postFormMock.mockReset()
    deleteMock.mockReset()
  })

  it('fetchAssets loads the active list', async () => {
    getMock.mockResolvedValueOnce([makeAsset()])
    const store = useMarketingStore()

    await store.fetchAssets()

    expect(store.items).toHaveLength(1)
    expect(store.loadError).toBe(false)
    expect(store.hasLoadedOnce).toBe(true)
  })

  it('fetchAssets sets loadError on failure', async () => {
    getMock.mockRejectedValueOnce(new Error('boom'))
    const store = useMarketingStore()

    await store.fetchAssets()

    expect(store.loadError).toBe(true)
  })

  it('createAsset posts a single multipart request with name, type, and image, and prepends the result', async () => {
    const created = makeAsset({ id: 'new-1' })
    postFormMock.mockResolvedValueOnce(created)
    const store = useMarketingStore()
    const file = new File(['data'], 'promo.jpg', { type: 'image/jpeg' })

    const result = await store.createAsset('Promo de verano', 'PROMOTION', file)

    expect(postFormMock).toHaveBeenCalledWith('/marketing/assets', expect.any(FormData))
    const formData = postFormMock.mock.calls[0][1] as FormData
    expect(formData.get('name')).toBe('Promo de verano')
    expect(formData.get('type')).toBe('PROMOTION')
    expect(formData.get('image')).toBe(file)
    expect(result).toEqual(created)
    expect(store.items[0]).toEqual(created)
  })

  it('softDeleteAsset removes the asset from the active list', async () => {
    const store = useMarketingStore()
    store.items = [makeAsset({ id: 'a1' }), makeAsset({ id: 'a2' })]
    deleteMock.mockResolvedValueOnce(undefined)

    await store.softDeleteAsset('a1')

    expect(store.items.map((a) => a.id)).toEqual(['a2'])
    expect(deleteMock).toHaveBeenCalledWith('/marketing/assets/a1')
  })

  it('fetchTrash loads soft-deleted assets', async () => {
    getMock.mockResolvedValueOnce([makeAsset({ id: 'trashed-1' })])
    const store = useMarketingStore()

    await store.fetchTrash()

    expect(store.trashedItems).toHaveLength(1)
    expect(store.trashLoadError).toBe(false)
  })

  it('restoreAsset removes the asset from the trash list', async () => {
    const store = useMarketingStore()
    store.trashedItems = [makeAsset({ id: 'a1' })]
    postMock.mockResolvedValueOnce(makeAsset({ id: 'a1', deleted_at: null }))

    const restored = await store.restoreAsset('a1')

    expect(postMock).toHaveBeenCalledWith('/marketing/assets/a1/restore')
    expect(store.trashedItems).toEqual([])
    expect(restored.id).toBe('a1')
  })
})
