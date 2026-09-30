import { expect, test } from 'bun:test'
import { MAX_HISTORY, pushCommand, stepHistory } from './history.ts'

test('blank commands and immediate repeats are not kept', () => {
  let h = pushCommand([], '  help ')
  h = pushCommand(h, '')
  h = pushCommand(h, 'help')
  expect(h).toEqual(['help'])
  expect(pushCommand(h, 'list_mods')).toEqual(['help', 'list_mods'])
  expect(pushCommand(['a', 'b'], 'a')).toEqual(['a', 'b', 'a'])
})

test('only the newest 100 are kept', () => {
  let h: string[] = []
  for (let i = 0; i < MAX_HISTORY + 5; i++) {
    h = pushCommand(h, `c${i}`)
  }
  expect(h).toHaveLength(MAX_HISTORY)
  expect(h[0]).toBe('c5')
  expect(h.at(-1)).toBe(`c${MAX_HISTORY + 4}`)
})

test('Up walks back and stops at the oldest, Down returns to the draft', () => {
  const h = ['a', 'b', 'c']
  let at = h.length
  at = stepHistory(h, at, -1)
  expect(at).toBe(2)
  at = stepHistory(h, at, -1)
  at = stepHistory(h, at, -1)
  at = stepHistory(h, at, -1)
  expect(at).toBe(0)
  at = stepHistory(h, at, 1)
  at = stepHistory(h, at, 1)
  at = stepHistory(h, at, 1)
  at = stepHistory(h, at, 1)
  expect(at).toBe(h.length)
  expect(stepHistory([], 0, -1)).toBe(0)
})
