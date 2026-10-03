import { expect, test } from 'bun:test'
import { formatBytes } from './bytes.ts'

test('formatBytes pluralises bytes and keeps short larger units', () => {
  expect(formatBytes(0)).toBe('0 bytes')
  expect(formatBytes(1)).toBe('1 byte')
  expect(formatBytes(1536)).toBe('1.5 kB')
})
