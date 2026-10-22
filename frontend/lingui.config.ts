import { defineConfig } from '@lingui/conf'
import { formatter } from '@lingui/format-po'
import { createSwcExtractor } from '@lingui/native-tools'
import { availableLocales } from './src/i18n/locales.ts'

// A regional locale carries only the messages where its region differs and falls back to its base language: es-AR to es.
const fallbackLocales: Record<string, string> = {}
for (const locale of availableLocales) {
  const [base] = locale.split('-')
  if (base !== undefined && base !== locale && availableLocales.some((l) => l === base)) {
    fallbackLocales[locale] = base
  }
}

const FEATURES = ['mods', 'settings', 'profiles', 'console']

export default defineConfig({
  sourceLocale: 'en',
  orderBy: 'messageId',
  locales: [...availableLocales],
  fallbackLocales,
  extractors: [createSwcExtractor()],
  // References name the file without its line, so moving code does not rewrite the catalog.
  format: formatter({ lineNumbers: false }),
  // A catalog per large feature folder gives translators pieces of a manageable size; compile merges them back into
  // one messages.ts per locale. A message several folders use is in each of their catalogs.
  catalogs: [
    ...FEATURES.map((name) => ({
      path: `<rootDir>/src/locales/{locale}/${name}`,
      include: [`src/${name}`],
    })),
    {
      path: '<rootDir>/src/locales/{locale}/app',
      include: ['src'],
      exclude: FEATURES.map((name) => `src/${name}/**`),
    },
  ],
  catalogsMergePath: '<rootDir>/src/locales/{locale}/messages',
})
