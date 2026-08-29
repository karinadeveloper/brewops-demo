import { defineStore } from 'pinia'
import { useApi } from '../composables/useApi'

export type MarketingAssetType = 'PRODUCT' | 'PROMOTION'

export interface MarketingAsset {
  id: string
  name: string
  image_url: string
  type: MarketingAssetType
  created_at: string
  deleted_at?: string | null
  deleted_by_email?: string | null
}

interface MarketingState {
  items: MarketingAsset[]
  isLoading: boolean
  loadError: boolean
  hasLoadedOnce: boolean
  trashedItems: MarketingAsset[]
  isLoadingTrash: boolean
  trashLoadError: boolean
}

export const useMarketingStore = defineStore('marketing', {
  state: (): MarketingState => ({
    items: [],
    isLoading: false,
    loadError: false,
    hasLoadedOnce: false,
    trashedItems: [],
    isLoadingTrash: false,
    trashLoadError: false,
  }),

  actions: {
    async fetchAssets() {
      this.isLoading = true
      this.loadError = false
      try {
        const { get } = useApi()
        this.items = await get<MarketingAsset[]>('/marketing/assets')
      } catch {
        this.loadError = true
      } finally {
        this.isLoading = false
        this.hasLoadedOnce = true
      }
    },

    // Unlike products, marketing assets have no separate pre-upload step:
    // POST /marketing/assets accepts name/type/image together in a single
    // multipart request and creates the record atomically — there's no
    // draft "uploaded but not yet a marketing asset" state to manage.
    async createAsset(name: string, type: MarketingAssetType, file: File): Promise<MarketingAsset> {
      const { postForm } = useApi()
      const formData = new FormData()
      formData.append('name', name)
      formData.append('type', type)
      formData.append('image', file)
      const asset = await postForm<MarketingAsset>('/marketing/assets', formData)
      this.items.unshift(asset)
      return asset
    },

    async softDeleteAsset(id: string) {
      const { delete: del } = useApi()
      await del(`/marketing/assets/${id}`)
      this.items = this.items.filter((a) => a.id !== id)
    },

    async fetchTrash() {
      this.isLoadingTrash = true
      this.trashLoadError = false
      try {
        const { get } = useApi()
        this.trashedItems = await get<MarketingAsset[]>('/marketing/assets/trash')
      } catch {
        this.trashLoadError = true
      } finally {
        this.isLoadingTrash = false
      }
    },

    // Unlike Product.restore, there is no business invariant that can
    // reject this (no stock involved) — a straightforward reactivation.
    async restoreAsset(id: string): Promise<MarketingAsset> {
      const { post } = useApi()
      const restored = await post<MarketingAsset>(`/marketing/assets/${id}/restore`)
      this.trashedItems = this.trashedItems.filter((a) => a.id !== id)
      return restored
    },
  },
})
