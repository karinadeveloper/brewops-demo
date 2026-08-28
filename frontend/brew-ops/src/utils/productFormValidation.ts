import type { ProductCategory } from '../stores/products'

export interface ProductFormValues {
  name: string
  category: ProductCategory
  salePricePesos: number | null
  costPesos: number | null
  currentStock: number | null
  minStock: number | null
}

export interface ProductFormErrors {
  name?: string
  salePricePesos?: string
  costPesos?: string
  currentStock?: string
  minStock?: string
}

// Pure and exhaustively tested — client-side validation before ever
// touching the network, per-field messages rather than a generic alert.
export function validateProductForm(values: ProductFormValues): ProductFormErrors {
  const errors: ProductFormErrors = {}

  if (!values.name.trim()) {
    errors.name = 'El nombre es obligatorio.'
  }
  if (values.salePricePesos === null || Number.isNaN(values.salePricePesos) || values.salePricePesos < 0) {
    errors.salePricePesos = 'El precio de venta no puede ser negativo.'
  }
  if (values.costPesos === null || Number.isNaN(values.costPesos) || values.costPesos < 0) {
    errors.costPesos = 'El costo no puede ser negativo.'
  }
  if (
    values.currentStock === null ||
    !Number.isInteger(values.currentStock) ||
    values.currentStock < 0
  ) {
    errors.currentStock = 'El stock actual debe ser un número entero, no negativo.'
  }
  if (values.minStock === null || !Number.isInteger(values.minStock) || values.minStock < 0) {
    errors.minStock = 'El stock mínimo debe ser un número entero, no negativo.'
  }

  return errors
}

export function hasFormErrors(errors: ProductFormErrors): boolean {
  return Object.keys(errors).length > 0
}
