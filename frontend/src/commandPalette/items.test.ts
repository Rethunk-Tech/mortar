import { expect, test } from 'bun:test'
import { buildPaletteItems, type PaletteLabels } from './items.ts'

const labels: PaletteLabels = {
  play: '',
  downloads: '',
  downloadsFolder: '',
  import: '',
  pasteLink: '',
  collectionReview: '',
  findCrashCause: '',
  configureMod: (name) => name,
  share: '',
  newProfile: '',
  streamOverlay: '',
  recentChanges: '',
  diagnostics: '',
  profileHint: '',
  modHint: '',
  settingsHint: '',
  needProfile: 'Open a profile first.',
  tabs: {},
  toggle: (name) => name,
}

const ids = (streamOverlay: boolean) =>
  buildPaletteItems({
    profiles: [],
    mods: [],
    sections: [],
    shortcuts: [],
    shortcutLabels: {},
    labels,
    streamOverlay,
    profileOpen: true,
  }).map((item) => item.id)

test('the Stream overlay action shows only for a loader that feeds the overlay', () => {
  expect(ids(true)).toContain('action:stream-overlay')
  expect(ids(false)).not.toContain('action:stream-overlay')
})

test('the crash check action carries the reason it cannot run', () => {
  const find = (crashCheckBlocked: string | null) =>
    buildPaletteItems({
      profiles: [],
      mods: [],
      sections: [],
      shortcuts: [],
      shortcutLabels: {},
      labels,
      crashCheckBlocked,
      profileOpen: true,
    }).find((item) => item.id === 'action:find-crash-cause')
  expect(find(null)?.disabled).toBeUndefined()
  expect(find('No crash to investigate.')?.disabled).toBe('No crash to investigate.')
})

const withShortcuts = (profileOpen: boolean) =>
  buildPaletteItems({
    profiles: [],
    mods: [],
    sections: [],
    shortcuts: [
      { id: 'play', keys: 'Ctrl+P', always: false, group: 'General' },
      { id: 'check-updates', keys: 'F5', always: false, group: 'General' },
      { id: 'tab-mods', keys: 'Ctrl+2', always: false, group: 'Tabs' },
      { id: 'tab-config', keys: 'Ctrl+6', always: false, group: 'Tabs' },
      { id: 'mod-up', keys: '↑', always: false, group: 'Mods list' },
    ],
    shortcutLabels: { 'check-updates': 'Check mods and Mortar for updates' },
    labels: { ...labels, tabs: { mods: 'Switch to Mods', config: 'Switch to Config' } },
    profileOpen,
  })

test('a shortcut that duplicates an action lends it its keys instead of a second row', () => {
  const items = withShortcuts(true)
  expect(items.find((i) => i.id === 'action:play')?.shortcut).toBe('Ctrl+P')
  expect(items.find((i) => i.id === 'tab:mods')?.shortcut).toBe('Ctrl+2')
  expect(items.find((i) => i.id === 'tab:config')?.label).toBe('Switch to Config')
  expect(items.find((i) => i.id === 'tab:config')?.shortcut).toBe('Ctrl+6')
  expect(items.filter((i) => i.id === 'shortcut:play' || i.id === 'shortcut:tab-mods')).toEqual([])
  expect(items.filter((i) => i.label.includes('for updates'))).toHaveLength(1)
})

test('list-focus shortcuts stay out of the palette', () => {
  expect(withShortcuts(true).some((i) => i.id === 'shortcut:mod-up')).toBe(false)
})

test('actions on the open profile say why they are off when none is open', () => {
  const items = withShortcuts(false)
  expect(items.find((i) => i.id === 'action:play')?.disabled).toBe('Open a profile first.')
  expect(items.find((i) => i.id === 'tab:mods')?.disabled).toBe('Open a profile first.')
  expect(items.find((i) => i.id === 'action:new-profile')?.disabled).toBeUndefined()
})
