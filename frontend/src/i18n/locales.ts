// The one list of languages Mortar ships: the frontend catalogs, the language setting and (through a test) the
// catalogs under internal/i18n/locales are all this set.
export const availableLocales = ['en', 'es', 'es-AR', 'es-PA', 'pt-BR'] as const

export type AvailableLocale = (typeof availableLocales)[number]

export function isAvailableLocale(value: string): value is AvailableLocale {
  return availableLocales.includes(value as AvailableLocale)
}

const baseOf = (locale: string) => locale.toLowerCase().split('-')[0]

// The locale closest to a system language: the same code, else the longest locale that is a prefix of it (es-MX takes
// es), else any locale of the same base language (pt-PT takes pt-BR), else English. internal/i18n resolve follows the
// same rule.
export function bestLocale(language: string): AvailableLocale {
  const system = language.toLowerCase()
  const exact = availableLocales.find((locale) => locale.toLowerCase() === system)
  if (exact) {
    return exact
  }
  const prefixes = availableLocales
    .filter((locale) => system.startsWith(`${locale.toLowerCase()}-`))
    .sort((a, b) => b.length - a.length)
  return prefixes[0] ?? availableLocales.find((locale) => baseOf(locale) === baseOf(system)) ?? 'en'
}
