import { expect, test } from 'bun:test'
import { typedMatch } from './typeahead.ts'

const names = ['Lethal Company', 'R.E.P.O.', 'Stardew Valley', 'The Sims 4', 'Auto-Gravestones']
const find = (typed: string) => typedMatch(names, typed, (n) => n)

test('typing finds a name whatever punctuation it has, by its start before its middle', () => {
  expect(find('let')).toBe('Lethal Company')
  expect(find('rep')).toBe('R.E.P.O.')
  expect(find('sims 4')).toBe('The Sims 4')
  expect(find('autogr')).toBe('Auto-Gravestones')
  expect(find('s')).toBe('Stardew Valley')
  expect(find('zzz')).toBeUndefined()
  expect(find('.')).toBeUndefined()
})
