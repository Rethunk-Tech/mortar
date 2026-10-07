import { expect, test } from 'bun:test'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { requirementNameIn } from './requirementName.ts'

const installed = { id: 'smapi:Pathoschild.ContentPatcher', name: 'Content Patcher' } as Mod

test('a requirement is named by an installed copy, else its bare local id, never a placeholder', () => {
  expect(
    requirementNameIn({ mods: [installed], problems: null }, 'Pathoschild.ContentPatcher'),
  ).toBe('Content Patcher')
  expect(requirementNameIn({ mods: [], problems: null }, 'Pathoschild.ContentPatcher')).toBe(
    'Pathoschild.ContentPatcher',
  )
})
