import { defineConfig } from '@lingui/conf'
import { availableLocales } from './src/i18n/locales.ts'

export default defineConfig({
  sourceLocale: 'en',
  orderBy: 'messageId',
  locales: [...availableLocales],
  catalogs: [{ path: '<rootDir>/src/locales/{locale}/messages', include: ['src'] }],
})
