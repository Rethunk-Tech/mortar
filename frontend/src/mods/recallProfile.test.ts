import { expect, test } from 'bun:test'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useMods } from './store.ts'
import { recallProfile } from './storeLoad.ts'

const mod = (id: string) => ({ id, key: id, name: id }) as unknown as Mod

test('a profile shown before comes back with its mods loaded, and an unseen one starts empty', () => {
  const shown = [mod('a'), mod('b')]
  useMods.setState({ mods: shown, loaded: true, modsFor: 'recall-p1' })
  useMods.setState({ mods: [], loaded: false })

  const back = recallProfile('recall-p1')
  expect(back.loaded).toBe(true)
  expect(back.mods).toEqual(shown)

  const fresh = recallProfile('recall-never-opened')
  expect(fresh.loaded).toBe(false)
  expect(fresh.mods).toEqual([])
  expect(fresh.problems).toBeNull()
})

test('a toggle made while the profile is open is what comes back', () => {
  useMods.setState({ mods: [mod('a')], loaded: true, modsFor: 'recall-p2' })
  const toggled = { ...mod('a'), enabled: true } as unknown as Mod
  useMods.setState({ mods: [toggled] })
  expect(recallProfile('recall-p2').mods).toEqual([toggled])
})
