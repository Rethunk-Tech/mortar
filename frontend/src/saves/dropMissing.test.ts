import { expect, test } from 'bun:test'
import type { Fit } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/models.ts'
import { dropMissing } from './dropMissing.ts'

function saveFit(folder: string, id: string): Fit {
  return {
    folder,
    farm: 'Sunny',
    farmer: 'A',
    season: 0,
    day: 1,
    year: 1,
    played: 0,
    whichFarm: 0,
    millisecondsPlayed: 0,
    money: 0,
    missing: [{ id, name: id, disabled: true, where: null }],
    lastProfileId: '',
    lastProfileAt: 0,
    lastProfileExists: false,
    lastMods: null,
    lastMissing: null,
  }
}

test('enabling a switched-off mod drops it from every save missing list', () => {
  const next = dropMissing(
    [saveFit('Farm_1', 'SpaceCore'), saveFit('Farm_2', 'SpaceCore')],
    'SpaceCore',
  )
  expect(next[0]?.missing).toEqual([])
  expect(next[1]?.missing).toEqual([])
})

test('dismiss drops a UniqueID only on that save folder', () => {
  const next = dropMissing(
    [saveFit('Farm_1', 'SpaceCore'), saveFit('Farm_2', 'SpaceCore')],
    'SpaceCore',
    'Farm_1',
  )
  expect(next[0]?.missing).toEqual([])
  expect(next[1]?.missing).toEqual([
    { id: 'SpaceCore', name: 'SpaceCore', disabled: true, where: null },
  ])
})
