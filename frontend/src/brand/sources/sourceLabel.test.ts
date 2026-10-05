import { expect, test } from 'bun:test'
import { sourceLabel } from './sourceLabel.ts'

test('ModDrop has a label and unknown sources show their id', () => {
  expect(sourceLabel('moddrop')).toBe('ModDrop')
  expect(sourceLabel('curseforge')).toBe('curseforge')
})
