import { i18n } from '../i18n'

// Demo-only (see CLAUDE.md's "DEMO MODE" section) — the app UI chrome's
// ES/EN toggle. Seed data (product names, sale history) is never affected
// by this: it always renders in Spanish regardless of the selected locale,
// since it represents a real Mexican business.
export type Locale = 'es' | 'en'

const STORAGE_KEY = 'brewops-locale'

function isLocale(value: string | null): value is Locale {
  return value === 'es' || value === 'en'
}

// Applied once, at module load — before the app ever mounts — so the first
// paint already reflects a previously chosen language instead of flashing
// Spanish first. Defaults to Spanish when nothing is stored yet or the
// stored value is unrecognized.
const stored = localStorage.getItem(STORAGE_KEY)
if (isLocale(stored)) {
  i18n.global.locale.value = stored
}

export function useLocale() {
  return {
    locale: i18n.global.locale,
    setLocale(next: Locale) {
      i18n.global.locale.value = next
      localStorage.setItem(STORAGE_KEY, next)
    },
  }
}
