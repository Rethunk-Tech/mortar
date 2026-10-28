import { expect, test } from 'bun:test'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { modById, modByKey } from './lookup.ts'

const mod = (id: string, key: string) => ({ id, key }) as Mod

test('mods are found by id ignoring case and space, and by key, first one winning', () => {
  const a = mod('Me.Mod', 'k1')
  const dup = mod('me.mod', 'k2')
  const mods = [a, dup]
  expect(modById(mods, '  ME.MOD ')).toBe(a)
  expect(modById(mods, 'other')).toBeUndefined()
  expect(modByKey(mods, 'k2')).toBe(dup)
  expect(modByKey(mods, 'none')).toBeUndefined()
})
