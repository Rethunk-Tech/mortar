import { expect, test } from 'bun:test'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { ListRow } from './listColumns.ts'
import { foldLibraries, nestImplicit } from './nestImplicit.ts'

const row = (id: string, needs: string[], implicit = false) =>
  ({ mod: { key: id, id, name: id, needs } as unknown as Mod, implicit }) as ListRow

test('implicit mods sit beneath the first mod that requires them, and an orphan keeps its place', () => {
  const items = [
    row('Lib.Core', ['Lib.Deep'], true),
    row('Me.A', ['Lib.Core']),
    row('Lib.Orphan', ['Lib.Own'], true),
    row('Me.B', ['Lib.Core', 'Me.A']),
    row('Lib.Deep', [], true),
    row('Lib.Own', [], true),
  ]
  const [group] = nestImplicit([{ key: '', items }])
  expect(group?.items.map((r) => `${r.mod.id}:${r.depth ?? 0}`)).toEqual([
    'Me.A:0',
    'Lib.Core:1',
    'Lib.Deep:2',
    'Lib.Orphan:0',
    'Lib.Own:1',
    'Me.B:0',
  ])
})

test('the card view folds nested implicit mods into the card above them', () => {
  const items = [
    row('Lib.Core', ['Lib.Deep'], true),
    row('Me.A', ['Lib.Core']),
    row('Lib.Deep', [], true),
    row('Me.B', []),
  ]
  const [group] = foldLibraries(nestImplicit([{ key: '', items }]))
  expect(group?.items.map((r) => r.mod.id)).toEqual(['Me.A', 'Me.B'])
  expect(group?.items[0]?.libraries?.map((r) => r.mod.id)).toEqual(['Lib.Core', 'Lib.Deep'])
  expect(group?.items[1]?.libraries).toBeUndefined()
})
