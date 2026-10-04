import { expect, test } from 'bun:test'
import { categoriesDiffer } from './customCategories.ts'

test('categoriesDiffer treats a renamed draft as dirty', () => {
  expect(
    categoriesDiffer(
      [{ id: 'a', name: 'Farm', color: '' }],
      [{ id: 'a', name: 'Farm', color: '' }],
    ),
  ).toBe(false)
  expect(
    categoriesDiffer(
      [{ id: 'a', name: 'Farm', color: '' }],
      [{ id: 'a', name: 'Ranch', color: '' }],
    ),
  ).toBe(true)
})
