import { i18n } from '@lingui/core'

const UNITS = ['byte', 'kilobyte', 'megabyte', 'gigabyte'] as const
const KIB = 1024

export function formatBytes(bytes: number): string {
  const n = Number.isFinite(bytes) ? Math.max(0, bytes) : 0
  let unitIndex = 0
  let value = n
  while (value >= KIB && unitIndex < UNITS.length - 1) {
    value /= KIB
    unitIndex += 1
  }
  return new Intl.NumberFormat(i18n.locale || 'en', {
    style: 'unit',
    unit: UNITS[unitIndex],
    unitDisplay: 'short',
    maximumFractionDigits: unitIndex === 0 ? 0 : 1,
  }).format(value)
}
