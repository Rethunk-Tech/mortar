import { expect, test } from 'bun:test'
import { compatOf, showCompatChip } from './compatChip.ts'

test('showCompatChip is only for non-ok statuses', () => {
  expect(showCompatChip('ok')).toBe(false)
  expect(showCompatChip('OK')).toBe(false)
  expect(showCompatChip('')).toBe(false)
  expect(showCompatChip(undefined)).toBe(false)
  expect(showCompatChip('broken')).toBe(true)
  expect(showCompatChip('unofficial')).toBe(true)
  expect(showCompatChip('optional')).toBe(true)
  expect(showCompatChip('obsolete')).toBe(true)
  expect(showCompatChip('abandoned')).toBe(true)
})

test('compatOf matches UniqueID and key', () => {
  const rows = [
    { key: 'a', uniqueId: 'Author.Broken', status: 'broken' },
    { key: 'b', uniqueId: 'Author.Ok', status: 'ok' },
  ]
  expect(compatOf(rows, { key: 'a', uniqueId: 'author.broken' })?.status).toBe('broken')
  expect(compatOf(rows, { key: 'b', uniqueId: 'Author.Ok' })?.status).toBe('ok')
  expect(compatOf(rows, { key: 'a', uniqueId: 'Author.Ok' })).toBeUndefined()
})
