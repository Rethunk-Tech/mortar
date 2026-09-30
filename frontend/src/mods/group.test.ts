import { expect, test } from 'bun:test'
import type { Entry } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { useSettings } from '../settings/store.ts'
import { firstTag, groupSorted, profileTags, sanitizeListGroupBy, takeTags } from './group.ts'

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
  expect(useSettings.getState().listGroupBy).toBe('none')
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
  expect(sanitizeListGroupBy('nope')).toBe('none')
  expect(sanitizeListGroupBy('tag')).toBe('tag')
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
