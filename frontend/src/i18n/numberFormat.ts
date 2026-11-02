import { i18n } from '@lingui/core'

const formats = new Map<string, Intl.NumberFormat>()

/** A shared Intl.NumberFormat per locale and options: building one per call is the slow part of toLocaleString. */
export function numberFormat(
  locale: string = i18n.locale,
  options?: Intl.NumberFormatOptions,
): Intl.NumberFormat {
  const key = `${locale}|${options ? JSON.stringify(options) : ''}`
  let format = formats.get(key)
  if (!format) {
    format = new Intl.NumberFormat(locale || undefined, options)
    formats.set(key, format)
  }
  return format
}

export function formatNumber(
  value: number,
  locale: string = i18n.locale,
  options?: Intl.NumberFormatOptions,
): string {
  return numberFormat(locale, options).format(value)
}
