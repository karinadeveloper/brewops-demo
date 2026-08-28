import { describe, expect, it } from 'vitest'
import { hasFormErrors, validateProductForm, type ProductFormValues } from './productFormValidation'

const validValues: ProductFormValues = {
  name: 'Jugo de naranja 1L',
  category: 'juice',
  salePricePesos: 45,
  costPesos: 20,
  currentStock: 50,
  minStock: 10,
}

describe('validateProductForm', () => {
  it('returns no errors for fully valid input', () => {
    expect(validateProductForm(validValues)).toEqual({})
  })

  it('rejects an empty name', () => {
    const errors = validateProductForm({ ...validValues, name: '   ' })
    expect(errors.name).toBeDefined()
  })

  it('rejects a negative sale price', () => {
    const errors = validateProductForm({ ...validValues, salePricePesos: -1 })
    expect(errors.salePricePesos).toBeDefined()
  })

  it('rejects a null sale price', () => {
    const errors = validateProductForm({ ...validValues, salePricePesos: null })
    expect(errors.salePricePesos).toBeDefined()
  })

  it('rejects a negative cost', () => {
    const errors = validateProductForm({ ...validValues, costPesos: -5 })
    expect(errors.costPesos).toBeDefined()
  })

  it('rejects a negative current stock', () => {
    const errors = validateProductForm({ ...validValues, currentStock: -1 })
    expect(errors.currentStock).toBeDefined()
  })

  it('rejects a non-integer current stock', () => {
    const errors = validateProductForm({ ...validValues, currentStock: 1.5 })
    expect(errors.currentStock).toBeDefined()
  })

  it('rejects a negative min stock', () => {
    const errors = validateProductForm({ ...validValues, minStock: -1 })
    expect(errors.minStock).toBeDefined()
  })

  it('accepts zero as a valid stock/price value', () => {
    const errors = validateProductForm({
      ...validValues,
      salePricePesos: 0,
      costPesos: 0,
      currentStock: 0,
      minStock: 0,
    })
    expect(errors).toEqual({})
  })
})

describe('hasFormErrors', () => {
  it('is false for an empty errors object', () => {
    expect(hasFormErrors({})).toBe(false)
  })

  it('is true when at least one error is present', () => {
    expect(hasFormErrors({ name: 'required' })).toBe(true)
  })
})
