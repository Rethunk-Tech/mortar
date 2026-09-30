import { defineConfig } from '@lingui/conf'

export default defineConfig({
  sourceLocale: 'en',
  orderBy: 'messageId',
  locales: ['en'],
  catalogs: [{ path: '<rootDir>/src/locales/{locale}/messages', include: ['src'] }],
})
