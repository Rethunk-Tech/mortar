import { expect, test } from 'bun:test'
import { shouldShowExternalImportDivider } from './importMenu.ts'

test('Profiles import menu shows a divider only when external importers exist', () => {
  expect(shouldShowExternalImportDivider(0)).toBe(false)
  expect(shouldShowExternalImportDivider(2)).toBe(true)
})
