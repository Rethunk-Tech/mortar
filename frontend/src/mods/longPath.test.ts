import { expect, test } from 'bun:test'
import type { Result } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { modStatusProblem, problemCount, problemsOf } from './lookup.ts'
import { problemSections } from './problemGroups.ts'

const result = {
  longPaths: [
    { key: 'nexus-1-2', name: 'Pack', files: 1, longest: 'x', length: 270, placed: false },
  ],
} as Result

test('a mod with a path too long for Windows is a counted problem with its own section', () => {
  expect(problemsOf(result).map((p) => p.kind)).toEqual(['longPath'])
  expect(problemCount(result)).toBe(1)
  expect(problemSections(result).map((s) => s.id)).toEqual(['longPaths'])
})

test('a long path flags its mod, and one in a folder no mod owns flags none', () => {
  const unowned = {
    longPaths: [{ key: '', name: 'Loose', files: 1, longest: 'x', length: 270, placed: false }],
  } as Result
  expect(modStatusProblem(result, { key: 'nexus-1-2' } as Mod)).toBe(true)
  expect(modStatusProblem(unowned, { key: 'nexus-1-2' } as Mod)).toBe(false)
  expect(problemCount(unowned)).toBe(1)
})
