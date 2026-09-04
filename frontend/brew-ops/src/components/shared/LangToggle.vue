<script setup lang="ts">
// Demo-only — plain text toggle, deliberately never a flag icon: a flag
// represents a country, not a language, and both Spanish and English are
// spoken across many countries.
import { useI18n } from 'vue-i18n'
import { useLocale, type Locale } from '../../composables/useLocale'

const { t } = useI18n()
const { locale, setLocale } = useLocale()

function select(next: Locale) {
  setLocale(next)
}
</script>

<template>
  <div
    class="lang-toggle"
    role="group"
    :aria-label="t('lang.ariaLabel')"
  >
    <button
      type="button"
      class="lang-option"
      :class="{ 'lang-option--active': locale === 'es' }"
      :aria-pressed="locale === 'es'"
      @click="select('es')"
    >
      {{ t('lang.es') }}
    </button>
    <span
      class="lang-separator"
      aria-hidden="true"
    >/</span>
    <button
      type="button"
      class="lang-option"
      :class="{ 'lang-option--active': locale === 'en' }"
      :aria-pressed="locale === 'en'"
      @click="select('en')"
    >
      {{ t('lang.en') }}
    </button>
  </div>
</template>

<style scoped>
.lang-toggle {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  font-size: 0.8rem;
}

.lang-option {
  border: none;
  background: none;
  padding: var(--space-1);
  color: var(--color-text-muted);
  cursor: pointer;
}

.lang-option--active {
  color: var(--color-text);
  font-weight: 700;
  text-decoration: underline;
}

.lang-separator {
  color: var(--color-text-muted);
}
</style>
