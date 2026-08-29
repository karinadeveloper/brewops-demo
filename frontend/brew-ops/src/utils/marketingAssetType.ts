import type { MarketingAssetType } from '../stores/marketing'

// Spanish display labels for the fixed type enum — CLAUDE.md's language
// policy puts all user-facing text in Spanish, while the value itself
// stays in English to match the backend's CHECK constraint.
export const MARKETING_ASSET_TYPE_LABELS: Record<MarketingAssetType, string> = {
  PRODUCT: 'Producto',
  PROMOTION: 'Promoción',
}

export const MARKETING_ASSET_TYPES: MarketingAssetType[] = ['PRODUCT', 'PROMOTION']
