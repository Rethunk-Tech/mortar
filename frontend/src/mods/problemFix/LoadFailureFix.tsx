import { useLingui } from '@lingui/react/macro'
import { useState } from 'react'
import { RemoveSkippedPlugin } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import { useBrowseView } from '../../browse/view.ts'
import { useTab } from '../../game/tab.ts'
import { useLoader } from '../../loader/store.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import type { Problem } from '../lookup.ts'
import { useMods } from '../store.ts'
import { openTarget } from '../storeView.ts'
import { useUpdates } from '../updates.ts'
import type { WarningButton } from './warningButton.tsx'

// The fix follows the cause the log names: a missing plugin is found, one built for other versions is updated, a
// broken loader is reinstalled, and a plugin that clashes or keeps failing is disabled.
export function LoadFailureFix({
  problem,
  button,
}: {
  problem: Extract<Problem, { kind: 'loadFailure' }>
  button: WarningButton
}) {
  const { t } = useLingui()
  const failure = problem.loadFailure
  const [removing, setRemoving] = useState(false)
  const owner = useMods((s) => s.mods.find((m) => failure.key !== '' && m.key === failure.key))
  const disable = () =>
    owner
      ? button(t`Disable`, () =>
          useMods.getState().setEnabled(owner, false).catch(reportUnexpected),
        )
      : null
  switch (failure.kind) {
    case 'duplicate-guid': {
      // A copy a package owns is switched off like any plugin; a leftover file in the plugins folder is deleted.
      const path = failure.path ?? ''
      if (path === '') {
        return owner ? disable() : null
      }
      return (
        <>
          {button(t`Remove old copy`, () => setRemoving(true))}
          <ConfirmDialog
            open={removing}
            title={t`Remove the older copy?`}
            body={t`BepInEx skipped ${path} because a newer copy of the plugin loaded. Mortar deletes the file.`}
            confirmLabel={t`Remove`}
            color="error"
            onCancel={() => setRemoving(false)}
            onConfirm={() => {
              setRemoving(false)
              const at = openTarget()
              if (at) {
                RemoveSkippedPlugin(at.game, at.id, path)
                  .then(() => useMods.getState().load())
                  .catch(reportUnexpected)
              }
            }}
          />
        </>
      )
    }
    case 'missing-dependency': {
      const dependency = failure.dependency ?? ''
      return dependency === ''
        ? null
        : button(t`Find ${dependency}`, () => {
            useBrowseView.getState().setPendingQuery(dependency)
            useTab.getState().setTab('browse')
          })
    }
    case 'incompatible-version':
    case 'load-exception':
    case 'patch-exception':
      return button(t`Update`, () => {
        useTab.getState().setTab('mods')
        useMods.getState().showUpdates()
        useUpdates.getState().setReviewing(true)
      })
    case 'preloader-patch':
    case 'loader-version':
    case 'chainloader':
      // A patcher that failed is the patcher's fault; only the preloader itself failing is the loader's.
      if (
        failure.kind === 'preloader-patch' &&
        !failure.message.startsWith('Could not run preloader')
      ) {
        return null
      }
      return button(t`Reinstall loader`, () => {
        const at = openTarget()
        if (at) {
          useLoader.getState().install(at.game).catch(reportUnexpected)
        }
      })
    default:
      return owner ? disable() : null
  }
}
