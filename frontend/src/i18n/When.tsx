import { i18n } from '@lingui/core'
import { absoluteWhen, useNowValue } from '@rethunk/react-load-hooks'
import type { ReactNode } from 'react'
import { formatWhen } from './formatWhen.ts'

interface WhenProps {
  value: string | number | Date
  withTime?: boolean
}

export function When({ value, withTime = false }: WhenProps): ReactNode {
  const text = useNowValue(() => formatWhen(value, { withTime }))
  if (!text) {
    return null
  }
  const title = absoluteWhen(value, i18n.locale)
  if (!title) {
    return <span>{text}</span>
  }
  return <span title={title}>{text}</span>
}
