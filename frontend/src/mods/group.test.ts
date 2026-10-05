import { expect, test } from 'bun:test'
import type {
  Entry,
  Mod,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { DriftKind } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useSettings } from '../settings/store.ts'
import {
  customCategoryById,
  firstTag,
  frameworkGroupKey,
  groupSorted,
  profileTags,
  resolvedCategoryLabel,
  rowGroupKey,
  SMAPI_MODS_GROUP,
  sanitizeListGroupBy,
  statusGroupKey,
  takeTags,
} from './group.ts'
import { modStatusProblem } from './lookup.ts'

const entry = (over: Partial<Entry> & Pick<Entry, 'key'>): Entry => {
  const out: Entry = {
    key: over.key,
    previousKey: over.previousKey ?? '',
    source: over.source ?? { kind: 'local', name: '' },
    mods: over.mods ?? [],
    disabled: over.disabled ?? [],
  }
  if (over.tags) {
    out.tags = over.tags
  }
  if (over.note) {
    out.note = over.note
  }
  return out
}

test('resets settings group-by from getInitialState', () => {
  useSettings.setState({ listGroupBy: 'tag' })
  useSettings.setState(useSettings.getInitialState(), true)
  expect(useSettings.getState().listGroupBy).toBe('status')
})

test('first tag and profile tag suggestions', () => {
  expect(firstTag(null)).toBe('')
  expect(firstTag(['  ', 'QoL', 'Farm'])).toBe('QoL')
  expect(
    profileTags([
      entry({ key: 'a', tags: ['Farm', 'farm', ' QoL '] }),
      entry({ key: 'b', tags: ['Crops'] }),
    ]),
  ).toEqual(['Crops', 'Farm', 'QoL'])
})

test('takeTags caps length, count and duplicates', () => {
  expect(takeTags(['  QoL ', 'qol', '', 'xxxxxxxxxxxxxxxxxxxxxxxxx', 'Farm'])).toEqual([
    'QoL',
    'Farm',
  ])
})

test('groups by first tag, empty last, sort within groups', () => {
  expect(sanitizeListGroupBy('nope')).toBe('status')
  expect(sanitizeListGroupBy('tag')).toBe('tag')
  expect(sanitizeListGroupBy('framework')).toBe('framework')
  expect(sanitizeListGroupBy('group')).toBe('group')
  const items = [
    { name: 'Zed', tag: 'b' },
    { name: 'Ann', tag: 'a' },
    { name: 'Bob', tag: 'a' },
    { name: 'Una', tag: '' },
  ]
  const groups = groupSorted(
    items,
    'tag',
    (item) => item.tag,
    (a, b) => a.name.localeCompare(b.name),
  )
  expect(groups.map((g) => [g.key, g.items.map((i) => i.name)])).toEqual([
    ['a', ['Ann', 'Bob']],
    ['b', ['Zed']],
    ['', ['Una']],
  ])
  const none = groupSorted(
    items,
    'none',
    (item) => item.tag,
    (a, b) => a.name.localeCompare(b.name),
  )
  expect(none).toHaveLength(1)
  expect(none[0]?.items.map((i) => i.name)).toEqual(['Ann', 'Bob', 'Una', 'Zed'])
})

test('status grouping puts drift-affected entries in Problems', () => {
  const mod = { key: 'k', id: 'me.a', name: 'A', enabled: true }
  const result = {
    missing: [],
    duplicates: [],
    broken: [],
    assetConflicts: [],
    settings: [],
    runErrors: [],
    drift: [{ kind: DriftKind.DriftChanged, folder: 'k', key: 'k' }],
    dismissed: [],
    unknown: false,
  }
  const fullMod = mod as Mod
  expect(modStatusProblem(result, fullMod)).toBe(true)
  const row = { source: '', tags: [], mod: fullMod }
  const emptyCustom = customCategoryById([])
  expect(
    rowGroupKey('status', row, {
      hasProblem: modStatusProblem(result, fullMod),
      hasUpdate: false,
      names: new Map(),
      customById: emptyCustom,
    }),
  ).toBe('problems')
})

