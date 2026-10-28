import { useLingui } from '@lingui/react/macro'
import { EyeOff } from 'lucide-react'
import { useState } from 'react'
import {
  DismissLoadFailure,
  RemoveSkippedPlugin,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import { BEPINEX } from '../../brand/productNames.ts'
import { useBrowseView } from '../../browse/view.ts'
import { useTab } from '../../game/tab.ts'
import { useLoader } from '../../loader/store.ts'
import { ConfirmationDialog } from '../../shell/dialog/ConfirmationDialog.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import { modByKey, type Problem } from '../lookup.ts'
import { useMods } from '../store.ts'
import { openTarget } from '../storeView.ts'
import { useUpdates } from '../updates.ts'
import { FixAction } from './fixActions.tsx'
import { RestoreButton } from './RestoreButton.tsx'
import type { WarningButton } from './warningButton.tsx'

// The log's rows stay until the next run says otherwise, so each can be dismissed for this run. The game's version and
// settings rows come from the install rather than a run.
export function LoadFailureFix({
  problem,
  dismissedToken,
  button,
}: {
  problem: Extract<Problem, { kind: 'loadFailure' }>
  dismissedToken?: string | undefined
  button: WarningButton
}) {
  const { t } = useLingui()
  const failure = problem.loadFailure
  const dismissable = failure.kind !== 'game-version' && failure.kind !== 'game-setting'
  if (dismissedToken !== undefined) {
    return <RestoreButton token={dismissedToken} />
  }
  return (
    <>
      <LoadFailureAction problem={problem} button={button} />
      {dismissable ? (
        <FixAction
          icon={<EyeOff size={16} aria-hidden={true} />}
          label={t`Dismiss`}
          weight="quiet"
          onClick={() => {
            const at = openTarget()
            if (at) {
              DismissLoadFailure(at.game, at.id, failure.kind, failure.plugin, failure.message)
                .then(() => useMods.getState().loadProblems())
                .catch(reportUnexpected)
            }
          }}
        />
      ) : null}
    </>
  )
}

// The fix follows the cause the log names: a missing plugin is found, one built for other versions is updated, a
// broken loader is reinstalled, and a plugin that clashes or keeps failing is disabled.
function LoadFailureAction({
  problem,
  button,
}: {
  problem: Extract<Problem, { kind: 'loadFailure' }>
  button: WarningButton
}) {
  const { t } = useLingui()
  const failure = problem.loadFailure
  const [removing, setRemoving] = useState(false)
  const owner = useMods((s) => (failure.key === '' ? undefined : modByKey(s.mods, failure.key)))
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
          {button(t`Delete old copy`, () => setRemoving(true))}
          <ConfirmationDialog
            open={removing}
            title={t`Delete the older copy?`}
            body={t`${{ loader: BEPINEX }} skipped ${path} because a newer copy of the plugin loaded. Mortar deletes the file.`}
            confirmLabel={t`Delete`}
            tone="danger"
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
