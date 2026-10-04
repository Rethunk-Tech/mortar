import { expect, test } from 'bun:test'
import type { Result } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { problemCount, problemsOf } from './lookup.ts'
import { problemSections } from './problemGroups.ts'

const result = {
  damaged: [{ key: 'nexus-1-2', name: 'Pack', missing: 1, changed: 0, extra: 0, files: ['a'] }],
} as Result

test('a damaged mod is a counted problem with its own section', () => {
  expect(problemsOf(result).map((p) => p.kind)).toEqual(['damaged'])
  expect(problemCount(result)).toBe(1)
  expect(problemSections(result).map((s) => s.id)).toEqual(['damaged'])
})
