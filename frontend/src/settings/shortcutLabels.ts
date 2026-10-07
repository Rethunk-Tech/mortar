import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import type { ShortcutId } from './shortcuts.ts'

/** What each shortcut does, shared by Settings › Shortcuts and the command palette. */
export function shortcutLabels(i18n: I18n): Record<ShortcutId, string> {
  return {
    'command-palette': i18n._(msg`Open the command palette`),
    'filter-mods': i18n._(msg`Focus the search`),
    play: i18n._(msg`Play the open profile`),
    'check-updates': i18n._(msg`Check mods and Mortar for updates`),
    'open-settings': i18n._(msg`Open settings`),
    dismiss: i18n._(msg`Close dialog or clear selection`),
    'select-all-mods': i18n._(msg`Select all mods`),
    'mod-up': i18n._(msg`Focus the previous mod`),
    'mod-down': i18n._(msg`Focus the next mod`),
    'mod-toggle': i18n._(msg`Switch the focused mod on or off`),
    'mod-details': i18n._(msg`Open focused mod details`),
    'mod-remove': i18n._(msg`Remove the focused mod`),
    'tab-browse': i18n._(msg`Switch to Browse`),
    'tab-load-order': i18n._(msg`Switch to Load order`),
    'tab-mods': i18n._(msg`Switch to Mods`),
    'tab-problems': i18n._(msg`Switch to Problems`),
    'tab-saves': i18n._(msg`Switch to Saves`),
    'tab-console': i18n._(msg`Switch to Console`),
    'tab-performance': i18n._(msg`Switch to Performance`),
    'new-profile': i18n._(msg`Create a new profile`),
    'duplicate-profile': i18n._(msg`Duplicate the open profile`),
    'rename-profile': i18n._(msg`Rename the open profile`),
    'find-all-mods': i18n._(msg`Find a mod in all profiles`),
    import: i18n._(msg`Open the import dialog`),
    'export-profile': i18n._(msg`Export or share the open profile`),
    downloads: i18n._(msg`Toggle downloads`),
    notifications: i18n._(msg`Open notification history`),
    'previous-profile': i18n._(msg`Open the previous profile`),
    'next-profile': i18n._(msg`Open the next profile`),
    'collapse-sidebar': i18n._(msg`Collapse or expand the profile sidebar`),
    back: i18n._(msg`Go back`),
    help: i18n._(msg`Get help`),
    'vanilla-play': i18n._(msg`Play the game without mods`),
  }
}
