import '@testing-library/jest-dom/vitest'
// jsdom doesn't implement IndexedDB at all — this polyfills the global
// indexedDB/IDBKeyRange used by src/composables/useOfflineSalesDb.ts.
import 'fake-indexeddb/auto'
import { config } from '@vue/test-utils'
import { i18n } from '../i18n'

// Registered once, here, so every render() call across every spec file gets
// $t/useI18n for free — @vue/test-utils concatenates this base config's
// plugins with whatever a given test's own render() call passes via
// global.plugins (e.g. pinia, router), it never replaces them.
config.global.plugins.push(i18n)
