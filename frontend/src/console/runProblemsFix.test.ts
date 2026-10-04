import { expect, test } from 'bun:test'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { duplicateCopies } from './runProblemsFix.ts'

const mod = (key: string, name: string): Mod =>
  ({
    key,
    uniqueId: key,
    name,
  }) as Mod

test('Remove duplicate keeps the first copy when two or more match', () => {
  expect(duplicateCopies([mod('a', 'Keep'), mod('b', 'Drop'), mod('c', 'Drop too')])).toEqual({
    keep: mod('a', 'Keep'),
    remove: [mod('b', 'Drop'), mod('c', 'Drop too')],
  })
})

test('a single matching copy is removed with nothing kept', () => {
  expect(duplicateCopies([mod('only', 'Only')])).toEqual({
    remove: [mod('only', 'Only')],
  })
})
