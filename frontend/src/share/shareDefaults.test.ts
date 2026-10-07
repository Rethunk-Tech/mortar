import { expect, test } from 'bun:test'
import { offersFomod, shareIncludeDefaults } from './shareDefaults.ts'

test('share include defaults match registry defaults', () => {
  expect(shareIncludeDefaults({})).toEqual({
    disabledMods: false,
    fomodChoices: true,
    notes: true,
    configFiles: true,
    problemChoices: true,
  })
  expect(
    shareIncludeDefaults({
      shareIncludeDisabledMods: true,
      shareIncludeFomodChoices: false,
      shareIncludeNotes: false,
      shareIncludeConfigFiles: false,
      shareIncludeProblemChoices: false,
    }),
  ).toEqual({
    disabledMods: true,
    fomodChoices: false,
    notes: false,
    configFiles: false,
    problemChoices: false,
  })
})

test('FOMOD choices are offered only when an entry has choices', () => {
  expect(offersFomod([])).toBe(false)
  expect(offersFomod([{ fomod: {} }, {}])).toBe(false)
  expect(offersFomod([{ fomod: { Main: { Pick: ['A'] } } }])).toBe(true)
  expect(offersFomod(null)).toBe(false)
})
