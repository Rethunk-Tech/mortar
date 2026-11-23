import { expect, test } from 'bun:test'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { shortFileNames } from './fileNames.ts'

const mod = (key: string, name: string) => ({ key, name }) as Mod

test('per-file names drop the folders their download shares', () => {
  const got = shortFileNames([
    mod('a#cet/mods/foo/init.lua', 'cet/mods/foo/init.lua'),
    mod('a#cet/mods/foo/lib/x.lua', 'cet/mods/foo/lib/x.lua'),
    mod('b#paks/one.pak', 'Renamed'),
    mod('c#root.pak', 'root.pak'),
    mod('d', 'Whole mod'),
  ]).map((m) => m.name)
  expect(got).toEqual(['init.lua', 'lib/x.lua', 'Renamed', 'root.pak', 'Whole mod'])
})
