import { expect, test } from 'bun:test'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useTab } from '../game/tab.ts'
import { useDetail } from './detail.ts'
import { openMod } from './openMod.ts'
import { useMods } from './store.ts'

const mod = (key: string, id: string) => ({ key, id, name: id }) as Mod

test('openMod opens the details of the mod with that id and entry key, loading the list when needed', async () => {
  useDetail.setState(useDetail.getInitialState(), true)
  useTab.getState().setTab('browse')
  let loads = 0
  useMods.setState({
    ...useMods.getInitialState(),
    mods: [],
    load: async () => {
      loads += 1
      useMods.setState({ mods: [mod('a', 'Same.Id'), mod('b', 'Same.Id')] })
    },
  })
  await openMod({ id: 'same.id', key: 'b' })
  expect(loads).toBe(1)
  expect(useTab.getState().tab).toBe('mods')
  expect(useDetail.getState().detailId).toBe('b/Same.Id')
})
