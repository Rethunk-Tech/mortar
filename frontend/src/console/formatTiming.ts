export function formatTiming(value: number, locale: string) {
  return value.toLocaleString(locale, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
