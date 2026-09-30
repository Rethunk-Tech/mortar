import { expect, test } from 'bun:test'
import { type InstalledMod, linksForModColumn, linksInText } from './consoleLinks.ts'

const mods: InstalledMod[] = [
  { name: 'Farm Type Manager', uniqueID: 'Esca.FarmTypeManager' },
  { name: 'Farm', uniqueID: 'Me.Farm' },
]

const roots = {
  modsDir: '/home/user/.local/share/mortar/profiles/p1/mods',
  gameDir: '/home/user/Games/Stardew Valley',
}

test('message links prefer the longest exact mod name and skip substrings', () => {
  const text = 'SMAPI loaded Farm Type Manager after Farm; skip Farming and Farmland'
  const links = linksInText(text, mods, roots).filter((l) => l.kind === 'mod')
  expect(links.map((l) => ({ t: text.slice(l.start, l.end), id: l.uniqueID }))).toEqual([
    { t: 'Farm Type Manager', id: 'Esca.FarmTypeManager' },
    { t: 'Farm', id: 'Me.Farm' },
  ])
})

test('UniqueID matches are exact and do not fire inside a longer id', () => {
  const extra: InstalledMod[] = [
    ...mods,
    { name: 'Patch', uniqueID: 'Me.Patch' },
    { name: 'Content Patcher', uniqueID: 'Pathoschild.ContentPatcher' },
  ]
  const text = 'Need Pathoschild.ContentPatcher; Me.Patch is separate'
  const links = linksInText(text, extra, roots).filter((l) => l.kind === 'mod')
  expect(links.map((l) => text.slice(l.start, l.end))).toEqual([
    'Pathoschild.ContentPatcher',
    'Me.Patch',
  ])
})

test('the mod column is a link only when it equals a name or UniqueID', () => {
  expect(linksForModColumn('Farm Type Manager', mods)).toEqual([
    { kind: 'mod', start: 0, end: 17, uniqueID: 'Esca.FarmTypeManager' },
  ])
  expect(linksForModColumn('Esca.FarmTypeManager', mods)).toEqual([
    { kind: 'mod', start: 0, end: 20, uniqueID: 'Esca.FarmTypeManager' },
  ])
  expect(linksForModColumn('Farm Type', mods)).toEqual([])
  expect(linksForModColumn('SMAPI', mods)).toEqual([])
})

test('paths under the profile mods folder or game folder become links', () => {
  const text =
    'wrote /home/user/.local/share/mortar/profiles/p1/mods/FTM/manifest.json and /home/user/Games/Stardew Valley/Content/Data/Farms.xml'
  const links = linksInText(text, mods, roots).filter((l) => l.kind === 'path')
  expect(links.map((l) => text.slice(l.start, l.end))).toEqual([
    '/home/user/.local/share/mortar/profiles/p1/mods/FTM/manifest.json',
    '/home/user/Games/Stardew Valley/Content/Data/Farms.xml',
  ])
})

test('paths outside the roots are not links', () => {
  const text = 'see /tmp/Farm and /home/user/Games/Other/manifest.json'
  expect(linksInText(text, mods, roots).filter((l) => l.kind === 'path')).toEqual([])
})
