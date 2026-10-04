import { expect, test } from 'bun:test'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import type { UpdatesResult } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { smapiUpdateNotes } from './smapiUpdateNotes.ts'

const row = (seq: number, message: string) =>
  ({ seq, level: 'ALERT', mod: 'SMAPI', message }) as Entry

test("SMAPI's update lines get Mortar's reason", () => {
  const rows = [
    row(1, 'You can update 3 mods:'),
    row(2, '   Aspen 0.0.53: https://www.nexusmods.com/stardewvalley/mods/6754 (you have 0.0.52)'),
    row(
      3,
      '   Love Festival 1.4.4: https://www.curseforge.com/stardewvalley/mods/love-festival (you have 1.4.3)',
    ),
    row(
      4,
      '   Fish Helper UI 1.2.0: https://www.nexusmods.com/stardewvalley/mods/33167 (you have 1.1.0)',
    ),
  ]
  const result = {
    updates: [{ name: 'Fish Helper UI', version: '1.2.0', nexusId: 33_167, githubRepo: '' }],
    held: [
      { name: 'Aspen', version: '0.0.53', have: '0.0.53', reason: 'current' },
      { name: 'Love Festival', version: '1.4.4', have: '1.4.3', reason: 'skipped' },
    ],
  } as unknown as UpdatesResult
  const notes = smapiUpdateNotes(rows, result)
  expect(notes.get(1)).toBeUndefined()
  expect(notes.get(2)).toEqual({ kind: 'current', have: '0.0.53' })
  expect(notes.get(3)).toEqual({ kind: 'skipped', have: '1.4.3' })
  expect(notes.get(4)).toEqual({ kind: 'listed' })
})

test('an update only on another site is named as such', () => {
  const notes = smapiUpdateNotes(
    [
      row(
        9,
        '   Thing 2.0.0: https://www.curseforge.com/stardewvalley/mods/thing (you have 1.0.0)',
      ),
    ],
    { updates: [], held: [] } as unknown as UpdatesResult,
  )
  expect(notes.get(9)).toEqual({ kind: 'elsewhere' })
})
