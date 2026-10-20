import { defineConfig } from '@lingui/conf'
import { formatter } from '@lingui/format-po'
import { createSwcExtractor } from '@lingui/native-tools'
import { availableLocales } from './src/i18n/locales.ts'

export default defineConfig({
  sourceLocale: 'en',
  orderBy: 'messageId',
  locales: [...availableLocales],
  extractors: [createSwcExtractor()],
  // References name the file without its line, so moving code does not rewrite the catalog.
  format: formatter({ lineNumbers: false }),
  catalogs: [{ path: '<rootDir>/src/locales/{locale}/messages', include: ['src'] }],
})
