import { expect, test } from 'bun:test'
import { nexusPageMark, offersNexusDownload } from './nexusMark.ts'

test('an unpublished or unavailable Nexus page is marked and cannot be downloaded', () => {
  expect(nexusPageMark('published', false, '2026-03-04T12:00:00Z', '2020-01-01')).toEqual({
    kind: 'hidden',
    date: '2026-03-04',
  })
  expect(nexusPageMark('under_moderation', true, '', '2024-06-01T00:00:00Z')).toEqual({
    kind: 'removed',
    date: '2024-06-01',
  })
  expect(nexusPageMark('published', true, 'x', 'y').kind).toBe('')
  expect(offersNexusDownload('hidden', true)).toBe(false)
  expect(offersNexusDownload('published', true)).toBe(true)
  expect(offersNexusDownload(undefined, undefined)).toBe(true)
})
