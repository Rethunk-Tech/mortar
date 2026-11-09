import { afterAll, expect, test } from 'bun:test'
import { i18n } from '@lingui/core'
import type {
  Entry,
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

i18n.load('en', {})
i18n.activate('en')

const testing = await import('@testing-library/react')
const { useSettings } = await import('../settings/store.ts')
const { MANUAL_SORT } = await import('./listColumns.ts')
const { useModGroups } = await import('./useModGroups.ts')

afterAll(() => {
  testing.cleanup()
})

// A library Mortar brought in for Host, and a mod that needs nothing. The player sorted by Manual and dragged the
// library above Host, which is the order the profile saved.
const entries: Entry[] = [
  {
    key: 'lib-key',
    previousKey: '',
    source: { kind: 'thunderstore', name: 'Author-Lib', version: '1.0.0' },
    mods: [
      { id: 'thunderstore:Author-Lib', version: '1.0.0', name: 'Lib', author: 'A', folder: '.' },
    ],
    disabled: [],
    implicit: true,
  },
  {
    key: 'host-key',
    previousKey: '',
    source: { kind: 'thunderstore', name: 'Author-Host', version: '1.0.0' },
    mods: [
      {
        id: 'thunderstore:Author-Host',
        version: '1.0.0',
        name: 'Host',
        author: 'A',
        folder: '.',
        needs: ['thunderstore:Author-Lib'],
      },
    ],
    disabled: [],
  },
  {
    key: 'other-key',
    previousKey: '',
    source: { kind: 'thunderstore', name: 'Author-Other', version: '1.0.0' },
    mods: [
      {
        id: 'thunderstore:Author-Other',
        version: '1.0.0',
        name: 'Other',
        author: 'A',
        folder: '.',
      },
    ],
    disabled: [],
  },
]
const mods: Mod[] = entries.map((e) => {
  const [first] = e.mods ?? []
  return {
    key: e.key,
    id: first?.id ?? '',
    name: first?.name ?? '',
    author: 'A',
    version: '1.0.0',
    enabled: true,
    picture: '',
    endorsements: 0,
    needs: first?.needs ?? [],
  }
})
const profile: Profile = {
  id: 'p',
  name: 'Fixture',
  notes: '',
  cover: '',
  order: 0,
  created: '',
  updated: '',
  entries,
  modOrder: ['other-key', 'lib-key', 'host-key'],
}

function renderList(column: string) {
  useSettings.setState({ listSortColumn: column, listSortDir: 'asc', listGroupBy: 'none' })
  return testing.renderHook(() => useModGroups(mods, profile))
}
const shown = (items: readonly { mod: { name: string }; depth?: number }[]) =>
  items.map((r) => `${r.mod.name}:${r.depth ?? 0}`)

test('a manual order keeps a library above the mod that needs it, where the player dropped it', () => {
  const { result, unmount } = renderList(MANUAL_SORT)
  expect(shown(result.current.groups[0]?.items ?? [])).toEqual(['Other:0', 'Lib:0', 'Host:0'])
  unmount()
})

test('a sorted order still nests the library beneath the mod that needs it', () => {
  const { result, unmount } = renderList('name')
  expect(shown(result.current.groups[0]?.items ?? [])).toEqual(['Host:0', 'Lib:1', 'Other:0'])
  unmount()
})
