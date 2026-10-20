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
    cacheClearing: {
      label: i18n._(msg`Clear game caches`),
      description: i18n._(
        msg`Whether Mortar deletes the game's rebuildable cache files on the first launch after this profile's mods change`,
      ),
      options: [
        {
          value: 'auto',
          label: i18n._(msg`Auto`),
          hint: i18n._(
            msg`Delete the caches once after the mod set changes; the game rebuilds them.`,
          ),
        },
        {
          value: 'off',
          label: i18n._(msg`Off`),
          hint: i18n._(msg`Leave the caches alone.`),
        },
      ],
    },
    dependencyVersions: {
      label: i18n._(msg`Dependency versions`),
      description: i18n._(
        msg`Which version of a Thunderstore mod's dependency to install when the mod names one`,
      ),
      options: [
        {
          value: 'newest',
          label: i18n._(msg`Newest`),
          hint: i18n._(msg`Install the newest version; the mod's version is a minimum.`),
        },
        {
          value: 'exact',
          label: i18n._(msg`Exact`),
          hint: i18n._(msg`Install the version the mod's author named, as Gale does.`),
        },
      ],
    },
    syncMode: {
      label: i18n._(msg`Profile sync`),
      description: i18n._(msg`Where this game's profiles are shared between your computers`),
      options: [
        {
          value: 'follow',
          label: i18n._(msg`Mortar-wide folder`),
          hint: i18n._(msg`Use the sync folder set under Mods and profiles, if any.`),
        },
        {
          value: 'own',
          label: i18n._(msg`This game's folder`),
          hint: i18n._(msg`Sync into a folder chosen for this game alone.`),
        },
        {
          value: 'off',
          label: i18n._(msg`Off`),
          hint: i18n._(msg`Do not sync this game's profiles.`),
        },
      ],
    },
    syncOwnFolder: {
      label: i18n._(msg`Sync folder for this game`),
      description: i18n._(msg`The folder this game's profiles sync into`),
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
    skipPlayCheck: {
      label: i18n._(msg`Skip the pre-Play check`),
      description: i18n._(msg`Start the game without the Before you play dialog`),
    },
    instances: {
      label: i18n._(msg`Instances`),
      description: i18n._(
        msg`How many game processes a direct Play starts. Hidden for Steam and store URL launches.`,
      ),
    },
    delaySeconds: {
      label: i18n._(msg`Delay seconds`),
      description: i18n._(msg`Seconds to wait between those processes`),
    },
  }
}
