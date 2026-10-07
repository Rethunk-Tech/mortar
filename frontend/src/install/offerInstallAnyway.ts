import { msg } from '@lingui/core/macro'
import { AllowUnscanned } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { type DetectedFile, useOverride } from '../toasts/avOverride.ts'
import { useToasts } from '../toasts/store.ts'

// A refused install the player may overrule: the toast's action asks first, then records the choice and installs again.
export function offerInstallAnyway(o: {
  title: string
  name: string
  flagged: DetectedFile
  game: string
  profile: string
  install: () => Promise<unknown>
}) {
  const { title, name, flagged, game, profile, install } = o
  if (flagged.removed) {
    useToasts.getState().push({
      kind: 'error',
      title,
      body: i18n._(
        msg`Windows removed the file as malware without saying which. If you trust it, restore it from Windows Security's protection history and add it again.`,
      ),
    })
    return
  }
  useToasts.getState().push({
    kind: 'error',
    title,
    body: i18n._(msg`The antivirus flagged it, so Mortar did not install it.`),
    detail: `${flagged.scanner}: ${flagged.name}${flagged.file ? ` in ${flagged.file}` : ''}`,
    action: {
      label: i18n._(msg`Install anyway…`),
      run: () =>
        useOverride.getState().ask({
          title: name,
          scanner: flagged.scanner,
          name: flagged.name,
          file: flagged.file,
          confirm: async () => {
            await AllowUnscanned(game, profile, flagged.key, name, flagged.name)
            await install()
          },
        }),
    },
  })
}
