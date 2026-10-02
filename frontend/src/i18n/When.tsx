import { i18n } from '@lingui/core'
import type { ReactNode } from 'react'
import { formatWhen } from './formatWhen.ts'
import { useNow } from './useNow.ts'

interface WhenProps {
  value: string | number | Date
  withTime?: boolean
}

export function When({ value, withTime = false }: WhenProps): ReactNode {
  useNow()
  const text = formatWhen(value, { withTime })
  if (!text) {
    return null
  }
  const date = value instanceof Date ? value : new Date(value)
  const title = new Intl.DateTimeFormat(i18n.locale, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
  return <span title={title}>{text}</span>
}
