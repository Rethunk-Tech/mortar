import { expect, test } from 'bun:test'
import type { DevProject } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { movedDirs, placeInSlots, sortProjects } from './projectOrder.ts'

const p = (dir: string, name = dir, profile = 'one'): DevProject =>
  ({ dir, name, profile, output: dir }) as DevProject

test('by name ignores case and reads numbers by value; manual keeps the stored order', () => {
  const list = [p('1', 'beta'), p('2', 'Alpha 10'), p('3', 'alpha 2')]
  expect(sortProjects(list, 'name').map((x) => x.dir)).toEqual(['3', '2', '1'])
  expect(sortProjects(list, 'manual')).toBe(list)
})

test('a move puts the dragged dir where the target was, and an unknown dir changes nothing', () => {
  const dirs = ['a', 'b', 'c']
  expect(movedDirs(dirs, 'a', 'c')).toEqual(['b', 'c', 'a'])
  expect(movedDirs(dirs, 'c', 'a')).toEqual(['c', 'a', 'b'])
  expect(movedDirs(dirs, 'x', 'a')).toBe(dirs)
})

test('reordering one profile leaves the other profile in its slots', () => {
  const all = [p('a'), p('x', 'x', 'two'), p('b'), p('c')]
  expect(placeInSlots(all, ['c', 'a', 'b']).map((q) => q.dir)).toEqual(['c', 'x', 'a', 'b'])
})
