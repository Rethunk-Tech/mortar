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

test('an implicit library that sorts before its requirer still nests beneath it', () => {
  const items = [
    row('thunderstore:FlipMods-ReservedItemSlotCore', [], true),
    row('thunderstore:Piggy-ReservedWalkieSlotKR', ['thunderstore:FlipMods-ReservedItemSlotCore']),
  ]
  const [group] = nestImplicit([{ key: 'enabled', items }])
  expect(group?.items.map((r) => `${r.mod.id}:${r.depth ?? 0}`)).toEqual([
    'thunderstore:Piggy-ReservedWalkieSlotKR:0',
    'thunderstore:FlipMods-ReservedItemSlotCore:1',
  ])
})

test('implicit mods follow their requirer into another group, and a group left empty is dropped', () => {
  const groups = [
    { key: 'problems', items: [row('Me.Bingo', ['Me.Detour5', 'Lib.Canvas'])] },
    {
      key: 'enabled',
      items: [
        row('Lib.Canvas', ['Lib.Wav'], true),
        row('Lib.Detour', [], true),
        row('Lib.Wav', [], true),
      ],
    },
    { key: 'disabled', items: [row('Me.Detour5', ['Lib.Detour'])] },
  ]
  const nested = nestImplicit(groups)
  expect(
    nested.map((g) => `${g.key}=${g.items.map((r) => `${r.mod.id}:${r.depth ?? 0}`).join(',')}`),
  ).toEqual(['problems=Me.Bingo:0,Lib.Canvas:1,Lib.Wav:2', 'disabled=Me.Detour5:0,Lib.Detour:1'])
})

test('a parent group with no mods of its own survives nesting; a group emptied by nesting does not', () => {
  const libs = [row('Lib', [], true)]
  const user = [row('Me', ['Lib'])]
  const out = nestImplicit([
    { key: 'Looks', items: [] },
    { key: 'Looks/Hats', items: user },
    { key: 'Libs', items: libs },
  ])
  expect(out.map((g) => g.key)).toEqual(['Looks', 'Looks/Hats'])
})
