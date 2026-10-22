import { expect, test } from 'bun:test'
import { arrangeFixes } from './arrangeFixes.ts'

test('the suggested fix is the button whatever order the fixes came in, and Dismiss ends the menu', () => {
  const { main, menu, quiet } = arrangeFixes([
    { weight: 'quiet', label: 'Dismiss' },
    { weight: 'other', label: 'Disable' },
    { weight: 'main', label: 'Set Map Edits to false' },
    { weight: 'main', label: 'Set MoreRecipes to false' },
  ])
  expect(main?.label).toBe('Set Map Edits to false')
  expect(menu.map((fix) => fix.label)).toEqual(['Set MoreRecipes to false', 'Disable'])
  expect(quiet.map((fix) => fix.label)).toEqual(['Dismiss'])
})

test('without a suggested fix the first other fix leads, and quiet fixes alone make no button', () => {
  expect(arrangeFixes([{ weight: 'other', label: 'Disable' }]).main?.label).toBe('Disable')
  const restoreOnly = arrangeFixes([{ weight: 'quiet', label: 'Restore' }])
  expect(restoreOnly.main).toBeUndefined()
  expect(restoreOnly.quiet.map((fix) => fix.label)).toEqual(['Restore'])
})
