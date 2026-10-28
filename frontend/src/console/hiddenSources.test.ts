import { expect, test } from 'bun:test'
import { sameSource } from './hiddenSources.ts'

test('sameSource treats a missing text or object as empty', () => {
  const a = { source: 's', label: 'x' }
  expect(sameSource(a, { ...a, text: '' })).toBe(true)
  expect(sameSource(a, { ...a, object: '' })).toBe(true)
  expect(sameSource(a, { ...a, text: 't' })).toBe(false)
  expect(sameSource(a, { ...a, source: 'o' })).toBe(false)
})
