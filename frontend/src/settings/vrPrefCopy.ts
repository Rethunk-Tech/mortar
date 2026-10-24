import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { ALVR, STEAMVR } from '../brand/productNames.ts'
import type { PrefCopy } from './prefCopyAntivirus.ts'

// The VR runtime Play can start before a game.
export function vrPrefs(i18n: I18n): Record<string, PrefCopy> {
  return {
    startFirst: {
      label: i18n._(msg`Start first`),
      description: i18n._(
        msg`Starts a VR runtime before the game; the ${{ alvr: ALVR }} dashboard opens ${{ runtime: STEAMVR }} itself. Leave it off in a profile that plays the game on a flat screen.`,
      ),
      options: [
        { value: 'off', label: i18n._(msg`Off`) },
        { value: 'steamvr', label: STEAMVR },
        { value: 'alvr', label: ALVR },
      ],
    },
    alvrProgram: {
      label: i18n._(msg`${{ name: ALVR }} program`),
      description: i18n._(
        msg`The ${{ alvr: ALVR }} dashboard program, started before a game set to start it first`,
      ),
    },
  }
}
