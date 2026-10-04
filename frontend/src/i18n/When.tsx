import { i18n } from '@lingui/core'
import type { ReactNode } from 'react'
import { formatWhen } from './formatWhen.ts'
import { useNow } from './useNow.ts'
import { absoluteWhen } from './when.ts'

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
  const title = absoluteWhen(value, i18n.locale)
  if (!title) {
    return <span>{text}</span>
  }
  return <span title={title}>{text}</span>
}
