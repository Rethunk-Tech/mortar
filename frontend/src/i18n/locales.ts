// The one list of languages Mortar ships: the frontend catalogs, the language setting and (through a test) the
// catalogs under internal/i18n/locales are all this set.
import { completeness } from '../locales/completeness.ts'

export const availableLocales = ['en', 'es', 'es-AR', 'es-PA', 'pt-BR'] as const

export type AvailableLocale = (typeof availableLocales)[number]

export function isAvailableLocale(value: string): value is AvailableLocale {
  return availableLocales.includes(value as AvailableLocale)
}

// The share of messages a locale translates, 0 to 1.
function share(locale: string, counts = completeness): number {
  const c = counts[locale]
  return c && c.total > 0 ? c.translated / c.total : 0
}

// A system language selects a locale only above this share: a window of mostly English with a few translated strings is
// worse than all English. Choosing a locale in Settings applies it at any share.
const minSystemShare = 0.8

export function systemLocales(complete = share): AvailableLocale[] {
  return availableLocales.filter((locale) => locale === 'en' || complete(locale) >= minSystemShare)
}

// The picker offers English, any locale with some translation, and the one currently chosen.
export function pickerLocales(selected: string, complete = share): AvailableLocale[] {
  return availableLocales.filter(
    (locale) => locale === 'en' || locale === selected || complete(locale) > 0,
  )
}

const PERCENT = 100
// An incomplete locale never rounds up to reading as complete.
const MOST_INCOMPLETE = PERCENT - 1

// The locale's own name, with its percentage until it is complete.
export function localeLabel(locale: string, complete = share): string {
  const name = new Intl.DisplayNames([locale], { type: 'language' }).of(locale) ?? locale
  const done = complete(locale)
  return done < 1
    ? `${name} (${Math.min(MOST_INCOMPLETE, Math.max(1, Math.round(done * PERCENT)))}%)`
    : name
}

const baseOf = (locale: string) => locale.toLowerCase().split('-')[0]

// The locale closest to a system language among those complete enough: the same code, else the longest locale that is a prefix of it (es-MX takes
// es), else any locale of the same base language (pt-PT takes pt-BR), else English. internal/i18n resolve follows the
// same rule.
export function bestLocale(
  language: string,
  locales: readonly AvailableLocale[] = systemLocales(),
): AvailableLocale {
  const system = language.toLowerCase()
  const exact = locales.find((locale) => locale.toLowerCase() === system)
  if (exact) {
    return exact
  }
  const prefixes = locales
    .filter((locale) => system.startsWith(`${locale.toLowerCase()}-`))
    .sort((a, b) => b.length - a.length)
  return prefixes[0] ?? locales.find((locale) => baseOf(locale) === baseOf(system)) ?? 'en'
}
