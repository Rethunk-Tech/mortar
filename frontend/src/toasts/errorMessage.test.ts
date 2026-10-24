import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { errorDetails, errorKind } from './errorKind.ts'

test("errorKind reads the Wails [kind] prefix for every kind Go's usererr sends", () => {
  const go = readFileSync(join(import.meta.dir, '../../../internal/usererr/usererr.go'), 'utf8')
  const sent = [...go.matchAll(/^\t\w+\s+Kind = "(\w+)"$/gm)].map((m) => m[1] ?? '')
  expect(sent.length).toBeGreaterThan(10)
  for (const kind of sent) {
    expect(errorKind(`[${kind}] x`)).toBe(kind)
  }
  expect(errorKind(new Error('[busy] stardew is running'))).toBe('busy')
  expect(errorDetails('[not_found] profile abc')).toBe('profile abc')
})

test('an untagged bound-call error takes its kind from the Wails cause', () => {
  const readOnly = new Error('open profile.json: permission denied', {
    cause: { kind: 'permission' },
  })
  expect(errorKind(readOnly)).toBe('permission')
  expect(errorDetails(readOnly)).toBe('open profile.json: permission denied')
  expect(errorKind(new Error('dial tcp', { cause: { kind: 'network' } }))).toBe('network')
  expect(errorKind(new Error('x', { cause: { kind: 'bogus' } }))).toBe('unknown')
})

test('untagged text stays unknown with the raw details', () => {
  expect(errorKind('plain failure')).toBe('unknown')
  expect(errorDetails('plain failure')).toBe('plain failure')
})
