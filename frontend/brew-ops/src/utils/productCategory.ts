import type { ProductCategory } from '../stores/products'

// i18n keys (see src/locales/*.json's "category" namespace) for the fixed
// category enum's display label — CLAUDE.md's language policy puts all
// user-facing text through the ES/EN demo toggle, while the category values
// themselves stay in English to match the backend's CHECK constraint and
// are never translated.
export const PRODUCT_CATEGORY_I18N_KEYS: Record<ProductCategory, string> = {
  juice: 'category.juice',
  water: 'category.water',
  soda: 'category.soda',
  other: 'category.other',
}

export const PRODUCT_CATEGORIES: ProductCategory[] = ['juice', 'water', 'soda', 'other']
