import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
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

test('errorMessage maps each kind to the same sentence as the CLI', () => {
  const src = readFileSync(join(import.meta.dir, 'report.ts'), 'utf8')
  expect(src).toContain('msg`That item could not be found.`')
  expect(src).toContain('msg`The game is already running.`')
  expect(src).toContain('msg`A network request failed.`')
  expect(src).toContain('msg`Mortar does not have permission to do that.`')
  expect(src).toContain('msg`The disk is full.`')
  expect(src).toContain('msg`That data could not be read.`')
  expect(src).toContain('msg`That request was not valid.`')
  expect(src).toContain('export function errorMessage')
})

test('untagged errors use the generic sentence; raw text stays in errorDetails', () => {
  const src = readFileSync(join(import.meta.dir, 'report.ts'), 'utf8')
  expect(src).toContain('return sentence(kindOf(e))')
  expect(src).toContain('msg`Something went wrong`')
  expect(src).not.toContain('untagged errors keep their text')
  expect(src).not.toContain('const raw = detailsOf(e)')
})
