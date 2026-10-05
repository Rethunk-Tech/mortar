import { expect, test } from 'bun:test'
import {
  changeStillLatest,
  HISTORY_CAP,
  historyActionState,
  laterEvents,
  prependHistory,
} from './history.ts'

test('prependHistory keeps newest first and drops past the cap', () => {
  const olderFirstNewest = Array.from({ length: HISTORY_CAP }, (_, i) => HISTORY_CAP - 1 - i)
  const items = prependHistory(olderFirstNewest, HISTORY_CAP)
  expect(items).toHaveLength(HISTORY_CAP)
  expect(items[0]).toBe(HISTORY_CAP)
  expect(items.at(-1)).toBe(1)
})

test('changeStillLatest allows undo only while that entry is still the latest for each mod', () => {
  const profile = {
    entries: [
      {
        key: 'k1',
        mods: [{ id: 'SpaceCore' }],
      },
    ],
  }
  expect(changeStillLatest(profile, 'k1', ['SpaceCore'])).toEqual({ disabled: false })
  expect(changeStillLatest(profile, 'old', ['SpaceCore']).disabled).toBe(true)
  expect(changeStillLatest(undefined, 'k1', ['SpaceCore']).disabled).toBe(true)
})

test('historyActionState prefers the lock reason over live()', () => {
  expect(
    historyActionState(() => ({ disabled: false }), true, 'Stop the game to change mods.'),
  ).toEqual({ disabled: true, reason: 'Stop the game to change mods.' })
  expect(historyActionState(() => ({ disabled: true, reason: 'stale' }), false, 'locked')).toEqual({
    disabled: true,
    reason: 'stale',
  })
})

test('laterEvents lists the changes newer than the one undone', () => {
  const events = [{ id: 'c' }, { id: 'b' }, { id: 'a' }]
  expect(laterEvents(events, 'a').map((e) => e.id)).toEqual(['c', 'b'])
  expect(laterEvents(events, 'c')).toEqual([])
  expect(laterEvents(events, 'zz')).toEqual([])
})
