import { expect, test } from 'bun:test'
import { baseName } from './baseName.ts'

test('baseName takes the last part with either separator', () => {
  expect(baseName('/home/me/mod.zip')).toBe('mod.zip')
  expect(baseName('C:\\Users\\me\\mod.zip')).toBe('mod.zip')
  expect(baseName('/home/me/Mods/')).toBe('Mods')
  expect(baseName('mod.zip')).toBe('mod.zip')
})
