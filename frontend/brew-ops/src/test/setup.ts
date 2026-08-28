import '@testing-library/jest-dom/vitest'
// jsdom doesn't implement IndexedDB at all — this polyfills the global
// indexedDB/IDBKeyRange used by src/composables/useOfflineSalesDb.ts.
import 'fake-indexeddb/auto'
