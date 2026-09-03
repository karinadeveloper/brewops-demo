import type { MarketingAssetType } from '../stores/marketing'

// i18n keys (see src/locales/*.json's "marketingType" namespace) for the
// fixed type enum's display label — the value itself stays in English to
// match the backend's CHECK constraint and is never translated.
export const MARKETING_ASSET_TYPE_I18N_KEYS: Record<MarketingAssetType, string> = {
  PRODUCT: 'marketingType.PRODUCT',
  PROMOTION: 'marketingType.PROMOTION',
}

export const MARKETING_ASSET_TYPES: MarketingAssetType[] = ['PRODUCT', 'PROMOTION']
