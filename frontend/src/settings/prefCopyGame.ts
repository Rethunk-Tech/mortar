import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import type { PrefCopy } from './prefCopyAntivirus.ts'

export function gamePrefs(i18n: I18n): Record<string, PrefCopy> {
  return {
    gameSettingsMode: {
      label: i18n._(msg`Game settings file`),
      description: i18n._(
        msg`What Mortar does about game settings that mods need switched on in the game's options file`,
      ),
      options: [
        {
          value: 'edit',
          label: i18n._(msg`Edit`),
          hint: i18n._(
            msg`Switch them on in this profile's own copy of the file before each launch. Your own file is never changed.`,
          ),
        },
        {
          value: 'warn',
          label: i18n._(msg`Warn`),
          hint: i18n._(msg`Leave the file alone and list the settings under Problems.`),
        },
      ],
    },
    oldFilesOnUpdate: {
      label: i18n._(msg`Files an update no longer includes`),
      description: i18n._(msg`What happens to files the new version of a mod leaves out`),
      options: [
        {
          value: 'ask',
          label: i18n._(msg`Ask`),
          hint: i18n._(msg`Set them aside and ask whether to keep or delete them.`),
        },
        {
          value: 'delete',
          label: i18n._(msg`Delete`),
          hint: i18n._(msg`Delete them; rolling back still restores the old version.`),
        },
        {
          value: 'keep',
          label: i18n._(msg`Keep`),
          hint: i18n._(msg`Carry them into the new version's folder.`),
        },
      ],
    },
  }
}
