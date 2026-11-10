import { expect, test } from 'bun:test'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { byRequirementName, requirementNameIn } from './requirementName.ts'

const installed = { id: 'smapi:Pathoschild.ContentPatcher', name: 'Content Patcher' } as Mod

test('a requirement is named by an installed copy, else its bare local id, never a placeholder', () => {
  expect(
    requirementNameIn({ mods: [installed], problems: null }, 'Pathoschild.ContentPatcher'),
  ).toBe('Content Patcher')
  expect(requirementNameIn({ mods: [], problems: null }, 'Pathoschild.ContentPatcher')).toBe(
    'Pathoschild.ContentPatcher',
  )
})

test('a tree level is ordered by the names it shows, not the manifest order', () => {
  const mods = [
    installed,
    { id: 'smapi:spacechase0.SpaceCore', name: 'SpaceCore' } as Mod,
    { id: 'smapi:aedenthorn.AdvancedMenu', name: 'advanced menu' } as Mod,
  ]
  const nodes = [
    { id: 'spacechase0.SpaceCore' },
    { id: 'Zed.Unlisted' },
    { id: 'Pathoschild.ContentPatcher' },
    { id: 'aedenthorn.AdvancedMenu' },
  ]
  expect(byRequirementName(nodes, { mods, problems: null }).map((n) => n.id)).toEqual([
    'aedenthorn.AdvancedMenu',
    'Pathoschild.ContentPatcher',
    'spacechase0.SpaceCore',
    'Zed.Unlisted',
  ])
})
