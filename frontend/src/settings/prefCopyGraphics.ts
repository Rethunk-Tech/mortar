import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import type { PrefCopy } from './prefCopyAntivirus.ts'

export function graphicsPrefs(i18n: I18n): Record<string, PrefCopy> {
  return {
    graphics: {
      label: i18n._(msg`Graphics`),
      description: i18n._(msg`Applies after Mortar restarts.`),
      options: [
        { value: 'automatic', label: i18n._(msg`Automatic`) },
        {
          value: 'smooth',
          label: i18n._(msg`Smoother scrolling`),
          hint: i18n._(
            msg`Try this if scrolling is slow or the fans spin up, as on some graphics cards.`,
          ),
        },
        {
          value: 'compatible',
          label: i18n._(msg`Compatible`),
          hint: i18n._(msg`Try this if the window stays blank or white.`),
        },
      ],
    },
  }
}
