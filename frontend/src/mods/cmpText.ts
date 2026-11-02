import { i18n } from '@lingui/core'

const collators = new Map<string, Intl.Collator>()

/** Orders text for the app's language: numbers by value, case and accents ignored. */
export function cmpText(a: string, b: string): number {
  const locale = i18n.locale
  let collator = collators.get(locale)
  if (!collator) {
    collator = new Intl.Collator(locale || undefined, { numeric: true, sensitivity: 'base' })
    collators.set(locale, collator)
  }
  return collator.compare(a, b)
}
