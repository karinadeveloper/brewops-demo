import type { ProductCategory } from '../stores/products'

// Spanish display labels for the fixed category enum — CLAUDE.md's
// language policy puts all user-facing text in Spanish, while the category
// values themselves stay in English to match the backend's CHECK constraint.
export const PRODUCT_CATEGORY_LABELS: Record<ProductCategory, string> = {
  juice: 'Jugo',
  water: 'Agua',
  soda: 'Refresco',
  other: 'Otro',
}

export const PRODUCT_CATEGORIES: ProductCategory[] = ['juice', 'water', 'soda', 'other']
