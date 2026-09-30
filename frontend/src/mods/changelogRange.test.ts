import { expect, test } from 'bun:test'
import type { Changelog } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexus/models.ts'
import { changelogsBetween } from './changelogRange.ts'

const log = (version: string): Changelog => ({ version, notes: [`notes ${version}`] })

test('keeps changelog versions newer than installed, including latest, newest first', () => {
  const logs = ['1.3.0', '1.2.0', '1.1.0', '1.0.0', '0.9.0'].map(log)
  expect(changelogsBetween(logs, '1.0.0', '1.3.0').map((c) => c.version)).toEqual([
    '1.3.0',
    '1.2.0',
    '1.1.0',
  ])
})

test('treats a leading v and pre-releases the way isNewer does', () => {
  const logs = ['1.1.0', '1.1.0-beta', '1.0.0'].map(log)
  expect(changelogsBetween(logs, '1.0.0', '1.1.0').map((c) => c.version)).toEqual([
    '1.1.0',
    '1.1.0-beta',
  ])
})

test('an empty or missing changelog list is empty', () => {
  expect(changelogsBetween(null, '1.0.0', '2.0.0')).toEqual([])
  expect(changelogsBetween([], '1.0.0', '2.0.0')).toEqual([])
})
