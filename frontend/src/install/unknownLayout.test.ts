import { expect, test } from 'bun:test'
import { unknownLayoutOf } from './unknownLayout.ts'

const detail = { game: 'cyberpunk-2077', key: 'local-ab', files: ['a/b.dat'], overridable: true }

test('unknownLayoutOf reads the typed detail an install error carries', () => {
  const e = new Error('[invalid] no layout', { cause: { kind: 'invalid', detail } })
  expect(unknownLayoutOf(e)).toEqual(detail)
})

test('unknownLayoutOf ignores any other error', () => {
  expect(
    unknownLayoutOf(new Error('x', { cause: { kind: 'invalid', detail: { key: 'k' } } })),
  ).toBeNull()
  expect(unknownLayoutOf('boom')).toBeNull()
})
