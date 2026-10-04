import { expect, test } from 'bun:test'
import { formatBytes, formatKb } from './bytes.ts'

test('formatKb formats kibibytes the same as formatBytes on the byte count', () => {
  expect(formatKb(0)).toBe(formatBytes(0))
  expect(formatKb(900)).toBe(formatBytes(900 * 1024))
  expect(formatKb(1536)).toBe(formatBytes(1536 * 1024))
})
