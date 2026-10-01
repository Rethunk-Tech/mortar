import { expect, test } from 'bun:test'
import type { Result } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { problemCount, problemSections } from './problemGroups.ts'

const emptyResult = (): Result => ({
  duplicates: [],
  broken: [],
  missing: [],
  assetConflicts: [],
  runErrors: [],
  drift: [],
  unknown: false,
})

test('problemSections omits empty groups and keeps order', () => {
  const result: Result = {
    ...emptyResult(),
    missing: [
      {
        dependentId: 'd',
        dependentName: 'Dep',
        uniqueId: 'a',
        minimumVersion: '',
        installedVersion: '',
        reason: 'absent',
        listed: false,
        note: '',
        optional: false,
        where: null,
      },
    ],
    duplicates: [
      {
        uniqueId: 'u',
        name: 'Dup',
        copies: null,
      },
    ],
  }
  const sections = problemSections(result)
  expect(sections.map((s) => s.id)).toEqual(['missing', 'duplicates'])
  expect(problemCount(result)).toBe(2)
})

test('problemCount is zero while result is null', () => {
  expect(problemCount(null)).toBe(0)
})
