import { expect, test } from 'bun:test'
import { missingDepName } from './describe.ts'

const nameOf = (id: string) => (id === 'smapi:A.Lib' ? 'A Lib' : id.replace(/^smapi:/, ''))
const dep = (id: string, pageName: string, listed = false) =>
  missingDepName({ id, listed, where: pageName === '' ? null : { pageName } }, nameOf)

test('a missing requirement is named by an installed copy, then its page, and by its id only last', () => {
  expect(dep('smapi:Pathoschild.ContentPatcher', 'Content Patcher')).toBe('Content Patcher')
  expect(dep('smapi:A.Lib', 'A Lib page')).toBe('A Lib')
  expect(dep('smapi:A.Lib', 'A Lib page', true)).toBe('A Lib page')
  expect(dep('smapi:No.Page', '')).toBe('No.Page')
})