test('category grouping uses entry override before Nexus', () => {
  const custom = customCategoryById([{ id: 'abc123', name: 'My QoL', color: 'teal' }])
  const row = {
    source: '',
    tags: [],
    categoryOverride: 'abc123',
    details: { category: 'User Interface' },
    mod: {
      id: 'A.Mod',
      author: '',
      enabled: true,
    },
  }
  expect(
    rowGroupKey('category', row, {
      hasProblem: false,
      hasUpdate: false,
      names: new Map(),
      customById: custom,
    }),
  ).toBe('My QoL')
  expect(resolvedCategoryLabel('Crops', 'User Interface', custom)).toBe('Crops')
})

test('status grouping uses Problems, Update available, Enabled, Disabled', () => {
  expect(statusGroupKey(true, true, false)).toBe('problems')
  expect(statusGroupKey(false, true, false)).toBe('update')
  expect(statusGroupKey(false, false, true)).toBe('enabled')
  expect(statusGroupKey(false, false, false)).toBe('disabled')
  const items = [
    { name: 'Off', key: 'disabled' },
    { name: 'New', key: 'update' },
    { name: 'Ok', key: 'enabled' },
    { name: 'Broke', key: 'problems' },
    { name: 'Also ok', key: 'enabled' },
  ]
  const groups = groupSorted(
    items,
    'status',
    (item) => item.key,
    (a, b) => a.name.localeCompare(b.name),
  )
  expect(groups.map((g) => [g.key, g.items.map((i) => i.name)])).toEqual([
    ['problems', ['Broke']],
    ['update', ['New']],
    ['enabled', ['Also ok', 'Ok']],
    ['disabled', ['Off']],
  ])
})

test('framework grouping uses contentPackFor when the pack has other required dependencies', () => {
  const names = new Map([['pathoschild.contentpatcher', 'Content Patcher']])
  const ctx = { hasProblem: false, hasUpdate: false, names, customById: customCategoryById([]) }
  const row = {
    source: '',
    tags: [],
    mod: {
      id: 'Author.Pack',
      author: '',
      enabled: true,
      needs: ['B.Req', 'Pathoschild.ContentPatcher'],
      contentPackFor: 'Pathoschild.ContentPatcher',
    },
  }
  expect(rowGroupKey('framework', row, ctx)).toBe('Content Patcher')
})

test('framework grouping names installed frameworks and keeps UniqueID when missing', () => {
  const names = new Map([
    ['pathoschild.contentpatcher', 'Content Patcher'],
    ['peacefulend.alternativetextures', 'Alternative Textures'],
    ['spacechase0.jsonassets', 'JSON Assets'],
  ])
  expect(frameworkGroupKey('Pathoschild.ContentPatcher', 'Some.Pack', names)).toBe(
    'Content Patcher',
  )
  expect(frameworkGroupKey('Missing.Framework', 'Some.Pack', names)).toBe('Missing.Framework')
  expect(frameworkGroupKey('', 'Pathoschild.ContentPatcher', names)).toBe(SMAPI_MODS_GROUP)
  expect(frameworkGroupKey('Pathoschild.ContentPatcher', 'Pathoschild.ContentPatcher', names)).toBe(
    SMAPI_MODS_GROUP,
  )
  const items = [
    { name: 'CP', key: frameworkGroupKey('', 'Pathoschild.ContentPatcher', names) },
    { name: 'Pack B', key: frameworkGroupKey('Pathoschild.ContentPatcher', 'B.Pack', names) },
    { name: 'Smapi', key: frameworkGroupKey('', 'Some.CSharp', names) },
    { name: 'Pack A', key: frameworkGroupKey('Pathoschild.ContentPatcher', 'A.Pack', names) },
    { name: 'Ghost pack', key: frameworkGroupKey('Gone.Mod', 'Ghost.Pack', names) },
  ]
  const groups = groupSorted(
    items,
    'framework',
    (item) => item.key,
    (a, b) => a.name.localeCompare(b.name),
  )
  expect(groups.map((g) => [g.key, g.items.map((i) => i.name)])).toEqual([
    ['Content Patcher', ['Pack A', 'Pack B']],
    ['Gone.Mod', ['Ghost pack']],
    [SMAPI_MODS_GROUP, ['CP', 'Smapi']],
  ])
})
