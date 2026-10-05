import { expect, test } from 'bun:test'
import { sourceLabel } from './sourceLabel.ts'

test('Unknown sources show their id', () => {
  expect(sourceLabel('curseforge')).toBe('curseforge')
})
