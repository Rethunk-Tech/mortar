import { expect, test } from 'bun:test'
import { formatPicks, parsePicks } from './browsePicks.ts'

test('picks round-trip, and a game never edited has none', () => {
  expect(parsePicks('')).toBeNull()
  expect(parsePicks('not json')).toBeNull()
  const picks = { include: ['Maps'], exclude: [], sort: 'updated' }
  expect(parsePicks(formatPicks(picks))).toEqual(picks)
  // An emptied picker is an edit: it keeps the default excludes from coming back.
  expect(parsePicks(formatPicks({ include: [], exclude: [], sort: 'downloads' }))).not.toBeNull()
})
