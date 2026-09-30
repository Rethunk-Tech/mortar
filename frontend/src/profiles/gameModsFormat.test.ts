import { describe, expect, test } from 'bun:test'
import {
  formatOutcomeDetail,
  formatPreviewRow,
  willImport,
  withSwitchedOff,
} from './gameModsFormat.ts'

describe('gameModsFormat', () => {
  test('marks switched-off mods and lists skip reasons instead of imports', () => {
    expect(willImport('imported')).toBe(true)
    expect(willImport('skipped')).toBe(false)
    expect(withSwitchedOff('NPC Map Locations', true, 'switched off')).toBe(
      'NPC Map Locations (switched off)',
    )
    expect(
      formatPreviewRow(
        {
          name: 'NPC Map Locations',
          version: '3.3.0',
          source: 'Nexus',
          status: 'imported',
          disabled: true,
        },
        'switched off',
      ),
    ).toBe('NPC Map Locations (switched off) · 3.3.0 · Nexus')
    expect(
      formatPreviewRow(
        {
          name: 'DisabledCopy',
          status: 'skipped',
          reason: 'same mod as NPCMapLocations',
        },
        'switched off',
      ),
    ).toBe('DisabledCopy · same mod as NPCMapLocations')
  })

  test('joins skipped and failed outcomes for toast detail', () => {
    expect(
      formatOutcomeDetail([
        { name: 'DisabledCopy', status: 'skipped', reason: 'same mod as NPCMapLocations' },
        { name: 'Loud', status: 'imported' },
        { name: 'Broken', status: 'failed', reason: 'The manifest is invalid' },
      ]),
    ).toBe('DisabledCopy · same mod as NPCMapLocations\nBroken · The manifest is invalid')
  })
})
