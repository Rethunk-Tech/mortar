import { expect, test } from 'bun:test'
import { errorDetails, errorKind } from './errorKind.ts'

test('errorKind reads the Wails [kind] prefix', () => {
  const tagged = '[not_found] profile abc'
  expect(errorKind(tagged)).toBe('not_found')
  expect(errorDetails(tagged)).toBe('profile abc')
  expect(errorKind(new Error('[busy] stardew is running'))).toBe('busy')
  expect(errorKind('[network] dial tcp')).toBe('network')
  expect(errorKind('[permission] open')).toBe('permission')
  expect(errorKind('[disk_full] write')).toBe('disk_full')
  expect(errorKind('[damaged] json')).toBe('damaged')
  expect(errorKind('[invalid] id')).toBe('invalid')
  expect(errorKind('[other_game] this share is for lethal-company')).toBe('other_game')
  expect(errorKind('[outdated] made by a newer Mortar')).toBe('outdated')
  expect(errorKind('[external] only on the site')).toBe('external')
  expect(errorKind('[process_denied] Access is denied.')).toBe('process_denied')
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
