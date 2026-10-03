import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { formatBytes } from '../i18n/bytes.ts'

export function shareLogConfirm(i18n: I18n, bytes: number): string {
  const size = formatBytes(bytes)
  return i18n._(
    msg`This log is ${size}. Sharing uploads it to smapi.io, where it becomes public at a link.`,
  )
}
