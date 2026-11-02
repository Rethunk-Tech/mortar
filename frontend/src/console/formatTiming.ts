import { formatNumber } from '../i18n/numberFormat.ts'

export function formatTiming(value: number, locale: string) {
  return formatNumber(value, locale, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
