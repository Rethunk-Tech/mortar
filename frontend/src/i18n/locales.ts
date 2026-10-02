export const availableLocales = ['en'] as const

export type AvailableLocale = (typeof availableLocales)[number]

export function isAvailableLocale(value: string): value is AvailableLocale {
  return availableLocales.includes(value as AvailableLocale)
}
