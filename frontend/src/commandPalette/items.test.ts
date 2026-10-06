import { expect, test } from 'bun:test'
import { buildPaletteItems, type PaletteLabels } from './items.ts'

const labels: PaletteLabels = {
  play: '',
  updates: '',
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
  }).map((item) => item.id)

test('the Stream overlay action shows only for a loader that feeds the overlay', () => {
  expect(ids(true)).toContain('action:stream-overlay')
  expect(ids(false)).not.toContain('action:stream-overlay')
})
