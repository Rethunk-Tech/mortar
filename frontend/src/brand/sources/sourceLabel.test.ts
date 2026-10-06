import { expect, test } from 'bun:test'
import { sourceLabel } from './sourceLabel.ts'

test('Known sources show their brand name', () => {
  expect(sourceLabel('curseforge')).toBe('CurseForge')
})

test('Unknown sources show their id', () => {
  expect(sourceLabel('somewhere')).toBe('somewhere')
})
