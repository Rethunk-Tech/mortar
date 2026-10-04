import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import type { PaletteLabels } from './items.ts'

export function paletteActionLabels(i18n: I18n): PaletteLabels {
  return {
    play: i18n._(msg`Play the open profile`),
    updates: i18n._(msg`Check for updates`),
    downloads: i18n._(msg`Open Downloads`),
    import: i18n._(msg`Import`),
    pasteLink: i18n._(msg`Paste a link…`),
    collectionReview: i18n._(msg`Review collection update`),
    findCrashCause: i18n._(msg`Find the crash cause`),
    configureMod: (name) => i18n._(msg`Configure ${name}…`),
    share: i18n._(msg`Share`),
    newProfile: i18n._(msg`New profile`),
    streamOverlay: i18n._(msg`Stream overlay`),
    recentChanges: i18n._(msg`Recent changes`),
    diagnostics: i18n._(msg`Diagnostics`),
    profileHint: i18n._(msg`Open profile`),
    modHint: i18n._(msg`Open mod`),
    settingsHint: i18n._(msg`Settings`),
    tabs: {
      browse: i18n._(msg`Switch to Browse`),
      mods: i18n._(msg`Switch to Mods`),
      problems: i18n._(msg`Switch to Problems`),
      'load-order': i18n._(msg`Switch to Load order`),
      saves: i18n._(msg`Switch to Saves`),
      notes: i18n._(msg`Switch to Notes`),
      console: i18n._(msg`Switch to Console`),
      performance: i18n._(msg`Switch to Performance`),
    },
    toggle: (name) => i18n._(msg`Toggle ${name}`),
  }
}
