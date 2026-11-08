import { expect, test } from 'bun:test'
import { i18n } from '../i18n/index.ts'
import { lockedInHint } from './lockedHint.ts'

test('lockedInHint names the profile the running game holds', () => {
  expect(i18n._(lockedInHint('Farm'))).toBe(
    'Farm is in use by the running game. Close the game to change its mods.',
  )
})
