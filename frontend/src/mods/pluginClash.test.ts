import { expect, test } from 'bun:test'
import type { Result } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { problemCount, problemsOf } from './lookup.ts'
import { problemSections } from './problemGroups.ts'

const result = {
  pluginClashes: [
    {
      guid: 'com.x.Cheats',
      keep: 'b',
      copies: [
        { key: 'a', id: 'thunderstore:Ann-Cheats', name: 'Ann-Cheats' },
        { key: 'b', id: 'thunderstore:Bob-Cheats', name: 'Bob-Cheats' },
      ],
    },
  ],
} as Result

test('a plugin shipped by two packages is a counted problem in its own section', () => {
  expect(problemsOf(result).map((p) => p.kind)).toEqual(['pluginClash'])
  expect(problemCount(result)).toBe(1)
  expect(problemSections(result).map((s) => s.id)).toEqual(['pluginClashes'])
})
