import { expect, test } from 'bun:test'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useMods } from './store.ts'

const mod = (id: string, needs: string[] = []) =>
  ({ key: id, id, name: id, enabled: true, needs }) as Mod

test('a disable that asks first settles only when the player answers, with nothing when they cancel', async () => {
  const lib = mod('Lib.Core')
  useMods.setState({ mods: [lib, mod('Uses.Core', ['Lib.Core'])], disabling: [] })
  let settled: Mod[] | undefined
  const pending = useMods
    .getState()
    .setEnabled(lib, false)
    .then((off) => {
      settled = off
    })
  await Promise.resolve()
  expect(useMods.getState().disabling.map((m) => m.id)).toEqual(['Lib.Core'])
  expect(settled).toBeUndefined()
  useMods.getState().askDisable(null)
  await pending
  expect(settled).toEqual([])
})
