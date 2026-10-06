import { expect, test } from 'bun:test'
import { offersFomod, shareIncludeDefaults } from './shareDefaults.ts'

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

test('FOMOD choices are offered only with Nexus or an entry that has choices', () => {
  expect(offersFomod([], ['thunderstore', 'github'])).toBe(false)
  expect(offersFomod([{ fomod: {} }, {}], ['thunderstore'])).toBe(false)
  expect(offersFomod([{ fomod: { Main: { Pick: ['A'] } } }], ['thunderstore'])).toBe(true)
  expect(offersFomod(null, ['nexus'])).toBe(true)
})
