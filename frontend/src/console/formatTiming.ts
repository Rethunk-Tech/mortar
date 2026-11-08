import { formatNumber } from '../i18n/numberFormat.ts'

const PERCENT = 100

export function formatTiming(value: number, locale: string) {
  return formatNumber(value, locale, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

export function percentLabel(share: number, locale: string) {
  return `${formatNumber(share * PERCENT, locale, { maximumFractionDigits: 1 })}%`
}
