// Money is always cents (int) over the wire and in state, never a float.
// Conversion to/from pesos happens only here, at the UI boundary (form
// inputs and display), never inside store logic.

const PESOS_FORMATTER = new Intl.NumberFormat('es-MX', {
  style: 'currency',
  currency: 'MXN',
})

// pesosToCents rounds rather than truncates, so a form input of "10.005"
// (a value a user could type, even if unrealistic) doesn't silently lose a
// cent to floating-point representation error.
export function pesosToCents(pesos: number): number {
  return Math.round(pesos * 100)
}

export function centsToPesos(cents: number): number {
  return cents / 100
}

export function formatCentsAsPesos(cents: number): string {
  return PESOS_FORMATTER.format(centsToPesos(cents))
}
