import { expect, test } from 'bun:test'
import { i18n } from '@lingui/core'
import { formatBytes } from '../i18n/bytes.ts'
import { shareLogConfirm } from './shareLog.ts'

test('shareLogConfirm names the size and that the link is public', () => {
  i18n.loadAndActivate({ locale: 'en', messages: {} })
  const bytes = 12_345
  const text = shareLogConfirm(i18n, bytes)
  expect(text).toContain(formatBytes(bytes))
  expect(text.toLowerCase()).toContain('public')
  expect(text).toContain('smapi.io')
})
