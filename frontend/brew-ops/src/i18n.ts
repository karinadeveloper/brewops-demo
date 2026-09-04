import { createI18n } from 'vue-i18n'
import es from './locales/es.json'
import en from './locales/en.json'

// Demo-only feature — the real product's UI is Spanish-only, so this whole
// module only exists in this repo. Default locale is always Spanish on
// first visit; useLocale.ts is the only place that changes it afterward,
// persisting the choice.
export const i18n = createI18n({
  legacy: false,
  locale: 'es',
  fallbackLocale: 'es',
  messages: { es, en },
})
