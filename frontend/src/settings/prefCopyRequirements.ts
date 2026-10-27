import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import type { PrefCopy } from './prefCopyAntivirus.ts'

// Settings › a game › what switching a mod on or off, or installing one, does to the mods around it.
export function requirementPrefs(i18n: I18n): Record<string, PrefCopy> {
  const alwaysAskNever = [
    { value: 'always', label: i18n._(msg`Always`) },
    { value: 'ask', label: i18n._(msg`Ask`) },
    { value: 'never', label: i18n._(msg`Never`) },
  ]
  return {
    enableRequirements: {
      label: i18n._(msg`Auto-enable requirements`),
      description: i18n._(
        msg`When you enable a mod, also enable its required mods already in the profile`,
      ),
      options: alwaysAskNever,
    },
    disableDependencies: {
      label: i18n._(msg`Disable dependencies`),
      description: i18n._(
        msg`When you disable a mod, also disable the mods that need it and the libraries only it needed`,
      ),
      options: alwaysAskNever,
    },
    missingRequirements: {
      label: i18n._(msg`Missing requirements on install`),
      description: i18n._(msg`What to do when an installed mod still needs other mods`),
      options: [
        { value: 'ask', label: i18n._(msg`Ask`) },
        { value: 'autodownload', label: i18n._(msg`Download them`) },
        { value: 'never', label: i18n._(msg`Never`) },
      ],
    },
  }
}
