import { expect, test } from 'bun:test'
import type { ProjectInfo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { isProjectRow } from './projectRow.ts'

test('a report row is the project by its name, its GUID, or an id ending in the GUID', () => {
  const info = { name: 'LandDrop', guid: 'com.nomadtest.LandDrop' } as ProjectInfo
  expect(isProjectRow(info, 'landdrop')).toBe(true)
  expect(isProjectRow(info, 'com.nomadtest.landdrop')).toBe(true)
  expect(isProjectRow(info, 'BepInEx plugin', 'bepinex:com.nomadtest.LandDrop')).toBe(true)
  expect(isProjectRow(info, 'ShopDrop', 'bepinex:com.other.ShopDrop')).toBe(false)
})
