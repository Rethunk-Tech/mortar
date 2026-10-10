import { expect, test } from 'bun:test'
import { gameOrder, initialOf } from './order.ts'

const games = ['Stardew Valley', 'Lethal Company', 'Valheim', 'R.E.P.O.', 'PEAK', 'The Sims 4'].map(
  (name) => ({ id: name.toLowerCase(), name }),
)
const names = (list: { name: string }[]) => list.map((g) => g.name)

test('the three most recently used games lead, newest first, and the rest follow by name', () => {
  const { recent, rest } = gameOrder(games, 'peak', {
    valheim: { at: '2026-10-01T00:00:00Z' },
    'stardew valley': { at: '2026-10-05T00:00:00Z' },
    'the sims 4': { at: '2026-10-03T00:00:00Z' },
    peak: { at: '2026-09-01T00:00:00Z' },
  })
  expect(names(recent)).toEqual(['PEAK', 'Stardew Valley', 'The Sims 4'])
  expect(names(rest)).toEqual(['Lethal Company', 'R.E.P.O.', 'Valheim'])
})

test('with nothing used yet every game is listed by name', () => {
  const { recent, rest } = gameOrder(games, '', undefined)
  expect(recent).toEqual([])
  expect(names(rest)).toEqual([
    'Lethal Company',
    'PEAK',
    'R.E.P.O.',
    'Stardew Valley',
    'The Sims 4',
    'Valheim',
  ])
})

test('a game is filed under its first letter, and under # when it starts with a digit', () => {
  expect(initialOf('R.E.P.O.')).toBe('R')
  expect(initialOf('20 Minutes Till Dawn')).toBe('#')
  expect(initialOf('Éclat')).toBe('E')
})
