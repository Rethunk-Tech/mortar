import { expect, test } from 'bun:test'
import type { Entry } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { useSettings } from '../settings/store.ts'
import {
  firstTag,
  frameworkGroupKey,
  groupSorted,
  profileTags,
  SMAPI_MODS_GROUP,
  sanitizeListGroupBy,
  statusGroupKey,
  takeTags,
} from './group.ts'

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
