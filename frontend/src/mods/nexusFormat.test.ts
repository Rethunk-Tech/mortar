import { expect, test } from 'bun:test'
import type { File } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexus/models.ts'
import {
  currentFiles,
  formatCount,
  formatDate,
  formatSize,
  isNewer,
  recentChangelogs,
} from './nexusFormat.ts'

test('formats sizes, counts and dates, leaving out a missing date', () => {
  expect(formatSize(617, 'en')).toBe('617 kB')
  expect(formatSize(2048, 'en')).toBe('2 MB')
  expect(formatCount(9_915_155, 'en')).toBe('9.9M')
  expect(formatDate('2026-03-15T02:54:41Z', 'en')).toBe('Mar 15, 2026')
  expect(formatDate('0001-01-01T00:00:00Z', 'en')).toBe('')
})

const file = (fileId: number, category: string, uploaded: string): File => ({
  fileId,
  fileName: `${fileId}.zip`,
  name: `${fileId}`,
  description: '',
  version: '1',
  modVersion: '1',
  category,
  sizeKb: 1,
  isPrimary: false,
  uploaded,
})

test('lists the installed file first, then current files newest first', () => {
  const files = [
    file(1, 'OLD_VERSION', '2020-01-01T00:00:00Z'),
    file(2, 'MAIN', '2024-01-01T00:00:00Z'),
    file(3, 'OPTIONAL', '2025-01-01T00:00:00Z'),
    file(4, 'ARCHIVED', '2026-01-01T00:00:00Z'),
    file(5, '', '2026-02-01T00:00:00Z'),
  ]
  expect(currentFiles(files, 1).map((f) => f.fileId)).toEqual([1, 3, 2])
  expect(currentFiles(files, 9).map((f) => f.fileId)).toEqual([3, 2])
})

test('a version is newer by its numbers, ignoring a leading v', () => {
  expect(isNewer('1.10.0', '1.9.2')).toBe(true)
  expect(isNewer('v1.8.2', '1.8.2')).toBe(false)
  expect(isNewer('1.8.1', '1.8.2')).toBe(false)
  expect(isNewer('', '1.0')).toBe(false)
})

test('a release outranks its pre-releases, matching SMAPI CompareVersions', () => {
  const cases: [string, string, number][] = [
    ['1.0', '1.0.0', 0],
    ['2.9.1', '2.10.0', -1],
    ['1.0.0', '1.0.0-beta', 1],
    ['1.0.0-beta', '1.0.0-beta.2', -1],
    ['1.0.0-beta.2', '1.0.0-beta.10', -1],
    ['1.0.0-alpha', '1.0.0-1', 1],
    ['1.0.0-RC1', '1.0.0-rc1', 0],
    ['v1.2.3+build5', '1.2.3', 0],
    ['1.2.3.4', '1.2.3', 1],
    ['3.0.0-unofficial.1-pathoschild', '3.0.0', -1],
  ]
  for (const [a, b, want] of cases) {
    expect(isNewer(a, b)).toBe(want > 0)
    expect(isNewer(b, a)).toBe(want < 0)
  }
  for (const bad of ['', '1', 'abc', '1.x']) {
    expect(isNewer(bad, '1.0')).toBe(false)
  }
})

test('recent changes keep the newest five versions', () => {
  const logs = [1, 2, 3, 4, 5, 6].map((n) => ({ version: `${n}` }))
  expect(recentChangelogs(logs).map((c) => c.version)).toEqual(['1', '2', '3', '4', '5'])
})
