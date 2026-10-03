import { expect, test } from 'bun:test'
import { shareIncludeDefaults } from './shareDefaults.ts'

test('share include defaults match registry defaults', () => {
  expect(shareIncludeDefaults({})).toEqual({
    disabledMods: false,
    fomodChoices: true,
    notes: true,
    configFiles: true,
  })
  expect(
    shareIncludeDefaults({
      shareIncludeDisabledMods: true,
      shareIncludeFomodChoices: false,
      shareIncludeNotes: false,
      shareIncludeConfigFiles: false,
    }),
  ).toEqual({
    disabledMods: true,
    fomodChoices: false,
    notes: false,
    configFiles: false,
  })
})
