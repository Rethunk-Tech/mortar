import { expect, test } from 'bun:test'
import { i18n } from '@lingui/core'
import { cmpText } from './cmpText.ts'

test('text orders by the active language and reads digits as numbers', () => {
  i18n.loadAndActivate({ locale: 'en', messages: {} })
  expect(['mod10', 'mod2'].sort(cmpText)).toEqual(['mod2', 'mod10'])
  expect(['ö', 'z', 'a'].sort(cmpText)).toEqual(['a', 'ö', 'z'])
  i18n.loadAndActivate({ locale: 'sv', messages: {} })
  expect(['ö', 'z', 'a'].sort(cmpText)).toEqual(['a', 'z', 'ö'])
  i18n.loadAndActivate({ locale: 'en', messages: {} })
})
