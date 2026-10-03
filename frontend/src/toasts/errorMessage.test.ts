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
