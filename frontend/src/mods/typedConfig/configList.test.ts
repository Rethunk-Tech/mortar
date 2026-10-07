import { expect, test } from 'bun:test'
import type { ModConfig } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/configsvc/models.ts'
import { chipOf, filterConfigMods, hasConfig, selectionOf } from './configList.ts'

const mod = (id: string, name: string, over: Partial<ModConfig> = {}): ModConfig => ({
  id,
  key: id,
  name,
  enabled: true,
  files: [{ name: 'config.json', format: 'smapi', changed: false }],
  hasMenu: false,
  pending: 0,
  ...over,
})

const mods = [
  mod('Pathoschild.ContentPatcher', 'Content Patcher', {
    files: [{ name: 'config.json', format: 'smapi', changed: true }],
  }),
  mod('Esca.FarmTypeManager', 'Farm Type Manager', { hasMenu: true }),
  mod('Spacechase0.SpaceCore', 'SpaceCore', { hasMenu: true, pending: 2 }),
]

test('a row carries one chip: waiting edits, then changed, then the in-game menu', () => {
  expect(chipOf(mods[2] as ModConfig)).toEqual({ kind: 'waiting', n: 2 })
  expect(chipOf(mods[0] as ModConfig)).toEqual({ kind: 'changed' })
  expect(chipOf(mods[1] as ModConfig)).toEqual({ kind: 'menu' })
  expect(chipOf(mod('A.B', 'AB'))).toBeNull()
})

test('the list filter takes the Show choice and a name or id fragment', () => {
  const names = (show: Parameters<typeof filterConfigMods>[2], q = '') =>
    filterConfigMods(mods, q, show).map((m) => m.name)
  expect(names('all')).toHaveLength(3)
  expect(names('changed')).toEqual(['Content Patcher'])
  expect(names('menu')).toEqual(['Farm Type Manager', 'SpaceCore'])
  expect(names('waiting')).toEqual(['SpaceCore'])
  expect(names('all', 'farm')).toEqual(['Farm Type Manager'])
  expect(names('all', 'pathoschild')).toEqual(['Content Patcher'])
  expect(names('menu', 'patcher')).toEqual([])
})

test('the remembered selection holds while its mod is listed, else the first mod is chosen', () => {
  const list = {
    mods,
    other: [{ name: 'BepInEx.cfg', format: 'bepinex', changed: false }],
    without: 0,
  }
  expect(selectionOf(list, 'mod:Esca.FarmTypeManager', 'p')).toBe('mod:Esca.FarmTypeManager')
  expect(selectionOf(list, 'file:BepInEx.cfg', 'p')).toBe('file:BepInEx.cfg')
  expect(selectionOf(list, 'mod:Gone.Mod', 'p')).toBe('mod:Pathoschild.ContentPatcher')
  expect(selectionOf({ mods: [], other: list.other, without: 0 }, 'x', 'p')).toBe(
    'file:BepInEx.cfg',
  )
  expect(selectionOf(undefined, 'x', 'p')).toBe('')
})

test('a mod has a config source when the list holds it', () => {
  expect(hasConfig({ mods, other: [], without: 0 }, 'Esca.FarmTypeManager')).toBe(true)
  expect(hasConfig({ mods, other: [], without: 0 }, 'No.Config')).toBe(false)
  expect(hasConfig(undefined, 'Esca.FarmTypeManager')).toBe(false)
})
