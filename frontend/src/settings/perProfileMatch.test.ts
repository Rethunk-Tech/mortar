import { expect, test } from 'bun:test'
import { matchPerProfile } from './perProfileMatch.ts'

const LABELS = ['Window mode', 'Zoom', 'UI scale', 'Music volume', 'Sound volume']

test('a search finds the per-profile settings whose label holds every word, in any case', () => {
  expect(matchPerProfile('zoom', LABELS)).toEqual(['Zoom'])
  expect(matchPerProfile('VOLUME', LABELS)).toEqual(['Music volume', 'Sound volume'])
  expect(matchPerProfile('ui sc', LABELS)).toEqual(['UI scale'])
  expect(matchPerProfile('mode window', LABELS)).toEqual(['Window mode'])
})

test('an empty or unmatched search finds nothing', () => {
  expect(matchPerProfile('  ', LABELS)).toEqual([])
  expect(matchPerProfile('gamma', LABELS)).toEqual([])
})
